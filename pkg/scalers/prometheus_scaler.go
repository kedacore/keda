package scalers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	url_pkg "net/url"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/authentication"
	"github.com/kedacore/keda/v2/pkg/scalers/aws"
	"github.com/kedacore/keda/v2/pkg/scalers/azure"
	"github.com/kedacore/keda/v2/pkg/scalers/gcp"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	promServerAddress       = "serverAddress"
	promQuery               = "query"
	promQueryParameters     = "queryParameters"
	promThreshold           = "threshold"
	promActivationThreshold = "activationThreshold"
	promNamespace           = "namespace"
	promCortexScopeOrgID    = "cortexOrgID"
	promCustomHeaders       = "customHeaders"
	ignoreNullValues        = "ignoreNullValues"
	unsafeSsl               = "unsafeSsl"
)

type prometheusScaler struct {
	metricType v2.MetricTargetType
	metadata   *prometheusMetadata
	httpClient *http.Client
	logger     logr.Logger
}

// sometimes should consider there is an error we can accept
// default value is true/t, to ignore the null value return from prometheus
// change to false/f if can not accept prometheus return null values
// https://github.com/kedacore/keda/issues/3065
type prometheusMetadata struct {
	prometheusAuth *authentication.AuthMeta
	triggerIndex   int

	ServerAddress       string            `keda:"name=serverAddress,       parsingOrder=triggerMetadata"`
	Query               string            `keda:"name=query,               parsingOrder=triggerMetadata"`
	QueryParameters     map[string]string `keda:"name=queryParameters,     parsingOrder=triggerMetadata, optional"`
	Threshold           float64           `keda:"name=threshold,           parsingOrder=triggerMetadata"`
	ActivationThreshold float64           `keda:"name=activationThreshold, parsingOrder=triggerMetadata, optional"`
	Namespace           string            `keda:"name=namespace,           parsingOrder=triggerMetadata, optional"`
	CustomHeaders       map[string]string `keda:"name=customHeaders,       parsingOrder=triggerMetadata, optional"`
	IgnoreNullValues    bool              `keda:"name=ignoreNullValues,    parsingOrder=triggerMetadata, optional, default=true"`
	UnsafeSSL           bool              `keda:"name=unsafeSsl,           parsingOrder=triggerMetadata, optional"`
	CortexOrgID         string            `keda:"name=cortexOrgID,         parsingOrder=triggerMetadata, optional, deprecated=use customHeaders instead"`
}

