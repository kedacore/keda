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
			"weatherParameter":       "temp",
			"weatherOperator":        ">",
			"weatherThreshold":       "10",
			"demandApiEndpoint":      "http://demand.api",
			"demandJsonPath":         "{.demand}",
			"targetDemandPerReplica": "100",
			"metricName":             "my-metric",
			"weatherApiKeyFromEnv":   "weatherApiKey",
			"demandApiKeyFromEnv":    "demandApiKey",
		},
		resolvedEnv: testWeatherAwareDemandResolvedEnv,
		isError:     false,
	},
	{
		name: "valid weather endpoint only",
		metadata: map[string]string{
			"weatherApiEndpoint":     "http://weather.api",
			"weatherLocation":        "london,uk",
			"weatherParameter":       "temp",
			"weatherOperator":        "<",
			"weatherThreshold":       "0",
			"targetDemandPerReplica": "100",
			"metricName":             "my-metric",
			"weatherApiKeyFromEnv":   "weatherApiKey",
			"demandApiKeyFromEnv":    "demandApiKey",
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
			"weatherApiKeyFromEnv":   "weatherApiKey",
			"demandApiKeyFromEnv":    "demandApiKey",
		},
		resolvedEnv: testWeatherAwareDemandResolvedEnv,
		isError:     false,
	},
	{
		name: "error - no endpoints",
		metadata: map[string]string{
			"targetDemandPerReplica": "100",
			"weatherApiKeyFromEnv":   "weatherApiKey",
			"demandApiKeyFromEnv":    "demandApiKey",
		},
		resolvedEnv:        testWeatherAwareDemandResolvedEnv,
		isError:            true,
		expectedErrMessage: "at least one of weatherApiEndpoint or demandApiEndpoint must be provided",
	},
	{
		name: "error - weather endpoint without location",
		metadata: map[string]string{
			"weatherApiEndpoint":   "http://weather.api",
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
			"weatherApiKeyFromEnv":   "weatherApiKey",
			"demandApiKeyFromEnv":    "demandApiKey",
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
			"weatherApiKeyFromEnv":     "weatherApiKey",
			"demandApiKeyFromEnv":      "demandApiKey",
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
	weatherParameter         string // New field
	weatherOperator          string // New field
	weatherThreshold         string // New field
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
			"targetDemandPerReplica": "50",       // Explicitly set for this test
			"weatherParameter":       "temp",     // Condition for scaling: temp < 0
			"weatherOperator":        "<",        // If temp is 10 (good), 10 < 0 is false -> demand 0
			"weatherThreshold":       "0",
		},
		resolvedEnv: map[string]string{
			"weatherApiKey": "test_weather_key",
			"demandApiKey":  "unused_demand_key",
		},
		mockWeatherAPIHandler: func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer test_weather_key" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			fmt.Fprint(w, `{"temp": 10, "rain": 0}`) // Good weather: temp = 10
		},
		expectedMetricValue:   0,     // Good weather, so demand is 0
		expectedActivity:      false, // 0 is not > activation (default 10)
		activationDemandLevel: "10",
	},
	{
		name: "weather only - bad weather, activation, default demand 0 but scaled",
		metadata: map[string]string{
			"weatherApiEndpoint":       "WEATHER_ENDPOINT_VAR",
			"weatherLocation":          "london,uk",
			"weatherParameter":         "temp",    // Condition for scaling: temp < 0
			"weatherOperator":          "<",       // If temp is -5 (bad), -5 < 0 is true -> demand = targetDemandPerReplica
			"weatherThreshold":         "0",
			// "targetDemandPerReplica" is not set here, so it defaults to 100 from the struct.
			// "weatherEffectScaleFactor": "1.5", // Not used in weather-only mode
		},
		resolvedEnv: map[string]string{
			"weatherApiKey": "test_weather_key",
			"demandApiKey":  "unused_demand_key",
		},
		mockWeatherAPIHandler: func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"temp": -5, "rain": 0}`) // Bad weather: temp = -5
		},
		expectedMetricValue:   100,   // Bad weather, demand is TargetDemandPerReplica (default 100)
		expectedActivity:      true,  // 100 > activation (default 10)
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
			"targetDemandPerReplica": "10",
			"weatherParameter":       "temp",
			"weatherOperator":        ">",
			"weatherThreshold":       "32",
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
			"weatherEffectScaleFactor": "2.0",
			"targetDemandPerReplica":   "10",
			"weatherParameter":         "temp",
			"weatherOperator":          "<",
			"weatherThreshold":         "0",
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
			"weatherEffectScaleFactor": "1.5",
			"weatherParameter":         "rain",
			"weatherOperator":          ">",
			"weatherThreshold":         "5",
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
			"weatherEffectScaleFactor": "1.2",
			"weatherParameter":         "wind",
			"weatherOperator":          ">",
			"weatherThreshold":         "20",
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
			"weatherParameter":     "temp",
			"weatherOperator":      "<",
			"weatherThreshold":     "0",
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
			"weatherParameter":     "temp",
			"weatherOperator":      ">",
			"weatherThreshold":     "0",
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
			if tc.weatherParameter != "" {
				currentMetadata["weatherParameter"] = tc.weatherParameter
			}
			if tc.weatherOperator != "" {
				currentMetadata["weatherOperator"] = tc.weatherOperator
			}
			if tc.weatherThreshold != "" {
				currentMetadata["weatherThreshold"] = tc.weatherThreshold
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
