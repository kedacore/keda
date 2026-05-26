package internal

import (
	"sync"
	"sync/atomic"
	"time"

	workerpb "go.temporal.io/api/worker/v1"
	"google.golang.org/protobuf/types/known/timestamppb"

	"go.temporal.io/sdk/internal/common/metrics"
)

// Metrics we capture for heartbeat reporting.
var (
	capturedCounters = map[string]struct{}{
		metrics.StickyCacheHit:                      {},
		metrics.StickyCacheMiss:                     {},
		metrics.WorkflowTaskExecutionFailureCounter: {},
		metrics.ActivityExecutionFailedCounter:      {},
		metrics.LocalActivityExecutionFailedCounter: {},
		metrics.NexusTaskExecutionFailedCounter:     {},
	}

	// Timer recordings are counted (not their latencies) to track tasks processed.
	capturedTimers = map[string]struct{}{
		metrics.WorkflowTaskExecutionLatency:  {},
		metrics.ActivityExecutionLatency:      {},
		metrics.LocalActivityExecutionLatency: {},
		metrics.NexusTaskExecutionLatency:     {},
	}
)

// heartbeatMetricsHandler wraps a metrics handler and captures specific metrics
// in memory for worker heartbeats.
type heartbeatMetricsHandler struct {
	underlying metrics.Handler
	workerType string
	pollerType string

	// Keys are metric names, or "metricName:workerType" / "metricName:pollerType" for typed metrics.
	metrics *sync.Map
}

// newHeartbeatMetricsHandler creates a new handler that captures specific metrics
// for worker heartbeats while passing all metrics to the underlying handler.
func newHeartbeatMetricsHandler(underlying metrics.Handler) *heartbeatMetricsHandler {
	return &heartbeatMetricsHandler{
		underlying: underlying,
		metrics:    &sync.Map{},
	}
}

// forWorker creates a new handler that captures metrics specific to a worker type, for worker heartbeating.
// This should be called explicitly before calling WithTags on the returned handler.
func (h *heartbeatMetricsHandler) forWorker(workerType string) metrics.Handler {
	cpy := *h
	cpy.workerType = workerType
	return &cpy
}

// forPoller creates a new handler that captures metrics specific to a poller type, for worker heartbeating.
// This should be called explicitly before calling WithTags on the returned handler.
func (h *heartbeatMetricsHandler) forPoller(pollerType string) metrics.Handler {
	cpy := *h
	cpy.pollerType = pollerType
	return &cpy
}

func (h *heartbeatMetricsHandler) WithTags(tags map[string]string) metrics.Handler {
	cpy := *h
	cpy.underlying = h.underlying.WithTags(tags)
	return &cpy
}

func (h *heartbeatMetricsHandler) Counter(name string) metrics.Counter {
	underlying := h.underlying.Counter(name)
	if _, ok := capturedCounters[name]; ok {
		return &capturingCounter{
			underlying: underlying,
			value:      h.getOrCreate(name),
		}
	}
	return underlying
}

func (h *heartbeatMetricsHandler) Gauge(name string) metrics.Gauge {
	underlying := h.underlying.Gauge(name)

	switch name {
	case metrics.StickyCacheSize:
		return &capturingGauge{
			underlying: underlying,
			value:      h.getOrCreate(name),
		}
	case metrics.WorkerTaskSlotsAvailable, metrics.WorkerTaskSlotsUsed:
		if h.workerType != "" {
			return &capturingGauge{
				underlying: underlying,
				value:      h.getOrCreate(name + ":" + h.workerType),
			}
		}
	case metrics.NumPoller:
		if h.pollerType != "" {
			return &capturingGauge{
				underlying: underlying,
				value:      h.getOrCreate(name + ":" + h.pollerType),
			}
		}
	}

	return underlying
}

func (h *heartbeatMetricsHandler) Timer(name string) metrics.Timer {
	underlying := h.underlying.Timer(name)
	if _, ok := capturedTimers[name]; ok {
		return &capturingTimer{
			underlying: underlying,
			counter:    h.getOrCreate(name),
		}
	}
	return underlying
}

