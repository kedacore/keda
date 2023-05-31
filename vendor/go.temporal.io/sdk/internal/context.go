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
	"fmt"
	"sync"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
)

// Context is a clone of context.Context with Done() returning Channel instead
// of native channel.
// A Context carries a deadline, a cancellation signal, and other values across
// API boundaries.
//
// Context's methods may be called by multiple goroutines simultaneously.
type Context interface {
	// Deadline returns the time when work done on behalf of this context
	// should be canceled.  Deadline returns ok==false when no deadline is
	// set.  Successive calls to Deadline return the same results.
	Deadline() (deadline time.Time, ok bool)

	// Done returns a channel that's closed when work done on behalf of this
	// context should be canceled.  Done may return nil if this context can
	// never be canceled.  Successive calls to Done return the same value.
	//
	// WithCancel arranges for Done to be closed when cancel is called;
	// WithDeadline arranges for Done to be closed when the deadline
	// expires; WithTimeout arranges for Done to be closed when the timeout
	// elapses.
	//
	// Done is provided for use in select statements:
	//
	//  // Stream generates values with DoSomething and sends them to out
	//  // until DoSomething returns an error or ctx.Done is closed.
	//  func Stream(ctx Context, out Channel) (err error) {
	//	for {
	//		v, err := DoSomething(ctx)
	//		if err != nil {
	//			return err
	//		}
	//		s := NewSelector(ctx)
	//		s.AddReceive(ctx.Done(),  func(v interface{}) { err = ctx.Err() })
	//		s.AddReceive(v, func(v interface{}, more bool) { out.Send(ctx, v) })
	//		s.Select(ctx)
	//		if err != nil {
	//			return err
	//		}
	//	}
	//  }
	//
	// See http://blog.golang.org/pipelines for more examples of how to use
	// a Done channel for cancellation.
	Done() Channel

	// Err returns a non-nil error value after Done is closed.  Err returns
	// Canceled if the context was canceled or DeadlineExceeded if the
	// context's deadline passed.  No other values for Err are defined.
	// After Done is closed, successive calls to Err return the same value.
	Err() error

	// Value returns the value associated with this context for key, or nil
	// if no value is associated with key.  Successive calls to Value with
	// the same key returns the same result.
	//
	// Use context values only for request-scoped data that transits
	// processes and API boundaries, not for passing optional parameters to
	// functions.
	//
	// A key identifies a specific value in a Context.  Functions that wish
	// to store values in Context typically allocate a key in a global
	// variable then use that key as the argument to context.WithValue and
	// Context.Value.  A key can be any type that supports equality;
	// packages should define keys as an unexported type to avoid
	// collisions.
	//
	// Packages that define a Context key should provide type-safe accessors
	// for the values stores using that key:
	//
	// 	// Package user defines a User type that's stored in Contexts.
	// 	package user
	//
	// 	import "context"
	//
	// 	// User is the type of value stored in the Contexts.
	// 	type User struct {...}
	//
	// 	// key is an unexported type for keys defined in this package.
	// 	// This prevents collisions with keys defined in other packages.
	// 	type key int
	//
	// 	// userKey is the key for user.User values in Contexts.  It is
	// 	// unexported; clients use user.NewContext and user.FromContext
	// 	// instead of using this key directly.
	// 	var userKey key = 0
	//
	// 	// NewContext returns a new Context that carries value u.
	// 	func NewContext(ctx context.Context, u *User) context.Context {
	// 		return context.WithValue(ctx, userKey, u)
	// 	}
	//
	// 	// FromContext returns the User value stored in ctx, if any.
	// 	func FromContext(ctx context.Context) (*User, bool) {
	// 		u, ok := ctx.Value(userKey).(*User)
	// 		return u, ok
	// 	}
	Value(key interface{}) interface{}
}

// An emptyCtx is never canceled, has no values, and has no deadline.  It is not
// struct{}, since vars of this type must have distinct addresses.
type emptyCtx int

func (*emptyCtx) Deadline() (deadline time.Time, ok bool) {
	return
}

func (*emptyCtx) Done() Channel {
	return nil
}

func (*emptyCtx) Err() error {
	return nil
}

func (*emptyCtx) Value(_ interface{}) interface{} {
	return nil
}

func (e *emptyCtx) String() string {
	switch e {
	case background:
		return "context.Background"
	case todo:
		return "context.TODO"
	}
	return "unknown empty Context"
}

var (
	background = new(emptyCtx)
	todo       = new(emptyCtx)
)

// Background returns a non-nil, empty Context. It is never canceled, has no
// values, and has no deadline
func Background() Context {
	return background
}

// ErrCanceled is the error returned by Context.Err when the context is canceled.
var ErrCanceled = NewCanceledError()

