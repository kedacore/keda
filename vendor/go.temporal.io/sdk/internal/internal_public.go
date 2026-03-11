package internal

// WARNING! WARNING! WARNING! WARNING! WARNING! WARNING! WARNING! WARNING! WARNING!
// Any of the APIs in this file are not supported for application level developers
// and are subject to change without any notice.
//
// APIs that are internal to Temporal system developers and are public from the Go
// point of view only to access them from other packages.

import (
	"time"

	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/workflowservice/v1"
)

type (
	workflowTaskHeartbeatFunc func(taskCompletion *workflowTaskCompletion, startTime time.Time) (*workflowTask, error)

	// HistoryIterator iterator through history events
	HistoryIterator interface {
		// GetNextPage returns next page of history events
		GetNextPage() (*historypb.History, error)
		// Reset resets the internal state so next GetNextPage() call will return first page of events from beginning.
		Reset()
		// HasNextPage returns if there are more page of events
		HasNextPage() bool
	}

	// WorkflowExecutionContext represents one instance of workflow execution state in memory. Lock must be obtained before
	// calling into any methods.
	WorkflowExecutionContext interface {
		Lock()
		Unlock(err error)
		ProcessWorkflowTask(workflowTask *workflowTask) (taskCompletion *workflowTaskCompletion, err error)
		ProcessLocalActivityResult(workflowTask *workflowTask, lar *localActivityResult) (*workflowTaskCompletion, error)
		// CompleteWorkflowTask try to complete current workflow task and get response that needs to be sent back to server.
		// The waitLocalActivity is used to control if we should wait for outstanding local activities.
		// If there is no outstanding local activities or if waitLocalActivity is false, the complete will return response
		// which will be one of following:
		//  - RespondWorkflowTaskCompletedRequest
		//  - RespondWorkflowTaskFailedRequest
		//  - RespondQueryTaskCompletedRequest
		// If waitLocalActivity is true, and there is outstanding local activities, this call will return nil.
		CompleteWorkflowTask(workflowTask *workflowTask, waitLocalActivity bool) workflowTaskCompletion
		// GetWorkflowTaskTimeout returns the WorkflowTaskTimeout
		GetWorkflowTaskTimeout() time.Duration
		GetCurrentWorkflowTask() *workflowservice.PollWorkflowTaskQueueResponse
		IsDestroyed() bool
		StackTrace() string
	}

	EventLevelResetter func(int64)

	// WorkflowTaskHandler represents workflow task handlers.
	WorkflowTaskHandler interface {
		WorkflowContextManager

		// Processes the workflow task
		// The response could be:
		//  - RespondWorkflowTaskCompletedRequest
		//  - RespondWorkflowTaskFailedRequest
		//  - RespondQueryTaskCompletedRequest
		ProcessWorkflowTask(
			task *workflowTask,
			ctx *workflowExecutionContextImpl,
			f workflowTaskHeartbeatFunc,
		) (taskCompletion *workflowTaskCompletion, err error)
	}

	WorkflowContextManager interface {
		// GetOrCreateWorkflowContext finds an existing cached context object
		// for the provided task's run ID or creates a new object, adds it to
		// cache, and returns it. In all non-error cases the returned context
		// object is in a locked state (i.e.
		// workflowExecutionContextImpl.Lock() has been called).
		GetOrCreateWorkflowContext(
			task *workflowservice.PollWorkflowTaskQueueResponse,
			historyIterator HistoryIterator,
		) (*workflowExecutionContextImpl, error)
	}

	// ActivityTaskHandler represents activity task handlers.
	ActivityTaskHandler interface {
		// Executes the activity task
		// The response is one of the types:
		//  - RespondActivityTaskCompletedRequest
		//  - RespondActivityTaskFailedRequest
		//  - RespondActivityTaskCanceledRequest
		Execute(taskQueue string, task *workflowservice.PollActivityTaskQueueResponse) (interface{}, error)
	}
)

var enableVerboseLogging = false

// EnableVerboseLogging enable or disable verbose logging. This is for internal use only.
func EnableVerboseLogging(enable bool) {
	enableVerboseLogging = enable
}
