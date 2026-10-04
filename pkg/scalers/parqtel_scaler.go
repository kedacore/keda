package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	url_pkg "net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/metricscollector"
	"github.com/kedacore/keda/v2/pkg/scalers/authentication"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	parqtelMetricName = "parqtel"

	// query types
	queryTypeInstant = "instant"
	queryTypeRange   = "range"

	// result aggregation modes
	aggFirst = "first"
	aggLast  = "last"
	aggSum   = "sum"
	aggMax   = "max"
	aggMin   = "min"
	aggAvg   = "avg"
	aggCount = "count"

	defaultRangeStep   = "60s"
	defaultRangeWindow = 5 * time.Minute
)

// parqtelScaler scales a workload based on the result of a PQL/PromQL query
// executed against a Parqtel instance's Prometheus-compatible query API.
type parqtelScaler struct {
	metricType         v2.MetricTargetType
	metadata           *parqtelMetadata
	httpClient         *http.Client
	logger             logr.Logger
	scalableObjectName string
	scalableObjectNS   string
	triggerName        string
	metricName         string
	resourceType       string
}

// parqtelMetadata holds the trigger configuration for the Parqtel scaler.
type parqtelMetadata struct {
	ParqtelAuth *authentication.Config `keda:"optional"`

	ServerAddress       string            `keda:"name=serverAddress,       order=triggerMetadata"`
	Query               string            `keda:"name=query,               order=triggerMetadata"`
	Threshold           float64           `keda:"name=threshold,           order=triggerMetadata"`
	ActivationThreshold float64           `keda:"name=activationThreshold, order=triggerMetadata, optional"`
	QueryType           string            `keda:"name=queryType,           order=triggerMetadata, default=instant, enum=instant;range"`
	RangeStart          string            `keda:"name=rangeStart,          order=triggerMetadata, optional"`
	RangeEnd            string            `keda:"name=rangeEnd,            order=triggerMetadata, optional"`
	RangeStep           string            `keda:"name=rangeStep,           order=triggerMetadata, optional"`
	ResultAggregation   string            `keda:"name=resultAggregation,   order=triggerMetadata, default=first, enum=first;last;sum;max;min;avg;count"`
	CustomHeaders       map[string]string `keda:"name=customHeaders,       order=triggerMetadata, optional"`
	QueryParameters     map[string]string `keda:"name=queryParameters,     order=triggerMetadata, optional"`
	IgnoreNullValues    bool              `keda:"name=ignoreNullValues,    order=triggerMetadata, default=true"`
	UnsafeSSL           bool              `keda:"name=unsafeSsl,           order=triggerMetadata, optional"`
	Timeout             time.Duration     `keda:"name=timeout,             order=triggerMetadata, optional"`

	triggerIndex int
}

// Validate restricts the supported auth modes and enforces query invariants.
func (m *parqtelMetadata) Validate() error {
	if m.ParqtelAuth == nil {
		m.ParqtelAuth = &authentication.Config{}
	}
	// Legacy bridge: enable basic auth for manifests that set username/password without authModes.
	if m.ParqtelAuth.Disabled() && m.ParqtelAuth.Username != "" {
		m.ParqtelAuth.Modes = []authentication.Type{authentication.BasicAuthType}
	}
	if err := m.ParqtelAuth.ValidateAllowed(
		authentication.BasicAuthType,
		authentication.BearerAuthType,
		authentication.CustomAuthType,
		authentication.TLSAuthType,
	); err != nil {
		return err
	}

	switch m.QueryType {
	case "", queryTypeInstant, queryTypeRange:
	default:
		return fmt.Errorf("unsupported queryType %q (allowed: %s, %s)", m.QueryType, queryTypeInstant, queryTypeRange)
	}

	switch m.ResultAggregation {
	case "", aggFirst, aggLast, aggSum, aggMax, aggMin, aggAvg, aggCount:
	default:
		return fmt.Errorf("unsupported resultAggregation %q (allowed: %s, %s, %s, %s, %s, %s, %s)",
			m.ResultAggregation, aggFirst, aggLast, aggSum, aggMax, aggMin, aggAvg, aggCount)
	}

	if m.QueryType == queryTypeRange && m.RangeStep != "" {
		if _, err := parseStepDuration(m.RangeStep); err != nil {
			return fmt.Errorf("invalid rangeStep %q: %w", m.RangeStep, err)
		}
	}
	return nil
}

// parqtelResultSeries is a single matched series in a Parqtel query response.
type parqtelResultSeries struct {
	Metric struct{} `json:"metric"`
	// Instant-vector sample: [timestamp, "value"]. Present for resultType=vector.
	Value []any `json:"value"`
	// Matrix sample list: [[timestamp, "value"], ...]. Present for resultType=matrix.
	Values [][]any `json:"values"`
}

// parqtelQueryResult mirrors the Prometheus-compatible response shape returned by
// Parqtel's /api/v1/query (vector) and /api/v1/query_range (matrix).
type parqtelQueryResult struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string                `json:"resultType"`
		Result     []parqtelResultSeries `json:"result"`
	} `json:"data"`
}

