package helper

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

var KEDATestConfig = TestConfig{
	KEDA: &KEDAConfig{
		InstallKeda:   true,
		InstallKafka:  true,
		ImageRegistry: "", // default to Makefile settings
		ImageRepo:     "", // default to Makefile settings
	},
	TestCategories: map[string]TestCategory{},
}

type TestConfig struct {
	KEDA           *KEDAConfig             `yaml:"keda"`
	TestCategories map[string]TestCategory `yaml:"testCategories"`
}

type KEDAConfig struct {
	InstallKeda   bool   `yaml:"installKeda"`
	InstallKafka  bool   `yaml:"installKafka"`
	ImageRegistry string `yaml:"imageRegistry"`
	ImageRepo     string `yaml:"imageRepo"`
}

func (tc *TestConfig) GetAllCategories() []string {
	return []string{"internals", "secret-providers", "sequential", "scalers"}
}

// Validate enforces that all categories have a mode, and that the mode is either include or exclude.
func (tc *TestConfig) Validate() error {
	// validate that testCategories exists
	if tc.TestCategories == nil {
		return fmt.Errorf("testCategories is a required field")
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

// LoadTestConfig builds a TestConfig and sets the global helper.TestConfig variable.
// If the env var is not set, then it is initialized to a default config.
func LoadTestConfig() error {
	configPath := os.Getenv("E2E_TEST_CONFIG")
	if configPath == "" {
		return nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var config TestConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return err
	}
	if err := config.Validate(); err != nil {
		return err
	}

	KEDATestConfig = config
	return nil
}
