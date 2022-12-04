package scalers

import (
	"context"
	"fmt"
	"strconv"

	monitoringpb "cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/labels"
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
	targetValue           int64
	activationTargetValue int64
	metricName            string

	gcpAuthorization *gcpAuthorizationMetadata
	aggregation      *monitoringpb.Aggregation
}

// NewStackdriverScaler creates a new stackdriverScaler
func NewStackdriverScaler(ctx context.Context, config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %s", err)
	}

	logger := InitializeLogger(config, "gcp_stackdriver_scaler")

	meta, err := parseStackdriverMetadata(config, logger)
	if err != nil {
		return nil, fmt.Errorf("error parsing Stackdriver metadata: %s", err)
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
	meta.metricName = GenerateMetricNameWithIndex(config.ScalerIndex, name)

	if val, ok := config.TriggerMetadata["targetValue"]; ok {
		targetValue, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			logger.Error(err, "Error parsing targetValue")
			return nil, fmt.Errorf("error parsing targetValue: %s", err.Error())
		}

		meta.targetValue = targetValue
	}

	meta.activationTargetValue = 0
	if val, ok := config.TriggerMetadata["activationTargetValue"]; ok {
		activationTargetValue, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("activationTargetValue parsing error %s", err.Error())
		}
		meta.activationTargetValue = activationTargetValue
	}

	auth, err := getGcpAuthorization(config, config.ResolvedEnv)
	if err != nil {
		return nil, err
	}
	meta.gcpAuthorization = auth

	meta.aggregation, err = parseAggregation(config, logger)
	if err != nil {
		return nil, err
	}

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
			return nil, fmt.Errorf("error parsing alignmentPeriodSeconds: %s", err.Error())
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

func (s *stackdriverScaler) IsActive(ctx context.Context) (bool, error) {
	value, err := s.getMetrics(ctx)
	if err != nil {
		s.logger.Error(err, "error getting metric value")
		return false, err
	}
	return value > s.metadata.activationTargetValue, nil
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
		Target: GetMetricTarget(s.metricType, s.metadata.targetValue),
	}

	// Create the metric spec for the HPA
	metricSpec := v2.MetricSpec{
		External: externalMetric,
		Type:     externalMetricType,
	}

	return []v2.MetricSpec{metricSpec}
}

// GetMetrics connects to Stack Driver and retrieves the metric
func (s *stackdriverScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	value, err := s.getMetrics(ctx)
	if err != nil {
		s.logger.Error(err, "error getting metric value")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := GenerateMetricInMili(metricName, float64(value))

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

// getMetrics gets metric type value from stackdriver api
func (s *stackdriverScaler) getMetrics(ctx context.Context) (int64, error) {
	val, err := s.client.GetMetrics(ctx, s.metadata.filter, s.metadata.projectID, s.metadata.aggregation)
	if err == nil {
		s.logger.V(1).Info(
			fmt.Sprintf("Getting metrics for project %s, filter %s and aggregation %v. Result: %d",
				s.metadata.projectID,
				s.metadata.filter,
				s.metadata.aggregation,
				val))
	}

	return val, err
}
