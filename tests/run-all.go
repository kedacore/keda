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
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
	"gopkg.in/yaml.v2"
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
	config, err := loadTestConfig()
	if err != nil {
		return "", err
	}

	// if there's a regex, use it
	e2eRegex := os.Getenv("E2E_TEST_REGEX")
	if e2eRegex != "" {
		return e2eRegex, nil
	}

	return buildRegexFromConfig(config)
}

// loadTestConfig loads the test configuration from a file path
func loadTestConfig() (*TestConfig, error) {
	configPath := os.Getenv("E2E_TEST_CONFIG")
	if configPath == "" {
		return nil, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config TestConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// buildRegexFromConfig builds a regex string from a TestConfig
func buildRegexFromConfig(config *TestConfig) (string, error) {
	if config == nil {
		return ".*_test.go", nil
	}

	var regexParts []string

	// For each known category, we get all the available tests under the tests/category directory.
	// We then filter the tests we actually run based on the config exclude and includes.
	// Then we incrementally build a regex based on the filtered tests.
	for _, category := range config.GetAllCategories() {
		availableTests, err := getAvailableTests("tests/" + category)
		if err != nil {
			return "", fmt.Errorf("error getting %q tests: %w", category, err)
		}

		var supportedTests []string
		if categoryConfig, exists := config.TestCategories[category]; exists {
			supportedTests = filterTests(availableTests, categoryConfig)
		} else {
			// use all tests in the category if no config is specified
			supportedTests = availableTests
		}

		if len(supportedTests) > 0 {
			// go regex doesn't support negative lookaheads, so we need to explicily include the tests we want to run
			regexParts = append(regexParts, fmt.Sprintf("%s/(%s)/.*", category, strings.Join(supportedTests, "|")))
		}
	}

	if len(regexParts) == 0 {
		return ".*_test.go", nil
	}

	return fmt.Sprintf("^tests/(%s)_test\\.go$", strings.Join(regexParts, "|")), nil
}

// getAvailableTests returns all available test suites/directories for a category, as a slice of strings
func getAvailableTests(categoryPath string) ([]string, error) {
	entries, err := os.ReadDir(categoryPath)
	if err != nil {
		return nil, err
	}

	var tests []string
	for _, entry := range entries {
		if entry.IsDir() {
			tests = append(tests, entry.Name())
		}
	}
	return tests, nil
}

// filterTests returns the filtered tests as a slice of strings, based on the TestCategory
func filterTests(tests []string, category TestCategory) []string {
	var result []string
	// If the mode is include, we include the tests that are listed in category.Tests, including all if empty.
	// If the mode is exclude, we exclude the tests that are listed in category.Tests, excluding all if empty.
	switch category.Mode {
	case TestCategoryModeInclude:
		if len(category.Tests) == 0 {
			return tests
		}
		for _, test := range category.Tests {
			if slices.Contains(tests, test) {
				result = append(result, test)
			}
		}
		return result
	case TestCategoryModeExclude:
		if len(category.Tests) == 0 {
			return []string{}
		}
		for _, test := range tests {
			if !slices.Contains(category.Tests, test) {
				result = append(result, test)
			}
		}
		return result
	default:
		// because of TestConfig.Validate(), we should never get here
		panic(fmt.Sprintf("invalid mode %q", category.Mode))
	}
}

type TestConfig struct {
	TestCategories map[string]TestCategory `yaml:"test_categories"`
}

func (tc *TestConfig) GetAllCategories() []string {
	return []string{"internals", "secret-providers", "sequential", "scalers"}
}

// Validate enforces that all categories have a mode, and that the mode is either include or exclude.
func (tc *TestConfig) Validate() error {
	// validate that test_categories exists
	if tc.TestCategories == nil {
		return fmt.Errorf("test_categories is a required field")
	}

	// validate that all categories have a mode, and that the mode is either include or exclude
	for name, cat := range tc.TestCategories {
		if cat.Mode == "" {
			return fmt.Errorf("category %q: mode is a required field", name)
		}
		switch cat.Mode {
		case TestCategoryModeInclude, TestCategoryModeExclude:
		default:
			return fmt.Errorf("category %q: invalid mode %q", name, cat.Mode)
		}
	}
	return nil
}

type TestCategory struct {
	Mode  TestCategoryMode `yaml:"mode"`
	Tests []string         `yaml:"tests,omitempty"`
}

type TestCategoryMode string

const (
	TestCategoryModeInclude TestCategoryMode = "include"
	TestCategoryModeExclude TestCategoryMode = "exclude"
)
