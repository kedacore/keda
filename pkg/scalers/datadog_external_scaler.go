package scalers

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	"github.com/kedacore/keda/v2/pkg/scalers/authentication"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

type datadogExternalScaler struct {
	metadata *datadogExternalMetadata
	logger   logr.Logger
}

type datadogExternalMetadata struct {

	// AuthParams
	datadogNamespace          string
	datadogMetricsService     string
	datadogMetricsServicePort int
	unsafeSsl                 bool

	// TriggerMetadata
	datadogMetricName     string
	targetValue           float64
	activationTargetValue float64
	fillValue             float64
	useFiller             bool
	vType                 v2.MetricTargetType

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

	_, err = newDatadogExternalConnection(ctx, meta, config, logger)
	if err != nil {
		return nil, fmt.Errorf("error establishing connection with Datadog Cluster Agent: %w", err)
	}
	return &datadogExternalScaler{
		metadata: meta,
		logger:   logger,
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

func buildClusterAgentURL(datadogMetricsService, datadogNamespace string, datadogMetricsServicePort int) string {

	return fmt.Sprintf("https://%s.%s:%d/apis/external.metrics.k8s.io/v1beta1/", datadogMetricsService, datadogNamespace, datadogMetricsServicePort)
}

// newDatadogConnection tests a connection to the Datadog Cluster Agent
func newDatadogExternalConnection(ctx context.Context, meta *datadogExternalMetadata, config *ScalerConfig, logger logr.Logger) (bool, error) {

	var req *http.Request
	var err error

	url := buildClusterAgentURL(meta.datadogMetricsService, meta.datadogNamespace, meta.datadogMetricsServicePort)

	logger.Info(fmt.Sprintf("URL: %s", url))

	tr := &http.Transport{}

	if meta.unsafeSsl {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	client := &http.Client{Transport: tr}

	// TODO: add TLS support
	switch {
	case meta.enableBearerAuth:
		req, err = http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return false, err
		}
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", meta.bearerToken))
		resp, err := client.Do(req)
		if err != nil {
			return false, err
		}
		logger.Info("Datadog Cluster Agent connection successful")
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)

		logger.Info(fmt.Sprintf("Response: %s", body))
	default:
		req, err = http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return false, err
		}
	}

	return true, nil
}

// No need to close connections
func (s *datadogExternalScaler) Close(context.Context) error {
	return nil
}

// getQueryResult returns result of the scaler query
func (s *datadogExternalScaler) getQueryResult(ctx context.Context) (float64, error) {

	return 0, nil
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

	return []external_metrics.ExternalMetricValue{}, false, nil
}
