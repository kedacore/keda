package scalers

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

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
	defaultRabbitMQQueueLength  = 20
	rabbitMetricType            = "External"
	rabbitIncludeUnacked        = "includeUnacked"
	defaultIncludeUnacked       = false
	amqps                       = "amqps"
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
	includeUnacked bool   // if true uses HTTP API and requires apiHost, if false uses AMQP and requires host
	ca             string //Certificate authority file for TLS client authentication. Optional. If authmode is sasl_ssl, this is required.
	cert           string //Certificate for client authentication. Optional. If authmode is sasl_ssl, this is required.
	key            string //Key for client authentication. Optional. If authmode is sasl_ssl, this is required.
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
	} else {
		conn, ch, err := getConnectionAndChannel(meta.host, meta.ca, meta.cert, meta.key)
		if err != nil {
			return nil, fmt.Errorf("error establishing rabbitmq connection: %s", err)
		}

		return &rabbitMQScaler{
			metadata:   meta,
			connection: conn,
			channel:    ch,
		}, nil
	}
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
		if val, ok := authParams["apiHost"]; ok {
			meta.apiHost = val
		} else if val, ok := metadata["apiHost"]; ok {
			hostSetting := val

			if val, ok := resolvedEnv[hostSetting]; ok {
				meta.apiHost = val
			}
		}

		if meta.apiHost == "" {
			return nil, fmt.Errorf("no apiHost setting given")
		}
	} else {
		if val, ok := authParams["host"]; ok {
			meta.host = val
		} else if val, ok := metadata["host"]; ok {
			hostSetting := val

			if val, ok := resolvedEnv[hostSetting]; ok {
				meta.host = val
			}
		}

		if meta.host == "" {
			return nil, fmt.Errorf("no host setting given")
		}
	}

	if strings.HasPrefix(meta.host, amqps) {
		if val, ok := authParams["ca"]; ok {
			meta.ca = val
		} else {
			return nil, fmt.Errorf("no ca given")
		}
		if val, ok := authParams["cert"]; ok {
			meta.cert = val
		} else {
			return nil, fmt.Errorf("no cert given")
		}
		if val, ok := authParams["key"]; ok {
			meta.key = val
		} else {
			return nil, fmt.Errorf("no key given")
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

func getConnectionAndChannel(host string, caFile string, certFile string, keyFile string) (*amqp.Connection, *amqp.Channel, error) {
	var conn *amqp.Connection
	var err error

	if strings.HasPrefix(host, amqps) {
		cfg := new(tls.Config)

		cfg.RootCAs = x509.NewCertPool()

		if ca, err := ioutil.ReadFile(caFile); err == nil {
			cfg.RootCAs.AppendCertsFromPEM(ca)
		}

		if cert, err := tls.LoadX509KeyPair(certFile, keyFile); err == nil {
			cfg.Certificates = append(cfg.Certificates, cert)
		}
		conn, err = amqp.DialTLS(host, cfg)
	} else {
		conn, err = amqp.Dial(host)
	}

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
		info, err := s.getQueueInfoViaHttp()
		if err != nil {
			return -1, err
		} else {
			// messages count includes count of ready and unack-ed
			return info.Messages, nil
		}
	} else {
		items, err := s.channel.QueueInspect(s.metadata.queueName)
		if err != nil {
			return -1, err
		} else {
			return items.Messages, nil
		}
	}
}

func getJson(url string, target interface{}) error {
	var client = &http.Client{Timeout: 5 * time.Second}
	r, err := client.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	if r.StatusCode == 200 {
		return json.NewDecoder(r.Body).Decode(target)
	} else {
		body, _ := ioutil.ReadAll(r.Body)
		return fmt.Errorf("error requesting rabbitMQ API status: %s, response: %s, from: %s", r.Status, body, url)
	}
}

func (s *rabbitMQScaler) getQueueInfoViaHttp() (*queueInfo, error) {
	parsedUrl, err := url.Parse(s.metadata.apiHost)

	if err != nil {
		return nil, err
	}

	vhost := parsedUrl.Path

	if vhost == "" || vhost == "/" || vhost == "//" {
		vhost = "/%2F"
	}

	parsedUrl.Path = ""

	getQueueInfoManagementURI := fmt.Sprintf("%s/%s%s/%s", parsedUrl.String(), "api/queues", vhost, s.metadata.queueName)

	info := queueInfo{}
	err = getJson(getQueueInfoManagementURI, &info)

	if err != nil {
		return nil, err
	} else {
		return &info, nil
	}
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
