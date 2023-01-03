package scalers

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	compositeSubscriptionIDPrefix                      = "projects/[a-z][a-zA-Z0-9-]*[a-zA-Z0-9]/subscriptions/[a-zA-Z][a-zA-Z0-9-_~%\\+\\.]*"
	defaultTargetSubscriptionSize                      = 5
	defaultTargetOldestUnackedMessageAge               = 10
	pubSubStackDriverSubscriptionSizeMetricName        = "pubsub.googleapis.com/subscription/num_undelivered_messages"
	pubSubStackDriverOldestUnackedMessageAgeMetricName = "pubsub.googleapis.com/subscription/oldest_unacked_message_age"

	pubsubModeSubscriptionSize        = "SubscriptionSize"
	pubsubModeOldestUnackedMessageAge = "OldestUnackedMessageAge"
)

var regexpCompositeSubscriptionIDPrefix = regexp.MustCompile(compositeSubscriptionIDPrefix)

type pubsubScaler struct {
	client     *StackDriverClient
	metricType v2.MetricTargetType
	metadata   *pubsubMetadata
	logger     logr.Logger
}

type pubsubMetadata struct {
	mode            string
	value           float64
	activationValue float64

	subscriptionName string
	gcpAuthorization *gcpAuthorizationMetadata
	scalerIndex      int
}

// NewPubSubScaler creates a new pubsubScaler
func NewPubSubScaler(config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "gcp_pub_sub_scaler")

	meta, err := parsePubSubMetadata(config, logger)
	if err != nil {
		return nil, fmt.Errorf("error parsing PubSub metadata: %w", err)
	}

	return &pubsubScaler{
		metricType: metricType,
		metadata:   meta,
		logger:     logger,
	}, nil
}

func parsePubSubMetadata(config *ScalerConfig, logger logr.Logger) (*pubsubMetadata, error) {
	meta := pubsubMetadata{}
	meta.mode = pubsubModeSubscriptionSize

	mode, modePresent := config.TriggerMetadata["mode"]
	value, valuePresent := config.TriggerMetadata["value"]

	if subSize, subSizePresent := config.TriggerMetadata["subscriptionSize"]; subSizePresent {
		if modePresent || valuePresent {
			return nil, errors.New("you can use either mode and value fields or subscriptionSize field")
		}
		logger.Info("subscriptionSize field is deprecated. Use mode and value fields instead")
		meta.mode = pubsubModeSubscriptionSize
		subSizeValue, err := strconv.ParseFloat(subSize, 64)
		if err != nil {
			return nil, fmt.Errorf("value parsing error %w", err)
		}
		meta.value = subSizeValue
	} else {
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

		if valuePresent {
			triggerValue, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return nil, fmt.Errorf("value parsing error %w", err)
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

	meta.activationValue = 0
	if val, ok := config.TriggerMetadata["activationValue"]; ok {
		activationValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("activationValue parsing error %w", err)
		}
		meta.activationValue = activationValue
	}

	auth, err := getGcpAuthorization(config, config.ResolvedEnv)
	if err != nil {
		return nil, err
	}
	meta.gcpAuthorization = auth
	meta.scalerIndex = config.ScalerIndex
	return &meta, nil
}

func (s *pubsubScaler) Close(context.Context) error {
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
func (s *pubsubScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, kedautil.NormalizeString(fmt.Sprintf("gcp-ps-%s", s.metadata.subscriptionName))),
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

// GetMetricsAndActivity connects to Stack Driver and finds the size of the pub sub subscription
func (s *pubsubScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	var value float64
	var err error

	switch s.metadata.mode {
	case pubsubModeSubscriptionSize:
		value, err = s.getMetrics(ctx, pubSubStackDriverSubscriptionSizeMetricName)
		if err != nil {
			s.logger.Error(err, "error getting subscription size")
			return []external_metrics.ExternalMetricValue{}, false, err
		}
	case pubsubModeOldestUnackedMessageAge:
		value, err = s.getMetrics(ctx, pubSubStackDriverOldestUnackedMessageAgeMetricName)
		if err != nil {
			s.logger.Error(err, "error getting oldest unacked message age")
			return []external_metrics.ExternalMetricValue{}, false, err
		}
	}

	metric := GenerateMetricInMili(metricName, value)

	return []external_metrics.ExternalMetricValue{metric}, value > s.metadata.activationValue, nil
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
func (s *pubsubScaler) getMetrics(ctx context.Context, metricType string) (float64, error) {
	if s.client == nil {
		err := s.setStackdriverClient(ctx)
		if err != nil {
			return -1, err
		}
	}
	subscriptionID, projectID := getSubscriptionData(s)
	filter := `metric.type="` + metricType + `" AND resource.labels.subscription_id="` + subscriptionID + `"`

	// Pubsub metrics are collected every 60 seconds so no need to aggregate them.
	// See: https://cloud.google.com/monitoring/api/metrics_gcp#gcp-pubsub
	return s.client.GetMetrics(ctx, filter, projectID, nil)
}

func getSubscriptionData(s *pubsubScaler) (string, string) {
	var subscriptionID string
	var projectID string

	if regexpCompositeSubscriptionIDPrefix.MatchString(s.metadata.subscriptionName) {
		subscriptionID = strings.Split(s.metadata.subscriptionName, "/")[3]
		projectID = strings.Split(s.metadata.subscriptionName, "/")[1]
	} else {
		subscriptionID = s.metadata.subscriptionName
	}
	return subscriptionID, projectID
}
