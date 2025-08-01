package metrics

import "time"

// Handler is a handler for metrics emitted by the SDK. This interface is
// intentionally limited to only what the SDK needs to emit metrics and is not
// built to be a general purpose metrics abstraction for all uses.
//
// A common implementation is at
// go.temporal.io/sdk/contrib/tally.NewMetricsHandler. The NopHandler is a noop
// handler. A handler may implement "Unwrap() Handler" if it wraps a handler.
type Handler interface {
	// WithTags returns a new handler with the given tags set for each metric
	// created from it. Old tags from the previous handler are either preserved
	// or overwritten, if an existing key is also present in the new tag set.
	WithTags(map[string]string) Handler

	// Counter obtains a counter for the given name.
	Counter(name string) Counter

	// Gauge obtains a gauge for the given name.
	Gauge(name string) Gauge

	// Timer obtains a timer for the given name.
	Timer(name string) Timer
}

// Counter is an ever-increasing counter.
type Counter interface {
	// Inc increments the counter value.
	Inc(int64)
}

// CounterFunc implements Counter with a single function.
type CounterFunc func(int64)

// Inc implements Counter.Inc.
func (c CounterFunc) Inc(d int64) { c(d) }

// Gauge can be set to any float.
type Gauge interface {
	// Update updates the gauge value.
	Update(float64)
}

// GaugeFunc implements Gauge with a single function.
type GaugeFunc func(float64)

// Update implements Gauge.Update.
func (g GaugeFunc) Update(d float64) { g(d) }

// Timer records time durations.
type Timer interface {
	// Record sets the timer value.
	Record(time.Duration)
}

// TimerFunc implements Timer with a single function.
type TimerFunc func(time.Duration)

// Record implements Timer.Record.
func (t TimerFunc) Record(d time.Duration) { t(d) }

// NopHandler is a noop handler that does nothing with the metrics.
var NopHandler Handler = nopHandler{}

type nopHandler struct{}

func (nopHandler) WithTags(map[string]string) Handler { return nopHandler{} }
func (nopHandler) Counter(string) Counter             { return nopHandler{} }
func (nopHandler) Gauge(string) Gauge                 { return nopHandler{} }
func (nopHandler) Timer(string) Timer                 { return nopHandler{} }
func (nopHandler) Inc(int64)                          {}
func (nopHandler) Update(float64)                     {}
func (nopHandler) Record(time.Duration)               {}

type replayAwareHandler struct {
	replay     *bool
	underlying Handler
}

// NewReplayAwareHandler is a handler that will not record any metrics if the
// boolean pointed to by "replay" is true.
func NewReplayAwareHandler(replay *bool, underlying Handler) Handler {
	return &replayAwareHandler{replay, underlying}
}

func (r *replayAwareHandler) WithTags(tags map[string]string) Handler {
	return NewReplayAwareHandler(r.replay, r.underlying.WithTags(tags))
}

func (r *replayAwareHandler) Counter(name string) Counter {
	underlying := r.underlying.Counter(name)
	return CounterFunc(func(d int64) {
		if !*r.replay {
			underlying.Inc(d)
		}
	})
}

func (r *replayAwareHandler) Gauge(name string) Gauge {
	underlying := r.underlying.Gauge(name)
	return GaugeFunc(func(d float64) {
		if !*r.replay {
			underlying.Update(d)
		}
	})
}

func (r *replayAwareHandler) Timer(name string) Timer {
	underlying := r.underlying.Timer(name)
	return TimerFunc(func(d time.Duration) {
		if !*r.replay {
			underlying.Record(d)
		}
	})
}

func (r *replayAwareHandler) Unwrap() Handler {
	return r.underlying
}
