// nosemgrep
package scalers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

// Mock ScalerConfig for tests
func newTestScalerConfig(metadata map[string]string, triggerIndex int, metricType v2.MetricTargetType) *scalersconfig.ScalerConfig {
	resolvedEnv := make(map[string]string)

	// For WeatherAPIKeyFromEnv (now keda:"name=weatherApiKey,order=triggerMetadata,optional")
	// its value will be metadata["weatherApiKey"].
	// We need to ensure that if metadata["weatherApiKey"] specifies an env var name,
	// that name exists as a key in ResolvedEnv for later lookup of the actual key value.
	if envVarNameForWeatherKey, ok := metadata["weatherApiKey"]; ok && envVarNameForWeatherKey != "" {
		resolvedEnv[envVarNameForWeatherKey] = "dummy_weather_key_value_from_env"
	}

	// For DemandAPIKeyFromEnv (keda:"name=demandApiKeyFromEnv,order=triggerMetadata,optional")
	// its value will be metadata["demandApiKeyFromEnv"].
	// If this were to be used to look up a value in ResolvedEnv (it isn't directly by TypedConfig for order=triggerMetadata),
	// similar logic would apply.
	if envVarNameForDemandKey, ok := metadata["demandApiKeyFromEnv"]; ok && envVarNameForDemandKey != "" {
		// Although not strictly necessary for TypedConfig for this field, if our code later uses
		// meta.DemandAPIKeyFromEnv to look up in ResolvedEnv, this pre-populates it.
		resolvedEnv[envVarNameForDemandKey] = "dummy_demand_key_value_from_env"
	}

	return &scalersconfig.ScalerConfig{
		TriggerMetadata:         metadata,
		TriggerIndex:            triggerIndex,
		MetricType:              metricType,
		GlobalHTTPTimeout:       3000 * time.Millisecond, // Default KEDA global HTTP timeout
		ResolvedEnv:             resolvedEnv,
		AuthParams:              make(map[string]string),
		ScalableObjectName:      "test-scaledobject",
		ScalableObjectNamespace: "test-namespace",
		ScalableObjectType:      "ScaledObject",
		Recorder:                nil, // Mock recorder if needed for event testing
	}
}

type testWeatherAwareDemandScalerMetadata struct {
	metadata   map[string]string
	metricType v2.MetricTargetType
	expected   *weatherAwareDemandScalerMetadata
	hasError   bool
}

