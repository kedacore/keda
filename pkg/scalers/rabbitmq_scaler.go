package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	"github.com/streadway/amqp"
	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	rabbitQueueLengthMetricName        = "queueLength"
	rabbitPublishedPerSecondMetricName = "publishRate"
	defaultRabbitMQQueueLength         = 20
	defaultRabbitMQPublishRate         = 0 // Default to zero to disable publish rate metering for back compat.
	rabbitMetricType                   = "External"
)

const (
	httpProtocol    = "http"
	amqpProtocol    = "amqp"
	autoProtocol    = "auto"
	defaultProtocol = autoProtocol
)

type rabbitMQScaler struct {
	metadata   *rabbitMQMetadata
	connection *amqp.Connection
	channel    *amqp.Channel
	httpClient *http.Client
}

type rabbitMQMetadata struct {
	queueName   string
	queueLength int
	publishRate float64 // Publish/sec. rate on the queue, requires HTTP protocol
	host        string  // connection string for either HTTP or AMQP protocol
	protocol    string  // either http or amqp protocol
	vhostName   *string // override the vhost from the connection info
}

type queueInfo struct {
	Messages               int         `json:"messages"`
	MessagesUnacknowledged int         `json:"messages_unacknowledged"`
	MessageStat            messageStat `json:"message_stats"`
	Name                   string      `json:"name"`
}

type messageStat struct {
	PublishDetail publishDetail `json:"publish_details"`
}

type publishDetail struct {
	Rate float64 `json:"rate"`
}

var rabbitmqLog = logf.Log.WithName("rabbitmq_scaler")

// NewRabbitMQScaler creates a new rabbitMQ scaler
func NewRabbitMQScaler(config *ScalerConfig) (Scaler, error) {
	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout)
	meta, err := parseRabbitMQMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing rabbitmq metadata: %s", err)
	}

	if meta.protocol == httpProtocol {
		return &rabbitMQScaler{
			metadata:   meta,
			httpClient: httpClient,
		}, nil
	}

	// Override vhost if requested.
	host := meta.host
	if meta.vhostName != nil {
		hostURI, err := amqp.ParseURI(host)
		if err != nil {
			return nil, fmt.Errorf("error parsing rabbitmq connection string: %s", err)
		}
		hostURI.Vhost = *meta.vhostName
		host = hostURI.String()
	}

	conn, ch, err := getConnectionAndChannel(host)
	if err != nil {
		return nil, fmt.Errorf("error establishing rabbitmq connection: %s", err)
	}

	return &rabbitMQScaler{
		metadata:   meta,
		connection: conn,
		channel:    ch,
		httpClient: httpClient,
	}, nil
}

func parseRabbitMQMetadata(config *ScalerConfig) (*rabbitMQMetadata, error) {
	meta := rabbitMQMetadata{}

	// Resolve protocol type
	meta.protocol = defaultProtocol
	if val, ok := config.AuthParams["protocol"]; ok {
		meta.protocol = val
	}
	if val, ok := config.TriggerMetadata["protocol"]; ok {
		meta.protocol = val
	}
	if meta.protocol != amqpProtocol && meta.protocol != httpProtocol && meta.protocol != autoProtocol {
		return nil, fmt.Errorf("the protocol has to be either `%s`, `%s`, or `%s` but is `%s`", amqpProtocol, httpProtocol, autoProtocol, meta.protocol)
	}

	// Resolve host value
	switch {
	case config.AuthParams["host"] != "":
		meta.host = config.AuthParams["host"]
	case config.TriggerMetadata["host"] != "":
		meta.host = config.TriggerMetadata["host"]
	case config.TriggerMetadata["hostFromEnv"] != "":
		meta.host = config.ResolvedEnv[config.TriggerMetadata["hostFromEnv"]]
	default:
		return nil, fmt.Errorf("no host setting given")
	}

	// If the protocol is auto, check the host scheme.
	if meta.protocol == autoProtocol {
		parsedURL, err := url.Parse(meta.host)
		if err != nil {
			return nil, fmt.Errorf("can't parse host to find protocol: %s", err)
		}
		switch parsedURL.Scheme {
		case "amqp", "amqps":
			meta.protocol = amqpProtocol
		case "http", "https":
			meta.protocol = httpProtocol
		default:
			return nil, fmt.Errorf("unknown host URL scheme `%s`", parsedURL.Scheme)
		}
	}

	// Resolve queueName
	if val, ok := config.TriggerMetadata["queueName"]; ok {
		meta.queueName = val
	} else {
		return nil, fmt.Errorf("no queue name given")
	}

	// Resolve publishRate
	if val, ok := config.TriggerMetadata[rabbitPublishedPerSecondMetricName]; ok {
		publishRate, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("can't parse %s: %s", rabbitPublishedPerSecondMetricName, err)
		}

		meta.publishRate = publishRate
	} else {
		meta.publishRate = defaultRabbitMQPublishRate
	}

	val, ok := config.TriggerMetadata[rabbitQueueLengthMetricName]
	switch {
	case ok:
		queueLength, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("can't parse %s: %s", rabbitQueueLengthMetricName, err)
		}
		meta.queueLength = queueLength
	case meta.publishRate > 0:
		meta.queueLength = 0
	default:
		meta.queueLength = defaultRabbitMQQueueLength
	}

	if meta.publishRate > 0 && meta.queueLength > 0 {
		return nil, fmt.Errorf("only one of queueLength or publishRate can be specified; use two separate triggers if both are desired")
	}

	if meta.publishRate > 0 && meta.protocol != httpProtocol {
		return nil, fmt.Errorf("protocol %s not supported; must be http to use publishRate", meta.protocol)
	}

	// Resolve vhostName
	if val, ok := config.TriggerMetadata["vhostName"]; ok {
		meta.vhostName = &val
	}

	return &meta, nil
}

