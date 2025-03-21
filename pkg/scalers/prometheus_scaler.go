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

type prometheusScaler struct {
	metricType v2.MetricTargetType
	metadata   *prometheusMetadata
	httpClient *http.Client
	logger     logr.Logger
}

// IgnoreNullValues - sometimes should consider there is an error we can accept
// default value is true/t, to ignore the null value return from prometheus
// change to false/f if can not accept prometheus return null values
// https://github.com/kedacore/keda/issues/3065
type prometheusMetadata struct {
	triggerIndex int

	PrometheusAuth      *authentication.Config `keda:"optional"`
	ServerAddress       string                 `keda:"name=serverAddress,       order=triggerMetadata"`
	Query               string                 `keda:"name=query,               order=triggerMetadata"`
	QueryParameters     map[string]string      `keda:"name=queryParameters,     order=triggerMetadata,    				optional"`
	Threshold           float64                `keda:"name=threshold,           order=triggerMetadata"`
	ActivationThreshold float64                `keda:"name=activationThreshold, order=triggerMetadata, 				    optional"`
	Namespace           string                 `keda:"name=namespace,           order=triggerMetadata, 				    optional"`
	CustomHeaders       map[string]string      `keda:"name=customHeaders,       order=triggerMetadata, 				    optional"`
	IgnoreNullValues    bool                   `keda:"name=ignoreNullValues,    order=triggerMetadata, 				    default=true"`
	UnsafeSSL           bool                   `keda:"name=unsafeSsl,           order=triggerMetadata, 				    optional"`
	AwsRegion           string                 `keda:"name=awsRegion, 			order=triggerMetadata;authParams, 		optional"`
	Timeout             int                    `keda:"name=timeout,             order=triggerMetadata, 					optional"` // custom HTTP client timeout
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

	// handle HTTP client timeout
	httpClientTimeout := config.GlobalHTTPTimeout
	if meta.Timeout > 0 {
		httpClientTimeout = time.Duration(meta.Timeout) * time.Millisecond
	}

	httpClient := kedautil.CreateHTTPClient(httpClientTimeout, meta.UnsafeSSL)

	if !meta.PrometheusAuth.Disabled() {
		if meta.PrometheusAuth.CA != "" || meta.PrometheusAuth.EnabledTLS() {
			// create http.RoundTripper with auth settings from ScalerConfig
			transport, err := authentication.CreateHTTPRoundTripper(
				authentication.NetHTTP,
				meta.PrometheusAuth.ToAuthMeta(),
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

		awsTransport, err := aws.NewSigV4RoundTripper(config, meta.AwsRegion)
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
	err = checkAuthConfigWithPodIdentity(config, meta)
	if err != nil {
		return nil, err
	}

	// validate the timeout
	if meta.Timeout < 0 {
		return nil, fmt.Errorf("timeout must be greater than 0: %d", meta.Timeout)
	}

	return meta, nil
}

func checkAuthConfigWithPodIdentity(config *scalersconfig.ScalerConfig, meta *prometheusMetadata) error {
	if meta == nil || meta.PrometheusAuth.Disabled() {
		return nil
	}
	if !(config.PodIdentity.Provider == kedav1alpha1.PodIdentityProviderNone || config.PodIdentity.Provider == "") {
		return fmt.Errorf("pod identity cannot be enabled with other auth types")
	}
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
	case s.metadata.PrometheusAuth.Disabled():
		break
	case s.metadata.PrometheusAuth.EnabledBearerAuth():
		req.Header.Set("Authorization", s.metadata.PrometheusAuth.GetBearerToken())
	case s.metadata.PrometheusAuth.EnabledBasicAuth():
		req.SetBasicAuth(s.metadata.PrometheusAuth.Username, s.metadata.PrometheusAuth.Password)
	case s.metadata.PrometheusAuth.EnabledCustomAuth():
		req.Header.Set(s.metadata.PrometheusAuth.CustomAuthHeader, s.metadata.PrometheusAuth.CustomAuthValue)
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
