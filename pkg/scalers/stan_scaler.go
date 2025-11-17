package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type monitorChannelInfo struct {
	Name         string                  `json:"name"`
	MsgCount     int64                   `json:"msgs"`
	LastSequence int64                   `json:"last_seq"`
	Subscriber   []monitorSubscriberInfo `json:"subscriptions"`
}

type monitorSubscriberInfo struct {
	ClientID     string `json:"client_id"`
	QueueName    string `json:"queue_name"`
	Inbox        string `json:"inbox"`
	AckInbox     string `json:"ack_inbox"`
	IsDurable    bool   `json:"is_durable"`
	IsOffline    bool   `json:"is_offline"`
	MaxInflight  int    `json:"max_inflight"`
	LastSent     int64  `json:"last_sent"`
	PendingCount int    `json:"pending_count"`
	IsStalled    bool   `json:"is_stalled"`
}

type stanScaler struct {
	channelInfo *monitorChannelInfo
	metricType  v2.MetricTargetType
	metadata    stanMetadata
	httpClient  *http.Client
	logger      logr.Logger
}

type stanMetadata struct {
	NatsServerMonitoringEndpoint string `keda:"name=natsServerMonitoringEndpoint, order=triggerMetadata;authParams"`
	QueueGroup                   string `keda:"name=queueGroup,                  order=triggerMetadata"`
	DurableName                  string `keda:"name=durableName,                 order=triggerMetadata"`
	Subject                      string `keda:"name=subject,                     order=triggerMetadata"`
	LagThreshold                 int64  `keda:"name=lagThreshold,                order=triggerMetadata, default=10"`
	ActivationLagThreshold       int64  `keda:"name=activationLagThreshold,      order=triggerMetadata, default=0"`
	UseHTTPS                     bool   `keda:"name=useHttps,                    order=triggerMetadata, optional"`

	// TLS
	TLS  string `keda:"name=tls,         order=authParams, enum=enable;disable, optional"`
	Cert string `keda:"name=cert,        order=authParams, optional"`
	Key  string `keda:"name=key,         order=authParams, optional"`
	CA   string `keda:"name=ca,          order=authParams, optional"`

	// Internal computed fields
	monitoringEndpoint   string
	stanChannelsEndpoint string
	enableTLS            bool
	triggerIndex         int
}

const (
	stanMetricType             = "External"
	natsStreamingHTTPProtocol  = "http"
	natsStreamingHTTPSProtocol = "https"
)

func (s *stanMetadata) Validate() error {
	if s.LagThreshold <= 0 {
		return fmt.Errorf("lagThreshold must be a positive number")
	}
	if s.ActivationLagThreshold < 0 {
		return fmt.Errorf("activationLagThreshold must be a non-negative number")
	}
	if s.TLS == stringEnable && ((s.Cert == "") != (s.Key == "")) {
		return fmt.Errorf("can't set only one of cert or key when using TLS")
	}
	return nil
}

// NewStanScaler creates a new stanScaler
func NewStanScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	stanMetadata, err := parseStanMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing stan metadata: %w", err)
	}
	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, false)
	if stanMetadata.enableTLS {
		tlsConfig, err := kedautil.NewTLSConfig(stanMetadata.Cert, stanMetadata.Key, stanMetadata.CA, false)
		if err != nil {
			return nil, err
		}
		httpClient.Transport = kedautil.CreateHTTPTransportWithTLSConfig(tlsConfig)
	}
	return &stanScaler{
		channelInfo: &monitorChannelInfo{},
		metricType:  metricType,
		metadata:    stanMetadata,
		httpClient:  httpClient,
		logger:      InitializeLogger(config, "stan_scaler"),
	}, nil
}

func parseStanMetadata(config *scalersconfig.ScalerConfig) (stanMetadata, error) {
	meta := stanMetadata{triggerIndex: config.TriggerIndex}
	if err := config.TypedConfig(&meta); err != nil {
		return meta, fmt.Errorf("error parsing stan metadata: %w", err)
	}

	// Handle TLS enable flag
	meta.enableTLS = meta.TLS == stringEnable

	// Build endpoints
	meta.stanChannelsEndpoint = getSTANChannelsEndpoint(meta.UseHTTPS, meta.NatsServerMonitoringEndpoint)
	meta.monitoringEndpoint = getMonitoringEndpoint(meta.stanChannelsEndpoint, meta.Subject)

	return meta, nil
}

