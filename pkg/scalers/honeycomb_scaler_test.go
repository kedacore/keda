package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

const honeycombTestQuery = `{"calculations":[{"op":"COUNT"}],"time_range":120}`

func honeycombTestMetadata(overrides map[string]string) map[string]string {
	m := map[string]string{"dataset": "my.dataset", "query": honeycombTestQuery, "threshold": "10"}
	for k, v := range overrides {
		if v == "" {
			delete(m, k)
		} else {
			m[k] = v
		}
	}
	return m
}

func TestHoneycombParseMetadata(t *testing.T) {
	auth := map[string]string{"apiKey": "key"}
	tests := []struct {
		desc           string
		metadata       map[string]string
		authParams     map[string]string
		asMetricSource bool
		wantErr        string
		wantHost       string
	}{
		{desc: "minimal", metadata: honeycombTestMetadata(nil), authParams: auth, wantHost: "https://api.honeycomb.io"},
		{desc: "EU host with trailing slash", metadata: honeycombTestMetadata(map[string]string{"apiHost": "https://api.eu1.honeycomb.io/"}), authParams: auth, wantHost: "https://api.eu1.honeycomb.io"},
		{desc: "apiKey in trigger metadata is not read", metadata: honeycombTestMetadata(map[string]string{"apiKey": "key"}), authParams: map[string]string{}, wantErr: "apiKey"},
		{desc: "missing dataset", metadata: honeycombTestMetadata(map[string]string{"dataset": ""}), authParams: auth, wantErr: "dataset"},
		{desc: "missing query", metadata: honeycombTestMetadata(map[string]string{"query": ""}), authParams: auth, wantErr: "query"},
		{desc: "query is not JSON", metadata: honeycombTestMetadata(map[string]string{"query": "COUNT where service.name = x"}), authParams: auth, wantErr: "query must be"},
		{desc: "query is a JSON array", metadata: honeycombTestMetadata(map[string]string{"query": `[{"op":"COUNT"}]`}), authParams: auth, wantErr: "query must be"},
		{desc: "apiHost without scheme", metadata: honeycombTestMetadata(map[string]string{"apiHost": "api.honeycomb.io"}), authParams: auth, wantErr: "apiHost"},
		{desc: "missing threshold", metadata: honeycombTestMetadata(map[string]string{"threshold": ""}), authParams: auth, wantErr: "threshold"},
		{desc: "missing threshold as metric source", metadata: honeycombTestMetadata(map[string]string{"threshold": ""}), authParams: auth, asMetricSource: true, wantHost: "https://api.honeycomb.io"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			meta, err := parseHoneycombMetadata(&scalersconfig.ScalerConfig{
				TriggerMetadata: tt.metadata,
				AuthParams:      tt.authParams,
				AsMetricSource:  tt.asMetricSource,
			})
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantHost, meta.APIHost)
		})
	}
}

func TestHoneycombGetMetricSpecForScaling(t *testing.T) {
	s, err := NewHoneycombScaler(&scalersconfig.ScalerConfig{
		TriggerMetadata: honeycombTestMetadata(nil),
		AuthParams:      map[string]string{"apiKey": "key"},
		TriggerIndex:    1,
	})
	require.NoError(t, err)
	spec := s.GetMetricSpecForScaling(context.Background())
	assert.Equal(t, "s1-honeycomb-my-dataset", spec[0].External.Metric.Name)
}

// fakeHoneycomb serves the three Query Data API endpoints. Each created query result
// completes on its second GET with the next entry of results.
type fakeHoneycomb struct {
	t       *testing.T
	results []string // JSON for the GET response fields "data" and "error"
	status  int      // if set, returned by POST /1/query_results

	mu            sync.Mutex
	queriesMade   int
	resultsMade   int
	getsForResult map[string]int
}

func (f *fakeHoneycomb) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	assert.Equal(f.t, "test-key", r.Header.Get("X-Honeycomb-Team"))

	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/1/queries/my.dataset":
		var q map[string]any
		require.NoError(f.t, json.NewDecoder(r.Body).Decode(&q))
		assert.Equal(f.t, float64(120), q["time_range"])
		f.queriesMade++
		f.writeJSON(w, map[string]any{"id": "q1"})
	case r.Method == http.MethodPost && r.URL.Path == "/1/query_results/my.dataset":
		if f.status != 0 {
			w.WriteHeader(f.status)
			f.writeJSON(w, map[string]any{"error": "nope"})
			return
		}
		var body map[string]any
		require.NoError(f.t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(f.t, "q1", body["query_id"])
		f.resultsMade++
		w.WriteHeader(http.StatusCreated)
		f.writeJSON(w, map[string]any{"id": fmt.Sprintf("r%d", f.resultsMade), "complete": false})
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/1/query_results/my.dataset/r"):
		id := strings.TrimPrefix(r.URL.Path, "/1/query_results/my.dataset/")
		f.getsForResult[id]++
		if f.getsForResult[id] < 2 {
			f.writeJSON(w, map[string]any{"id": id, "complete": false})
			return
		}
		n, err := strconv.Atoi(strings.TrimPrefix(id, "r"))
		require.NoError(f.t, err)
		f.writeJSON(w, json.RawMessage(fmt.Sprintf(`{"id":%q,"complete":true,%s}`, id, f.results[n-1])))
	default:
		f.t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}
}

