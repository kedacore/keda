package scalers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

// Simple mock logger for tests
type simpleTestLogger struct {
	logr.LogSink // Embed to satisfy the interface minimally
	// We can add fields here to capture logs if needed for assertions
}

func (l *simpleTestLogger) Init(info logr.RuntimeInfo) { /* no-op */ }
func (l *simpleTestLogger) Enabled(level int) bool     { return true } // Enable all levels for testing
func (l *simpleTestLogger) Info(level int, msg string, keysAndValues ...interface{}) {
	fmt.Printf("[TEST LOGGER INFO] %s %v\n", msg, keysAndValues)
}
func (l *simpleTestLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	fmt.Printf("[TEST LOGGER ERROR] %s: %v %v\n", msg, err, keysAndValues)
}
func (l *simpleTestLogger) WithName(name string) logr.LogSink {
	// In a real complex scenario, you might return a new sink with the name.
	// For many tests, returning self is fine if name context isn't asserted.
	return l
}
func (l *simpleTestLogger) WithValues(keysAndValues ...interface{}) logr.LogSink {
	return l // Similar to WithName, return self for simplicity here.
}

func newSimpleTestLogger() logr.Logger {
	return logr.New(&simpleTestLogger{})
}

var testWeatherAwareDemandResolvedEnv = map[string]string{
	"weatherApiKey": "test_weather_key",
	"demandApiKey":  "test_demand_key",
}

type weatherAwareDemandScalerTestConfig struct {
	name               string
	metadata           map[string]string
	resolvedEnv        map[string]string
	isError            bool
	expectedErrMessage string
}

var weatherAwareDemandScalerTestConfigs = []weatherAwareDemandScalerTestConfig{
	{
		name: "valid both endpoints",
		metadata: map[string]string{
			"weatherApiEndpoint":     "http://weather.api",
			"weatherLocation":        "london,uk",
			"demandApiEndpoint":      "http://demand.api",
			"demandJsonPath":         "{.demand}",
			"targetDemandPerReplica": "100",
			"metricName":             "my-metric",
			"weatherApiKeyFromEnv": "weatherApiKey",
			"demandApiKeyFromEnv":  "demandApiKey",
		},
		resolvedEnv: testWeatherAwareDemandResolvedEnv,
		isError:     false,
	},
	{
		name: "valid weather endpoint only",
		metadata: map[string]string{
			"weatherApiEndpoint":     "http://weather.api",
			"weatherLocation":        "london,uk",
			"targetDemandPerReplica": "100",
			"metricName":             "my-metric",
			"weatherApiKeyFromEnv": "weatherApiKey",
			"demandApiKeyFromEnv":  "demandApiKey",
		},
		resolvedEnv: testWeatherAwareDemandResolvedEnv,
		isError:     false,
	},
	{
		name: "valid demand endpoint only",
		metadata: map[string]string{
			"demandApiEndpoint":      "http://demand.api",
			"demandJsonPath":         "{.demand}",
			"targetDemandPerReplica": "100",
			"metricName":             "my-metric",
			"weatherApiKeyFromEnv": "weatherApiKey",
			"demandApiKeyFromEnv":  "demandApiKey",
		},
		resolvedEnv: testWeatherAwareDemandResolvedEnv,
		isError:     false,
	},
	{
		name: "error - no endpoints",
		metadata: map[string]string{
			"targetDemandPerReplica": "100",
			"weatherApiKeyFromEnv": "weatherApiKey",
			"demandApiKeyFromEnv":  "demandApiKey",
		},
		resolvedEnv:        testWeatherAwareDemandResolvedEnv,
		isError:            true,
		expectedErrMessage: "at least one of weatherApiEndpoint or demandApiEndpoint must be provided",
	},
	{
		name: "error - weather endpoint without location",
		metadata: map[string]string{
			"weatherApiEndpoint": "http://weather.api",
			"weatherApiKeyFromEnv": "weatherApiKey",
			"demandApiKeyFromEnv":  "demandApiKey",
		},
		resolvedEnv:        testWeatherAwareDemandResolvedEnv,
		isError:            true,
		expectedErrMessage: "weatherLocation is required when weatherApiEndpoint is provided",
	},
	{
		name: "error - invalid targetDemandPerReplica",
		metadata: map[string]string{
			"demandApiEndpoint":      "http://demand.api",
			"demandJsonPath":         "{.demand}",
			"targetDemandPerReplica": "0",
			"weatherApiKeyFromEnv": "weatherApiKey",
			"demandApiKeyFromEnv":  "demandApiKey",
		},
		resolvedEnv:        testWeatherAwareDemandResolvedEnv,
		isError:            true,
		expectedErrMessage: "targetDemandPerReplica must be greater than 0",
	},
	{
		name: "error - invalid weatherEffectScaleFactor",
		metadata: map[string]string{
			"demandApiEndpoint":        "http://demand.api",
			"demandJsonPath":           "{.demand}",
			"weatherEffectScaleFactor": "0",
			"weatherApiKeyFromEnv": "weatherApiKey",
			"demandApiKeyFromEnv":  "demandApiKey",
		},
		resolvedEnv:        testWeatherAwareDemandResolvedEnv,
		isError:            true,
		expectedErrMessage: "weatherEffectScaleFactor must be greater than 0",
	},
}

