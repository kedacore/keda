package scalers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	datadog "github.com/DataDog/datadog-api-client-go/api/v1/datadog"
	"github.com/go-logr/logr"
	"github.com/tidwall/gjson"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/authentication"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type datadogScaler struct {
	metadata             *datadogMetadata
	apiClient            *datadog.APIClient
	httpClient           *http.Client
	logger               logr.Logger
	useClusterAgentProxy bool
	metricType           v2.MetricTargetType
}

// TODO: Need to check whether we can deprecate vType and how should we proceed with it
type datadogMetadata struct {
	// AuthParams Cluster Agent Proxy
	DatadogNamespace          string `keda:"name=datadogNamespace,          order=authParams, optional"`
	DatadogMetricsService     string `keda:"name=datadogMetricsService,     order=authParams, optional"`
	DatadogMetricsServicePort int    `keda:"name=datadogMetricsServicePort, order=authParams, default=8443"`
	UnsafeSsl                 bool   `keda:"name=unsafeSsl,                 order=authParams, default=false"`

	// bearer auth Cluster Agent Proxy
	AuthMode         string `keda:"name=authMode,  order=authParams, optional"`
	EnableBearerAuth bool
	BearerToken      string `keda:"name=token,     order=authParams,      optional"`

	// TriggerMetadata Cluster Agent Proxy
	DatadogMetricServiceURL string
	DatadogMetricName       string  `keda:"name=datadogMetricName,       order=triggerMetadata, optional"`
	DatadogMetricNamespace  string  `keda:"name=datadogMetricNamespace,  order=triggerMetadata, optional"`
	ActivationTargetValue   float64 `keda:"name=activationTargetValue,   order=triggerMetadata, default=0"`

	// AuthParams Datadog API
	APIKey      string `keda:"name=apiKey,      order=authParams, optional"`
	AppKey      string `keda:"name=appKey,      order=authParams, optional"`
	DatadogSite string `keda:"name=datadogSite, order=authParams, default=datadoghq.com"`

	// TriggerMetadata Datadog API
	Query                    string  `keda:"name=query,                   order=triggerMetadata, optional"`
	QueryAggegrator          string  `keda:"name=queryAggregator,         order=triggerMetadata, optional, enum=average;max"`
	ActivationQueryValue     float64 `keda:"name=activationQueryValue,    order=triggerMetadata, default=0"`
	Age                      int     `keda:"name=age,                     order=triggerMetadata, default=90"`
	TimeWindowOffset         int     `keda:"name=timeWindowOffset,        order=triggerMetadata, default=0"`
	LastAvailablePointOffset int     `keda:"name=lastAvailablePointOffset,order=triggerMetadata, default=0"`

	// TriggerMetadata Common
	HpaMetricName string  `keda:"name=hpaMetricName,          order=triggerMetadata, optional"`
	FillValue     float64 `keda:"name=metricUnavailableValue, order=triggerMetadata, default=0"`
	UseFiller     bool
	TargetValue   float64 `keda:"name=targetValue;queryValue, order=triggerMetadata, default=-1"`
	vType         v2.MetricTargetType
}

const avgString = "average"

var filter *regexp.Regexp

func init() {
	filter = regexp.MustCompile(`.*\{.*\}.*`)
}

// NewDatadogScaler creates a new Datadog scaler
func NewDatadogScaler(ctx context.Context, config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}
	logger := InitializeLogger(config, "datadog_scaler")

	var useClusterAgentProxy bool
	var meta *datadogMetadata
	var apiClient *datadog.APIClient
	var httpClient *http.Client

	if val, ok := config.TriggerMetadata["useClusterAgentProxy"]; ok {
		useClusterAgentProxy, err = strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("error parsing useClusterAgentProxy: %w", err)
		}
	}

	if useClusterAgentProxy {
		meta, err = parseDatadogClusterAgentMetadata(config, logger)
		if err != nil {
			return nil, fmt.Errorf("error parsing Datadog metadata: %w", err)
		}
		httpClient = kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, meta.UnsafeSsl)
	} else {
		meta, err = parseDatadogAPIMetadata(config, logger)
		if err != nil {
			return nil, fmt.Errorf("error parsing Datadog metadata: %w", err)
		}
		apiClient, err = newDatadogAPIConnection(ctx, meta, config)
		if err != nil {
			return nil, fmt.Errorf("error establishing Datadog connection: %w", err)
		}
	}

	return &datadogScaler{
		metricType:           metricType,
		metadata:             meta,
		apiClient:            apiClient,
		httpClient:           httpClient,
		logger:               logger,
		useClusterAgentProxy: useClusterAgentProxy,
	}, nil
}

