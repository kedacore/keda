/*
Copyright 2023 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package scalers

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	mqttMetricType = "External"

	// mqttSubackFailure is the SUBACK return code a broker sends when it
	// rejects a subscription. Paho reports it via the token's Result()
	// rather than Error(), so it has to be checked explicitly.
	mqttSubackFailure byte = 0x80
)

// mqttTLSSchemes are the broker URI schemes for which paho dials a TLS
// connection. Every other scheme (tcp, mqtt, ws, ...) is plaintext, and
// paho silently ignores any TLS config set on the client options.
var mqttTLSSchemes = map[string]bool{
	"ssl":      true,
	"tls":      true,
	"mqtts":    true,
	"mqtt+ssl": true,
	"tcps":     true,
	"wss":      true,
}

// mqttScalerMetadata holds the parsed trigger configuration for a
// ScaledObject using `type: mqtt`.
type mqttScalerMetadata struct {
	triggerIndex int

	BrokerAddress string `keda:"name=brokerAddress, order=triggerMetadata"`
	Topic         string `keda:"name=topic, order=triggerMetadata"`
	WindowSeconds int    `keda:"name=windowSeconds, order=triggerMetadata, optional, default=30"`
	QueryValue    int64  `keda:"name=queryValue, order=triggerMetadata"`
	QoS           int    `keda:"name=qos, order=triggerMetadata, optional, default=1"`

	ConnectRetryIntervalSeconds int  `keda:"name=connectRetryIntervalSeconds, order=triggerMetadata, optional, default=2"`
	MaxReconnectIntervalSeconds int  `keda:"name=maxReconnectIntervalSeconds, order=triggerMetadata, optional, default=60"`
	EnableTLS                   bool `keda:"name=enableTLS, order=triggerMetadata, optional"`
	UnsafeSsl                   bool `keda:"name=unsafeSsl, order=triggerMetadata, optional"`

	Username    string `keda:"name=username, order=authParams, optional"`
	Password    string `keda:"name=password, order=authParams, optional"`
	Ca          string `keda:"name=ca, order=authParams, optional"`
	Cert        string `keda:"name=cert, order=authParams, optional"`
	Key         string `keda:"name=key, order=authParams, optional"`
	KeyPassword string `keda:"name=keyPassword, order=authParams, optional"`
}

func parseMqttScalerMetadata(config *scalersconfig.ScalerConfig) (mqttScalerMetadata, error) {
	meta := mqttScalerMetadata{triggerIndex: config.TriggerIndex}
	if err := config.TypedConfig(&meta); err != nil {
		return meta, fmt.Errorf("error parsing mqtt scaler metadata: %w", err)
	}
	if err := meta.validate(); err != nil {
		return meta, err
	}
	return meta, nil
}

func (m *mqttScalerMetadata) validate() error {
	if m.QueryValue <= 0 {
		return errors.New("queryValue must be greater than zero")
	}
	if m.WindowSeconds <= 0 {
		return errors.New("windowSeconds must be greater than zero")
	}
	if m.QoS < 0 || m.QoS > 2 {
		return errors.New("qos must be 0, 1, or 2")
	}
	if m.ConnectRetryIntervalSeconds <= 0 {
		return errors.New("connectRetryIntervalSeconds must be greater than zero")
	}
	if m.MaxReconnectIntervalSeconds <= 0 {
		return errors.New("maxReconnectIntervalSeconds must be greater than zero")
	}

	broker, err := url.Parse(m.BrokerAddress)
	if err != nil || broker.Scheme == "" || broker.Host == "" {
		return fmt.Errorf("brokerAddress must be a URI of the form scheme://host:port, got %q", m.BrokerAddress)
	}
	isTLSScheme := mqttTLSSchemes[strings.ToLower(broker.Scheme)]
	if m.EnableTLS && !isTLSScheme {
		return fmt.Errorf("enableTLS requires a TLS brokerAddress scheme (ssl, tls, mqtts, mqtt+ssl, tcps or wss), got %q", broker.Scheme)
	}
	if m.Username != "" && !isTLSScheme {
		return fmt.Errorf("username/password must not be sent over the plaintext %q scheme; use a TLS brokerAddress scheme (ssl, tls, mqtts, mqtt+ssl, tcps or wss)", broker.Scheme)
	}
	return nil
}

// messageWindow is a thread-safe sliding window over recent MQTT
// message arrivals, plus a time-bounded "retained message seen" signal.
//
// It is written to from the MQTT client's own callback goroutine and
// read from GetMetricsAndActivity, which KEDA calls independently and
// concurrently (both the KEDA operator's own polling loop and the
// Kubernetes metrics adapter serving the HPA call this on their own
// schedules) — hence the mutex.
type messageWindow struct {
	mu             sync.Mutex
	timestamps     []time.Time
	window         time.Duration
	lastRetainedAt time.Time // zero value means "never seen"
}

func newMessageWindow(window time.Duration) *messageWindow {
	return &messageWindow{window: window}
}

func (w *messageWindow) record() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.timestamps = append(w.timestamps, time.Now())
}

func (w *messageWindow) markRetained() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.lastRetainedAt = time.Now()
}

// count prunes timestamps older than the window and returns the current
// message count, plus whether a retained message has been seen within
// the window. Both values are derived from wall-clock time rather than
// being consumed on read, so multiple independent, concurrent callers
// all get a consistent answer regardless of call order.
func (w *messageWindow) count() (int64, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()

	cutoff := time.Now().Add(-w.window)

	i := 0
	for i < len(w.timestamps) && w.timestamps[i].Before(cutoff) {
		i++
	}
	w.timestamps = w.timestamps[i:]

	retainedActive := !w.lastRetainedAt.IsZero() && w.lastRetainedAt.After(cutoff)
	return int64(len(w.timestamps)), retainedActive
}

type mqttScaler struct {
	metricType v2.MetricTargetType
	metadata   mqttScalerMetadata
	window     *messageWindow
	logger     logr.Logger

	// subscribeRetryInterval and maxSubscribeRetryInterval bound the
	// backoff used when the broker rejects a subscription on an otherwise
	// healthy connection.
	subscribeRetryInterval    time.Duration
	maxSubscribeRetryInterval time.Duration

	// mu guards the client lifecycle: Run and Close are started on
	// separate goroutines by KEDA, so Close may run before, during or
	// after Run creates the client. It also guards activeClosed so the
	// publish handler never sends on the active channel after Run has
	// closed it.
	mu           sync.Mutex
	client       mqtt.Client
	closed       bool
	activeClosed bool

	// newClient constructs the MQTT client. Defaults to mqtt.NewClient in
	// production; tests override this to inject a fake, so Run can be
	// exercised without a real network connection.
	newClient func(*mqtt.ClientOptions) mqtt.Client
}

// NewMqttScaler creates a new MQTT scaler. It implements PushScaler
// (via Run) rather than being purely poll-based: MQTT brokers don't
// expose a queryable queue-depth the way most other trigger sources do,
// so this scaler must already be subscribed and counting before KEDA
// asks for a metric, not query on demand.
func NewMqttScaler(config *scalersconfig.ScalerConfig) (PushScaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting mqtt scaler metric type: %w", err)
	}

	meta, err := parseMqttScalerMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing mqtt scaler metadata: %w", err)
	}

	return &mqttScaler{
		metricType: metricType,
		metadata:   meta,
		window:     newMessageWindow(time.Duration(meta.WindowSeconds) * time.Second),
		logger:     InitializeLogger(config, "mqtt_scaler"),
		newClient:  mqtt.NewClient,

		subscribeRetryInterval:    time.Duration(meta.ConnectRetryIntervalSeconds) * time.Second,
		maxSubscribeRetryInterval: time.Duration(meta.MaxReconnectIntervalSeconds) * time.Second,
	}, nil
}

func (s *mqttScaler) buildClientOptions() (*mqtt.ClientOptions, error) {
	opts := mqtt.NewClientOptions().
		AddBroker(s.metadata.BrokerAddress).
		SetConnectRetry(true).
		SetConnectRetryInterval(time.Duration(s.metadata.ConnectRetryIntervalSeconds) * time.Second).
		SetAutoReconnect(true).
		SetMaxReconnectInterval(time.Duration(s.metadata.MaxReconnectIntervalSeconds) * time.Second)

	if s.metadata.Username != "" {
		opts.SetUsername(s.metadata.Username)
		opts.SetPassword(s.metadata.Password)
	}

	if s.metadata.EnableTLS {
		tlsConfig, err := kedautil.NewTLSConfigWithPassword(
			s.metadata.Cert, s.metadata.Key, s.metadata.KeyPassword, s.metadata.Ca, s.metadata.UnsafeSsl)
		if err != nil {
			return nil, fmt.Errorf("error building TLS config for mqtt scaler: %w", err)
		}
		opts.SetTLSConfig(tlsConfig)
	}

	return opts, nil
}

// Run opens the MQTT connection and subscribes for the lifetime of the
// scaler. This is the "live" half of the scaler: GetMetricsAndActivity
// only ever reads state this maintains, it does no querying of its own.
func (s *mqttScaler) Run(ctx context.Context, active chan<- bool) {
	defer s.closeActive(active)

	opts, err := s.buildClientOptions()
	if err != nil {
		s.logger.Error(err, "failed to build mqtt client options")
		return
	}

	opts.SetDefaultPublishHandler(func(_ mqtt.Client, msg mqtt.Message) {
		if msg.Retained() {
			s.window.markRetained()
		} else {
			s.window.record()
		}
		// Push activation on every message, not just retained ones: KEDA
		// polls GetMetricsAndActivity at pollingInterval, and a burst can
		// fall out of a windowSeconds shorter than that before it is
		// ever observed. Only `true` is ever sent: inactivation events
		// are ignored by KEDA's push scaler loop, and deactivation
		// happens through the polled GetMetricsAndActivity path instead.
		s.notifyActive(active)
	})

	// Fires on every successful connect, including reconnects, so
	// resubscription after a dropped connection is automatic. Paho runs
	// this on its own goroutine, so retrying here does not block the
	// client.
	opts.SetOnConnectHandler(func(c mqtt.Client) {
		s.subscribe(ctx, c)
	})
	opts.SetConnectionLostHandler(func(_ mqtt.Client, err error) {
		s.logger.Error(err, "lost connection to broker, will reconnect")
	})
	opts.SetReconnectingHandler(func(_ mqtt.Client, _ *mqtt.ClientOptions) {
		s.logger.V(1).Info("attempting to reconnect to broker")
	})

	s.mu.Lock()
	if s.closed || ctx.Err() != nil {
		s.mu.Unlock()
		return
	}
	client := s.newClient(opts)
	s.client = client
	s.mu.Unlock()

	// With ConnectRetry enabled, Connect() retries in the background on
	// its own schedule rather than failing once, so this token reflects
	// only the first attempt — we don't treat its failure as fatal.
	token := client.Connect()
	go func() {
		if token.Wait() && token.Error() != nil {
			s.logger.Error(token.Error(), "initial connect attempt failed, will keep retrying")
		}
	}()

	<-ctx.Done()
}

// subscribe subscribes to the configured topic, retrying with bounded
// backoff if the broker rejects the subscription. Without this, a failed
// SUBACK on a healthy connection would leave the scaler connected but
// receiving nothing until the next reconnect. Retrying stops when ctx is
// cancelled or the connection drops (the next OnConnect resubscribes).
func (s *mqttScaler) subscribe(ctx context.Context, c mqtt.Client) {
	backoff := s.subscribeRetryInterval
	if backoff <= 0 {
		backoff = time.Second
	}
	for {
		s.logger.Info("subscribing to topic", "topic", s.metadata.Topic)
		err := subscribeError(c.Subscribe(s.metadata.Topic, byte(s.metadata.QoS), nil), s.metadata.Topic)
		if err == nil {
			return
		}
		s.logger.Error(err, "failed to subscribe to topic, will retry", "topic", s.metadata.Topic, "retryIn", backoff)

		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		if !c.IsConnectionOpen() {
			return
		}
		backoff *= 2
		if s.maxSubscribeRetryInterval > 0 && backoff > s.maxSubscribeRetryInterval {
			backoff = s.maxSubscribeRetryInterval
		}
	}
}

// subscribeError waits for a subscribe token and returns an error if the
// subscription failed, either at the transport level or because the
// broker answered with a failure SUBACK.
func subscribeError(token mqtt.Token, topic string) error {
	token.Wait()
	if err := token.Error(); err != nil {
		return err
	}
	if st, ok := token.(*mqtt.SubscribeToken); ok {
		if code, ok := st.Result()[topic]; ok && code == mqttSubackFailure {
			return fmt.Errorf("broker rejected subscription to %q", topic)
		}
	}
	return nil
}

// notifyActive sends a non-blocking activation event. KEDA's receiver is
// unbuffered and may be busy scaling, in which case the event is dropped:
// scaling is already in progress, and the next poll sees the same window.
func (s *mqttScaler) notifyActive(active chan<- bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeClosed {
		return
	}
	select {
	case active <- true:
	default:
	}
}

func (s *mqttScaler) closeActive(active chan<- bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeClosed = true
	close(active)
}

func (s *mqttScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex,
				kedautil.NormalizeString(fmt.Sprintf("mqtt-%s", s.metadata.Topic))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.QueryValue),
	}
	return []v2.MetricSpec{{External: externalMetric, Type: mqttMetricType}}
}

// GetMetricsAndActivity is what KEDA (and, separately, the metrics
// adapter serving the HPA) calls on its own polling schedule. All the
// real work already happened asynchronously in Run — this just reads
// the current window state.
func (s *mqttScaler) GetMetricsAndActivity(_ context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	count, retained := s.window.count()
	metric := GenerateMetricInMili(metricName, float64(count))
	isActive := count > 0 || retained
	return []external_metrics.ExternalMetricValue{metric}, isActive, nil
}

func (s *mqttScaler) Close(context.Context) error {
	s.mu.Lock()
	s.closed = true
	client := s.client
	s.client = nil
	s.mu.Unlock()

	// Disconnect also aborts an in-progress connect retry loop, so it is
	// called even if the client never finished connecting.
	if client != nil {
		client.Disconnect(250)
	}
	return nil
}
