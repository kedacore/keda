package internal

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/semaphore"

	"go.temporal.io/sdk/internal/common/metrics"
	"go.temporal.io/sdk/log"
)

// WorkerTuner allows for the dynamic customization of some aspects of worker behavior.
//
// WARNING: Custom implementations of SlotSupplier are currently experimental.
//
// Exposed as: [go.temporal.io/sdk/worker.WorkerTuner]
type WorkerTuner interface {
	// GetWorkflowTaskSlotSupplier returns the SlotSupplier used for workflow tasks.
	GetWorkflowTaskSlotSupplier() SlotSupplier
	// GetActivityTaskSlotSupplier returns the SlotSupplier used for activity tasks.
	GetActivityTaskSlotSupplier() SlotSupplier
	// GetLocalActivitySlotSupplier returns the SlotSupplier used for local activities.
	GetLocalActivitySlotSupplier() SlotSupplier
	// GetNexusSlotSupplier returns the SlotSupplier used for nexus tasks.
	GetNexusSlotSupplier() SlotSupplier
	// GetSessionActivitySlotSupplier returns the SlotSupplier used for activities within sessions.
	GetSessionActivitySlotSupplier() SlotSupplier
}

// SlotPermit is a permit to use a slot.
//
// WARNING: Custom implementations of SlotSupplier are currently experimental.
//
// Exposed as: [go.temporal.io/sdk/worker.SlotPermit]
type SlotPermit struct {
	// UserData is a field that can be used to store arbitrary on a permit by SlotSupplier
	// implementations.
	UserData any
	// Specifically eager activities need to keep track of their own concurrent max separately and
	// this helps them do that. It can be used for other specific use cases in the future.
	extraReleaseCallback func()
}

// SlotReservationInfo contains information that SlotSupplier instances can use during
// reservation calls. It embeds a standard Context.
//
// Exposed as: [go.temporal.io/sdk/worker.SlotReservationInfo]
type SlotReservationInfo interface {
	// TaskQueue returns the task queue for which a slot is being reserved. In the case of local
	// activities, this is the same as the workflow's task queue.
	TaskQueue() string
	// WorkerBuildId returns the build ID of the worker that is reserving the slot.
	WorkerBuildId() string
	// WorkerBuildId returns the build ID of the worker that is reserving the slot.
	WorkerIdentity() string
	// NumIssuedSlots returns the current number of slots that have already been issued by the
	// supplier. This value may change over the course of the reservation.
	NumIssuedSlots() int
	// Logger returns an appropriately tagged logger.
	Logger() log.Logger
	// MetricsHandler returns an appropriately tagged metrics handler that can be used to record
	// custom metrics.
	MetricsHandler() metrics.Handler
}

// SlotMarkUsedInfo contains information that SlotSupplier instances can use during
// SlotSupplier.MarkSlotUsed calls.
//
// Exposed as: [go.temporal.io/sdk/worker.SlotMarkUsedInfo]
type SlotMarkUsedInfo interface {
	// Permit returns the permit that is being marked as used.
	Permit() *SlotPermit
	// Logger returns an appropriately tagged logger.
	Logger() log.Logger
	// MetricsHandler returns an appropriately tagged metrics handler that can be used to record
	// custom metrics.
	MetricsHandler() metrics.Handler
}

// SlotReleaseReason describes the reason that a slot is being released.
type SlotReleaseReason int

const (
	SlotReleaseReasonTaskProcessed SlotReleaseReason = iota
	SlotReleaseReasonUnused
)

// SlotReleaseInfo contains information that SlotSupplier instances can use during
// SlotSupplier.ReleaseSlot calls.
//
// Exposed as: [go.temporal.io/sdk/worker.SlotReleaseInfo]
type SlotReleaseInfo interface {
	// Permit returns the permit that is being released.
	Permit() *SlotPermit
	// Reason returns the reason that the slot is being released.
	Reason() SlotReleaseReason
	// Logger returns an appropriately tagged logger.
	Logger() log.Logger
	// MetricsHandler returns an appropriately tagged metrics handler that can be used to record
	// custom metrics.
	MetricsHandler() metrics.Handler
}

