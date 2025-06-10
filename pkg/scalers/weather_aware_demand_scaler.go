package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/tidwall/gjson"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

// weatherAwareDemandScalerMetadata holds the metadata parsed from the ScaledObject
type weatherAwareDemandScalerMetadata struct {
	// Weather API Configuration
	WeatherAPIEndpoint   string `keda:"name=weatherApiEndpoint,order=triggerMetadata,optional"`
	WeatherAPIKeyFromEnv string `keda:"name=weatherApiKey,order=resolvedEnv"`
	WeatherLocation      string `keda:"name=weatherLocation,order=triggerMetadata,optional"`             // e.g., "city,country" or "lat,lon"
	WeatherUnits         string `keda:"name=weatherUnits,order=triggerMetadata,optional,default=metric"` // "metric" or "imperial"
	// BadWeatherConditions string `keda:"name=badWeatherConditions,order=triggerMetadata,optional"`        // e.g., "temp_below:0,rain_above:5,wind_above:10" (temp in C, rain mm/hr, wind km/hr if metric)
	WeatherParameter string `keda:"name=weatherParameter,order=triggerMetadata,optional"` // e.g., "temperature", "humidity", "windSpeed"
	WeatherOperator  string `keda:"name=weatherOperator,order=triggerMetadata,optional"`  // e.g., ">", "<", "==", ">=", "<="
	WeatherThreshold string `keda:"name=weatherThreshold,order=triggerMetadata,optional"` // e.g., "30", "0.5"

	// Demand API Configuration
	DemandAPIEndpoint string `keda:"name=demandApiEndpoint,order=triggerMetadata,optional"`
	DemandAPIKey      string `keda:"name=demandApiKey,order=resolvedEnv"`
	DemandJSONPath    string `keda:"name=demandJsonPath,order=triggerMetadata,optional"` // JSONPath to extract the demand value, e.g., "{.current_demand}"

	// Scaling Logic
	TargetDemandPerReplica   int64   `keda:"name=targetDemandPerReplica,order=triggerMetadata,optional,default=100"`
	ActivationDemandLevel    int64   `keda:"name=activationDemandLevel,order=triggerMetadata,optional,default=10"`
	WeatherEffectScaleFactor float64 `keda:"name=weatherEffectScaleFactor,order=triggerMetadata,optional,default=1.0"` // e.g., 1.5 for 50% increase in perceived demand during bad weather
	MetricName               string  `keda:"name=metricName,order=triggerMetadata,optional,default=weather-aware-ride-demand"`

	// Internal fields
	triggerIndex    int               // Stores the trigger index
	triggerMetadata map[string]string // To store trigger metadata for simplified API key access
}

func (m *weatherAwareDemandScalerMetadata) Validate() error {
	if m.WeatherAPIEndpoint == "" && m.DemandAPIEndpoint == "" {
		return fmt.Errorf("at least one of weatherApiEndpoint or demandApiEndpoint must be provided")
	}
	if m.WeatherAPIEndpoint != "" {
		if m.WeatherLocation == "" {
			return fmt.Errorf("weatherLocation is required when weatherApiEndpoint is provided")
		}
		if m.WeatherParameter == "" {
			return fmt.Errorf("weatherParameter is required when weatherApiEndpoint is provided")
		}
		if m.WeatherOperator == "" {
			return fmt.Errorf("weatherOperator is required when weatherApiEndpoint is provided")
		}
		allowedOperators := map[string]bool{">": true, "<": true, "==": true, ">=": true, "<=": true, "!=": true}
		if !allowedOperators[m.WeatherOperator] {
			return fmt.Errorf("invalid weatherOperator: %s. Allowed operators are >, <, ==, >=, <=, !=", m.WeatherOperator)
		}
		if m.WeatherThreshold == "" {
			return fmt.Errorf("weatherThreshold is required when weatherApiEndpoint is provided")
		}
		if _, err := strconv.ParseFloat(m.WeatherThreshold, 64); err != nil {
			return fmt.Errorf("weatherThreshold must be a valid number: %w", err)
		}
	}

	if m.TargetDemandPerReplica <= 0 {
		return fmt.Errorf("targetDemandPerReplica must be greater than 0")
	}
	if m.WeatherEffectScaleFactor <= 0 {
		return fmt.Errorf("weatherEffectScaleFactor must be greater than 0")
	}
	return nil
}

