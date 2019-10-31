package scalers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/streadway/amqp"
	v2beta1 "k8s.io/api/autoscaling/v2beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	rabbitQueueLengthMetricName = "queueLength"
	rabbitMetricType            = "External"
)

type rabbitMQScaler struct {
	metadata   *rabbitMQMetadata
	connection *amqp.Connection
	channel    *amqp.Channel
}

type rabbitMQMetadata struct {
	queueName   string
	host        string
	queueLength int
}

var rabbitmqLog = logf.Log.WithName("rabbitmq_scaler")

// NewRabbitMQScaler creates a new rabbitMQ scaler
func NewRabbitMQScaler(resolvedEnv, metadata map[string]string) (Scaler, error) {
	meta, err := parseRabbitMQMetadata(resolvedEnv, metadata)
	if err != nil {
		return nil, fmt.Errorf("error parsing rabbitmq metadata: %s", err)
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

func parseRabbitMQMetadata(resolvedEnv, metadata map[string]string) (*rabbitMQMetadata, error) {
	meta := rabbitMQMetadata{}

	if val, ok := metadata["host"]; ok {
		hostSetting := val

		if val, ok := resolvedEnv[hostSetting]; ok {
			meta.host = val
		}
	}

	if meta.host == "" {
		return nil, fmt.Errorf("no host setting given")
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
		return nil, fmt.Errorf("no queue length given")
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
	err := s.connection.Close()
	if err != nil {
		rabbitmqLog.Error(err, "Error closing rabbitmq connection")
		return err
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
	items, err := s.channel.QueueInspect(s.metadata.queueName)
	if err != nil {
		return -1, err
	}

	return items.Messages, nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *rabbitMQScaler) GetMetricSpecForScaling() []v2beta1.MetricSpec {
	return []v2beta1.MetricSpec{
		{
			External: &v2beta1.ExternalMetricSource{
				MetricName:         rabbitQueueLengthMetricName,
				TargetAverageValue: resource.NewQuantity(int64(s.metadata.queueLength), resource.DecimalSI),
			},
			Type: rabbitMetricType,
		},
	}
}

// GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *rabbitMQScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	messages, err := s.getQueueMessages()
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, fmt.Errorf("error inspecting rabbitMQ: %s", err)
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: rabbitQueueLengthMetricName,
		Value:      *resource.NewQuantity(int64(messages), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}