// NewParqtelScaler creates a new parqtelScaler.
func NewParqtelScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "parqtel_scaler")

	meta, err := parseParqtelMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing parqtel metadata: %w", err)
	}

	// handle HTTP client timeout
	httpClientTimeout := config.GlobalHTTPTimeout
	if meta.Timeout > 0 {
		httpClientTimeout = meta.Timeout
	}

	httpClient := kedautil.CreateHTTPClient(httpClientTimeout, meta.UnsafeSSL)

	// Set up a client-certificate (mTLS) transport when TLS auth is enabled.
	if meta.ParqtelAuth.EnabledTLS() {
		tlsConfig, err := meta.ParqtelAuth.NewTLSConfig(meta.UnsafeSSL)
		if err != nil {
			logger.V(1).Error(err, "init parqtel client http transport")
			return nil, err
		}
		httpClient.Transport = kedautil.CreateRTWithTLSConfig(tlsConfig)
	}

	return &parqtelScaler{
		metricType:         metricType,
		metadata:           meta,
		httpClient:         httpClient,
		logger:             logger,
		scalableObjectName: config.ScalableObjectName,
		scalableObjectNS:   config.ScalableObjectNamespace,
		triggerName:        config.TriggerName,
		metricName:         GenerateMetricNameWithIndex(meta.triggerIndex, kedautil.NormalizeString(parqtelMetricName)),
		resourceType:       config.ScalableObjectType,
	}, nil
}

func parseParqtelMetadata(config *scalersconfig.ScalerConfig) (*parqtelMetadata, error) {
	meta := &parqtelMetadata{triggerIndex: config.TriggerIndex}
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing parqtel metadata: %w", err)
	}
	return meta, nil
}

func (s *parqtelScaler) Close(context.Context) error {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}

func (s *parqtelScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(parqtelMetricName)),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.Threshold),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

// ExecuteQuery runs the configured Parqtel query and returns the reduced metric value.
func (s *parqtelScaler) ExecuteQuery(ctx context.Context) (float64, error) {
	queryURL, err := s.buildQueryURL()
	if err != nil {
		return -1, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, queryURL, nil)
	if err != nil {
		return -1, err
	}

	for headerName, headerValue := range s.metadata.CustomHeaders {
		req.Header.Add(headerName, headerValue)
	}

	// Apply the configured authentication.
	switch {
	case s.metadata.ParqtelAuth.EnabledBearerAuth():
		req.Header.Set("Authorization", s.metadata.ParqtelAuth.GetBearerToken())
	case s.metadata.ParqtelAuth.EnabledBasicAuth():
		req.SetBasicAuth(s.metadata.ParqtelAuth.Username, s.metadata.ParqtelAuth.Password)
	case s.metadata.ParqtelAuth.EnabledCustomAuth():
		req.Header.Set(s.metadata.ParqtelAuth.CustomAuthHeader, s.metadata.ParqtelAuth.CustomAuthValue)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		_, _ = io.Copy(io.Discard, resp.Body)
		err := fmt.Errorf("parqtel query api returned error. status: %d", resp.StatusCode)
		s.logger.Error(err, "parqtel query api returned error")
		return -1, err
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return -1, err
	}

	return s.parseQueryResult(b)
}

func (s *parqtelScaler) parseQueryResult(body []byte) (float64, error) {
	var result parqtelQueryResult
	if err := json.Unmarshal(body, &result); err != nil {
		return -1, err
	}

	if result.Status != "" && result.Status != "success" {
		return -1, fmt.Errorf("parqtel query returned status %q", result.Status)
	}

	var values []float64
	for _, series := range result.Data.Result {
		raw, ok := seriesValue(series)
		if !ok {
			continue
		}
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			s.logger.Error(err, "error converting parqtel value", "value", raw)
			return -1, err
		}
		if math.IsInf(v, 0) || math.IsNaN(v) {
			if s.metadata.IgnoreNullValues {
				continue
			}
			err := fmt.Errorf("parqtel query %q returned %f", s.metadata.Query, v)
			s.logger.Error(err, "error converting parqtel value")
			return -1, err
		}
		values = append(values, v)
	}

	if len(values) == 0 {
		metricscollector.RecordEmptyUpstreamResponse(
			s.scalableObjectNS, s.scalableObjectName, s.triggerName, s.metricName, s.resourceType, s.metadata.IgnoreNullValues,
		)
		if s.metadata.IgnoreNullValues {
			return 0, nil
		}
		return -1, fmt.Errorf("parqtel query %q returned no usable values", s.metadata.Query)
	}

	return aggregateValues(values, s.metadata.ResultAggregation)
}

// GetMetricsAndActivity returns the current Parqtel query value and whether the
// workload should be kept active.
func (s *parqtelScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	val, err := s.ExecuteQuery(ctx)
	if err != nil {
		s.logger.Error(err, "error executing parqtel query")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, val)
	return []external_metrics.ExternalMetricValue{metric}, val > s.metadata.ActivationThreshold, nil
}

