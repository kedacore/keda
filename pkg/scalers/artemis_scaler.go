package scalers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
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
	TriggerIndex          int
	ManagementEndpoint    string `keda:"name=managementEndpoint,    order=triggerMetadata, optional"`
	QueueName             string `keda:"name=queueName,             order=triggerMetadata, optional"`
	BrokerName            string `keda:"name=brokerName,            order=triggerMetadata, optional"`
	BrokerAddress         string `keda:"name=brokerAddress,         order=triggerMetadata, optional"`
	Username              string `keda:"name=username,              order=authParams;triggerMetadata;resolvedEnv"`
	Password              string `keda:"name=password,              order=authParams;triggerMetadata;resolvedEnv"`
	RestAPITemplate       string `keda:"name=restApiTemplate,       order=triggerMetadata, optional"`
	QueueLength           int64  `keda:"name=queueLength,           order=triggerMetadata, default=10"`
	ActivationQueueLength int64  `keda:"name=activationQueueLength, order=triggerMetadata, default=10"`
	CorsHeader            string `keda:"name=corsHeader,            order=triggerMetadata, optional"`
	UnsafeSsl             bool   `keda:"name=unsafeSsl,             order=triggerMetadata, default=false"`
}

//revive:enable:var-naming

type artemisMonitoring struct {
	MsgCount  int   `json:"value"`
	Status    int   `json:"status"`
	Timestamp int64 `json:"timestamp"`
}

const (
	artemisMetricType      = "External"
	defaultRestAPITemplate = "http://<<managementEndpoint>>/console/jolokia/read/org.apache.activemq.artemis:broker=\"<<brokerName>>\",component=addresses,address=\"<<brokerAddress>>\",subcomponent=queues,routing-type=\"anycast\",queue=\"<<queueName>>\"/MessageCount"
	defaultCorsHeader      = "http://%s"
)

func (a *artemisMetadata) Validate() error {
	if a.RestAPITemplate != "" {
		var err error
		if *a, err = getAPIParameters(*a); err != nil {
			return fmt.Errorf("can't parse restApiTemplate : %s ", err)
		}
	} else {
		a.RestAPITemplate = defaultRestAPITemplate
		if a.ManagementEndpoint == "" {
			return errors.New("no management endpoint given")
		}
		if a.QueueName == "" {
			return errors.New("no queue name given")
		}
		if a.BrokerName == "" {
			return errors.New("no broker name given")
		}
		if a.BrokerAddress == "" {
			return errors.New("no broker address given")
		}
	}
	if a.CorsHeader == "" {
		a.CorsHeader = fmt.Sprintf(defaultCorsHeader, a.ManagementEndpoint)
	}
	return nil
}

// NewArtemisQueueScaler creates a new artemis queue Scaler
func NewArtemisQueueScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	artemisMetadata, err := parseArtemisMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing artemis metadata: %w", err)
	}

	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, artemisMetadata.UnsafeSsl)

	return &artemisScaler{
		metricType: metricType,
		metadata:   artemisMetadata,
		httpClient: httpClient,
		logger:     InitializeLogger(config, "artemis_queue_scaler"),
	}, nil
}

func parseArtemisMetadata(config *scalersconfig.ScalerConfig) (*artemisMetadata, error) {
	meta := &artemisMetadata{}
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing prometheus metadata: %w", err)
	}

	meta.TriggerIndex = config.TriggerIndex

	return meta, nil
}

// getAPIParameters parse restAPITemplate to provide managementEndpoint , brokerName, brokerAddress, queueName
func getAPIParameters(meta artemisMetadata) (artemisMetadata, error) {
	u, err := url.ParseRequestURI(meta.RestAPITemplate)
	if err != nil {
		return meta, fmt.Errorf("unable to parse the artemis restAPITemplate: %w", err)
	}
	meta.ManagementEndpoint = u.Host
	splitURL := strings.Split(strings.Split(u.RawPath, ":")[1], "/")[0] // This returns : broker="<<brokerName>>",component=addresses,address="<<brokerAddress>>",subcomponent=queues,routing-type="anycast",queue="<<queueName>>"
	replacer := strings.NewReplacer(",", "&", "\"\"", "")
	v, err := url.ParseQuery(replacer.Replace(splitURL)) // This returns a map with key: string types and element type [] string. : map[address:["<<brokerAddress>>"] broker:["<<brokerName>>"] component:[addresses] queue:["<<queueName>>"] routing-type:["anycast"] subcomponent:[queues]]
	if err != nil {
		return meta, fmt.Errorf("unable to parse the artemis restAPITemplate: %w", err)
	}

	if len(v["address"][0]) == 0 {
		return meta, errors.New("no brokerAddress given")
	}
	meta.BrokerAddress = v["address"][0]

	if len(v["queue"][0]) == 0 {
		return meta, errors.New("no queueName is given")
	}
	meta.QueueName = v["queue"][0]

	if len(v["broker"][0]) == 0 {
		return meta, fmt.Errorf("no brokerName given: %s", meta.RestAPITemplate)
	}
	meta.BrokerName = v["broker"][0]

	return meta, nil
}

func (s *artemisScaler) getMonitoringEndpoint() string {
	replacer := strings.NewReplacer("<<managementEndpoint>>", s.metadata.ManagementEndpoint,
		"<<queueName>>", s.metadata.QueueName,
		"<<brokerName>>", s.metadata.BrokerName,
		"<<brokerAddress>>", s.metadata.BrokerAddress)

	monitoringEndpoint := replacer.Replace(s.metadata.RestAPITemplate)

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
	req.SetBasicAuth(s.metadata.Username, s.metadata.Password)
	req.Header.Set("Origin", s.metadata.CorsHeader)

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

	s.logger.V(1).Info(fmt.Sprintf("Artemis scaler: Providing metrics based on current queue length %d queue length limit %d", messageCount, s.metadata.QueueLength))

	return messageCount, nil
}

func (s *artemisScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.TriggerIndex, kedautil.NormalizeString(fmt.Sprintf("artemis-%s", s.metadata.QueueName))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.QueueLength),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: artemisMetricType}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *artemisScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	messages, err := s.getQueueMessageCount(ctx)

	if err != nil {
		s.logger.Error(err, "Unable to access the artemis management endpoint", "managementEndpoint", s.metadata.ManagementEndpoint)
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, float64(messages))

	return []external_metrics.ExternalMetricValue{metric}, messages > s.metadata.ActivationQueueLength, nil
}

func (s *artemisScaler) Close(context.Context) error {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}
