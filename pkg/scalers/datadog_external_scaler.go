package scalers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	"github.com/kedacore/keda/v2/pkg/scalers/authentication"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
	"github.com/tidwall/gjson"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

type datadogExternalScaler struct {
	metadata   *datadogExternalMetadata
	httpClient *http.Client
	logger     logr.Logger
}

type datadogExternalMetadata struct {

	// AuthParams
	datadogNamespace          string
	datadogMetricsService     string
	datadogMetricsServicePort int
	unsafeSsl                 bool

	// TriggerMetadata
	datadogMetricServiceUrl string
	datadogMetricName       string
	datadogMetricNamespace  string
	targetValue             float64
	activationTargetValue   float64
	fillValue               float64
	useFiller               bool
	vType                   v2.MetricTargetType

	// client certification
	enableTLS bool
	cert      string
	key       string
	ca        string

	// bearer
	enableBearerAuth bool
	bearerToken      string
}

// NewDatadogScaler creates a new Datadog External scaler
func NewDatadogExternalScaler(ctx context.Context, config *ScalerConfig) (Scaler, error) {
	logger := InitializeLogger(config, "datadog_external_scaler")

	meta, err := parseDatadogExternalMetadata(config, logger)
	if err != nil {
		return nil, fmt.Errorf("error parsing Datadog metadata: %w", err)
	}

	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, meta.unsafeSsl)

	if meta.enableTLS || len(meta.ca) > 0 {
		config, err := kedautil.NewTLSConfig(meta.cert, meta.key, meta.ca, meta.unsafeSsl)
		if err != nil {
			return nil, err
		}
		httpClient.Transport = kedautil.CreateHTTPTransportWithTLSConfig(config)
	}

	return &datadogExternalScaler{
		metadata:   meta,
		httpClient: httpClient,
		logger:     logger,
	}, nil
}

func parseDatadogExternalMetadata(config *ScalerConfig, logger logr.Logger) (*datadogExternalMetadata, error) {
	meta := datadogExternalMetadata{}

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

// buildClusterAgentURL builds the URL for the Cluster Agent Metrics API service
func buildClusterAgentURL(datadogMetricsService, datadogNamespace string, datadogMetricsServicePort int) string {

	return fmt.Sprintf("https://%s.%s:%d/apis/external.metrics.k8s.io/v1beta1", datadogMetricsService, datadogNamespace, datadogMetricsServicePort)
}

// buildMetricURL builds the URL for the Datadog metric
func buildMetricURL(datadogClusterAgentURL, datadogMetricNamespace, datadogMetricName string) string {
	return fmt.Sprintf("%s/namespaces/%s/%s", datadogClusterAgentURL, datadogMetricNamespace, datadogMetricName)
}

func (s *datadogExternalScaler) getDatadogMetricValue(req *http.Request) (float64, error) {
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

func (s *datadogExternalScaler) getDatadogExternalHTTPRequest(ctx context.Context, url string) (*http.Request, error) {

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

// No need to close connections
func (s *datadogExternalScaler) Close(context.Context) error {
	return nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *datadogExternalScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: s.metadata.datadogMetricName,
		},
		Target: GetMetricTargetMili(s.metadata.vType, s.metadata.targetValue),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *datadogExternalScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {

	url := buildMetricURL(s.metadata.datadogMetricServiceUrl, s.metadata.datadogMetricNamespace, s.metadata.datadogMetricName)

	s.logger.Info(fmt.Sprintf("URL: %s", url))

	req, err := s.getDatadogExternalHTTPRequest(ctx, url)
	if (err != nil) || (req == nil) {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error generating http request: %w", err)
	}

	num, err := s.getDatadogMetricValue(req)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error getting metric value: %w", err)
	}

	metric := GenerateMetricInMili(metricName, num)
	return []external_metrics.ExternalMetricValue{metric}, num > s.metadata.activationTargetValue, nil
}
