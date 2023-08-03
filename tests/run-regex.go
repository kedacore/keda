// +build e2e-regex

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	concurrentTests        = 15
	regularTestsTimeout    = "20m"
	regularTestsRetries    = 3
	sequentialTestsTimeout = "20m"
	sequentialTestsRetries = 1
)

func main() {
	e2eRegex := os.Getenv("E2E_TEST_REGEX")
	if e2eRegex == "" {
		e2eRegex = ".*_test.go"
	}

	//
	// Detect test cases
	//
	regularTestFiles := getRegularTestFiles(e2eRegex)
	sequentialTestFiles := getSequentialTestFiles(e2eRegex)
	if len(regularTestFiles) == 0 && len(sequentialTestFiles) == 0 {
		fmt.Printf("No test will be executed, please review your regex: '%s'\n", e2eRegex)
		os.Exit(1)
	}

	// Store the test files in environment variables
	os.Setenv("REGULAR_TEST_FILES", strings.Join(regularTestFiles, ","))
	os.Setenv("SEQUENTIAL_TEST_FILES", strings.Join(sequentialTestFiles, ","))
}

func getRegularTestFiles(e2eRegex string) []string {
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

	return testFiles
}
