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

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type artemisScaler struct {
	metricType v2.MetricTargetType
	metadata   *artemisMetadata
	httpClient *http.Client
	logger     logr.Logger
}

//revive:disable:var-naming breaking change on restApiTemplate, wouldn't bring any benefit to users
type artemisMetadata struct {
	managementEndpoint    string
	queueName             string
	brokerName            string
	brokerAddress         string
	username              string
	password              string
	restAPITemplate       string
	queueLength           int64
	activationQueueLength int64
	corsHeader            string
	scalerIndex           int
}

//revive:enable:var-naming

type artemisMonitoring struct {
	MsgCount  int   `json:"value"`
	Status    int   `json:"status"`
	Timestamp int64 `json:"timestamp"`
}

const (
	artemisMetricType                   = "External"
	defaultArtemisQueueLength           = 10
	defaultArtemisActivationQueueLength = 0
	defaultRestAPITemplate              = "http://<<managementEndpoint>>/console/jolokia/read/org.apache.activemq.artemis:broker=\"<<brokerName>>\",component=addresses,address=\"<<brokerAddress>>\",subcomponent=queues,routing-type=\"anycast\",queue=\"<<queueName>>\"/MessageCount"
	defaultCorsHeader                   = "http://%s"
)

// NewArtemisQueueScaler creates a new artemis queue Scaler
func NewArtemisQueueScaler(config *ScalerConfig) (Scaler, error) {
	// do we need to guarantee this timeout for a specific
	// reason? if not, we can have buildScaler pass in
	// the global client
	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, false)

	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	artemisMetadata, err := parseArtemisMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing artemis metadata: %w", err)
	}

	return &artemisScaler{
		metricType: metricType,
		metadata:   artemisMetadata,
		httpClient: httpClient,
		logger:     InitializeLogger(config, "artemis_queue_scaler"),
	}, nil
}

func parseArtemisMetadata(config *ScalerConfig) (*artemisMetadata, error) {
	meta := artemisMetadata{}

	meta.queueLength = defaultArtemisQueueLength
	meta.activationQueueLength = defaultArtemisActivationQueueLength

	if val, ok := config.TriggerMetadata["restApiTemplate"]; ok && val != "" {
		meta.restAPITemplate = config.TriggerMetadata["restApiTemplate"]
		var err error
		if meta, err = getAPIParameters(meta); err != nil {
			return nil, fmt.Errorf("can't parse restApiTemplate : %s ", err)
		}
	} else {
		meta.restAPITemplate = defaultRestAPITemplate
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
	}

	if val, ok := config.TriggerMetadata["corsHeader"]; ok && val != "" {
		meta.corsHeader = config.TriggerMetadata["corsHeader"]
	} else {
		meta.corsHeader = fmt.Sprintf(defaultCorsHeader, meta.managementEndpoint)
	}

	if val, ok := config.TriggerMetadata["queueLength"]; ok {
		queueLength, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("can't parse queueLength: %w", err)
		}

		meta.queueLength = queueLength
	}

	if val, ok := config.TriggerMetadata["activationQueueLength"]; ok {
		activationQueueLength, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("can't parse activationQueueLength: %w", err)
		}

		meta.activationQueueLength = activationQueueLength
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

	meta.scalerIndex = config.ScalerIndex

	return &meta, nil
}

// getAPIParameters parse restAPITemplate to provide managementEndpoint , brokerName, brokerAddress, queueName
func getAPIParameters(meta artemisMetadata) (artemisMetadata, error) {
	u, err := url.ParseRequestURI(meta.restAPITemplate)
	if err != nil {
		return meta, fmt.Errorf("unable to parse the artemis restAPITemplate: %w", err)
	}
	meta.managementEndpoint = u.Host
	splitURL := strings.Split(strings.Split(u.RawPath, ":")[1], "/")[0] // This returns : broker="<<brokerName>>",component=addresses,address="<<brokerAddress>>",subcomponent=queues,routing-type="anycast",queue="<<queueName>>"
	replacer := strings.NewReplacer(",", "&", "\"\"", "")
	v, err := url.ParseQuery(replacer.Replace(splitURL)) // This returns a map with key: string types and element type [] string. : map[address:["<<brokerAddress>>"] broker:["<<brokerName>>"] component:[addresses] queue:["<<queueName>>"] routing-type:["anycast"] subcomponent:[queues]]
	if err != nil {
		return meta, fmt.Errorf("unable to parse the artemis restAPITemplate: %w", err)
	}

	if len(v["address"][0]) == 0 {
		return meta, errors.New("no brokerAddress given")
	}
	meta.brokerAddress = v["address"][0]

	if len(v["queue"][0]) == 0 {
		return meta, errors.New("no queueName is given")
	}
	meta.queueName = v["queue"][0]

	if len(v["broker"][0]) == 0 {
		return meta, fmt.Errorf("no brokerName given: %s", meta.restAPITemplate)
	}
	meta.brokerName = v["broker"][0]

	return meta, nil
}

func (s *artemisScaler) getMonitoringEndpoint() string {
	replacer := strings.NewReplacer("<<managementEndpoint>>", s.metadata.managementEndpoint,
		"<<queueName>>", s.metadata.queueName,
		"<<brokerName>>", s.metadata.brokerName,
		"<<brokerAddress>>", s.metadata.brokerAddress)

	monitoringEndpoint := replacer.Replace(s.metadata.restAPITemplate)

	return monitoringEndpoint
}

func (s *artemisScaler) getQueueMessageCount(ctx context.Context) (int64, error) {
	var monitoringInfo *artemisMonitoring
	var messageCount int64

	client := s.httpClient
	url := s.getMonitoringEndpoint()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return -1, err
	}
	req.SetBasicAuth(s.metadata.username, s.metadata.password)
	req.Header.Set("Origin", s.metadata.corsHeader)

	resp, err := client.Do(req)
	if err != nil {
		return -1, err
	}

	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&monitoringInfo); err != nil {
		return -1, err
	}
	if resp.StatusCode == 200 && monitoringInfo.Status == 200 {
		messageCount = int64(monitoringInfo.MsgCount)
	} else {
		return -1, fmt.Errorf("artemis management endpoint response error code : %d %d", resp.StatusCode, monitoringInfo.Status)
	}

	s.logger.V(1).Info(fmt.Sprintf("Artemis scaler: Providing metrics based on current queue length %d queue length limit %d", messageCount, s.metadata.queueLength))

	return messageCount, nil
}

func (s *artemisScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, kedautil.NormalizeString(fmt.Sprintf("artemis-%s", s.metadata.queueName))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.queueLength),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: artemisMetricType}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *artemisScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	messages, err := s.getQueueMessageCount(ctx)

	if err != nil {
		s.logger.Error(err, "Unable to access the artemis management endpoint", "managementEndpoint", s.metadata.managementEndpoint)
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, float64(messages))

	return []external_metrics.ExternalMetricValue{metric}, messages > s.metadata.activationQueueLength, nil
}

func (s *artemisScaler) Close(context.Context) error {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}
