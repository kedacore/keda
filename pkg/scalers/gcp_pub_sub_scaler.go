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

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	defaultTargetSubscriptionSize = 5
	pubSubStackDriverMetricName   = "pubsub.googleapis.com/subscription/num_undelivered_messages"
)

type gcpAuthorizationMetadata struct {
	GoogleApplicationCredentials string
	podIdentityOwner             bool
}

type pubsubScaler struct {
	client   *StackDriverClient
	metadata *pubsubMetadata
}

type pubsubMetadata struct {
	targetSubscriptionSize int
	subscriptionName       string
	gcpAuthorization       gcpAuthorizationMetadata
}

var gcpPubSubLog = logf.Log.WithName("gcp_pub_sub_scaler")

// NewPubSubScaler creates a new pubsubScaler
func NewPubSubScaler(config *ScalerConfig) (Scaler, error) {
	meta, err := parsePubSubMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing PubSub metadata: %s", err)
	}

	return &pubsubScaler{
		metadata: meta,
	}, nil
}

func parsePubSubMetadata(config *ScalerConfig) (*pubsubMetadata, error) {
	meta := pubsubMetadata{}
	meta.targetSubscriptionSize = defaultTargetSubscriptionSize

	if val, ok := config.TriggerMetadata["subscriptionSize"]; ok {
		subscriptionSize, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("subscription Size parsing error %s", err.Error())
		}

		meta.targetSubscriptionSize = subscriptionSize
	}

	if val, ok := config.TriggerMetadata["subscriptionName"]; ok {
		if val == "" {
			return nil, fmt.Errorf("no subscription name given")
		}

		meta.subscriptionName = val
	} else {
		return nil, fmt.Errorf("no subscription name given")
	}

	auth, err := getGcpAuthorization(config.AuthParams, config.TriggerMetadata, config.ResolvedEnv)
	if err != nil {
		return nil, err
	}
	meta.gcpAuthorization = *auth
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
		s.client = nil
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
	if s.client == nil {
		client, err := NewStackDriverClient(ctx, s.metadata.gcpAuthorization.GoogleApplicationCredentials)
		if err != nil {
			return -1, err
		}
		s.client = client
	}

	filter := `metric.type="` + pubSubStackDriverMetricName + `" AND resource.labels.subscription_id="` + s.metadata.subscriptionName + `"`

	return s.client.GetMetrics(ctx, filter)
}

func getGcpAuthorization(authParams, metadata, resolvedEnv map[string]string) (*gcpAuthorizationMetadata, error) {
	meta := gcpAuthorizationMetadata{}
	if metadata["identityOwner"] == "operator" {
		meta.podIdentityOwner = false
	} else if metadata["identityOwner"] == "" || metadata["identityOwner"] == "pod" {
		meta.podIdentityOwner = true
		if authParams["GoogleApplicationCredentials"] != "" {
			meta.GoogleApplicationCredentials = authParams["GoogleApplicationCredentials"]
		} else {
			if metadata["credentialsFromEnv"] != "" {
				meta.GoogleApplicationCredentials = resolvedEnv[metadata["credentialsFromEnv"]]
			} else {
				return nil, fmt.Errorf("GoogleApplicationCredentials not found")
			}
		}
	}
	return &meta, nil
}
