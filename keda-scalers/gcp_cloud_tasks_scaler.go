package scalers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/keda-scalers/gcp"
	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	cloudTasksStackDriverQueueSize = "cloudtasks.googleapis.com/queue/depth"
)

type gcpCloudTasksScaler struct {
	client     *gcp.StackDriverClient
	metricType v2.MetricTargetType
	metadata   *gcpCloudTaskMetadata
	logger     logr.Logger
}

type gcpCloudTaskMetadata struct {
	Value           float64 `keda:"name=value, order=triggerMetadata, optional, default=100"`
	ActivationValue float64 `keda:"name=activationValue, order=triggerMetadata, optional, default=0"`
	FilterDuration  int64   `keda:"name=filterDuration, order=triggerMetadata, optional"`

	QueueName        string `keda:"name=queueName, order=triggerMetadata"`
	ProjectID        string `keda:"name=projectID, order=triggerMetadata"`
	gcpAuthorization *gcp.AuthorizationMetadata
	triggerIndex     int
}

// NewGcpCloudTasksScaler creates a new cloudTaskScaler
func NewGcpCloudTasksScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "gcp_cloud_tasks_scaler")

	meta, err := parseGcpCloudTasksMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing Cloud Tasks metadata: %w", err)
	}

	return &gcpCloudTasksScaler{
		metricType: metricType,
		metadata:   meta,
		logger:     logger,
	}, nil
}

func parseGcpCloudTasksMetadata(config *scalersconfig.ScalerConfig) (*gcpCloudTaskMetadata, error) {
	meta := &gcpCloudTaskMetadata{}
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing Gcp cloud task metadata: %w", err)
	}

	auth, err := gcp.GetGCPAuthorization(config)
	if err != nil {
		return nil, err
	}

	meta.gcpAuthorization = auth
	meta.triggerIndex = config.TriggerIndex
	return meta, nil
}

func (s *gcpCloudTasksScaler) Close(context.Context) error {
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
func (s *gcpCloudTasksScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("gcp-ct-%s", s.metadata.QueueName))),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.Value),
	}

	// Create the metric spec for the HPA
	metricSpec := v2.MetricSpec{
		External: externalMetric,
		Type:     externalMetricType,
	}

	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity connects to Stack Driver and finds the size of the cloud task
func (s *gcpCloudTasksScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	metricType := cloudTasksStackDriverQueueSize

	value, err := s.getMetrics(ctx, metricType)
	if err != nil {
		s.logger.Error(err, "error getting metric", "metricType", metricType)
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, value)

	return []external_metrics.ExternalMetricValue{metric}, value > s.metadata.ActivationValue, nil
}

func (s *gcpCloudTasksScaler) setStackdriverClient(ctx context.Context) error {
	var client *gcp.StackDriverClient
	var err error
	if s.metadata.gcpAuthorization.PodIdentityProviderEnabled {
		client, err = gcp.NewStackDriverClientPodIdentity(ctx)
	} else {
		client, err = gcp.NewStackDriverClient(ctx, s.metadata.gcpAuthorization.GoogleApplicationCredentials)
	}

	if err != nil {
		return err
	}
	s.client = client
	return nil
}

// getMetrics gets metric type value from stackdriver api
func (s *gcpCloudTasksScaler) getMetrics(ctx context.Context, metricType string) (float64, error) {
	if s.client == nil {
		err := s.setStackdriverClient(ctx)
		if err != nil {
			return -1, err
		}
	}
	filter := `metric.type="` + metricType + `" AND resource.labels.queue_id="` + s.metadata.QueueName + `"`

	// Cloud Tasks metrics are collected every 60 seconds so no need to aggregate them.
	// See: https://cloud.google.com/monitoring/api/metrics_gcp#gcp-cloudtasks
	return s.client.GetMetrics(ctx, filter, s.metadata.ProjectID, nil, nil, s.metadata.FilterDuration)
}
