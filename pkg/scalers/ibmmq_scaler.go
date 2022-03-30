package scalers

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"

	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

// Default variables and settings
const (
	ibmMqQueueDepthMetricName = "currentQueueDepth"
	defaultTargetQueueDepth   = 20
	defaultTLSDisabled        = false
)

// IBMMQScaler assigns struct data pointer to metadata variable
type IBMMQScaler struct {
	metricType         v2beta2.MetricTargetType
	metadata           *IBMMQMetadata
	defaultHTTPTimeout time.Duration
}

// IBMMQMetadata Metadata used by KEDA to query IBM MQ queue depth and scale
type IBMMQMetadata struct {
	host             string
	queueManager     string
	queueName        string
	username         string
	password         string
	targetQueueDepth int64
	tlsDisabled      bool
	scalerIndex      int
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
func NewIBMMQScaler(config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %s", err)
	}

	meta, err := parseIBMMQMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing IBM MQ metadata: %s", err)
	}

	return &IBMMQScaler{
		metricType:         metricType,
		metadata:           meta,
		defaultHTTPTimeout: config.GlobalHTTPTimeout,
	}, nil
}

// Close closes and returns nil
func (s *IBMMQScaler) Close(context.Context) error {
	return nil
}

// parseIBMMQMetadata checks the existence of and validates the MQ connection data provided
func parseIBMMQMetadata(config *ScalerConfig) (*IBMMQMetadata, error) {
	meta := IBMMQMetadata{}

	if val, ok := config.TriggerMetadata["host"]; ok {
		_, err := url.ParseRequestURI(val)
		if err != nil {
			return nil, fmt.Errorf("invalid URL: %s", err)
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
			return nil, fmt.Errorf("invalid targetQueueDepth - must be an integer")
		}
		meta.targetQueueDepth = queueDepth
	} else {
		fmt.Println("No target depth defined - setting default")
		meta.targetQueueDepth = defaultTargetQueueDepth
	}

	if val, ok := config.TriggerMetadata["tls"]; ok {
		tlsDisabled, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("invalid tls setting: %s", err)
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
	meta.scalerIndex = config.ScalerIndex
	return &meta, nil
}

// IsActive returns true if there are messages to be processed/if we need to scale from zero
func (s *IBMMQScaler) IsActive(ctx context.Context) (bool, error) {
	queueDepth, err := s.getQueueDepthViaHTTP(ctx)
	if err != nil {
		return false, fmt.Errorf("error inspecting IBM MQ queue depth: %s", err)
	}
	return queueDepth > 0, nil
}

// getQueueDepthViaHTTP returns the depth of the MQ Queue from the Admin endpoint
func (s *IBMMQScaler) getQueueDepthViaHTTP(ctx context.Context) (int64, error) {
	queue := s.metadata.queueName
	url := s.metadata.host

	var requestJSON = []byte(`{"type": "runCommandJSON", "command": "display", "qualifier": "qlocal", "name": "` + queue + `", "responseParameters" : ["CURDEPTH"]}`)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestJSON))
	if err != nil {
		return 0, fmt.Errorf("failed to request queue depth: %s", err)
	}
	req.Header.Set("ibm-mq-rest-csrf-token", "value")
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(s.metadata.username, s.metadata.password)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: s.metadata.tlsDisabled},
	}
	client := kedautil.CreateHTTPClient(s.defaultHTTPTimeout, false)
	client.Transport = tr

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to contact MQ via REST: %s", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to ready body of request: %s", err)
	}

	var response CommandResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return 0, fmt.Errorf("failed to parse JSON: %s", err)
	}

	if response.CommandResponse == nil || len(response.CommandResponse) == 0 {
		return 0, fmt.Errorf("failed to parse response from REST call: %s", err)
	}
	return int64(response.CommandResponse[0].Parameters.Curdepth), nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *IBMMQScaler) GetMetricSpecForScaling(context.Context) []v2beta2.MetricSpec {
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, kedautil.NormalizeString(fmt.Sprintf("ibmmq-%s", s.metadata.queueName))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.targetQueueDepth),
	}
	metricSpec := v2beta2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2beta2.MetricSpec{metricSpec}
}

// GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *IBMMQScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	queueDepth, err := s.getQueueDepthViaHTTP(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, fmt.Errorf("error inspecting IBM MQ queue depth: %s", err)
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: ibmMqQueueDepthMetricName,
		Value:      *resource.NewQuantity(queueDepth, resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}
