package scalers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	dynatraceMetricDataPointsAPI = "api/v2/metrics/query"
	dynatraceDQLAPI              = "platform/storage/query/v1/query"
	dynatraceRunningState        = "RUNNING"
	dynatraceSucceededState      = "SUCCEEDED"
)

type dynatraceScaler struct {
	metricType          v2.MetricTargetType
	metadata            *dynatraceMetadata
	httpClient          *http.Client
	logger              logr.Logger
	queryRequestPayload []byte
}

type dynatraceMetadata struct {
	Host                string  `keda:"name=host, order=triggerMetadata;authParams"`
	Token               string  `keda:"name=token, order=authParams"`
	MetricSelector      string  `keda:"name=metricSelector, order=triggerMetadata, optional"`
	DQLQuery            string  `keda:"name=query, order=triggerMetadata, optional"`
	FromTimestamp       string  `keda:"name=from, order=triggerMetadata, optional"`
	Threshold           float64 `keda:"name=threshold, order=triggerMetadata"`
	ActivationThreshold float64 `keda:"name=activationThreshold, order=triggerMetadata, optional"`
	TriggerIndex        int
}

func (meta *dynatraceMetadata) Validate() error {
	if meta.DQLQuery == "" && meta.MetricSelector == "" {
		return errors.New("either 'metricSelector' or 'query' must be provided")
	}
	if meta.DQLQuery != "" && meta.MetricSelector != "" {
		return errors.New("'metricSelector' and 'query' are mutually exclusive; only one can be provided")
	}
	if meta.MetricSelector != "" && meta.FromTimestamp == "" {
		meta.FromTimestamp = "now-2h"
	}
	if meta.DQLQuery != "" && meta.FromTimestamp != "" {
		return errors.New("'from' can't be used with query, set time range in the DQL query itself")
	}
	return nil
}

// Model of relevant part of Dynatrace's Metric Data Points API Response
// as per https://docs.dynatrace.com/docs/dynatrace-api/environment-api/metric-v2/get-data-points#definition--MetricData
type dynatraceMetricsResponse struct {
	Result []struct {
		Data []struct {
			Values []float64 `json:"values"`
		} `json:"data"`
	} `json:"result"`
}

type dynatraceExecuteQueryResponse struct {
	State        string `json:"state"`
	RequestToken string `json:"requestToken"`
}

type dynatraceQueryResponse struct {
	State  string `json:"state"`
	Result struct {
		Records []struct {
			R float64 `json:"r"`
		} `json:"records"`
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

	queryRequestPayload := []byte{}
	if meta.DQLQuery != "" {
		queryRequest := struct {
			Query               string `json:"query"`
			FetchTimeoutSeconds int    `json:"fetchTimeoutSeconds"`
		}{
			Query:               meta.DQLQuery,
			FetchTimeoutSeconds: 10,
		}
		queryRequestPayload, err = json.Marshal(queryRequest)
		if err != nil {
			return nil, fmt.Errorf("error caching DQL query: %w", err)
		}
	}

	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, false)

	logMsg := fmt.Sprintf("Initializing Dynatrace Scaler (Host: %s)", meta.Host)

	logger.Info(logMsg)

	return &dynatraceScaler{
		metricType:          metricType,
		metadata:            meta,
		httpClient:          httpClient,
		queryRequestPayload: queryRequestPayload,
		logger:              logger}, nil
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
func validateDynatraceResponse(response *dynatraceMetricsResponse) error {
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
	dynatraceURL, _ := url.Parse(dynatraceAPIURL)
	queryString := dynatraceURL.Query()
	queryString.Set("metricSelector", s.metadata.MetricSelector)
	queryString.Set("from", s.metadata.FromTimestamp)
	dynatraceURL.RawQuery = queryString.Encode()

	req, err = http.NewRequestWithContext(ctx, "GET", dynatraceURL.String(), nil)
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
	var dynatraceResponse *dynatraceMetricsResponse
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

func (s *dynatraceScaler) GetQueryValue(ctx context.Context) (float64, error) {
	requestToken, err := s.executeDQL(ctx)
	if err != nil {
		return 0, err
	}

	dynatraceAPIURLResult := fmt.Sprintf("%s/%s:poll?request-token=%s", strings.TrimRight(s.metadata.Host, "/"), dynatraceDQLAPI, url.QueryEscape(requestToken))
	const maxAttempts = 5
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		val, retry, err := s.pollDQLResult(ctx, dynatraceAPIURLResult)
		if err != nil {
			return 0, err
		}
		if !retry {
			return val, nil
		}
		time.Sleep(time.Second)
	}
	return 0, fmt.Errorf("DQL query did not complete within %d attempts", maxAttempts)
}

func (s *dynatraceScaler) executeDQL(ctx context.Context) (string, error) {
	var req *http.Request
	var err error

	dynatraceAPIURLExecute := fmt.Sprintf("%s/%s:execute", strings.TrimRight(s.metadata.Host, "/"), dynatraceDQLAPI)

	req, err = http.NewRequestWithContext(ctx, "POST", dynatraceAPIURLExecute, bytes.NewBuffer(s.queryRequestPayload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", s.metadata.Token))

	r, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusAccepted {
		msg := fmt.Sprintf("%s: api returned %d", r.Request.URL.Path, r.StatusCode)
		return "", errors.New(msg)
	}

	b, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}
	var dynatraceResponse *dynatraceExecuteQueryResponse
	err = json.Unmarshal(b, &dynatraceResponse)
	if err != nil {
		return "", fmt.Errorf("error starting DQL query: %w", err)
	}
	if dynatraceResponse.State == dynatraceRunningState || dynatraceResponse.State == dynatraceSucceededState {
		return dynatraceResponse.RequestToken, nil
	}
	return "", fmt.Errorf("error starting DQL query: unknown state %s", dynatraceResponse.State)
}

func (s *dynatraceScaler) pollDQLResult(ctx context.Context, url string) (float64, bool, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return -1, false, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", s.metadata.Token))

	r, err := s.httpClient.Do(req)
	if err != nil {
		return -1, false, err
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("%s: api returned %d", r.Request.URL.Path, r.StatusCode)
		return -1, false, errors.New(msg)
	}

	b, err := io.ReadAll(r.Body)
	if err != nil {
		return -1, false, err
	}
	var dynatraceResponse *dynatraceQueryResponse
	err = json.Unmarshal(b, &dynatraceResponse)
	if err != nil {
		return -1, false, fmt.Errorf("error parsing DQL response: %w", err)
	}
	if dynatraceResponse.State == dynatraceRunningState {
		return -1, true, nil
	}
	if dynatraceResponse.State == dynatraceSucceededState {
		if len(dynatraceResponse.Result.Records) > 0 {
			return dynatraceResponse.Result.Records[0].R, false, nil
		}
		return -1, false, errors.New("error executing DQL query: empty result")
	}
	return -1, false, fmt.Errorf("error executing DQL query: unknown state: %s", dynatraceResponse.State)
}

func (s *dynatraceScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	var val float64
	var err error
	if s.metadata.MetricSelector != "" {
		val, err = s.GetMetricValue(ctx)
	} else {
		val, err = s.GetQueryValue(ctx)
	}

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
