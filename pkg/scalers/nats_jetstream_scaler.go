package scalers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	jetStreamMetricType          = "External"
	defaultJetStreamLagThreshold = 10
	natsHTTPProtocol             = "http"
	natsHTTPSProtocol            = "https"
)

type natsJetStreamScaler struct {
	metricType v2.MetricTargetType
	stream     *streamDetail
	metadata   natsJetStreamMetadata
	httpClient *http.Client
	logger     logr.Logger
}

type natsJetStreamMetadata struct {
	monitoringEndpoint     string
	account                string
	stream                 string
	consumer               string
	lagThreshold           int64
	activationLagThreshold int64
	scalerIndex            int
}

type jetStreamEndpointResponse struct {
	Accounts []accountDetail `json:"account_details"`
}

type accountDetail struct {
	Name    string          `json:"name"`
	Streams []*streamDetail `json:"stream_detail"`
}

type streamDetail struct {
	Name      string           `json:"name"`
	Config    streamConfig     `json:"config"`
	State     streamState      `json:"state"`
	Consumers []consumerDetail `json:"consumer_detail"`
}

type streamConfig struct {
	Subjects []string `json:"subjects"`
}

type streamState struct {
	MsgCount     int64 `json:"messages"`
	LastSequence int64 `json:"last_seq"`
}

type consumerDetail struct {
	StreamName     string                 `json:"stream_name"`
	Name           string                 `json:"name"`
	NumAckPending  int                    `json:"num_ack_pending"`
	NumRedelivered int                    `json:"num_redelivered"`
	NumWaiting     int                    `json:"num_waiting"`
	NumPending     int                    `json:"num_pending"`
	Config         consumerConfig         `json:"config"`
	DeliveryStatus consumerDeliveryStatus `json:"delivery"`
}

type consumerConfig struct {
	DurableName string `json:"durable_name"`
}

type consumerDeliveryStatus struct {
	ConsumerSequence int64 `json:"customer_seq"`
	StreamSequence   int64 `json:"stream_seq"`
}

func NewNATSJetStreamScaler(config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %s", err)
	}

	jsMetadata, err := parseNATSJetStreamMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing NATS JetStream metadata: %s", err)
	}

	return &natsJetStreamScaler{
		metricType: metricType,
		stream:     &streamDetail{},
		metadata:   jsMetadata,
		httpClient: kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, false),
		logger:     InitializeLogger(config, "nats_jetstream_scaler"),
	}, nil
}

func parseNATSJetStreamMetadata(config *ScalerConfig) (natsJetStreamMetadata, error) {
	meta := natsJetStreamMetadata{}

	if config.TriggerMetadata["account"] == "" {
		return meta, errors.New("no account name given")
	}
	meta.account = config.TriggerMetadata["account"]

	if config.TriggerMetadata["stream"] == "" {
		return meta, errors.New("no stream name given")
	}
	meta.stream = config.TriggerMetadata["stream"]

	if config.TriggerMetadata["consumer"] == "" {
		return meta, errors.New("no consumer name given")
	}
	meta.consumer = config.TriggerMetadata["consumer"]

	meta.lagThreshold = defaultJetStreamLagThreshold

	if val, ok := config.TriggerMetadata[lagThresholdMetricName]; ok {
		t, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return meta, fmt.Errorf("error parsing %s: %s", lagThresholdMetricName, err)
		}
		meta.lagThreshold = t
	}

	meta.activationLagThreshold = 0
	if val, ok := config.TriggerMetadata["activationLagThreshold"]; ok {
		activationTargetQueryValue, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return meta, fmt.Errorf("activationLagThreshold parsing error %s", err.Error())
		}
		meta.activationLagThreshold = activationTargetQueryValue
	}

	meta.scalerIndex = config.ScalerIndex

	natsServerEndpoint, err := GetFromAuthOrMeta(config, "natsServerMonitoringEndpoint")
	if err != nil {
		return meta, err
	}
	useHTTPS := false
	if val, ok := config.TriggerMetadata["useHttps"]; ok {
		useHTTPS, err = strconv.ParseBool(val)
		if err != nil {
			return meta, fmt.Errorf("useHTTPS parsing error %s", err.Error())
		}
	}
	meta.monitoringEndpoint = getNATSJetStreamEndpoint(useHTTPS, natsServerEndpoint, meta.account)

	return meta, nil
}

func getNATSJetStreamEndpoint(useHTTPS bool, natsServerEndpoint string, account string) string {
	protocol := natsHTTPProtocol
	if useHTTPS {
		protocol = natsHTTPSProtocol
	}

	return fmt.Sprintf("%s://%s/jsz?acc=%s&consumers=true&config=true", protocol, natsServerEndpoint, account)
}

func (s *natsJetStreamScaler) IsActive(ctx context.Context) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.metadata.monitoringEndpoint, nil)
	if err != nil {
		return false, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Error(err, "unable to access NATS JetStream monitoring endpoint", "natsServerMonitoringEndpoint", s.metadata.monitoringEndpoint)
		return false, err
	}

	defer resp.Body.Close()
	var jsAccountResp jetStreamEndpointResponse
	if err = json.NewDecoder(resp.Body).Decode(&jsAccountResp); err != nil {
		s.logger.Error(err, "unable to decode JetStream account response")
		return false, err
	}

	// Find and assign the stream that we are looking for.
	for _, account := range jsAccountResp.Accounts {
		if account.Name == s.metadata.account {
			for _, stream := range account.Streams {
				if stream.Name == s.metadata.stream {
					s.stream = stream
				}
			}
		}
	}
	return s.getMaxMsgLag() > s.metadata.activationLagThreshold, nil
}

func (s *natsJetStreamScaler) getMaxMsgLag() int64 {
	consumerName := s.metadata.consumer

	for _, consumer := range s.stream.Consumers {
		if consumer.Name == consumerName {
			return int64(consumer.NumPending + consumer.NumAckPending)
		}
	}
	return s.stream.State.LastSequence
}

func (s *natsJetStreamScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	metricName := kedautil.NormalizeString(fmt.Sprintf("nats-jetstream-%s", s.metadata.stream))
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, metricName),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.lagThreshold),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: jetStreamMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

func (s *natsJetStreamScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.metadata.monitoringEndpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Error(err, "unable to access NATS JetStream monitoring endpoint", "natsServerMonitoringEndpoint", s.metadata.monitoringEndpoint)
		return []external_metrics.ExternalMetricValue{}, err
	}

	defer resp.Body.Close()
	var jsAccountResp jetStreamEndpointResponse
	if err = json.NewDecoder(resp.Body).Decode(&jsAccountResp); err != nil {
		s.logger.Error(err, "unable to decode JetStream account details")
		return []external_metrics.ExternalMetricValue{}, err
	}

	// Find and assign the stream that we are looking for.
	for _, account := range jsAccountResp.Accounts {
		if account.Name == s.metadata.account {
			for _, stream := range account.Streams {
				if stream.Name == s.metadata.stream {
					s.stream = stream
				}
			}
		}
	}

	totalLag := s.getMaxMsgLag()
	s.logger.V(1).Info("NATS JetStream Scaler: Providing metrics based on totalLag, threshold", "totalLag", totalLag, "lagThreshold", s.metadata.lagThreshold)

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(totalLag, resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

func (s *natsJetStreamScaler) Close(context.Context) error {
	return nil
}
