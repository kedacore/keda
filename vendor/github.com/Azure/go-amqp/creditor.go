package amqp

import (
	"context"
	"errors"
	"sync"
)

type creditor struct {
	mu sync.Mutex

	// future values for the next flow frame.
	pendingDrain bool
	creditsToAdd uint32

	// drained is set when a drain is active and we're waiting
	// for the corresponding flow from the remote.
	drained chan struct{}
}

var (
	errLinkDraining    = errors.New("link is currently draining, no credits can be added")
	errAlreadyDraining = errors.New("drain already in process")
)

// EndDrain ends the current drain, unblocking any active Drain calls.
func (mc *creditor) EndDrain() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.drained != nil {
		close(mc.drained)
		mc.drained = nil
	}
}

// FlowBits gets gets the proper values for the next flow frame
// and resets the internal state.
// Returns:
//
//	(drain: true, credits: 0) if a flow is needed (drain)
//	(drain: false, credits > 0) if a flow is needed (issue credit)
//	(drain: false, credits == 0) if no flow needed.
func (mc *creditor) FlowBits(currentCredits uint32) (bool, uint32) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	drain := mc.pendingDrain
	var credits uint32

	if mc.pendingDrain {
		// only send one drain request
		mc.pendingDrain = false
	}

	// either:
	// drain is true (ie, we're going to send a drain frame, and the credits for it should be 0)
	// mc.creditsToAdd == 0 (no flow frame needed, no new credits are being issued)
	if drain || mc.creditsToAdd == 0 {
		credits = 0
	} else {
		credits = mc.creditsToAdd + currentCredits
	}

	mc.creditsToAdd = 0

	return drain, credits
}

// Drain initiates a drain and blocks until EndDrain is called.
// If the context's deadline expires or is cancelled before the operation
// completes, the drain might not have happened.
func (mc *creditor) Drain(ctx context.Context, r *Receiver) error {
	mc.mu.Lock()

	if mc.drained != nil {
		mc.mu.Unlock()
		return errAlreadyDraining
	}

	mc.drained = make(chan struct{})
	// use a local copy to avoid racing with EndDrain()
	drained := mc.drained
	mc.pendingDrain = true

	mc.mu.Unlock()

	// cause mux() to check our flow conditions.
	select {
	case r.receiverReady <- struct{}{}:
	default:
	}

	// send drain, wait for responding flow frame
	select {
	case <-drained:
		return nil
	case <-r.l.done:
		return r.l.doneErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

// IssueCredit queues up additional credits to be requested at the next
// call of FlowBits()
func (mc *creditor) IssueCredit(credits uint32) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.drained != nil {
		return errLinkDraining
	}

	mc.creditsToAdd += credits
	return nil
}
