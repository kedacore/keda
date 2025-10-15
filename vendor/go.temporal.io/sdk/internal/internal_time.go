package internal

import (
	"time"
)

// All code in this file is private to the package.

type (
	// TimerID contains id of the timer
	TimerID struct {
		id string
	}

	// WorkflowTimerClient wraps the async workflow timer functionality.
	WorkflowTimerClient interface {

		// Now - Current time when the workflow task is started or replayed.
		// the workflow need to use this for wall clock to make the flow logic deterministic.
		Now() time.Time

		// NewTimer - Creates a new timer that will fire callback after d(resolution is in seconds).
		// The callback indicates the error(TimerCanceledError) if the timer is canceled.
		NewTimer(d time.Duration, options TimerOptions, callback ResultHandler) *TimerID

		// RequestCancelTimer - Requests cancel of a timer, this one doesn't wait for cancellation request
		// to complete, instead invokes the ResultHandler with TimerCanceledError
		// If the timer is not started then it is a no-operation.
		RequestCancelTimer(timerID TimerID)
	}
)

func (i TimerID) String() string {
	return i.id
}

// ParseTimerID returns TimerID constructed from its string representation.
// The string representation should be obtained through TimerID.String()
func ParseTimerID(id string) (TimerID, error) {
	return TimerID{id: id}, nil
}