func (h *heartbeatMetricsHandler) getOrCreate(key string) *atomic.Int64 {
	if v, ok := h.metrics.Load(key); ok {
		return v.(*atomic.Int64)
	}
	v := new(atomic.Int64)
	actual, _ := h.metrics.LoadOrStore(key, v)
	return actual.(*atomic.Int64)
}

func (h *heartbeatMetricsHandler) get(key string) int64 {
	if v, ok := h.metrics.Load(key); ok {
		return v.(*atomic.Int64).Load()
	}
	return 0
}

// populateHeartbeatOptions contains extra information needed to populate heartbeats.
type populateHeartbeatOptions struct {
	workflowSlotSupplierKind      string
	activitySlotSupplierKind      string
	localActivitySlotSupplierKind string
	nexusSlotSupplierKind         string

	workflowPollerBehavior PollerBehavior
	activityPollerBehavior PollerBehavior
	nexusPollerBehavior    PollerBehavior

	// For delta calculations between heartbeats (mutated by PopulateHeartbeat).
	prevWorkflowProcessed      *int64
	prevWorkflowFailed         *int64
	prevActivityProcessed      *int64
	prevActivityFailed         *int64
	prevLocalActivityProcessed *int64
	prevLocalActivityFailed    *int64
	prevNexusProcessed         *int64
	prevNexusFailed            *int64

	pollTimeTracker *pollTimeTracker
}

// PopulateHeartbeat fills in the metrics-related fields of the WorkerHeartbeat proto.
func (h *heartbeatMetricsHandler) PopulateHeartbeat(hb *workerpb.WorkerHeartbeat, opts *populateHeartbeatOptions) {
	hb.TotalStickyCacheHit = int32(h.get(metrics.StickyCacheHit))
	hb.TotalStickyCacheMiss = int32(h.get(metrics.StickyCacheMiss))
	hb.CurrentStickyCacheSize = int32(h.get(metrics.StickyCacheSize))

	if opts.workflowSlotSupplierKind != "" {
		hb.WorkflowTaskSlotsInfo = buildSlotsInfo(
			opts.workflowSlotSupplierKind,
			int32(h.get(metrics.WorkerTaskSlotsAvailable+":"+"WorkflowWorker")),
			int32(h.get(metrics.WorkerTaskSlotsUsed+":"+"WorkflowWorker")),
			h.get(metrics.WorkflowTaskExecutionLatency),
			h.get(metrics.WorkflowTaskExecutionFailureCounter),
			opts.prevWorkflowProcessed,
			opts.prevWorkflowFailed,
		)
	}

	if opts.activitySlotSupplierKind != "" {
		hb.ActivityTaskSlotsInfo = buildSlotsInfo(
			opts.activitySlotSupplierKind,
			int32(h.get(metrics.WorkerTaskSlotsAvailable+":"+"ActivityWorker")),
			int32(h.get(metrics.WorkerTaskSlotsUsed+":"+"ActivityWorker")),
			h.get(metrics.ActivityExecutionLatency),
			h.get(metrics.ActivityExecutionFailedCounter),
			opts.prevActivityProcessed,
			opts.prevActivityFailed,
		)
	}

	if opts.localActivitySlotSupplierKind != "" {
		hb.LocalActivitySlotsInfo = buildSlotsInfo(
			opts.localActivitySlotSupplierKind,
			int32(h.get(metrics.WorkerTaskSlotsAvailable+":"+"LocalActivityWorker")),
			int32(h.get(metrics.WorkerTaskSlotsUsed+":"+"LocalActivityWorker")),
			h.get(metrics.LocalActivityExecutionLatency),
			h.get(metrics.LocalActivityExecutionFailedCounter),
			opts.prevLocalActivityProcessed,
			opts.prevLocalActivityFailed,
		)
	}

	if opts.nexusSlotSupplierKind != "" {
		hb.NexusTaskSlotsInfo = buildSlotsInfo(
			opts.nexusSlotSupplierKind,
			int32(h.get(metrics.WorkerTaskSlotsAvailable+":"+"NexusWorker")),
			int32(h.get(metrics.WorkerTaskSlotsUsed+":"+"NexusWorker")),
			h.get(metrics.NexusTaskExecutionLatency),
			h.get(metrics.NexusTaskExecutionFailedCounter),
			opts.prevNexusProcessed,
			opts.prevNexusFailed,
		)
	}

	hb.WorkflowPollerInfo = buildPollerInfo(
		int32(h.get(metrics.NumPoller+":"+metrics.PollerTypeWorkflowTask)),
		opts.pollTimeTracker.getLastPollTime(metrics.PollerTypeWorkflowTask),
		opts.workflowPollerBehavior,
	)
	hb.WorkflowStickyPollerInfo = buildPollerInfo(
		int32(h.get(metrics.NumPoller+":"+metrics.PollerTypeWorkflowStickyTask)),
		opts.pollTimeTracker.getLastPollTime(metrics.PollerTypeWorkflowStickyTask),
		opts.workflowPollerBehavior,
	)
	hb.ActivityPollerInfo = buildPollerInfo(
		int32(h.get(metrics.NumPoller+":"+metrics.PollerTypeActivityTask)),
		opts.pollTimeTracker.getLastPollTime(metrics.PollerTypeActivityTask),
		opts.activityPollerBehavior,
	)
	hb.NexusPollerInfo = buildPollerInfo(
		int32(h.get(metrics.NumPoller+":"+metrics.PollerTypeNexusTask)),
		opts.pollTimeTracker.getLastPollTime(metrics.PollerTypeNexusTask),
		opts.nexusPollerBehavior,
	)
}

