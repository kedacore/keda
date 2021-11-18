package scalers

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	defaultTargetSubscriptionSize                      = 5
	defaultTargetOldestUnackedMessageAge               = 10
	pubSubStackDriverSubscriptionSizeMetricName        = "pubsub.googleapis.com/subscription/num_undelivered_messages"
	pubSubStackDriverOldestUnackedMessageAgeMetricName = "pubsub.googleapis.com/subscription/oldest_unacked_message_age"

	pubsubModeSubscriptionSize        = "SubscriptionSize"
	pubsubModeOldestUnackedMessageAge = "OldestUnackedMessageAge"
)

type gcpAuthorizationMetadata struct {
	GoogleApplicationCredentials string
	podIdentityOwner             bool
	podIdentityProviderEnabled   bool
}

type pubsubScaler struct {
	client   *StackDriverClient
	metadata *pubsubMetadata
}

type pubsubMetadata struct {
	mode  string
	value int

	// deprecated
	subscriptionSize int

	subscriptionName string
	gcpAuthorization gcpAuthorizationMetadata
	scalerIndex      int
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
	meta.mode = pubsubModeSubscriptionSize

	if subSize, subSizePresent := config.TriggerMetadata["subscriptionSize"]; subSizePresent {
		gcpPubSubLog.Info("subscriptionSize field is deprecated. Use mode and value fields instead")
		meta.mode = pubsubModeSubscriptionSize
		subSizeValue, err := strconv.Atoi(subSize)
		if err != nil {
			return nil, fmt.Errorf("value parsing error %s", err.Error())
		}
		meta.value = subSizeValue
	} else {
		mode, modePresent := config.TriggerMetadata["mode"]
		if modePresent {
			meta.mode = mode
		}

		switch meta.mode {
		case pubsubModeSubscriptionSize:
			meta.value = defaultTargetSubscriptionSize
		case pubsubModeOldestUnackedMessageAge:
			meta.value = defaultTargetOldestUnackedMessageAge
		default:
			return nil, fmt.Errorf("trigger mode %s must be one of %s, %s", meta.mode, pubsubModeSubscriptionSize, pubsubModeOldestUnackedMessageAge)
		}

		if val, ok := config.TriggerMetadata["value"]; ok {
			triggerValue, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("value parsing error %s", err.Error())
			}
			meta.value = triggerValue
		}
	}

	if val, ok := config.TriggerMetadata["subscriptionName"]; ok {
		if val == "" {
			return nil, fmt.Errorf("no subscription name given")
		}

		meta.subscriptionName = val
	} else {
		return nil, fmt.Errorf("no subscription name given")
	}

	auth, err := getGcpAuthorization(config, config.ResolvedEnv)
	if err != nil {
		return nil, err
	}
	meta.gcpAuthorization = *auth
	meta.scalerIndex = config.ScalerIndex
	return &meta, nil
}

// IsActive checks if there are any messages in the subscription
func (s *pubsubScaler) IsActive(ctx context.Context) (bool, error) {
	switch s.metadata.mode {
	case pubsubModeSubscriptionSize:
		size, err := s.getMetrics(ctx, pubSubStackDriverSubscriptionSizeMetricName)
		if err != nil {
			gcpPubSubLog.Error(err, "error getting Active Status")
			return false, err
		}
		return size > 0, nil
	case pubsubModeOldestUnackedMessageAge:
		_, err := s.getMetrics(ctx, pubSubStackDriverOldestUnackedMessageAgeMetricName)
		if err != nil {
			gcpPubSubLog.Error(err, "error getting Active Status")
			return false, err
		}
		return true, nil
	default:
		return false, errors.New("unknown mode")
	}
}

func (s *pubsubScaler) Close(context.Context) error {
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
func (s *pubsubScaler) GetMetricSpecForScaling(context.Context) []v2beta2.MetricSpec {
	// Construct the target value as a quantity
	targetValueQty := resource.NewQuantity(int64(s.metadata.value), resource.DecimalSI)

	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, kedautil.NormalizeString(fmt.Sprintf("%s-%s", "gcp", s.metadata.subscriptionName))),
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: targetValueQty,
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
	var value int64
	var err error

	switch s.metadata.mode {
	case pubsubModeSubscriptionSize:
		value, err = s.getMetrics(ctx, pubSubStackDriverSubscriptionSizeMetricName)
		if err != nil {
			gcpPubSubLog.Error(err, "error getting subscription size")
			return []external_metrics.ExternalMetricValue{}, err
		}
	case pubsubModeOldestUnackedMessageAge:
		value, err = s.getMetrics(ctx, pubSubStackDriverOldestUnackedMessageAgeMetricName)
		if err != nil {
			gcpPubSubLog.Error(err, "error getting oldest unacked message age")
			return []external_metrics.ExternalMetricValue{}, err
		}
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(value, resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

func (s *pubsubScaler) setStackdriverClient(ctx context.Context) error {
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
func (s *pubsubScaler) getMetrics(ctx context.Context, metricType string) (int64, error) {
	if s.client == nil {
		err := s.setStackdriverClient(ctx)
		if err != nil {
			return -1, err
		}
	}

	filter := `metric.type="` + metricType + `" AND resource.labels.subscription_id="` + s.metadata.subscriptionName + `"`

	return s.client.GetMetrics(ctx, filter)
}

func getGcpAuthorization(config *ScalerConfig, resolvedEnv map[string]string) (*gcpAuthorizationMetadata, error) {
	metadata := config.TriggerMetadata
	authParams := config.AuthParams
	meta := gcpAuthorizationMetadata{}
	if metadata["identityOwner"] == "operator" {
		meta.podIdentityOwner = false
	} else if metadata["identityOwner"] == "" || metadata["identityOwner"] == "pod" {
		meta.podIdentityOwner = true
		switch {
		case config.PodIdentity == kedav1alpha1.PodIdentityProviderGCP:
			// do nothing, rely on underneath metadata google
			meta.podIdentityProviderEnabled = true
		case authParams["GoogleApplicationCredentials"] != "":
			meta.GoogleApplicationCredentials = authParams["GoogleApplicationCredentials"]
		default:
			if metadata["credentialsFromEnv"] != "" {
				meta.GoogleApplicationCredentials = resolvedEnv[metadata["credentialsFromEnv"]]
			} else {
				return nil, fmt.Errorf("GoogleApplicationCredentials not found")
			}
		}
	}
	return &meta, nil
}
