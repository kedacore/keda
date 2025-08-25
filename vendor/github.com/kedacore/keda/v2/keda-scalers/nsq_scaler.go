package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type nsqScaler struct {
	metricType v2.MetricTargetType
	metadata   nsqMetadata
	httpClient *http.Client
	scheme     string
	logger     logr.Logger
}

type nsqMetadata struct {
	NSQLookupdHTTPAddresses  []string `keda:"name=nsqLookupdHTTPAddresses,  order=triggerMetadata;resolvedEnv"`
	Topic                    string   `keda:"name=topic,                    order=triggerMetadata;resolvedEnv"`
	Channel                  string   `keda:"name=channel,                  order=triggerMetadata;resolvedEnv"`
	DepthThreshold           int64    `keda:"name=depthThreshold,           order=triggerMetadata;resolvedEnv, default=10"`
	ActivationDepthThreshold int64    `keda:"name=activationDepthThreshold, order=triggerMetadata;resolvedEnv, default=0"`
	UseHTTPS                 bool     `keda:"name=useHttps,                 order=triggerMetadata;resolvedEnv, default=false"`
	UnsafeSSL                bool     `keda:"name=unsafeSsl,                order=triggerMetadata;resolvedEnv, default=false"`

	triggerIndex int
}

const (
	nsqMetricType = "External"
)

func NewNSQScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "nsq_scaler")

	nsqMetadata, err := parseNSQMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing NSQ metadata: %w", err)
	}

	scheme := "http"
	if nsqMetadata.UseHTTPS {
		scheme = "https"
	}

	return &nsqScaler{
		metricType: metricType,
		metadata:   nsqMetadata,
		httpClient: kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, nsqMetadata.UnsafeSSL),
		scheme:     scheme,
		logger:     logger,
	}, nil
}

func (m nsqMetadata) Validate() error {
	if len(m.NSQLookupdHTTPAddresses) == 0 {
		return fmt.Errorf("no nsqLookupdHTTPAddresses given")
	}

	if m.DepthThreshold <= 0 {
		return fmt.Errorf("depthThreshold must be a positive integer")
	}

	if m.ActivationDepthThreshold < 0 {
		return fmt.Errorf("activationDepthThreshold must be greater than or equal to 0")
	}

	return nil
}

func parseNSQMetadata(config *scalersconfig.ScalerConfig) (nsqMetadata, error) {
	meta := nsqMetadata{triggerIndex: config.TriggerIndex}
	if err := config.TypedConfig(&meta); err != nil {
		return meta, fmt.Errorf("error parsing nsq metadata: %w", err)
	}

	return meta, nil
}

func (s nsqScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	depth, err := s.getTopicChannelDepth(ctx)

	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	s.logger.V(1).Info("GetMetricsAndActivity", "metricName", metricName, "depth", depth)

	metric := GenerateMetricInMili(metricName, float64(depth))

	return []external_metrics.ExternalMetricValue{metric}, depth > s.metadata.ActivationDepthThreshold, nil
}

func (s nsqScaler) getTopicChannelDepth(ctx context.Context) (int64, error) {
	nsqdHosts, err := s.getTopicProducers(ctx, s.metadata.Topic)
	if err != nil {
		return -1, fmt.Errorf("error getting nsqd hosts: %w", err)
	}

	if len(nsqdHosts) == 0 {
		s.logger.V(1).Info("no nsqd hosts found for topic", "topic", s.metadata.Topic)
		return 0, nil
	}

	depth, err := s.aggregateDepth(ctx, nsqdHosts, s.metadata.Topic, s.metadata.Channel)
	if err != nil {
		return -1, fmt.Errorf("error getting topic/channel depth: %w", err)
	}

	return depth, nil
}

func (s nsqScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	metricName := fmt.Sprintf("nsq-%s-%s", s.metadata.Topic, s.metadata.Channel)

	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(metricName)),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.DepthThreshold),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: nsqMetricType}
	return []v2.MetricSpec{metricSpec}
}

func (s nsqScaler) Close(context.Context) error {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}

type lookupResponse struct {
	Producers []struct {
		HTTPPort         int    `json:"http_port"`
		BroadcastAddress string `json:"broadcast_address"`
	}
}

type lookupResult struct {
	host           string
	lookupResponse *lookupResponse
	err            error
}

