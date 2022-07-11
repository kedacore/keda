package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	url_pkg "net/url"
	"strconv"
	"time"

	"k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kedacore/keda/v2/pkg/scalers/authentication"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	promServerAddress    = "serverAddress"
	promMetricName       = "metricName"
	promQuery            = "query"
	promThreshold        = "threshold"
	promNamespace        = "namespace"
	promCortexScopeOrgID = "cortexOrgID"
	promCortexHeaderKey  = "X-Scope-OrgID"
	ignoreNullValues     = "ignoreNullValues"
)

var (
	defaultIgnoreNullValues = true
)

type prometheusScaler struct {
	metricType v2beta2.MetricTargetType
	metadata   *prometheusMetadata
	httpClient *http.Client
}

type prometheusMetadata struct {
	serverAddress  string
	metricName     string
	query          string
	threshold      float64
	prometheusAuth *authentication.AuthMeta
	namespace      string
	scalerIndex    int
	cortexOrgID    string
	// sometimes should consider there is an error we can accept
	// default value is true/t, to ignore the null value return from prometheus
	// change to false/f if can not accept prometheus return null values
	// https://github.com/kedacore/keda/issues/3065
	ignoreNullValues bool
}

type promQueryResult struct {
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

var prometheusLog = logf.Log.WithName("prometheus_scaler")

// NewPrometheusScaler creates a new prometheusScaler
func NewPrometheusScaler(config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %s", err)
	}

	meta, err := parsePrometheusMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing prometheus metadata: %s", err)
	}

	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, false)

	if meta.prometheusAuth != nil && (meta.prometheusAuth.CA != "" || meta.prometheusAuth.EnableTLS) {
		// create http.RoundTripper with auth settings from ScalerConfig
		if httpClient.Transport, err = authentication.CreateHTTPRoundTripper(
			authentication.NetHTTP,
			meta.prometheusAuth,
		); err != nil {
			predictKubeLog.V(1).Error(err, "init Prometheus client http transport")
			return nil, err
		}
	}

	return &prometheusScaler{
		metricType: metricType,
		metadata:   meta,
		httpClient: httpClient,
	}, nil
}

func parsePrometheusMetadata(config *ScalerConfig) (meta *prometheusMetadata, err error) {
	meta = &prometheusMetadata{}

	if val, ok := config.TriggerMetadata[promServerAddress]; ok && val != "" {
		meta.serverAddress = val
	} else {
		return nil, fmt.Errorf("no %s given", promServerAddress)
	}

	if val, ok := config.TriggerMetadata[promQuery]; ok && val != "" {
		meta.query = val
	} else {
		return nil, fmt.Errorf("no %s given", promQuery)
	}

	if val, ok := config.TriggerMetadata[promMetricName]; ok && val != "" {
		meta.metricName = val
	} else {
		return nil, fmt.Errorf("no %s given", promMetricName)
	}

	if val, ok := config.TriggerMetadata[promThreshold]; ok && val != "" {
		t, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing %s: %s", promThreshold, err)
		}

		meta.threshold = t
	} else {
		return nil, fmt.Errorf("no %s given", promThreshold)
	}

	if val, ok := config.TriggerMetadata[promNamespace]; ok && val != "" {
		meta.namespace = val
	}

	if val, ok := config.TriggerMetadata[promCortexScopeOrgID]; ok && val != "" {
		meta.cortexOrgID = val
	}

	meta.ignoreNullValues = defaultIgnoreNullValues
	if val, ok := config.TriggerMetadata[ignoreNullValues]; ok && val != "" {
		ignoreNullValues, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("err incorrect value for ignoreNullValues given: %s, "+
				"please use true or false", val)
		}
		meta.ignoreNullValues = ignoreNullValues
	}

	meta.scalerIndex = config.ScalerIndex

	// parse auth configs from ScalerConfig
	meta.prometheusAuth, err = authentication.GetAuthConfigs(config.TriggerMetadata, config.AuthParams)
	if err != nil {
		return nil, err
	}

	return meta, nil
}

func (s *prometheusScaler) IsActive(ctx context.Context) (bool, error) {
	val, err := s.ExecutePromQuery(ctx)
	if err != nil {
		prometheusLog.Error(err, "error executing prometheus query")
		return false, err
	}

	return val > 0, nil
}

func (s *prometheusScaler) Close(context.Context) error {
	return nil
}

func (s *prometheusScaler) GetMetricSpecForScaling(context.Context) []v2beta2.MetricSpec {
	metricName := kedautil.NormalizeString(fmt.Sprintf("prometheus-%s", s.metadata.metricName))
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, metricName),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.threshold),
	}
	metricSpec := v2beta2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2beta2.MetricSpec{metricSpec}
}

func (s *prometheusScaler) ExecutePromQuery(ctx context.Context) (float64, error) {
	t := time.Now().UTC().Format(time.RFC3339)
	queryEscaped := url_pkg.QueryEscape(s.metadata.query)
	url := fmt.Sprintf("%s/api/v1/query?query=%s&time=%s", s.metadata.serverAddress, queryEscaped, t)

	// set 'namespace' parameter for namespaced Prometheus requests (eg. for Thanos Querier)
	if s.metadata.namespace != "" {
		url = fmt.Sprintf("%s&namespace=%s", url, s.metadata.namespace)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return -1, err
	}

	if s.metadata.prometheusAuth != nil && s.metadata.prometheusAuth.EnableBearerAuth {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", s.metadata.prometheusAuth.BearerToken))
	} else if s.metadata.prometheusAuth != nil && s.metadata.prometheusAuth.EnableBasicAuth {
		req.SetBasicAuth(s.metadata.prometheusAuth.Username, s.metadata.prometheusAuth.Password)
	}

	if s.metadata.cortexOrgID != "" {
		req.Header.Add(promCortexHeaderKey, s.metadata.cortexOrgID)
	}

	r, err := s.httpClient.Do(req)
	if err != nil {
		return -1, err
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return -1, err
	}
	_ = r.Body.Close()

	if !(r.StatusCode >= 200 && r.StatusCode <= 299) {
		return -1, fmt.Errorf("prometheus query api returned error. status: %d response: %s", r.StatusCode, string(b))
	}

	var result promQueryResult
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
		return -1, fmt.Errorf("prometheus metrics %s target may be lost, the result is empty", s.metadata.metricName)
	} else if len(result.Data.Result) > 1 {
		return -1, fmt.Errorf("prometheus query %s returned multiple elements", s.metadata.query)
	}

	valueLen := len(result.Data.Result[0].Value)
	if valueLen == 0 {
		if s.metadata.ignoreNullValues {
			return 0, nil
		}
		return -1, fmt.Errorf("prometheus metrics %s target may be lost, the value list is empty", s.metadata.metricName)
	} else if valueLen < 2 {
		return -1, fmt.Errorf("prometheus query %s didn't return enough values", s.metadata.query)
	}

	val := result.Data.Result[0].Value[1]
	if val != nil {
		s := val.(string)
		v, err = strconv.ParseFloat(s, 64)
		if err != nil {
			prometheusLog.Error(err, "Error converting prometheus value", "prometheus_value", s)
			return -1, err
		}
	}

	return v, nil
}

func (s *prometheusScaler) GetMetrics(ctx context.Context, metricName string, _ labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	val, err := s.ExecutePromQuery(ctx)
	if err != nil {
		prometheusLog.Error(err, "error executing prometheus query")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := GenerateMetricInMili(metricName, val)

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}
