package scalers

import (
	"context"
	"fmt"
	"strconv"

	monitoringpb "cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/gcp"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type stackdriverScaler struct {
	client     *gcp.StackDriverClient
	metricType v2.MetricTargetType
	metadata   *stackdriverMetadata
	logger     logr.Logger
}

type stackdriverMetadata struct {
	ProjectID             string  `keda:"name=projectId, order=triggerMetadata"`
	Filter                string  `keda:"name=filter, order=triggerMetadata"`
	TargetValue           float64 `keda:"name=targetValue, order=triggerMetadata, default=5"`
	ActivationTargetValue float64 `keda:"name=activationTargetValue, order=triggerMetadata, default=0"`
	metricName            string
	ValueIfNull           *float64 `keda:"name=valueIfNull, order=triggerMetadata, optional"`
	FilterDuration        int64    `keda:"name=filterDuration, order=triggerMetadata, optional"`

	gcpAuthorization *gcp.AuthorizationMetadata
	aggregation      *monitoringpb.Aggregation
}

// NewStackdriverScaler creates a new stackdriverScaler
func NewStackdriverScaler(ctx context.Context, config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "gcp_stackdriver_scaler")

	meta, err := parseStackdriverMetadata(config, logger)
	if err != nil {
		return nil, fmt.Errorf("error parsing Stackdriver metadata: %w", err)
	}

	client, err := initializeStackdriverClient(ctx, meta.gcpAuthorization, logger)
	if err != nil {
		logger.Error(err, "Failed to create stack driver client")
		return nil, err
	}

	return &stackdriverScaler{
		metricType: metricType,
		metadata:   meta,
		client:     client,
		logger:     logger,
	}, nil
}

func parseStackdriverMetadata(config *scalersconfig.ScalerConfig, logger logr.Logger) (*stackdriverMetadata, error) {
	meta := &stackdriverMetadata{}

	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing Stackdriver metadata: %w", err)
	}

	name := kedautil.NormalizeString(fmt.Sprintf("gcp-stackdriver-%s", meta.ProjectID))
	meta.metricName = GenerateMetricNameWithIndex(config.TriggerIndex, name)

	auth, err := gcp.GetGCPAuthorization(config)
	if err != nil {
		return nil, err
	}
	meta.gcpAuthorization = auth

	aggregation, err := parseAggregation(config, logger)
	if err != nil {
		return nil, err
	}
	meta.aggregation = aggregation

	return meta, nil
}

func parseAggregation(config *scalersconfig.ScalerConfig, logger logr.Logger) (*monitoringpb.Aggregation, error) {
	if period, ok := config.TriggerMetadata["alignmentPeriodSeconds"]; ok {
		if period == "" {
			return nil, nil
		}

		val, err := strconv.ParseInt(period, 10, 64)
		if val < 60 {
			logger.Error(err, "Error parsing alignmentPeriodSeconds - must be at least 60")
			return nil, fmt.Errorf("error parsing alignmentPeriodSeconds - must be at least 60")
		}
		if err != nil {
			logger.Error(err, "Error parsing alignmentPeriodSeconds")
			return nil, fmt.Errorf("error parsing alignmentPeriodSeconds: %w", err)
		}

		return gcp.NewStackdriverAggregator(val, config.TriggerMetadata["alignmentAligner"], config.TriggerMetadata["alignmentReducer"])
	}

	return nil, nil
}

func initializeStackdriverClient(ctx context.Context, gcpAuthorization *gcp.AuthorizationMetadata, logger logr.Logger) (*gcp.StackDriverClient, error) {
	var client *gcp.StackDriverClient
	var err error
	if gcpAuthorization.PodIdentityProviderEnabled {
		client, err = gcp.NewStackDriverClientPodIdentity(ctx)
	} else {
		client, err = gcp.NewStackDriverClient(ctx, gcpAuthorization.GoogleApplicationCredentials)
	}

	if err != nil {
		logger.Error(err, "Failed to create stack driver client")
		return nil, err
	}
	return client, nil
}

func (s *stackdriverScaler) Close(context.Context) error {
	if s.client != nil {
		err := s.client.Close()
		s.client = nil
		if err != nil {
			s.logger.Error(err, "error closing StackDriver client")
		}
	}
	return nil
}

// GetMetricSpecForScaling returns the metric spec for the HPA
func (s *stackdriverScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: s.metadata.metricName,
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.TargetValue),
	}

	// Create the metric spec for the HPA
	metricSpec := v2.MetricSpec{
		External: externalMetric,
		Type:     externalMetricType,
	}

	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity connects to Stack Driver and retrieves the metric
func (s *stackdriverScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	value, err := s.getMetrics(ctx)
	if err != nil {
		s.logger.Error(err, "error getting metric value")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, value)

	return []external_metrics.ExternalMetricValue{metric}, value > s.metadata.ActivationTargetValue, nil
}

// getMetrics gets metric type value from stackdriver api
func (s *stackdriverScaler) getMetrics(ctx context.Context) (float64, error) {
	val, err := s.client.GetMetrics(ctx, s.metadata.Filter, s.metadata.ProjectID, s.metadata.aggregation, s.metadata.ValueIfNull, s.metadata.FilterDuration)
	if err == nil {
		s.logger.V(1).Info(
			fmt.Sprintf("Getting metrics for project %s, filter %s and aggregation %v. Result: %f",
				s.metadata.ProjectID,
				s.metadata.Filter,
				s.metadata.aggregation,
				val))
	}

	return val, err
}
