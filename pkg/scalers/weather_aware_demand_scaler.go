package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

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
	BadWeatherConditions string `keda:"name=badWeatherConditions,order=triggerMetadata,optional"`        // e.g., "temp_below:0,rain_above:5,wind_above:10" (temp in C, rain mm/hr, wind km/hr if metric)

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
	if m.WeatherAPIEndpoint != "" && m.WeatherLocation == "" {
		return fmt.Errorf("weatherLocation is required when weatherApiEndpoint is provided")
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

	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, false)

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

// isBadWeather evaluates if current weather conditions are "bad"
func (s *weatherAwareDemandScaler) isBadWeather(weatherData map[string]interface{}) (bool, error) {
	if s.metadata.BadWeatherConditions == "" || weatherData == nil {
		return false, nil // No conditions defined or no data, assume good weather
	}

	conditions := strings.Split(s.metadata.BadWeatherConditions, ",")
	for _, cond := range conditions {
		parts := strings.Split(cond, ":")
		if len(parts) != 2 {
			return false, fmt.Errorf("invalid bad weather condition format: %s", cond)
		}
		key := strings.TrimSpace(parts[0])
		valStr := strings.TrimSpace(parts[1])

		// Validate that the key has the proper format (ends with _below or _above)
		if !strings.HasSuffix(key, "_below") && !strings.HasSuffix(key, "_above") {
			return false, fmt.Errorf("invalid bad weather condition format: %s, must end with '_below' or '_above'", key)
		}

		// Example: temp_below:0, rain_above:5 (value from weatherData must be numeric)
		weatherVal, ok := weatherData[strings.Split(key, "_")[0]] // e.g., "temp" from "temp_below"
		if !ok {
			s.logger.V(1).Info("Weather key not found in weather data", "key", strings.Split(key, "_")[0])
			continue // Key not in weather data, skip this condition
		}

		weatherNum, ok := weatherVal.(float64) // Assuming weather values are numbers (e.g. temp: -5.0, rain: 10.0)
		if !ok {
			return false, fmt.Errorf("weather data for key '%s' is not a number: %v", strings.Split(key, "_")[0], weatherVal)
		}

		threshold, err := strconv.ParseFloat(valStr, 64)
		if err != nil {
			return false, fmt.Errorf("invalid threshold value in bad weather condition '%s': %w", cond, err)
		}

		if strings.HasSuffix(key, "_below") && weatherNum < threshold {
			return true, nil
		}
		if strings.HasSuffix(key, "_above") && weatherNum > threshold {
			return true, nil
		}
	}
	return false, nil
}

func (s *weatherAwareDemandScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	s.logger.V(1).Info("Fetching metrics for Weather-Aware Ride Demand Scaler")

	// 1. Fetch Demand Data
	var currentDemand float64
	if s.metadata.DemandAPIEndpoint != "" {
		var demandDataRaw interface{} // Use interface{} for raw JSON data
		err := s.fetchJSONData(ctx, s.metadata.DemandAPIEndpoint, s.metadata.DemandAPIKey, &demandDataRaw)
		if err != nil {
			s.logger.Error(err, "Failed to fetch demand data")
			return nil, false, fmt.Errorf("error fetching demand data: %w", err)
		}

		currentDemand, err = extractValueWithJSONPath(demandDataRaw, s.metadata.DemandJSONPath, s.logger)
		if err != nil {
			s.logger.Error(err, "Failed to extract demand value from response", "jsonPath", s.metadata.DemandJSONPath)
			return nil, false, fmt.Errorf("error extracting demand value: %w", err)
		}
		s.logger.V(1).Info("Successfully fetched demand data", "rawDemand", currentDemand)
	} else {
		currentDemand = 0 // Default demand if no endpoint
		s.logger.V(1).Info("DemandAPIEndpoint not configured, using default demand", "demand", currentDemand)
	}

	// 2. Fetch Weather Data
	var weatherData map[string]interface{} // Assuming weather API returns a flat JSON object

	if s.metadata.WeatherAPIEndpoint != "" {
		// Construct weather API URL with location and units
		// This is a simplified example. A real implementation would use url.Values for query params.
		weatherURL := fmt.Sprintf("%s?location=%s&units=%s", s.metadata.WeatherAPIEndpoint, s.metadata.WeatherLocation, s.metadata.WeatherUnits)

		err := s.fetchJSONData(ctx, weatherURL, s.metadata.WeatherAPIKeyFromEnv, &weatherData)
		if err != nil {
			s.logger.Error(err, "Failed to fetch weather data. Proceeding without weather adjustment or failing based on config.")
			// Depending on policy, we might want to return an error here or proceed with default weather.
			// For now, log and proceed, effectively assuming good weather or neutral impact.
		} else {
			s.logger.V(1).Info("Successfully fetched weather data", "data", weatherData)
		}
	} else {
		s.logger.V(1).Info("WeatherAPIEndpoint not configured, assuming good weather conditions.")
	}

	// 3. Apply Weather-Aware Logic
	adjustedDemand := currentDemand
	badWeather, err := s.isBadWeather(weatherData)
	if err != nil {
		s.logger.Error(err, "Failed to evaluate bad weather conditions, proceeding without weather adjustment.")
		// Potentially return error here if strict parsing of BadWeatherConditions is required
	}

	if badWeather {
		adjustedDemand = currentDemand * s.metadata.WeatherEffectScaleFactor
		s.logger.V(1).Info("Bad weather detected, adjusting demand", "originalDemand", currentDemand, "scaleFactor", s.metadata.WeatherEffectScaleFactor, "adjustedDemand", adjustedDemand)
	}

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
