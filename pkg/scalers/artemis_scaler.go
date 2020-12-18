package scalers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type artemisScaler struct {
	metadata   *artemisMetadata
	httpClient *http.Client
}

//revive:disable:var-naming breaking change on restApiTemplate, wouldn't bring any benefit to users
type artemisMetadata struct {
	managementEndpoint string
	queueName          string
	brokerName         string
	brokerAddress      string
	username           string
	password           string
	restAPITemplate    string
	queueLength        int
}

//revive:enable:var-naming

type artemisMonitoring struct {
	MsgCount  int   `json:"value"`
	Status    int   `json:"status"`
	Timestamp int64 `json:"timestamp"`
}

const (
	artemisMetricType         = "External"
	defaultArtemisQueueLength = 10
	defaultRestAPITemplate    = "http://<<managementEndpoint>>/console/jolokia/read/org.apache.activemq.artemis:broker=\"<<brokerName>>\",component=addresses,address=\"<<brokerAddress>>\",subcomponent=queues,routing-type=\"anycast\",queue=\"<<queueName>>\"/MessageCount"
)

var artemisLog = logf.Log.WithName("artemis_queue_scaler")

// NewArtemisQueueScaler creates a new artemis queue Scaler
func NewArtemisQueueScaler(config *ScalerConfig) (Scaler, error) {
	// do we need to guarantee this timeout for a specific
	// reason? if not, we can have buildScaler pass in
	// the global client
	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout)

	artemisMetadata, err := parseArtemisMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing artemis metadata: %s", err)
	}

	return &artemisScaler{
		metadata:   artemisMetadata,
		httpClient: httpClient,
	}, nil
}

func parseArtemisMetadata(config *ScalerConfig) (*artemisMetadata, error) {
	meta := artemisMetadata{}

	meta.queueLength = defaultArtemisQueueLength

	if val, ok := config.TriggerMetadata["restApiTemplate"]; ok && val != "" {
		meta.restAPITemplate = config.TriggerMetadata["restApiTemplate"]
	} else {
		meta.restAPITemplate = defaultRestAPITemplate
	}

	if config.TriggerMetadata["managementEndpoint"] == "" {
		return nil, errors.New("no management endpoint given")
	}
	meta.managementEndpoint = config.TriggerMetadata["managementEndpoint"]

	if config.TriggerMetadata["queueName"] == "" {
		return nil, errors.New("no queue name given")
	}
	meta.queueName = config.TriggerMetadata["queueName"]

	if config.TriggerMetadata["brokerName"] == "" {
		return nil, errors.New("no broker name given")
	}
	meta.brokerName = config.TriggerMetadata["brokerName"]

	if config.TriggerMetadata["brokerAddress"] == "" {
		return nil, errors.New("no broker address given")
	}
	meta.brokerAddress = config.TriggerMetadata["brokerAddress"]

	if val, ok := config.TriggerMetadata["queueLength"]; ok {
		queueLength, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("can't parse queueLength: %s", err)
		}

		meta.queueLength = queueLength
	}

	if val, ok := config.AuthParams["username"]; ok && val != "" {
		meta.username = val
	} else if val, ok := config.TriggerMetadata["username"]; ok && val != "" {
		username := val

		if val, ok := config.ResolvedEnv[username]; ok && val != "" {
			meta.username = val
		} else {
			meta.username = username
		}
	}

	if meta.username == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}

	if val, ok := config.AuthParams["password"]; ok && val != "" {
		meta.password = val
	} else if val, ok := config.TriggerMetadata["password"]; ok && val != "" {
		password := val

		if val, ok := config.ResolvedEnv[password]; ok && val != "" {
			meta.password = val
		} else {
			meta.password = password
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

func (s *artemisScaler) getMonitoringEndpoint() string {
	replacer := strings.NewReplacer("<<managementEndpoint>>", s.metadata.managementEndpoint,
		"<<queueName>>", s.metadata.queueName,
		"<<brokerName>>", s.metadata.brokerName,
		"<<brokerAddress>>", s.metadata.brokerAddress)

	monitoringEndpoint := replacer.Replace(s.metadata.restAPITemplate)

	return monitoringEndpoint
}

func (s *artemisScaler) getQueueMessageCount() (int, error) {
	var monitoringInfo *artemisMonitoring
	messageCount := 0

	client := s.httpClient
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

	if err := json.NewDecoder(resp.Body).Decode(&monitoringInfo); err != nil {
		return -1, err
	}
	if resp.StatusCode == 200 && monitoringInfo.Status == 200 {
		messageCount = monitoringInfo.MsgCount
	} else {
		return -1, fmt.Errorf("artemis management endpoint response error code : %d", resp.StatusCode)
	}

	artemisLog.V(1).Info(fmt.Sprintf("Artemis scaler: Providing metrics based on current queue length %d queue length limit %d", messageCount, s.metadata.queueLength))

	return messageCount, nil
}

func (s *artemisScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	targetMetricValue := resource.NewQuantity(int64(s.metadata.queueLength), resource.DecimalSI)
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: kedautil.NormalizeString(fmt.Sprintf("%s-%s-%s", "artemis", s.metadata.brokerName, s.metadata.queueName)),
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: targetMetricValue,
		},
	}
	metricSpec := v2beta2.MetricSpec{External: externalMetric, Type: artemisMetricType}
	return []v2beta2.MetricSpec{metricSpec}
}

// GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
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
