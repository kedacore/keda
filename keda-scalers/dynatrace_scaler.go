package scalers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"strings"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	dynatraceMetricDataPointsAPI = "api/v2/metrics/query"
)

type dynatraceScaler struct {
	metricType v2.MetricTargetType
	metadata   *dynatraceMetadata
	httpClient *http.Client
	logger     logr.Logger
}

type dynatraceMetadata struct {
	Host                string  `keda:"name=host, order=triggerMetadata;authParams"`
	Token               string  `keda:"name=token, order=authParams"`
	MetricSelector      string  `keda:"name=metricSelector, order=triggerMetadata"`
	FromTimestamp       string  `keda:"name=from, order=triggerMetadata, default=now-2h, optional"`
	Threshold           float64 `keda:"name=threshold, order=triggerMetadata"`
	ActivationThreshold float64 `keda:"name=activationThreshold, order=triggerMetadata, optional"`
	TriggerIndex        int
}

// Model of relevant part of Dynatrace's Metric Data Points API Response
// as per https://docs.dynatrace.com/docs/dynatrace-api/environment-api/metric-v2/get-data-points#definition--MetricData
type dynatraceResponse struct {
	Result []struct {
		Data []struct {
			Values []float64 `json:"values"`
		} `json:"data"`
	} `json:"result"`
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

	logMsg := fmt.Sprintf("Initializing Dynatrace Scaler (Host: %s)", meta.Host)

	logger.Info(logMsg)

	return &dynatraceScaler{
		metricType: metricType,
		metadata:   meta,
		httpClient: httpClient,
		logger:     logger}, nil
}

func parseDynatraceMetadata(config *scalersconfig.ScalerConfig) (*dynatraceMetadata, error) {
	meta := dynatraceMetadata{}

	meta.TriggerIndex = config.TriggerIndex
	if err := config.TypedConfig(&meta); err != nil {
		return nil, fmt.Errorf("error parsing dynatrace metadata: %w", err)
	}
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

	// Append host information to appropriate API endpoint
	// Trailing slashes are removed from provided host information to avoid double slashes in the URL
	dynatraceAPIURL := fmt.Sprintf("%s/%s", strings.TrimRight(s.metadata.Host, "/"), dynatraceMetricDataPointsAPI)

	// Add query parameters to the URL
	url, _ := neturl.Parse(dynatraceAPIURL)
	queryString := url.Query()
	queryString.Set("metricSelector", s.metadata.MetricSelector)
	queryString.Set("from", s.metadata.FromTimestamp)
	url.RawQuery = queryString.Encode()

	req, err = http.NewRequestWithContext(ctx, "GET", url.String(), nil)
	if err != nil {
		return 0, err
	}

	// Authentication header as per https://docs.dynatrace.com/docs/dynatrace-api/basics/dynatrace-api-authentication#authenticate
	req.Header.Add("Authorization", fmt.Sprintf("Api-Token %s", s.metadata.Token))

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

	return dynatraceResponse.Result[0].Data[0].Values[0], nil
}

func (s *dynatraceScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	val, err := s.GetMetricValue(ctx)

	if err != nil {
		s.logger.Error(err, "error executing Dynatrace query")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, val)

	return []external_metrics.ExternalMetricValue{metric}, val > s.metadata.ActivationThreshold, nil
}

func (s *dynatraceScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.TriggerIndex, kedautil.NormalizeString("dynatrace")),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.Threshold),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}
