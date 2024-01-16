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

	"github.com/kedacore/keda/v2/pkg/scalers/gcp"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	compositeSubscriptionIDPrefix = "projects/[a-z][a-zA-Z0-9-]*[a-zA-Z0-9]/(subscriptions|topics)/[a-zA-Z][a-zA-Z0-9-_~%\\+\\.]*"
	prefixPubSubResource          = "pubsub.googleapis.com/"

	resourceTypePubSubSubscription = "subscription"
	resourceTypePubSubTopic        = "topic"

	pubSubModeSubscriptionSize = "SubscriptionSize"
	pubSubDefaultValue         = 10
)

var regexpCompositeSubscriptionIDPrefix = regexp.MustCompile(compositeSubscriptionIDPrefix)

type pubsubScaler struct {
	client     *gcp.StackDriverClient
	metricType v2.MetricTargetType
	metadata   *pubsubMetadata
	logger     logr.Logger
}

type pubsubMetadata struct {
	mode            string
	value           float64
	activationValue float64

	// a resource is one of subscription or topic
	resourceType     string
	resourceName     string
	gcpAuthorization *gcp.AuthorizationMetadata
	triggerIndex     int
	aggregation      string
}

// NewPubSubScaler creates a new pubsubScaler
func NewPubSubScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
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

func parsePubSubMetadata(config *scalersconfig.ScalerConfig, logger logr.Logger) (*pubsubMetadata, error) {
	meta := pubsubMetadata{mode: pubSubModeSubscriptionSize, value: pubSubDefaultValue}

	mode, modePresent := config.TriggerMetadata["mode"]
	value, valuePresent := config.TriggerMetadata["value"]

	if subSize, subSizePresent := config.TriggerMetadata["subscriptionSize"]; subSizePresent {
		if modePresent || valuePresent {
			return nil, errors.New("you can use either mode and value fields or subscriptionSize field")
		}
		if _, topicPresent := config.TriggerMetadata["topicName"]; topicPresent {
			return nil, errors.New("you cannot use subscriptionSize field together with topicName field. Use subscriptionName field instead")
		}
		logger.Info("subscriptionSize field is deprecated. Use mode and value fields instead")
		subSizeValue, err := strconv.ParseFloat(subSize, 64)
		if err != nil {
			return nil, fmt.Errorf("value parsing error %w", err)
		}
		meta.value = subSizeValue
	} else {
		if modePresent {
			meta.mode = mode
		}

		if valuePresent {
			triggerValue, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return nil, fmt.Errorf("value parsing error %w", err)
			}
			meta.value = triggerValue
		}
	}

	meta.aggregation = config.TriggerMetadata["aggregation"]

	sub, subPresent := config.TriggerMetadata["subscriptionName"]
	topic, topicPresent := config.TriggerMetadata["topicName"]
	if (!subPresent && !topicPresent) || (subPresent && topicPresent) {
		return nil, fmt.Errorf("exactly one of subscription or topic name must be given")
	}

	if subPresent {
		if sub == "" {
			return nil, fmt.Errorf("no subscription name given")
		}

		meta.resourceName = sub
		meta.resourceType = resourceTypePubSubSubscription
	} else {
		if topic == "" {
			return nil, fmt.Errorf("no topic name given")
		}

		meta.resourceName = topic
		meta.resourceType = resourceTypePubSubTopic
	}

	meta.activationValue = 0
	if val, ok := config.TriggerMetadata["activationValue"]; ok {
		activationValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("activationValue parsing error %w", err)
		}
		meta.activationValue = activationValue
	}

	auth, err := gcp.GetGCPAuthorization(config)
	if err != nil {
		return nil, err
	}
	meta.gcpAuthorization = auth
	meta.triggerIndex = config.TriggerIndex
	return &meta, nil
}

func (s *pubsubScaler) Close(context.Context) error {
	if s.client != nil {
		err := s.client.MetricsClient.Close()
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
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("gcp-ps-%s", s.metadata.resourceName))),
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
	mode := s.metadata.mode

	// SubscriptionSize is actually NumUndeliveredMessages in GCP PubSub.
	// Considering backward compatibility, fallback "SubscriptionSize" to "NumUndeliveredMessages"
	if mode == pubSubModeSubscriptionSize {
		mode = "NumUndeliveredMessages"
	}

	prefix := prefixPubSubResource + s.metadata.resourceType + "/"
	metricType := prefix + snakeCase(mode)
	value, err := s.getMetrics(ctx, metricType)
	if err != nil {
		s.logger.Error(err, "error getting metric", "metricType", metricType)
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, value)

	return []external_metrics.ExternalMetricValue{metric}, value >= s.metadata.activationValue, nil
}

func (s *pubsubScaler) setStackdriverClient(ctx context.Context) error {
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
func (s *pubsubScaler) getMetrics(ctx context.Context, metricType string) (float64, error) {
	if s.client == nil {
		if err := s.setStackdriverClient(ctx); err != nil {
			return -1, err
		}
	}
	resourceID, projectID := getResourceData(s)
	query, err := s.client.BuildMQLQuery(
		projectID, s.metadata.resourceType, metricType, resourceID, s.metadata.aggregation,
	)
	if err != nil {
		return -1, err
	}

	// Pubsub metrics are collected every 60 seconds so no need to aggregate them.
	// See: https://cloud.google.com/monitoring/api/metrics_gcp#gcp-pubsub
	return s.client.QueryMetrics(ctx, projectID, query)
}

func getResourceData(s *pubsubScaler) (string, string) {
	var resourceID string
	var projectID string

	if regexpCompositeSubscriptionIDPrefix.MatchString(s.metadata.resourceName) {
		resourceID = strings.Split(s.metadata.resourceName, "/")[3]
		projectID = strings.Split(s.metadata.resourceName, "/")[1]
	} else {
		resourceID = s.metadata.resourceName
	}
	return resourceID, projectID
}

var (
	regexpFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	regexpAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

func snakeCase(camelCase string) string {
	snake := regexpFirstCap.ReplaceAllString(camelCase, "${1}_${2}")
	snake = regexpAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}
