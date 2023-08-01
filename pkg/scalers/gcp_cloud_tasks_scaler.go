package scalers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	cloudTasksStackDriverQueueSize = "cloudtasks.googleapis.com/queue/depth"

	cloudTaskDefaultValue = 100
)

type cloudTasksScaler struct {
	client     *StackDriverClient
	metricType v2.MetricTargetType
	metadata   *cloudTaskMetadata
	logger     logr.Logger
}

type cloudTaskMetadata struct {
	value           float64
	activationValue float64

	queueName        string
	projectID        string
	gcpAuthorization *gcpAuthorizationMetadata
	scalerIndex      int
}

// NewCloudTaskScaler creates a new cloudTaskScaler
func NewCloudTasksScaler(config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "gcp_cloud_tasks_scaler")

	meta, err := parseCloudTasksMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing Cloud Tasks metadata: %w", err)
	}

	return &cloudTasksScaler{
		metricType: metricType,
		metadata:   meta,
		logger:     logger,
	}, nil
}

func parseCloudTasksMetadata(config *ScalerConfig) (*cloudTaskMetadata, error) {
	meta := cloudTaskMetadata{value: cloudTaskDefaultValue}

	value, valuePresent := config.TriggerMetadata["value"]

	if valuePresent {
		triggerValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, fmt.Errorf("value parsing error %w", err)
		}
		meta.value = triggerValue
	}

	if val, ok := config.TriggerMetadata["queueName"]; ok {
		if val == "" {
			return nil, fmt.Errorf("no queue name given")
		}

		meta.queueName = val
	} else {
		return nil, fmt.Errorf("no queue name given")
	}

	meta.activationValue = 0
	if val, ok := config.TriggerMetadata["activationValue"]; ok {
		activationValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("activationValue parsing error %w", err)
		}
		meta.activationValue = activationValue
	}

	if val, ok := config.TriggerMetadata["projectID"]; ok {
		if val == "" {
			return nil, fmt.Errorf("no project id given")
		}

		meta.projectID = val
	} else {
		return nil, fmt.Errorf("no project id given")
	}

	auth, err := getGCPAuthorization(config)
	if err != nil {
		return nil, err
	}
	meta.gcpAuthorization = auth
	meta.scalerIndex = config.ScalerIndex
	return &meta, nil
}

func (s *cloudTasksScaler) Close(context.Context) error {
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
func (s *cloudTasksScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, kedautil.NormalizeString(fmt.Sprintf("gcp-ct-%s", s.metadata.queueName))),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.value),
	}

	// Create the metric spec for the HPA
	metricSpec := v2.MetricSpec{
		External: externalMetric,
		Type:     externalMetricType,
	}

	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity connects to Stack Driver and finds the size of the cloud task
func (s *cloudTasksScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	metricType := cloudTasksStackDriverQueueSize

	value, err := s.getMetrics(ctx, metricType)
	if err != nil {
		s.logger.Error(err, "error getting metric", "metricType", metricType)
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, value)

	return []external_metrics.ExternalMetricValue{metric}, value > s.metadata.activationValue, nil
}

func (s *cloudTasksScaler) setStackdriverClient(ctx context.Context) error {
	var client *StackDriverClient
	var err error
	if s.metadata.gcpAuthorization.podIdentityProviderEnabled {
		client, err = NewStackDriverClientPodIdentity(ctx)
	} else {
		client, err = NewStackDriverClient(ctx, s.metadata.gcpAuthorization.GoogleApplicationCredentials)
	}

	if err != nil {
		return err
	}
	s.client = client
	return nil
}

// getMetrics gets metric type value from stackdriver api
func (s *cloudTasksScaler) getMetrics(ctx context.Context, metricType string) (float64, error) {
	if s.client == nil {
		err := s.setStackdriverClient(ctx)
		if err != nil {
			return -1, err
		}
	}
	filter := `metric.type="` + metricType + `" AND resource.labels.queue_id="` + s.metadata.queueName + `"`

	// Cloud Tasks metrics are collected every 60 seconds so no need to aggregate them.
	// See: https://cloud.google.com/monitoring/api/metrics_gcp#gcp-cloudtasks
	return s.client.GetMetrics(ctx, filter, s.metadata.projectID, nil)
}
