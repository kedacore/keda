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
	"time"

	semaphore "golang.org/x/sync/semaphore"
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
	Tries    []string
}

func main() {
	ctx := context.Background()
	sem := semaphore.NewWeighted(int64(concurrentTests))
	regularTestResults := []TestResult{}
	sequentialTestResults := []TestResult{}

	e2eRegex := os.Getenv("E2E_TEST_REGEX")
	if e2eRegex == "" {
		e2eRegex = ".*_test.go"
	}

	//
	// Install KEDA
	//

	installation := executeTest(ctx, "tests/utils/setup_test.go", "15m", 1)
	fmt.Print(installation.Tries[0])
	if !installation.Passed {
		os.Exit(1)
	}

	//
	// Detect test cases
	//

	regularTestFiles := getRegularTestFiles(e2eRegex)
	sequentialTestFiles := getSequentialTestFiles(e2eRegex)
	if len(regularTestFiles) == 0 && len(sequentialTestFiles) == 0 {
		fmt.Printf("No test has been executed, please review your regex: '%s'", e2eRegex)
		os.Exit(1)
	}

	//
	// Execute regular tests
	//

	for _, testFile := range regularTestFiles {
		if err := sem.Acquire(ctx, 1); err != nil {
			fmt.Printf("Failed to acquire semaphore: %v", err)
			os.Exit(1)
		}

		go func(file string) {
			defer sem.Release(1)
			testExecution := executeTest(ctx, file, regularTestsTimeout, regularTestsRetries)
			regularTestResults = append(regularTestResults, testExecution)
		}(testFile)
	}

	// Wait until all secuential tests ends
	if err := sem.Acquire(ctx, int64(concurrentTests)); err != nil {
		log.Printf("Failed to acquire semaphore: %v", err)
	}

	//
	// Print regular logs
	//

	for _, result := range regularTestResults {
		status := "failed"
		if result.Passed {
			status = "passed"
		}
		fmt.Printf("%s has %s after %d tries \n", result.TestCase, status, len(result.Tries))
		for index, log := range result.Tries {
			fmt.Printf("try number %d\n", index+1)
			fmt.Println(log)
		}
	}

	if len(regularTestResults) > 0 {
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
	//
	// Execute secuential tests
	//

	for _, testFile := range sequentialTestFiles {
		testExecution := executeTest(ctx, testFile, sequentialTestsTimeout, sequentialTestsRetries)
		sequentialTestResults = append(sequentialTestResults, testExecution)
	}

	//
	// Print secuential logs
	//

	for _, result := range sequentialTestResults {
		status := "failed"
		if result.Passed {
			status = "passed"
		}
		fmt.Printf("%s has %s after %d tries \n", result.TestCase, status, len(result.Tries))
		for index, log := range result.Tries {
			fmt.Printf("try number %d\n", index+1)
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

	//
	// Uninstall KEDA
	//

	removal := executeTest(ctx, "tests/utils/cleanup_test.go", "15m", 1)
	fmt.Print(removal.Tries[0])
	if !removal.Passed {
		os.Exit(1)
	}

	exitCode := 0
	testResults := []TestResult{}
	testResults = append(testResults, regularTestResults...)
	testResults = append(testResults, sequentialTestResults...)
	passSummary := []string{}
	failSummary := []string{}

	for _, result := range testResults {
		if !result.Passed {
			message := fmt.Sprintf("\tExecution of %s, has failed after %d tries", result.TestCase, len(result.Tries))
			failSummary = append(failSummary, message)
			exitCode = 1
			continue
		}
		message := fmt.Sprintf("\tExecution of %s, has passed after %d tries", result.TestCase, len(result.Tries))
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

	os.Exit(exitCode)
}

func executeTest(ctx context.Context, file string, timeout string, tries int) TestResult {
	result := TestResult{
		TestCase: file,
		Passed:   false,
		Tries:    []string{},
	}
	for i := 1; i <= tries; i++ {
		fmt.Printf("Executing %s, try '%d'\n", file, i)
		cmd := exec.CommandContext(ctx, "go", "test", "-v", "-tags", "e2e", "-timeout", timeout, file)
		stdout, err := cmd.Output()
		logFile := fmt.Sprintf("%s.%d.log", file, i)
		fileError := os.WriteFile(logFile, stdout, 0644)
		if fileError != nil {
			fmt.Printf("Execution of %s, try '%d' has failed writing the logs : %s\n", file, i, fileError)
		}
		result.Tries = append(result.Tries, string(stdout))
		if err == nil {
			fmt.Printf("Execution of %s, try '%d' has passed\n", file, i)
			result.Passed = true
			break
		}
		fmt.Printf("Execution of %s, try '%d' has failed: %s \n", file, i, err)
	}
	return result
}

func getRegularTestFiles(e2eRegex string) []string {
	// We exclude utils and chaos folders and helper files
	filter := func(path string, file string) bool {
		return !strings.HasPrefix(path, "tests") ||
			strings.Contains(path, "utils") ||
			strings.Contains(path, "sequential") ||
			!strings.HasSuffix(file, "_test.go")
	}
	return getTestFiles(e2eRegex, filter)
}

func getSequentialTestFiles(e2eRegex string) []string {
	filter := func(path string, file string) bool {
		return !strings.HasPrefix(path, "tests") ||
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
			// We exclude utils and chaos folders and helper files
			if filter(path, info.Name()) {
				return nil
			}
			if regex.MatchString(info.Name()) {
				testFiles = append(testFiles, path)
			}

			return nil
		})

	if err != nil {
		return []string{}
	}

	// We randomize the executions
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(testFiles), func(i, j int) {
		testFiles[i], testFiles[j] = testFiles[j], testFiles[i]
	})

	return testFiles
}