func (h *heartbeatMetricsHandler) Unwrap() metrics.Handler {
	return h.underlying
}

func buildSlotsInfo(
	supplierKind string,
	slotsAvailable int32,
	slotsUsed int32,
	totalProcessed int64,
	totalFailed int64,
	prevProcessed *int64,
	prevFailed *int64,
) *workerpb.WorkerSlotsInfo {
	intervalProcessed := totalProcessed - *prevProcessed
	intervalFailed := totalFailed - *prevFailed

	*prevProcessed = totalProcessed
	*prevFailed = totalFailed

	return &workerpb.WorkerSlotsInfo{
		CurrentAvailableSlots:      slotsAvailable,
		CurrentUsedSlots:           slotsUsed,
		SlotSupplierKind:           supplierKind,
		TotalProcessedTasks:        int32(totalProcessed),
		TotalFailedTasks:           int32(totalFailed),
		LastIntervalProcessedTasks: int32(intervalProcessed),
		LastIntervalFailureTasks:   int32(intervalFailed),
	}
}

func buildPollerInfo(currentPollers int32, lastSuccessfulPollTime time.Time, pollerBehavior PollerBehavior) *workerpb.WorkerPollerInfo {
	var isAutoscaling bool
	switch pollerBehavior.(type) {
	case *pollerBehaviorAutoscaling:
		isAutoscaling = true
	}
	var pollTime *timestamppb.Timestamp
	if !lastSuccessfulPollTime.IsZero() {
		pollTime = timestamppb.New(lastSuccessfulPollTime)
	}

	return &workerpb.WorkerPollerInfo{
		CurrentPollers:         currentPollers,
		LastSuccessfulPollTime: pollTime,
		IsAutoscaling:          isAutoscaling,
	}
}

// capturingCounter wraps a counter and captures its value in memory.
type capturingCounter struct {
	underlying metrics.Counter
	value      *atomic.Int64
}

func (c *capturingCounter) Inc(delta int64) {
	c.underlying.Inc(delta)
	if delta > 0 {
		c.value.Add(delta)
	}
}

// capturingGauge wraps a gauge and captures its value in memory.
type capturingGauge struct {
	underlying metrics.Gauge
	value      *atomic.Int64
}

func (g *capturingGauge) Update(f float64) {
	g.underlying.Update(f)
	g.value.Store(int64(f))
}

// capturingTimer wraps a timer and increments a counter each time Record is called.
type capturingTimer struct {
	underlying metrics.Timer
	counter    *atomic.Int64
}

func (t *capturingTimer) Record(d time.Duration) {
	t.underlying.Record(d)
	t.counter.Add(1)
}
