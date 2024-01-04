package scalers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
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
}

type datadogMetadata struct {

	// AuthParams Cluster Agent Proxy
	datadogNamespace          string
	datadogMetricsService     string
	datadogMetricsServicePort int
	unsafeSsl                 bool

	// client certification Cluster Agent Proyx
	enableTLS bool
	cert      string
	key       string
	ca        string

	// bearer auth Cluster Agent Proxy
	enableBearerAuth bool
	bearerToken      string

	// TriggerMetadata Cluster Agent Proxy
	datadogMetricServiceUrl string
	datadogMetricName       string
	datadogMetricNamespace  string
	activationTargetValue   float64

	// AuthParams Datadog API
	apiKey      string
	appKey      string
	datadogSite string

	// TriggerMetadata Datadog API
	query                    string
	queryAggegrator          string
	activationQueryValue     float64
	age                      int
	timeWindowOffset         int
	lastAvailablePointOffset int

	// TriggerMetadata Common
	hpaMetricName string
	fillValue     float64
	targetValue   float64
	useFiller     bool
	vType         v2.MetricTargetType
}

const maxString = "max"
const avgString = "average"

var filter *regexp.Regexp

func init() {
	filter = regexp.MustCompile(`.*\{.*\}.*`)
}

// NewDatadogScaler creates a new Datadog scaler
func NewDatadogScaler(ctx context.Context, config *scalersconfig.ScalerConfig) (Scaler, error) {
	logger := InitializeLogger(config, "datadog_scaler")

	var useClusterAgentProxy bool
	var meta *datadogMetadata
	var err error
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
		httpClient = kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, meta.unsafeSsl)

		if meta.enableTLS || len(meta.ca) > 0 {
			config, err := kedautil.NewTLSConfig(meta.cert, meta.key, meta.ca, meta.unsafeSsl)
			if err != nil {
				return nil, err
			}
			httpClient.Transport = kedautil.CreateHTTPTransportWithTLSConfig(config)
		}

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
	meta := datadogMetadata{}

	if val, ok := config.TriggerMetadata["age"]; ok {
		age, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("age parsing error %w", err)
		}
		meta.age = age

		if age < 0 {
			return nil, fmt.Errorf("age should not be smaller than 0 seconds")
		}
		if age < 60 {
			logger.Info("selecting a window smaller than 60 seconds can cause Datadog not finding a metric value for the query")
		}
	} else {
		meta.age = 90 // Default window 90 seconds
	}

	if val, ok := config.TriggerMetadata["timeWindowOffset"]; ok {
		timeWindowOffset, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("timeWindowOffset parsing error %w", err)
		}
		if timeWindowOffset < 0 {
			return nil, fmt.Errorf("timeWindowOffset should not be smaller than 0 seconds")
		}
		meta.timeWindowOffset = timeWindowOffset
	} else {
		meta.timeWindowOffset = 0 // Default delay 0 seconds
	}

	if val, ok := config.TriggerMetadata["lastAvailablePointOffset"]; ok {
		lastAvailablePointOffset, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("lastAvailablePointOffset parsing error %w", err)
		}

		if lastAvailablePointOffset < 0 {
			return nil, fmt.Errorf("lastAvailablePointOffset should not be smaller than 0")
		}
		meta.lastAvailablePointOffset = lastAvailablePointOffset
	} else {
		meta.lastAvailablePointOffset = 0 // Default use the last point
	}

	if val, ok := config.TriggerMetadata["query"]; ok {
		_, err := parseDatadogQuery(val)

		if err != nil {
			return nil, fmt.Errorf("error in query: %w", err)
		}
		meta.query = val
	} else {
		return nil, fmt.Errorf("no query given")
	}

	if val, ok := config.TriggerMetadata["targetValue"]; ok {
		targetValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("targetValue parsing error %w", err)
		}
		meta.targetValue = targetValue
	} else if val, ok := config.TriggerMetadata["queryValue"]; ok {
		targetValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("queryValue parsing error %w", err)
		}
		meta.targetValue = targetValue
	} else {
		if config.AsMetricSource {
			meta.targetValue = 0
		} else {
			return nil, fmt.Errorf("no targetValue or queryValue given")
		}
	}

	if val, ok := config.TriggerMetadata["queryAggregator"]; ok && val != "" {
		queryAggregator := strings.ToLower(val)
		switch queryAggregator {
		case avgString, maxString:
			meta.queryAggegrator = queryAggregator
		default:
			return nil, fmt.Errorf("queryAggregator value %s has to be one of '%s, %s'", queryAggregator, avgString, maxString)
		}
	} else {
		meta.queryAggegrator = ""
	}

	meta.activationQueryValue = 0
	if val, ok := config.TriggerMetadata["activationQueryValue"]; ok {
		activationQueryValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("queryValue parsing error %w", err)
		}
		meta.activationQueryValue = activationQueryValue
	}

	if val, ok := config.TriggerMetadata["metricUnavailableValue"]; ok {
		fillValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("metricUnavailableValue parsing error %w", err)
		}
		meta.fillValue = fillValue
		meta.useFiller = true
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

	if val, ok := config.AuthParams["apiKey"]; ok {
		meta.apiKey = val
	} else {
		return nil, fmt.Errorf("no api key given")
	}

	if val, ok := config.AuthParams["appKey"]; ok {
		meta.appKey = val
	} else {
		return nil, fmt.Errorf("no app key given")
	}

	siteVal := "datadoghq.com"

	if val, ok := config.AuthParams["datadogSite"]; ok && val != "" {
		siteVal = val
	}

	meta.datadogSite = siteVal

	hpaMetricName := meta.query[0:strings.Index(meta.query, "{")]
	meta.hpaMetricName = GenerateMetricNameWithIndex(config.ScalerIndex, kedautil.NormalizeString(fmt.Sprintf("datadog-%s", hpaMetricName)))

	return &meta, nil
}

