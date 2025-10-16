package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	url_pkg "net/url"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type graphiteScaler struct {
	metricType v2.MetricTargetType
	metadata   *graphiteMetadata
	httpClient *http.Client
	logger     logr.Logger
}

type graphiteMetadata struct {
	ServerAddress       string  `keda:"name=serverAddress,       order=triggerMetadata"`
	Query               string  `keda:"name=query,               order=triggerMetadata"`
	Threshold           float64 `keda:"name=threshold,           order=triggerMetadata, default=100"`
	ActivationThreshold float64 `keda:"name=activationThreshold, order=triggerMetadata, default=0"`
	QueryTime           string  `keda:"name=queryTime,           order=triggerMetadata"`

	// basic auth
	AuthMode string `keda:"name=authMode,        order=triggerMetadata, optional"`
	Username string `keda:"name=username,        order=authParams,      optional"`
	Password string `keda:"name=password,        order=authParams,      optional"`

	metricName      string
	enableBasicAuth bool
	triggerIndex    int
}

func (g *graphiteMetadata) Validate() error {
	if g.AuthMode != "" && g.AuthMode != "basic" {
		return fmt.Errorf("authMode must be 'basic'")
	}
	if g.AuthMode == "basic" && g.Username == "" {
		return fmt.Errorf("username is required when authMode is 'basic'")
	}

	return nil
}

type grapQueryResult []struct {
	Target     string                 `json:"target"`
	Tags       map[string]interface{} `json:"tags"`
	Datapoints [][]*float64           `json:"datapoints,omitempty"`
}

// NewGraphiteScaler creates a new graphiteScaler
func NewGraphiteScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, err := parseGraphiteMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing graphite metadata: %w", err)
	}

	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, false)

	return &graphiteScaler{
		metricType: metricType,
		metadata:   meta,
		httpClient: httpClient,
		logger:     InitializeLogger(config, "graphite_scaler"),
	}, nil
}

func parseGraphiteMetadata(config *scalersconfig.ScalerConfig) (*graphiteMetadata, error) {
	meta := &graphiteMetadata{triggerIndex: config.TriggerIndex}
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing graphite metadata: %w", err)
	}

	meta.enableBasicAuth = true

	return meta, nil
}

func (s *graphiteScaler) Close(context.Context) error {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}

func (s *graphiteScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, "graphite"),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.Threshold),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

func (s *graphiteScaler) executeGrapQuery(ctx context.Context) (float64, error) {
	queryEscaped := url_pkg.QueryEscape(s.metadata.Query)
	url := fmt.Sprintf("%s/render?from=%s&target=%s&format=json", s.metadata.ServerAddress, s.metadata.QueryTime, queryEscaped)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return -1, err
	}
	if s.metadata.enableBasicAuth {
		req.SetBasicAuth(s.metadata.Username, s.metadata.Password)
	}
	r, err := s.httpClient.Do(req)
	if err != nil {
		return -1, err
	}

	b, err := io.ReadAll(r.Body)
	if err != nil {
		return -1, err
	}
	r.Body.Close()

	var result grapQueryResult
	err = json.Unmarshal(b, &result)
	if err != nil {
		return -1, err
	}

	if len(result) == 0 {
		return 0, nil
	} else if len(result) > 1 {
		return -1, fmt.Errorf("graphite query %s returned multiple series", s.metadata.Query)
	}

	// https://graphite-api.readthedocs.io/en/latest/api.html#json
	if len(result[0].Datapoints) == 0 {
		return 0, nil
	}

	// Return the most recent non-null datapoint
	for i := len(result[0].Datapoints) - 1; i >= 0; i-- {
		if datapoint := result[0].Datapoints[i][0]; datapoint != nil {
			return *datapoint, nil
		}
	}

	return -1, fmt.Errorf("no valid non-null response in query %s, try increasing your queryTime or check your query", s.metadata.Query)
}

func (s *graphiteScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	val, err := s.executeGrapQuery(ctx)
	if err != nil {
		s.logger.Error(err, "error executing graphite query")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, val)

	return []external_metrics.ExternalMetricValue{metric}, val > s.metadata.ActivationThreshold, nil
}
