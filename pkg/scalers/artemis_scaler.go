package scalers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type artemisScaler struct {
	metadata *artemisMetadata
}

type artemisMetadata struct {
	managementEndpoint string
	queueName          string
	brokerName         string
	brokerAddress      string
	username           string
	password           string
	queueLength        int
}

type artemisMonitoring struct {
	Request   string `json:"request"`
	MsgCount  int    `json:"value"`
	Status    int    `json:"status"`
	Timestamp int64  `json:"timestamp"`
}

const (
	artemisQueueLengthMetricName = "queueLength"
	artemisMetricType            = "External"
	defaultArtemisQueueLength    = 10
)

var artemisLog = logf.Log.WithName("artemis_queue_scaler")

// NewArtemisQueueScaler creates a new artemis queue Scaler
func NewArtemisQueueScaler(resolvedSecrets, metadata, authParams map[string]string) (Scaler, error) {
	artemisMetadata, err := parseArtemisMetadata(resolvedSecrets, metadata, authParams)
	if err != nil {
		return nil, fmt.Errorf("error parsing artemis metadata: %s", err)
	}

	return &artemisScaler{
		metadata: artemisMetadata,
	}, nil
}

func parseArtemisMetadata(resolvedEnv, metadata, authParams map[string]string) (*artemisMetadata, error) {

	meta := artemisMetadata{}

	meta.queueLength = defaultArtemisQueueLength

	if metadata["managementEndpoint"] == "" {
		return nil, errors.New("no management endpoint given")
	}
	meta.managementEndpoint = metadata["managementEndpoint"]

	if metadata["queueName"] == "" {
		return nil, errors.New("no queue name given")
	}
	meta.queueName = metadata["queueName"]

	if metadata["brokerName"] == "" {
		return nil, errors.New("no broker name given")
	}
	meta.brokerName = metadata["brokerName"]

	if metadata["brokerAddress"] == "" {
		return nil, errors.New("no broker address given")
	}
	meta.brokerAddress = metadata["brokerAddress"]

	if val, ok := metadata["queueLength"]; ok {
		queueLength, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("can't parse queueLength: %s", err)
		}

		meta.queueLength = queueLength
	}

	if val, ok := authParams["username"]; ok {
		meta.username = val
	} else if val, ok := metadata["username"]; ok {
		username := val

		if val, ok := resolvedEnv[username]; ok {
			meta.username = val
		}
	}

	if meta.username == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}

	if val, ok := authParams["password"]; ok {
		meta.password = val
	} else if val, ok := metadata["password"]; ok {
		password := val

		if val, ok := resolvedEnv[password]; ok {
			meta.password = val
		}
	}

	if meta.password == "" {
		return nil, fmt.Errorf("password cannot be empty")
	}
	return &meta, nil
}

// IsActive determines if we need to scale from zero
func (s *artemisScaler) IsActive(ctx context.Context) (bool, error) {
	messages, err := s.getQueueMessageCount()
	if err != nil {
		artemisLog.Error(err, "Unable to access the artemis management endpoint", "managementEndpoint", s.metadata.managementEndpoint)
		return false, err
	}

	return messages > 0, nil
}

func (s *artemisScaler) getArtemisManagementEndpoint() string {
	return "http://" + s.metadata.managementEndpoint
}

func (s *artemisScaler) getMonitoringEndpoint() string {
	monitoringEndpoint := fmt.Sprintf("%s/console/jolokia/read/org.apache.activemq.artemis:broker=\"%s\",component=addresses,address=\"%s\",subcomponent=queues,routing-type=\"anycast\",queue=\"%s\"/MessageCount",
		s.getArtemisManagementEndpoint(), s.metadata.brokerName, s.metadata.brokerAddress, s.metadata.queueName)
	return monitoringEndpoint
}

func (s *artemisScaler) getQueueMessageCount() (int, error) {
	var messageCount int
	var monitoringInfo *artemisMonitoring
	messageCount = 0

	client := &http.Client{
		Timeout: time.Second * 3,
	}
	url := s.getMonitoringEndpoint()

	req, err := http.NewRequest("GET", url, nil)

	req.SetBasicAuth(s.metadata.username, s.metadata.password)

	if err != nil {
		return -1, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return -1, err
	}

	defer resp.Body.Close()

	json.NewDecoder(resp.Body).Decode(&monitoringInfo)
	if resp.StatusCode == 200 && monitoringInfo.Status == 200 {
		messageCount = monitoringInfo.MsgCount
	} else {
		return -1, fmt.Errorf("Artemis management endpoint response error code : %d", resp.StatusCode)
	}

	artemisLog.V(1).Info("Artemis scaler: Providing metrics based on current queue length ", messageCount, "queue length limit", s.metadata.queueLength)

	return messageCount, nil
}

func (s *artemisScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	targetMetricValue := resource.NewQuantity(int64(s.metadata.queueLength), resource.DecimalSI)
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: artemisQueueLengthMetricName,
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: targetMetricValue,
		},
	}
	metricSpec := v2beta2.MetricSpec{External: externalMetric, Type: artemisMetricType}
	return []v2beta2.MetricSpec{metricSpec}
}

//GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *artemisScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	messages, err := s.getQueueMessageCount()

	if err != nil {
		artemisLog.Error(err, "Unable to access the artemis management endpoint", "managementEndpoint", s.metadata.managementEndpoint)
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(messages), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

// Nothing to close here.
func (s *artemisScaler) Close() error {
	return nil
}
