package scalers

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/signalfx/signalflow-client-go/v2/signalflow"
	"github.com/signalfx/signalflow-client-go/v2/signalflow/messages"
	"go.uber.org/atomic"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

// splunkO11yStreamMargin bounds the stream read beyond the configured Duration.
const splunkO11yStreamMargin = 10 * time.Second

// splunkO11yDrainTimeout is a short best-effort budget for draining after Stop.
const splunkO11yDrainTimeout = 2 * time.Second

// splunkO11yStartupTimeout bounds eager persistent-stream startup.
const splunkO11yStartupTimeout = 30 * time.Second

type splunkObservabilityMetadata struct {
	TriggerIndex int

	AccessToken           string  `keda:"name=accessToken,           order=authParams"`
	Realm                 string  `keda:"name=realm,                 order=authParams"`
	Query                 string  `keda:"name=query,                 order=triggerMetadata"`
	Duration              int     `keda:"name=duration,              order=triggerMetadata"`
	TargetValue           float64 `keda:"name=targetValue,   	     order=triggerMetadata"`
	QueryAggregator       string  `keda:"name=queryAggregator,       order=triggerMetadata, enum=min;max;avg;sum;count;latest, default=avg"`
	ActivationTargetValue float64 `keda:"name=activationTargetValue, order=triggerMetadata"`
	PersistentStream      bool    `keda:"name=persistentStream,      order=triggerMetadata, default=false"`
}

type splunkO11ySample struct {
	t time.Time
	v float64
}

type splunkO11yRollupState struct {
	maxValue    float64
	minValue    float64
	valueSum    float64
	valueCount  int
	latestValue float64
}

func (r *splunkO11yRollupState) addMessage(msg *messages.DataMessage, logger logr.Logger) error {
	for i := range msg.Payloads {
		value, err := splunkO11yPayloadValue(&msg.Payloads[i])
		if err != nil {
			return err
		}
		logger.V(1).Info(fmt.Sprintf("Encountering value %.4f\n", value))
		r.maxValue = math.Max(r.maxValue, value)
		r.minValue = math.Min(r.minValue, value)
		r.valueSum += value
		r.valueCount++
		r.latestValue = value
	}
	return nil
}

func splunkO11yPayloadValue(payload *messages.DataPayload) (float64, error) {
	switch value := payload.Value().(type) {
	case float64:
		return value, nil
	case int64:
		return float64(value), nil
	case int32:
		return float64(value), nil
	default:
		return 0, fmt.Errorf("could not convert Splunk Observability metric value to float64")
	}
}

type splunkO11yComputation interface {
	Data() <-chan *messages.DataMessage
	Stop(context.Context) error
}

type splunkO11yClient interface {
	Execute(context.Context, *signalflow.ExecuteRequest) (*signalflow.Computation, error)
	Close()
}

type splunkObservabilityScaler struct {
	metadata      *splunkObservabilityMetadata
	apiClient     splunkO11yClient
	logger        logr.Logger
	mu            sync.Mutex
	closed        bool
	comp          splunkO11yComputation
	samples       []splunkO11ySample
	streamErr     error
	stopOnce      sync.Once
	nextQueryID   uint64
	activeQueries map[uint64]context.CancelFunc
	inFlight      sync.WaitGroup
	executions    sync.WaitGroup
	closeDone     chan struct{}
	unhealthy     atomic.Bool
}

func parseSplunkObservabilityMetadata(config *scalersconfig.ScalerConfig) (*splunkObservabilityMetadata, error) {
	meta := &splunkObservabilityMetadata{}
	meta.TriggerIndex = config.TriggerIndex

	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing splunk observability metadata: %w", err)
	}

	if meta.PersistentStream && meta.Duration <= 0 {
		return nil, fmt.Errorf("duration must be greater than 0 when persistentStream is enabled, got %d", meta.Duration)
	}

	return meta, nil
}

func newSplunkO11yConnection(meta *splunkObservabilityMetadata, logger logr.Logger) (*signalflow.Client, error) {
	apiClient, err := signalflow.NewClient(
		signalflow.StreamURLForRealm(meta.Realm),
		signalflow.AccessToken(meta.AccessToken),
		signalflow.OnError(func(err error) {
			logger.Error(err, "error in SignalFlow client")
		}))
	if err != nil {
		return nil, fmt.Errorf("error creating SignalFlow client: %w", err)
	}

	return apiClient, nil
}

func NewSplunkObservabilityScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	logger := InitializeLogger(config, "splunk_observability_scaler")

	meta, err := parseSplunkObservabilityMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing Splunk metadata: %w", err)
	}

	apiClient, err := newSplunkO11yConnection(meta, logger)
	if err != nil {
		return nil, fmt.Errorf("error establishing Splunk Observability Cloud connection: %w", err)
	}

	scaler := &splunkObservabilityScaler{
		metadata:  meta,
		apiClient: apiClient,
		logger:    logger,
	}
	if meta.PersistentStream {
		if err := scaler.startPersistentStream(); err != nil {
			_ = scaler.Close(context.Background())
			return nil, err
		}
	}
	return scaler, nil
}

// stopAndDrain stops the computation and keeps reading Data() until it closes or a
// short grace period elapses, so the SignalFlow client's goroutines are not left
// blocked on sends to an unconsumed channel. If process is non-nil, drained messages
// are passed to it; a process error ends the drain and is returned.
func (s *splunkObservabilityScaler) stopAndDrain(comp splunkO11yComputation, process func(*messages.DataMessage) error) error {
	stopCtx, cancel := context.WithTimeout(context.Background(), splunkO11yDrainTimeout)
	defer cancel()

	if err := comp.Stop(stopCtx); err != nil {
		s.logger.V(1).Info("Failed to stop SignalFlow computation", "error", err)
	}

	dataCh := comp.Data()
	for {
		// Give the deadline priority so a backend that keeps sending cannot extend the grace budget.
		select {
		case <-stopCtx.Done():
			s.logger.V(1).Info("Gave up draining SignalFlow data channel after stop")
			return nil
		default:
		}

		select {
		case msg, ok := <-dataCh:
			if !ok {
				return nil
			}
			if process == nil {
				continue
			}
			if err := process(msg); err != nil {
				return err
			}
		case <-stopCtx.Done():
			s.logger.V(1).Info("Gave up draining SignalFlow data channel after stop")
			return nil
		}
	}
}

func (s *splunkObservabilityScaler) startPersistentStream() error {
	startupCtx, cancel := context.WithTimeout(context.Background(), splunkO11yStartupTimeout)
	defer cancel()
	return s.startPersistentStreamWithContext(startupCtx)
}

// executeGuarded bounds the caller's wait even when the SDK ignores ctx.
// An unbuffered handoff gives either the caller or timeout cleanup ownership
// of the returned computation, including when Execute returns after cancellation.
func (s *splunkObservabilityScaler) executeGuarded(ctx context.Context, apiClient splunkO11yClient, req *signalflow.ExecuteRequest) (*signalflow.Computation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	type executeResult struct {
		comp *signalflow.Computation
		err  error
	}
	resCh := make(chan executeResult)
	s.executions.Add(1)
	go func() {
		defer s.executions.Done()
		comp, err := apiClient.Execute(ctx, req)
		select {
		case resCh <- executeResult{comp: comp, err: err}:
		case <-ctx.Done():
			if comp != nil {
				_ = s.stopAndDrain(comp, nil)
			}
		}
	}()
	select {
	case res := <-resCh:
		return res.comp, res.err
	case <-ctx.Done():
		s.unhealthy.Store(true)
		return nil, ctx.Err()
	}
}

func (s *splunkObservabilityScaler) startPersistentStreamWithContext(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.apiClient == nil {
		return fmt.Errorf("splunk observability scaler is closed")
	}
	comp, err := s.executeGuarded(ctx, s.apiClient, &signalflow.ExecuteRequest{
		Program: s.metadata.Query,
	})
	if err != nil {
		return fmt.Errorf("could not execute signalflow query: %w", err)
	}
	s.comp = comp
	go s.readPersistentStream(comp)
	return nil
}

func (s *splunkObservabilityScaler) stopPersistentStream(comp splunkO11yComputation) {
	s.stopOnce.Do(func() {
		_ = s.stopAndDrain(comp, nil)
	})
}

func (s *splunkObservabilityScaler) readPersistentStream(comp splunkO11yComputation) {
	for msg := range comp.Data() {
		if err := s.ingestPersistentMessage(msg); err != nil {
			s.mu.Lock()
			s.streamErr = err
			s.mu.Unlock()
			s.stopPersistentStream(comp)
			return
		}
	}
	s.mu.Lock()
	if s.streamErr == nil {
		s.streamErr = fmt.Errorf("splunk observability persistent stream ended")
	}
	s.mu.Unlock()
}