// parseDatadogQuery checks correctness of the user query
func parseDatadogQuery(q string) (bool, error) {
	// Wellformed Datadog queries require a filter (between curly brackets)
	if !filter.MatchString(q) {
		return false, fmt.Errorf("malformed Datadog query: missing query scope")
	}

	return true, nil
}

// buildClusterAgentURL builds the URL for the Cluster Agent Metrics API service
func buildClusterAgentURL(datadogMetricsService, datadogNamespace string, datadogMetricsServicePort int) string {
	return fmt.Sprintf("https://%s.%s:%d/apis/external.metrics.k8s.io/v1beta1", datadogMetricsService, datadogNamespace, datadogMetricsServicePort)
}

// buildMetricURL builds the URL for the Datadog metric
func buildMetricURL(datadogClusterAgentURL, datadogMetricNamespace, datadogMetricName string) string {
	return fmt.Sprintf("%s/namespaces/%s/%s", datadogClusterAgentURL, datadogMetricNamespace, datadogMetricName)
}

func parseDatadogAPIMetadata(config *scalersconfig.ScalerConfig, logger logr.Logger) (*datadogMetadata, error) {
	meta := &datadogMetadata{}
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing Datadog metadata: %w", err)
	}

	if meta.Age < 60 {
		logger.Info("selecting a window smaller than 60 seconds can cause Datadog not finding a metric value for the query")
	}
	if meta.AppKey == "" {
		return nil, fmt.Errorf("error parsing Datadog metadata: missing AppKey")
	}
	if meta.APIKey == "" {
		return nil, fmt.Errorf("error parsing Datadog metadata: missing APIKey")
	}
	if meta.TargetValue == -1 {
		if config.AsMetricSource {
			meta.TargetValue = 0
		} else {
			return nil, fmt.Errorf("no targetValue or queryValue given")
		}
	}
	if val, ok := config.TriggerMetadata["type"]; ok {
		logger.V(0).Info("trigger.metadata.type is deprecated in favor of trigger.metricType")
		if config.MetricType != "" {
			return nil, fmt.Errorf("only one of trigger.metadata.type or trigger.metricType should be defined")
		}
		val = strings.ToLower(val)
		switch val {
		case avgString:
			meta.vType = v2.AverageValueMetricType
		case "global":
			meta.vType = v2.ValueMetricType
		default:
			return nil, fmt.Errorf("type has to be global or average")
		}
	} else {
		metricType, err := GetMetricTargetType(config)
		if err != nil {
			return nil, fmt.Errorf("error getting scaler metric type: %w", err)
		}
		meta.vType = metricType
	}
	if meta.Query == "" {
		return nil, fmt.Errorf("error parsing Datadog metadata: missing Query")
	}

	if meta.Query != "" {
		meta.HpaMetricName = meta.Query[0:strings.Index(meta.Query, "{")]
		meta.HpaMetricName = GenerateMetricNameWithIndex(config.TriggerIndex, kedautil.NormalizeString(fmt.Sprintf("datadog-%s", meta.HpaMetricName)))
	} else {
		meta.HpaMetricName = "datadogmetric@" + meta.DatadogMetricNamespace + ":" + meta.DatadogMetricName
	}

	return meta, nil
}

func parseDatadogClusterAgentMetadata(config *scalersconfig.ScalerConfig, logger logr.Logger) (*datadogMetadata, error) {
	meta := &datadogMetadata{}
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing Datadog metadata: %w", err)
	}
	if meta.DatadogMetricsService == "" {
		return nil, fmt.Errorf("datadog metrics service is required")
	}

	if meta.DatadogMetricName == "" {
		return nil, fmt.Errorf("datadog metric name is required")
	}
	if meta.DatadogNamespace == "" {
		return nil, fmt.Errorf("datadog namespace is required")
	}
	if meta.DatadogMetricNamespace == "" {
		return nil, fmt.Errorf("datadog metric namespace is required")
	}
	if meta.TargetValue == -1 {
		if config.AsMetricSource {
			meta.TargetValue = 0
		} else {
			return nil, fmt.Errorf("no targetValue or queryValue given")
		}
	}
	if val, ok := config.TriggerMetadata["type"]; ok {
		logger.V(0).Info("trigger.metadata.type is deprecated in favor of trigger.metricType")
		if config.MetricType != "" {
			return nil, fmt.Errorf("only one of trigger.metadata.type or trigger.metricType should be defined")
		}
		val = strings.ToLower(val)
		switch val {
		case avgString:
			meta.vType = v2.AverageValueMetricType
		case "global":
			meta.vType = v2.ValueMetricType
		default:
			return nil, fmt.Errorf("type has to be global or average")
		}
	} else {
		metricType, err := GetMetricTargetType(config)
		if err != nil {
			return nil, fmt.Errorf("error getting scaler metric type: %w", err)
		}
		meta.vType = metricType
	}
	meta.HpaMetricName = "datadogmetric@" + meta.DatadogMetricNamespace + ":" + meta.DatadogMetricName

	meta.DatadogMetricServiceURL = buildClusterAgentURL(meta.DatadogMetricsService, meta.DatadogMetricNamespace, meta.DatadogMetricsServicePort)

	return meta, nil
}

