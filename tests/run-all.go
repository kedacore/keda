//go:build e2e
// +build e2e

package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/kedacore/keda/v2/tests/helper"
)

var (
	concurrentTests        = 15
	regularTestsTimeout    = "20m"
	regularTestsRetries    = 3
	sequentialTestsTimeout = "20m"
	sequentialTestsRetries = 1
)

type TestResult struct {
	TestCase string
	Passed   bool
	Attempts []string
}

func main() {
	ctx := context.Background()

	e2eRegex := os.Getenv("E2E_TEST_REGEX")
	if e2eRegex == "" {
		e2eRegex = ".*_test.go"
	}

	//
	// Install KEDA
	//
	installation := executeTest(ctx, "tests/utils/setup_test.go", "15m", 1)
	fmt.Print(installation.Attempts[0])
	if !installation.Passed {
		printKedaLogs()
		uninstallKeda(ctx)
		os.Exit(1)
	}

	//
	// Detect test cases
	//
	regularTestFiles := getRegularTestFiles(e2eRegex)
	sequentialTestFiles := getSequentialTestFiles(e2eRegex)
	if len(regularTestFiles) == 0 && len(sequentialTestFiles) == 0 {
		uninstallKeda(ctx)
		fmt.Printf("No test has been executed, please review your regex: '%s'\n", e2eRegex)
		os.Exit(1)
	}

	//
	// Execute regular tests
	//
	regularTestResults := executeRegularTests(ctx, regularTestFiles)

	//
	// Execute secuential tests
	//
	sequentialTestResults := executeSequentialTests(ctx, sequentialTestFiles)

	//
	// Uninstall KEDA
	//
	passed := uninstallKeda(ctx)
	if !passed {
		os.Exit(1)
	}

	//
	// Generate execution outcome
	//
	testResults := []TestResult{}
	testResults = append(testResults, regularTestResults...)
	testResults = append(testResults, sequentialTestResults...)
	exitCode := evaluateExecution(testResults)

	os.Exit(exitCode)
}

func executeTest(ctx context.Context, file string, timeout string, tries int) TestResult {
	result := TestResult{
		TestCase: file,
		Passed:   false,
		Attempts: []string{},
	}
	for i := 1; i <= tries; i++ {
		fmt.Printf("Executing %s, attempt %q\n", file, numberToWord(i))
		cmd := exec.CommandContext(ctx, "go", "test", "-v", "-tags", "e2e", "-timeout", timeout, file)
		stdout, err := cmd.Output()
		logFile := fmt.Sprintf("%s.%d.log", file, i)
		fileError := os.WriteFile(logFile, stdout, 0644)
		if fileError != nil {
			fmt.Printf("Execution of %s, attempt %q has failed writing the logs : %s\n", file, numberToWord(i), fileError)
		}
		result.Attempts = append(result.Attempts, string(stdout))
		if err == nil {
			fmt.Printf("Execution of %s, attempt %q has passed\n", file, numberToWord(i))
			result.Passed = true
			break
		}
		fmt.Printf("Execution of %s, attempt %q has failed: %s \n", file, numberToWord(i), err)
	}
	return result
}

func getRegularTestFiles(e2eRegex string) []string {
	// We exclude utils and sequential folders and helper files
	filter := func(path string, file string) bool {
		return !strings.HasPrefix(path, "tests") || // we need this condition to skip non e2e test from execution
			strings.Contains(path, "utils") ||
			strings.Contains(path, "sequential") ||
			!strings.HasSuffix(file, "_test.go")
	}
	return getTestFiles(e2eRegex, filter)
}

func getSequentialTestFiles(e2eRegex string) []string {
	filter := func(path string, file string) bool {
		return !strings.HasPrefix(path, "tests") || // we need this condition to skip non e2e test from execution
			!strings.Contains(path, "sequential") ||
			!strings.HasSuffix(file, "_test.go")
	}
	return getTestFiles(e2eRegex, filter)
}

func getTestFiles(e2eRegex string, filter func(path string, file string) bool) []string {
	testFiles := []string{}
	regex, err := regexp.Compile(e2eRegex)

	if err != nil {
		return testFiles
	}

	err = filepath.Walk(".",
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			// We exclude utils and sequential folders and helper files
			if filter(path, info.Name()) {
				return nil
			}
			if regex.MatchString(path) {
				testFiles = append(testFiles, path)
			}

			return nil
		})

	if err != nil {
		return []string{}
	}

	// We randomize the executions
	rand.New(rand.NewSource(time.Now().UnixNano()))
	rand.Shuffle(len(testFiles), func(i, j int) {
		testFiles[i], testFiles[j] = testFiles[j], testFiles[i]
	})

	return testFiles
}

