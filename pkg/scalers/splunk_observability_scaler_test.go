package scalers

import (
	"context"
	"encoding/binary"
	"log"
	"math"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/signalfx/signalflow-client-go/v2/signalflow"
	"github.com/signalfx/signalflow-client-go/v2/signalflow/messages"
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

type splunkO11yExecutionObserver struct {
	started chan struct{}
	once    sync.Once
}

func (o *splunkO11yExecutionObserver) Write(p []byte) (int, error) {
	if strings.Contains(string(p), "Executing SignalFlow program "+splunkO11yFakeProgram) {
		o.once.Do(func() { close(o.started) })
	}
	return len(p), nil
}

func newFakeSplunkO11yScalerWithBackend(t *testing.T, duration int) (*splunkObservabilityScaler, *signalflow.FakeBackend, <-chan struct{}, func()) {
	t.Helper()

	fake := signalflow.NewRunningFakeBackend()
	started := &splunkO11yExecutionObserver{started: make(chan struct{})}
	fake.SetLogger(log.New(started, "", 0))
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
	return scaler, fake, started.started, cleanup
}

// newFakeSplunkO11yScaler wires a scaler to a fake backend that streams indefinitely without closing.
func newFakeSplunkO11yScaler(t *testing.T, duration int) (*splunkObservabilityScaler, func()) {
	t.Helper()
	scaler, _, _, stop := newFakeSplunkO11yScalerWithBackend(t, duration)
	return scaler, stop
}

func waitForSplunkO11yJob(t *testing.T, started <-chan struct{}) {
	t.Helper()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("SignalFlow fake backend did not start the query")
	}
}

// Regression guard: a stuck stream must not block getQueryResult past the parent context deadline.
func TestSplunkObservabilityGetQueryResultReturnsOnParentContextCancel(t *testing.T) {
	// Large duration so the stopTimer never fires; the parent context must bound the call.
	scaler, _, started, stop := newFakeSplunkO11yScalerWithBackend(t, 3600)
	defer stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := scaler.getQueryResult(ctx)
		done <- err
	}()
	waitForSplunkO11yJob(t, started)

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
	case <-time.After(3 * splunkO11yDrainTimeout):
		t.Fatal("getQueryResult did not return after parent context was cancelled; it is hanging")
	}
}

