package scalers

import (
	"context"
	"fmt"
	"strconv"

	monitoringpb "cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	defaultStackdriverTargetValue = 5
)

type stackdriverScaler struct {
	client     *StackDriverClient
	metricType v2.MetricTargetType
	metadata   *stackdriverMetadata
	logger     logr.Logger
}

type stackdriverMetadata struct {
	projectID             string
	filter                string
	targetValue           float64
	activationTargetValue float64
	metricName            string
	valueIfNull           *float64

	gcpAuthorization *gcpAuthorizationMetadata
	aggregation      *monitoringpb.Aggregation
}

// NewStackdriverScaler creates a new stackdriverScaler
func NewStackdriverScaler(ctx context.Context, config *ScalerConfig) (Scaler, error) {
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

func parseStackdriverMetadata(config *ScalerConfig, logger logr.Logger) (*stackdriverMetadata, error) {
	meta := stackdriverMetadata{}
	meta.targetValue = defaultStackdriverTargetValue

	if val, ok := config.TriggerMetadata["projectId"]; ok {
		if val == "" {
			return nil, fmt.Errorf("no projectId name given")
		}

		meta.projectID = val
	} else {
		return nil, fmt.Errorf("no projectId name given")
	}

	if val, ok := config.TriggerMetadata["filter"]; ok {
		if val == "" {
			return nil, fmt.Errorf("no filter given")
		}

		meta.filter = val
	} else {
		return nil, fmt.Errorf("no filter given")
	}

	name := kedautil.NormalizeString(fmt.Sprintf("gcp-stackdriver-%s", meta.projectID))
	meta.metricName = GenerateMetricNameWithIndex(config.TriggerIndex, name)

	if val, ok := config.TriggerMetadata["targetValue"]; ok {
		targetValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			logger.Error(err, "Error parsing targetValue")
			return nil, fmt.Errorf("error parsing targetValue: %w", err)
		}

		meta.targetValue = targetValue
	}

	meta.activationTargetValue = 0
	if val, ok := config.TriggerMetadata["activationTargetValue"]; ok {
		activationTargetValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("activationTargetValue parsing error %w", err)
		}
		meta.activationTargetValue = activationTargetValue
	}

	if val, ok := config.TriggerMetadata["valueIfNull"]; ok && val != "" {
		valueIfNull, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("valueIfNull parsing error %w", err)
		}
		meta.valueIfNull = &valueIfNull
	}

	auth, err := getGCPAuthorization(config)
	if err != nil {
		return nil, err
	}
	meta.gcpAuthorization = auth

	aggregation, err := parseAggregation(config, logger)
	if err != nil {
		return nil, err
	}
	meta.aggregation = aggregation

	return &meta, nil
}

func parseAggregation(config *ScalerConfig, logger logr.Logger) (*monitoringpb.Aggregation, error) {
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

		return NewStackdriverAggregator(val, config.TriggerMetadata["alignmentAligner"], config.TriggerMetadata["alignmentReducer"])
	}

	return nil, nil
}

func initializeStackdriverClient(ctx context.Context, gcpAuthorization *gcpAuthorizationMetadata, logger logr.Logger) (*StackDriverClient, error) {
	var client *StackDriverClient
	var err error
	if gcpAuthorization.podIdentityProviderEnabled {
		client, err = NewStackDriverClientPodIdentity(ctx)
	} else {
		client, err = NewStackDriverClient(ctx, gcpAuthorization.GoogleApplicationCredentials)
	}

	if err != nil {
		logger.Error(err, "Failed to create stack driver client")
		return nil, err
	}
	return client, nil
}

func (s *stackdriverScaler) Close(context.Context) error {
	if s.client != nil {
		err := s.client.metricsClient.Close()
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
		Target: GetMetricTargetMili(s.metricType, s.metadata.targetValue),
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

	return []external_metrics.ExternalMetricValue{metric}, value > s.metadata.activationTargetValue, nil
}

// getMetrics gets metric type value from stackdriver api
func (s *stackdriverScaler) getMetrics(ctx context.Context) (float64, error) {
	val, err := s.client.GetMetrics(ctx, s.metadata.filter, s.metadata.projectID, s.metadata.aggregation, s.metadata.valueIfNull)
	if err == nil {
		s.logger.V(1).Info(
			fmt.Sprintf("Getting metrics for project %s, filter %s and aggregation %v. Result: %f",
				s.metadata.projectID,
				s.metadata.filter,
				s.metadata.aggregation,
				val))
	}

	return val, err
}