func TestNewWeatherAwareDemandScaler(t *testing.T) {
	for _, config := range weatherAwareDemandScalerTestConfigs {
		t.Run(config.name, func(t *testing.T) {
			cfg := &scalersconfig.ScalerConfig{
				TriggerMetadata: config.metadata,
				ResolvedEnv:     config.resolvedEnv,
				TriggerIndex:    0,
			}

			_, err := NewWeatherAwareDemandScaler(cfg)
			if config.isError {
				assert.Error(t, err)
				if config.expectedErrMessage != "" {
					assert.Contains(t, err.Error(), config.expectedErrMessage)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

type testGetMetricsCase struct {
	name                     string
	metadata                 map[string]string
	resolvedEnv              map[string]string
	mockDemandAPIHandler     http.HandlerFunc
	mockWeatherAPIHandler    http.HandlerFunc
	expectedMetricValue      int64
	expectedActivity         bool
	isError                  bool
	expectedErrMessage       string
	activationDemandLevel    string // Keep as string to test parsing
	weatherEffectScaleFactor string // Keep as string to test parsing
	badWeatherConditions     string
}

// Helper to create a test server for weather aware demand scaler tests
func createWeatherTestServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

var testGetMetricsCases = []testGetMetricsCase{
	{
		name: "demand only - active",
		metadata: map[string]string{
			"demandApiEndpoint":      "DEMAND_ENDPOINT_VAR", // Will be replaced by server URL
			"demandJsonPath":         "{.value}",
			"targetDemandPerReplica": "50",
			"metricName":             "test-demand",
		},
		resolvedEnv: map[string]string{
			"demandApiKey":  "test_demand_key",
			"weatherApiKey": "unused_weather_key", // Required by resolvedEnv
		},
		mockDemandAPIHandler: func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer test_demand_key" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			fmt.Fprint(w, `{"value": 100}`)
		},
		expectedMetricValue:   100,
		expectedActivity:      true,
		activationDemandLevel: "10",
	},
	{
		name: "demand only - inactive",
		metadata: map[string]string{
			"demandApiEndpoint":      "DEMAND_ENDPOINT_VAR",
			"demandJsonPath":         "{.value}",
			"targetDemandPerReplica": "50",
		},
		resolvedEnv: map[string]string{
			"demandApiKey":  "test_demand_key",
			"weatherApiKey": "unused_weather_key", // Required by resolvedEnv
		},
		mockDemandAPIHandler: func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"value": 5}`)
		},
		expectedMetricValue:   5,
		expectedActivity:      false,
		activationDemandLevel: "10",
	},
	{
		name: "demand only - API error",
		metadata: map[string]string{
			"demandApiEndpoint": "DEMAND_ENDPOINT_VAR",
			"demandJsonPath":    "{.value}",
		},
		resolvedEnv: map[string]string{
			"demandApiKey":  "test_demand_key",
			"weatherApiKey": "unused_weather_key", // Required by resolvedEnv
		},
		mockDemandAPIHandler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		},
		isError:            true,
		expectedErrMessage: "error request to",
	},
	{
		name: "demand only - bad JSON path",
		metadata: map[string]string{
			"demandApiEndpoint": "DEMAND_ENDPOINT_VAR",
			"demandJsonPath":    "{.nonexistent}",
		},
		resolvedEnv: map[string]string{
			"demandApiKey":  "test_demand_key",
			"weatherApiKey": "unused_weather_key", // Required by resolvedEnv
		},
		mockDemandAPIHandler: func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"value": 100}`)
		},
		isError:            true,
		expectedErrMessage: "error extracting demand value",
	},
	{
		name: "weather only - good weather, activation based on default demand 0",
		metadata: map[string]string{
			"weatherApiEndpoint":     "WEATHER_ENDPOINT_VAR",
			"weatherLocation":        "london,uk",
			"targetDemandPerReplica": "50",
			"badWeatherConditions":   "temp_below:0", // Temp is 10, so good weather
		},
		resolvedEnv: map[string]string{
			"weatherApiKey": "test_weather_key",
			"demandApiKey":  "unused_demand_key", // Required by resolvedEnv
		},
		mockWeatherAPIHandler: func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer test_weather_key" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			fmt.Fprint(w, `{"temp": 10, "rain": 0}`)
		},
		expectedMetricValue:   0,     // Default demand is 0
		expectedActivity:      false, // 0 is not > activation (default 10)
		activationDemandLevel: "10",
	},
	{
		name: "weather only - bad weather, activation, default demand 0 but scaled",
		metadata: map[string]string{
			"weatherApiEndpoint":       "WEATHER_ENDPOINT_VAR",
			"weatherLocation":          "london,uk",
			"badWeatherConditions":     "temp_below:0", // Temp is -5, so bad weather
			"weatherEffectScaleFactor": "1.5",          // This will be applied to default demand (0), still results in 0
		},
		resolvedEnv: map[string]string{
			"weatherApiKey": "test_weather_key",
			"demandApiKey":  "unused_demand_key", // Required by resolvedEnv
		},
		mockWeatherAPIHandler: func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"temp": -5, "rain": 0}`)
		},
		expectedMetricValue:   0,     // 0 * 1.5 = 0
		expectedActivity:      false, // 0 not > activation (default 10)
		activationDemandLevel: "10",
	},
	{
		name: "both APIs - good weather - active",
		metadata: map[string]string{
			"demandApiEndpoint":      "DEMAND_ENDPOINT_VAR",
			"demandJsonPath":         "{.value}",
			"weatherApiEndpoint":     "WEATHER_ENDPOINT_VAR",
			"weatherLocation":        "paris,fr",
			"weatherUnits":           "imperial",
			"badWeatherConditions":   "temp_below:32", // Temp is 40F (good)
			"targetDemandPerReplica": "10",
		},
		resolvedEnv: map[string]string{
			"demandApiKey":  "test_demand_key",
			"weatherApiKey": "test_weather_key",
		},
		mockDemandAPIHandler: func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"value": 100}`)
		},
		mockWeatherAPIHandler: func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"temp": 40, "rain": 0.1}`) // Assuming 40F
		},
		expectedMetricValue:   100, // No scaling
		expectedActivity:      true,
		activationDemandLevel: "10",
	},
	{
		name: "both APIs - bad weather (temp_below) - demand scaled - active",
		metadata: map[string]string{
			"demandApiEndpoint":        "DEMAND_ENDPOINT_VAR",
			"demandJsonPath":           "{.value}",
			"weatherApiEndpoint":       "WEATHER_ENDPOINT_VAR",
			"weatherLocation":          "berlin,de",
			"badWeatherConditions":     "temp_below:0,rain_above:10", // Temp is -5C (bad)
			"weatherEffectScaleFactor": "2.0",
			"targetDemandPerReplica":   "10",
		},
		resolvedEnv: map[string]string{
			"demandApiKey":  "test_demand_key",
			"weatherApiKey": "test_weather_key",
		},
		mockDemandAPIHandler: func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"value": 75}`)
		},
		mockWeatherAPIHandler: func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"temp": -5, "rain": 5}`) // Temp is bad, rain is not
		},
		expectedMetricValue:   150, // 75 * 2.0
		expectedActivity:      true,
		activationDemandLevel: "10",
	},
	{
		name: "both APIs - bad weather (rain_above) - demand scaled - active",
		metadata: map[string]string{
			"demandApiEndpoint":        "DEMAND_ENDPOINT_VAR",
			"demandJsonPath":           "{.value}",
			"weatherApiEndpoint":       "WEATHER_ENDPOINT_VAR",
			"weatherLocation":          "rome,it",
			"badWeatherConditions":     "temp_below:0,rain_above:5", // Rain is 12mm (bad)
			"weatherEffectScaleFactor": "1.5",
		},
		resolvedEnv: map[string]string{
			"demandApiKey":  "test_demand_key",
			"weatherApiKey": "test_weather_key",
		},
		mockDemandAPIHandler: func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"value": 60}`)
		},
		mockWeatherAPIHandler: func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"temp": 10, "rain": 12}`) // Rain is bad, temp is not
		},
		expectedMetricValue:   90, // 60 * 1.5
		expectedActivity:      true,
		activationDemandLevel: "10",
	},
	{
		name: "both APIs - bad weather (wind_above) - demand scaled - active",
		metadata: map[string]string{
			"demandApiEndpoint":        "DEMAND_ENDPOINT_VAR",
			"demandJsonPath":           "{.value}",
			"weatherApiEndpoint":       "WEATHER_ENDPOINT_VAR",
			"weatherLocation":          "rome,it",
			"badWeatherConditions":     "wind_above:20",
			"weatherEffectScaleFactor": "1.2",
		},
		resolvedEnv: map[string]string{
			"demandApiKey":  "test_demand_key",
			"weatherApiKey": "test_weather_key",
		},
		mockDemandAPIHandler: func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"value": 50}`)
		},
		mockWeatherAPIHandler: func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"wind": 25}`)
		},
		expectedMetricValue:   60, // 50 * 1.2
		expectedActivity:      true,
		activationDemandLevel: "10",
	},
	{
		name: "weather API error - proceeds with demand only",
		metadata: map[string]string{
			"demandApiEndpoint":    "DEMAND_ENDPOINT_VAR",
			"demandJsonPath":       "{.value}",
			"weatherApiEndpoint":   "WEATHER_ENDPOINT_VAR",
			"weatherLocation":      "london,uk",
			"badWeatherConditions": "temp_below:0",
		},
		resolvedEnv: map[string]string{
			"demandApiKey":  "test_demand_key",
			"weatherApiKey": "test_weather_key",
		},
		mockDemandAPIHandler: func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"value": 120}`)
		},
		mockWeatherAPIHandler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		},
		expectedMetricValue:   120, // Uses demand, weather error is logged but doesn't fail metric
		expectedActivity:      true,
		activationDemandLevel: "10",
	},
	{
		name: "bad weather condition format - error in isBadWeather",
		metadata: map[string]string{
			"demandApiEndpoint":    "DEMAND_ENDPOINT_VAR",
			"demandJsonPath":       "{.value}",
			"weatherApiEndpoint":   "WEATHER_ENDPOINT_VAR",
			"weatherLocation":      "london,uk",
			"badWeatherConditions": "temp_is_low:0", // Invalid format
		},
		resolvedEnv: map[string]string{
			"demandApiKey":  "test_demand_key",
			"weatherApiKey": "test_weather_key",
		},
		mockDemandAPIHandler: func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"value": 100}`)
		},
		mockWeatherAPIHandler: func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"temp": -5}`)
		},
		expectedMetricValue:   100, // Error in isBadWeather, proceeds as if good weather
		expectedActivity:      true,
		activationDemandLevel: "10",
	},
}

