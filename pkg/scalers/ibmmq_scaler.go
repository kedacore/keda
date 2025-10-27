package scalers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type ibmmqScaler struct {
	metricType v2.MetricTargetType
	metadata   ibmmqMetadata
	httpClient *http.Client
	logger     logr.Logger
}

type ibmmqMetadata struct {
	Host                 string   `keda:"name=host,                 order=triggerMetadata"`
	QueueName            []string `keda:"name=queueName;queueNames, order=triggerMetadata"`
	QueueDepth           int64    `keda:"name=queueDepth,           order=triggerMetadata, default=20"`
	ActivationQueueDepth int64    `keda:"name=activationQueueDepth, order=triggerMetadata, default=0"`
	Operation            string   `keda:"name=operation,            order=triggerMetadata, enum=max;avg;sum, default=max"`
	Username             string   `keda:"name=username,             order=authParams;resolvedEnv;triggerMetadata"`
	Password             string   `keda:"name=password,             order=authParams;resolvedEnv;triggerMetadata"`
	UnsafeSsl            bool     `keda:"name=unsafeSsl,            order=triggerMetadata, default=false"`
	TLS                  bool     `keda:"name=tls,                  order=triggerMetadata, default=false, deprecated=The 'tls' setting is DEPRECATED and is removed in v2.18 - Use 'unsafeSsl' instead"`
	CA                   string   `keda:"name=ca,                   order=authParams, optional"`
	Cert                 string   `keda:"name=cert,                 order=authParams, optional"`
	Key                  string   `keda:"name=key,                  order=authParams, optional"`
	KeyPassword          string   `keda:"name=keyPassword,          order=authParams, optional"`

	triggerIndex int
}

// CommandResponse Full structured response from MQ admin REST query
type CommandResponse struct {
	CommandResponse []Response `json:"commandResponse"`
}

// Response The body of the response returned from the MQ admin query
type Response struct {
	Parameters *Parameters `json:"parameters"`
	Message    []string    `json:"message"`
}

// ErrorResponse Structure for error messages from IBM MQ
type ErrorResponse struct {
	Error []struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Parameters Contains the current depth of the IBM MQ Queue
type Parameters struct {
	Curdepth int `json:"curdepth"`
}

func (m *ibmmqMetadata) Validate() error {
	_, err := url.ParseRequestURI(m.Host)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if (m.Cert == "") != (m.Key == "") {
		return fmt.Errorf("both cert and key must be provided when using TLS")
	}

	return nil
}

func NewIBMMQScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "ibm_mq_scaler")

	meta, err := parseIBMMQMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing IBM MQ metadata: %w", err)
	}

	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, meta.UnsafeSsl)

	if meta.Cert != "" && meta.Key != "" {
		tlsConfig, err := kedautil.NewTLSConfigWithPassword(meta.Cert, meta.Key, meta.KeyPassword, meta.CA, meta.UnsafeSsl)
		if err != nil {
			return nil, err
		}
		httpClient.Transport = kedautil.CreateHTTPTransportWithTLSConfig(tlsConfig)
	}

	scaler := &ibmmqScaler{
		metricType: metricType,
		metadata:   meta,
		httpClient: httpClient,
		logger:     logger,
	}

	return scaler, nil
}

func (s *ibmmqScaler) Close(context.Context) error {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}

func parseIBMMQMetadata(config *scalersconfig.ScalerConfig) (ibmmqMetadata, error) {
	meta := ibmmqMetadata{triggerIndex: config.TriggerIndex}
	if err := config.TypedConfig(&meta); err != nil {
		return meta, err
	}
	return meta, nil
}

func (s *ibmmqScaler) getQueueDepthViaHTTP(ctx context.Context) (int64, error) {
	depths := make([]int64, 0, len(s.metadata.QueueName))

	for _, queueName := range s.metadata.QueueName {
		requestJSON := []byte(fmt.Sprintf(`{"type": "runCommandJSON", "command": "display", "qualifier": "qlocal", "name": "%s", "responseParameters": ["CURDEPTH"]}`, queueName))

		req, err := http.NewRequestWithContext(ctx, "POST", s.metadata.Host, bytes.NewBuffer(requestJSON))
		if err != nil {
			return 0, fmt.Errorf("failed to create HTTP request for queue %s: %w", queueName, err)
		}

		req.Header.Set("ibm-mq-rest-csrf-token", "value")
		req.Header.Set("Content-Type", "application/json")
		req.SetBasicAuth(s.metadata.Username, s.metadata.Password)

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return 0, fmt.Errorf("failed to contact MQ via REST for queue %s: %w", queueName, err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			return 0, fmt.Errorf("failed to read body of request for queue %s: %w", queueName, err)
		}

		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return 0, fmt.Errorf("authentication failed: incorrect username or password")
		case http.StatusNotFound:
			var errorResponse ErrorResponse
			if err := json.Unmarshal(body, &errorResponse); err != nil {
				return 0, fmt.Errorf("failed to parse error response JSON for queue %s: %w", queueName, err)
			}
			if len(errorResponse.Error) > 0 && errorResponse.Error[0].Message != "" {
				return 0, fmt.Errorf("%s", errorResponse.Error[0].Message)
			}
			return 0, fmt.Errorf("failed to get the current queue depth parameter for queue %s", queueName)
		case http.StatusOK:
			var response CommandResponse
			if err := json.Unmarshal(body, &response); err != nil {
				return 0, fmt.Errorf("failed to parse JSON for queue %s: %w", queueName, err)
			}

			// Check for valid response with message
			if len(response.CommandResponse) > 0 && len(response.CommandResponse[0].Message) > 0 {
				return 0, fmt.Errorf("%s", response.CommandResponse[0].Message[0])
			}

			// Check for valid response with parameters
			if len(response.CommandResponse) == 0 || response.CommandResponse[0].Parameters == nil {
				return 0, fmt.Errorf("failed to get the current queue depth parameter for queue %s", queueName)
			}

			depths = append(depths, int64(response.CommandResponse[0].Parameters.Curdepth))
		default:
			return 0, fmt.Errorf("unexpected status code %d for queue %s", resp.StatusCode, queueName)
		}
	}

	return calculateDepth(depths, s.metadata.Operation), nil
}

func calculateDepth(depths []int64, operation string) int64 {
	if len(depths) == 0 {
		return 0
	}

	switch operation {
	case sumOperation:
		return sumDepths(depths)
	case avgOperation:
		return avgDepths(depths)
	case maxOperation:
		return slices.Max(depths)
	default:
		return 0
	}
}

func sumDepths(depths []int64) int64 {
	var sum int64
	for _, depth := range depths {
		sum += depth
	}
	return sum
}

func avgDepths(depths []int64) int64 {
	if len(depths) == 0 {
		return 0
	}
	return sumDepths(depths) / int64(len(depths))
}

func (s *ibmmqScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	metricName := kedautil.NormalizeString(fmt.Sprintf("ibmmq-%s", s.metadata.QueueName[0]))
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, metricName),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.QueueDepth),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

func (s *ibmmqScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	queueDepth, err := s.getQueueDepthViaHTTP(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error inspecting IBM MQ queue depth: %w", err)
	}

	metric := GenerateMetricInMili(metricName, float64(queueDepth))

	return []external_metrics.ExternalMetricValue{metric}, queueDepth > s.metadata.ActivationQueueDepth, nil
}
