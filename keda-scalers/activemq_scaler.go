package scalers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const defaultActiveMQRestAPITemplate = "http://{{.ManagementEndpoint}}/api/jolokia/read/org.apache.activemq:type=Broker,brokerName={{.BrokerName}},destinationType=Queue,destinationName={{.DestinationName}}/QueueSize"

type activeMQScaler struct {
	metricType v2.MetricTargetType
	metadata   *activeMQMetadata
	httpClient *http.Client
	logger     logr.Logger
}

type activeMQMetadata struct {
	metricName   string
	triggerIndex int

	ManagementEndpoint string `keda:"name=managementEndpoint, order=triggerMetadata, optional"`
	DestinationName    string `keda:"name=destinationName,    order=triggerMetadata, optional"`
	BrokerName         string `keda:"name=brokerName,         order=triggerMetadata, optional"`

	// auth
	Username string `keda:"name=username, order=authParams;resolvedEnv;triggerMetadata"`
	Password string `keda:"name=password, order=authParams;resolvedEnv;triggerMetadata"`

	CorsHeader string `keda:"name=corsHeader, order=triggerMetadata, optional"`

	RestAPITemplate           string `keda:"name=restAPITemplate,           order=triggerMetadata, optional"`
	TargetQueueSize           int64  `keda:"name=targetQueueSize,           order=triggerMetadata, default=10"`
	ActivationTargetQueueSize int64  `keda:"name=activationTargetQueueSize, order=triggerMetadata, default=0"`
}

func (a *activeMQMetadata) Validate() error {
	if a.RestAPITemplate != "" {
		// parse restAPITemplate to provide managementEndpoint, brokerName, destinationName
		u, err := url.ParseRequestURI(a.RestAPITemplate)
		if err != nil {
			return fmt.Errorf("unable to parse ActiveMQ restAPITemplate: %w", err)
		}
		a.ManagementEndpoint = u.Host
		// This returns : type=Broker,brokerName=<<brokerName>>,destinationType=Queue,destinationName=<<destinationName>>
		splitURL := strings.Split(strings.Split(u.Path, ":")[1], "/")[0]
		replacer := strings.NewReplacer(",", "&")
		// This returns a map with key: string types and element type [] string. : map[brokerName:[<<brokerName>>] destinationName:[<<destinationName>>] destinationType:[Queue] type:[Broker]]
		v, err := url.ParseQuery(replacer.Replace(splitURL))
		if err != nil {
			return fmt.Errorf("unable to parse ActiveMQ restAPITemplate: %w", err)
		}
		if len(v["destinationName"][0]) == 0 {
			return fmt.Errorf("no destinationName is given")
		}
		a.DestinationName = v["destinationName"][0]
		if len(v["brokerName"][0]) == 0 {
			return fmt.Errorf("no brokerName given: %s", a.RestAPITemplate)
		}
		a.BrokerName = v["brokerName"][0]
	} else {
		a.RestAPITemplate = defaultActiveMQRestAPITemplate
		if a.ManagementEndpoint == "" {
			return fmt.Errorf("no management endpoint given")
		}
		if a.DestinationName == "" {
			return fmt.Errorf("no destination name given")
		}
		if a.BrokerName == "" {
			return fmt.Errorf("no broker name given")
		}
	}
	if a.CorsHeader == "" {
		a.CorsHeader = fmt.Sprintf(defaultCorsHeader, a.ManagementEndpoint)
	}
	a.metricName = GenerateMetricNameWithIndex(a.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("activemq-%s", a.DestinationName)))
	return nil
}

type activeMQMonitoring struct {
	MsgCount  int   `json:"value"`
	Status    int   `json:"status"`
	Timestamp int64 `json:"timestamp"`
}

// NewActiveMQScaler creates a new activeMQ Scaler
func NewActiveMQScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, err := parseActiveMQMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing ActiveMQ metadata: %w", err)
	}
	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, false)

	return &activeMQScaler{
		metricType: metricType,
		metadata:   meta,
		httpClient: httpClient,
		logger:     InitializeLogger(config, "active_mq_scaler"),
	}, nil
}

func parseActiveMQMetadata(config *scalersconfig.ScalerConfig) (*activeMQMetadata, error) {
	meta := &activeMQMetadata{}
	meta.triggerIndex = config.TriggerIndex
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing prometheus metadata: %w", err)
	}
	return meta, nil
}

func (s *activeMQScaler) getMonitoringEndpoint() (string, error) {
	var buf bytes.Buffer
	endpoint := map[string]string{
		"ManagementEndpoint": s.metadata.ManagementEndpoint,
		"BrokerName":         s.metadata.BrokerName,
		"DestinationName":    s.metadata.DestinationName,
	}
	template, err := template.New("monitoring_endpoint").Parse(s.metadata.RestAPITemplate)
	if err != nil {
		return "", fmt.Errorf("error parsing template: %w", err)
	}
	err = template.Execute(&buf, endpoint)
	if err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}
	monitoringEndpoint := buf.String()
	return monitoringEndpoint, nil
}

func (s *activeMQScaler) getQueueMessageCount(ctx context.Context) (int64, error) {
	var monitoringInfo *activeMQMonitoring
	var queueMessageCount int64

	client := s.httpClient
	url, err := s.getMonitoringEndpoint()
	if err != nil {
		return -1, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return -1, err
	}

	// Add HTTP Auth and Headers
	req.SetBasicAuth(s.metadata.Username, s.metadata.Password)
	req.Header.Set("Content-Type", "application/json")
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
		queueMessageCount = int64(monitoringInfo.MsgCount)
	} else {
		return -1, fmt.Errorf("ActiveMQ management endpoint response error code : %d %d", resp.StatusCode, monitoringInfo.Status)
	}

	s.logger.V(1).Info(fmt.Sprintf("ActiveMQ scaler: Providing metrics based on current queue size %d queue size limit %d", queueMessageCount, s.metadata.TargetQueueSize))

	return queueMessageCount, nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *activeMQScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: s.metadata.metricName,
		},
		Target: GetMetricTarget(s.metricType, s.metadata.TargetQueueSize),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

func (s *activeMQScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	queueSize, err := s.getQueueMessageCount(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("error inspecting ActiveMQ queue size: %w", err)
	}

	metric := GenerateMetricInMili(metricName, float64(queueSize))

	return []external_metrics.ExternalMetricValue{metric}, queueSize > s.metadata.ActivationTargetQueueSize, nil
}

func (s *activeMQScaler) Close(context.Context) error {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}
