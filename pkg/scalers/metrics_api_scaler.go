package scalers

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/go-logr/logr"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/prometheus/promql/parser"
	"github.com/tidwall/gjson"
	"golang.org/x/sync/semaphore"
	"gopkg.in/yaml.v3"
	v2 "k8s.io/api/autoscaling/v2"
	discoveryV1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kedacore/keda/v2/pkg/scalers/authentication"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type metricsAPIScaler struct {
	metricType v2.MetricTargetType
	metadata   *metricsAPIScalerMetadata
	httpClient *http.Client
	logger     logr.Logger
	kubeClient client.Client
}

type metricsAPIScalerMetadata struct {
	TargetValue                       float64         `keda:"name=targetValue,order=triggerMetadata,optional"`
	ActivationTargetValue             float64         `keda:"name=activationTargetValue,order=triggerMetadata,default=0"`
	URL                               string          `keda:"name=url,order=triggerMetadata"`
	Format                            APIFormat       `keda:"name=format,order=triggerMetadata,default=json,enum=prometheus;json;xml;yaml"`
	ValueLocation                     string          `keda:"name=valueLocation,order=triggerMetadata"`
	UnsafeSsl                         bool            `keda:"name=unsafeSsl,order=triggerMetadata,default=false"`
	AggregateFromKubeServiceEndpoints bool            `keda:"name=aggregateFromKubeServiceEndpoints,order=triggerMetadata,default=false"`
	AggregationType                   AggregationType `keda:"name=aggregationType,order=triggerMetadata,default=average,enum=average;sum;max;min"`
	// Authentication parameters for connecting to the metrics API
	MetricsAPIAuth *authentication.Config `keda:"optional"`

	triggerIndex int
}

const (
	methodValueQuery           = "query"
	valueLocationWrongErrorMsg = "valueLocation must point to value of type number or a string representing a Quantity got: '%s'"
)

const secureHTTPScheme = "https"

type APIFormat string

// Options for APIFormat:
const (
	PrometheusFormat APIFormat = "prometheus"
	JSONFormat       APIFormat = "json"
	XMLFormat        APIFormat = "xml"
	YAMLFormat       APIFormat = "yaml"
)

type AggregationType string

// Options for APIFormat:
const (
	AverageAggregationType AggregationType = "average"
	SumAggregationType     AggregationType = "sum"
	MaxAggregationType     AggregationType = "max"
	MinAggregationType     AggregationType = "min"
)

// NewMetricsAPIScaler creates a new HTTP scaler
func NewMetricsAPIScaler(config *scalersconfig.ScalerConfig, kubeClient client.Client) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, err := parseMetricsAPIMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing metric API metadata: %w", err)
	}

	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, meta.UnsafeSsl)

	// Handle TLS configuration with authentication config
	if meta.MetricsAPIAuth != nil && meta.MetricsAPIAuth.EnabledTLS() {
		tlsConfig, err := kedautil.NewTLSConfig(meta.MetricsAPIAuth.Cert, meta.MetricsAPIAuth.Key, meta.MetricsAPIAuth.CA, meta.UnsafeSsl)
		if err != nil {
			return nil, err
		}
		httpClient.Transport = kedautil.CreateHTTPTransportWithTLSConfig(tlsConfig)
	}

	return &metricsAPIScaler{
		metricType: metricType,
		metadata:   meta,
		httpClient: httpClient,
		kubeClient: kubeClient,
		logger:     InitializeLogger(config, "metrics_api_scaler"),
	}, nil
}

func parseMetricsAPIMetadata(config *scalersconfig.ScalerConfig) (*metricsAPIScalerMetadata, error) {
	meta := &metricsAPIScalerMetadata{}
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing metrics API metadata: %w", err)
	}

	meta.triggerIndex = config.TriggerIndex

	// Special validation for targetValue when not used as metric source
	if meta.TargetValue == 0 && !config.AsMetricSource {
		return nil, fmt.Errorf("no targetValue given in metadata")
	}

	return meta, nil
}

// GetValueFromResponse uses provided valueLocation to access the numeric value in provided body using the format specified.
func GetValueFromResponse(body []byte, valueLocation string, format APIFormat) (float64, error) {
	switch format {
	case PrometheusFormat:
		return getValueFromPrometheusResponse(body, valueLocation)
	case JSONFormat:
		return getValueFromJSONResponse(body, valueLocation)
	case XMLFormat:
		return getValueFromXMLResponse(body, valueLocation)
	case YAMLFormat:
		return getValueFromYAMLResponse(body, valueLocation)
	}

	return 0, fmt.Errorf("format %s not supported", format)
}