func (s *splunkObservabilityScaler) startQuery(ctx context.Context) (splunkO11yClient, context.Context, func(), error) {
	queryCtx, cancel := context.WithCancel(ctx)

	s.mu.Lock()
	if s.closed || s.apiClient == nil {
		s.mu.Unlock()
		cancel()
		return nil, nil, nil, fmt.Errorf("splunk observability scaler is closed")
	}
	if s.unhealthy.Load() {
		s.mu.Unlock()
		cancel()
		return nil, nil, nil, fmt.Errorf("splunk observability signalflow client is unhealthy")
	}

	queryID := s.nextQueryID
	s.nextQueryID++
	if s.activeQueries == nil {
		s.activeQueries = make(map[uint64]context.CancelFunc)
	}
	s.activeQueries[queryID] = cancel
	s.inFlight.Add(1)
	apiClient := s.apiClient
	s.mu.Unlock()

	finishQuery := func() {
		s.mu.Lock()
		delete(s.activeQueries, queryID)
		s.mu.Unlock()
		cancel()
		s.inFlight.Done()
	}

	return apiClient, queryCtx, finishQuery, nil
}

func (s *splunkObservabilityScaler) ingestPersistentMessage(msg *messages.DataMessage) error {
	if len(msg.Payloads) == 0 {
		return nil
	}
	// Freshness is measured against arrival time: SignalFlow logical timestamps
	// lag wall-clock by resolution + maxDelay, so comparing them against
	// time.Now() would mark fresh data stale.
	ts := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := ts.Add(-time.Duration(s.metadata.Duration) * time.Second)
	kept := s.samples[:0]
	for _, sample := range s.samples {
		if sample.t.After(cutoff) {
			kept = append(kept, sample)
		}
	}
	s.samples = kept
	for i := range msg.Payloads {
		value, err := splunkO11yPayloadValue(&msg.Payloads[i])
		if err != nil {
			return err
		}
		s.samples = append(s.samples, splunkO11ySample{t: ts, v: value})
	}
	return nil
}

func (s *splunkObservabilityScaler) persistentQueryResult() (float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.apiClient == nil {
		return -1, fmt.Errorf("splunk observability scaler is closed")
	}
	if s.unhealthy.Load() {
		return -1, fmt.Errorf("splunk observability signalflow client is unhealthy")
	}
	cutoff := time.Now().Add(-time.Duration(s.metadata.Duration) * time.Second)
	var window []splunkO11ySample
	for _, sample := range s.samples {
		if sample.t.After(cutoff) {
			window = append(window, sample)
		}
	}
	if s.streamErr != nil {
		return -1, fmt.Errorf("splunk observability persistent stream ended: %w", s.streamErr)
	}
	if len(window) == 0 {
		if len(s.samples) > 0 {
			return -1, fmt.Errorf("splunk observability persistent stream is stale")
		}
		return 0, fmt.Errorf("query returned no data points")
	}
	return aggregateSplunkO11ySamples(s.metadata.QueryAggregator, window)
}

func aggregateSplunkO11ySamples(aggregator string, samples []splunkO11ySample) (float64, error) {
	maxValue := math.Inf(-1)
	minValue := math.Inf(1)
	valueSum := 0.0
	latestValue := samples[len(samples)-1].v
	for _, sample := range samples {
		maxValue = math.Max(maxValue, sample.v)
		minValue = math.Min(minValue, sample.v)
		valueSum += sample.v
	}
	return splunkObservabilityRollup(aggregator, maxValue, minValue, valueSum, len(samples), latestValue)
}

func (s *splunkObservabilityScaler) getQueryResult(ctx context.Context) (float64, error) {
	if s.metadata.PersistentStream {
		return s.persistentQueryResult()
	}
	return s.getPollingQueryResult(ctx)
}

func (s *splunkObservabilityScaler) getPollingQueryResult(ctx context.Context) (float64, error) {
	apiClient, queryCtx, finishQuery, err := s.startQuery(ctx)
	if err != nil {
		return -1, err
	}
	cleanupStarted := false
	defer func() {
		if !cleanupStarted {
			finishQuery()
		}
	}()

	comp, err := s.executeGuarded(queryCtx, apiClient, &signalflow.ExecuteRequest{
		Program: s.metadata.Query,
	})
	if err != nil {
		return -1, fmt.Errorf("could not execute signalflow query: %w", err)
	}

	s.logger.V(1).Info("Started MTS stream.")

	streamDuration := time.Duration(s.metadata.Duration) * time.Second
	// Hard deadline beyond the Duration window so a non-responsive backend cannot block forever.
	streamCtx, cancel := context.WithTimeout(queryCtx, streamDuration+splunkO11yStreamMargin)
	defer cancel()

	stopTimer := time.NewTimer(streamDuration)
	defer stopTimer.Stop()

	rollup := splunkO11yRollupState{
		maxValue: math.Inf(-1),
		minValue: math.Inf(1),
	}

	process := func(msg *messages.DataMessage) error {
		if len(msg.Payloads) == 0 {
			s.logger.V(1).Info("No data retrieved.")
			return nil
		}
		if err := rollup.addMessage(msg, s.logger); err != nil {
			return err
		}
		return nil
	}

	s.logger.V(1).Info("Now iterating over results.")

	// timedOut handles the hard-deadline path: stop, drain, and return the timeout error.
	timedOut := func() error {
		s.logger.V(1).Info("Context done before stream completed; stopping computation.")
		cleanupStarted = true
		go func() {
			_ = s.stopAndDrain(comp, nil)
			finishQuery()
		}()
		return fmt.Errorf("splunk observability query ended before stream completed: %w", streamCtx.Err())
	}

	if err := s.readQuery(streamCtx, comp, process, stopTimer.C, timedOut); err != nil {
		return -1, err
	}

	result, err := splunkObservabilityRollup(s.metadata.QueryAggregator, rollup.maxValue, rollup.minValue, rollup.valueSum, rollup.valueCount, rollup.latestValue)
	if err != nil {
		return result, err
	}
	s.logger.V(1).Info(fmt.Sprintf("Returning %s value: %.4f\n", s.metadata.QueryAggregator, result))
	return result, nil
}