func TestGetMetricsAndActivity(t *testing.T) {
	for _, tc := range testGetMetricsCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock servers
			var demandServer, weatherServer *httptest.Server

			currentMetadata := make(map[string]string)
			for k, v := range tc.metadata {
				currentMetadata[k] = v
			}

			// Add FromEnv keys for API key parsing
			currentMetadata["weatherApiKeyFromEnv"] = "weatherApiKey"
			currentMetadata["demandApiKeyFromEnv"] = "demandApiKey"

			if tc.mockDemandAPIHandler != nil {
				demandServer = createWeatherTestServer(tc.mockDemandAPIHandler)
				defer demandServer.Close()
				currentMetadata["demandApiEndpoint"] = demandServer.URL
			}
			if tc.mockWeatherAPIHandler != nil {
				weatherServer = createWeatherTestServer(tc.mockWeatherAPIHandler)
				defer weatherServer.Close()
				currentMetadata["weatherApiEndpoint"] = weatherServer.URL
			}

			// Set default activation level if not specified in test case
			if tc.activationDemandLevel != "" {
				currentMetadata["activationDemandLevel"] = tc.activationDemandLevel
			}
			if tc.weatherEffectScaleFactor != "" {
				currentMetadata["weatherEffectScaleFactor"] = tc.weatherEffectScaleFactor
			}
			if tc.badWeatherConditions != "" {
				currentMetadata["badWeatherConditions"] = tc.badWeatherConditions
			}

			cfg := &scalersconfig.ScalerConfig{
				TriggerMetadata:   currentMetadata,
				ResolvedEnv:       tc.resolvedEnv,
				TriggerIndex:      0,
				GlobalHTTPTimeout: 60000, // Increased to 60s
			}
			if tc.resolvedEnv == nil {
				cfg.ResolvedEnv = make(map[string]string)
				// Ensure default keys if not provided, though most test cases now do
				if _, ok := cfg.ResolvedEnv["weatherApiKey"]; !ok {
					cfg.ResolvedEnv["weatherApiKey"] = "default_test_weather_key"
				}
				if _, ok := cfg.ResolvedEnv["demandApiKey"]; !ok {
					cfg.ResolvedEnv["demandApiKey"] = "default_test_demand_key"
				}
			}

			scaler, err := NewWeatherAwareDemandScaler(cfg)
			assert.NoError(t, err, "NewWeatherAwareDemandScaler should not error for valid test setup")
			if err != nil {
				return // Avoid panic if scaler is nil
			}

			// Inject mock logger into the scaler instance
			if ws, ok := scaler.(*weatherAwareDemandScaler); ok {
				ws.logger = newSimpleTestLogger() // Use the new simple logger
			}

			metrics, activity, err := scaler.GetMetricsAndActivity(context.Background(), "test-metric")

			if tc.isError {
				assert.Error(t, err)
				if tc.expectedErrMessage != "" {
					assert.True(t, strings.Contains(err.Error(), tc.expectedErrMessage), "expected error string '%s' not found in '%s'", tc.expectedErrMessage, err.Error())
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedActivity, activity)
				if len(metrics) > 0 {
					assert.Equal(t, tc.expectedMetricValue, metrics[0].Value.Value())
				} else if tc.expectedMetricValue != 0 { // if we expected a metric value but got none
					t.Errorf("Expected metric value %d, but got no metrics", tc.expectedMetricValue)
				}
			}
		})
	}
}

