package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	url_pkg "net/url"
	"strconv"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	graphiteServerAddress              = "serverAddress"
	graphiteQuery                      = "query"
	graphiteThreshold                  = "threshold"
	graphiteActivationThreshold        = "activationThreshold"
	graphiteQueryTime                  = "queryTime"
	defaultGraphiteThreshold           = 100
	defaultGraphiteActivationThreshold = 0
)

type graphiteScaler struct {
	metricType v2.MetricTargetType
	metadata   *graphiteMetadata
	httpClient *http.Client
	logger     logr.Logger
}

type graphiteMetadata struct {
	serverAddress       string
	query               string
	threshold           float64
	activationThreshold float64
	from                string

	// basic auth
	enableBasicAuth bool
	username        string
	password        string // +optional
	triggerIndex    int
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
	meta := graphiteMetadata{}

	if val, ok := config.TriggerMetadata[graphiteServerAddress]; ok && val != "" {
		meta.serverAddress = val
	} else {
		return nil, fmt.Errorf("no %s given", graphiteServerAddress)
	}

	if val, ok := config.TriggerMetadata[graphiteQuery]; ok && val != "" {
		meta.query = val
	} else {
		return nil, fmt.Errorf("no %s given", graphiteQuery)
	}

	if val, ok := config.TriggerMetadata[graphiteQueryTime]; ok && val != "" {
		meta.from = val
	} else {
		return nil, fmt.Errorf("no %s given", graphiteQueryTime)
	}

	meta.threshold = defaultGraphiteThreshold
	if val, ok := config.TriggerMetadata[graphiteThreshold]; ok && val != "" {
		t, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing %s: %w", graphiteThreshold, err)
		}

		meta.threshold = t
	}

	meta.activationThreshold = defaultGraphiteActivationThreshold
	if val, ok := config.TriggerMetadata[graphiteActivationThreshold]; ok {
		t, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("activationTargetValue parsing error %w", err)
		}

		meta.activationThreshold = t
	}

	meta.triggerIndex = config.TriggerIndex

	val, ok := config.TriggerMetadata["authMode"]
	// no authMode specified
	if !ok {
		return &meta, nil
	}
	if val != "basic" {
		return nil, fmt.Errorf("authMode must be 'basic'")
	}

	if len(config.AuthParams["username"]) == 0 {
		return nil, fmt.Errorf("no username given")
	}

	meta.username = config.AuthParams["username"]
	// password is optional. For convenience, many application implement basic auth with
	// username as apikey and password as empty
	meta.password = config.AuthParams["password"]
	meta.enableBasicAuth = true

	return &meta, nil
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
		Target: GetMetricTargetMili(s.metricType, s.metadata.threshold),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

func (s *graphiteScaler) executeGrapQuery(ctx context.Context) (float64, error) {
	queryEscaped := url_pkg.QueryEscape(s.metadata.query)
	url := fmt.Sprintf("%s/render?from=%s&target=%s&format=json", s.metadata.serverAddress, s.metadata.from, queryEscaped)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return -1, err
	}
	if s.metadata.enableBasicAuth {
		req.SetBasicAuth(s.metadata.username, s.metadata.password)
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
		return -1, fmt.Errorf("graphite query %s returned multiple series", s.metadata.query)
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

	return -1, fmt.Errorf("no valid non-null response in query %s, try increasing your queryTime or check your query", s.metadata.query)
}

func (s *graphiteScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	val, err := s.executeGrapQuery(ctx)
	if err != nil {
		s.logger.Error(err, "error executing graphite query")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, val)

	return []external_metrics.ExternalMetricValue{metric}, val > s.metadata.activationThreshold, nil
}
