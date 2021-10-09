package scalers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type activeMQScaler struct {
	metadata   *activeMQMetadata
	httpClient *http.Client
}

type activeMQMetadata struct {
	managementEndpoint string
	destinationName    string
	brokerName         string
	username           string
	password           string
	restAPITemplate    string
	queueSize          int
	corsHeader         string
}

type activeMQMonitoring struct {
	MsgCount  int   `json:"value"`
	Status    int   `json:"status"`
	Timestamp int64 `json:"timestamp"`
}

const (
	activeMQMetricType             = "External"
	defaultActiveMQQueueSize       = 10
	defaultActiveMQrestAPITemplate = "http://<<managementEndpoint>>/api/jolokia/read/org.apache.activemq:type=Broker,brokerName=<<brokerName>>,destinationType=Queue,destinationName=<<destinationName>>/QueueSize"
	defaultActiveMQCorsHeader      = "http://%s"
)

var activeMQLog = logf.Log.WithName("activeMQ_scaler")

// NewActiveMQScaler creates a new activeMQ Scaler
func NewActiveMQScaler(config *ScalerConfig) (Scaler, error) {
	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout)

	activeMQMetadata, err := parseActiveMQMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing activeMQ metadata: %s", err)
	}

	return &activeMQScaler{
		metadata:   activeMQMetadata,
		httpClient: httpClient,
	}, nil
}

func parseActiveMQMetadata(config *ScalerConfig) (*activeMQMetadata, error) {
	meta := activeMQMetadata{}

	meta.queueSize = defaultActiveMQQueueSize

	if val, ok := config.TriggerMetadata["restAPITemplate"]; ok && val != "" {
		meta.restAPITemplate = config.TriggerMetadata["restAPITemplate"]
		var err error
		if meta, err = getRestAPIParameters(meta); err != nil {
			return nil, fmt.Errorf("can't parse restAPITemplate : %s ", err)
		}
	} else {
		meta.restAPITemplate = defaultActiveMQrestAPITemplate
		if config.TriggerMetadata["managementEndpoint"] == "" {
			return nil, errors.New("no management endpoint given")
		}
		meta.managementEndpoint = config.TriggerMetadata["managementEndpoint"]

		if config.TriggerMetadata["destinationName"] == "" {
			return nil, errors.New("no destination name given")
		}
		meta.destinationName = config.TriggerMetadata["destinationName"]

		if config.TriggerMetadata["brokerName"] == "" {
			return nil, errors.New("no broker name given")
		}
		meta.brokerName = config.TriggerMetadata["brokerName"]
	}
	if val, ok := config.TriggerMetadata["corsHeader"]; ok && val != "" {
		meta.corsHeader = config.TriggerMetadata["corsHeader"]
	} else {
		meta.corsHeader = fmt.Sprintf(defaultActiveMQCorsHeader, meta.managementEndpoint)
	}

	if val, ok := config.TriggerMetadata["queueSize"]; ok {
		queueSize, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("can't parse queueSize: %s", err)
		}

		meta.queueSize = queueSize
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

func (s *activeMQScaler) IsActive(ctx context.Context) (bool, error) {
	messages, err := s.getQueueMessageCount()
	if err != nil {
		activeMQLog.Error(err, "Unable to access the activeMQ management endpoint", "managementEndpoint", s.metadata.managementEndpoint)
		return false, err
	}

	return messages > 0, nil
}

// getRestAPIParameters parse restAPITemplate to provide managementEndpoint, brokerName, destinationName
func getRestAPIParameters(meta activeMQMetadata) (activeMQMetadata, error) {
	u, err := url.ParseRequestURI(meta.restAPITemplate)
	if err != nil {
		return meta, fmt.Errorf("unable to parse the activeMQ restAPITemplate: %s", err)
	}

	meta.managementEndpoint = u.Host
	splitURL := strings.Split(strings.Split(u.Path, ":")[1], "/")[0] // This returns : type=Broker,brokerName=<<brokerName>>,destinationType=Queue,destinationName=<<destinationName>>
	replacer := strings.NewReplacer(",", "&")
	v, err := url.ParseQuery(replacer.Replace(splitURL)) // This returns a map with key: string types and element type [] string. : map[brokerName:[<<brokerName>>] destinationName:[<<destinationName>>] destinationType:[Queue] type:[Broker]]
	if err != nil {
		return meta, fmt.Errorf("unable to parse the activeMQ restAPITemplate: %s", err)
	}

	if len(v["destinationName"][0]) == 0 {
		return meta, errors.New("no destinationName is given")
	}
	meta.destinationName = v["destinationName"][0]

	if len(v["brokerName"][0]) == 0 {
		return meta, fmt.Errorf("no brokerName given: %s", meta.restAPITemplate)
	}
	meta.brokerName = v["brokerName"][0]

	return meta, nil
}

func (s *activeMQScaler) getMonitoringEndpoint() string {
	replacer := strings.NewReplacer("<<managementEndpoint>>", s.metadata.managementEndpoint,
		"<<brokerName>>", s.metadata.brokerName,
		"<<destinationName>>", s.metadata.destinationName)

	monitoringEndpoint := replacer.Replace(s.metadata.restAPITemplate)

	return monitoringEndpoint
}

func (s *activeMQScaler) getQueueMessageCount() (int, error) {
	var monitoringInfo *activeMQMonitoring
	var queueMessageCount int

	client := s.httpClient
	url := s.getMonitoringEndpoint()

	req, err := http.NewRequest("GET", url, nil)

	req.SetBasicAuth(s.metadata.username, s.metadata.password)
	req.Header.Set("Origin", s.metadata.corsHeader)

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
		queueMessageCount = monitoringInfo.MsgCount
	} else {
		return -1, fmt.Errorf("activeMQ management endpoint response error code : %d %d", resp.StatusCode, monitoringInfo.Status)
	}

	activeMQLog.V(1).Info(fmt.Sprintf("ActiveMQ scaler: Providing metrics based on current queue size %d queue size limit %d", queueMessageCount, s.metadata.queueSize))

	return queueMessageCount, nil
}

func (s *activeMQScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	targetMetricValue := resource.NewQuantity(int64(s.metadata.queueSize), resource.DecimalSI)
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: kedautil.NormalizeString(fmt.Sprintf("%s-%s-%s", "activeMQ", s.metadata.brokerName, s.metadata.destinationName)),
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: targetMetricValue,
		},
	}
	metricSpec := v2beta2.MetricSpec{External: externalMetric, Type: activeMQMetricType}
	return []v2beta2.MetricSpec{metricSpec}
}

func (s *activeMQScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	messages, err := s.getQueueMessageCount()

	if err != nil {
		activeMQLog.Error(err, "Unable to access the activeMQ management endpoint", "managementEndpoint", s.metadata.managementEndpoint)
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
func (s *activeMQScaler) Close() error {
	return nil
}