func (f *fakeHoneycomb) writeJSON(w http.ResponseWriter, v any) {
	require.NoError(f.t, json.NewEncoder(w).Encode(v))
}

func newHoneycombTestScaler(t *testing.T, fake *fakeHoneycomb, metadata map[string]string) (*honeycombScaler, *time.Time) {
	fake.t = t
	fake.getsForResult = map[string]int{}
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)

	oldDelay := honeycombInitialPollDelay
	honeycombInitialPollDelay = time.Millisecond
	t.Cleanup(func() { honeycombInitialPollDelay = oldDelay })

	metadata["apiHost"] = server.URL
	s, err := NewHoneycombScaler(&scalersconfig.ScalerConfig{
		TriggerMetadata:   metadata,
		AuthParams:        map[string]string{"apiKey": "test-key"},
		GlobalHTTPTimeout: time.Second,
	})
	require.NoError(t, err)
	hs := s.(*honeycombScaler)
	now := time.Date(2026, 1, 1, 12, 0, 10, 0, time.UTC)
	hs.now = func() time.Time { return now }
	return hs, &now
}

func TestHoneycombRunsQueryOncePerMinute(t *testing.T) {
	fake := &fakeHoneycomb{results: []string{
		`"data":{"results":[{"data":{"COUNT":42,"P99(duration_ms)":7.5}}]}`,
		`"data":{"results":[{"data":{"COUNT":43,"P99(duration_ms)":9.25}}]}`,
	}}
	s, now := newHoneycombTestScaler(t, fake, honeycombTestMetadata(map[string]string{"resultField": "P99(duration_ms)"}))
	ctx := context.Background()

	steps := []struct {
		at          time.Time
		wantValue   float64
		wantResults int
	}{
		{time.Date(2026, 1, 1, 12, 0, 10, 0, time.UTC), 7.5, 1},
		{time.Date(2026, 1, 1, 12, 0, 59, 0, time.UTC), 7.5, 1},
		{time.Date(2026, 1, 1, 12, 1, 0, 0, time.UTC), 9.25, 2},
	}
	for _, step := range steps {
		*now = step.at
		metrics, active, err := s.GetMetricsAndActivity(ctx, "s0-honeycomb")
		require.NoError(t, err)
		assert.InDelta(t, step.wantValue, metrics[0].Value.AsApproximateFloat64(), 0.001, "at %s", step.at)
		assert.True(t, active)
		assert.Equal(t, step.wantResults, fake.resultsMade, "query results created by %s", step.at)
	}
	assert.Equal(t, 1, fake.queriesMade, "the query spec is created once and reused")
}

func TestHoneycombQueryResults(t *testing.T) {
	tests := []struct {
		desc        string
		result      string
		resultField string
		status      int
		wantValue   float64
		wantActive  bool
		wantErr     string
	}{
		{desc: "no matching events is zero", result: `"data":{"results":[]}`, wantValue: 0, wantActive: false},
		{desc: "single numeric field next to a breakdown", result: `"data":{"results":[{"data":{"service.name":"checkout","COUNT":12}}]}`, wantValue: 12, wantActive: true},
		{desc: "two numeric fields need resultField", result: `"data":{"results":[{"data":{"COUNT":1,"AVG(duration_ms)":2}}]}`, wantErr: "set resultField"},
		{desc: "resultField missing from row", result: `"data":{"results":[{"data":{"COUNT":1}}]}`, resultField: "SUM(bytes)", wantErr: `"SUM(bytes)" not found`},
		{desc: "failed query", result: `"error":"query timed out"`, wantErr: "query timed out"},
		{desc: "plan without Query Data API", status: http.StatusForbidden, wantErr: "Enterprise"},
		{desc: "rate limited", status: http.StatusTooManyRequests, wantErr: "rate limited"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			fake := &fakeHoneycomb{results: []string{tt.result}, status: tt.status}
			s, _ := newHoneycombTestScaler(t, fake, honeycombTestMetadata(map[string]string{"resultField": tt.resultField}))
			metrics, active, err := s.GetMetricsAndActivity(context.Background(), "s0-honeycomb")
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.InDelta(t, tt.wantValue, metrics[0].Value.AsApproximateFloat64(), 0.001)
			assert.Equal(t, tt.wantActive, active)
		})
	}
}

func TestHoneycombPollingStopsAtContextDeadline(t *testing.T) {
	fake := &fakeHoneycomb{}
	s, _ := newHoneycombTestScaler(t, fake, honeycombTestMetadata(nil))
	// Every GET for r1 answers incomplete because getsForResult never reaches 2.
	fake.getsForResult = map[string]int{"r1": -1 << 30}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, _, err := s.GetMetricsAndActivity(ctx, "s0-honeycomb")
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(start), 2*time.Second)
}
