package internal

import (
	"context"
	"sync"
	"time"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
)

type deadlockDetector struct {
	lock    sync.RWMutex // Applies to all fields below
	tickers map[*deadlockTicker]struct{}
	paused  bool
}

type deadlockTicker struct {
	d *deadlockDetector

	lock                sync.Mutex // Applies to all fields below
	t                   *time.Ticker
	paused              bool
	expectedExpiration  time.Time
	pausedWithRemaining time.Duration
}

// PauseDeadlockDetector pauses the deadlock detector for all coroutines.
func PauseDeadlockDetector(ctx Context) {
	if d := getDeadlockDetector(ctx); d != nil {
		d.pause()
	}
}

// ResumeDeadlockDetector resumes the deadlock detector for all coroutines.
func ResumeDeadlockDetector(ctx Context) {
	if d := getDeadlockDetector(ctx); d != nil {
		d.resume()
	}
}

// DataConverterWithoutDeadlockDetection returns a data converter that disables
// workflow deadlock detection for each call on the data converter. This should
// be used for advanced data converters that may perform remote calls or
// otherwise intentionally execute longer than the default deadlock detection
// timeout.
//
// Exposed as: [go.temporal.io/sdk/workflow.DataConverterWithoutDeadlockDetection]
func DataConverterWithoutDeadlockDetection(c converter.DataConverter) converter.DataConverter {
	return &dataConverterWithoutDeadlock{underlying: c}
}

// getDeadlockDetector returns the deadlock detector if the context represents
// a running workflow or nil if not.
func getDeadlockDetector(ctx Context) *deadlockDetector {
	if s := getStateIfRunning(ctx); s != nil {
		return s.dispatcher.deadlockDetector
	}
	return nil
}

func newDeadlockDetector() *deadlockDetector {
	return &deadlockDetector{tickers: map[*deadlockTicker]struct{}{}}
}

// begin starts a new deadlock detection ticker which may start as paused
// depending on the state of the detector. Callers must call end to clean up the
// ticker.
func (d *deadlockDetector) begin(timeout time.Duration) *deadlockTicker {
	d.lock.Lock()
	defer d.lock.Unlock()
	t := &deadlockTicker{d: d, paused: d.paused}
	// Set different values based on whether paused or not
	if d.paused {
		t.t = time.NewTicker(unlimitedDeadlockDetectionTimeout)
		t.pausedWithRemaining = timeout
	} else {
		t.t = time.NewTicker(timeout)
		t.expectedExpiration = time.Now().Add(timeout)
	}
	d.tickers[t] = struct{}{}
	return t
}

func (d *deadlockDetector) pause() {
	d.lock.RLock()
	defer d.lock.RUnlock()
	for t := range d.tickers {
		t.pause()
	}
	d.paused = true
}

func (d *deadlockDetector) resume() {
	d.lock.RLock()
	defer d.lock.RUnlock()
	for t := range d.tickers {
		t.resume()
	}
	d.paused = false
}

func (d *deadlockTicker) reached() <-chan time.Time {
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.t == nil {
		return nil
	}
	return d.t.C
}

func (d *deadlockTicker) pause() {
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.paused || d.t == nil {
		return
	}
	d.t.Stop()
	d.paused = true
	d.pausedWithRemaining = time.Until(d.expectedExpiration)
	// To prevent later panic, we make this at least 1
	if d.pausedWithRemaining < 1 {
		d.pausedWithRemaining = 1
	}
}

func (d *deadlockTicker) resume() {
	d.lock.Lock()
	defer d.lock.Unlock()
	if !d.paused || d.t == nil {
		return
	}
	d.paused = false
	d.t.Reset(d.pausedWithRemaining)
	// We intentionally put this after reset and accept that this is later than
	// the reset time to be safe
	d.expectedExpiration = time.Now().Add(d.pausedWithRemaining)
}

func (d *deadlockTicker) end() {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.d.lock.Lock()
	delete(d.d.tickers, d)
	d.d.lock.Unlock()
	d.t.Stop()
	d.t = nil
}

type dataConverterWithoutDeadlock struct {
	context    Context
	underlying converter.DataConverter
}

// Exposed as: [go.temporal.io/sdk/workflow.ContextAware]
var _ ContextAware = &dataConverterWithoutDeadlock{}

func (d *dataConverterWithoutDeadlock) ToPayload(value interface{}) (*commonpb.Payload, error) {
	PauseDeadlockDetector(d.context)
	defer ResumeDeadlockDetector(d.context)
	return d.underlying.ToPayload(value)
}

func (d *dataConverterWithoutDeadlock) FromPayload(payload *commonpb.Payload, valuePtr interface{}) error {
	PauseDeadlockDetector(d.context)
	defer ResumeDeadlockDetector(d.context)
	return d.underlying.FromPayload(payload, valuePtr)
}

func (d *dataConverterWithoutDeadlock) ToPayloads(value ...interface{}) (*commonpb.Payloads, error) {
	PauseDeadlockDetector(d.context)
	defer ResumeDeadlockDetector(d.context)
	return d.underlying.ToPayloads(value...)
}

func (d *dataConverterWithoutDeadlock) FromPayloads(payloads *commonpb.Payloads, valuePtrs ...interface{}) error {
	PauseDeadlockDetector(d.context)
	defer ResumeDeadlockDetector(d.context)
	return d.underlying.FromPayloads(payloads, valuePtrs...)
}

func (d *dataConverterWithoutDeadlock) ToString(input *commonpb.Payload) string {
	PauseDeadlockDetector(d.context)
	defer ResumeDeadlockDetector(d.context)
	return d.underlying.ToString(input)
}

func (d *dataConverterWithoutDeadlock) ToStrings(input *commonpb.Payloads) []string {
	PauseDeadlockDetector(d.context)
	defer ResumeDeadlockDetector(d.context)
	return d.underlying.ToStrings(input)
}

func (d *dataConverterWithoutDeadlock) WithWorkflowContext(ctx Context) converter.DataConverter {
	return &dataConverterWithoutDeadlock{context: ctx, underlying: WithWorkflowContext(ctx, d.underlying)}
}

func (d *dataConverterWithoutDeadlock) WithContext(ctx context.Context) converter.DataConverter {
	return &dataConverterWithoutDeadlock{context: d.context, underlying: WithContext(ctx, d.underlying)}
}
