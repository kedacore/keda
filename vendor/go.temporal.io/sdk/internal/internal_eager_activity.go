// The MIT License
//
// Copyright (c) 2022 Temporal Technologies Inc.  All rights reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

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

	activityWorker *activityWorker
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
) (amountActivitySlotsReserved int) {
	// Don't allow more than this hardcoded amount per workflow task for now
	const maxPerTask = 3

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
				attrs.RequestEagerExecution = e.reserveOnePendingSlot()
				if attrs.RequestEagerExecution {
					amountActivitySlotsReserved++
					eagerRequestsThisTask++
				}
			}
		}
	}
	return
}

func (e *eagerActivityExecutor) reserveOnePendingSlot() bool {
	// Lock during count checks. Nothing in here blocks including the channel
	// receive to serve a slot.
	e.countLock.Lock()
	defer e.countLock.Unlock()
	// Confirm that, if we have a max, held count isn't already there
	if e.maxConcurrent > 0 && e.heldSlotCount >= e.maxConcurrent {
		// No more room
		return false
	}
	// Reserve a spot for our request via a non-blocking attempt to take a poller
	// request entry which essentially reserves a spot
	select {
	case <-e.activityWorker.worker.pollerRequestCh:
	default:
		return false
	}

	// We can request, so increase the held count
	e.heldSlotCount++
	return true
}

func (e *eagerActivityExecutor) handleResponse(
	resp *workflowservice.RespondWorkflowTaskCompletedResponse,
	amountActivitySlotsReserved int,
) {
	// Ignore disabled or none present
	if e == nil || e.activityWorker == nil || e.disabled || (len(resp.GetActivityTasks()) == 0 && amountActivitySlotsReserved == 0) {
		return
	} else if len(resp.GetActivityTasks()) > amountActivitySlotsReserved {
		panic(fmt.Sprintf("Unexpectedly received %v eager activities though we only requested %v",
			len(resp.GetActivityTasks()), amountActivitySlotsReserved))
	}

	// Update counts under lock
	e.countLock.Lock()
	// Give back unfulfilled slots and record for later use
	unfulfilledSlots := amountActivitySlotsReserved - len(resp.GetActivityTasks())
	e.heldSlotCount -= unfulfilledSlots
	e.countLock.Unlock()

	// Put every unfulfilled slot back on the poller channel
	for i := 0; i < unfulfilledSlots; i++ {
		// Like other parts that push onto this channel, we assume there is room
		// because we took it, so we do a blocking send
		e.activityWorker.worker.pollerRequestCh <- struct{}{}
	}

	// Start each activity asynchronously
	for _, activity := range resp.GetActivityTasks() {
		// Before starting the goroutine we have to increase the wait group counter
		// that the poller would have otherwise increased
		e.activityWorker.worker.stopWG.Add(1)
		// Asynchronously execute
		task := &activityTask{activity}
		go func() {
			// Mark completed when complete
			defer func() {
				// Like other sends to this channel, we assume there is room because we
				// reserved it, so we make a blocking send. The processTask does not do
				// this itself because our task is *activityTask, not *polledTask.
				e.activityWorker.worker.pollerRequestCh <- struct{}{}
				// Decrement executing count
				e.countLock.Lock()
				e.heldSlotCount--
				e.countLock.Unlock()
			}()

			// Process the task synchronously. We call the processor on the base
			// worker instead of a higher level so we can get the benefits of metrics,
			// stop wait group update, etc.
			e.activityWorker.worker.processTask(task)
		}()
	}
}
