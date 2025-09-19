package scalers

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

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

	pubSubDefaultModeSubscriptionSize = "SubscriptionSize"
	pubSubDefaultValue                = 10
)

var regexpCompositeSubscriptionIDPrefix = regexp.MustCompile(compositeSubscriptionIDPrefix)

type pubsubScaler struct {
	client     *gcp.StackDriverClient
	metricType v2.MetricTargetType
	metadata   *pubsubMetadata
	logger     logr.Logger
}

type pubsubMetadata struct {
	SubscriptionSize int           `keda:"name=subscriptionSize, order=triggerMetadata, optional, deprecatedAnnounce=The 'subscriptionSize' setting is DEPRECATED and will be removed in v2.20 - Use 'mode' and 'value' instead"`
	Mode             string        `keda:"name=mode, order=triggerMetadata, default=SubscriptionSize"`
	Value            float64       `keda:"name=value, order=triggerMetadata, default=10, deprecatedAnnounce=This scaler is deprecated. More info -> 'https://keda.sh/blog/2025-09-15-gcp-deprecations'"`
	ActivationValue  float64       `keda:"name=activationValue, order=triggerMetadata, default=0"`
	Aggregation      string        `keda:"name=aggregation, order=triggerMetadata, optional"`
	TimeHorizon      time.Duration `keda:"name=timeHorizon, order=triggerMetadata, optional"`
	ValueIfNull      *float64      `keda:"name=valueIfNull, order=triggerMetadata, optional"`
	SubscriptionName string        `keda:"name=subscriptionName, order=triggerMetadata;resolvedEnv, optional"`
	TopicName        string        `keda:"name=topicName, order=triggerMetadata;resolvedEnv, optional"`
	// a resource is one of subscription or topic
	resourceType     string
	resourceName     string
	gcpAuthorization *gcp.AuthorizationMetadata
	triggerIndex     int
}

// NewPubSubScaler creates a new pubsubScaler
func NewPubSubScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, err := parsePubSubMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing PubSub metadata: %w", err)
	}

	return &pubsubScaler{
		metricType: metricType,
		metadata:   meta,
	}, nil
}

func parsePubSubMetadata(config *scalersconfig.ScalerConfig) (*pubsubMetadata, error) {
	meta := &pubsubMetadata{}

	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing gcp pubsub metadata: %w", err)
	}

	if meta.SubscriptionSize != 0 {
		meta.Mode = pubSubDefaultModeSubscriptionSize
		meta.Value = float64(meta.SubscriptionSize)
	}

	if meta.SubscriptionName != "" {
		meta.resourceName = meta.SubscriptionName
		meta.resourceType = resourceTypePubSubSubscription
	} else {
		meta.resourceName = meta.TopicName
		meta.resourceType = resourceTypePubSubTopic
	}

	auth, err := gcp.GetGCPAuthorization(config)
	if err != nil {
		return nil, err
	}
	meta.gcpAuthorization = auth
	meta.triggerIndex = config.TriggerIndex

	return meta, nil
}

func (s *pubsubScaler) Close(context.Context) error {
	if s.client != nil {
		err := s.client.Close()
		s.client = nil
		if err != nil {
			s.logger.Error(err, "error closing StackDriver client")
		}
	}
	return nil
}

func (meta *pubsubMetadata) Validate() error {
	if meta.SubscriptionSize != 0 {
		if meta.TopicName != "" {
			return fmt.Errorf("you cannot use subscriptionSize field together with topicName field. Use subscriptionName field instead")
		}
	}

	hasSub := meta.SubscriptionName != ""
	hasTopic := meta.TopicName != ""
	if (!hasSub && !hasTopic) || (hasSub && hasTopic) {
		return fmt.Errorf("exactly one of subscription or topic name must be given")
	}

	return nil
}

// GetMetricSpecForScaling returns the metric spec for the HPA
func (s *pubsubScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("gcp-ps-%s", s.metadata.resourceName))),
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

// GetMetricsAndActivity connects to Stack Driver and finds the size of the pub sub subscription
func (s *pubsubScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	mode := s.metadata.Mode

	// SubscriptionSize is actually NumUndeliveredMessages in GCP PubSub.
	// Considering backward compatibility, fallback "SubscriptionSize" to "NumUndeliveredMessages"
	if mode == pubSubDefaultModeSubscriptionSize {
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

	return []external_metrics.ExternalMetricValue{metric}, value > s.metadata.ActivationValue, nil
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
		projectID, s.metadata.resourceType, metricType, resourceID, s.metadata.Aggregation, s.metadata.TimeHorizon,
	)
	if err != nil {
		return -1, err
	}

	// Pubsub metrics are collected every 60 seconds so no need to aggregate them.
	// See: https://cloud.google.com/monitoring/api/metrics_gcp#gcp-pubsub
	return s.client.QueryMetrics(ctx, projectID, query, s.metadata.ValueIfNull)
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