var weatherAwareDemandScalerMetadataTestDataset = []testWeatherAwareDemandScalerMetadata{
	{ // Basic valid case
		metadata: map[string]string{
			"weatherApiEndpoint":     "http://weather.example.com",
			"weatherLocation":        "London,UK",
			"demandApiEndpoint":      "http://demand.example.com",
			"demandJsonPath":         "{.value}",
			"targetDemandPerReplica": "50",
			"activationDemandLevel":  "5",
			"metricName":             "custom-ride-demand",
			// No "weatherApiKey" here. WeatherAPIKeyFromEnv is optional & order=triggerMetadata.
		},
		metricType: v2.AverageValueMetricType,
		expected: &weatherAwareDemandScalerMetadata{
			WeatherAPIEndpoint:       "http://weather.example.com",
			WeatherAPIKeyFromEnv:     "", // Should be empty as not in metadata["weatherApiKey"]
			WeatherLocation:          "London,UK",
			WeatherUnits:             "metric",
			DemandAPIEndpoint:        "http://demand.example.com",
			DemandJSONPath:           "{.value}",
			TargetDemandPerReplica:   50,
			ActivationDemandLevel:    5,
			MetricName:               "custom-ride-demand",
			WeatherEffectScaleFactor: 1.0,
		},
		hasError: false,
	},
	{ // All optional fields provided
		metadata: map[string]string{
			"weatherApiEndpoint":       "http://weather.example.com",
			"weatherApiKey":            "WEATHER_API_KEY", // This will populate WeatherAPIKeyFromEnv
			"weatherLocation":          "NewYork,US",
			"weatherUnits":             "imperial",
			"badWeatherConditions":     "temp_below:32,rain_above:0.5",
			"demandApiEndpoint":        "http://demand.example.com/v2",
			"demandApiKeyFromEnv":      "DEMAND_API_KEY",
			"demandJsonPath":           "{.data.demand_level}",
			"targetDemandPerReplica":   "20",
			"activationDemandLevel":    "2",
			"weatherEffectScaleFactor": "1.75",
			"metricName":               "nyc-demand",
		},
		metricType: v2.ValueMetricType,
		expected: &weatherAwareDemandScalerMetadata{
			WeatherAPIEndpoint:       "http://weather.example.com",
			WeatherAPIKeyFromEnv:     "WEATHER_API_KEY", // Correctly gets from metadata["weatherApiKey"]
			WeatherLocation:          "NewYork,US",
			WeatherUnits:             "imperial",
			BadWeatherConditions:     "temp_below:32,rain_above:0.5",
			DemandAPIEndpoint:        "http://demand.example.com/v2",
			DemandAPIKeyFromEnv:      "DEMAND_API_KEY",
			DemandJSONPath:           "{.data.demand_level}",
			TargetDemandPerReplica:   20,
			ActivationDemandLevel:    2,
			WeatherEffectScaleFactor: 1.75,
			MetricName:               "nyc-demand",
		},
		hasError: false,
	},
	{ // Malformed number for targetDemandPerReplica (TypedConfig error)
		metadata: map[string]string{
			"demandApiEndpoint":      "http://demand.example.com", // Needed to pass initial validation if TypedConfig didn't fail first
			"targetDemandPerReplica": "not-a-number",
		},
		metricType: v2.AverageValueMetricType,
		expected:   nil,
		hasError:   true,
	},
	{ // Missing both API endpoints (Validate error)
		metadata: map[string]string{
			"targetDemandPerReplica": "10", // Valid to pass TypedConfig
			"activationDemandLevel":  "5",
		},
		metricType: v2.AverageValueMetricType,
		expected:   nil,
		hasError:   true,
	},
	{ // Weather API endpoint provided, but weatherLocation missing (Validate error)
		metadata: map[string]string{
			"weatherApiEndpoint":     "http://weather.example.com",
			"targetDemandPerReplica": "10",
		},
		metricType: v2.AverageValueMetricType,
		expected:   nil,
		hasError:   true,
	},
	{ // targetDemandPerReplica is 0 (Validate error)
		metadata: map[string]string{
			"demandApiEndpoint":      "http://demand.example.com",
			"targetDemandPerReplica": "0",
		},
		metricType: v2.AverageValueMetricType,
		expected:   nil,
		hasError:   true,
	},
	{ // targetDemandPerReplica is negative (Validate error)
		metadata: map[string]string{
			"demandApiEndpoint":      "http://demand.example.com",
			"targetDemandPerReplica": "-10",
		},
		metricType: v2.AverageValueMetricType,
		expected:   nil,
		hasError:   true,
	},
	{ // weatherEffectScaleFactor is 0 (Validate error)
		metadata: map[string]string{
			"demandApiEndpoint":        "http://demand.example.com",
			"weatherEffectScaleFactor": "0",
		},
		metricType: v2.AverageValueMetricType,
		expected:   nil,
		hasError:   true,
	},
	{ // weatherEffectScaleFactor is negative (Validate error)
		metadata: map[string]string{
			"demandApiEndpoint":        "http://demand.example.com",
			"weatherEffectScaleFactor": "-1.5",
		},
		metricType: v2.AverageValueMetricType,
		expected:   nil,
		hasError:   true,
	},
	{ // Valid case: Only Demand API endpoint provided (ensure no weatherApiKey needed here for this specific test)
		metadata: map[string]string{
			"demandApiEndpoint":      "http://demand.example.com",
			"demandJsonPath":         "{.value}",
			"targetDemandPerReplica": "30",
			"activationDemandLevel":  "3",
		},
		metricType: v2.AverageValueMetricType,
		expected: &weatherAwareDemandScalerMetadata{
			DemandAPIEndpoint:        "http://demand.example.com",
			DemandJSONPath:           "{.value}",
			TargetDemandPerReplica:   30,
			ActivationDemandLevel:    3,
			WeatherUnits:             "metric",
			WeatherEffectScaleFactor: 1.0,
			MetricName:               "weather-aware-ride-demand",
		},
		hasError: false,
	},
	{ // Valid case: Only Weather API endpoint provided (with location)
		metadata: map[string]string{
			"weatherApiEndpoint":     "http://weather.example.com",
			"weatherApiKey":          "ONLY_WEATHER_KEY", // This will populate WeatherAPIKeyFromEnv
			"weatherLocation":        "Paris,FR",
			"targetDemandPerReplica": "25",
			"activationDemandLevel":  "2",
		},
		metricType: v2.ValueMetricType,
		expected: &weatherAwareDemandScalerMetadata{
			WeatherAPIEndpoint:       "http://weather.example.com",
			WeatherAPIKeyFromEnv:     "ONLY_WEATHER_KEY", // Correctly gets from metadata["weatherApiKey"]
			WeatherLocation:          "Paris,FR",
			WeatherUnits:             "metric",
			TargetDemandPerReplica:   25,
			ActivationDemandLevel:    2,
			WeatherEffectScaleFactor: 1.0,
			MetricName:               "weather-aware-ride-demand",
		},
		hasError: false,
	},
}