func executeRegularTests(ctx context.Context, testCases []string) []TestResult {
	sem := semaphore.NewWeighted(int64(concurrentTests))
	mutex := &sync.RWMutex{}
	testResults := []TestResult{}

	//
	// Execute regular tests
	//
	for _, testFile := range testCases {
		if err := sem.Acquire(ctx, 1); err != nil {
			fmt.Printf("Failed to acquire semaphore: %v", err)
			uninstallKeda(ctx)
			os.Exit(1)
		}

		go func(file string) {
			defer sem.Release(1)
			testExecution := executeTest(ctx, file, regularTestsTimeout, regularTestsRetries)
			mutex.Lock()
			testResults = append(testResults, testExecution)
			mutex.Unlock()
		}(testFile)
	}

	// Wait until all secuential tests ends
	if err := sem.Acquire(ctx, int64(concurrentTests)); err != nil {
		log.Printf("Failed to acquire semaphore: %v", err)
	}

	//
	// Print regular logs
	//

	for _, result := range testResults {
		status := "failed"
		if result.Passed {
			status = "passed"
		}
		fmt.Printf("%s has %s after %q attempts \n", result.TestCase, status, numberToWord(len(result.Attempts)))
		for index, log := range result.Attempts {
			fmt.Printf("attempt number %q\n", numberToWord(index+1))
			fmt.Println(log)
		}
	}

	if len(testResults) > 0 {
		printKedaLogs()
	}
	return testResults
}

func executeSequentialTests(ctx context.Context, testCases []string) []TestResult {
	testResults := []TestResult{}

	//
	// Execute secuential tests
	//

	for _, testFile := range testCases {
		testExecution := executeTest(ctx, testFile, sequentialTestsTimeout, sequentialTestsRetries)
		testResults = append(testResults, testExecution)
	}

	//
	// Print secuential logs
	//

	for _, result := range testResults {
		status := "failed"
		if result.Passed {
			status = "passed"
		}
		fmt.Printf("%s has %s after %q attempts \n", result.TestCase, status, numberToWord(len(result.Attempts)))
		for index, log := range result.Attempts {
			fmt.Printf("attempt number %q\n", numberToWord(index+1))
			fmt.Println(log)
		}
		dir := filepath.Dir(result.TestCase)
		files, _ := os.ReadDir(dir)
		fmt.Println(">>> KEDA Operator log <<<")
		for _, file := range files {
			if strings.Contains(file.Name(), "operator") {
				fmt.Println("##############################################")
				content, _ := os.ReadFile(path.Join(dir, file.Name()))
				fmt.Println(string(content))
				fmt.Println("##############################################")
			}
		}

		fmt.Println(">>> KEDA Metrics Server log <<<")
		for _, file := range files {
			if strings.Contains(file.Name(), "metrics-server") {
				fmt.Println("##############################################")
				content, _ := os.ReadFile(path.Join(dir, file.Name()))
				fmt.Println(string(content))
				fmt.Println("##############################################")
			}
		}
	}

	return testResults
}

func uninstallKeda(ctx context.Context) bool {
	removal := executeTest(ctx, "tests/utils/cleanup_test.go", "15m", 1)
	fmt.Print(removal.Attempts[0])
	return removal.Passed
}

func evaluateExecution(testResults []TestResult) int {
	passSummary := []string{}
	failSummary := []string{}
	exitCode := 0

	for _, result := range testResults {
		if !result.Passed {
			message := fmt.Sprintf("\tExecution of %s, has failed after %q attempts", result.TestCase, numberToWord(len(result.Attempts)))
			failSummary = append(failSummary, message)
			exitCode = 1
			continue
		}
		message := fmt.Sprintf("\tExecution of %s, has passed after %q attempts", result.TestCase, numberToWord(len(result.Attempts)))
		passSummary = append(passSummary, message)
	}

	fmt.Println("##############################################")
	fmt.Println("##############################################")
	fmt.Println("EXECUTION SUMMARY")
	fmt.Println("##############################################")
	fmt.Println("##############################################")

	if len(passSummary) > 0 {
		fmt.Println("Passed tests:")
		for _, message := range passSummary {
			fmt.Println(message)
		}
	}

	if len(failSummary) > 0 {
		fmt.Println("Failed tests:")
		for _, message := range failSummary {
			fmt.Println(message)
		}
	}

	return exitCode
}

// numberToWord converts input integer (0-20) to corresponding word (zero-twenty)
// numbers > 20 are just converted from int to string.
// We need to do this hack, because GitHub Actions obfuscate numbers in the log (eg. 1 -> ***),
// which is very not very helpful :(
func numberToWord(num int) string {
	if num >= 0 && num <= 20 {
		words := []string{
			"zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine",
			"ten", "eleven", "twelve", "thirteen", "fourteen", "fifteen", "sixteen",
			"seventeen", "eighteen", "nineteen", "twenty",
		}
		return words[num]
	}
	return fmt.Sprintf("%d", num)
}

func printKedaLogs() {
	kubeConfig, _ := config.GetConfig()
	kubeClient, _ := kubernetes.NewForConfig(kubeConfig)

	operatorLogs, err := helper.FindPodLogs(kubeClient, "keda", "app=keda-operator")
	if err == nil {
		fmt.Println(">>> KEDA Operator log <<<")
		fmt.Println(operatorLogs)
		fmt.Println("##############################################")
		fmt.Println("##############################################")
	}

	msLogs, err := helper.FindPodLogs(kubeClient, "keda", "app=keda-metrics-apiserver")
	if err == nil {
		fmt.Println(">>> KEDA Metrics Server log <<<")
		fmt.Println(msLogs)
		fmt.Println("##############################################")
		fmt.Println("##############################################")
	}

	hooksLogs, err := helper.FindPodLogs(kubeClient, "keda", "app=keda-admission-webhooks")
	if err == nil {
		fmt.Println(">>> KEDA Admission Webhooks log <<<")
		fmt.Println(hooksLogs)
		fmt.Println("##############################################")
		fmt.Println("##############################################")
	}
}