type testIsBadWeatherCase struct {
	name                 string
	badWeatherConditions string
	weatherData          map[string]interface{}
	expectedIsBad        bool
	isError              bool
	expectedErrMessage   string
}

var testIsBadWeatherCases = []testIsBadWeatherCase{
	{
		name:                 "no conditions defined",
		badWeatherConditions: "",
		weatherData:          map[string]interface{}{"temp": 10},
		expectedIsBad:        false,
	},
	{
		name:                 "no weather data",
		badWeatherConditions: "temp_below:0",
		weatherData:          nil,
		expectedIsBad:        false,
	},
	{
		name:                 "temp_below:0, current temp -5 (bad)",
		badWeatherConditions: "temp_below:0",
		weatherData:          map[string]interface{}{"temp": -5.0},
		expectedIsBad:        true,
	},
	{
		name:                 "temp_below:0, current temp 5 (good)",
		badWeatherConditions: "temp_below:0",
		weatherData:          map[string]interface{}{"temp": 5.0},
		expectedIsBad:        false,
	},
	{
		name:                 "rain_above:5, current rain 10 (bad)",
		badWeatherConditions: "rain_above:5",
		weatherData:          map[string]interface{}{"rain": 10.0},
		expectedIsBad:        true,
	},
	{
		name:                 "rain_above:5, current rain 2 (good)",
		badWeatherConditions: "rain_above:5",
		weatherData:          map[string]interface{}{"rain": 2.0},
		expectedIsBad:        false,
	},
	{
		name:                 "wind_above:20, current wind 25 (bad)",
		badWeatherConditions: "wind_above:20",
		weatherData:          map[string]interface{}{"wind": 25.0},
		expectedIsBad:        true,
	},
	{
		name:                 "wind_above:20, current wind 15 (good)",
		badWeatherConditions: "wind_above:20",
		weatherData:          map[string]interface{}{"wind": 15.0},
		expectedIsBad:        false,
	},
	{
		name:                 "multiple conditions - temp_below:0 (bad), rain_above:10 (good)",
		badWeatherConditions: "temp_below:0,rain_above:10",
		weatherData:          map[string]interface{}{"temp": -2.0, "rain": 5.0},
		expectedIsBad:        true, // Temp condition met
	},
	{
		name:                 "multiple conditions - temp_below:0 (good), rain_above:10 (bad)",
		badWeatherConditions: "temp_below:0,rain_above:10",
		weatherData:          map[string]interface{}{"temp": 2.0, "rain": 15.0},
		expectedIsBad:        true, // Rain condition met
	},
	{
		name:                 "multiple conditions - all good",
		badWeatherConditions: "temp_below:0,rain_above:10,wind_above:30",
		weatherData:          map[string]interface{}{"temp": 5.0, "rain": 5.0, "wind": 10.0},
		expectedIsBad:        false,
	},
	{
		name:                 "weather key not found in data",
		badWeatherConditions: "temp_below:0",
		weatherData:          map[string]interface{}{"humidity": 50.0}, // "temp" is missing
		expectedIsBad:        false,                                    // Skips condition
	},
	{
		name:                 "invalid condition format - no colon",
		badWeatherConditions: "temp_below0",
		weatherData:          map[string]interface{}{"temp": -5.0},
		isError:              true,
		expectedErrMessage:   "invalid bad weather condition format: temp_below0",
	},
	{
		name:                 "invalid condition format - wrong suffix",
		badWeatherConditions: "temp_is:0",
		weatherData:          map[string]interface{}{"temp": -5.0},
		isError:              true,
		expectedErrMessage:   "invalid bad weather condition format: temp_is, must end with '_below' or '_above'",
	},
	{
		name:                 "weather data for key not a number",
		badWeatherConditions: "temp_below:0",
		weatherData:          map[string]interface{}{"temp": "cold"},
		isError:              true,
		expectedErrMessage:   "weather data for key 'temp' is not a number: cold",
	},
	{
		name:                 "invalid threshold value in condition",
		badWeatherConditions: "temp_below:abc",
		weatherData:          map[string]interface{}{"temp": -5.0},
		isError:              true,
		expectedErrMessage:   "invalid threshold value in bad weather condition 'temp_below:abc'",
	},
}