// SlotSupplier controls how slots are handed out for workflow and activity tasks as well as
// local activities when used in conjunction with a WorkerTuner.
//
// WARNING: Custom implementations of SlotSupplier are currently experimental.
//
// Exposed as: [go.temporal.io/sdk/worker.SlotSupplier]
type SlotSupplier interface {
	// ReserveSlot is called before polling for new tasks. The implementation should block until
	// a slot is available, then return a permit to use that slot. Implementations must be
	// thread-safe.
	//
	// Any returned error besides context.Canceled will be logged and the function will be retried.
	ReserveSlot(ctx context.Context, info SlotReservationInfo) (*SlotPermit, error)

	// TryReserveSlot is called when attempting to reserve slots for eager workflows and activities.
	// It should return a permit if a slot is available, and nil otherwise. Implementations must be
	// thread-safe.
	TryReserveSlot(info SlotReservationInfo) *SlotPermit

	// MarkSlotUsed is called once a slot is about to be used for actually processing a task.
	// Because slots are reserved before task polling, not all reserved slots will be used.
	// Implementations must be thread-safe.
	MarkSlotUsed(info SlotMarkUsedInfo)

	// ReleaseSlot is called when a slot is no longer needed, which is typically after the task
	// has been processed, but may also be called upon shutdown or other situations where the
	// slot is no longer needed. Implementations must be thread-safe.
	ReleaseSlot(info SlotReleaseInfo)

	// MaxSlots returns the maximum number of slots that this supplier will ever issue.
	// Implementations may return 0 if there is no well-defined upper limit. In such cases the
	// available task slots metric will not be emitted.
	MaxSlots() int
}

// CompositeTuner allows you to build a tuner from multiple slot suppliers.
//
// WARNING: Custom implementations of SlotSupplier are currently experimental.
type CompositeTuner struct {
	workflowSlotSupplier        SlotSupplier
	activitySlotSupplier        SlotSupplier
	localActivitySlotSupplier   SlotSupplier
	nexusSlotSupplier           SlotSupplier
	sessionActivitySlotSupplier SlotSupplier
}

func (c *CompositeTuner) GetWorkflowTaskSlotSupplier() SlotSupplier {
	return c.workflowSlotSupplier
}
func (c *CompositeTuner) GetActivityTaskSlotSupplier() SlotSupplier {
	return c.activitySlotSupplier
}
func (c *CompositeTuner) GetLocalActivitySlotSupplier() SlotSupplier {
	return c.localActivitySlotSupplier
}
func (c *CompositeTuner) GetNexusSlotSupplier() SlotSupplier {
	return c.nexusSlotSupplier
}
func (c *CompositeTuner) GetSessionActivitySlotSupplier() SlotSupplier {
	return c.sessionActivitySlotSupplier
}

// CompositeTunerOptions are the options used by NewCompositeTuner.
//
// Exposed as: [go.temporal.io/sdk/worker.CompositeTunerOptions]
type CompositeTunerOptions struct {
	// WorkflowSlotSupplier is the SlotSupplier used for workflow tasks.
	WorkflowSlotSupplier SlotSupplier
	// ActivitySlotSupplier is the SlotSupplier used for activity tasks.
	ActivitySlotSupplier SlotSupplier
	// LocalActivitySlotSupplier is the SlotSupplier used for local activities.
	LocalActivitySlotSupplier SlotSupplier
	// NexusSlotSupplier is the SlotSupplier used for nexus tasks.
	NexusSlotSupplier SlotSupplier
	// SessionActivitySlotSupplier is the SlotSupplier used for activities within sessions.
	SessionActivitySlotSupplier SlotSupplier
}

// NewCompositeTuner creates a WorkerTuner that uses a combination of slot suppliers.
//
// WARNING: Custom implementations of SlotSupplier are currently experimental.
//
// Exposed as: [go.temporal.io/sdk/worker.NewCompositeTuner]
func NewCompositeTuner(options CompositeTunerOptions) (WorkerTuner, error) {
	return &CompositeTuner{
		workflowSlotSupplier:        options.WorkflowSlotSupplier,
		activitySlotSupplier:        options.ActivitySlotSupplier,
		localActivitySlotSupplier:   options.LocalActivitySlotSupplier,
		nexusSlotSupplier:           options.NexusSlotSupplier,
		sessionActivitySlotSupplier: options.SessionActivitySlotSupplier,
	}, nil
}

// FixedSizeTunerOptions are the options used by NewFixedSizeTuner.
//
// Exposed as: [go.temporal.io/sdk/worker.FixedSizeTunerOptions]
type FixedSizeTunerOptions struct {
	// NumWorkflowSlots is the number of slots available for workflow tasks.
	NumWorkflowSlots int
	// NumActivitySlots is the number of slots available for activity tasks.
	NumActivitySlots int
	// NumLocalActivitySlots is the number of slots available for local activities.
	NumLocalActivitySlots int
	// NumNexusSlots is the number of slots available for nexus tasks.
	NumNexusSlots int
}

