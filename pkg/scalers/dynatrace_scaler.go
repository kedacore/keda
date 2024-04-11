package scalers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type dynatraceScaler struct {
	metricType v2.MetricTargetType
	metadata   *dynatraceMetadata
	httpClient *http.Client
	logger     logr.Logger
}

type dynatraceMetadata struct {
	apiURL              string
	apiKey              string
	metricsSelector     string
	threshold           float64
	activationThreshold float64
	triggerIndex        int
}

// Model of relevant part of Dynatrace's Metric Data Points API Response
// as per https://docs.dynatrace.com/docs/dynatrace-api/environment-api/metric-v2/get-data-points#definition--MetricData
type dynatraceResponse struct {
	Result []struct {
		Data []struct {
			Values []int `json:"values"`
		} `json:"data"`
	} `json:"response"`
}

func NewDynatraceScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "dynatrace_scaler")

	meta, err := parseDynatraceMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing dynatrace metadata: %w", err)
	}

	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, false)

	logMsg := fmt.Sprintf("Initializing Dynatrace Scaler (API URL: %s)", meta.apiURL)

	logger.Info(logMsg)

	return &dynatraceScaler{
		metricType: metricType,
		metadata:   meta,
		httpClient: httpClient,
		logger:     logger}, nil
}

func parseDynatraceMetadata(config *scalersconfig.ScalerConfig) (*dynatraceMetadata, error) {
	meta := dynatraceMetadata{}
	var err error

	apiURL, err := GetFromAuthOrMeta(config, "apiUrl")
	if err != nil {
		return nil, err
	}
	meta.apiURL = apiURL

	apiKey, err := GetFromAuthOrMeta(config, "apiKey")
	if err != nil {
		return nil, err
	}
	meta.apiKey = apiKey

	if val, ok := config.TriggerMetadata["metricsSelector"]; ok && val != "" {
		meta.metricsSelector = val
	} else {
		return nil, fmt.Errorf("no metricsSelector given")
	}

	if val, ok := config.TriggerMetadata["threshold"]; ok && val != "" {
		t, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing threshold")
		}
		meta.threshold = t
	} else {
		if config.AsMetricSource {
			meta.threshold = 0
		} else {
			return nil, fmt.Errorf("missing threshold value")
		}
	}

	meta.activationThreshold = 0
	if val, ok := config.TriggerMetadata["activationThreshold"]; ok {
		activationThreshold, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("queryValue parsing error %w", err)
		}
		meta.activationThreshold = activationThreshold
	}

	meta.triggerIndex = config.TriggerIndex
	return &meta, nil
}

func (s *dynatraceScaler) Close(context.Context) error {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}

// Validate that response object contains the minimum expected structure
// as per https://docs.dynatrace.com/docs/dynatrace-api/environment-api/metric-v2/get-data-points#definition--MetricData
func validateDynatraceResponse(response *dynatraceResponse) error {
	if len(response.Result) == 0 {
		return errors.New("dynatrace response does not contain any results")
	}
	if len(response.Result[0].Data) == 0 {
		return errors.New("dynatrace response does not contain any metric series")
	}
	if len(response.Result[0].Data[0].Values) == 0 {
		return errors.New("dynatrace response does not contain any values for the metric series")
	}
	return nil
}

func (s *dynatraceScaler) GetMetricValue(ctx context.Context) (float64, error) {
	/*
	 * Build request
	 */
	var req *http.Request
	var err error

	req, err = http.NewRequestWithContext(ctx, "GET", s.metadata.apiURL, nil)
	if err != nil {
		return 0, err
	}

	// Authentication header as per https://docs.dynatrace.com/docs/dynatrace-api/basics/dynatrace-api-authentication#authenticate
	req.Header.Add("Authorization", fmt.Sprintf("Api-Token %s", s.metadata.apiKey))

	/*
	 * Execute request
	 */
	r, err := s.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("%s: api returned %d", r.Request.URL.Path, r.StatusCode)
		return 0, errors.New(msg)
	}

	/*
	 * Parse response
	 */
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return 0, err
	}
	var dynatraceResponse *dynatraceResponse
	err = json.Unmarshal(b, &dynatraceResponse)
	if err != nil {
		return -1, fmt.Errorf("unable to parse Dynatrace Metric Data Points API response: %w", err)
	}

	err = validateDynatraceResponse(dynatraceResponse)
	if err != nil {
		return 0, err
	}

	return float64(dynatraceResponse.Result[0].Data[0].Values[0]), nil
}

func (s *dynatraceScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	val, err := s.GetMetricValue(ctx)

	if err != nil {
		s.logger.Error(err, "error executing Dynatrace query")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, val)

	return []external_metrics.ExternalMetricValue{metric}, val > s.metadata.activationThreshold, nil
}

func (s *dynatraceScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString("dynatrace")),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.threshold),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}
