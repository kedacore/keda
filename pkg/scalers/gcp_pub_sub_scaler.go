package scalers

import (
	"context"
	"fmt"
	"strconv"

	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedautil "github.com/kedacore/keda/pkg/util"
)

const (
	defaultTargetSubscriptionSize = 5
	pubSubStackDriverMetricName   = "pubsub.googleapis.com/subscription/num_undelivered_messages"
)

type pubsubScaler struct {
	client   *StackDriverClient
	metadata *pubsubMetadata
}

type pubsubMetadata struct {
	targetSubscriptionSize int
	subscriptionName       string
	credentials            string
}

var gcpPubSubLog = logf.Log.WithName("gcp_pub_sub_scaler")

// NewPubSubScaler creates a new pubsubScaler
func NewPubSubScaler(resolvedEnv, metadata map[string]string) (Scaler, error) {
	meta, err := parsePubSubMetadata(metadata, resolvedEnv)
	if err != nil {
		return nil, fmt.Errorf("error parsing PubSub metadata: %s", err)
	}

	return &pubsubScaler{
		metadata: meta,
	}, nil
}

func parsePubSubMetadata(metadata, resolvedEnv map[string]string) (*pubsubMetadata, error) {
	meta := pubsubMetadata{}
	meta.targetSubscriptionSize = defaultTargetSubscriptionSize

	if val, ok := metadata["subscriptionSize"]; ok {
		subscriptionSize, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("subscription Size parsing error %s", err.Error())
		}

		meta.targetSubscriptionSize = subscriptionSize
	}

	if val, ok := metadata["subscriptionName"]; ok {
		if val == "" {
			return nil, fmt.Errorf("no subscription name given")
		}

		meta.subscriptionName = val
	} else {
		return nil, fmt.Errorf("no subscription name given")
	}

	if metadata["credentialsFromEnv"] != "" {
		meta.credentials = resolvedEnv[metadata["credentialsFromEnv"]]
	}

	if len(meta.credentials) == 0 {
		return nil, fmt.Errorf("no credentials given. Need GCP service account credentials in json format")
	}

	return &meta, nil
}

// IsActive checks if there are any messages in the subscription
func (s *pubsubScaler) IsActive(ctx context.Context) (bool, error) {
	size, err := s.GetSubscriptionSize(ctx)

	if err != nil {
		gcpPubSubLog.Error(err, "error getting Active Status")
		return false, err
	}

	return size > 0, nil
}

func (s *pubsubScaler) Close() error {
	if s.client != nil {
		err := s.client.metricsClient.Close()
		if err != nil {
			gcpPubSubLog.Error(err, "error closing StackDriver client")
		}
	}

	return nil
}

// GetMetricSpecForScaling returns the metric spec for the HPA
func (s *pubsubScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	// Construct the target subscription size as a quantity
	targetSubscriptionSizeQty := resource.NewQuantity(int64(s.metadata.targetSubscriptionSize), resource.DecimalSI)

	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: kedautil.NormalizeString(fmt.Sprintf("%s-%s", "gcp", s.metadata.subscriptionName)),
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: targetSubscriptionSizeQty,
		},
	}

	// Create the metric spec for the HPA
	metricSpec := v2beta2.MetricSpec{
		External: externalMetric,
		Type:     externalMetricType,
	}

	return []v2beta2.MetricSpec{metricSpec}
}

// GetMetrics connects to Stack Driver and finds the size of the pub sub subscription
func (s *pubsubScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	size, err := s.GetSubscriptionSize(ctx)

	if err != nil {
		gcpPubSubLog.Error(err, "error getting subscription size")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(size, resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

// GetSubscriptionSize gets the number of messages in a subscription by calling the
// Stackdriver api
func (s *pubsubScaler) GetSubscriptionSize(ctx context.Context) (int64, error) {
	client, err := NewStackDriverClient(ctx, s.metadata.credentials)
	if err != nil {
		return -1, err
	}
	s.client = client

	filter := `metric.type="` + pubSubStackDriverMetricName + `" AND resource.labels.subscription_id="` + s.metadata.subscriptionName + `"`

	return client.GetMetrics(ctx, filter)
}
