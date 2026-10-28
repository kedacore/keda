package scalers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	honeycombDefaultAPIHost = "https://api.honeycomb.io"
	// Honeycomb runs a query result for at most 10 seconds; leave headroom for the API calls around it.
	honeycombQueryResultTimeout = 15 * time.Second
	honeycombMaxPollDelay       = 2 * time.Second
	honeycombMaxErrorBodyBytes  = 1024
)

// honeycombInitialPollDelay is a variable so tests can poll faster.
var honeycombInitialPollDelay = 250 * time.Millisecond

type honeycombScaler struct {
	metricType v2.MetricTargetType
	metadata   *honeycombMetadata
	httpClient *http.Client
	logger     logr.Logger
	now        func() time.Time

	// mu serializes queries so that concurrent callers (the operator and the metrics server)
	// share one query result instead of each spending the Honeycomb rate limit.
	mu        sync.Mutex
	queryID   string
	lastRun   time.Time
	lastValue float64
}

type honeycombMetadata struct {
	APIKey              string        `keda:"name=apiKey,              order=authParams"`
	APIHost             string        `keda:"name=apiHost,             order=triggerMetadata, default=https://api.honeycomb.io"`
	Dataset             string        `keda:"name=dataset,             order=triggerMetadata"`
	Query               string        `keda:"name=query,               order=triggerMetadata"`
	ResultField         string        `keda:"name=resultField,         order=triggerMetadata, optional"`
	Threshold           float64       `keda:"name=threshold,           order=triggerMetadata, optional"`
	ActivationThreshold float64       `keda:"name=activationThreshold, order=triggerMetadata, default=0"`
	Timeout             time.Duration `keda:"name=timeout,             order=triggerMetadata, optional"`

	triggerIndex int
}

func (m *honeycombMetadata) Validate() error {
	u, err := url.Parse(m.APIHost)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("apiHost must be a URL such as %s, got %q", honeycombDefaultAPIHost, m.APIHost)
	}
	m.APIHost = strings.TrimSuffix(m.APIHost, "/")

	var query map[string]any
	if err := json.Unmarshal([]byte(m.Query), &query); err != nil {
		return fmt.Errorf("query must be a Honeycomb query JSON object: %w", err)
	}
	return nil
}

// NewHoneycombScaler creates a scaler that scales on the result of a Honeycomb query.
func NewHoneycombScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, err := parseHoneycombMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing Honeycomb metadata: %w", err)
	}

	timeout := meta.Timeout
	if timeout == 0 {
		timeout = config.GlobalHTTPTimeout
	}

	return &honeycombScaler{
		metricType: metricType,
		metadata:   meta,
		httpClient: kedautil.CreateHTTPClient(timeout, false),
		logger:     InitializeLogger(config, "honeycomb_scaler"),
		now:        time.Now,
	}, nil
}

func parseHoneycombMetadata(config *scalersconfig.ScalerConfig) (*honeycombMetadata, error) {
	meta := &honeycombMetadata{triggerIndex: config.TriggerIndex}
	if err := config.TypedConfig(meta); err != nil {
		return nil, err
	}
	if !config.AsMetricSource && meta.Threshold <= 0 {
		return nil, errors.New("threshold must be greater than 0")
	}
	return meta, nil
}

func (s *honeycombScaler) Close(context.Context) error {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}

func (s *honeycombScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString("honeycomb-"+s.metadata.Dataset)),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.Threshold),
	}
	return []v2.MetricSpec{{External: externalMetric, Type: externalMetricType}}
}

func (s *honeycombScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	value, err := s.getValue(ctx)
	if err != nil {
		s.logger.Error(err, "error getting Honeycomb query result", "dataset", s.metadata.Dataset)
		return []external_metrics.ExternalMetricValue{}, false, err
	}
	metric := GenerateMetricInMili(metricName, value)
	return []external_metrics.ExternalMetricValue{metric}, value > s.metadata.ActivationThreshold, nil
}

// getValue returns the query value, running the query at most once per wall-clock minute.
// For time ranges of up to 6 hours, Honeycomb answers repeated runs of a query within the
// same minute with the same result, while each run still counts against the rate limit on
// creating query results (10 per minute by default).
func (s *honeycombScaler) getValue(ctx context.Context) (float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	if !s.lastRun.IsZero() && s.lastRun.Truncate(time.Minute).Equal(now.Truncate(time.Minute)) {
		return s.lastValue, nil
	}

	value, err := s.runQuery(ctx)
	if err != nil {
		return 0, err
	}
	s.lastRun, s.lastValue = now, value
	return value, nil
}

