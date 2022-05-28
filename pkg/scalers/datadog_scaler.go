package scalers

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	datadog "github.com/DataDog/datadog-api-client-go/api/v1/datadog"
	"k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type datadogScaler struct {
	metadata  *datadogMetadata
	apiClient *datadog.APIClient
}

type datadogMetadata struct {
	apiKey      string
	appKey      string
	datadogSite string
	query       string
	queryValue  int64
	vType       v2beta2.MetricTargetType
	metricName  string
	age         int
	useFiller   bool
	fillValue   float64
}

var datadogLog = logf.Log.WithName("datadog_scaler")

var filter *regexp.Regexp

func init() {
	filter = regexp.MustCompile(`.*\{.*\}.*`)
}

// NewDatadogScaler creates a new Datadog scaler
func NewDatadogScaler(ctx context.Context, config *ScalerConfig) (Scaler, error) {
	meta, err := parseDatadogMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing Datadog metadata: %s", err)
	}

	apiClient, err := newDatadogConnection(ctx, meta, config)
	if err != nil {
		return nil, fmt.Errorf("error establishing Datadog connection: %s", err)
	}
	return &datadogScaler{
		metadata:  meta,
		apiClient: apiClient,
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

func parseDatadogMetadata(config *ScalerConfig) (*datadogMetadata, error) {
	meta := datadogMetadata{}

	if val, ok := config.TriggerMetadata["age"]; ok {
		age, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("age parsing error %s", err.Error())
		}
		meta.age = age

		if age < 60 {
			datadogLog.Info("selecting a window smaller than 60 seconds can cause Datadog not finding a metric value for the query")
		}
	} else {
		meta.age = 90 // Default window 90 seconds
	}

	if val, ok := config.TriggerMetadata["query"]; ok {
		_, err := parseDatadogQuery(val)

		if err != nil {
			return nil, fmt.Errorf("error in query: %s", err.Error())
		}
		meta.query = val
	} else {
		return nil, fmt.Errorf("no query given")
	}

	if val, ok := config.TriggerMetadata["queryValue"]; ok {
		queryValue, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("queryValue parsing error %s", err.Error())
		}
		meta.queryValue = queryValue
	} else {
		return nil, fmt.Errorf("no queryValue given")
	}

	if val, ok := config.TriggerMetadata["metricUnavailableValue"]; ok {
		fillValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("metricUnavailableValue parsing error %s", err.Error())
		}
		meta.fillValue = fillValue
		meta.useFiller = true
	}

	if val, ok := config.TriggerMetadata["type"]; ok {
		datadogLog.V(0).Info("trigger.metadata.type is deprecated in favor of trigger.metricType")
		if config.MetricType != "" {
			return nil, fmt.Errorf("only one of trigger.metadata.type or trigger.metricType should be defined")
		}
		val = strings.ToLower(val)
		switch val {
		case "average":
			meta.vType = v2beta2.AverageValueMetricType
		case "global":
			meta.vType = v2beta2.ValueMetricType
		default:
			return nil, fmt.Errorf("type has to be global or average")
		}
	} else {
		metricType, err := GetMetricTargetType(config)
		if err != nil {
			return nil, fmt.Errorf("error getting scaler metric type: %s", err)
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
		return nil, fmt.Errorf("error connecting to Datadog API endpoint: %v", err)
	}

	return apiClient, nil
}

// No need to close connections
func (s *datadogScaler) Close(context.Context) error {
	return nil
}

// IsActive checks whether the scaler is active
func (s *datadogScaler) IsActive(ctx context.Context) (bool, error) {
	num, err := s.getQueryResult(ctx)

	if err != nil {
		return false, err
	}

	return num > 0, nil
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

	resp, r, err := s.apiClient.MetricsApi.QueryMetrics(ctx, time.Now().Unix()-int64(s.metadata.age), time.Now().Unix(), s.metadata.query) //nolint:bodyclose
	if err != nil {
		return -1, fmt.Errorf("error when retrieving Datadog metrics: %s", err)
	}

	if r.StatusCode == 429 {
		rateLimit := r.Header.Get("X-Ratelimit-Limit")
		rateLimitReset := r.Header.Get("X-Ratelimit-Reset")

		return -1, fmt.Errorf("your Datadog account reached the %s queries per hour rate limit, next limit reset will happen in %s seconds", rateLimit, rateLimitReset)
	}

	if r.StatusCode != 200 {
		return -1, fmt.Errorf("error when retrieving Datadog metrics")
	}

	if resp.GetStatus() == "error" {
		if msg, ok := resp.GetErrorOk(); ok {
			return -1, fmt.Errorf("error when retrieving Datadog metrics: %s", *msg)
		}
		return -1, fmt.Errorf("error when retrieving Datadog metrics")
	}

	series := resp.GetSeries()

	if len(series) > 1 {
		return 0, fmt.Errorf("query returned more than 1 series; modify the query to return only 1 series")
	}

	if len(series) == 0 {
		if !s.metadata.useFiller {
			return 0, fmt.Errorf("no Datadog metrics returned for the given time window")
		}
		return s.metadata.fillValue, nil
	}

	points := series[0].GetPointlist()

	if len(points) == 0 || len(points[0]) < 2 {
		if !s.metadata.useFiller {
			return 0, fmt.Errorf("no Datadog metrics returned for the given time window")
		}
		return s.metadata.fillValue, nil
	}

	// Return the last point from the series
	index := len(points) - 1
	return *points[index][1], nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *datadogScaler) GetMetricSpecForScaling(context.Context) []v2beta2.MetricSpec {
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: s.metadata.metricName,
		},
		Target: GetMetricTarget(s.metadata.vType, s.metadata.queryValue),
	}
	metricSpec := v2beta2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2beta2.MetricSpec{metricSpec}
}

// GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *datadogScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	num, err := s.getQueryResult(ctx)
	if err != nil {
		datadogLog.Error(err, "error getting metrics from Datadog")
		return []external_metrics.ExternalMetricValue{}, fmt.Errorf("error getting metrics from Datadog: %s", err)
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: s.metadata.metricName,
		Value:      *resource.NewMilliQuantity(int64(num*1000), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}