func TestNewWeatherAwareDemandScaler(t *testing.T) {
	for i, testData := range weatherAwareDemandScalerMetadataTestDataset {
		t.Run(fmt.Sprintf("TestNewWeatherAwareDemandScaler_%d", i), func(t *testing.T) {
			config := newTestScalerConfig(testData.metadata, 0, testData.metricType)
			scaler, err := NewWeatherAwareDemandScaler(config)

			if testData.hasError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if scaler == nil {
				t.Fatalf("Scaler is nil")
			}

			typedScaler, ok := scaler.(*weatherAwareDemandScaler)
			if !ok {
				t.Fatalf("Scaler is not of type *weatherAwareDemandScaler")
			}

			// Update expected triggerIndex and triggerMetadata based on config
			// NewWeatherAwareDemandScaler copies config.TriggerMetadata to typedScaler.metadata.triggerMetadata
			if testData.expected != nil { // only if we don't expect an error
				testData.expected.triggerIndex = config.TriggerIndex
				// Create a new map for expected.triggerMetadata and copy values
				// This is important because the scaler internally sets its own map instance.
				expectedTriggerMeta := make(map[string]string)
				for k, v := range config.TriggerMetadata {
					expectedTriggerMeta[k] = v
				}
				testData.expected.triggerMetadata = expectedTriggerMeta
			}

			if !reflect.DeepEqual(typedScaler.metadata, testData.expected) {
				t.Errorf("Metadata mismatch:\nGot:      %+v\nExpected: %+v", typedScaler.metadata, testData.expected)
			}

			if typedScaler.metricType != testData.metricType {
				t.Errorf("MetricType mismatch: Got %s, Expected %s", typedScaler.metricType, testData.metricType)
			}
		})
	}
}

type testIsBadWeatherCase struct {
	name                 string
	badWeatherConditions string
	weatherData          map[string]interface{}
	logger               logr.Logger
	expectedIsBad        bool
	expectedError        bool
}