// NewFixedSizeTuner creates a WorkerTuner that uses fixed size slot suppliers.
//
// Exposed as: [go.temporal.io/sdk/worker.NewFixedSizeTuner]
func NewFixedSizeTuner(options FixedSizeTunerOptions) (WorkerTuner, error) {
	if options.NumWorkflowSlots <= 0 {
		options.NumWorkflowSlots = defaultMaxConcurrentTaskExecutionSize
	}
	if options.NumActivitySlots <= 0 {
		options.NumActivitySlots = defaultMaxConcurrentActivityExecutionSize
	}
	if options.NumLocalActivitySlots <= 0 {
		options.NumLocalActivitySlots = defaultMaxConcurrentLocalActivityExecutionSize
	}
	if options.NumNexusSlots <= 0 {
		options.NumNexusSlots = defaultMaxConcurrentTaskExecutionSize
	}
	wfSS, err := NewFixedSizeSlotSupplier(options.NumWorkflowSlots)
	if err != nil {
		return nil, err
	}
	actSS, err := NewFixedSizeSlotSupplier(options.NumActivitySlots)
	if err != nil {
		return nil, err
	}
	laSS, err := NewFixedSizeSlotSupplier(options.NumLocalActivitySlots)
	if err != nil {
		return nil, err
	}
	nexusSS, err := NewFixedSizeSlotSupplier(options.NumNexusSlots)
	if err != nil {
		return nil, err
	}
	sessSS, err := NewFixedSizeSlotSupplier(options.NumActivitySlots)
	if err != nil {
		return nil, err
	}
	return &CompositeTuner{
		workflowSlotSupplier:        wfSS,
		activitySlotSupplier:        actSS,
		localActivitySlotSupplier:   laSS,
		nexusSlotSupplier:           nexusSS,
		sessionActivitySlotSupplier: sessSS,
	}, nil
}

// FixedSizeSlotSupplier is a slot supplier that will only ever issue at most a fixed number of
// slots.
type FixedSizeSlotSupplier struct {
	numSlots int
	sem      *semaphore.Weighted
}

// NewFixedSizeSlotSupplier creates a new FixedSizeSlotSupplier with the given number of slots.
//
// Exposed as: [go.temporal.io/sdk/worker.NewFixedSizeSlotSupplier]
func NewFixedSizeSlotSupplier(numSlots int) (*FixedSizeSlotSupplier, error) {
	if numSlots <= 0 {
		return nil, fmt.Errorf("NumSlots must be positive")
	}
	return &FixedSizeSlotSupplier{
		numSlots: numSlots,
		sem:      semaphore.NewWeighted(int64(numSlots)),
	}, nil
}

func (f *FixedSizeSlotSupplier) ReserveSlot(ctx context.Context, _ SlotReservationInfo) (
	*SlotPermit, error) {
	err := f.sem.Acquire(ctx, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire slot: %w", err)
	}

	return &SlotPermit{}, nil
}
func (f *FixedSizeSlotSupplier) TryReserveSlot(SlotReservationInfo) *SlotPermit {
	if f.sem.TryAcquire(1) {
		return &SlotPermit{}
	}
	return nil
}
func (f *FixedSizeSlotSupplier) MarkSlotUsed(SlotMarkUsedInfo) {}
func (f *FixedSizeSlotSupplier) ReleaseSlot(SlotReleaseInfo) {
	f.sem.Release(1)
}
func (f *FixedSizeSlotSupplier) MaxSlots() int {
	return f.numSlots
}

type slotReservationData struct {
	taskQueue string
}

type slotReserveInfoImpl struct {
	taskQueue      string
	workerBuildId  string
	workerIdentity string
	issuedSlots    *atomic.Int32
	logger         log.Logger
	metrics        metrics.Handler
}

func (s slotReserveInfoImpl) TaskQueue() string {
	return s.taskQueue
}

func (s slotReserveInfoImpl) WorkerBuildId() string {
	return s.workerBuildId
}

func (s slotReserveInfoImpl) WorkerIdentity() string {
	return s.workerIdentity
}

func (s slotReserveInfoImpl) NumIssuedSlots() int {
	return int(s.issuedSlots.Load())
}

func (s slotReserveInfoImpl) Logger() log.Logger {
	return s.logger
}

func (s slotReserveInfoImpl) MetricsHandler() metrics.Handler {
	return s.metrics
}

type slotMarkUsedContextImpl struct {
	permit  *SlotPermit
	logger  log.Logger
	metrics metrics.Handler
}

func (s slotMarkUsedContextImpl) Permit() *SlotPermit {
	return s.permit
}

func (s slotMarkUsedContextImpl) Logger() log.Logger {
	return s.logger
}

func (s slotMarkUsedContextImpl) MetricsHandler() metrics.Handler {
	return s.metrics
}

type slotReleaseContextImpl struct {
	permit  *SlotPermit
	reason  SlotReleaseReason
	logger  log.Logger
	metrics metrics.Handler
}

func (s slotReleaseContextImpl) Permit() *SlotPermit {
	return s.permit
}

func (s slotReleaseContextImpl) Reason() SlotReleaseReason {
	return s.reason
}

