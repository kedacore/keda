package scalers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

// Default variables and settings
const (
	defaultTargetQueueDepth = 20
	defaultTLSDisabled      = false
)

// IBMMQScaler assigns struct data pointer to metadata variable
type IBMMQScaler struct {
	metricType         v2.MetricTargetType
	metadata           *IBMMQMetadata
	defaultHTTPTimeout time.Duration
	logger             logr.Logger
}

// IBMMQMetadata Metadata used by KEDA to query IBM MQ queue depth and scale
type IBMMQMetadata struct {
	host                 string
	queueManager         string
	queueName            string
	username             string
	password             string
	queueDepth           int64
	activationQueueDepth int64
	tlsDisabled          bool
	triggerIndex         int
}

// CommandResponse Full structured response from MQ admin REST query
type CommandResponse struct {
	CommandResponse []Response `json:"commandResponse"`
}

// Response The body of the response returned from the MQ admin query
type Response struct {
	Parameters Parameters `json:"parameters"`
}

// Parameters Contains the current depth of the IBM MQ Queue
type Parameters struct {
	Curdepth int `json:"curdepth"`
}

// NewIBMMQScaler creates a new IBM MQ scaler
func NewIBMMQScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, err := parseIBMMQMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing IBM MQ metadata: %w", err)
	}

	return &IBMMQScaler{
		metricType:         metricType,
		metadata:           meta,
		defaultHTTPTimeout: config.GlobalHTTPTimeout,
		logger:             InitializeLogger(config, "ibm_mq_scaler"),
	}, nil
}

// Close closes and returns nil
func (s *IBMMQScaler) Close(context.Context) error {
	return nil
}

// parseIBMMQMetadata checks the existence of and validates the MQ connection data provided
func parseIBMMQMetadata(config *scalersconfig.ScalerConfig) (*IBMMQMetadata, error) {
	meta := IBMMQMetadata{}

	if val, ok := config.TriggerMetadata["host"]; ok {
		_, err := url.ParseRequestURI(val)
		if err != nil {
			return nil, fmt.Errorf("invalid URL: %w", err)
		}
		meta.host = val
	} else {
		return nil, fmt.Errorf("no host URI given")
	}

	if val, ok := config.TriggerMetadata["queueManager"]; ok {
		meta.queueManager = val
	} else {
		return nil, fmt.Errorf("no queue manager given")
	}

	if val, ok := config.TriggerMetadata["queueName"]; ok {
		meta.queueName = val
	} else {
		return nil, fmt.Errorf("no queue name given")
	}

	if val, ok := config.TriggerMetadata["queueDepth"]; ok && val != "" {
		queueDepth, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid queueDepth - must be an integer")
		}
		meta.queueDepth = queueDepth
	} else {
		fmt.Println("No target depth defined - setting default")
		meta.queueDepth = defaultTargetQueueDepth
	}

	meta.activationQueueDepth = 0
	if val, ok := config.TriggerMetadata["activationQueueDepth"]; ok && val != "" {
		activationQueueDepth, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid activationQueueDepth - must be an integer")
		}
		meta.activationQueueDepth = activationQueueDepth
	}

	if val, ok := config.TriggerMetadata["tls"]; ok {
		tlsDisabled, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("invalid tls setting: %w", err)
		}
		meta.tlsDisabled = tlsDisabled
	} else {
		fmt.Println("No tls setting defined - setting default")
		meta.tlsDisabled = defaultTLSDisabled
	}
	val, ok := config.AuthParams["username"]
	switch {
	case ok && val != "":
		meta.username = val
	case config.TriggerMetadata["usernameFromEnv"] != "":
		meta.username = config.ResolvedEnv[config.TriggerMetadata["usernameFromEnv"]]
	default:
		return nil, fmt.Errorf("no username given")
	}
	pwdValue, booleanValue := config.AuthParams["password"] // booleanValue reports whether the type assertion succeeded or not
	switch {
	case booleanValue && pwdValue != "":
		meta.password = pwdValue
	case config.TriggerMetadata["passwordFromEnv"] != "":
		meta.password = config.ResolvedEnv[config.TriggerMetadata["passwordFromEnv"]]
	default:
		return nil, fmt.Errorf("no password given")
	}
	meta.triggerIndex = config.TriggerIndex
	return &meta, nil
}

// getQueueDepthViaHTTP returns the depth of the MQ Queue from the Admin endpoint
func (s *IBMMQScaler) getQueueDepthViaHTTP(ctx context.Context) (int64, error) {
	queue := s.metadata.queueName
	url := s.metadata.host

	var requestJSON = []byte(`{"type": "runCommandJSON", "command": "display", "qualifier": "qlocal", "name": "` + queue + `", "responseParameters" : ["CURDEPTH"]}`)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestJSON))
	if err != nil {
		return 0, fmt.Errorf("failed to request queue depth: %w", err)
	}
	req.Header.Set("ibm-mq-rest-csrf-token", "value")
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(s.metadata.username, s.metadata.password)

	client := kedautil.CreateHTTPClient(s.defaultHTTPTimeout, s.metadata.tlsDisabled)

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to contact MQ via REST: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to ready body of request: %w", err)
	}

	var response CommandResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return 0, fmt.Errorf("failed to parse JSON: %w", err)
	}

	if response.CommandResponse == nil || len(response.CommandResponse) == 0 {
		return 0, fmt.Errorf("failed to parse response from REST call: %w", err)
	}
	return int64(response.CommandResponse[0].Parameters.Curdepth), nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *IBMMQScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("ibmmq-%s", s.metadata.queueName))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.queueDepth),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *IBMMQScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	queueDepth, err := s.getQueueDepthViaHTTP(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error inspecting IBM MQ queue depth: %w", err)
	}

	metric := GenerateMetricInMili(metricName, float64(queueDepth))

	return []external_metrics.ExternalMetricValue{metric}, queueDepth > s.metadata.activationQueueDepth, nil
}