var isBadWeatherTestDataset = []testIsBadWeatherCase{
	{
		name:                 "GoodWeather_EmptyConditions",
		badWeatherConditions: "",
		weatherData:          map[string]interface{}{"temp": 20.0, "rain": 0.0},
		logger:               logr.Discard(),
		expectedIsBad:        false,
		expectedError:        false,
	},
	{
		name:                 "GoodWeather_NilData",
		badWeatherConditions: "temp_below:0",
		weatherData:          nil,
		logger:               logr.Discard(),
		expectedIsBad:        false,
		expectedError:        false,
	},
	{
		name:                 "BadWeather_TempBelow",
		badWeatherConditions: "temp_below:0",
		weatherData:          map[string]interface{}{"temp": -5.0},
		logger:               logr.Discard(),
		expectedIsBad:        true,
		expectedError:        false,
	},
	{
		name:                 "GoodWeather_TempAbove",
		badWeatherConditions: "temp_below:0",
		weatherData:          map[string]interface{}{"temp": 5.0},
		logger:               logr.Discard(),
		expectedIsBad:        false,
		expectedError:        false,
	},
	{
		name:                 "BadWeather_RainAbove",
		badWeatherConditions: "rain_above:5",
		weatherData:          map[string]interface{}{"rain": 10.0},
		logger:               logr.Discard(),
		expectedIsBad:        true,
		expectedError:        false,
	},
	{
		name:                 "GoodWeather_RainBelow",
		badWeatherConditions: "rain_above:5",
		weatherData:          map[string]interface{}{"rain": 2.0},
		logger:               logr.Discard(),
		expectedIsBad:        false,
		expectedError:        false,
	},
	{
		name:                 "BadWeather_Combined_TempAndRain_TempTriggers",
		badWeatherConditions: "temp_below:0,rain_above:5,wind_above:20",
		weatherData:          map[string]interface{}{"temp": -2.0, "rain": 2.0, "wind": 10.0},
		logger:               logr.Discard(),
		expectedIsBad:        true,
		expectedError:        false,
	},
	{
		name:                 "BadWeather_Combined_TempAndRain_RainTriggers",
		badWeatherConditions: "temp_below:0,rain_above:5,wind_above:20",
		weatherData:          map[string]interface{}{"temp": 2.0, "rain": 10.0, "wind": 10.0},
		logger:               logr.Discard(),
		expectedIsBad:        true,
		expectedError:        false,
	},
	{
		name:                 "GoodWeather_Combined_NoTrigger",
		badWeatherConditions: "temp_below:0,rain_above:5,wind_above:20",
		weatherData:          map[string]interface{}{"temp": 5.0, "rain": 2.0, "wind": 10.0},
		logger:               logr.Discard(),
		expectedIsBad:        false,
		expectedError:        false,
	},
	{
		name:                 "Error_MalformedCondition_MissingValue",
		badWeatherConditions: "temp_below", // Missing value
		weatherData:          map[string]interface{}{"temp": -5.0},
		logger:               logr.Discard(),
		expectedIsBad:        false,
		expectedError:        true,
	},
	{
		name:                 "Error_MalformedCondition_InvalidRuleFormat", // Added based on implementation detail
		badWeatherConditions: "temp-equals:0",
		weatherData:          map[string]interface{}{"temp": -5.0},
		logger:               logr.Discard(),
		expectedIsBad:        false,
		expectedError:        true,
	},
	{
		name:                 "Error_MalformedThreshold",
		badWeatherConditions: "temp_below:abc",
		weatherData:          map[string]interface{}{"temp": -5.0},
		logger:               logr.Discard(),
		expectedIsBad:        false,
		expectedError:        true,
	},
	{
		name:                 "Skipped_KeyNotInWeatherData",
		badWeatherConditions: "temp_below:0,snow_above:1", // snow key not in data
		weatherData:          map[string]interface{}{"temp": 5.0, "rain": 0.0},
		logger:               logr.Discard(),
		expectedIsBad:        false, // because temp is not below 0, and snow is skipped
		expectedError:        false,
	},
	{
		name:                 "Error_WeatherDataNotNumber",
		badWeatherConditions: "temp_below:0",
		weatherData:          map[string]interface{}{"temp": "not-a-number"},
		logger:               logr.Discard(),
		expectedIsBad:        false,
		expectedError:        true,
	},
}

