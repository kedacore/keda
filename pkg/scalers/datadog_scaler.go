package scalers

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	datadog "github.com/DataDog/datadog-api-client-go/api/v1/datadog"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type datadogScaler struct {
	metadata  *datadogMetadata
	apiClient *datadog.APIClient
	logger    logr.Logger
}

type datadogMetadata struct {
	apiKey                   string
	appKey                   string
	datadogSite              string
	query                    string
	queryValue               float64
	queryAggegrator          string
	activationQueryValue     float64
	vType                    v2.MetricTargetType
	metricName               string
	age                      int
	timeWindowOffset         int
	lastAvailablePointOffset int
	useFiller                bool
	fillValue                float64
}

const maxString = "max"
const avgString = "average"

var filter *regexp.Regexp

func init() {
	filter = regexp.MustCompile(`.*\{.*\}.*`)
}

// NewDatadogScaler creates a new Datadog scaler
func NewDatadogScaler(ctx context.Context, config *ScalerConfig) (Scaler, error) {
	logger := InitializeLogger(config, "datadog_scaler")

	meta, err := parseDatadogMetadata(config, logger)
	if err != nil {
		return nil, fmt.Errorf("error parsing Datadog metadata: %w", err)
	}

	apiClient, err := newDatadogConnection(ctx, meta, config)
	if err != nil {
		return nil, fmt.Errorf("error establishing Datadog connection: %w", err)
	}
	return &datadogScaler{
		metadata:  meta,
		apiClient: apiClient,
		logger:    logger,
	}, nil
}

// parseDatadogQuery checks correctness of the user query
func parseDatadogQuery(q string) (bool, error) {
	// Wellformed Datadog queries require a filter (between curly brackets)
	if !filter.MatchString(q) {
		return false, fmt.Errorf("malformed Datadog query: missing query scope")
	}

	return true, nil
}

