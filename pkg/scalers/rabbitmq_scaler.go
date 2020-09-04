package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/streadway/amqp"
	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	rabbitQueueLengthMetricName = "queueLength"
	defaultRabbitMQQueueLength  = 20
	rabbitMetricType            = "External"
	rabbitIncludeUnacked        = "includeUnacked"
	defaultIncludeUnacked       = false
)

type rabbitMQScaler struct {
	metadata   *rabbitMQMetadata
	connection *amqp.Connection
	channel    *amqp.Channel
}

type rabbitMQMetadata struct {
	queueName      string
	host           string // connection string for AMQP protocol
	apiHost        string // connection string for management API requests
	queueLength    int
	includeUnacked bool // if true uses HTTP API and requires apiHost, if false uses AMQP and requires host
}

type queueInfo struct {
	Messages               int    `json:"messages"`
	MessagesUnacknowledged int    `json:"messages_unacknowledged"`
	Name                   string `json:"name"`
}

var rabbitmqLog = logf.Log.WithName("rabbitmq_scaler")

// NewRabbitMQScaler creates a new rabbitMQ scaler
func NewRabbitMQScaler(resolvedEnv, metadata, authParams map[string]string) (Scaler, error) {
	meta, err := parseRabbitMQMetadata(resolvedEnv, metadata, authParams)
	if err != nil {
		return nil, fmt.Errorf("error parsing rabbitmq metadata: %s", err)
	}

	if meta.includeUnacked {
		return &rabbitMQScaler{metadata: meta}, nil
	}
	
	conn, ch, err := getConnectionAndChannel(meta.host)
	if err != nil {
		return nil, fmt.Errorf("error establishing rabbitmq connection: %s", err)
	}

	return &rabbitMQScaler{
		metadata:   meta,
		connection: conn,
		channel:    ch,
	}, nil
}

func parseRabbitMQMetadata(resolvedEnv, metadata, authParams map[string]string) (*rabbitMQMetadata, error) {
	meta := rabbitMQMetadata{}

	meta.includeUnacked = defaultIncludeUnacked
	if val, ok := metadata[rabbitIncludeUnacked]; ok {
		includeUnacked, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("includeUnacked parsing error %s", err.Error())
		}
		meta.includeUnacked = includeUnacked
	}

	if meta.includeUnacked {
		if authParams["apiHost"] != "" {
			meta.apiHost = authParams["apiHost"]
		} else if metadata["apiHost"] != "" {
			meta.apiHost = metadata["apiHost"]
		} else if metadata["apiHostFromEnv"] != "" {
			meta.apiHost = resolvedEnv[metadata["apiHostFromEnv"]]
		}

		if meta.apiHost == "" {
			return nil, fmt.Errorf("no apiHost setting given")
		}
	} else {
		if authParams["host"] != "" {
			meta.host = authParams["host"]
		} else if metadata["host"] != "" {
			meta.host = metadata["host"]
		} else if metadata["hostFromEnv"] != "" {
			meta.host = resolvedEnv[metadata["hostFromEnv"]]
		}

		if meta.host == "" {
			return nil, fmt.Errorf("no host setting given")
		}
	}

	if val, ok := metadata["queueName"]; ok {
		meta.queueName = val
	} else {
		return nil, fmt.Errorf("no queue name given")
	}

	if val, ok := metadata[rabbitQueueLengthMetricName]; ok {
		queueLength, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("can't parse %s: %s", rabbitQueueLengthMetricName, err)
		}

		meta.queueLength = queueLength
	} else {
		meta.queueLength = defaultRabbitMQQueueLength
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
	messages, err := s.getQueueMessages()
	if err != nil {
		return false, fmt.Errorf("error inspecting rabbitMQ: %s", err)
	}

	return messages > 0, nil
}

func (s *rabbitMQScaler) getQueueMessages() (int, error) {
	if s.metadata.includeUnacked {
		info, err := s.getQueueInfoViaHTTP()
		if err != nil {
			return -1, err
		}

		// messages count includes count of ready and unack-ed
		return info.Messages, nil
	}

	items, err := s.channel.QueueInspect(s.metadata.queueName)
	if err != nil {
		return -1, err
	}

	return items.Messages, nil
}

func getJSON(url string, target interface{}) error {
	var client = &http.Client{Timeout: 5 * time.Second}
	r, err := client.Get(url)
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
	parsedURL, err := url.Parse(s.metadata.apiHost)

	if err != nil {
		return nil, err
	}

	vhost := parsedURL.Path

	if vhost == "" || vhost == "/" || vhost == "//" {
		vhost = "/%2F"
	}

	parsedURL.Path = ""

	getQueueInfoManagementURI := fmt.Sprintf("%s/%s%s/%s", parsedURL.String(), "api/queues", vhost, s.metadata.queueName)

	info := queueInfo{}
	err = getJSON(getQueueInfoManagementURI, &info)

	if err != nil {
		return nil, err
	}

	return &info, nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *rabbitMQScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	targetMetricValue := resource.NewQuantity(int64(s.metadata.queueLength), resource.DecimalSI)
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: fmt.Sprintf("%s-%s", "rabbitmq", s.metadata.queueName),
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: targetMetricValue,
		},
	}
	metricSpec := v2beta2.MetricSpec{
		External: externalMetric, Type: rabbitMetricType,
	}
	return []v2beta2.MetricSpec{metricSpec}
}

// GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *rabbitMQScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	messages, err := s.getQueueMessages()
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, fmt.Errorf("error inspecting rabbitMQ: %s", err)
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(messages), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}
