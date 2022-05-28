package scalers

import (
	"context"
	"fmt"
	"strconv"

	"k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	defaultStackdriverTargetValue = 5
)

type stackdriverScaler struct {
	client     *StackDriverClient
	metricType v2beta2.MetricTargetType
	metadata   *stackdriverMetadata
}

type stackdriverMetadata struct {
	projectID   string
	filter      string
	targetValue int64
	metricName  string

	gcpAuthorization *gcpAuthorizationMetadata
}

var gcpStackdriverLog = logf.Log.WithName("gcp_stackdriver_scaler")

// NewStackdriverScaler creates a new stackdriverScaler
func NewStackdriverScaler(ctx context.Context, config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %s", err)
	}

	meta, err := parseStackdriverMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing Stackdriver metadata: %s", err)
	}

	client, err := initializeStackdriverClient(ctx, meta.gcpAuthorization)
	if err != nil {
		gcpStackdriverLog.Error(err, "Failed to create stack driver client")
		return nil, err
	}

	return &stackdriverScaler{
		metricType: metricType,
		metadata:   meta,
		client:     client,
	}, nil
}

func parseStackdriverMetadata(config *ScalerConfig) (*stackdriverMetadata, error) {
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
			gcpStackdriverLog.Error(err, "Error parsing targetValue")
			return nil, fmt.Errorf("error parsing targetValue: %s", err.Error())
		}

		meta.targetValue = targetValue
	}

	auth, err := getGcpAuthorization(config, config.ResolvedEnv)
	if err != nil {
		return nil, err
	}
	meta.gcpAuthorization = auth
	return &meta, nil
}

func initializeStackdriverClient(ctx context.Context, gcpAuthorization *gcpAuthorizationMetadata) (*StackDriverClient, error) {
	var client *StackDriverClient
	var err error
	if gcpAuthorization.podIdentityProviderEnabled {
		client, err = NewStackDriverClientPodIdentity(ctx)
	} else {
		client, err = NewStackDriverClient(ctx, gcpAuthorization.GoogleApplicationCredentials)
	}

	if err != nil {
		gcpStackdriverLog.Error(err, "Failed to create stack driver client")
		return nil, err
	}
	return client, nil
}

func (s *stackdriverScaler) IsActive(ctx context.Context) (bool, error) {
	value, err := s.getMetrics(ctx)
	if err != nil {
		gcpStackdriverLog.Error(err, "error getting metric value")
		return false, err
	}
	return value > 0, nil
}

func (s *stackdriverScaler) Close(context.Context) error {
	if s.client != nil {
		err := s.client.metricsClient.Close()
		s.client = nil
		if err != nil {
			gcpStackdriverLog.Error(err, "error closing StackDriver client")
		}
	}

	return nil
}

// GetMetricSpecForScaling returns the metric spec for the HPA
func (s *stackdriverScaler) GetMetricSpecForScaling(context.Context) []v2beta2.MetricSpec {
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: s.metadata.metricName,
		},
		Target: GetMetricTarget(s.metricType, s.metadata.targetValue),
	}

	// Create the metric spec for the HPA
	metricSpec := v2beta2.MetricSpec{
		External: externalMetric,
		Type:     externalMetricType,
	}

	return []v2beta2.MetricSpec{metricSpec}
}

// GetMetrics connects to Stack Driver and retrieves the metric
func (s *stackdriverScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	value, err := s.getMetrics(ctx)
	if err != nil {
		gcpStackdriverLog.Error(err, "error getting metric value")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(value, resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

// getMetrics gets metric type value from stackdriver api
func (s *stackdriverScaler) getMetrics(ctx context.Context) (int64, error) {
	val, err := s.client.GetMetrics(ctx, s.metadata.filter, s.metadata.projectID)
	if err == nil {
		gcpStackdriverLog.V(1).Info(
			fmt.Sprintf("Getting metrics for project %s and filter %s. Result: %d", s.metadata.projectID, s.metadata.filter, val))
	}

	return val, err
}
