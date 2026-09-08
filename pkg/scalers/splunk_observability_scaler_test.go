package scalers

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/signalfx/signalflow-client-go/v2/signalflow"
	"github.com/signalfx/signalfx-go/idtool"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type parseSplunkObservabilityMetadataTestData struct {
	metadata   map[string]string
	authParams map[string]string
	isError    bool
}

type SplunkObservabilityMetricIdentifier struct {
	metadataTestData *parseSplunkObservabilityMetadataTestData
	triggerIndex     int
	metricName       string
}

var validSplunkObservabilityAuthParams = map[string]string{
	"accessToken": "my-super-secret-access-token",
	"realm":       "my-realm",
}

var invalidSplunkObservabilityAuthParams = map[string]string{
	"accessToken": "",
	"realm":       "my-realm",
}

var validSplunkObservabilityMetadata = map[string]string{
	"query":                 "data('demo.trans.latency').max().publish()",
	"duration":              "10",
	"targetValue":           "200.0",
	"queryAggregator":       "avg",
	"activationTargetValue": "1.1",
}

var testSplunkObservabilityMetadata = []parseSplunkObservabilityMetadataTestData{
	// Valid metadata and valid auth params, pass.
	{validSplunkObservabilityMetadata, validSplunkObservabilityAuthParams, false},
	// no params at all, fail
	{map[string]string{}, map[string]string{}, true},
	// No metadada but valid auth, fail.
	{map[string]string{}, validSplunkObservabilityAuthParams, true},
	// Valid metadada but no auth params, fail.
	{validSplunkObservabilityMetadata, map[string]string{}, true},
	// Missing 'query' field, fail
	{map[string]string{"duration": "10", "targetValue": "200.0", "queryAggregator": "avg", "activationTargetValue": "1.1"}, validSplunkObservabilityAuthParams, true},
	// Missing 'duration' field, fail
	{map[string]string{"query": "data('demo.trans.latency').max().publish()", "targetValue": "200.0", "queryAggregator": "avg", "activationTargetValue": "1.1"}, validSplunkObservabilityAuthParams, true},
	// Missing 'targetValue' field, fail
	{map[string]string{"query": "data('demo.trans.latency').max().publish()", "duration": "10", "queryAggregator": "avg", "activationTargetValue": "1.1"}, validSplunkObservabilityAuthParams, true},
	// Missing 'queryAggregator' field defaults to avg, pass
	{map[string]string{"query": "data('demo.trans.latency').max().publish()", "duration": "10", "targetValue": "200.0", "activationTargetValue": "1.1"}, validSplunkObservabilityAuthParams, false},
	// Missing 'activationTargetValue' field, fail
	{map[string]string{"query": "data('demo.trans.latency').max().publish()", "duration": "10", "targetValue": "200.0", "queryAggregator": "avg"}, validSplunkObservabilityAuthParams, true},
	// Unsupported 'queryAggregator' value, fail
	{map[string]string{"query": "data('demo.trans.latency').max().publish()", "duration": "10", "targetValue": "200.0", "queryAggregator": "median", "activationTargetValue": "1.1"}, validSplunkObservabilityAuthParams, true},
	// Empty 'accessToken' field
	{map[string]string{"query": "data('demo.trans.latency').max().publish()", "duration": "10", "targetValue": "200.0", "queryAggregator": "avg"}, invalidSplunkObservabilityAuthParams, true},
}

var SplunkObservabilityMetricIdentifiers = []SplunkObservabilityMetricIdentifier{
	{&testSplunkObservabilityMetadata[0], 0, "s0-signalfx"},
	{&testSplunkObservabilityMetadata[0], 1, "s1-signalfx"},
}