type honeycombQueryResult struct {
	ID       string `json:"id"`
	Complete bool   `json:"complete"`
	Error    string `json:"error"`
	Data     struct {
		Results []struct {
			Data map[string]any `json:"data"`
		} `json:"results"`
	} `json:"data"`
}

func (s *honeycombScaler) runQuery(ctx context.Context) (float64, error) {
	ctx, cancel := context.WithTimeout(ctx, honeycombQueryResultTimeout)
	defer cancel()

	dataset := url.PathEscape(s.metadata.Dataset)

	// One query spec serves every run: its relative time_range is evaluated each time a
	// query result is created. Honeycomb also deduplicates identical query specs.
	if s.queryID == "" {
		var created struct {
			ID string `json:"id"`
		}
		if err := s.doRequest(ctx, http.MethodPost, "/1/queries/"+dataset, json.RawMessage(s.metadata.Query), http.StatusOK, &created); err != nil {
			return 0, fmt.Errorf("creating query: %w", err)
		}
		if created.ID == "" {
			return 0, errors.New("creating query: response has no query id")
		}
		s.queryID = created.ID
	}

	var result honeycombQueryResult
	createBody := map[string]any{"query_id": s.queryID, "disable_series": true}
	if err := s.doRequest(ctx, http.MethodPost, "/1/query_results/"+dataset, createBody, http.StatusCreated, &result); err != nil {
		return 0, fmt.Errorf("creating query result: %w", err)
	}
	if result.ID == "" {
		return 0, errors.New("creating query result: response has no query result id")
	}

	// The create response never carries result data, so always fetch the result.
	resultPath := "/1/query_results/" + dataset + "/" + url.PathEscape(result.ID)
	delay := honeycombInitialPollDelay
	for {
		select {
		case <-ctx.Done():
			return 0, fmt.Errorf("waiting for query result %s: %w", result.ID, ctx.Err())
		case <-time.After(delay):
		}
		delay = min(delay*2, honeycombMaxPollDelay)

		result = honeycombQueryResult{}
		if err := s.doRequest(ctx, http.MethodGet, resultPath, nil, http.StatusOK, &result); err != nil {
			return 0, fmt.Errorf("getting query result: %w", err)
		}
		if result.Complete {
			break
		}
	}

	if result.Error != "" {
		return 0, fmt.Errorf("query result %s failed: %s", result.ID, result.Error)
	}
	if len(result.Data.Results) == 0 {
		// No matching events, which is the normal state for a workload that can scale to zero.
		return 0, nil
	}
	return honeycombResultValue(result.Data.Results[0].Data, s.metadata.ResultField)
}

// honeycombResultValue reads the metric value from a result row. Without a field name the
// row must contain exactly one numeric value, so the choice never depends on map order.
func honeycombResultValue(row map[string]any, field string) (float64, error) {
	if field != "" {
		v, ok := row[field]
		if !ok {
			return 0, fmt.Errorf("resultField %q not found in query result, available fields: %s", field, strings.Join(slices.Sorted(maps.Keys(row)), ", "))
		}
		f, ok := v.(float64)
		if !ok {
			return 0, fmt.Errorf("resultField %q is not numeric: %v", field, v)
		}
		return f, nil
	}

	var numeric []string
	for k, v := range row {
		if _, ok := v.(float64); ok {
			numeric = append(numeric, k)
		}
	}
	if len(numeric) != 1 {
		slices.Sort(numeric)
		return 0, fmt.Errorf("query result has %d numeric fields (%s); set resultField to choose one", len(numeric), strings.Join(numeric, ", "))
	}
	return row[numeric[0]].(float64), nil
}

func (s *honeycombScaler) doRequest(ctx context.Context, method, path string, body any, wantStatus int, out any) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, s.metadata.APIHost+path, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("X-Honeycomb-Team", s.metadata.APIKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != wantStatus {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, honeycombMaxErrorBodyBytes))
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return fmt.Errorf("unauthorized (401), check the API key: %s", msg)
		case http.StatusForbidden:
			return fmt.Errorf("forbidden (403), the Query Data API needs a Honeycomb Enterprise plan and a Configuration API key with the Run Queries and Manage Queries and Columns permissions: %s", msg)
		case http.StatusTooManyRequests:
			return fmt.Errorf("rate limited (429): %s", msg)
		default:
			return fmt.Errorf("unexpected status %s: %s", resp.Status, msg)
		}
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
