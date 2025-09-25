package scalers

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/solarwinds/swo-sdk-go/swov1"
	"github.com/solarwinds/swo-sdk-go/swov1/models/components"
	"github.com/solarwinds/swo-sdk-go/swov1/models/operations"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type solarWindsScaler struct {
	metricType v2.MetricTargetType
	metadata   *solarWindsMetadata
}

type solarWindsMetadata struct {
	APIToken        string  `keda:"name=apiToken,        order=authParams, optional"`
	Host            string  `keda:"name=host,            order=triggerMetadata"`
	TargetValue     float64 `keda:"name=targetValue,     order=triggerMetadata"`
	ActivationValue float64 `keda:"name=activationValue, order=triggerMetadata"`
	MetricName      string  `keda:"name=metricName,      order=triggerMetadata"`
	Aggregation     string  `keda:"name=aggregation,     order=triggerMetadata, enum=COUNT;MIN;MAX;AVG;SUM;LAST"`
	IntervalS       int     `keda:"name=intervalS,       order=triggerMetadata"`
	Filter          string  `keda:"name=filter,          order=triggerMetadata, optional"`

	triggerIndex int
}

func NewSolarWindsScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, err := parseSolarWindsMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing SolarWinds metadata: %w", err)
	}

	return &solarWindsScaler{
		metricType: metricType,
		metadata:   meta,
	}, nil
}

func parseSolarWindsMetadata(config *scalersconfig.ScalerConfig) (*solarWindsMetadata, error) {
	meta := &solarWindsMetadata{}
	meta.triggerIndex = config.TriggerIndex

	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing solarwinds metadata: %w", err)
	}

	_, err := url.ParseRequestURI(meta.Host)
	if err != nil {
		return meta, errors.New("invalid value for host. Must be a valid URL such as 'https://api.na-01.cloud.solarwinds.com'")
	}

	return meta, nil
}

func (s *solarWindsScaler) Close(context.Context) error {
	return nil
}

func (s *solarWindsScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, "solarwinds"),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.TargetValue),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: "External"}
	return []v2.MetricSpec{metricSpec}
}

func (s *solarWindsScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	value, err := s.getMetricValueViaSDK(ctx, metricName)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error getting SolarWinds metric: %w", err)
	}

	metric := GenerateMetricInMili(metricName, value)

	return []external_metrics.ExternalMetricValue{metric}, value > s.metadata.ActivationValue, nil
}

func (s *solarWindsScaler) getMetricValueViaSDK(ctx context.Context, _ string) (float64, error) {
	session := swov1.New(
		swov1.WithSecurity(s.metadata.APIToken),
		swov1.WithServerURL(s.metadata.Host),
	)

	now := time.Now()
	startTime := now.Add(-time.Duration(s.metadata.IntervalS) * time.Second).UTC()
	endTime := now.UTC()
	var filter *string
	if s.metadata.Filter != "" {
		filter = &s.metadata.Filter
	}

	// Use the actual SolarWinds metric name, not the KEDA-generated metric name
	res, err := session.Metrics.ListMetricMeasurements(ctx, operations.ListMetricMeasurementsRequest{
		Name:        s.metadata.MetricName,
		Filter:      filter,
		AggregateBy: s.convertAggregation(s.metadata.Aggregation),
		StartTime:   &startTime,
		EndTime:     &endTime,
		SeriesType:  components.MetricSeriesTypeScalar,
	})

	if err != nil {
		return 0, err
	}

	return s.getFirstMeasurement(res)
}

func (s *solarWindsScaler) convertAggregation(aggregation string) *components.MetricsAggregationMethods {
	aggregation = strings.ToUpper(aggregation)
	switch aggregation {
	case "COUNT":
		return components.MetricsAggregationMethodsCount.ToPointer()
	case "MIN":
		return components.MetricsAggregationMethodsMin.ToPointer()
	case "MAX":
		return components.MetricsAggregationMethodsMax.ToPointer()
	case "AVG":
		return components.MetricsAggregationMethodsAvg.ToPointer()
	case "SUM":
		return components.MetricsAggregationMethodsSum.ToPointer()
	case "LAST":
		return components.MetricsAggregationMethodsLast.ToPointer()
	default:
		return nil
	}
}

func (s *solarWindsScaler) getFirstMeasurement(res *operations.ListMetricMeasurementsResponse) (float64, error) {
	var firstValue float64
	if res.Object != nil {
		if res.Object.Groupings == nil {
			return 0, fmt.Errorf("no groupings found in response")
		}
		for _, group := range res.Object.Groupings {
			if group.Measurements == nil {
				return 0, fmt.Errorf("no measurements found in response")
			}

			for _, measurement := range group.Measurements {
				firstValue = measurement.Value
				return firstValue, nil
			}
		}
	}
	return 0, fmt.Errorf("no measurements found in response")
}