func parseDatadogMetadata(config *ScalerConfig, logger logr.Logger) (*datadogMetadata, error) {
	meta := datadogMetadata{}

	if val, ok := config.TriggerMetadata["age"]; ok {
		age, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("age parsing error %w", err)
		}
		meta.age = age

		if age < 0 {
			return nil, fmt.Errorf("age should not be smaller than 0 seconds")
		}
		if age < 60 {
			logger.Info("selecting a window smaller than 60 seconds can cause Datadog not finding a metric value for the query")
		}
	} else {
		meta.age = 90 // Default window 90 seconds
	}

	if val, ok := config.TriggerMetadata["timeWindowOffset"]; ok {
		timeWindowOffset, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("timeWindowOffset parsing error %w", err)
		}
		if timeWindowOffset < 0 {
			return nil, fmt.Errorf("timeWindowOffset should not be smaller than 0 seconds")
		}
		meta.timeWindowOffset = timeWindowOffset
	} else {
		meta.timeWindowOffset = 0 // Default delay 0 seconds
	}

	if val, ok := config.TriggerMetadata["lastAvailablePointOffset"]; ok {
		lastAvailablePointOffset, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("lastAvailablePointOffset parsing error %w", err)
		}

		if lastAvailablePointOffset < 0 {
			return nil, fmt.Errorf("lastAvailablePointOffset should not be smaller than 0")
		}
		meta.lastAvailablePointOffset = lastAvailablePointOffset
	} else {
		meta.lastAvailablePointOffset = 0 // Default use the last point
	}

	if val, ok := config.TriggerMetadata["query"]; ok {
		_, err := parseDatadogQuery(val)

		if err != nil {
			return nil, fmt.Errorf("error in query: %w", err)
		}
		meta.query = val
	} else {
		return nil, fmt.Errorf("no query given")
	}

	if val, ok := config.TriggerMetadata["queryValue"]; ok {
		queryValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("queryValue parsing error %w", err)
		}
		meta.queryValue = queryValue
	} else {
		if config.AsMetricSource {
			meta.queryValue = 0
		} else {
			return nil, fmt.Errorf("no queryValue given")
		}
	}

	if val, ok := config.TriggerMetadata["queryAggregator"]; ok && val != "" {
		queryAggregator := strings.ToLower(val)
		switch queryAggregator {
		case avgString, maxString:
			meta.queryAggegrator = queryAggregator
		default:
			return nil, fmt.Errorf("queryAggregator value %s has to be one of '%s, %s'", queryAggregator, avgString, maxString)
		}
	} else {
		meta.queryAggegrator = ""
	}

	meta.activationQueryValue = 0
	if val, ok := config.TriggerMetadata["activationQueryValue"]; ok {
		activationQueryValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("queryValue parsing error %w", err)
		}
		meta.activationQueryValue = activationQueryValue
	}

	if val, ok := config.TriggerMetadata["metricUnavailableValue"]; ok {
		fillValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("metricUnavailableValue parsing error %w", err)
		}
		meta.fillValue = fillValue
		meta.useFiller = true
	}

	if val, ok := config.TriggerMetadata["type"]; ok {
		logger.V(0).Info("trigger.metadata.type is deprecated in favor of trigger.metricType")
		if config.MetricType != "" {
			return nil, fmt.Errorf("only one of trigger.metadata.type or trigger.metricType should be defined")
		}
		val = strings.ToLower(val)
		switch val {
		case avgString:
			meta.vType = v2.AverageValueMetricType
		case "global":
			meta.vType = v2.ValueMetricType
		default:
			return nil, fmt.Errorf("type has to be global or average")
		}
	} else {
		metricType, err := GetMetricTargetType(config)
		if err != nil {
			return nil, fmt.Errorf("error getting scaler metric type: %w", err)
		}
		meta.vType = metricType
	}

	if val, ok := config.AuthParams["apiKey"]; ok {
		meta.apiKey = val
	} else {
		return nil, fmt.Errorf("no api key given")
	}

	if val, ok := config.AuthParams["appKey"]; ok {
		meta.appKey = val
	} else {
		return nil, fmt.Errorf("no app key given")
	}

	siteVal := "datadoghq.com"

	if val, ok := config.AuthParams["datadogSite"]; ok && val != "" {
		siteVal = val
	}

	meta.datadogSite = siteVal

	metricName := meta.query[0:strings.Index(meta.query, "{")]
	meta.metricName = GenerateMetricNameWithIndex(config.ScalerIndex, kedautil.NormalizeString(fmt.Sprintf("datadog-%s", metricName)))

	return &meta, nil
}

// newDatddogConnection tests a connection to the Datadog API
func newDatadogConnection(ctx context.Context, meta *datadogMetadata, config *ScalerConfig) (*datadog.APIClient, error) {
	ctx = context.WithValue(
		ctx,
		datadog.ContextAPIKeys,
		map[string]datadog.APIKey{
			"apiKeyAuth": {
				Key: meta.apiKey,
			},
			"appKeyAuth": {
				Key: meta.appKey,
			},
		},
	)

	ctx = context.WithValue(ctx,
		datadog.ContextServerVariables,
		map[string]string{
			"site": meta.datadogSite,
		})

	configuration := datadog.NewConfiguration()
	configuration.HTTPClient = kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, false)
	apiClient := datadog.NewAPIClient(configuration)

	_, _, err := apiClient.AuthenticationApi.Validate(ctx) //nolint:bodyclose
	if err != nil {
		return nil, fmt.Errorf("error connecting to Datadog API endpoint: %w", err)
	}

	return apiClient, nil
}

// No need to close connections
func (s *datadogScaler) Close(context.Context) error {
	return nil
}