// getValueFromPrometheusResponse uses provided valueLocation to access the numeric value in provided body
func getValueFromPrometheusResponse(body []byte, valueLocation string) (float64, error) {
	matchers, err := parser.ParseMetricSelector(valueLocation)
	if err != nil {
		return 0, err
	}
	metricName := ""
	for _, v := range matchers {
		if v.Name == "__name__" {
			metricName = v.Value
		}
	}

	// Ensure EOL
	bodyStr := strings.ReplaceAll(string(body), "\r\n", "\n")

	// Check if newline is present
	if len(bodyStr) > 0 && !strings.HasSuffix(bodyStr, "\n") {
		bodyStr += "\n"
	}

	reader := strings.NewReader(bodyStr)
	familiesParser := expfmt.TextParser{}
	families, err := familiesParser.TextToMetricFamilies(reader)
	if err != nil {
		return 0, fmt.Errorf("prometheus format parsing error: %w", err)
	}

	family, ok := families[metricName]
	if !ok {
		return 0, fmt.Errorf("metric '%s' not found", metricName)
	}

	metrics := family.GetMetric()
	for _, metric := range metrics {
		labels := metric.GetLabel()
		match := true
		for _, matcher := range matchers {
			matcherFound := false
			if matcher == nil {
				continue
			}
			// The name has been already validated,
			// so we can skip it and check the other labels
			if matcher.Name == "__name__" {
				continue
			}
			for _, label := range labels {
				if *label.Name == matcher.Name &&
					*label.Value == matcher.Value {
					matcherFound = true
				}
			}
			if !matcherFound {
				match = false
			}
		}
		if match {
			untyped := metric.GetUntyped()
			if untyped != nil && untyped.Value != nil {
				return *untyped.Value, nil
			}
			counter := metric.GetCounter()
			if counter != nil && counter.Value != nil {
				return *counter.Value, nil
			}
			gauge := metric.GetGauge()
			if gauge != nil && gauge.Value != nil {
				return *gauge.Value, nil
			}
		}
	}

	return 0, fmt.Errorf("value %s not found", valueLocation)
}

// getValueFromJSONResponse uses provided valueLocation to access the numeric value in provided body using GJSON
func getValueFromJSONResponse(body []byte, valueLocation string) (float64, error) {
	r := gjson.GetBytes(body, valueLocation)
	if r.Type == gjson.String {
		v, err := resource.ParseQuantity(r.String())
		if err != nil {
			return 0, fmt.Errorf(valueLocationWrongErrorMsg, r.String())
		}
		return v.AsApproximateFloat64(), nil
	}
	if r.Type != gjson.Number {
		return 0, fmt.Errorf(valueLocationWrongErrorMsg, r.Type.String())
	}
	return r.Num, nil
}

// getValueFromXMLResponse uses provided valueLocation to access the numeric value in provided body
func getValueFromXMLResponse(body []byte, valueLocation string) (float64, error) {
	var xmlMap map[string]interface{}
	err := xml.Unmarshal(body, &xmlMap)
	if err != nil {
		return 0, err
	}

	path, err := kedautil.GetValueByPath(xmlMap, valueLocation)
	if err != nil {
		return 0, err
	}

	switch v := path.(type) {
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case float64:
		return v, nil
	case string:
		r, err := resource.ParseQuantity(v)
		if err != nil {
			return 0, fmt.Errorf(valueLocationWrongErrorMsg, v)
		}
		return r.AsApproximateFloat64(), nil
	default:
		return 0, fmt.Errorf(valueLocationWrongErrorMsg, v)
	}
}

// getValueFromYAMLResponse uses provided valueLocation to access the numeric value in provided body
// using generic ketautil.GetValueByPath
func getValueFromYAMLResponse(body []byte, valueLocation string) (float64, error) {
	var yamlMap map[string]interface{}
	err := yaml.Unmarshal(body, &yamlMap)
	if err != nil {
		return 0, err
	}

	path, err := kedautil.GetValueByPath(yamlMap, valueLocation)
	if err != nil {
		return 0, err
	}

	switch v := path.(type) {
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case float64:
		return v, nil
	case string:
		r, err := resource.ParseQuantity(v)
		if err != nil {
			return 0, fmt.Errorf(valueLocationWrongErrorMsg, v)
		}
		return r.AsApproximateFloat64(), nil
	default:
		return 0, fmt.Errorf(valueLocationWrongErrorMsg, v)
	}
}