func getConnectionAndChannel(host string) (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.Dial(host)
	if err != nil {
		return nil, nil, err
	}

	channel, err := conn.Channel()
	if err != nil {
		return nil, nil, err
	}

	return conn, channel, nil
}

// Close disposes of RabbitMQ connections
func (s *rabbitMQScaler) Close() error {
	if s.connection != nil {
		err := s.connection.Close()
		if err != nil {
			rabbitmqLog.Error(err, "Error closing rabbitmq connection")
			return err
		}
	}
	return nil
}

// IsActive returns true if there are pending messages to be processed
func (s *rabbitMQScaler) IsActive(ctx context.Context) (bool, error) {
	messages, publishRate, err := s.getQueueStatus()
	if err != nil {
		return false, fmt.Errorf("error inspecting rabbitMQ: %s", err)
	}

	if s.metadata.queueLength > 0 {
		return messages > 0, nil
	}
	return publishRate > 0, nil
}

func (s *rabbitMQScaler) getQueueStatus() (int, float64, error) {
	if s.metadata.protocol == httpProtocol {
		info, err := s.getQueueInfoViaHTTP()
		if err != nil {
			return -1, -1, err
		}

		// messages count includes count of ready and unack-ed
		return info.Messages, info.MessageStat.PublishDetail.Rate, nil
	}

	items, err := s.channel.QueueInspect(s.metadata.queueName)
	if err != nil {
		return -1, -1, err
	}

	return items.Messages, 0, nil
}

func getJSON(httpClient *http.Client, url string, target interface{}) error {
	r, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	if r.StatusCode == 200 {
		return json.NewDecoder(r.Body).Decode(target)
	}

	body, _ := ioutil.ReadAll(r.Body)
	return fmt.Errorf("error requesting rabbitMQ API status: %s, response: %s, from: %s", r.Status, body, url)
}

func (s *rabbitMQScaler) getQueueInfoViaHTTP() (*queueInfo, error) {
	parsedURL, err := url.Parse(s.metadata.host)

	if err != nil {
		return nil, err
	}

	vhost := parsedURL.Path

	// Override vhost if requested.
	if s.metadata.vhostName != nil {
		vhost = "/" + *s.metadata.vhostName
	}

	if vhost == "" || vhost == "/" || vhost == "//" {
		vhost = "/%2F"
	}

	parsedURL.Path = ""

	getQueueInfoManagementURI := fmt.Sprintf("%s/%s%s/%s", parsedURL.String(), "api/queues", vhost, s.metadata.queueName)

	info := queueInfo{}
	err = getJSON(s.httpClient, getQueueInfoManagementURI, &info)

	if err != nil {
		return nil, err
	}

	return &info, nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *rabbitMQScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	var metricName string
	var metricValue *resource.Quantity
	if s.metadata.queueLength > 0 {
		metricName = kedautil.NormalizeString(fmt.Sprintf("%s-%s", "rabbitmq", s.metadata.queueName))
		metricValue = resource.NewQuantity(int64(s.metadata.queueLength), resource.DecimalSI)
	} else {
		metricName = kedautil.NormalizeString(fmt.Sprintf("%s-%s", "rabbitmq-rate", s.metadata.queueName))
		metricValue = resource.NewMilliQuantity(int64(s.metadata.publishRate*1000), resource.DecimalSI)
	}

	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: metricName,
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: metricValue,
		},
	}
	metricSpec := v2beta2.MetricSpec{
		External: externalMetric, Type: rabbitMetricType,
	}

	return []v2beta2.MetricSpec{metricSpec}
}

// GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *rabbitMQScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	messages, publishRate, err := s.getQueueStatus()
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, fmt.Errorf("error inspecting rabbitMQ: %s", err)
	}

	var metricValue resource.Quantity
	if s.metadata.queueLength > 0 {
		metricValue = *resource.NewQuantity(int64(messages), resource.DecimalSI)
	} else {
		metricValue = *resource.NewMilliQuantity(int64(publishRate*1000), resource.DecimalSI)
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      metricValue,
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}
