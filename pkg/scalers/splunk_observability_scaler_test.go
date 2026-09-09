package scalers

import (
	"context"
	"runtime"
	"strings"
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
	// Missing 'queryAggregator' field, fail
	{map[string]string{"query": "data('demo.trans.latency').max().publish()", "duration": "10", "targetValue": "200.0", "activationTargetValue": "1.1"}, validSplunkObservabilityAuthParams, true},
	// Missing 'activationTargetValue' field, fail
	{map[string]string{"query": "data('demo.trans.latency').max().publish()", "duration": "10", "targetValue": "200.0", "queryAggregator": "avg"}, validSplunkObservabilityAuthParams, true},
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

func countSplunkO11ySignalflowGoroutines() int {
	buf := make([]byte, 1<<20)
	n := runtime.Stack(buf, true)
	count := 0
	for _, stack := range strings.Split(string(buf[:n]), "\n\n") {
		if strings.Contains(stack, "signalflow-client-go") {
			count++
		}
	}
	return count
}

const splunkO11yFakeProgram = "data('demo.trans.latency').max().publish()"

func newFakeSplunkO11yScalerWithBackend(t *testing.T, duration int) (*splunkObservabilityScaler, *signalflow.FakeBackend, func()) {
	t.Helper()

	fake := signalflow.NewRunningFakeBackend()
	client, err := fake.Client()
	if err != nil {
		fake.Stop()
		t.Fatal("could not create fake backend client:", err)
	}

	tsid := idtool.ID(1)
	fake.AddProgramTSIDs(splunkO11yFakeProgram, []idtool.ID{tsid})
	fake.SetTSIDFloatData(tsid, 42.0)

	scaler := &splunkObservabilityScaler{
		metadata: &splunkObservabilityMetadata{
			Query:           splunkO11yFakeProgram,
			Duration:        duration,
			QueryAggregator: "max",
		},
		apiClient: client,
		logger:    logr.Discard(),
	}

	cleanup := func() {
		_ = scaler.Close(context.Background())
		fake.Stop()
	}
	return scaler, fake, cleanup
}

// newFakeSplunkO11yScaler wires a scaler to a fake backend that streams indefinitely without closing.
func newFakeSplunkO11yScaler(t *testing.T, duration int) (*splunkObservabilityScaler, func()) {
	t.Helper()
	scaler, _, stop := newFakeSplunkO11yScalerWithBackend(t, duration)
	return scaler, stop
}

func waitForSplunkO11yJob(t *testing.T, fake *signalflow.FakeBackend) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for fake.RunningJobsForProgram(splunkO11yFakeProgram) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("SignalFlow fake backend did not start the query")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// Regression guard: a stuck stream must not block getQueryResult past the parent context deadline.
func TestSplunkObservabilityGetQueryResultReturnsOnParentContextCancel(t *testing.T) {
	scaler, fake, stop := newFakeSplunkO11yScalerWithBackend(t, 3600)
	defer stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := scaler.getQueryResult(ctx)
		done <- err
	}()
	waitForSplunkO11yJob(t, fake)

	start := time.Now()
	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected cancelled query to return an error")
		}
		if elapsed := time.Since(start); elapsed >= splunkO11yDrainTimeout/2 {
			t.Fatalf("getQueryResult returned after %v, waited for timeout cleanup", elapsed)
		}
	case <-time.After(splunkO11yDrainTimeout):
		t.Fatal("getQueryResult did not return after parent context was cancelled; it is hanging")
	}
}

func TestSplunkObservabilityCloseCancelsActiveQuery(t *testing.T) {
	scaler, fake, stop := newFakeSplunkO11yScalerWithBackend(t, 3600)
	defer stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	queryDone := make(chan error, 1)
	go func() {
		_, err := scaler.getQueryResult(ctx)
		queryDone <- err
	}()
	waitForSplunkO11yJob(t, fake)

	closeDone := make(chan error, 1)
	go func() {
		closeDone <- scaler.Close(context.Background())
	}()

	select {
	case err := <-queryDone:
		if err == nil {
			t.Fatal("expected Close to cancel the active query")
		}
	case <-time.After(time.Second):
		cancel()
		select {
		case <-queryDone:
		case <-time.After(splunkO11yDrainTimeout):
		}
		t.Fatal("Close did not cancel the active query")
	}

	select {
	case err := <-closeDone:
		if err != nil {
			t.Fatalf("Close: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not wait for query cleanup")
	}
}

func TestSplunkObservabilityCloseReapsClientGoroutines(t *testing.T) {
	before := countSplunkO11ySignalflowGoroutines()

	scaler, stop := newFakeSplunkO11yScaler(t, 1)
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if _, err := scaler.getQueryResult(ctx); err != nil {
		t.Fatalf("getQueryResult: %v", err)
	}
	if during := countSplunkO11ySignalflowGoroutines(); during <= before {
		t.Fatal("expected signalflow goroutines while the client is open")
	}

	if err := scaler.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	var after int
	for {
		after = countSplunkO11ySignalflowGoroutines()
		if after <= before+1 || time.Now().After(deadline) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if after > before+1 {
		t.Fatalf("expected at most 1 leftover signalflow goroutine after Close, got %d (baseline %d)", after-before, before)
	}
}

func TestSplunkObservabilityGetQueryResultAfterClose(t *testing.T) {
	scaler, stop := newFakeSplunkO11yScaler(t, 1)
	defer stop()

	if err := scaler.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := scaler.getQueryResult(ctx)
	if err == nil {
		t.Fatal("expected error from getQueryResult after Close")
	}
	if !strings.Contains(err.Error(), "splunk observability scaler is closed") {
		t.Fatalf("expected closed error, got %v", err)
	}
}

func TestSplunkObservabilityCloseIsIdempotent(t *testing.T) {
	scaler, stop := newFakeSplunkO11yScaler(t, 1)
	defer stop()

	if err := scaler.Close(context.Background()); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := scaler.Close(context.Background()); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestSplunkObservabilityReconnectsAfterWebsocketDrop(t *testing.T) {
	scaler, fake, stop := newFakeSplunkO11yScalerWithBackend(t, 1)
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if _, err := scaler.getQueryResult(ctx); err != nil {
		t.Fatalf("healthy poll: %v", err)
	}

	fake.KillExistingConnections()

	deadline := time.Now().Add(20 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		pctx, pcancel := context.WithTimeout(context.Background(), 20*time.Second)
		_, lastErr = scaler.getQueryResult(pctx)
		pcancel()
		if lastErr == nil {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("library did not recover on the same client within 20s: %v", lastErr)
}