func TestIsBadWeather(t *testing.T) {
	// Create a dummy scaler instance with a logger
	mockScaler := &weatherAwareDemandScaler{
		logger:   newSimpleTestLogger(), // Use the new simple logger
		metadata: &weatherAwareDemandScalerMetadata{},
	}

	for _, tc := range testIsBadWeatherCases {
		t.Run(tc.name, func(t *testing.T) {
			mockScaler.metadata.BadWeatherConditions = tc.badWeatherConditions
			isBad, err := mockScaler.isBadWeather(tc.weatherData)

			if tc.isError {
				assert.Error(t, err)
				if tc.expectedErrMessage != "" {
					assert.Contains(t, err.Error(), tc.expectedErrMessage)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedIsBad, isBad)
			}
		})
	}
}

func TestWeatherAwareDemandGetMetricSpecForScaling(t *testing.T) {
	tests := []struct {
		name         string
		metadata     *weatherAwareDemandScalerMetadata
		metricType   v2.MetricTargetType
		expectedName string
	}{
		{
			name: "average value metric type",
			metadata: &weatherAwareDemandScalerMetadata{
				triggerIndex:           0,
				MetricName:             "custom-metric",
				TargetDemandPerReplica: 100,
			},
			metricType:   v2.AverageValueMetricType,
			expectedName: "s0-custom-metric",
		},
		{
			name: "value metric type",
			metadata: &weatherAwareDemandScalerMetadata{
				triggerIndex:           1,
				MetricName:             "another_metric-name with spaces",
				TargetDemandPerReplica: 50,
			},
			metricType:   v2.ValueMetricType,
			expectedName: "s1-another_metric-name with spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scaler := &weatherAwareDemandScaler{
				metadata:   tt.metadata,
				metricType: tt.metricType,
				logger:     newSimpleTestLogger(),
			}
			specs := scaler.GetMetricSpecForScaling(context.Background())
			assert.Len(t, specs, 1)
			spec := specs[0]
			assert.Equal(t, v2.ExternalMetricSourceType, spec.Type)
			assert.NotNil(t, spec.External)
			assert.Equal(t, tt.expectedName, spec.External.Metric.Name)
			if tt.metricType == v2.AverageValueMetricType {
				assert.Equal(t, resource.NewQuantity(tt.metadata.TargetDemandPerReplica, resource.DecimalSI), spec.External.Target.AverageValue)
				assert.Equal(t, v2.AverageValueMetricType, spec.External.Target.Type)

			} else {
				assert.Equal(t, resource.NewQuantity(tt.metadata.TargetDemandPerReplica, resource.DecimalSI), spec.External.Target.Value)
				assert.Equal(t, v2.ValueMetricType, spec.External.Target.Type)
			}
		})
	}
}

