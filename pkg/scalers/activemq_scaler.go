package scalers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type activeMQScaler struct {
	metricType v2.MetricTargetType
	metadata   *activeMQMetadata
	httpClient *http.Client
	logger     logr.Logger
}

type activeMQMetadata struct {
	managementEndpoint        string
	destinationName           string
	brokerName                string
	username                  string
	password                  string
	restAPITemplate           string
	targetQueueSize           int64
	activationTargetQueueSize int64
	corsHeader                string
	metricName                string
	scalerIndex               int
}

type activeMQMonitoring struct {
	MsgCount  int   `json:"value"`
	Status    int   `json:"status"`
	Timestamp int64 `json:"timestamp"`
}

const (
	defaultTargetQueueSize           = 10
	defaultActivationTargetQueueSize = 0
	defaultActiveMQRestAPITemplate   = "http://{{.ManagementEndpoint}}/api/jolokia/read/org.apache.activemq:type=Broker,brokerName={{.BrokerName}},destinationType=Queue,destinationName={{.DestinationName}}/QueueSize"
)

// NewActiveMQScaler creates a new activeMQ Scaler
func NewActiveMQScaler(config *ScalerConfig) (Scaler, error) {
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

func parseActiveMQMetadata(config *ScalerConfig) (*activeMQMetadata, error) {
	meta := activeMQMetadata{}

	if val, ok := config.TriggerMetadata["restAPITemplate"]; ok && val != "" {
		meta.restAPITemplate = config.TriggerMetadata["restAPITemplate"]
		var err error
		if meta, err = getRestAPIParameters(meta); err != nil {
			return nil, fmt.Errorf("can't parse restAPITemplate : %s ", err)
		}
	} else {
		meta.restAPITemplate = defaultActiveMQRestAPITemplate
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

	if val, ok := config.TriggerMetadata["targetQueueSize"]; ok {
		queueSize, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid targetQueueSize - must be an integer")
		}

		meta.targetQueueSize = queueSize
	} else {
		meta.targetQueueSize = defaultTargetQueueSize
	}

	if val, ok := config.TriggerMetadata["activationTargetQueueSize"]; ok {
		activationTargetQueueSize, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid activationTargetQueueSize - must be an integer")
		}
		meta.activationTargetQueueSize = activationTargetQueueSize
	} else {
		meta.activationTargetQueueSize = defaultActivationTargetQueueSize
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

	if val, ok := config.TriggerMetadata["corsHeader"]; ok && val != "" {
		meta.corsHeader = config.TriggerMetadata["corsHeader"]
	} else {
		meta.corsHeader = fmt.Sprintf(defaultCorsHeader, meta.managementEndpoint)
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

	meta.metricName = GenerateMetricNameWithIndex(config.ScalerIndex, kedautil.NormalizeString(fmt.Sprintf("activemq-%s", meta.destinationName)))

	meta.scalerIndex = config.ScalerIndex

	return &meta, nil
}

// getRestAPIParameters parse restAPITemplate to provide managementEndpoint, brokerName, destinationName
func getRestAPIParameters(meta activeMQMetadata) (activeMQMetadata, error) {
	u, err := url.ParseRequestURI(meta.restAPITemplate)
	if err != nil {
		return meta, fmt.Errorf("unable to parse ActiveMQ restAPITemplate: %w", err)
	}

	meta.managementEndpoint = u.Host
	splitURL := strings.Split(strings.Split(u.Path, ":")[1], "/")[0] // This returns : type=Broker,brokerName=<<brokerName>>,destinationType=Queue,destinationName=<<destinationName>>
	replacer := strings.NewReplacer(",", "&")
	v, err := url.ParseQuery(replacer.Replace(splitURL)) // This returns a map with key: string types and element type [] string. : map[brokerName:[<<brokerName>>] destinationName:[<<destinationName>>] destinationType:[Queue] type:[Broker]]
	if err != nil {
		return meta, fmt.Errorf("unable to parse ActiveMQ restAPITemplate: %w", err)
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

func (s *activeMQScaler) getMonitoringEndpoint() (string, error) {
	var buf bytes.Buffer
	endpoint := map[string]string{
		"ManagementEndpoint": s.metadata.managementEndpoint,
		"BrokerName":         s.metadata.brokerName,
		"DestinationName":    s.metadata.destinationName,
	}
	template, err := template.New("monitoring_endpoint").Parse(s.metadata.restAPITemplate)
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
	req.SetBasicAuth(s.metadata.username, s.metadata.password)
	req.Header.Set("Content-Type", "application/json")
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
		queueMessageCount = int64(monitoringInfo.MsgCount)
	} else {
		return -1, fmt.Errorf("ActiveMQ management endpoint response error code : %d %d", resp.StatusCode, monitoringInfo.Status)
	}

	s.logger.V(1).Info(fmt.Sprintf("ActiveMQ scaler: Providing metrics based on current queue size %d queue size limit %d", queueMessageCount, s.metadata.targetQueueSize))

	return queueMessageCount, nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *activeMQScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: s.metadata.metricName,
		},
		Target: GetMetricTarget(s.metricType, s.metadata.targetQueueSize),
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

	return []external_metrics.ExternalMetricValue{metric}, queueSize > s.metadata.activationTargetQueueSize, nil
}

func (s *activeMQScaler) Close(context.Context) error {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}