func (s *nsqScaler) getTopicProducers(ctx context.Context, topic string) ([]string, error) {
	var wg sync.WaitGroup
	resultCh := make(chan lookupResult, len(s.metadata.NSQLookupdHTTPAddresses))

	for _, host := range s.metadata.NSQLookupdHTTPAddresses {
		wg.Add(1)
		go func(host string, topic string) {
			defer wg.Done()
			resp, err := s.getLookup(ctx, host, topic)
			resultCh <- lookupResult{host, resp, err}
		}(host, topic)
	}

	wg.Wait()
	close(resultCh)

	var nsqdHostMap = make(map[string]bool)
	for result := range resultCh {
		if result.err != nil {
			return nil, fmt.Errorf("error getting lookup from host '%s': %w", result.host, result.err)
		}

		if result.lookupResponse == nil {
			// topic is not found on a single nsqlookupd host, it may exist on another
			continue
		}

		for _, producer := range result.lookupResponse.Producers {
			nsqdHost := net.JoinHostPort(producer.BroadcastAddress, strconv.Itoa(producer.HTTPPort))
			nsqdHostMap[nsqdHost] = true
		}
	}

	var nsqdHosts []string
	for nsqdHost := range nsqdHostMap {
		nsqdHosts = append(nsqdHosts, nsqdHost)
	}

	return nsqdHosts, nil
}

func (s *nsqScaler) getLookup(ctx context.Context, host string, topic string) (*lookupResponse, error) {
	lookupURL := url.URL{
		Scheme: s.scheme,
		Host:   host,
		Path:   "lookup",
	}
	req, err := http.NewRequestWithContext(ctx, "GET", lookupURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json; charset=utf-8")

	params := url.Values{"topic": {topic}}
	req.URL.RawQuery = params.Encode()

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code '%s'", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var lookupResponse lookupResponse
	err = json.Unmarshal(body, &lookupResponse)
	if err != nil {
		return nil, err
	}

	return &lookupResponse, nil
}

type statsResponse struct {
	Topics []struct {
		TopicName string `json:"topic_name"`
		Depth     int64  `json:"depth"`
		Channels  []struct {
			ChannelName string `json:"channel_name"`
			Depth       int64  `json:"depth"`  // num messages in the queue (mem + disk)
			Paused      bool   `json:"paused"` // if paused, consumers will not receive messages
		}
	}
}

type statsResult struct {
	host          string
	statsResponse *statsResponse
	err           error
}

func (s *nsqScaler) aggregateDepth(ctx context.Context, nsqdHosts []string, topic string, channel string) (int64, error) {
	wg := sync.WaitGroup{}
	resultCh := make(chan statsResult, len(nsqdHosts))

	for _, host := range nsqdHosts {
		wg.Add(1)
		go func(host string, topic string) {
			defer wg.Done()
			resp, err := s.getStats(ctx, host, topic)
			resultCh <- statsResult{host, resp, err}
		}(host, topic)
	}

	wg.Wait()
	close(resultCh)

	var depth int64
	for result := range resultCh {
		if result.err != nil {
			return -1, fmt.Errorf("error getting stats from host '%s': %w", result.host, result.err)
		}

		for _, t := range result.statsResponse.Topics {
			if t.TopicName != topic {
				// this should never happen as we make the /stats call with the "topic" param
				continue
			}

			if len(t.Channels) == 0 {
				// topic exists with no channels, but there are messages in the topic -> we should still scale to bootstrap
				s.logger.V(1).Info("no channels exist for topic", "topic", topic, "channel", channel, "host", result.host)
				depth += t.Depth
				continue
			}

			channelExists := false
			for _, ch := range t.Channels {
				if ch.ChannelName != channel {
					continue
				}
				channelExists = true
				if ch.Paused {
					// if it's paused on a single nsqd host, it's depth should not go into the aggregate
					// meaning if paused on all nsqd hosts => depth == 0
					s.logger.V(1).Info("channel is paused", "topic", topic, "channel", channel, "host", result.host)
					continue
				}
				depth += ch.Depth
			}
			if !channelExists {
				// topic exists with channels, but not the one in question - fallback to topic depth
				s.logger.V(1).Info("channel does not exist for topic", "topic", topic, "channel", channel, "host", result.host)
				depth += t.Depth
			}
		}
	}

	return depth, nil
}

func (s *nsqScaler) getStats(ctx context.Context, host string, topic string) (*statsResponse, error) {
	statsURL := url.URL{
		Scheme: s.scheme,
		Host:   host,
		Path:   "stats",
	}
	req, err := http.NewRequestWithContext(ctx, "GET", statsURL.String(), nil)
	if err != nil {
		return nil, err
	}

	// "channel" is a query param as well, but if used and the channel does not exist
	// we do not receive any stats for the existing topic
	params := url.Values{
		"format":          {"json"},
		"include_clients": {"false"},
		"include_mem":     {"false"},
		"topic":           {topic},
	}
	req.URL.RawQuery = params.Encode()

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code '%s'", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var statsResponse statsResponse
	err = json.Unmarshal(body, &statsResponse)
	if err != nil {
		return nil, err
	}

	return &statsResponse, nil
}