func TestSplunkObservabilityParseMetadata(t *testing.T) {
	for _, testData := range testSplunkObservabilityMetadata {
		_, err := parseSplunkObservabilityMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		} else if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

func TestSplunkObservabilityQueryAggregatorDefault(t *testing.T) {
	meta, err := parseSplunkObservabilityMetadata(&scalersconfig.ScalerConfig{
		TriggerMetadata: map[string]string{
			"query":                 "data('demo.trans.latency').max().publish()",
			"duration":              "10",
			"targetValue":           "200.0",
			"activationTargetValue": "1.1",
		},
		AuthParams: validSplunkObservabilityAuthParams,
	})
	if err != nil {
		t.Fatal("expected omitted queryAggregator to parse:", err)
	}
	if meta.QueryAggregator != "avg" {
		t.Errorf("expected default queryAggregator %q, got %q", "avg", meta.QueryAggregator)
	}
}

func TestSplunkObservabilityGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range SplunkObservabilityMetricIdentifiers {
		ctx := context.Background()
		meta, err := parseSplunkObservabilityMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, AuthParams: validSplunkObservabilityAuthParams, TriggerIndex: testData.triggerIndex})
		if err != nil {
			t.Fatal("Could not parse Splunk Observability metadata:", err)
		}
		mockSplunkObservabilityScaler := splunkObservabilityScaler{
			metadata: meta,
		}

		metricSpec := mockSplunkObservabilityScaler.GetMetricSpecForScaling(ctx)
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.metricName {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

// newFakeSplunkO11yScaler wires a scaler to a fake backend that streams indefinitely without closing.
func newFakeSplunkO11yScaler(t *testing.T, program string, duration int) (*splunkObservabilityScaler, func()) {
	t.Helper()

	fake := signalflow.NewRunningFakeBackend()
	client, err := fake.Client()
	if err != nil {
		fake.Stop()
		t.Fatal("could not create fake backend client:", err)
	}

	tsid := idtool.ID(1)
	fake.AddProgramTSIDs(program, []idtool.ID{tsid})
	fake.SetTSIDFloatData(tsid, 42.0)

	scaler := &splunkObservabilityScaler{
		metadata: &splunkObservabilityMetadata{
			Query:           program,
			Duration:        duration,
			QueryAggregator: "max",
		},
		apiClient: client,
		logger:    logr.Discard(),
	}

	return scaler, fake.Stop
}

// Regression guard: a stuck stream must not block getQueryResult past the parent context deadline.
func TestSplunkObservabilityGetQueryResultReturnsOnParentContextCancel(t *testing.T) {
	const program = "data('demo.trans.latency').max().publish()"
	// Large duration so the stopTimer never fires; the parent deadline must bound the call.
	scaler, stop := newFakeSplunkO11yScaler(t, program, 3600)
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	start := time.Now()
	go func() {
		defer close(done)
		_, _ = scaler.getQueryResult(ctx)
	}()

	select {
	case <-done:
		if elapsed := time.Since(start); elapsed > 5*time.Second {
			t.Fatalf("getQueryResult returned after %v, far longer than the context deadline", elapsed)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("getQueryResult did not return after parent context was cancelled; it is hanging")
	}
}

func TestSplunkObservabilityRollup(t *testing.T) {
	tests := []struct {
		name       string
		aggregator string
		maxValue   float64
		minValue   float64
		valueSum   float64
		valueCount int
		latest     float64
		want       float64
		wantErr    bool
	}{
		{name: "max", aggregator: "max", maxValue: 30, minValue: 10, valueSum: 60, valueCount: 3, latest: 30, want: 30},
		{name: "min", aggregator: "min", maxValue: 30, minValue: 10, valueSum: 60, valueCount: 3, latest: 30, want: 10},
		{name: "avg", aggregator: "avg", maxValue: 30, minValue: 10, valueSum: 60, valueCount: 3, latest: 30, want: 20},
		{name: "sum", aggregator: "sum", maxValue: 30, minValue: 10, valueSum: 60, valueCount: 3, latest: 30, want: 60},
		{name: "count", aggregator: "count", maxValue: 30, minValue: 10, valueSum: 60, valueCount: 3, latest: 30, want: 3},
		{name: "latest", aggregator: "latest", maxValue: 30, minValue: 10, valueSum: 60, valueCount: 3, latest: 25, want: 25},
		{name: "single series empty aggregator", aggregator: "", maxValue: 42, minValue: 42, valueSum: 42, valueCount: 1, latest: 42, wantErr: true},
		{name: "multi series empty aggregator", aggregator: "", maxValue: 30, minValue: 10, valueSum: 60, valueCount: 3, latest: 30, wantErr: true},
		{name: "invalid", aggregator: "median", maxValue: 30, minValue: 10, valueSum: 60, valueCount: 3, latest: 30, wantErr: true},
		{name: "no data", aggregator: "max", valueCount: 0, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := splunkObservabilityRollup(tt.aggregator, tt.maxValue, tt.minValue, tt.valueSum, tt.valueCount, tt.latest)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %v want %v", got, tt.want)
			}
		})
	}
}