// weatherAwareDemandScaler is the scaler implementation
type weatherAwareDemandScaler struct {
	metricType v2.MetricTargetType
	metadata   *weatherAwareDemandScalerMetadata
	httpClient *http.Client
	logger     logr.Logger
	// config *scalersconfig.ScalerConfig // If direct access to ResolvedEnv is needed later
}

// Ensure weatherAwareDemandScaler implements the Scaler interface
var _ Scaler = (*weatherAwareDemandScaler)(nil)

// NewWeatherAwareDemandScaler creates a new WeatherAwareDemandScaler
func NewWeatherAwareDemandScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	logger := InitializeLogger(config, "weather_aware_demand_scaler")

	meta := &weatherAwareDemandScalerMetadata{}
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing weather aware demand scaler metadata: %w", err)
	}
	if err := meta.Validate(); err != nil {
		return nil, fmt.Errorf("error validating weather aware demand scaler metadata: %w", err)
	}
	meta.triggerIndex = config.TriggerIndex

	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	timeoutDuration := time.Duration(config.GlobalHTTPTimeout) * time.Millisecond
	httpClient := kedautil.CreateHTTPClient(timeoutDuration, false)

	return &weatherAwareDemandScaler{
		metadata:   meta,
		metricType: metricType,
		httpClient: httpClient,
		logger:     logger,
	}, nil
}

// Helper function to fetch and parse JSON from an HTTP endpoint
func (s *weatherAwareDemandScaler) fetchJSONData(ctx context.Context, endpoint string, actualAPIKey string, result interface{}) error {
	if endpoint == "" {
		return fmt.Errorf("endpoint is not configured")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		s.logger.Error(err, "Error creating HTTP request")
		return fmt.Errorf("error creating request for %s: %w", endpoint, err)
	}

	if actualAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+actualAPIKey) // Or other auth mechanism
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making request to %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error request to %s returned status %d", endpoint, resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("error decoding json response from %s: %w", endpoint, err)
	}
	return nil
}

// Helper to extract a float64 value using GJSON
func extractValueWithJSONPath(data interface{}, path string, logger logr.Logger) (float64, error) {
	if path == "" {
		// If no path, try to convert data directly if it's a number, or assume it's a simple map[string]interface{} with a "value" field.
		switch v := data.(type) {
		case float64:
			return v, nil
		case int64:
			return float64(v), nil
		case int:
			return float64(v), nil
		case map[string]interface{}:
			if val, ok := v["value"]; ok {
				if fVal, okVal := val.(float64); okVal {
					return fVal, nil
				}
			}
		}
		logger.V(1).Info("Path is empty, attempting to use 'value' field or direct conversion, data might not be in expected format", "data", data)
		return 0, fmt.Errorf("path is empty and data is not a simple numeric value or map with 'value' field")
	}

	// Convert data to JSON bytes for gjson
	jsonData, err := json.Marshal(data)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal data to JSON: %w", err)
	}

	// Use gjson to extract the value
	// Convert kubectl-style JSONPath {.field.subfield} to gjson style field.subfield
	gjsonPath := strings.Trim(path, "{}")
	gjsonPath = strings.TrimPrefix(gjsonPath, ".")

	result := gjson.GetBytes(jsonData, gjsonPath)
	if !result.Exists() {
		return 0, fmt.Errorf("path '%s' yielded no results", path)
	}

	// Handle different result types similar to metrics_api_scaler
	if result.Type == gjson.String {
		// Try to parse as a quantity first (for K8s-style values)
		if val, err := resource.ParseQuantity(result.String()); err == nil {
			return val.AsApproximateFloat64(), nil
		}
		// Fall back to regular float parsing
		return strconv.ParseFloat(result.String(), 64)
	}

	if result.Type != gjson.Number {
		return 0, fmt.Errorf("path '%s' does not point to a numeric value, got: %s", path, result.Type.String())
	}

	return result.Num, nil
}

