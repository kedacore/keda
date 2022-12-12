package racing

import (
	"context"
	"time"
)

type LockTimeoutError string

func (l LockTimeoutError) Error() string {
	return string(l)
}

type Mutex interface {
	// TryLock attempts to acquire the lock before the deadline set by the provided context
	// If the context does not define a deadline, TryLock will block until the lock is acquired
	// Returns true if the lock is acquired, false otherwise
	TryLock(ctx context.Context) bool
	// Unlock frees this lock, if locked
	// It is a run-time error if the lock is not locked on entry to Unlock.
	Unlock()
}

type contextLock struct {
	ch chan struct{}
}

func NewMutex() Mutex {
	return &contextLock{ch: make(chan struct{}, 1)}
}

func (c *contextLock) TryLock(ctx context.Context) bool {
	deadline, hasDeadline := ctx.Deadline()
	err := ctx.Err()
	switch {
	case !hasDeadline && err == nil:
		c.ch <- struct{}{}
		return true
	case deadline.Before(time.Now()) || err != nil:
		return false
	}

	select {
	case c.ch <- struct{}{}:
		return true
	case <-ctx.Done():
		return false
	}
}

func (c *contextLock) Unlock() {
	select {
	case <-c.ch:
		return
	default:
		panic("Lock is not locked")
	}

}
