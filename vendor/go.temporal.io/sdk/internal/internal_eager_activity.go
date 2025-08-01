package internal

import (
	"fmt"
	"sync"

	"go.temporal.io/api/workflowservice/v1"
)

// eagerActivityExecutor is a worker-scoped executor for eager activities that
// are returned from workflow task completion responses.
type eagerActivityExecutor struct {
	eagerActivityExecutorOptions

	activityWorker eagerWorker
	heldSlotCount  int
	countLock      sync.Mutex
}

type eagerActivityExecutorOptions struct {
	disabled  bool
	taskQueue string
	// If 0, there is no maximum
	maxConcurrent int
}

// newEagerActivityExecutor creates a new worker-scoped executor without an
// activityWorker set. The activityWorker must be set on the responding executor
// before it will be able to execute activities.
func newEagerActivityExecutor(options eagerActivityExecutorOptions) *eagerActivityExecutor {
	return &eagerActivityExecutor{eagerActivityExecutorOptions: options}
}

func (e *eagerActivityExecutor) applyToRequest(
	req *workflowservice.RespondWorkflowTaskCompletedRequest,
) []*SlotPermit {
	// Don't allow more than this hardcoded amount per workflow task for now
	const maxPerTask = 3
	reservedPermits := make([]*SlotPermit, 0)

	// Go over every command checking for activities that can be eagerly executed
	eagerRequestsThisTask := 0
	for _, command := range req.Commands {
		if attrs := command.GetScheduleActivityTaskCommandAttributes(); attrs != nil {
			// If not present, disabled, not requested, no activity worker, on a
			// different task queue, or reached max for task, we must mark as
			// explicitly disabled
			eagerDisallowed := e == nil ||
				e.disabled ||
				!attrs.RequestEagerExecution ||
				e.activityWorker == nil ||
				e.taskQueue != attrs.TaskQueue.GetName() ||
				eagerRequestsThisTask >= maxPerTask
			if eagerDisallowed {
				attrs.RequestEagerExecution = false
			} else {
				// If it has been requested, attempt to reserve one pending
				maybePermit := e.reserveOnePendingSlot()
				if maybePermit != nil {
					reservedPermits = append(reservedPermits, maybePermit)
					attrs.RequestEagerExecution = true
					eagerRequestsThisTask++
				} else {
					attrs.RequestEagerExecution = false
				}
			}
		}
	}
	return reservedPermits
}

func (e *eagerActivityExecutor) reserveOnePendingSlot() *SlotPermit {
	// Confirm that, if we have a max, issued count isn't already there
	e.countLock.Lock()
	defer e.countLock.Unlock()
	// Confirm that, if we have a max, held count isn't already there
	if e.maxConcurrent > 0 && e.heldSlotCount >= e.maxConcurrent {
		// No more room
		return nil
	}
	// Reserve a spot for our request via a non-blocking attempt
	maybePermit := e.activityWorker.tryReserveSlot()
	if maybePermit != nil {
		// Ensure that on release we decrement the held count
		maybePermit.extraReleaseCallback = func() {
			e.countLock.Lock()
			defer e.countLock.Unlock()
			e.heldSlotCount--
		}
		e.heldSlotCount++
	}
	return maybePermit
}

func (e *eagerActivityExecutor) handleResponse(
	resp *workflowservice.RespondWorkflowTaskCompletedResponse,
	reservedPermits []*SlotPermit,
) {
	// Ignore disabled or none present
	amountSlotsReserved := len(reservedPermits)
	if e == nil || e.activityWorker == nil || e.disabled ||
		(len(resp.GetActivityTasks()) == 0 && amountSlotsReserved == 0) {
		return
	} else if len(resp.GetActivityTasks()) > amountSlotsReserved {
		panic(fmt.Sprintf("Unexpectedly received %v eager activities though we only requested %v",
			len(resp.GetActivityTasks()), amountSlotsReserved))
	}

	// Give back unfulfilled slots and record for later use
	unfulfilledSlots := amountSlotsReserved - len(resp.GetActivityTasks())
	// Release unneeded permits
	for i := 0; i < unfulfilledSlots; i++ {
		unneededPermit := reservedPermits[len(reservedPermits)-1]
		reservedPermits = reservedPermits[:len(reservedPermits)-1]
		e.activityWorker.releaseSlot(unneededPermit, SlotReleaseReasonUnused)
	}

	// Start each activity asynchronously
	for i, activity := range resp.GetActivityTasks() {
		// Asynchronously execute
		e.activityWorker.pushEagerTask(
			eagerTask{
				task:   &activityTask{task: activity, permit: reservedPermits[i]},
				permit: reservedPermits[i],
			})
	}
}
