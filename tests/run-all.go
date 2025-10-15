//go:build e2e
// +build e2e

package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/kedacore/keda/v2/tests/helper"
)

var (
	concurrentTests        = 25
	regularTestsTimeout    = "20m"
	regularTestsRetries    = 3
	sequentialTestsTimeout = "20m"
	sequentialTestsRetries = 2
)

type TestResult struct {
	TestCase string
	Passed   bool
	Attempts []string
}

func main() {
	ctx := context.Background()

	setAbsoluteConfigPath()

	//
	// Detect test cases
	//

	e2eRegex, err := getE2eRegex()
	if err != nil {
		fmt.Printf("Error getting e2e regex: %v\n", err)
		os.Exit(1)
	}

	regularTestFiles := getRegularTestFiles(e2eRegex)
	sequentialTestFiles := getSequentialTestFiles(e2eRegex)
	if len(regularTestFiles) == 0 && len(sequentialTestFiles) == 0 {
		fmt.Printf("No test has been executed, please review your regex: '%s'\n", e2eRegex)
		os.Exit(1)
	}

	if len(os.Args) > 1 && os.Args[1] == "regex-check" {
		return
	}

	if helper.KEDATestConfig.DryRun {
		showDryRunOutput(regularTestFiles, sequentialTestFiles, e2eRegex)
		return
	}

	//
	// Install KEDA
	//
	if helper.KEDATestConfig.KEDA.SkipSetup {
		fmt.Println("Skipping KEDA setup")
	} else {
		installation := executeTest(ctx, "tests/utils/setup_test.go", "15m", 1)
		fmt.Print(installation.Attempts[0])
		if !installation.Passed {
			printKedaLogs()
			uninstallKeda(ctx)
			os.Exit(1)
		}
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
	if helper.KEDATestConfig.KEDA.SkipCleanup {
		fmt.Println("Skipping KEDA cleanup")
	} else {
		passed := uninstallKeda(ctx)
		if !passed {
			os.Exit(1)
		}
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

	operatorLogs, err := helper.FindPodLogs(kubeClient, "keda", "app=keda-operator", true)
	if err == nil {
		fmt.Println(">>> KEDA Operator log <<<")
		fmt.Println(operatorLogs)
		fmt.Println("##############################################")
		fmt.Println("##############################################")
		saveLogToFile("keda-operator.log", operatorLogs)
	}

	msLogs, err := helper.FindPodLogs(kubeClient, "keda", "app=keda-metrics-apiserver", true)
	if err == nil {
		fmt.Println(">>> KEDA Metrics Server log <<<")
		fmt.Println(msLogs)
		fmt.Println("##############################################")
		fmt.Println("##############################################")
		saveLogToFile("keda-metrics-server.log", msLogs)
	}

	hooksLogs, err := helper.FindPodLogs(kubeClient, "keda", "app=keda-admission-webhooks", true)
	if err == nil {
		fmt.Println(">>> KEDA Admission Webhooks log <<<")
		fmt.Println(hooksLogs)
		fmt.Println("##############################################")
		fmt.Println("##############################################")
		saveLogToFile("keda-webhooks.log", hooksLogs)
	}
}

func saveLogToFile(file string, lines []string) {
	f, err := os.Create(file)
	if err != nil {
		fmt.Print(err)
	}
	defer f.Close()
	for _, line := range lines {
		_, err := f.WriteString(line + "\n")
		if err != nil {
			fmt.Print(err)
		}
	}
}

// getE2eRegex gets the regex to filter which tests to run.
// If E2E_TEST_REGEX is set, it overrides and uses that regex.
// If not, it uses a config to build a regex. If the config is nil, it uses the default regex which is to run all tests.
func getE2eRegex() (string, error) {
	// if there's a regex, use it
	e2eRegex := os.Getenv("E2E_TEST_REGEX")
	if e2eRegex != "" {
		return e2eRegex, nil
	}

	return buildRegexFromConfig(helper.KEDATestConfig)
}

// buildRegexFromConfig builds a regex string from a TestConfig
func buildRegexFromConfig(config helper.TestConfig) (string, error) {
	// if user didn't specify any categories and TestCategories is empty, but non nil, then this means there's no filter
	if len(config.TestCategories) == 0 {
		return ".*_test.go", nil
	}

	var regexParts []string

	// For each known category, we get all the available tests under the tests/category directory.
	// We then filter the tests we actually run based on the config exclude and includes.
	// Then we incrementally build a regex based on the filtered tests.
	for _, category := range config.GetAllCategories() {
		var supportedTests []string
		var err error
		categoryConfig, exists := config.TestCategories[category]
		if exists {
			supportedTests, err = getAvailableTests("tests/"+category, categoryConfig)
			if err != nil {
				return "", fmt.Errorf("error getting %q tests: %w", category, err)
			}
		}

		if len(supportedTests) > 0 {
			// go regex doesn't support negative lookaheads, so we need to explicily include the tests we want to run
			regexParts = append(regexParts, fmt.Sprintf("%s/(%s)/.*", category, strings.Join(supportedTests, "|")))
		}
		// if there's no tests for that category, we don't need to add anything to the regex
	}

	if len(regexParts) == 0 {
		return "", fmt.Errorf("no tests found for any category, but at least one include was specified. check your filters")
	}

	return fmt.Sprintf("^tests/(%s)_test\\.go$", strings.Join(regexParts, "|")), nil
}

// getAvailableTests returns all available test suites/directories for a category, as a slice of strings
func getAvailableTests(categoryPath string, categoryConfig helper.TestCategory) ([]string, error) {
	var tests []string

	err := filepath.WalkDir(categoryPath, func(testPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// don't count root directory
		if testPath == categoryPath {
			// unless there are no tests, in which case we can skip the entire walk
			if len(categoryConfig.Tests) == 0 {
				switch categoryConfig.Mode {
				case helper.TestCategoryModeInclude:
					tests = append(tests, ".*") // match everything
					return fs.SkipDir
				case helper.TestCategoryModeExclude:
					return fs.SkipDir
				default:
					// because of TestConfig.Validate(), we should never get here
					panic(fmt.Sprintf("invalid mode %q", categoryConfig.Mode))
				}
			}
		}
		if d.Name() == "helper" || d.Name() == "helpers" {
			return fs.SkipDir
		}
		if !d.IsDir() {
			return nil
		}

		// we don't want to include the root in the path for UX purposes
		// this is so that users can just specify paths like
		// "aws" or "aws/aws_cloudwatch" instead of tests/aws/aws_cloudwatch
		trimmedTestPath := strings.TrimPrefix(testPath, categoryPath+string(filepath.Separator))
		if trimmedTestPath == "" {
			return nil
		}

		switch categoryConfig.Mode {
		case helper.TestCategoryModeInclude:
			// we don't need to keep looking in this directory
			// we can save some time and regex building by just skipping early
			if slices.Contains(categoryConfig.Tests, trimmedTestPath) {
				tests = append(tests, trimmedTestPath)
				return fs.SkipDir
			}
			return nil
		case helper.TestCategoryModeExclude:
			// we don't need to keep looking in this directory
			// we can save some time and regex building by just skipping early
			if slices.Contains(categoryConfig.Tests, trimmedTestPath) {
				return fs.SkipDir
			}

			// but otherwise, we have to include this suite UNLESS it is a non-leaf directory (e.g., aws)
			// in which case, just continue because we can't prematurely include all tests within a non-leaf
			isLeaf, err := isLeafDir(testPath)
			if err != nil {
				return err
			}
			if isLeaf {
				tests = append(tests, trimmedTestPath)
			}
			return nil
		default:
			panic(fmt.Sprintf("invalid mode %q", categoryConfig.Mode))
		}
	})

	return tests, err
}

// setAbsoluteConfigPath converts the potentially relative path E2E_TEST_CONFIG environment variable to an absolute path and sets it.
// This is because this process executes sub processes (setup_test.go, etc.) and will cause the relative paths to be incorrect.
func setAbsoluteConfigPath() {
	configPath := os.Getenv("E2E_TEST_CONFIG")
	if configPath != "" {
		absConfigPath, err := filepath.Abs(configPath)
		if err != nil {
			fmt.Printf("Error resolving config path: %v\n", err)
			os.Exit(1)
		}
		os.Setenv("E2E_TEST_CONFIG", absConfigPath)
	}
}

func showDryRunOutput(regularTestFiles, sequentialTestFiles []string, e2eRegex string) {
	slices.Sort(regularTestFiles)
	slices.Sort(sequentialTestFiles)

	fmt.Println("##############################################")
	fmt.Println("##############################################")
	fmt.Println("DRY-RUN SUMMARY")
	fmt.Println("##############################################")
	fmt.Println("##############################################")

	fmt.Printf("\nConverted test filter regex: %s\n", e2eRegex)
	fmt.Printf("Total Regular Tests: %d\n", len(regularTestFiles))
	fmt.Printf("Total Sequential Tests: %d\n", len(sequentialTestFiles))
	fmt.Printf("Total Tests: %d\n", len(regularTestFiles)+len(sequentialTestFiles))

	fmt.Println("\nTests to be executed:")

	if len(regularTestFiles) > 0 {
		fmt.Println("├── Regular Tests (concurrent)")
		for i, file := range regularTestFiles {
			prefix := "│   ├── "
			if i == len(regularTestFiles)-1 && len(sequentialTestFiles) == 0 {
				prefix = "│   └── "
			}
			fmt.Printf("%s%s\n", prefix, file)
		}
	}

	if len(sequentialTestFiles) > 0 {
		fmt.Println("├── Sequential Tests")
		for i, file := range sequentialTestFiles {
			prefix := "│   ├── "
			if i == len(sequentialTestFiles)-1 {
				prefix = "│   └── "
			}
			fmt.Printf("%s%s\n", prefix, file)
		}
	}

	// Show configuration summary
	fmt.Println("\nConfiguration:")
	fmt.Printf("├── Skip Setup: %t\n", helper.KEDATestConfig.KEDA.SkipSetup)
	fmt.Printf("├── Skip Cleanup: %t\n", helper.KEDATestConfig.KEDA.SkipCleanup)
	fmt.Printf("├── Image Registry: %s\n", helper.KEDATestConfig.KEDA.ImageRegistry)
	fmt.Printf("├── Image Repo: %s\n", helper.KEDATestConfig.KEDA.ImageRepo)

	// Show test categories configuration
	if len(helper.KEDATestConfig.TestCategories) > 0 {
		fmt.Println("\nTest Categories:")
		for category, config := range helper.KEDATestConfig.TestCategories {
			fmt.Printf("├── %s: %s", category, config.Mode)
			if len(config.Tests) > 0 {
				fmt.Printf(" (%s)\n", strings.Join(config.Tests, ", "))
			} else {
				fmt.Println()
			}
		}
	}

	fmt.Println("\nThis was a dry-run. No actual tests were executed.")
}

// isLeafDir checks if a given path is a directory that contains no other directories.
func isLeafDir(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			return false, nil
		}
	}

	return true, nil
}