func getSTANChannelsEndpoint(useHTTPS bool, natsServerEndpoint string) string {
	protocol := natsStreamingHTTPProtocol
	if useHTTPS {
		protocol = natsStreamingHTTPSProtocol
	}
	return fmt.Sprintf("%s://%s/streaming/channelsz", protocol, natsServerEndpoint)
}

func getMonitoringEndpoint(stanChannelsEndpoint string, subject string) string {
	return fmt.Sprintf("%s?channel=%s&subs=1", stanChannelsEndpoint, subject)
}

func (s *stanScaler) getMaxMsgLag() int64 {
	maxValue := int64(0)
	combinedQueueName := s.metadata.DurableName + ":" + s.metadata.QueueGroup

	for _, subs := range s.channelInfo.Subscriber {
		if subs.LastSent > maxValue && subs.QueueName == combinedQueueName {
			maxValue = subs.LastSent
		}
	}

	return s.channelInfo.LastSequence - maxValue
}

func (s *stanScaler) hasPendingMessage() bool {
	subscriberFound := false
	combinedQueueName := s.metadata.DurableName + ":" + s.metadata.QueueGroup

	for _, subs := range s.channelInfo.Subscriber {
		if subs.QueueName == combinedQueueName {
			subscriberFound = true

			if subs.PendingCount > 0 {
				return true
			}

			break
		}
	}

	if !subscriberFound {
		s.logger.Info("The STAN subscription was not found.", "combinedQueueName", combinedQueueName)
	}

	return false
}

func (s *stanScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	metricName := kedautil.NormalizeString(fmt.Sprintf("stan-%s", s.metadata.Subject))
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, metricName),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.LagThreshold),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: stanMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *stanScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.metadata.monitoringEndpoint, nil)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, err
	}
	resp, err := s.httpClient.Do(req)

	if err != nil {
		s.logger.Error(err, "Unable to access the nats streaming broker monitoring endpoint", "monitoringEndpoint", s.metadata.monitoringEndpoint)
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	if resp.StatusCode == 404 {
		req, err := http.NewRequestWithContext(ctx, "GET", s.metadata.stanChannelsEndpoint, nil)
		if err != nil {
			return []external_metrics.ExternalMetricValue{}, false, err
		}
		baseResp, err := s.httpClient.Do(req)
		if err != nil {
			return []external_metrics.ExternalMetricValue{}, false, err
		}
		defer baseResp.Body.Close()
		if baseResp.StatusCode == 404 {
			s.logger.Info("Streaming broker endpoint returned 404. Please ensure it has been created", "url", s.metadata.monitoringEndpoint, "channelName", s.metadata.Subject)
		} else {
			s.logger.Info("Unable to connect to STAN. Please ensure you have configured the ScaledObject with the correct endpoint.", "baseResp.StatusCode", baseResp.StatusCode, "monitoringEndpoint", s.metadata.monitoringEndpoint)
		}

		return []external_metrics.ExternalMetricValue{}, false, err
	}

	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&s.channelInfo); err != nil {
		s.logger.Error(err, "Unable to decode channel info as %v", err)
		return []external_metrics.ExternalMetricValue{}, false, err
	}
	totalLag := s.getMaxMsgLag()
	s.logger.V(1).Info("Stan scaler: Providing metrics based on totalLag, threshold", "totalLag", totalLag, "lagThreshold", s.metadata.LagThreshold)
	s.logger.Info("The Stan scaler (NATS Streaming) is DEPRECATED and will be removed in v2.19 - Use scaler 'nats-jetstream' instead")

	metric := GenerateMetricInMili(metricName, float64(totalLag))

	return []external_metrics.ExternalMetricValue{metric}, s.hasPendingMessage() || totalLag > s.metadata.ActivationLagThreshold, nil
}

// Nothing to close here.
func (s *stanScaler) Close(context.Context) error {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}