func (s *metricsAPIScaler) getEndpointsUrlsFromServiceURL(ctx context.Context, serviceURL string) (endpointUrls []string, err error) {
	// parse service name from s.meta.url
	url, err := neturl.Parse(serviceURL)
	if err != nil {
		s.logger.Error(err, "Failed parsing url for metrics API")
		return nil, err
	}

	splittedHost := strings.Split(url.Host, ".")
	if len(splittedHost) < 2 {
		return nil, fmt.Errorf("invalid hostname %s : expected at least 2 elements, first being service name and second being the namespace", url.Host)
	}
	serviceName := splittedHost[0]
	namespace := splittedHost[1]
	podPort := url.Port()
	// infer port from service scheme when not set explicitly
	if podPort == "" {
		if url.Scheme == secureHTTPScheme {
			podPort = "443"
		} else {
			podPort = "80"
		}
	}
	// get service serviceEndpointsSlices
	serviceEndpointsSlices := &discoveryV1.EndpointSliceList{}
	serviceNameSelector := labels.NewSelector()
	serviceNameSelector.Matches(labels.Set(map[string]string{
		discoveryV1.LabelServiceName: serviceName,
	}))
	err = s.kubeClient.List(ctx, serviceEndpointsSlices, &client.ListOptions{
		LabelSelector: serviceNameSelector,
		Namespace:     namespace,
	})
	if err != nil {
		return nil, err
	}
	for _, endpointSlice := range serviceEndpointsSlices.Items {
		for _, eps := range endpointSlice.Endpoints {
			foundPort := ""
			for _, port := range endpointSlice.Ports {
				if port.Port != nil && strconv.Itoa(int(*port.Port)) == podPort {
					foundPort = fmt.Sprintf(":%d", *port.Port)
					break
				}
			}
			if foundPort == "" {
				s.logger.V(1).Info(fmt.Sprintf("Warning : could not find port %s in endpoint slice for service %s.%s definition. Will infer port from %s scheme", podPort, serviceName, namespace, url.Scheme))
			}
			for _, address := range eps.Addresses {
				if eps.NodeName != nil {
					endpointUrls = append(endpointUrls, fmt.Sprintf("%s://%s%s%s", url.Scheme, address, foundPort, url.Path))
				}
			}
		}
	}
	return endpointUrls, err
}

func (s *metricsAPIScaler) getMetricValue(ctx context.Context) (float64, error) {
	// if we wish to aggregate metric from a kubernetes service then we need to query each endpoint behind the service
	if s.metadata.AggregateFromKubeServiceEndpoints {
		endpointsUrls, err := s.getEndpointsUrlsFromServiceURL(ctx, s.metadata.URL)
		if err != nil {
			return 0, fmt.Errorf("failed to get kubernetes endpoints urls from configured service URL")
		}
		if len(endpointsUrls) == 0 {
			return 0, fmt.Errorf("no endpoints URLs were given for the service name")
		}
		return s.aggregateMetricsFromMultipleEndpoints(ctx, endpointsUrls)
	}
	// get single/unaggregated metric
	metric, err := s.getMetricValueFromURL(ctx, nil)
	if err == nil {
		s.logger.V(1).Info(fmt.Sprintf("fetched single metric from metrics API url : %s. Value is %v\n", s.metadata.URL, metric))
	}
	return metric, err
}