// ErrDeadlineExceeded is the error returned by Context.Err when the context's
// deadline passes.
var ErrDeadlineExceeded = NewTimeoutError("deadline exceeded", enumspb.TIMEOUT_TYPE_SCHEDULE_TO_CLOSE, nil)

// A CancelFunc tells an operation to abandon its work.
// A CancelFunc does not wait for the work to stop.
// After the first call, subsequent calls to a CancelFunc do nothing.
type CancelFunc func()

// WithCancel returns a copy of parent with a new Done channel. The returned
// context's Done channel is closed when the returned cancel function is called
// or when the parent context's Done channel is closed, whichever happens first.
//
// Canceling this context releases resources associated with it, so code should
// call cancel as soon as the operations running in this Context complete.
func WithCancel(parent Context) (ctx Context, cancel CancelFunc) {
	c := newCancelCtx(parent)
	propagateCancel(parent, c)
	return c, func() { c.cancel(true, ErrCanceled) }
}

// NewDisconnectedContext returns a new context that won't propagate parent's cancellation to the new child context.
// One common use case is to do cleanup work after workflow is canceled.
//  err := workflow.ExecuteActivity(ctx, ActivityFoo).Get(ctx, &activityFooResult)
//  if err != nil && temporal.IsCanceledError(ctx.Err()) {
//    // activity failed, and workflow context is canceled
//    disconnectedCtx, _ := workflow.NewDisconnectedContext(ctx);
//    workflow.ExecuteActivity(disconnectedCtx, handleCancellationActivity).Get(disconnectedCtx, nil)
//    return err // workflow return CanceledError
//  }
func NewDisconnectedContext(parent Context) (ctx Context, cancel CancelFunc) {
	c := newCancelCtx(parent)
	return c, func() { c.cancel(true, ErrCanceled) }
}

// newCancelCtx returns an initialized cancelCtx.
func newCancelCtx(parent Context) *cancelCtx {
	return &cancelCtx{
		Context: parent,
		done:    NewNamedChannel(parent, "cancelCtx-done-channel"),
	}
}

// propagateCancel arranges for child to be canceled when parent is.
func propagateCancel(parent Context, child canceler) {
	if parent.Done() == nil {
		return // parent is never canceled
	}
	if p, ok := parentCancelCtx(parent); ok {
		if parentErr := p.Err(); parentErr != nil {
			// parent has already been canceled
			child.cancel(false, parentErr)
		} else {
			p.childrenLock.Lock()
			if p.children == nil {
				p.children = make(map[canceler]bool)
			}
			p.children[child] = true
			p.childrenLock.Unlock()
		}
	} else {
		go func() {
			s := NewSelector(parent)
			s.AddReceive(parent.Done(), func(c ReceiveChannel, more bool) {
				child.cancel(false, parent.Err())
			})
			s.AddReceive(child.Done(), func(c ReceiveChannel, more bool) {})
			s.Select(parent)
		}()
	}
}

// parentCancelCtx follows a chain of parent references until it finds a
// *cancelCtx.  This function understands how each of the concrete types in this
// package represents its parent.
func parentCancelCtx(parent Context) (*cancelCtx, bool) {
	for {
		switch c := parent.(type) {
		case *cancelCtx:
			return c, true
		// TODO: Uncomment once timer story is implemented
		//case *timerCtx:
		//	return c.cancelCtx, true
		case *valueCtx:
			parent = c.Context
		default:
			return nil, false
		}
	}
}

// removeChild removes a context from its parent.
func removeChild(parent Context, child canceler) {
	p, ok := parentCancelCtx(parent)
	if !ok {
		return
	}
	p.childrenLock.Lock()
	if p.children != nil {
		delete(p.children, child)
	}
	p.childrenLock.Unlock()
}

// A canceler is a context type that can be canceled directly.  The
// implementations are *cancelCtx and *timerCtx.
type canceler interface {
	cancel(removeFromParent bool, err error)
	Done() Channel
}

// A cancelCtx can be canceled.  When canceled, it also cancels any children
// that implement canceler.
type cancelCtx struct {
	Context

	done Channel // closed by the first cancel call.

	children     map[canceler]bool // set to nil by the first cancel call
	childrenLock sync.Mutex
	err          error // set to non-nil by the first cancel call
	errLock      sync.RWMutex
}

func (c *cancelCtx) Done() Channel {
	return c.done
}

func (c *cancelCtx) Err() error {
	c.errLock.RLock()
	defer c.errLock.RUnlock()
	return c.err
}

func (c *cancelCtx) String() string {
	return fmt.Sprintf("%v.WithCancel", c.Context)
}