// evaluateWeatherCondition evaluates if the configured weather condition is met
func (s *weatherAwareDemandScaler) evaluateWeatherCondition(weatherData map[string]interface{}) (bool, error) {
	if s.metadata.WeatherAPIEndpoint == "" || s.metadata.WeatherParameter == "" || weatherData == nil {
		return false, nil // No weather API, parameter, or data, so condition cannot be met.
	}

	weatherVal, ok := weatherData[s.metadata.WeatherParameter]
	if !ok {
		s.logger.V(1).Info("Weather parameter not found in weather data", "parameter", s.metadata.WeatherParameter)
		return false, nil // Parameter not in weather data, condition not met.
	}

	weatherNum, ok := weatherVal.(float64)
	if !ok {
		return false, fmt.Errorf("weather data for parameter '%s' is not a number: %v", s.metadata.WeatherParameter, weatherVal)
	}

	threshold, err := strconv.ParseFloat(s.metadata.WeatherThreshold, 64)
	if err != nil {
		// This should ideally be caught by metadata validation, but good to have a safeguard.
		return false, fmt.Errorf("invalid weatherThreshold value '%s': %w", s.metadata.WeatherThreshold, err)
	}

	s.logger.V(1).Info("Evaluating weather condition", "parameter", s.metadata.WeatherParameter, "operator", s.metadata.WeatherOperator, "threshold", threshold, "actualValue", weatherNum)

	switch s.metadata.WeatherOperator {
	case ">":
		return weatherNum > threshold, nil
	case "<":
		return weatherNum < threshold, nil
	case "==":
		return weatherNum == threshold, nil
	case ">=":
		return weatherNum >= threshold, nil
	case "<=":
		return weatherNum <= threshold, nil
	case "!=":
		return weatherNum != threshold, nil
	default:
		// This should also be caught by metadata validation.
		return false, fmt.Errorf("invalid weatherOperator: %s", s.metadata.WeatherOperator)
	}
}