func (s *splunkObservabilityScaler) readQuery(ctx context.Context, comp splunkO11yComputation, process func(*messages.DataMessage) error, stop <-chan time.Time, timedOut func() error) error {
	dataCh := comp.Data()
	for {
		// Give the hard deadline priority: select has no fairness, so a continuously
		// ready dataCh could otherwise starve the ctx.Done() case.
		select {
		case <-ctx.Done():
			return timedOut()
		default:
		}

		select {
		case <-ctx.Done():
			return timedOut()
		case <-stop:
			s.logger.V(1).Info("Stopping MTS stream after duration.")
			// Stop, but keep processing any remaining buffered messages for this window.
			return s.stopAndDrain(comp, process)
		case msg, ok := <-dataCh:
			if !ok {
				return nil
			}
			if err := process(msg); err != nil {
				_ = s.stopAndDrain(comp, nil)
				return err
			}
		}
	}
}

func splunkObservabilityRollup(aggregator string, maxValue, minValue, valueSum float64, valueCount int, latestValue float64) (float64, error) {
	if valueCount == 0 {
		return 0, fmt.Errorf("query returned no data points")
	}
	if valueCount > 1 && aggregator == "" {
		return 0, fmt.Errorf("query returned more than 1 series; modify the query to return only 1 series or add a queryAggregator")
	}
	switch aggregator {
	case "max":
		return maxValue, nil
	case "min":
		return minValue, nil
	case "avg":
		return valueSum / float64(valueCount), nil
	case "sum":
		return valueSum, nil
	case "count":
		return float64(valueCount), nil
	case "latest":
		return latestValue, nil
	default:
		return 0, fmt.Errorf("invalid queryAggregator: %q", aggregator)
	}
}

func (s *splunkObservabilityScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	num, err := s.getQueryResult(ctx)

	if err != nil {
		s.logger.Error(err, "error getting metrics from Splunk Observability Cloud.")
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error getting metrics from Splunk Observability Cloud: %w", err)
	}
	metric := GenerateMetricInMili(metricName, num)

	return []external_metrics.ExternalMetricValue{metric}, num > s.metadata.ActivationTargetValue, nil
}

func (s *splunkObservabilityScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	metricName := kedautil.NormalizeString("signalfx")
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.TriggerIndex, metricName),
		},
		Target: GetMetricTargetMili(v2.ValueMetricType, s.metadata.TargetValue),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

func (s *splunkObservabilityScaler) closeClient(apiClient splunkO11yClient) {
	if apiClient == nil {
		return
	}
	closeClient := func() {
		s.executions.Wait()
		apiClient.Close()
	}
	if s.unhealthy.Load() {
		// Execute may still hold an SDK lock after its caller times out. Defer
		// closure until it returns and any late computation has been drained.
		go closeClient()
		return
	}
	closeClient()
}

func (s *splunkObservabilityScaler) Close(context.Context) error {
	s.mu.Lock()
	if s.closed {
		done := s.closeDone
		s.mu.Unlock()
		if done != nil {
			<-done
		}
		return nil
	}
	s.closed = true
	s.closeDone = make(chan struct{})
	done := s.closeDone
	apiClient := s.apiClient
	s.apiClient = nil
	comp := s.comp
	s.comp = nil
	cancels := make([]context.CancelFunc, 0, len(s.activeQueries))
	for _, cancel := range s.activeQueries {
		cancels = append(cancels, cancel)
	}
	s.mu.Unlock()

	for _, cancel := range cancels {
		cancel()
	}
	if comp != nil {
		s.stopPersistentStream(comp)
	}
	s.inFlight.Wait()
	s.closeClient(apiClient)
	close(done)
	return nil
}
