package scalers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	jetStreamMetricType             = "External"
	defaultJetStreamLagThreshold    = 10
	natsHTTPProtocol                = "http"
	natsHTTPSProtocol               = "https"
	jetStreamLagThresholdMetricName = "lagThreshold"
)

type natsJetStreamScaler struct {
	metricType v2.MetricTargetType
	stream     *streamDetail
	metadata   natsJetStreamMetadata
	httpClient *http.Client
	logger     logr.Logger
}

type natsJetStreamMetadata struct {
	account                string
	stream                 string
	consumer               string
	consumerLeader         string
	monitoringURL          string
	monitoringLeaderURL    string
	lagThreshold           int64
	activationLagThreshold int64
	clusterSize            int
	triggerIndex           int
}

type jetStreamEndpointResponse struct {
	Accounts    []accountDetail `json:"account_details"`
	MetaCluster metaCluster     `json:"meta_cluster"`
}

type jetStreamServerEndpointResponse struct {
	ConnectUrls []string         `json:"connect_urls"`
	Cluster     jetStreamCluster `json:"cluster"`
	ServerName  string           `json:"server_name"`
}

type jetStreamCluster struct {
	HostUrls []string `json:"urls"`
}

type accountDetail struct {
	Name    string          `json:"name"`
	Streams []*streamDetail `json:"stream_detail"`
}

type metaCluster struct {
	ClusterSize int `json:"cluster_size"`
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
	Cluster        consumerCluster        `json:"cluster"`
}