func TestSplunkObservabilityCloseCancelsActiveQuery(t *testing.T) {
	scaler, _, started, stop := newFakeSplunkO11yScalerWithBackend(t, 3600)
	defer stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	queryDone := make(chan error, 1)
	go func() {
		_, err := scaler.getQueryResult(ctx)
		queryDone <- err
	}()
	waitForSplunkO11yJob(t, started)

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
	scaler, fake, _, stop := newFakeSplunkO11yScalerWithBackend(t, 1)
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

func newFakePersistentSplunkO11yScaler(t *testing.T, duration int) (*splunkObservabilityScaler, *signalflow.FakeBackend, func()) {
	t.Helper()
	scaler, fake, _, stop := newFakeSplunkO11yScalerWithBackend(t, duration)
	scaler.metadata.PersistentStream = true
	if err := scaler.startPersistentStream(); err != nil {
		stop()
		t.Fatal(err)
	}
	return scaler, fake, stop
}

func waitForSplunkO11yValue(t *testing.T, scaler *splunkObservabilityScaler) float64 {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		v, err := scaler.getQueryResult(context.Background())
		if err == nil {
			return v
		}
		lastErr = err
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for persistent stream data: %v", lastErr)
	return 0
}

func TestSplunkObservabilityDefaultPathStopsJob(t *testing.T) {
	scaler, fake, _, stop := newFakeSplunkO11yScalerWithBackend(t, 2)
	defer stop()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if _, err := scaler.getQueryResult(ctx); err != nil {
		t.Fatalf("poll 1: %v", err)
	}
	if jobs := fake.RunningJobsForProgram(splunkO11yFakeProgram); jobs != 0 {
		t.Fatalf("default path left %d running jobs after poll", jobs)
	}
	if _, err := scaler.getQueryResult(ctx); err != nil {
		t.Fatalf("poll 2: %v", err)
	}
	if jobs := fake.RunningJobsForProgram(splunkO11yFakeProgram); jobs != 0 {
		t.Fatalf("default path left %d running jobs after second poll", jobs)
	}
}

func TestSplunkObservabilityPersistentStreamOneExecute(t *testing.T) {
	scaler, fake, stop := newFakePersistentSplunkO11yScaler(t, 10)
	defer stop()
	if waitForSplunkO11yValue(t, scaler) != 42 {
		t.Fatalf("unexpected first poll value")
	}
	if jobs := fake.RunningJobsForProgram(splunkO11yFakeProgram); jobs != 1 {
		t.Fatalf("expected 1 running job after start, got %d", jobs)
	}
	if waitForSplunkO11yValue(t, scaler) != 42 {
		t.Fatalf("unexpected second poll value")
	}
	if jobs := fake.RunningJobsForProgram(splunkO11yFakeProgram); jobs != 1 {
		t.Fatalf("expected 1 running job after two polls, got %d", jobs)
	}
}

func TestSplunkObservabilityPersistentStreamEmpty(t *testing.T) {
	scaler, fake, _, stop := newFakeSplunkO11yScalerWithBackend(t, 10)
	defer stop()
	fake.RemoveTSIDData(idtool.ID(1))
	scaler.metadata.PersistentStream = true
	scaler.metadata.QueryAggregator = "max"
	if err := scaler.startPersistentStream(); err != nil {
		t.Fatal(err)
	}
	_, err := scaler.getQueryResult(context.Background())
	if err == nil || !strings.Contains(err.Error(), "query returned no data points") {
		t.Fatalf("expected empty-window error, got %v", err)
	}
}

func TestSplunkObservabilityPersistentStreamStale(t *testing.T) {
	scaler, fake, stop := newFakePersistentSplunkO11yScaler(t, 3)
	defer stop()
	waitForSplunkO11yValue(t, scaler)
	fake.RemoveTSIDData(idtool.ID(1))
	deadline := time.Now().Add(5 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		_, lastErr = scaler.getQueryResult(context.Background())
		if lastErr != nil && strings.Contains(lastErr.Error(), "persistent stream is stale") {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("expected stale error, got %v", lastErr)
}

func TestSplunkObservabilityPersistentStreamUsesMessageTimestamp(t *testing.T) {
	scaler, _, _, stop := newFakeSplunkO11yScalerWithBackend(t, 1)
	defer stop()
	scaler.metadata.PersistentStream = true

	var value [8]byte
	binary.BigEndian.PutUint64(value[:], math.Float64bits(42))
	message := &messages.DataMessage{
		TimestampedMessage: messages.TimestampedMessage{
			TimestampMillis: uint64(time.Now().Add(-2 * time.Second).UnixMilli()),
		},
		Payloads: []messages.DataPayload{{
			Type: messages.ValTypeDouble,
			TSID: idtool.ID(1),
			Val:  value,
		}},
	}
	if err := scaler.ingestPersistentMessage(message); err != nil {
		t.Fatal(err)
	}

	_, err := scaler.getQueryResult(context.Background())
	if err == nil || !strings.Contains(err.Error(), "persistent stream is stale") {
		t.Fatalf("expected stale error for an old message, got %v", err)
	}
}

type splunkO11yTestComputation struct {
	data     <-chan *messages.DataMessage
	stopped  chan struct{}
	stopOnce sync.Once
}

func (c *splunkO11yTestComputation) Data() <-chan *messages.DataMessage {
	return c.data
}

func (c *splunkO11yTestComputation) Stop(context.Context) error {
	c.stopOnce.Do(func() { close(c.stopped) })
	return nil
}

func TestSplunkObservabilityPersistentStreamIngestErrorStopsComputation(t *testing.T) {
	data := make(chan *messages.DataMessage, 1)
	data <- &messages.DataMessage{Payloads: []messages.DataPayload{{Type: messages.ValType(99)}}}
	close(data)
	computation := &splunkO11yTestComputation{data: data, stopped: make(chan struct{})}
	scaler := &splunkObservabilityScaler{
		metadata: &splunkObservabilityMetadata{Duration: 1},
		logger:   logr.Discard(),
	}

	scaler.readPersistentStream(computation)
	select {
	case <-computation.stopped:
	case <-time.After(time.Second):
		t.Fatal("persistent stream ingest error did not stop the computation")
	}
}

func TestSplunkObservabilityPersistentStreamStartupHonorsContext(t *testing.T) {
	fake := signalflow.NewRunningFakeBackend()
	client, err := fake.Client()
	if err != nil {
		fake.Stop()
		t.Fatal("could not create fake backend client:", err)
	}
	fake.Stop()

	scaler := &splunkObservabilityScaler{
		metadata:  &splunkObservabilityMetadata{Query: splunkO11yFakeProgram},
		apiClient: client,
		logger:    logr.Discard(),
	}
	defer scaler.Close(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- scaler.startPersistentStreamWithContext(ctx)
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected persistent stream startup to fail when its context expires")
		}
	case <-time.After(time.Second):
		t.Fatal("persistent stream startup ignored its context")
	}
}

func TestSplunkObservabilityPersistentStreamEndsWithoutRestart(t *testing.T) {
	scaler, fake, stop := newFakePersistentSplunkO11yScaler(t, 10)
	defer stop()
	waitForSplunkO11yValue(t, scaler)
	fake.KillExistingConnections()
	if jobs := fake.RunningJobsForProgram(splunkO11yFakeProgram); jobs != 0 {
		time.Sleep(200 * time.Millisecond)
		if jobs = fake.RunningJobsForProgram(splunkO11yFakeProgram); jobs != 0 {
			t.Fatalf("expected 0 running jobs after kill, got %d", jobs)
		}
	}
	deadline := time.Now().Add(5 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		_, lastErr = scaler.getQueryResult(context.Background())
		if lastErr != nil {
			if jobs := fake.RunningJobsForProgram(splunkO11yFakeProgram); jobs != 0 {
				t.Fatalf("killed stream restarted a job, running=%d err=%v", jobs, lastErr)
			}
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("expected stream-ended error without restart, got %v", lastErr)
}

func TestSplunkObservabilityPersistentStreamCloseStopsJob(t *testing.T) {
	scaler, fake, stop := newFakePersistentSplunkO11yScaler(t, 10)
	defer stop()
	waitForSplunkO11yValue(t, scaler)
	if err := scaler.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := scaler.Close(context.Background()); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if fake.RunningJobsForProgram(splunkO11yFakeProgram) == 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if jobs := fake.RunningJobsForProgram(splunkO11yFakeProgram); jobs != 0 {
		t.Fatalf("expected 0 running jobs after Close, got %d", jobs)
	}
	_, err := scaler.getQueryResult(context.Background())
	if err == nil || !strings.Contains(err.Error(), "splunk observability scaler is closed") {
		t.Fatalf("expected closed error, got %v", err)
	}
}

func TestSplunkObservabilityPersistentStreamMax(t *testing.T) {
	scaler, fake, _, stop := newFakeSplunkO11yScalerWithBackend(t, 10)
	defer stop()
	fake.SetTSIDFloatData(idtool.ID(1), 10)
	fake.SetTSIDFloatData(idtool.ID(2), 30)
	fake.AddProgramTSIDs(splunkO11yFakeProgram, []idtool.ID{idtool.ID(1), idtool.ID(2)})
	scaler.metadata.PersistentStream = true
	scaler.metadata.QueryAggregator = "max"
	if err := scaler.startPersistentStream(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		v, err := scaler.getQueryResult(context.Background())
		if err == nil && v == 30 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("expected max 30 from persistent window")
}