func (s *metricsAPIScaler) aggregateMetricsFromMultipleEndpoints(ctx context.Context, endpointsUrls []string) (float64, error) {
	// call s.getMetricValueFromURL() for each endpointsUrls in parallel goroutines (maximum 5 at a time) and sum them up
	const maxGoroutines = 5
	var mu sync.Mutex
	sem := semaphore.NewWeighted(maxGoroutines)
	expectedNbMetrics := len(endpointsUrls)
	nbErrors := 0
	var err error
	var firstMetricEncountered bool
	var aggregation float64
	for _, endpointURL := range endpointsUrls {
		if err := sem.Acquire(ctx, 1); err != nil {
			s.logger.Error(err, "Failed to acquire semaphore")
			continue
		}
		go func(url string) {
			defer sem.Release(1)
			metric, err := s.getMetricValueFromURL(ctx, &endpointURL)

			if err != nil {
				s.logger.V(1).Info(fmt.Sprintf("Error fetching metric for %s: %v\n", url, err))
				// we will ignore metric for computing aggregation when encountering error : decrease expectedNbMetrics
				mu.Lock()
				expectedNbMetrics--
				nbErrors++
				mu.Unlock()
			} else {
				mu.Lock()
				switch s.metadata.AggregationType {
				case MinAggregationType:
					if !firstMetricEncountered || metric < aggregation {
						firstMetricEncountered = true
						aggregation = metric
					}
				case MaxAggregationType:
					if !firstMetricEncountered || metric > aggregation {
						firstMetricEncountered = true
						aggregation = metric
					}
				default:
					// sum metrics if we are not looking for min or max value
					aggregation += metric
				}
				mu.Unlock()
			}
		}(endpointURL)
	}

	if nbErrors > 0 && nbErrors == len(endpointsUrls) {
		err = fmt.Errorf("could not get any metric successfully from the %d provided endpoints", len(endpointsUrls))
	}
	if s.metadata.AggregationType == AverageAggregationType {
		aggregation /= float64(expectedNbMetrics)
	}
	s.logger.V(1).Info(fmt.Sprintf("fetched %d metrics out of %d endpoints from kubernetes service : %s is %v\n", expectedNbMetrics, len(endpointsUrls), s.metadata.AggregationType, aggregation))
	return aggregation, err
}

func (s *metricsAPIScaler) getMetricValueFromURL(ctx context.Context, url *string) (float64, error) {
	request, err := getMetricAPIServerRequest(ctx, s.metadata, url)
	if err != nil {
		return 0, err
	}

	r, err := s.httpClient.Do(request)
	if err != nil {
		return 0, err
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("%s: api returned %d", r.Request.URL.Path, r.StatusCode)
		return 0, errors.New(msg)
	}

	b, err := io.ReadAll(r.Body)
	if err != nil {
		return 0, err
	}
	v, err := GetValueFromResponse(b, s.metadata.ValueLocation, s.metadata.Format)
	if err != nil {
		return 0, err
	}
	return v, nil
}

// Close does nothing in case of metricsAPIScaler
func (s *metricsAPIScaler) Close(context.Context) error {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *metricsAPIScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("metric-api-%s", s.metadata.ValueLocation))),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.TargetValue),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *metricsAPIScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	val, err := s.getMetricValue(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error requesting metrics endpoint: %w", err)
	}

	metric := GenerateMetricInMili(metricName, val)

	return []external_metrics.ExternalMetricValue{metric}, val > s.metadata.ActivationTargetValue, nil
}

func getMetricAPIServerRequest(ctx context.Context, meta *metricsAPIScalerMetadata, url *string) (*http.Request, error) {
	var requestURL string
	if url == nil {
		url = &meta.URL
	}
	// Handle API Key as query parameter if needed
	if meta.MetricsAPIAuth != nil && meta.MetricsAPIAuth.EnabledAPIKeyAuth() && meta.MetricsAPIAuth.Method == methodValueQuery {
		url, _ := neturl.Parse(*url)
		queryString := url.Query()
		if meta.MetricsAPIAuth.KeyParamName == "" {
			queryString.Set("api_key", meta.MetricsAPIAuth.APIKey)
		} else {
			queryString.Set(meta.MetricsAPIAuth.KeyParamName, meta.MetricsAPIAuth.APIKey)
		}
		url.RawQuery = queryString.Encode()
		requestURL = url.String()
	} else {
		requestURL = *url
	}

	// Create the request
	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, err
	}

	// Add API Key as header if needed
	if meta.MetricsAPIAuth != nil && meta.MetricsAPIAuth.EnabledAPIKeyAuth() && meta.MetricsAPIAuth.Method != methodValueQuery {
		if meta.MetricsAPIAuth.KeyParamName == "" {
			req.Header.Add("X-API-KEY", meta.MetricsAPIAuth.APIKey)
		} else {
			req.Header.Add(meta.MetricsAPIAuth.KeyParamName, meta.MetricsAPIAuth.APIKey)
		}
	}

	// Add Basic Auth if enabled
	if meta.MetricsAPIAuth != nil && meta.MetricsAPIAuth.EnabledBasicAuth() {
		req.SetBasicAuth(meta.MetricsAPIAuth.Username, meta.MetricsAPIAuth.Password)
	}

	// Add Bearer token if enabled
	if meta.MetricsAPIAuth != nil && meta.MetricsAPIAuth.EnabledBearerAuth() {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", meta.MetricsAPIAuth.BearerToken))
	}

	return req, nil
}