// getQueryResult returns result of the scaler query
func (s *datadogScaler) getQueryResult(ctx context.Context) (float64, error) {
	ctx = context.WithValue(
		ctx,
		datadog.ContextAPIKeys,
		map[string]datadog.APIKey{
			"apiKeyAuth": {
				Key: s.metadata.apiKey,
			},
			"appKeyAuth": {
				Key: s.metadata.appKey,
			},
		},
	)

	ctx = context.WithValue(ctx,
		datadog.ContextServerVariables,
		map[string]string{
			"site": s.metadata.datadogSite,
		})

	timeWindowTo := time.Now().Unix() - int64(s.metadata.timeWindowOffset)
	timeWindowFrom := timeWindowTo - int64(s.metadata.age)
	resp, r, err := s.apiClient.MetricsApi.QueryMetrics(ctx, timeWindowFrom, timeWindowTo, s.metadata.query) //nolint:bodyclose

	if r != nil {
		if r.StatusCode == 429 {
			rateLimit := r.Header.Get("X-Ratelimit-Limit")
			rateLimitReset := r.Header.Get("X-Ratelimit-Reset")
			rateLimitPeriod := r.Header.Get("X-Ratelimit-Period")

			return -1, fmt.Errorf("your Datadog account reached the %s queries per %s seconds rate limit, next limit reset will happen in %s seconds", rateLimit, rateLimitPeriod, rateLimitReset)
		}

		if r.StatusCode != 200 {
			if err != nil {
				return -1, fmt.Errorf("error when retrieving Datadog metrics: %w", err)
			}
			return -1, fmt.Errorf("error when retrieving Datadog metrics")
		}
	}

	if err != nil {
		return -1, fmt.Errorf("error when retrieving Datadog metrics: %w", err)
	}

	if resp.GetStatus() == "error" {
		if msg, ok := resp.GetErrorOk(); ok {
			return -1, fmt.Errorf("error when retrieving Datadog metrics: %s", *msg)
		}
		return -1, fmt.Errorf("error when retrieving Datadog metrics")
	}

	series := resp.GetSeries()

	if len(series) == 0 {
		if !s.metadata.useFiller {
			return 0, fmt.Errorf("no Datadog metrics returned for the given time window")
		}
		return s.metadata.fillValue, nil
	}

	// Require queryAggregator be set explicitly for multi-query
	if len(series) > 1 && s.metadata.queryAggegrator == "" {
		return 0, fmt.Errorf("query returned more than 1 series; modify the query to return only 1 series or add a queryAggregator")
	}

	// Collect all latest point values from any/all series
	results := make([]float64, len(series))
	for i := 0; i < len(series); i++ {
		points := series[i].GetPointlist()
		index := len(points) - 1
		// Find out the last point != nil
		for j := index; j >= 0; j-- {
			if len(points[j]) >= 2 && points[j][1] != nil {
				index = j
				break
			}
		}
		if index < s.metadata.lastAvailablePointOffset {
			return 0, fmt.Errorf("index is smaller than the lastAvailablePointOffset")
		}
		index -= s.metadata.lastAvailablePointOffset

		if len(points) == 0 || len(points[index]) < 2 || points[index][1] == nil {
			if !s.metadata.useFiller {
				return 0, fmt.Errorf("no Datadog metrics returned for the given time window")
			}
			return s.metadata.fillValue, nil
		}
		// Return the last point from the series
		results[i] = *points[index][1]
	}

	switch s.metadata.queryAggegrator {
	case avgString:
		return AvgFloatFromSlice(results), nil
	default:
		// Aggregate Results - default Max value:
		return MaxFloatFromSlice(results), nil
	}
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *datadogScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: s.metadata.metricName,
		},
		Target: GetMetricTargetMili(s.metadata.vType, s.metadata.queryValue),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *datadogScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	num, err := s.getQueryResult(ctx)
	if err != nil {
		s.logger.Error(err, "error getting metrics from Datadog")
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error getting metrics from Datadog: %w", err)
	}

	metric := GenerateMetricInMili(metricName, num)

	return []external_metrics.ExternalMetricValue{metric}, num > s.metadata.activationQueryValue, nil
}

// MaxFloatFromSlice finds the largest value in a slice of floats
func MaxFloatFromSlice(results []float64) float64 {
	max := results[0]
	for _, result := range results {
		if result > max {
			max = result
		}
	}
	return max
}

// AvgFloatFromSlice finds the average value in a slice of floats
func AvgFloatFromSlice(results []float64) float64 {
	total := 0.0
	for _, result := range results {
		total += result
	}
	return total / float64(len(results))
}
