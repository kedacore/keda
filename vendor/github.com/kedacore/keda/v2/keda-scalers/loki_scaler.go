package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/keda-scalers/authentication"
	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	defaultIgnoreNullValues = true
	tenantNameHeaderKey     = "X-Scope-OrgID"
)

type lokiScaler struct {
	metricType v2.MetricTargetType
	metadata   lokiMetadata
	httpClient *http.Client
	logger     logr.Logger
}

type lokiMetadata struct {
	ServerAddress       string  `keda:"name=serverAddress,order=triggerMetadata"`
	Query               string  `keda:"name=query,order=triggerMetadata"`
	Threshold           float64 `keda:"name=threshold,order=triggerMetadata"`
	ActivationThreshold float64 `keda:"name=activationThreshold,order=triggerMetadata,default=0"`
	TenantName          string  `keda:"name=tenantName,order=triggerMetadata,optional"`
	IgnoreNullValues    bool    `keda:"name=ignoreNullValues,order=triggerMetadata,default=true"`
	UnsafeSsl           bool    `keda:"name=unsafeSsl,order=triggerMetadata,default=false"`
	TriggerIndex        int
	Auth                *authentication.AuthMeta
}

type lokiQueryResult struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric struct{}      `json:"metric"`
			Value  []interface{} `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

func NewLokiScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, err := parseLokiMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing loki metadata: %w", err)
	}

	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, meta.UnsafeSsl)

	return &lokiScaler{
		metricType: metricType,
		metadata:   meta,
		httpClient: httpClient,
		logger:     InitializeLogger(config, "loki_scaler"),
	}, nil
}

func parseLokiMetadata(config *scalersconfig.ScalerConfig) (lokiMetadata, error) {
	meta := lokiMetadata{}
	err := config.TypedConfig(&meta)
	if err != nil {
		return meta, fmt.Errorf("error parsing loki metadata: %w", err)
	}

	if config.AsMetricSource {
		meta.Threshold = 0
	}

	auth, err := authentication.GetAuthConfigs(config.TriggerMetadata, config.AuthParams)
	if err != nil {
		return meta, err
	}
	meta.Auth = auth
	meta.TriggerIndex = config.TriggerIndex

	return meta, nil
}

func (s *lokiScaler) Close(context.Context) error {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}

func (s *lokiScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.TriggerIndex, "loki"),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.Threshold),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

func (s *lokiScaler) ExecuteLokiQuery(ctx context.Context) (float64, error) {
	u, err := url.ParseRequestURI(s.metadata.ServerAddress)
	if err != nil {
		return -1, err
	}
	u.Path = "/loki/api/v1/query"
	u.RawQuery = url.Values{"query": []string{s.metadata.Query}}.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return -1, err
	}

	if s.metadata.Auth != nil {
		if s.metadata.Auth.EnableBearerAuth {
			req.Header.Add("Authorization", authentication.GetBearerToken(s.metadata.Auth))
		} else if s.metadata.Auth.EnableBasicAuth {
			req.SetBasicAuth(s.metadata.Auth.Username, s.metadata.Auth.Password)
		}
	}

	if s.metadata.TenantName != "" {
		req.Header.Add(tenantNameHeaderKey, s.metadata.TenantName)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return -1, err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return -1, fmt.Errorf("loki query api returned error. status: %d response: %s", resp.StatusCode, string(body))
	}

	var result lokiQueryResult
	if err := json.Unmarshal(body, &result); err != nil {
		return -1, err
	}

	return s.parseQueryResult(result)
}

func (s *lokiScaler) parseQueryResult(result lokiQueryResult) (float64, error) {
	if len(result.Data.Result) == 0 {
		if s.metadata.IgnoreNullValues {
			return 0, nil
		}
		return -1, fmt.Errorf("loki metrics may be lost, the result is empty")
	}

	if len(result.Data.Result) > 1 {
		return -1, fmt.Errorf("loki query %s returned multiple elements", s.metadata.Query)
	}

	values := result.Data.Result[0].Value
	if len(values) == 0 {
		if s.metadata.IgnoreNullValues {
			return 0, nil
		}
		return -1, fmt.Errorf("loki metrics may be lost, the value list is empty")
	}

	if len(values) < 2 {
		return -1, fmt.Errorf("loki query %s didn't return enough values", s.metadata.Query)
	}

	if values[1] == nil {
		return 0, nil
	}

	str, ok := values[1].(string)
	if !ok {
		return -1, fmt.Errorf("failed to parse loki value as string")
	}

	v, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return -1, fmt.Errorf("error converting loki value %s: %w", str, err)
	}

	return v, nil
}

// GetMetricsAndActivity returns an external metric value for the loki
func (s *lokiScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	val, err := s.ExecuteLokiQuery(ctx)
	if err != nil {
		s.logger.Error(err, "error executing loki query")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, val)
	return []external_metrics.ExternalMetricValue{metric}, val > s.metadata.ActivationThreshold, nil
}