func TestIsBadWeather(t *testing.T) {
	// Create a dummy scaler instance for testing isBadWeather
	// Metadata within this dummy scaler is what isBadWeather will use.
	dummyMetadata := &weatherAwareDemandScalerMetadata{} // Will be updated per test case
	dummyScaler := &weatherAwareDemandScaler{metadata: dummyMetadata, logger: logr.Discard()}

	for _, tc := range isBadWeatherTestDataset {
		t.Run(tc.name, func(t *testing.T) {
			// Update the BadWeatherConditions in the dummy scaler's metadata for each test case
			dummyScaler.metadata.BadWeatherConditions = tc.badWeatherConditions
			dummyScaler.logger = tc.logger // Assign logger from test case

			isBad, err := dummyScaler.isBadWeather(tc.weatherData)

			if tc.expectedError {
				if err == nil {
					t.Errorf("Expected an error, but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Did not expect an error, but got: %v", err)
				}
				if isBad != tc.expectedIsBad {
					t.Errorf("Expected isBad to be %t, but got %t", tc.expectedIsBad, isBad)
				}
			}
		})
	}
}

type testGetMetricsCase struct {
	name                 string
	metadata             map[string]string
	demandAPIResponse    string
	demandAPIStatusCode  int
	demandAPIError       bool // simulate a network error for demand API
	weatherAPIResponse   string
	weatherAPIStatusCode int
	weatherAPIError      bool // simulate a network error for weather API
	metricType           v2.MetricTargetType
	triggerIndex         int
	expectedMetricValue  float64
	expectedActive       bool
	expectError          bool
	expectedErrorMessage string // substring to check in error message
	metricIdentifier     string // metric name passed to GetMetricsAndActivity
}

// Simplified: For this subtask, API key resolution from env is not deeply tested in GetMetricsAndActivity mocks.
// The NewWeatherAwareDemandScaler test already covers that metadata fields for keys are parsed.
// The fetchJSONData helper has a basic API key logic that relies on triggerMetadata.

