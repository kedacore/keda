package scalers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
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
	monitoringEndpoint     string
	stanChannelsEndpoint   string
	queueGroup             string
	durableName            string
	subject                string
	lagThreshold           int64
	activationLagThreshold int64
	triggerIndex           int

	// TLS
	enableTLS bool
	cert      string
	key       string
	ca        string
}

const (
	stanMetricType             = "External"
	defaultStanLagThreshold    = 10
	natsStreamingHTTPProtocol  = "http"
	natsStreamingHTTPSProtocol = "https"
	stanTLSEnable              = "enable"
	stanTLSDisable             = "disable"
)

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
		config, err := kedautil.NewTLSConfig(stanMetadata.cert, stanMetadata.key, stanMetadata.ca, false)
		if err != nil {
			return nil, err
		}
		httpClient.Transport = kedautil.CreateHTTPTransportWithTLSConfig(config)
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
	meta := stanMetadata{}

	if config.TriggerMetadata["queueGroup"] == "" {
		return meta, errors.New("no queue group given")
	}
	meta.queueGroup = config.TriggerMetadata["queueGroup"]

	if config.TriggerMetadata["durableName"] == "" {
		return meta, errors.New("no durable name group given")
	}
	meta.durableName = config.TriggerMetadata["durableName"]

	if config.TriggerMetadata["subject"] == "" {
		return meta, errors.New("no subject given")
	}
	meta.subject = config.TriggerMetadata["subject"]

	meta.lagThreshold = defaultStanLagThreshold

	if val, ok := config.TriggerMetadata[lagThresholdMetricName]; ok {
		t, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return meta, fmt.Errorf("error parsing %s: %w", lagThresholdMetricName, err)
		}
		meta.lagThreshold = t
	}

	meta.activationLagThreshold = 0
	if val, ok := config.TriggerMetadata["activationLagThreshold"]; ok {
		activationTargetQueryValue, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return meta, fmt.Errorf("activationLagThreshold parsing error %w", err)
		}
		meta.activationLagThreshold = activationTargetQueryValue
	}

	meta.triggerIndex = config.TriggerIndex

	var err error

	meta.enableTLS = false // Default value for enableTLS
	useHTTPS := false
	if val, ok := config.TriggerMetadata["useHttps"]; ok {
		useHTTPS, err = strconv.ParseBool(val)
		if err != nil {
			return meta, fmt.Errorf("useHTTPS parsing error %w", err)
		}
		if val, ok := config.AuthParams["tls"]; ok {
			val = strings.TrimSpace(val)
			if val == stanTLSEnable {
				certGiven := config.AuthParams["cert"] != ""
				keyGiven := config.AuthParams["key"] != ""
				if certGiven && !keyGiven {
					return meta, errors.New("no key given")
				}
				if keyGiven && !certGiven {
					return meta, errors.New("no cert given")
				}
				meta.cert = config.AuthParams["cert"]
				meta.key = config.AuthParams["key"]
				meta.ca = config.AuthParams["ca"]
				meta.enableTLS = true
			} else if val != stanTLSDisable {
				return meta, fmt.Errorf("err incorrect value for TLS given: %s", val)
			}
		}
	}
	natsServerEndpoint, err := GetFromAuthOrMeta(config, "natsServerMonitoringEndpoint")
	if err != nil {
		return meta, err
	}
	meta.stanChannelsEndpoint = getSTANChannelsEndpoint(useHTTPS, natsServerEndpoint)
	meta.monitoringEndpoint = getMonitoringEndpoint(meta.stanChannelsEndpoint, meta.subject)

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
	combinedQueueName := s.metadata.durableName + ":" + s.metadata.queueGroup

	for _, subs := range s.channelInfo.Subscriber {
		if subs.LastSent > maxValue && subs.QueueName == combinedQueueName {
			maxValue = subs.LastSent
		}
	}

	return s.channelInfo.LastSequence - maxValue
}

func (s *stanScaler) hasPendingMessage() bool {
	subscriberFound := false
	combinedQueueName := s.metadata.durableName + ":" + s.metadata.queueGroup

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
	metricName := kedautil.NormalizeString(fmt.Sprintf("stan-%s", s.metadata.subject))
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, metricName),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.lagThreshold),
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
			s.logger.Info("Streaming broker endpoint returned 404. Please ensure it has been created", "url", s.metadata.monitoringEndpoint, "channelName", s.metadata.subject)
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
	s.logger.V(1).Info("Stan scaler: Providing metrics based on totalLag, threshold", "totalLag", totalLag, "lagThreshold", s.metadata.lagThreshold)
	s.logger.Info("The Stan scaler (NATS Streaming) is DEPRECATED and will be removed in v2.19 - Use scaler 'nats-jetstream' instead")

	metric := GenerateMetricInMili(metricName, float64(totalLag))

	return []external_metrics.ExternalMetricValue{metric}, s.hasPendingMessage() || totalLag > s.metadata.activationLagThreshold, nil
}

// Nothing to close here.
func (s *stanScaler) Close(context.Context) error {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}