type consumerCluster struct {
	Leader string `json:"leader"`
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
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	jsMetadata, err := parseNATSJetStreamMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing NATS JetStream metadata: %w", err)
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

	account, err := GetFromAuthOrMeta(config, "account")
	if err != nil {
		return meta, err
	}
	meta.account = account

	if config.TriggerMetadata["stream"] == "" {
		return meta, errors.New("no stream name given")
	}
	meta.stream = config.TriggerMetadata["stream"]

	if config.TriggerMetadata["consumer"] == "" {
		return meta, errors.New("no consumer name given")
	}
	meta.consumer = config.TriggerMetadata["consumer"]

	meta.lagThreshold = defaultJetStreamLagThreshold

	if val, ok := config.TriggerMetadata[jetStreamLagThresholdMetricName]; ok {
		t, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return meta, fmt.Errorf("error parsing %s: %w", jetStreamLagThresholdMetricName, err)
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

	natsServerEndpoint, err := GetFromAuthOrMeta(config, "natsServerMonitoringEndpoint")
	if err != nil {
		return meta, err
	}
	useHTTPS := false
	if val, ok := config.TriggerMetadata["useHttps"]; ok {
		useHTTPS, err = strconv.ParseBool(val)
		if err != nil {
			return meta, fmt.Errorf("useHTTPS parsing error %w", err)
		}
	}
	meta.monitoringURL = getNATSJetStreamMonitoringURL(useHTTPS, natsServerEndpoint, meta.account)

	return meta, nil
}

func (s *natsJetStreamScaler) getNATSJetstreamMonitoringData(ctx context.Context, natsJetStreamMonitoringURL string) error {
	// save the leader URL, then we can check if it has changed
	cachedConsumerLeader := s.metadata.consumerLeader
	// default URL (standalone)
	monitoringURL := natsJetStreamMonitoringURL
	// use the leader URL if we already have it
	if s.metadata.monitoringLeaderURL != "" {
		monitoringURL = s.metadata.monitoringLeaderURL
	}

	jetStreamAccountResp, err := s.getNATSJetstreamMonitoringRequest(ctx, monitoringURL)
	if err != nil {
		return err
	}

	consumerFound := s.setNATSJetStreamMonitoringData(jetStreamAccountResp, "")

	// invalidate the cached data if we used it but nothing was found
	if cachedConsumerLeader != "" && !consumerFound {
		s.invalidateNATSJetStreamCachedMonitoringData()
	}

	// the leader name hasn't changed from the previous run, we can assume we just queried the correct leader node
	if consumerFound && cachedConsumerLeader != "" && cachedConsumerLeader == s.metadata.consumerLeader {
		return nil
	}

	if s.metadata.clusterSize > 1 {
		// we know who the consumer leader and its monitoring url is, query it directly
		if s.metadata.consumerLeader != "" && s.metadata.monitoringLeaderURL != "" {
			natsJetStreamMonitoringLeaderURL := s.metadata.monitoringLeaderURL

			jetStreamAccountResp, err = s.getNATSJetstreamMonitoringRequest(ctx, natsJetStreamMonitoringLeaderURL)
			if err != nil {
				return err
			}

			s.setNATSJetStreamMonitoringData(jetStreamAccountResp, natsJetStreamMonitoringLeaderURL)
			return nil
		}

		// we haven't found the consumer yet, grab the list of hosts and try each one
		natsJetStreamMonitoringServerURL, err := s.getNATSJetStreamMonitoringServerURL("")
		if err != nil {
			return err
		}

		jetStreamServerResp, err := s.getNATSJetstreamServerInfo(ctx, natsJetStreamMonitoringServerURL)
		if err != nil {
			return err
		}

		isNodeAdvertised := true
		clusterUrls := jetStreamServerResp.ConnectUrls
		if len(clusterUrls) == 0 {
			isNodeAdvertised = false
			// append current node's `server_name` to check if it is a leader
			// even though `server_name` is not an url, it will be split by first . (dot)
			// to get the node's name anyway
			clusterUrls = append(clusterUrls, jetStreamServerResp.ServerName)
			clusterUrls = append(clusterUrls, jetStreamServerResp.Cluster.HostUrls...)
		}

		for _, clusterURL := range clusterUrls {
			var (
				node                           string
				natsJetStreamMonitoringNodeURL string
			)

			if isNodeAdvertised {
				// get hostname from the url
				// nats-1.nats.svc.cluster.local:4221 -> nats-1.nats.svc.cluster.local, or
				// 172.0.1.3:4221 -> 172.0.1.3
				nodeHostname := strings.Split(clusterURL, ":")[0]
				natsJetStreamMonitoringServerURL, err := s.getNATSJetStreamMonitoringServerURL(nodeHostname)
				if err != nil {
					return err
				}

				// Query server info to get its name
				jetStreamServerResp, err := s.getNATSJetstreamServerInfo(ctx, natsJetStreamMonitoringServerURL)
				if err != nil {
					return err
				}

				node = jetStreamServerResp.ServerName

				natsJetStreamMonitoringNodeURL, err = s.getNATSJetStreamMonitoringNodeURL(nodeHostname)
				if err != nil {
					return err
				}
			} else {
				node = strings.Split(clusterURL, ".")[0]
				natsJetStreamMonitoringNodeURL, err = s.getNATSJetStreamMonitoringNodeURLByNode(node)
				if err != nil {
					return err
				}
			}

			jetStreamAccountResp, err = s.getNATSJetstreamMonitoringRequest(ctx, natsJetStreamMonitoringNodeURL)
			if err != nil {
				return err
			}

			for _, jetStreamAccount := range jetStreamAccountResp.Accounts {
				if jetStreamAccount.Name == s.metadata.account {
					for _, stream := range jetStreamAccount.Streams {
						if stream.Name == s.metadata.stream {
							for _, consumer := range stream.Consumers {
								if consumer.Name == s.metadata.consumer {
									// this node is the consumer leader
									if node == consumer.Cluster.Leader {
										s.setNATSJetStreamMonitoringData(jetStreamAccountResp, natsJetStreamMonitoringNodeURL)
										return nil
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return nil
}

func (s *natsJetStreamScaler) setNATSJetStreamMonitoringData(jetStreamAccountResp *jetStreamEndpointResponse, leaderURL string) bool {
	s.metadata.clusterSize = jetStreamAccountResp.MetaCluster.ClusterSize

	// find and assign the stream that we are looking for.
	for _, jsAccount := range jetStreamAccountResp.Accounts {
		if jsAccount.Name == s.metadata.account {
			for _, stream := range jsAccount.Streams {
				if stream.Name == s.metadata.stream {
					s.stream = stream

					for _, consumer := range stream.Consumers {
						if consumer.Name == s.metadata.consumer {
							s.metadata.consumerLeader = consumer.Cluster.Leader
							if leaderURL != "" {
								s.metadata.monitoringLeaderURL = leaderURL
							}
							return true
						}
					}
				}
			}
		}
	}
	return false
}

func (s *natsJetStreamScaler) invalidateNATSJetStreamCachedMonitoringData() {
	s.metadata.consumerLeader = ""
	s.metadata.monitoringLeaderURL = ""
	s.stream = nil
}

func (s *natsJetStreamScaler) getNATSJetstreamServerInfo(ctx context.Context, natsJetStreamMonitoringServerURL string) (*jetStreamServerEndpointResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, natsJetStreamMonitoringServerURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Error(err, "unable to access NATS JetStream monitoring server endpoint", "natsServerMonitoringURL", natsJetStreamMonitoringServerURL)
		return nil, err
	}

	defer resp.Body.Close()
	var jetStreamServerResp *jetStreamServerEndpointResponse
	if err = json.NewDecoder(resp.Body).Decode(&jetStreamServerResp); err != nil {
		s.logger.Error(err, "unable to decode NATS JetStream server details")
		return nil, err
	}

	return jetStreamServerResp, nil
}

func (s *natsJetStreamScaler) getNATSJetstreamMonitoringRequest(ctx context.Context, natsJetStreamMonitoringURL string) (*jetStreamEndpointResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, natsJetStreamMonitoringURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Error(err, "unable to access NATS JetStream monitoring endpoint", "natsServerMonitoringURL", natsJetStreamMonitoringURL)
		return nil, err
	}

	defer resp.Body.Close()
	var jsAccountResp *jetStreamEndpointResponse
	if err = json.NewDecoder(resp.Body).Decode(&jsAccountResp); err != nil {
		s.logger.Error(err, "unable to decode NATS JetStream account details")
		return nil, err
	}
	return jsAccountResp, nil
}

func getNATSJetStreamMonitoringURL(useHTTPS bool, natsServerEndpoint string, account string) string {
	scheme := natsHTTPProtocol
	if useHTTPS {
		scheme = natsHTTPSProtocol
	}
	return fmt.Sprintf("%s://%s/jsz?acc=%s&consumers=true&config=true", scheme, natsServerEndpoint, account)
}

func (s *natsJetStreamScaler) getNATSJetStreamMonitoringServerURL(nodeHostname string) (string, error) {
	jsURL, err := url.Parse(s.metadata.monitoringURL)
	if err != nil {
		s.logger.Error(err, "unable to parse monitoring URL to create server URL", "natsServerMonitoringURL", s.metadata.monitoringURL)
		return "", err
	}

	host := jsURL.Host
	if nodeHostname != "" {
		host = nodeHostname

		if port := jsURL.Port(); port != "" {
			host = net.JoinHostPort(host, port)
		}
	}

	return fmt.Sprintf("%s://%s/varz", jsURL.Scheme, host), nil
}

func (s *natsJetStreamScaler) getNATSJetStreamMonitoringNodeURL(nodeHostname string) (string, error) {
	jsURL, err := url.Parse(s.metadata.monitoringURL)
	if err != nil {
		s.logger.Error(err, "unable to parse monitoring URL to create node URL", "natsServerMonitoringURL", s.metadata.monitoringURL)
		return "", err
	}

	// set the port to the monitoringURL port if exists
	if port := jsURL.Port(); port != "" {
		nodeHostname = net.JoinHostPort(nodeHostname, port)
	}

	return fmt.Sprintf("%s://%s%s?%s", jsURL.Scheme, nodeHostname, jsURL.Path, jsURL.RawQuery), nil
}

func (s *natsJetStreamScaler) getNATSJetStreamMonitoringNodeURLByNode(node string) (string, error) {
	jsURL, err := url.Parse(s.metadata.monitoringURL)
	if err != nil {
		s.logger.Error(err, "unable to parse monitoring URL to create node URL", "natsServerMonitoringURL", s.metadata.monitoringURL)
		return "", err
	}
	return fmt.Sprintf("%s://%s.%s%s?%s", jsURL.Scheme, node, jsURL.Host, jsURL.Path, jsURL.RawQuery), nil
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
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, metricName),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.lagThreshold),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric,
		Type:     jetStreamMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

func (s *natsJetStreamScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	err := s.getNATSJetstreamMonitoringData(ctx, s.metadata.monitoringURL)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	if s.stream == nil {
		return []external_metrics.ExternalMetricValue{}, false, errors.New("stream not found")
	}

	totalLag := s.getMaxMsgLag()
	s.logger.V(1).Info("NATS JetStream Scaler: Providing metrics based on totalLag, threshold", "totalLag", totalLag, "lagThreshold", s.metadata.lagThreshold)

	metric := GenerateMetricInMili(metricName, float64(totalLag))

	return []external_metrics.ExternalMetricValue{metric}, totalLag > s.metadata.activationLagThreshold, nil
}

func (s *natsJetStreamScaler) Close(context.Context) error {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}