// buildQueryURL assembles the Parqtel query endpoint URL for the configured query type.
func (s *parqtelScaler) buildQueryURL() (string, error) {
	queryEscaped := url_pkg.QueryEscape(s.metadata.Query)

	var base string
	if s.metadata.QueryType == queryTypeRange {
		nowSecs := float64(time.Now().Unix())
		startSecs := nowSecs - defaultRangeWindow.Seconds()
		endSecs := nowSecs

		if s.metadata.RangeStart != "" {
			v, err := parseTimestampToUnixSeconds(s.metadata.RangeStart)
			if err != nil {
				return "", err
			}
			startSecs = v
		}
		if s.metadata.RangeEnd != "" {
			v, err := parseTimestampToUnixSeconds(s.metadata.RangeEnd)
			if err != nil {
				return "", err
			}
			endSecs = v
		}

		step := s.metadata.RangeStep
		if step == "" {
			step = defaultRangeStep
		}
		if _, err := parseStepDuration(step); err != nil {
			return "", err
		}

		base = fmt.Sprintf("%s/api/v1/query_range?query=%s&start=%s&end=%s&step=%s",
			s.metadata.ServerAddress,
			queryEscaped,
			strconv.FormatFloat(startSecs, 'f', -1, 64),
			strconv.FormatFloat(endSecs, 'f', -1, 64),
			url_pkg.QueryEscape(step),
		)
	} else {
		base = fmt.Sprintf("%s/api/v1/query?query=%s", s.metadata.ServerAddress, queryEscaped)
	}

	for queryParameterKey, queryParameterValue := range s.metadata.QueryParameters {
		queryParameterKeyEscaped := url_pkg.QueryEscape(queryParameterKey)
		queryParameterValueEscaped := url_pkg.QueryEscape(queryParameterValue)
		base = fmt.Sprintf("%s&%s=%s", base, queryParameterKeyEscaped, queryParameterValueEscaped)
	}

	return base, nil
}

// seriesValue extracts the raw string sample value from a series, preferring the
// instant-vector value and falling back to the most recent range sample.
func seriesValue(series parqtelResultSeries) (string, bool) {
	if len(series.Value) >= 2 {
		if str, ok := series.Value[1].(string); ok {
			return str, true
		}
	}
	if n := len(series.Values); n > 0 {
		if last := series.Values[n-1]; len(last) >= 2 {
			if str, ok := last[1].(string); ok {
				return str, true
			}
		}
	}
	return "", false
}

// aggregateValues reduces a set of per-series values to a single value using the
// configured aggregation mode.
func aggregateValues(values []float64, mode string) (float64, error) {
	switch mode {
	case "", aggFirst:
		return values[0], nil
	case aggLast:
		return values[len(values)-1], nil
	case aggSum:
		sum := 0.0
		for _, v := range values {
			sum += v
		}
		return sum, nil
	case aggMax:
		m := values[0]
		for _, v := range values[1:] {
			if v > m {
				m = v
			}
		}
		return m, nil
	case aggMin:
		m := values[0]
		for _, v := range values[1:] {
			if v < m {
				m = v
			}
		}
		return m, nil
	case aggAvg:
		sum := 0.0
		for _, v := range values {
			sum += v
		}
		return sum / float64(len(values)), nil
	case aggCount:
		return float64(len(values)), nil
	default:
		return -1, fmt.Errorf("unsupported resultAggregation %q", mode)
	}
}

// parseStepDuration parses a step duration in the format accepted by Parqtel: a plain
// number of seconds ("60") or a duration with a s/m/h/d unit ("60s", "5m").
func parseStepDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty step")
	}
	if secs, err := strconv.ParseFloat(s, 64); err == nil {
		if secs <= 0 || math.IsInf(secs, 0) || math.IsNaN(secs) {
			return 0, fmt.Errorf("invalid step %q", s)
		}
		return time.Duration(secs * float64(time.Second)), nil
	}
	if len(s) >= 2 {
		if val, err := strconv.ParseFloat(s[:len(s)-1], 64); err == nil {
			var mult time.Duration
			switch s[len(s)-1:] {
			case "s":
				mult = time.Second
			case "m":
				mult = time.Minute
			case "h":
				mult = time.Hour
			case "d":
				mult = 24 * time.Hour
			default:
				return 0, fmt.Errorf("invalid step unit in %q", s)
			}
			if val <= 0 {
				return 0, fmt.Errorf("invalid step %q", s)
			}
			return time.Duration(val * float64(mult)), nil
		}
	}
	return 0, fmt.Errorf("invalid step %q", s)
}

// parseTimestampToUnixSeconds accepts either a unix-seconds number or an RFC3339 timestamp.
func parseTimestampToUnixSeconds(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty timestamp")
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return float64(t.Unix()), nil
	}
	return 0, fmt.Errorf("invalid timestamp %q (want unix seconds or RFC3339)", s)
}