func parseDatadogClusterAgentMetadata(config *ScalerConfig, logger logr.Logger) (*datadogMetadata, error) {
	meta := datadogMetadata{}

	if val, ok := config.AuthParams["datadogNamespace"]; ok {
		meta.datadogNamespace = val
	} else {
		return nil, fmt.Errorf("no datadogNamespace key given")
	}

	if val, ok := config.AuthParams["datadogMetricsService"]; ok {
		meta.datadogMetricsService = val
	} else {
		meta.datadogMetricsService = "datadog-cluster-agent-metrics-api"
	}

	if val, ok := config.AuthParams["datadogMetricsServicePort"]; ok {
		port, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("datadogMetricServicePort parsing error %w", err)
		}
		meta.datadogMetricsServicePort = port
	} else {
		meta.datadogMetricsServicePort = 8443
	}

	meta.datadogMetricServiceUrl = buildClusterAgentURL(meta.datadogMetricsService, meta.datadogNamespace, meta.datadogMetricsServicePort)

	meta.unsafeSsl = false
	if val, ok := config.AuthParams["unsafeSsl"]; ok {
		unsafeSsl, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("error parsing unsafeSsl: %w", err)
		}
		meta.unsafeSsl = unsafeSsl
	}

	if val, ok := config.TriggerMetadata["datadogMetricName"]; ok {
		meta.datadogMetricName = val
	} else {
		return nil, fmt.Errorf("no datadogMetricName key given")
	}

	if val, ok := config.TriggerMetadata["datadogMetricNamespace"]; ok {
		meta.datadogMetricNamespace = val
	} else {
		return nil, fmt.Errorf("no datadogMetricNamespace key given")
	}

	meta.hpaMetricName = "datadogmetric@" + meta.datadogMetricNamespace + ":" + meta.datadogMetricName

	if val, ok := config.TriggerMetadata["targetValue"]; ok {
		targetValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("targetValue parsing error %w", err)
		}
		meta.targetValue = targetValue
	} else {
		if config.AsMetricSource {
			meta.targetValue = 0
		} else {
			return nil, fmt.Errorf("no targetValue given")
		}
	}

	meta.activationTargetValue = 0
	if val, ok := config.TriggerMetadata["activationTargetValue"]; ok {
		activationTargetValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("activationTargetValue parsing error %w", err)
		}
		meta.activationTargetValue = activationTargetValue
	}

	if val, ok := config.TriggerMetadata["metricUnavailableValue"]; ok {
		fillValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("metricUnavailableValue parsing error %w", err)
		}
		meta.fillValue = fillValue
		meta.useFiller = true
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

	authMode, ok := config.TriggerMetadata["authMode"]
	// no authMode specified
	if !ok {
		return &meta, nil
	}

	authType := authentication.Type(strings.TrimSpace(authMode))
	switch authType {
	case authentication.TLSAuthType:
		if len(config.AuthParams["ca"]) == 0 {
			return nil, fmt.Errorf("no ca given")
		}

		if len(config.AuthParams["cert"]) == 0 {
			return nil, fmt.Errorf("no cert given")
		}
		meta.cert = config.AuthParams["cert"]

		if len(config.AuthParams["key"]) == 0 {
			return nil, fmt.Errorf("no key given")
		}

		meta.key = config.AuthParams["key"]
		meta.enableTLS = true
	case authentication.BearerAuthType:
		if len(config.AuthParams["token"]) == 0 {
			return nil, errors.New("no token provided")
		}

		meta.bearerToken = config.AuthParams["token"]
		meta.enableBearerAuth = true
	default:
		return nil, fmt.Errorf("err incorrect value for authMode is given: %s", authMode)
	}

	if len(config.AuthParams["ca"]) > 0 {
		meta.ca = config.AuthParams["ca"]
	}

	return &meta, nil
}