func TestClose(t *testing.T) {
	server := createWeatherTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "OK")
	}))
	defer server.Close()

	cfg := &scalersconfig.ScalerConfig{
		TriggerMetadata: map[string]string{
			"demandApiEndpoint": server.URL, // Provide a valid URL to init httpclient
			"demandJsonPath":    "{.value}",
			// Add FromEnv keys for API key parsing
			"weatherApiKeyFromEnv": "weatherApiKey",
			"demandApiKeyFromEnv":  "demandApiKey",
		},
		ResolvedEnv:       map[string]string{"demandApiKey": "dummykey", "weatherApiKey": "dummyweatherkey"}, // Provide necessary resolved env
		TriggerIndex:      0,
		GlobalHTTPTimeout: 5000, // Increased to 5s
	}

	scalerInterface, err := NewWeatherAwareDemandScaler(cfg)
	assert.NoError(t, err, "NewWeatherAwareDemandScaler failed in TestClose setup")
	if err != nil {
		t.FailNow() // Stop test if setup fails
	}

	scaler, ok := scalerInterface.(*weatherAwareDemandScaler)
	assert.True(t, ok, "Scaler should be of type *weatherAwareDemandScaler in TestClose")
	if !ok {
		t.FailNow()
	}

	// Make a call to ensure connection is used (if http client is relevant to Close)
	// This might not be strictly necessary if Close only handles the client itself.
	_, _, _ = scaler.GetMetricsAndActivity(context.Background(), "test-close-metric")

	err = scaler.Close(context.Background())
	assert.NoError(t, err, "Close should not return an error")
}
