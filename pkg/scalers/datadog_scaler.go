package scalers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"slices"
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
	"github.com/kedacore/keda/v2/version"
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
	UseClusterAgentProxy bool     `keda:"name=useClusterAgentProxy, order=triggerMetadata, default=false"`
	HpaMetricName        string   `keda:"name=hpaMetricName,          order=triggerMetadata, optional"`
	FillValue            *float64 `keda:"name=metricUnavailableValue, order=triggerMetadata, optional"`
	UseFiller            bool
	TargetValue          float64       `keda:"name=targetValue;queryValue, order=triggerMetadata, default=-1"`
	Timeout              time.Duration `keda:"name=timeout,             	order=triggerMetadata, optional"`
	TriggerIndex         int
	vType                v2.MetricTargetType
}

const avgString = "average"

var filter *regexp.Regexp

func init() {
	filter = regexp.MustCompile(`.*\{.*\}.*`)
}

// NewDatadogScaler creates a new Datadog scaler
func NewDatadogScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}
	logger := InitializeLogger(config, "datadog_scaler")

	var apiClient *datadog.APIClient
	var httpClient *http.Client

	meta := &datadogMetadata{}
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing Datadog metadata: %w", err)
	}
	meta.TriggerIndex = config.TriggerIndex

	if meta.Timeout == 0 {
		meta.Timeout = config.GlobalHTTPTimeout
	}

	if meta.UseClusterAgentProxy {
		if err := validateClusterAgentMetadata(meta, config, logger); err != nil {
			return nil, err
		}
		httpClient = kedautil.CreateHTTPClient(meta.Timeout, meta.UnsafeSsl)
	} else {
		if err := validateAPIMetadata(meta, config, logger); err != nil {
			return nil, err
		}
		apiClient = newDatadogAPIClient(meta)
	}

	return &datadogScaler{
		metricType:           metricType,
		metadata:             meta,
		apiClient:            apiClient,
		httpClient:           httpClient,
		logger:               logger,
		useClusterAgentProxy: meta.UseClusterAgentProxy,
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

func validateAPIMetadata(meta *datadogMetadata, config *scalersconfig.ScalerConfig, logger logr.Logger) error {
	if meta.Age < 60 {
		logger.Info("selecting a window smaller than 60 seconds can cause Datadog not finding a metric value for the query")
	}
	if meta.AppKey == "" {
		return fmt.Errorf("error parsing Datadog metadata: missing AppKey")
	}
	if meta.APIKey == "" {
		return fmt.Errorf("error parsing Datadog metadata: missing APIKey")
	}
	if meta.TargetValue == -1 {
		if config.AsMetricSource {
			meta.TargetValue = 0
		} else {
			return fmt.Errorf("no targetValue or queryValue given")
		}
	}

	if val, ok := config.TriggerMetadata["type"]; ok {
		logger.V(0).Info("trigger.metadata.type is deprecated in favor of trigger.metricType")
		if config.MetricType != "" {
			return fmt.Errorf("only one of trigger.metadata.type or trigger.metricType should be defined")
		}
		val = strings.ToLower(val)
		switch val {
		case avgString:
			meta.vType = v2.AverageValueMetricType
		case "global":
			meta.vType = v2.ValueMetricType
		default:
			return fmt.Errorf("type has to be global or average")
		}
	} else {
		metricType, err := GetMetricTargetType(config)
		if err != nil {
			return fmt.Errorf("error getting scaler metric type: %w", err)
		}
		meta.vType = metricType
	}

	if meta.Query == "" {
		return fmt.Errorf("error parsing Datadog metadata: missing Query")
	}

	idx := strings.Index(meta.Query, "{")
	if idx == -1 {
		return fmt.Errorf("error parsing Datadog metadata: Query must contain '{' character")
	}
	meta.HpaMetricName = meta.Query[0:idx]
	meta.HpaMetricName = GenerateMetricNameWithIndex(config.TriggerIndex, kedautil.NormalizeString(fmt.Sprintf("datadog-%s", meta.HpaMetricName)))

	// Set UseFiller flag if metricUnavailableValue is explicitly configured
	if meta.FillValue != nil {
		meta.UseFiller = true
	}
	meta.HpaMetricName = meta.Query[0:idx]
	meta.HpaMetricName = GenerateMetricNameWithIndex(config.TriggerIndex, kedautil.NormalizeString(fmt.Sprintf("datadog-%s", meta.HpaMetricName)))

	return nil
}

func validateClusterAgentMetadata(meta *datadogMetadata, config *scalersconfig.ScalerConfig, logger logr.Logger) error {
	if meta.DatadogMetricsService == "" {
		return fmt.Errorf("datadog metrics service is required")
	}
	if meta.DatadogMetricName == "" {
		return fmt.Errorf("datadog metric name is required")
	}
	if meta.DatadogNamespace == "" {
		return fmt.Errorf("datadog namespace is required")
	}
	if meta.DatadogMetricNamespace == "" {
		return fmt.Errorf("datadog metric namespace is required")
	}
	if meta.TargetValue == -1 {
		if config.AsMetricSource {
			meta.TargetValue = 0
		} else {
			return fmt.Errorf("no targetValue or queryValue given")
		}
	}

	if val, ok := config.TriggerMetadata["type"]; ok {
		logger.V(0).Info("trigger.metadata.type is deprecated in favor of trigger.metricType")
		if config.MetricType != "" {
			return fmt.Errorf("only one of trigger.metadata.type or trigger.metricType should be defined")
		}
		val = strings.ToLower(val)
		switch val {
		case avgString:
			meta.vType = v2.AverageValueMetricType
		case "global":
			meta.vType = v2.ValueMetricType
		default:
			return fmt.Errorf("type has to be global or average")
		}
	} else {
		metricType, err := GetMetricTargetType(config)
		if err != nil {
			return fmt.Errorf("error getting scaler metric type: %w", err)
		}
		meta.vType = metricType
	}

	meta.HpaMetricName = "datadogmetric@" + meta.DatadogMetricNamespace + ":" + meta.DatadogMetricName
	meta.DatadogMetricServiceURL = buildClusterAgentURL(meta.DatadogMetricsService, meta.DatadogNamespace, meta.DatadogMetricsServicePort)

	// Set UseFiller flag if metricUnavailableValue is explicitly configured
	if meta.FillValue != nil {
		meta.UseFiller = true
	}

	return nil
}

func newDatadogAPIClient(s *datadogMetadata) *datadog.APIClient {
	configuration := datadog.NewConfiguration()
	configuration.UserAgent = fmt.Sprintf("%s - KEDA/%s", configuration.UserAgent, version.Version)
	configuration.HTTPClient = kedautil.CreateHTTPClient(s.Timeout, false)
	apiClient := datadog.NewAPIClient(configuration)

	return apiClient
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

	httpClientTimeout := s.apiClient.GetConfig().HTTPClient.Timeout
	startTime := time.Now()
	resp, r, err := s.apiClient.MetricsApi.QueryMetrics(ctx, timeWindowFrom, timeWindowTo, s.metadata.Query) //nolint:bodyclose
	elapsed := time.Since(startTime)

	if r != nil {
		if r.StatusCode == 403 {
			return -1, fmt.Errorf("unauthorized to connect to Datadog. Check that your Datadog API and App keys are correct and that your App key has the correct permissions: %w", err)
		}

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
		// Check for general network timeouts
		if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
			if httpClientTimeout > 0 {
				return -1, fmt.Errorf("KEDA reached a network timeout while retrieving Datadog metrics (HTTP client timeout: %s, request took: %s: %w)", httpClientTimeout, elapsed, err)
			}
			return -1, fmt.Errorf("KEDA reached a network timeout while retrieving Datadog metrics (request took: %s: %w)", elapsed, err)
		}

		if errors.Is(err, context.DeadlineExceeded) {
			if deadline, ok := ctx.Deadline(); ok {
				now := time.Now()
				exceeded := now.Sub(deadline)
				return -1, fmt.Errorf("KEDA reached a context deadline while retrieving Datadog metrics (deadline was: %v, exceeded by: %v: %w)", deadline.Format(time.RFC3339), exceeded, err)
			}
			return -1, fmt.Errorf("KEDA reached a context deadline while retrieving Datadog metrics: %w", err)
		}
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
		return *s.metadata.FillValue, nil
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
			return *s.metadata.FillValue, nil
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

// getDatadogMetricValue retrieves metric value from Datadog Cluster Agent
func (s *datadogScaler) getDatadogMetricValue(req *http.Request) (float64, error) {
	resp, err := s.httpClient.Do(req)

	if err != nil {
		return 0, fmt.Errorf("error getting metric value: %w", err)
	}

	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		r := gjson.GetBytes(body, "message")
		errMessage := ""
		if r.Type == gjson.String {
			errMessage = r.String()
		}

		if resp.StatusCode == http.StatusUnprocessableEntity {
			if s.metadata.UseFiller {
				s.logger.V(1).Info("Datadog metric unavailable, using FillValue",
					"statusCode", resp.StatusCode,
					"fillValue", *s.metadata.FillValue,
					"message", errMessage)
				return *s.metadata.FillValue, nil
			}
		}

		// Return error if no FillValue configured
		if errMessage != "" {
			return 0, fmt.Errorf("error getting metric value (status %d): %s", resp.StatusCode, errMessage)
		}
		return 0, fmt.Errorf("error getting metric value: unexpected status code %d", resp.StatusCode)
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