// newDatadogAPIConnection tests a connection to the Datadog API
func newDatadogAPIConnection(ctx context.Context, meta *datadogMetadata, config *scalersconfig.ScalerConfig) (*datadog.APIClient, error) {
	ctx = context.WithValue(
		ctx,
		datadog.ContextAPIKeys,
		map[string]datadog.APIKey{
			"apiKeyAuth": {
				Key: meta.APIKey,
			},
			"appKeyAuth": {
				Key: meta.AppKey,
			},
		},
	)

	ctx = context.WithValue(ctx,
		datadog.ContextServerVariables,
		map[string]string{
			"site": meta.DatadogSite,
		})

	configuration := datadog.NewConfiguration()
	configuration.HTTPClient = kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, false)
	apiClient := datadog.NewAPIClient(configuration)

	_, _, err := apiClient.AuthenticationApi.Validate(ctx) //nolint:bodyclose
	if err != nil {
		return nil, fmt.Errorf("error connecting to Datadog API endpoint: %w", err)
	}

	return apiClient, nil
}

// No need to close connections
func (s *datadogScaler) Close(context.Context) error {
	if s.apiClient != nil {
		s.apiClient.GetConfig().HTTPClient.CloseIdleConnections()
	}
	return nil
}

// getQueryResult returns result of the scaler query
func (s *datadogScaler) getQueryResult(ctx context.Context) (float64, error) {
	ctx = context.WithValue(
		ctx,
		datadog.ContextAPIKeys,
		map[string]datadog.APIKey{
			"apiKeyAuth": {
				Key: s.metadata.APIKey,
			},
			"appKeyAuth": {
				Key: s.metadata.AppKey,
			},
		},
	)

	ctx = context.WithValue(ctx,
		datadog.ContextServerVariables,
		map[string]string{
			"site": s.metadata.DatadogSite,
		})

	timeWindowTo := time.Now().Unix() - int64(s.metadata.TimeWindowOffset)
	timeWindowFrom := timeWindowTo - int64(s.metadata.Age)
	resp, r, err := s.apiClient.MetricsApi.QueryMetrics(ctx, timeWindowFrom, timeWindowTo, s.metadata.Query) //nolint:bodyclose

	if r != nil {
		if r.StatusCode == 429 {
			rateLimit := r.Header.Get("X-Ratelimit-Limit")
			rateLimitReset := r.Header.Get("X-Ratelimit-Reset")
			rateLimitPeriod := r.Header.Get("X-Ratelimit-Period")

			return -1, fmt.Errorf("your Datadog account reached the %s queries per %s seconds rate limit, next limit reset will happen in %s seconds", rateLimit, rateLimitPeriod, rateLimitReset)
		}

		if r.StatusCode != 200 {
			if err != nil {
				return -1, fmt.Errorf("error when retrieving Datadog metrics: %w", err)
			}
			return -1, fmt.Errorf("error when retrieving Datadog metrics")
		}
	}

	if err != nil {
		return -1, fmt.Errorf("error when retrieving Datadog metrics: %w", err)
	}

	if resp.GetStatus() == "error" {
		if msg, ok := resp.GetErrorOk(); ok {
			return -1, fmt.Errorf("error when retrieving Datadog metrics: %s", *msg)
		}
		return -1, fmt.Errorf("error when retrieving Datadog metrics")
	}

	series := resp.GetSeries()

	if len(series) == 0 {
		if !s.metadata.UseFiller {
			return 0, fmt.Errorf("no Datadog metrics returned for the given time window")
		}
		return s.metadata.FillValue, nil
	}

	// Require queryAggregator be set explicitly for multi-query
	if len(series) > 1 && s.metadata.QueryAggegrator == "" {
		return 0, fmt.Errorf("query returned more than 1 series; modify the query to return only 1 series or add a queryAggregator")
	}

	// Collect all latest point values from any/all series
	results := make([]float64, len(series))
	for i := 0; i < len(series); i++ {
		points := series[i].GetPointlist()
		index := len(points) - 1
		// Find out the last point != nil
		for j := index; j >= 0; j-- {
			if len(points[j]) >= 2 && points[j][1] != nil {
				index = j
				break
			}
		}
		if index < s.metadata.LastAvailablePointOffset {
			return 0, fmt.Errorf("index is smaller than the lastAvailablePointOffset")
		}
		index -= s.metadata.LastAvailablePointOffset

		if len(points) == 0 || len(points[index]) < 2 || points[index][1] == nil {
			if !s.metadata.UseFiller {
				return 0, fmt.Errorf("no Datadog metrics returned for the given time window")
			}
			return s.metadata.FillValue, nil
		}
		// Return the last point from the series
		results[i] = *points[index][1]
	}

	switch s.metadata.QueryAggegrator {
	case avgString:
		return AvgFloatFromSlice(results), nil
	default:
		// Aggregate Results - default Max value:
		return slices.Max(results), nil
	}
}