var getMetricsTestDataset = []testGetMetricsCase{
	{
		name: "GoodDemand_GoodWeather_Active",
		metadata: map[string]string{
			"weatherApiEndpoint":     "placeholder", // Will be replaced by mock
			"weatherApiKey":          "GOOD_WEATHER_KEY", // Added for explicit env var name
			"weatherLocation":        "testville",
			"badWeatherConditions":   "temp_below:0",
			"demandApiEndpoint":      "placeholder", // Will be replaced by mock
			"demandJsonPath":         "{.value}",
			"targetDemandPerReplica": "50",
			"activationDemandLevel":  "10",
			"metricName":             "test-metric",
		},
		demandAPIResponse:    `{"value": 20}`,
		demandAPIStatusCode:  http.StatusOK,
		weatherAPIResponse:   `{"temp": 10.0}`, // Good weather
		weatherAPIStatusCode: http.StatusOK,
		metricType:           v2.AverageValueMetricType,
		triggerIndex:         0,
		expectedMetricValue:  20,
		expectedActive:       true,
		expectError:          false,
		metricIdentifier:     "s0-test-metric",
	},
	{
		name: "HighDemand_BadWeather_ScaledActive",
		metadata: map[string]string{
			"weatherApiEndpoint":       "placeholder",
			"weatherApiKey":            "BAD_WEATHER_KEY", // Added
			"weatherLocation":          "coldplace",
			"badWeatherConditions":     "temp_below:0",
			"weatherEffectScaleFactor": "2.0",
			"demandApiEndpoint":        "placeholder",
			"demandJsonPath":           "{.current.demand}",
			"targetDemandPerReplica":   "50",
			"activationDemandLevel":    "10",
			"metricName":               "test-metric-bad",
		},
		demandAPIResponse:    `{"current": {"demand": 30}}`,
		demandAPIStatusCode:  http.StatusOK,
		weatherAPIResponse:   `{"temp": -5.0}`, // Bad weather
		weatherAPIStatusCode: http.StatusOK,
		metricType:           v2.AverageValueMetricType,
		triggerIndex:         1,
		expectedMetricValue:  60, // 30 * 2.0
		expectedActive:       true,
		expectError:          false,
		metricIdentifier:     "s1-test-metric-bad",
	},
	{
		name: "LowDemand_GoodWeather_Inactive",
		metadata: map[string]string{
			"weatherApiEndpoint":     "placeholder",
			"weatherApiKey":          "LOW_GOOD_WEATHER_KEY", // Added
			"weatherLocation":        "warmplace",
			"badWeatherConditions":   "temp_below:0",
			"demandApiEndpoint":      "placeholder",
			"demandJsonPath":         "{.val}",
			"targetDemandPerReplica": "100",
			"activationDemandLevel":  "50",
			"metricName":             "test-metric-inactive",
		},
		demandAPIResponse:    `{"val": 5}`,
		demandAPIStatusCode:  http.StatusOK,
		weatherAPIResponse:   `{"temp": 25.0}`, // Good weather
		weatherAPIStatusCode: http.StatusOK,
		metricType:           v2.AverageValueMetricType,
		triggerIndex:         2,
		expectedMetricValue:  5,
		expectedActive:       false,
		expectError:          false,
		metricIdentifier:     "s2-test-metric-inactive",
	},
	{
		name: "DemandAPIError",
		metadata: map[string]string{
			"demandApiEndpoint":     "placeholder", // Will cause fetch attempt
			"demandJsonPath":        "{.val}",
			"activationDemandLevel": "10",
			"metricName":            "test-metric-demand-err",
			"weatherApiEndpoint":    "placeholder", // For good weather
			"weatherApiKey":         "DEMAND_ERR_WEATHER_KEY", // Added
			"weatherLocation":       "anyplace",    // Required for weatherApiEndpoint to be valid
		},
		demandAPIError:       true, // Simulate network error
		weatherAPIResponse:   `{"temp": 10.0}`,
		weatherAPIStatusCode: http.StatusOK,
		metricType:           v2.AverageValueMetricType,
		triggerIndex:         3,
		expectError:          true,
		expectedErrorMessage: "error fetching demand data",
		metricIdentifier:     "s3-test-metric-demand-err",
	},
	{
		name: "WeatherAPIError_ProceedsAsGoodWeather",
		metadata: map[string]string{
			"weatherApiEndpoint":       "placeholder", // Will cause fetch attempt
			"weatherApiKey":            "WEATHER_ERR_KEY", // Added
			"weatherLocation":          "anywhere",
			"badWeatherConditions":     "temp_below:0",
			"weatherEffectScaleFactor": "2.0",
			"demandApiEndpoint":        "placeholder",
			"demandJsonPath":           "{.value}",
			"activationDemandLevel":    "10",
			"metricName":               "test-metric-weather-err",
		},
		demandAPIResponse:   `{"value": 20}`,
		demandAPIStatusCode: http.StatusOK,
		weatherAPIError:     true, // Simulate network error
		metricType:          v2.AverageValueMetricType,
		triggerIndex:        4,
		expectedMetricValue: 20, // Not scaled
		expectedActive:      true,
		expectError:         false,
		metricIdentifier:    "s4-test-metric-weather-err",
	},
	{
		name: "DemandJSONPathError",
		metadata: map[string]string{
			"demandApiEndpoint":     "placeholder",
			"demandJsonPath":        "{.nonexistent}", // Path that won't find data
			"activationDemandLevel": "10",
			"metricName":            "test-metric-jsonpath-err",
			"weatherApiEndpoint":    "placeholder",
			"weatherApiKey":         "JSON_PATH_WEATHER_KEY", // Added
			"weatherLocation":       "anyplace", // Required for weatherApiEndpoint to be valid
		},
		demandAPIResponse:    `{"value": 20}`,
		demandAPIStatusCode:  http.StatusOK,
		weatherAPIResponse:   `{"temp": 10.0}`,
		weatherAPIStatusCode: http.StatusOK,
		metricType:           v2.AverageValueMetricType,
		triggerIndex:         5,
		expectError:          true,
		expectedErrorMessage: "error extracting demand value",
		metricIdentifier:     "s5-test-metric-jsonpath-err",
	},
}