func (s slotReleaseContextImpl) Logger() log.Logger {
	return s.logger
}

func (s slotReleaseContextImpl) MetricsHandler() metrics.Handler {
	return s.metrics
}

type trackingSlotSupplier struct {
	inner          SlotSupplier
	logger         log.Logger
	metrics        metrics.Handler
	workerBuildId  string
	workerIdentity string

	issuedSlotsAtomic atomic.Int32
	slotsMutex        sync.Mutex
	// Values should eventually become slot info types
	usedSlots               map[*SlotPermit]struct{}
	taskSlotsAvailableGauge metrics.Gauge
	taskSlotsUsedGauge      metrics.Gauge
}

type trackingSlotSupplierOptions struct {
	logger         log.Logger
	metricsHandler metrics.Handler
	workerBuildId  string
	workerIdentity string
}

func newTrackingSlotSupplier(inner SlotSupplier, options trackingSlotSupplierOptions) *trackingSlotSupplier {
	tss := &trackingSlotSupplier{
		inner:                   inner,
		logger:                  options.logger,
		metrics:                 options.metricsHandler,
		workerBuildId:           options.workerBuildId,
		workerIdentity:          options.workerIdentity,
		usedSlots:               make(map[*SlotPermit]struct{}),
		taskSlotsAvailableGauge: options.metricsHandler.Gauge(metrics.WorkerTaskSlotsAvailable),
		taskSlotsUsedGauge:      options.metricsHandler.Gauge(metrics.WorkerTaskSlotsUsed),
	}
	return tss
}

func (t *trackingSlotSupplier) ReserveSlot(
	ctx context.Context,
	data *slotReservationData,
) (*SlotPermit, error) {
	permit, err := t.inner.ReserveSlot(ctx, slotReserveInfoImpl{
		taskQueue:      data.taskQueue,
		workerBuildId:  t.workerBuildId,
		workerIdentity: t.workerIdentity,
		issuedSlots:    &t.issuedSlotsAtomic,
		logger:         t.logger,
		metrics:        t.metrics,
	})
	if err != nil {
		return nil, err
	}
	if permit == nil {
		return nil, fmt.Errorf("slot supplier returned nil permit")
	}
	t.issuedSlotsAtomic.Add(1)
	t.slotsMutex.Lock()
	usedSlots := len(t.usedSlots)
	t.slotsMutex.Unlock()
	t.publishMetrics(usedSlots)
	return permit, nil
}

func (t *trackingSlotSupplier) TryReserveSlot(data *slotReservationData) *SlotPermit {
	permit := t.inner.TryReserveSlot(slotReserveInfoImpl{
		taskQueue:      data.taskQueue,
		workerBuildId:  t.workerBuildId,
		workerIdentity: t.workerIdentity,
		issuedSlots:    &t.issuedSlotsAtomic,
		logger:         t.logger,
		metrics:        t.metrics,
	})
	if permit != nil {
		t.issuedSlotsAtomic.Add(1)
		t.slotsMutex.Lock()
		usedSlots := len(t.usedSlots)
		t.slotsMutex.Unlock()
		t.publishMetrics(usedSlots)
	}
	return permit
}

func (t *trackingSlotSupplier) MarkSlotUsed(permit *SlotPermit) {
	if permit == nil {
		panic("Cannot mark nil permit as used")
	}
	t.slotsMutex.Lock()
	t.usedSlots[permit] = struct{}{}
	usedSlots := len(t.usedSlots)
	t.slotsMutex.Unlock()
	t.inner.MarkSlotUsed(&slotMarkUsedContextImpl{
		permit:  permit,
		logger:  t.logger,
		metrics: t.metrics,
	})
	t.publishMetrics(usedSlots)
}

func (t *trackingSlotSupplier) ReleaseSlot(permit *SlotPermit, reason SlotReleaseReason) {
	if permit == nil {
		panic("Cannot release with nil permit")
	}
	t.slotsMutex.Lock()
	delete(t.usedSlots, permit)
	usedSlots := len(t.usedSlots)
	t.slotsMutex.Unlock()
	t.inner.ReleaseSlot(&slotReleaseContextImpl{
		permit:  permit,
		reason:  reason,
		logger:  t.logger,
		metrics: t.metrics,
	})
	t.issuedSlotsAtomic.Add(-1)
	if permit.extraReleaseCallback != nil {
		permit.extraReleaseCallback()
	}
	t.publishMetrics(usedSlots)
}

func (t *trackingSlotSupplier) publishMetrics(usedSlots int) {
	if t.inner.MaxSlots() != 0 {
		t.taskSlotsAvailableGauge.Update(float64(t.inner.MaxSlots() - usedSlots))
	}
	t.taskSlotsUsedGauge.Update(float64(usedSlots))
}