func (s *datadogScaler) getDatadogMetricValue(req *http.Request) (float64, error) {
	resp, err := s.httpClient.Do(req)

	if err != nil {
		return 0, fmt.Errorf("error getting metric value: %w", err)
	}

	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		r := gjson.GetBytes(body, "message")
		if r.Type == gjson.String {
			return 0, fmt.Errorf("error getting metric value: %s", r.String())
		}
	}

	valueLocation := "items.0.value"
	r := gjson.GetBytes(body, valueLocation)
	errorMsg := "the metric value must be of type number or a string representing a Quantity got: '%s'"

	if r.Type == gjson.String {
		v, err := resource.ParseQuantity(r.String())
		if err != nil {
			return 0, fmt.Errorf(errorMsg, r.String())
		}
		return v.AsApproximateFloat64(), nil
	}
	if r.Type != gjson.Number {
		return 0, fmt.Errorf(errorMsg, r.Type.String())
	}
	return r.Num, nil
}

func (s *datadogScaler) getDatadogClusterAgentHTTPRequest(ctx context.Context, url string) (*http.Request, error) {
	var req *http.Request
	var err error

	switch {
	case s.metadata.EnableBearerAuth:
		req, err = http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", s.metadata.BearerToken))
		if err != nil {
			return nil, err
		}
		return req, nil

	default:
		req, err = http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return req, err
		}
	}

	return nil, nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *datadogScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: s.metadata.HpaMetricName,
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.TargetValue),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *datadogScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	var metric external_metrics.ExternalMetricValue
	var num float64
	var err error

	if s.useClusterAgentProxy {
		url := buildMetricURL(s.metadata.DatadogMetricServiceURL, s.metadata.DatadogMetricNamespace, s.metadata.HpaMetricName)

		req, err := s.getDatadogClusterAgentHTTPRequest(ctx, url)
		if (err != nil) || (req == nil) {
			return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error generating http request: %w", err)
		}

		num, err = s.getDatadogMetricValue(req)
		if err != nil {
			return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error getting metric value: %w", err)
		}

		metric = GenerateMetricInMili(metricName, num)
		return []external_metrics.ExternalMetricValue{metric}, num > s.metadata.ActivationTargetValue, nil
	}
	num, err = s.getQueryResult(ctx)
	if err != nil {
		s.logger.Error(err, "error getting metrics from Datadog")
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error getting metrics from Datadog: %w", err)
	}

	metric = GenerateMetricInMili(metricName, num)
	return []external_metrics.ExternalMetricValue{metric}, num > s.metadata.ActivationQueryValue, nil
}

// AvgFloatFromSlice finds the average value in a slice of floats
func AvgFloatFromSlice(results []float64) float64 {
	total := 0.0
	for _, result := range results {
		total += result
	}
	return total / float64(len(results))
}

func (s *datadogMetadata) Validate() error {
	if s.Age < 0 {
		return fmt.Errorf("age should not be smaller than 0 seconds")
	}

	if s.TimeWindowOffset < 0 {
		return fmt.Errorf("timeWindowOffset should not be smaller than 0 seconds")
	}
	if s.LastAvailablePointOffset < 0 {
		return fmt.Errorf("lastAvailablePointOffset should not be smaller than 0")
	}
	if s.Query != "" {
		if _, err := parseDatadogQuery(s.Query); err != nil {
			return fmt.Errorf("error in query: %w", err)
		}
	}
	if s.FillValue == 0 {
		s.UseFiller = false
	}
	if s.AuthMode != "" {
		authType := authentication.Type(strings.TrimSpace(s.AuthMode))
		switch authType {
		case authentication.BearerAuthType:
			if s.BearerToken == "" {
				return fmt.Errorf("BearerToken is required")
			}
			s.EnableBearerAuth = true
		default:
			return fmt.Errorf("err incorrect value for authMode is given: %s", s.AuthMode)
		}
	}
	return nil
}
