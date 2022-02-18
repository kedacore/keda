package scalers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	datadog "github.com/DataDog/datadog-api-client-go/api/v1/datadog"
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
	queryValue  int
	vType       v2beta2.MetricTargetType
	metricName  string
	age         int
}

var datadogLog = logf.Log.WithName("datadog_scaler")

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

func parseDatadogMetadata(config *ScalerConfig) (*datadogMetadata, error) {
	meta := datadogMetadata{}

	if val, ok := config.TriggerMetadata["query"]; ok {
		meta.query = val
	} else {
		return nil, fmt.Errorf("no query given")
	}
	if !strings.Contains(meta.query, "{") {
		return nil, fmt.Errorf("expecting query to contain `{`, got %s", meta.query)
	}

	if val, ok := config.TriggerMetadata["queryValue"]; ok {
		queryValue, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("queryValue parsing error %s", err.Error())
		}
		meta.queryValue = queryValue
	} else {
		return nil, fmt.Errorf("no queryValue given")
	}

	if val, ok := config.TriggerMetadata["age"]; ok {
		age, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("age parsing error %s", err.Error())
		}
		meta.age = age
	} else {
		meta.age = 90 // Default window 90 seconds
	}

	// For all the points in a given window, we take the rollup to the window size
	rollup := fmt.Sprintf(".rollup(avg, %d)", meta.age)
	meta.query += rollup

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

	_, _, err := apiClient.AuthenticationApi.Validate(ctx)
	if err != nil {
		return nil, fmt.Errorf("error connecting to Datadog API endpoint: %v", err)
	}

	return apiClient, nil
}

// No need to close connections
func (s *datadogScaler) Close(context.Context) error {
	return nil
}

// IsActive returns true if we are able to get metrics from Datadog
func (s *datadogScaler) IsActive(ctx context.Context) (bool, error) {
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

	resp, _, err := s.apiClient.MetricsApi.QueryMetrics(ctx, time.Now().Unix()-int64(s.metadata.age), time.Now().Unix(), s.metadata.query)

	if err != nil {
		return false, err
	}

	series := resp.GetSeries()

	if len(series) == 0 {
		return false, nil
	}

	points := series[0].GetPointlist()

	if len(points) == 0 {
		return false, nil
	}

	return true, nil
}

// getQueryResult returns result of the scaler query
func (s *datadogScaler) getQueryResult(ctx context.Context) (int, error) {
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

	resp, _, err := s.apiClient.MetricsApi.QueryMetrics(ctx, time.Now().Unix()-int64(s.metadata.age), time.Now().Unix(), s.metadata.query)
	if err != nil {
		return -1, fmt.Errorf("error when retrieving Datadog metrics: %s", err)
	}

	series := resp.GetSeries()

	if len(series) == 0 {
		return 0, fmt.Errorf("no Datadog metrics returned")
	}

	points := series[0].GetPointlist()

	if len(points) == 0 || len(points[0]) < 2 {
		return 0, fmt.Errorf("no Datadog metrics returned")
	}

	return int(*points[0][1]), nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *datadogScaler) GetMetricSpecForScaling(context.Context) []v2beta2.MetricSpec {
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: s.metadata.metricName,
		},
		Target: GetMetricTarget(s.metadata.vType, int64(s.metadata.queryValue)),
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
		Value:      *resource.NewQuantity(int64(num), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}