func TestGetMetricsAndActivity(t *testing.T) {
	for _, tc := range getMetricsTestDataset {
		t.Run(tc.name, func(t *testing.T) {
			// Mock Demand API Server
			demandServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.demandAPIError {
					http.Error(w, "demand API unavailable", http.StatusServiceUnavailable)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.demandAPIStatusCode)
				if tc.demandAPIResponse != "" {
					// nosemgrep: no-direct-write-to-responsewriter
					_, _ = w.Write([]byte(tc.demandAPIResponse))
				}
			}))
			defer demandServer.Close()

			// Mock Weather API Server
			weatherServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.weatherAPIError {
					http.Error(w, "weather API unavailable", http.StatusServiceUnavailable)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.weatherAPIStatusCode)
				if tc.weatherAPIResponse != "" {
					// nosemgrep: no-direct-write-to-responsewriter
					_, _ = w.Write([]byte(tc.weatherAPIResponse))
				}
			}))
			defer weatherServer.Close()

			currentMetadata := make(map[string]string)
			for k, v := range tc.metadata {
				currentMetadata[k] = v
			}

			// Set demandApiEndpoint only if it's part of the test case's intent
			if _, present := tc.metadata["demandApiEndpoint"]; present || tc.demandAPIError {
				currentMetadata["demandApiEndpoint"] = demandServer.URL
			} else {
				delete(currentMetadata, "demandApiEndpoint") // Ensure it's not set if not intended
			}

			// Set weatherApiEndpoint only if it's part of the test case's intent
			if _, present := tc.metadata["weatherApiEndpoint"]; present || tc.weatherAPIError {
				currentMetadata["weatherApiEndpoint"] = weatherServer.URL
			} else {
				delete(currentMetadata, "weatherApiEndpoint") // Ensure it's not set if not intended
			}

			config := newTestScalerConfig(currentMetadata, tc.triggerIndex, tc.metricType)

			scaler, err := NewWeatherAwareDemandScaler(config)
			if err != nil {
				t.Fatalf("Error creating scaler: %v", err)
			}
			typedScaler := scaler.(*weatherAwareDemandScaler)
			// Logger is initialized in NewWeatherAwareDemandScaler

			metrics, active, err := typedScaler.GetMetricsAndActivity(context.TODO(), tc.metricIdentifier)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tc.expectedErrorMessage != "" && !strings.Contains(err.Error(), tc.expectedErrorMessage) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tc.expectedErrorMessage, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if active != tc.expectedActive {
				t.Errorf("Expected active status %t, got %t", tc.expectedActive, active)
			}

			if len(metrics) != 1 {
				// If inactive and value is 0, KEDA might return 0 metrics or 1 metric with value 0.
				// The current implementation returns 1 metric even for 0.
				// If it's not active and we expect 0, it's fine if metrics are empty or value is 0.
				if !(tc.expectedMetricValue == 0 && !tc.expectedActive && len(metrics) == 0) {
					t.Fatalf("Expected 1 metric value, got %d", len(metrics))
				}
			}

			if len(metrics) == 1 { // Only check value if a metric was returned
				expectedQuantity := resource.NewQuantity(int64(tc.expectedMetricValue), resource.DecimalSI)
				if metrics[0].Value.Cmp(*expectedQuantity) != 0 {
					t.Errorf("Expected metric value %v, got %v", expectedQuantity.String(), metrics[0].Value.String())
				}

				if metrics[0].MetricName != tc.metricIdentifier {
					t.Errorf("Expected metric name %s, got %s", tc.metricIdentifier, metrics[0].MetricName)
				}
			} else if tc.expectedMetricValue != 0 || tc.expectedActive { // If no metrics, but we expected a value or active
				t.Errorf("Expected metrics but got none. Expected value: %f, active: %t", tc.expectedMetricValue, tc.expectedActive)
			}
		})
	}
}

type testGetMetricSpecCase struct {
	name               string
	metadata           map[string]string
	metricType         v2.MetricTargetType // From ScalerConfig
	triggerIndex       int
	expectedMetricName string // Expected fully qualified metric name
	expectedTarget     v2.MetricTarget
}

