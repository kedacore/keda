// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
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
		NewTimer(d time.Duration, callback ResultHandler) *TimerID

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