func (s *weatherAwareDemandScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	s.logger.V(1).Info("Fetching metrics for Weather-Aware Demand Scaler")

	currentDemand := float64(0)
	demandFetched := false

	// 1. Fetch Demand Data (if configured)
	if s.metadata.DemandAPIEndpoint != "" {
		var demandDataRaw interface{}
		err := s.fetchJSONData(ctx, s.metadata.DemandAPIEndpoint, s.metadata.DemandAPIKey, &demandDataRaw)
		if err != nil {
			s.logger.Error(err, "Failed to fetch demand data")
			return nil, false, fmt.Errorf("error fetching demand data: %w", err)
		}

		extractedDemand, err := extractValueWithJSONPath(demandDataRaw, s.metadata.DemandJSONPath, s.logger)
		if err != nil {
			s.logger.Error(err, "Failed to extract demand value from response", "jsonPath", s.metadata.DemandJSONPath)
			return nil, false, fmt.Errorf("error extracting demand value: %w", err)
		}
		currentDemand = extractedDemand
		demandFetched = true
		s.logger.V(1).Info("Successfully fetched demand data", "rawDemand", currentDemand)
	} else {
		s.logger.V(1).Info("DemandAPIEndpoint not configured.")
	}

	// 2. Fetch Weather Data (if configured)
	weatherData := make(map[string]interface{}) // Initialize to avoid nil map if not fetched
	weatherFetched := false
	if s.metadata.WeatherAPIEndpoint != "" {
		weatherURL := fmt.Sprintf("%s?location=%s&units=%s", s.metadata.WeatherAPIEndpoint, s.metadata.WeatherLocation, s.metadata.WeatherUnits)
		err := s.fetchJSONData(ctx, weatherURL, s.metadata.WeatherAPIKeyFromEnv, &weatherData)
		if err != nil {
			s.logger.Error(err, "Failed to fetch weather data. Depending on the mode, this might lead to default behavior or an error.")
			// If weather-only mode, this is a critical failure.
			if !demandFetched {
				return nil, false, fmt.Errorf("failed to fetch weather data in weather-only mode: %w", err)
			}
			// In demand+weather mode, we might proceed with no weather adjustment.
		} else {
			weatherFetched = true
			s.logger.V(1).Info("Successfully fetched weather data", "data", weatherData)
		}
	} else {
		s.logger.V(1).Info("WeatherAPIEndpoint not configured.")
	}

	// 3. Apply Scaling Logic based on configuration
	adjustedDemand := float64(0)
	var scalingMode string

	if weatherFetched && !demandFetched { // Weather-only mode
		scalingMode = "weather-only"
		conditionMet, err := s.evaluateWeatherCondition(weatherData)
		if err != nil {
			s.logger.Error(err, "Failed to evaluate weather condition in weather-only mode.")
			return nil, false, fmt.Errorf("error evaluating weather condition: %w", err)
		}
		if conditionMet {
			adjustedDemand = float64(s.metadata.TargetDemandPerReplica) // Scale up if condition is met
			s.logger.V(1).Info("Weather condition met in weather-only mode, setting demand to target.", "targetDemand", adjustedDemand)
		} else {
			adjustedDemand = 0 // No scaling if condition not met
			s.logger.V(1).Info("Weather condition not met in weather-only mode.")
		}
	} else if weatherFetched && demandFetched { // Demand + Weather mode
		scalingMode = "demand+weather"
		adjustedDemand = currentDemand // Start with current demand
		conditionMet, err := s.evaluateWeatherCondition(weatherData)
		if err != nil {
			s.logger.Error(err, "Failed to evaluate weather condition in demand+weather mode, proceeding without weather adjustment.")
			// Not returning error here, just proceeding without adjustment
		} else if conditionMet {
			adjustedDemand = currentDemand * s.metadata.WeatherEffectScaleFactor
			s.logger.V(1).Info("Weather condition met, adjusting demand", "originalDemand", currentDemand, "scaleFactor", s.metadata.WeatherEffectScaleFactor, "adjustedDemand", adjustedDemand)
		} else {
			s.logger.V(1).Info("Weather condition not met, using original demand.", "originalDemand", currentDemand)
		}
	} else if demandFetched && !weatherFetched { // Demand-only mode
		scalingMode = "demand-only"
		adjustedDemand = currentDemand
		s.logger.V(1).Info("Operating in demand-only mode.", "currentDemand", adjustedDemand)
	} else { // Neither configured - this should be caught by Validate, but handle defensively
		scalingMode = "unconfigured"
		s.logger.Error(nil, "Neither WeatherAPIEndpoint nor DemandAPIEndpoint are configured. This should be caught by validation.")
		return nil, false, fmt.Errorf("scaler is not configured with any API endpoint")
	}
	s.logger.V(1).Info("Scaling logic applied", "mode", scalingMode, "finalAdjustedDemand", adjustedDemand)

	// 4. Determine Activity
	isActive := adjustedDemand > float64(s.metadata.ActivationDemandLevel)
	s.logger.V(1).Info("Scaler activity check", "adjustedDemand", adjustedDemand, "activationDemandLevel", s.metadata.ActivationDemandLevel, "isActive", isActive)

	// 5. Return Metric
	metricValue := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(adjustedDemand), resource.DecimalSI), // KEDA typically uses whole numbers for metrics
		Timestamp:  metav1.Now(),
	}
	// If you need mili-units, use GenerateMetricInMili(metricName, adjustedDemand)

	return []external_metrics.ExternalMetricValue{metricValue}, isActive, nil
}

func (s *weatherAwareDemandScaler) GetMetricSpecForScaling(_ context.Context) []v2.MetricSpec {
	// Use the MetricName from metadata, normalized and prefixed with trigger index
	metricIdentifier := GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(s.metadata.MetricName))

	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: metricIdentifier,
		},
		Target: GetMetricTarget(s.metricType, s.metadata.TargetDemandPerReplica),
	}

	metricSpec := v2.MetricSpec{
		External: externalMetric,
		Type:     v2.ExternalMetricSourceType, // "External"
	}
	return []v2.MetricSpec{metricSpec}
}

// Close closes the http client
func (s *weatherAwareDemandScaler) Close(_ context.Context) error {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
		s.logger.V(1).Info("Closed idle HTTP connections for Weather-Aware Ride Demand Scaler")
	}
	return nil
}