// cancel closes c.done, cancels each of c's children, and, if
// removeFromParent is true, removes c from its parent's children.
func (c *cancelCtx) cancel(removeFromParent bool, err error) {
	if err == nil {
		panic("context: internal error: missing cancel error")
	}
	// This can be called from separate goroutines concurrently, so we use the
	// presence of the error under lock to prevent duplicate calls
	c.errLock.Lock()
	alreadyCancelled := c.err != nil
	if !alreadyCancelled {
		c.err = err
	}
	c.errLock.Unlock()
	if alreadyCancelled {
		return
	}
	c.done.Close()
	c.childrenLock.Lock()
	children := c.children
	c.children = nil
	c.childrenLock.Unlock()
	for child := range children {
		// NOTE: acquiring the child's lock while holding parent's lock.
		child.cancel(false, err)
	}

	if removeFromParent {
		removeChild(c.Context, c)
	}
}

// Commented out until workflow time API is exposed.
// WithDeadline returns a copy of the parent context with the deadline adjusted
// to be no later than d.  If the parent's deadline is already earlier than d,
// WithDeadline(parent, d) is semantically equivalent to parent.  The returned
// context's Done channel is closed when the deadline expires, when the returned
// cancel function is called, or when the parent context's Done channel is
// closed, whichever happens first.
//
// Canceling this context releases resources associated with it, so code should
// call cancel as soon as the operations running in this Context complete.
//func WithDeadline(parent Context, deadline time.Time) (Context, CancelFunc) {
//	if cur, ok := parent.Deadline(); ok && cur.Before(deadline) {
//		// The current deadline is already sooner than the new one.
//		return WithCancel(parent)
//	}
//	c := &timerCtx{
//		cancelCtx: newCancelCtx(parent),
//		deadline:  deadline,
//	}
//	propagateCancel(parent, c)
//	d := deadline.Sub(time.Now())
//	if d <= 0 {
//		c.cancel(true, DeadlineExceeded) // deadline has already passed
//		return c, func() { c.cancel(true, Canceled) }
//	}
//	if c.err == nil {
//		c.timer = time.AfterFunc(d, func() {
//			c.cancel(true, DeadlineExceeded)
//		})
//	}
//	return c, func() { c.cancel(true, Canceled) }
//}
//
//// A timerCtx carries a timer and a deadline.  It embeds a cancelCtx to
//// implement Done and Err.  It implements cancel by stopping its timer then
//// delegating to cancelCtx.cancel.
//type timerCtx struct {
//	*cancelCtx
//	timer *time.Timer // Under cancelCtx.mu.
//
//	deadline time.Time
//}
//
//func (c *timerCtx) Deadline() (deadline time.Time, ok bool) {
//	return c.deadline, true
//}
//
//func (c *timerCtx) String() string {
//	return fmt.Sprintf("%v.WithDeadline(%s [%s])", c.cancelCtx.Context, c.deadline, c.deadline.Sub(time.Now()))
//}
//
//func (c *timerCtx) cancel(removeFromParent bool, err error) {
//	c.cancelCtx.cancel(false, err)
//	if removeFromParent {
//		// Remove this timerCtx from its parent cancelCtx's children.
//		removeChild(c.cancelCtx.Context, c)
//	}
//	if c.timer != nil {
//		c.timer.Stop()
//		c.timer = nil
//	}
//}
//
// WithTimeout returns WithDeadline(parent, time.Now().Add(timeout)).
//
// Canceling this context releases resources associated with it, so code should
// call cancel as soon as the operations running in this Context complete:
//
// 	func slowOperationWithTimeout(ctx context.Context) (Result, error) {
// 		ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
// 		defer cancel()  // releases resources if slowOperation completes before timeout elapses
// 		return slowOperation(ctx)
// 	}
//func WithTimeout(parent Context, timeout time.Duration) (Context, CancelFunc) {
//	return WithDeadline(parent, time.Now().Add(timeout))
//}

// WithValue returns a copy of parent in which the value associated with key is
// val.
//
// Use context Values only for request-scoped data that transits processes and
// APIs, not for passing optional parameters to functions.
func WithValue(parent Context, key interface{}, val interface{}) Context {
	return &valueCtx{parent, key, val}
}

// A valueCtx carries a key-value pair.  It implements Value for that key and
// delegates all other calls to the embedded Context.
type valueCtx struct {
	Context
	key, val interface{}
}

func (c *valueCtx) String() string {
	return fmt.Sprintf("%v.WithValue(%#v, %#v)", c.Context, c.key, c.val)
}

func (c *valueCtx) Value(key interface{}) interface{} {
	if c.key == key {
		return c.val
	}
	return c.Context.Value(key)
}
