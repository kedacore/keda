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

	"github.com/kedacore/keda/v2/pkg/scalers/authentication"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	lokiServerAddress       = "serverAddress"
	lokiQuery               = "query"
	lokiThreshold           = "threshold"
	lokiActivationThreshold = "activationThreshold"
	lokiNamespace           = "namespace"
	tenantName              = "tenantName"
	tenantNameHeaderKey     = "X-Scope-OrgID"
	lokiIgnoreNullValues    = "ignoreNullValues"
)

var (
	lokiDefaultIgnoreNullValues = true
)

type lokiScaler struct {
	metricType v2.MetricTargetType
	metadata   *lokiMetadata
	httpClient *http.Client
	logger     logr.Logger
}

type lokiMetadata struct {
	serverAddress       string
	query               string
	threshold           float64
	activationThreshold float64
	lokiAuth            *authentication.AuthMeta
	triggerIndex        int
	tenantName          string
	ignoreNullValues    bool
	unsafeSsl           bool
}

type lokiQueryResult struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric struct {
			} `json:"metric"`
			Value []interface{} `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

// NewLokiScaler returns a new lokiScaler
func NewLokiScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "loki_scaler")

	meta, err := parseLokiMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing loki metadata: %w", err)
	}

	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, meta.unsafeSsl)

	return &lokiScaler{
		metricType: metricType,
		metadata:   meta,
		httpClient: httpClient,
		logger:     logger,
	}, nil
}

func parseLokiMetadata(config *scalersconfig.ScalerConfig) (meta *lokiMetadata, err error) {
	meta = &lokiMetadata{}

	if val, ok := config.TriggerMetadata[lokiServerAddress]; ok && val != "" {
		meta.serverAddress = val
	} else {
		return nil, fmt.Errorf("no %s given", lokiServerAddress)
	}

	if val, ok := config.TriggerMetadata[lokiQuery]; ok && val != "" {
		meta.query = val
	} else {
		return nil, fmt.Errorf("no %s given", lokiQuery)
	}

	if val, ok := config.TriggerMetadata[lokiThreshold]; ok && val != "" {
		t, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing %s: %w", lokiThreshold, err)
		}

		meta.threshold = t
	} else {
		if config.AsMetricSource {
			meta.threshold = 0
		} else {
			return nil, fmt.Errorf("no %s given", lokiThreshold)
		}
	}

	meta.activationThreshold = 0
	if val, ok := config.TriggerMetadata[lokiActivationThreshold]; ok {
		t, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("activationThreshold parsing error %w", err)
		}

		meta.activationThreshold = t
	}

	if val, ok := config.TriggerMetadata[tenantName]; ok && val != "" {
		meta.tenantName = val
	}

	meta.ignoreNullValues = lokiDefaultIgnoreNullValues
	if val, ok := config.TriggerMetadata[lokiIgnoreNullValues]; ok && val != "" {
		ignoreNullValues, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("err incorrect value for ignoreNullValues given: %s please use true or false", val)
		}
		meta.ignoreNullValues = ignoreNullValues
	}

	meta.unsafeSsl = false
	if val, ok := config.TriggerMetadata[unsafeSsl]; ok && val != "" {
		unsafeSslValue, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("error parsing %s: %w", unsafeSsl, err)
		}

		meta.unsafeSsl = unsafeSslValue
	}

	meta.triggerIndex = config.TriggerIndex

	// parse auth configs from ScalerConfig
	auth, err := authentication.GetAuthConfigs(config.TriggerMetadata, config.AuthParams)
	if err != nil {
		return nil, err
	}
	meta.lokiAuth = auth

	return meta, nil
}

// Close returns a nil error
func (s *lokiScaler) Close(context.Context) error {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *lokiScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, "loki"),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.threshold),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

// ExecuteLokiQuery returns the result of the LogQL query execution
func (s *lokiScaler) ExecuteLokiQuery(ctx context.Context) (float64, error) {
	u, err := url.ParseRequestURI(s.metadata.serverAddress)
	if err != nil {
		return -1, err
	}
	u.Path = "/loki/api/v1/query"

	u.RawQuery = url.Values{
		"query": []string{s.metadata.query},
	}.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return -1, err
	}

	if s.metadata.lokiAuth != nil && s.metadata.lokiAuth.EnableBearerAuth {
		req.Header.Add("Authorization", authentication.GetBearerToken(s.metadata.lokiAuth))
	} else if s.metadata.lokiAuth != nil && s.metadata.lokiAuth.EnableBasicAuth {
		req.SetBasicAuth(s.metadata.lokiAuth.Username, s.metadata.lokiAuth.Password)
	}

	if s.metadata.tenantName != "" {
		req.Header.Add(tenantNameHeaderKey, s.metadata.tenantName)
	}

	r, err := s.httpClient.Do(req)
	if err != nil {
		return -1, err
	}
	defer r.Body.Close()

	b, err := io.ReadAll(r.Body)
	if err != nil {
		return -1, err
	}

	if !(r.StatusCode >= 200 && r.StatusCode <= 299) {
		err := fmt.Errorf("loki query api returned error. status: %d response: %s", r.StatusCode, string(b))
		s.logger.Error(err, "loki query api returned error")
		return -1, err
	}

	var result lokiQueryResult
	err = json.Unmarshal(b, &result)
	if err != nil {
		return -1, err
	}

	var v float64 = -1

	// allow for zero element or single element result sets
	if len(result.Data.Result) == 0 {
		if s.metadata.ignoreNullValues {
			return 0, nil
		}
		return -1, fmt.Errorf("loki metrics may be lost, the result is empty")
	} else if len(result.Data.Result) > 1 {
		return -1, fmt.Errorf("loki query %s returned multiple elements", s.metadata.query)
	}

	valueLen := len(result.Data.Result[0].Value)
	if valueLen == 0 {
		if s.metadata.ignoreNullValues {
			return 0, nil
		}
		return -1, fmt.Errorf("loki metrics may be lost, the value list is empty")
	} else if valueLen < 2 {
		return -1, fmt.Errorf("loki query %s didn't return enough values", s.metadata.query)
	}

	val := result.Data.Result[0].Value[1]
	if val != nil {
		str := val.(string)
		v, err = strconv.ParseFloat(str, 64)
		if err != nil {
			s.logger.Error(err, "Error converting loki value", "loki_value", str)
			return -1, err
		}
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

	return []external_metrics.ExternalMetricValue{metric}, val > s.metadata.activationThreshold, nil
}
