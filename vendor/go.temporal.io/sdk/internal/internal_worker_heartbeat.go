package internal

import (
	"context"
	"fmt"
	ilog "go.temporal.io/sdk/internal/log"
	"sync"
	"sync/atomic"
	"time"

	workerpb "go.temporal.io/api/worker/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// heartbeatManager manages heartbeat workers across namespaces for a client.
type heartbeatManager struct {
	client   *WorkflowClient
	interval time.Duration
	logger   log.Logger

	workersMutex sync.Mutex
	workers      map[string]*sharedNamespaceWorker // namespace -> worker
}

// newHeartbeatManager creates a new heartbeatManager.
func newHeartbeatManager(client *WorkflowClient, interval time.Duration, logger log.Logger) *heartbeatManager {
	if logger == nil {
		logger = ilog.NewDefaultLogger()
	}
	return &heartbeatManager{
		client:   client,
		interval: interval,
		logger:   logger,
		workers:  make(map[string]*sharedNamespaceWorker),
	}
}

// registerWorker registers a worker's heartbeat callback with the shared heartbeat worker for the namespace.
func (m *heartbeatManager) registerWorker(
	worker *AggregatedWorker,
) error {
	capabilities, err := m.client.loadNamespaceCapabilities(worker.heartbeatMetrics)
	if err != nil {
		return fmt.Errorf("failed to get namespace capabilities: %w", err)
	}
	if !capabilities.GetWorkerHeartbeats() {
		if m.logger != nil {
			m.logger.Debug("Worker heartbeating configured, but server version does not support it.")
		}
		return nil
	}

	namespace := worker.executionParams.Namespace
	m.workersMutex.Lock()
	defer m.workersMutex.Unlock()

	hw, ok := m.workers[namespace]
	// If this is the first worker on the namespace, start a new shared namespace worker.
	if !ok {
		heartbeatCtx, heartbeatCancel := context.WithCancel(context.Background())
		hw = &sharedNamespaceWorker{
			client:          m.client,
			namespace:       namespace,
			interval:        m.interval,
			heartbeatCtx:    heartbeatCtx,
			heartbeatCancel: heartbeatCancel,
			callbacks:       make(map[string]func() *workerpb.WorkerHeartbeat),
			stopC:           make(chan struct{}),
			stoppedC:        make(chan struct{}),
			logger:          m.logger,
		}
		m.workers[namespace] = hw
		if hw.started.Swap(true) {
			panic("heartbeat worker already started")
		}
		go hw.run()
	}

	hw.callbacksMutex.Lock()
	hw.callbacks[worker.workerInstanceKey] = worker.heartbeatCallback
	hw.callbacksMutex.Unlock()

	return nil
}

// unregisterWorker removes a worker's heartbeat callback. If no callbacks remain for the namespace,
// the shared heartbeat worker is stopped.
func (m *heartbeatManager) unregisterWorker(worker *AggregatedWorker) {
	m.workersMutex.Lock()
	defer m.workersMutex.Unlock()

	namespace := worker.executionParams.Namespace
	hw, ok := m.workers[namespace]
	if !ok {
		return
	}

	hw.callbacksMutex.Lock()
	delete(hw.callbacks, worker.workerInstanceKey)
	remaining := len(hw.callbacks)
	hw.callbacksMutex.Unlock()

	if remaining == 0 {
		hw.stop()
		delete(m.workers, namespace)
	}
}

// sharedNamespaceWorker handles heartbeating for all workers in a specific namespace for a specific client.
type sharedNamespaceWorker struct {
	client    *WorkflowClient
	namespace string
	interval  time.Duration
	logger    log.Logger

	heartbeatCtx    context.Context
	heartbeatCancel context.CancelFunc

	// callbacksMutex should only be unlocked under
	callbacksMutex sync.RWMutex
	callbacks      map[string]func() *workerpb.WorkerHeartbeat // workerInstanceKey -> callback

	stopC    chan struct{}
	stoppedC chan struct{}
	started  atomic.Bool
}

func (hw *sharedNamespaceWorker) run() {
	defer close(hw.stoppedC)

	ticker := time.NewTicker(hw.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := hw.sendHeartbeats(); err != nil {
				hw.logger.Warn("Stopping heartbeat worker", "error", err)
				return
			}
		case <-hw.stopC:
			return
		}
	}
}

func (hw *sharedNamespaceWorker) sendHeartbeats() error {
	hw.callbacksMutex.RLock()
	callbacks := make([]func() *workerpb.WorkerHeartbeat, 0, len(hw.callbacks))
	for _, cb := range hw.callbacks {
		callbacks = append(callbacks, cb)
	}
	hw.callbacksMutex.RUnlock()

	if len(callbacks) == 0 {
		return nil
	}

	heartbeats := make([]*workerpb.WorkerHeartbeat, 0, len(callbacks))
	for _, cb := range callbacks {
		hb := cb()
		heartbeats = append(heartbeats, hb)
	}

	_, err := hw.client.recordWorkerHeartbeat(hw.heartbeatCtx, &workflowservice.RecordWorkerHeartbeatRequest{
		Namespace:       hw.namespace,
		WorkerHeartbeat: heartbeats,
	})

	if err != nil {
		if status.Code(err) == codes.Unimplemented {
			// Server doesn't support heartbeats; return error to stop the worker.
			return fmt.Errorf("server does not support worker heartbeats: %w", err)
		}
		// For other errors, log and continue heartbeating
		hw.logger.Warn("Failed to send heartbeat", "Error", err)
	}
	return nil
}

func (hw *sharedNamespaceWorker) stop() {
	if !hw.started.CompareAndSwap(true, false) {
		return
	}
	if hw.heartbeatCancel != nil {
		hw.heartbeatCancel()
	}

	close(hw.stopC)
	<-hw.stoppedC
}

// pollTimeTracker tracks the last successful poll time for each poller type.
type pollTimeTracker struct {
	times sync.Map // pollerType (string) -> time.Time (stored as int64 nanos)
}

func (p *pollTimeTracker) recordPollSuccess(pollerType string) {
	p.times.Store(pollerType, time.Now().UnixNano())
}

func (p *pollTimeTracker) getLastPollTime(pollerType string) time.Time {
	if v, ok := p.times.Load(pollerType); ok {
		return time.Unix(0, v.(int64))
	}
	return time.Time{}
}