// newDatadogAPIConnection tests a connection to the Datadog API
func newDatadogAPIConnection(ctx context.Context, meta *datadogMetadata, config *scalersconfig.ScalerConfig) (*datadog.APIClient, error) {
	ctx = context.WithValue(
		ctx,
		datadog.ContextAPIKeys,
		map[string]datadog.APIKey{
			"apiKeyAuth": {
				Key: meta.apiKey,
			},
			"appKeyAuth": {
				Key: meta.appKey,
			},
		},
	)

	ctx = context.WithValue(ctx,
		datadog.ContextServerVariables,
		map[string]string{
			"site": meta.datadogSite,
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
				Key: s.metadata.apiKey,
			},
			"appKeyAuth": {
				Key: s.metadata.appKey,
			},
		},
	)

	ctx = context.WithValue(ctx,
		datadog.ContextServerVariables,
		map[string]string{
			"site": s.metadata.datadogSite,
		})

	timeWindowTo := time.Now().Unix() - int64(s.metadata.timeWindowOffset)
	timeWindowFrom := timeWindowTo - int64(s.metadata.age)
	resp, r, err := s.apiClient.MetricsApi.QueryMetrics(ctx, timeWindowFrom, timeWindowTo, s.metadata.query) //nolint:bodyclose

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
		if !s.metadata.useFiller {
			return 0, fmt.Errorf("no Datadog metrics returned for the given time window")
		}
		return s.metadata.fillValue, nil
	}

	// Require queryAggregator be set explicitly for multi-query
	if len(series) > 1 && s.metadata.queryAggegrator == "" {
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
		if index < s.metadata.lastAvailablePointOffset {
			return 0, fmt.Errorf("index is smaller than the lastAvailablePointOffset")
		}
		index -= s.metadata.lastAvailablePointOffset

		if len(points) == 0 || len(points[index]) < 2 || points[index][1] == nil {
			if !s.metadata.useFiller {
				return 0, fmt.Errorf("no Datadog metrics returned for the given time window")
			}
			return s.metadata.fillValue, nil
		}
		// Return the last point from the series
		results[i] = *points[index][1]
	}

	switch s.metadata.queryAggegrator {
	case avgString:
		return AvgFloatFromSlice(results), nil
	default:
		// Aggregate Results - default Max value:
		return MaxFloatFromSlice(results), nil
	}
}

func (s *datadogScaler) getDatadogMetricValue(req *http.Request) (float64, error) {
	resp, err := s.httpClient.Do(req)

	if err != nil {
		return 0, fmt.Errorf("error getting metric value: %w", err)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	s.logger.Info(fmt.Sprintf("Response: %s", body))

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

	// TODO: add TLS support
	switch {
	case s.metadata.enableBearerAuth:
		req, err = http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", s.metadata.bearerToken))
		if err != nil {
			return nil, err
		}

		s.logger.Info(fmt.Sprintf("Request correctly created"))
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
			Name: s.metadata.hpaMetricName,
		},
		Target: GetMetricTargetMili(s.metadata.vType, s.metadata.targetValue),
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

	if s.useClusterAgentProxy {
		url := buildMetricURL(s.metadata.datadogMetricServiceUrl, s.metadata.datadogMetricNamespace, s.metadata.hpaMetricName)

		req, err := s.getDatadogClusterAgentHTTPRequest(ctx, url)
		if (err != nil) || (req == nil) {
			return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error generating http request: %w", err)
		}

		num, err := s.getDatadogMetricValue(req)
		if err != nil {
			return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error getting metric value: %w", err)
		}

		metric = GenerateMetricInMili(metricName, num)
	} else {
		num, err := s.getQueryResult(ctx)
		if err != nil {
			s.logger.Error(err, "error getting metrics from Datadog")
			return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error getting metrics from Datadog: %w", err)
		}

		metric = GenerateMetricInMili(metricName, num)
	}

	return []external_metrics.ExternalMetricValue{metric}, num > s.metadata.activationQueryValue, nil
}

// MaxFloatFromSlice finds the largest value in a slice of floats
func MaxFloatFromSlice(results []float64) float64 {
	max := results[0]
	for _, result := range results {
		if result > max {
			max = result
		}
	}
	return max
}

// AvgFloatFromSlice finds the average value in a slice of floats
func AvgFloatFromSlice(results []float64) float64 {
	total := 0.0
	for _, result := range results {
		total += result
	}
	return total / float64(len(results))
}