var getMetricSpecTestDataset = []testGetMetricSpecCase{
	{
		name: "AverageValue_Simple",
		metadata: map[string]string{
			"metricName":             "my-demand",
			"targetDemandPerReplica": "100",
			"demandApiEndpoint":      "http://dummy.com", // Added to pass validation
		},
		metricType:         v2.AverageValueMetricType,
		triggerIndex:       0,
		expectedMetricName: "s0-my-demand",
		expectedTarget: v2.MetricTarget{
			Type:         v2.AverageValueMetricType,
			AverageValue: resource.NewQuantity(100, resource.DecimalSI),
		},
	},
	{
		name: "ValueMetricType_WithNormalization",
		metadata: map[string]string{
			"metricName":             "My Custom Metric", // Needs normalization
			"targetDemandPerReplica": "75",
			"demandApiEndpoint":      "http://dummy.com", // Added to pass validation
		},
		metricType:         v2.ValueMetricType,
		triggerIndex:       1,
		expectedMetricName: "s1-My Custom Metric", // NormalizeString doesn't change spaces or case
		expectedTarget: v2.MetricTarget{
			Type:  v2.ValueMetricType,
			Value: resource.NewQuantity(75, resource.DecimalSI),
		},
	},
	{
		name: "DefaultMetricName",
		metadata: map[string]string{
			// metricName not provided, should use default "weather-aware-ride-demand"
			"targetDemandPerReplica": "120",
			"demandApiEndpoint":      "http://dummy.com", // Added to pass validation
		},
		metricType:         v2.AverageValueMetricType,
		triggerIndex:       2,
		expectedMetricName: "s2-weather-aware-ride-demand",
		expectedTarget: v2.MetricTarget{
			Type:         v2.AverageValueMetricType,
			AverageValue: resource.NewQuantity(120, resource.DecimalSI),
		},
	},
}

func TestWeatherAwareDemandScalerGetMetricSpecForScaling(t *testing.T) {
	for _, tc := range getMetricSpecTestDataset {
		t.Run(tc.name, func(t *testing.T) {
			config := newTestScalerConfig(tc.metadata, tc.triggerIndex, tc.metricType)
			scaler, err := NewWeatherAwareDemandScaler(config)
			if err != nil {
				t.Fatalf("Error creating scaler: %v", err)
			}

			metricSpecs := scaler.GetMetricSpecForScaling(context.TODO())
			if len(metricSpecs) != 1 {
				t.Fatalf("Expected 1 metric spec, got %d", len(metricSpecs))
			}

			spec := metricSpecs[0]
			if spec.External == nil {
				t.Fatalf("spec.External is nil")
			}

			if spec.External.Metric.Name != tc.expectedMetricName {
				t.Errorf("Expected metric name %s, got %s", tc.expectedMetricName, spec.External.Metric.Name)
			}

			if spec.External.Target.Type != tc.expectedTarget.Type {
				t.Errorf("Expected target type %s, got %s", tc.expectedTarget.Type, spec.External.Target.Type)
			}

			// Check AverageValue
			if tc.expectedTarget.AverageValue != nil {
				if spec.External.Target.AverageValue == nil {
					t.Errorf("Expected AverageValue %v, got nil", tc.expectedTarget.AverageValue.String())
				} else if spec.External.Target.AverageValue.Cmp(*tc.expectedTarget.AverageValue) != 0 {
					t.Errorf("Expected AverageValue %v, got %v", tc.expectedTarget.AverageValue.String(), spec.External.Target.AverageValue.String())
				}
			} else if spec.External.Target.AverageValue != nil {
				t.Errorf("Expected AverageValue to be nil, got %v", spec.External.Target.AverageValue.String())
			}

			// Check Value
			if tc.expectedTarget.Value != nil {
				if spec.External.Target.Value == nil {
					t.Errorf("Expected Value %v, got nil", tc.expectedTarget.Value.String())
				} else if spec.External.Target.Value.Cmp(*tc.expectedTarget.Value) != 0 {
					t.Errorf("Expected Value %v, got %v", tc.expectedTarget.Value.String(), spec.External.Target.Value.String())
				}
			} else if spec.External.Target.Value != nil {
				t.Errorf("Expected Value to be nil, got %v", spec.External.Target.Value.String())
			}

			if spec.Type != v2.ExternalMetricSourceType {
				t.Errorf("Expected spec type %s, got %s", v2.ExternalMetricSourceType, spec.Type)
			}
		})
	}
}