type promQueryResult struct {
	Status string `json:"status"`

	Data struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric struct{}      `json:"metric"`
			Value  []interface{} `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

// NewPrometheusScaler creates a new prometheusScaler
func NewPrometheusScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "prometheus_scaler")

	meta, err := parsePrometheusMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing prometheus metadata: %w", err)
	}

	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, meta.UnsafeSSL)

	if meta.prometheusAuth != nil {
		if meta.prometheusAuth.CA != "" || meta.prometheusAuth.EnableTLS {
			// create http.RoundTripper with auth settings from ScalerConfig
			transport, err := authentication.CreateHTTPRoundTripper(
				authentication.NetHTTP,
				meta.prometheusAuth,
			)
			if err != nil {
				logger.V(1).Error(err, "init Prometheus client http transport")
				return nil, err
			}
			httpClient.Transport = transport
		}
	} else {
		// could be the case of azure managed prometheus. Try and get the round-tripper.
		// If it's not the case of azure managed prometheus, we will get both transport and err as nil and proceed assuming no auth.
		azureTransport, err := azure.TryAndGetAzureManagedPrometheusHTTPRoundTripper(logger, config.PodIdentity, config.TriggerMetadata)
		if err != nil {
			logger.V(1).Error(err, "error while init Azure Managed Prometheus client http transport")
			return nil, err
		}

		// transport should not be nil if its a case of azure managed prometheus
		if azureTransport != nil {
			httpClient.Transport = azureTransport
		}

		gcpTransport, err := gcp.GetGCPOAuth2HTTPTransport(config, httpClient.Transport, gcp.GcpScopeMonitoringRead)
		if err != nil && !errors.Is(err, gcp.ErrGoogleApplicationCrendentialsNotFound) {
			logger.V(1).Error(err, "failed to get GCP client HTTP transport (either using Google application credentials or workload identity)")
			return nil, err
		}

		if err == nil && gcpTransport != nil {
			httpClient.Transport = gcpTransport
		}

		awsTransport, err := aws.NewSigV4RoundTripper(config)
		if err != nil {
			logger.V(1).Error(err, "failed to get AWS client HTTP transport ")
			return nil, err
		}

		if err == nil && awsTransport != nil {
			httpClient.Transport = awsTransport
		}
	}

	return &prometheusScaler{
		metricType: metricType,
		metadata:   meta,
		httpClient: httpClient,
		logger:     logger,
	}, nil
}

func parsePrometheusMetadata(config *scalersconfig.ScalerConfig) (meta *prometheusMetadata, err error) {
	meta = &prometheusMetadata{}
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing prometheus metadata: %w", err)
	}

	meta.triggerIndex = config.TriggerIndex
	err = parseAuthConfig(config, meta)
	if err != nil {
		return nil, err
	}

	return meta, nil
}

func parseAuthConfig(config *scalersconfig.ScalerConfig, meta *prometheusMetadata) error {
	// parse auth configs from ScalerConfig
	auth, err := authentication.GetAuthConfigs(config.TriggerMetadata, config.AuthParams)
	if err != nil {
		return err
	}

	if auth != nil && !(config.PodIdentity.Provider == kedav1alpha1.PodIdentityProviderNone || config.PodIdentity.Provider == "") {
		return fmt.Errorf("pod identity cannot be enabled with other auth types")
	}
	meta.prometheusAuth = auth

	return nil
}

func (s *prometheusScaler) Close(context.Context) error {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}

func (s *prometheusScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	metricName := kedautil.NormalizeString("prometheus")
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, metricName),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.Threshold),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

func (s *prometheusScaler) ExecutePromQuery(ctx context.Context) (float64, error) {
	t := time.Now().UTC().Format(time.RFC3339)
	queryEscaped := url_pkg.QueryEscape(s.metadata.Query)
	url := fmt.Sprintf("%s/api/v1/query?query=%s&time=%s", s.metadata.ServerAddress, queryEscaped, t)

	// set 'namespace' parameter for namespaced Prometheus requests (e.g. for Thanos Querier)
	if s.metadata.Namespace != "" {
		url = fmt.Sprintf("%s&namespace=%s", url, s.metadata.Namespace)
	}

	for queryParameterKey, queryParameterValue := range s.metadata.QueryParameters {
		queryParameterKeyEscaped := url_pkg.QueryEscape(queryParameterKey)
		queryParameterValueEscaped := url_pkg.QueryEscape(queryParameterValue)
		url = fmt.Sprintf("%s&%s=%s", url, queryParameterKeyEscaped, queryParameterValueEscaped)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return -1, err
	}

	for headerName, headerValue := range s.metadata.CustomHeaders {
		req.Header.Add(headerName, headerValue)
	}

	switch {
	case s.metadata.prometheusAuth == nil:
		break
	case s.metadata.prometheusAuth.EnableBearerAuth:
		req.Header.Set("Authorization", authentication.GetBearerToken(s.metadata.prometheusAuth))
	case s.metadata.prometheusAuth.EnableBasicAuth:
		req.SetBasicAuth(s.metadata.prometheusAuth.Username, s.metadata.prometheusAuth.Password)
	case s.metadata.prometheusAuth.EnableCustomAuth:
		req.Header.Set(s.metadata.prometheusAuth.CustomAuthHeader, s.metadata.prometheusAuth.CustomAuthValue)
	}

	r, err := s.httpClient.Do(req)
	if err != nil {
		return -1, err
	}

	b, err := io.ReadAll(r.Body)
	if err != nil {
		return -1, err
	}
	defer r.Body.Close()

	if !(r.StatusCode >= 200 && r.StatusCode <= 299) {
		err := fmt.Errorf("prometheus query api returned error. status: %d response: %s", r.StatusCode, string(b))
		s.logger.Error(err, "prometheus query api returned error")
		return -1, err
	}

	var result promQueryResult
	err = json.Unmarshal(b, &result)
	if err != nil {
		return -1, err
	}

	var v float64 = -1

	// allow for zero element or single element result sets
	if len(result.Data.Result) == 0 {
		if s.metadata.IgnoreNullValues {
			return 0, nil
		}
		return -1, fmt.Errorf("prometheus metrics 'prometheus' target may be lost, the result is empty")
	} else if len(result.Data.Result) > 1 {
		return -1, fmt.Errorf("prometheus query %s returned multiple elements", s.metadata.Query)
	}

	valueLen := len(result.Data.Result[0].Value)
	if valueLen == 0 {
		if s.metadata.IgnoreNullValues {
			return 0, nil
		}
		return -1, fmt.Errorf("prometheus metrics 'prometheus' target may be lost, the value list is empty")
	} else if valueLen < 2 {
		return -1, fmt.Errorf("prometheus query %s didn't return enough values", s.metadata.Query)
	}

	val := result.Data.Result[0].Value[1]
	if val != nil {
		str := val.(string)
		v, err = strconv.ParseFloat(str, 64)
		if err != nil {
			s.logger.Error(err, "Error converting prometheus value", "prometheus_value", str)
			return -1, err
		}
	}

	if math.IsInf(v, 0) {
		if s.metadata.IgnoreNullValues {
			return 0, nil
		}
		err := fmt.Errorf("promtheus query returns %f", v)
		s.logger.Error(err, "Error converting prometheus value")
		return -1, err
	}

	return v, nil
}

func (s *prometheusScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	val, err := s.ExecutePromQuery(ctx)
	if err != nil {
		s.logger.Error(err, "error executing prometheus query")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, val)

	return []external_metrics.ExternalMetricValue{metric}, val > s.metadata.ActivationThreshold, nil
}
