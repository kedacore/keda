package amqp

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Azure/go-amqp/internal/encoding"
	"github.com/Azure/go-amqp/internal/frames"
)

type messageDisposition struct {
	id    uint32
	state encoding.DeliveryState
}

// Receiver receives messages on a single AMQP link.
type Receiver struct {
	link           *link                   // underlying link
	batching       bool                    // enable batching of message dispositions
	batchMaxAge    time.Duration           // maximum time between the start n batch and sending the batch to the server
	dispositions   chan messageDisposition // message dispositions are sent on this channel when batching is enabled
	maxCredit      uint32                  // maximum allowed inflight messages
	inFlight       inFlight                // used to track message disposition when rcv-settle-mode == second
	manualCreditor *manualCreditor         // allows for credits to be managed manually (via calls to IssueCredit/DrainCredit)
}

// IssueCredit adds credits to be requested in the next flow
// request.
func (r *Receiver) IssueCredit(credit uint32) error {
	return r.link.IssueCredit(credit)
}

// DrainCredit sets the drain flag on the next flow frame and
// waits for the drain to be acknowledged.
func (r *Receiver) DrainCredit(ctx context.Context) error {
	return r.link.DrainCredit(ctx)
}

// Prefetched returns the next message that is stored in the Receiver's
// prefetch cache. It does NOT wait for the remote sender to send messages
// and returns immediately if the prefetch cache is empty. To receive from the
// prefetch and wait for messages from the remote Sender use `Receive`.
//
// When using ModeSecond, you *must* take an action on the message by calling
// one of the following: AcceptMessage, RejectMessage, ReleaseMessage, ModifyMessage.
// When using ModeFirst, the message is spontaneously Accepted at reception.
func (r *Receiver) Prefetched(ctx context.Context) (*Message, error) {
	if atomic.LoadUint32(&r.link.Paused) == 1 {
		select {
		case r.link.ReceiverReady <- struct{}{}:
		default:
		}
	}

	// non-blocking receive to ensure buffered messages are
	// delivered regardless of whether the link has been closed.
	select {
	case msg := <-r.link.Messages:
		debug(3, "Receive() non blocking %d", msg.deliveryID)
		msg.link = r.link
		return &msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// done draining messages
		return nil, nil
	}
}

// Receive returns the next message from the sender.
//
// Blocks until a message is received, ctx completes, or an error occurs.
// When using ModeSecond, you *must* take an action on the message by calling
// one of the following: AcceptMessage, RejectMessage, ReleaseMessage, ModifyMessage.
// When using ModeFirst, the message is spontaneously Accepted at reception.
func (r *Receiver) Receive(ctx context.Context) (*Message, error) {
	msg, err := r.Prefetched(ctx)

	if err != nil || msg != nil {
		return msg, err
	}

	// wait for the next message
	select {
	case msg := <-r.link.Messages:
		debug(3, "Receive() blocking %d", msg.deliveryID)
		msg.link = r.link
		return &msg, nil
	case <-r.link.Detached:
		return nil, r.link.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Accept notifies the server that the message has been
// accepted and does not require redelivery.
func (r *Receiver) AcceptMessage(ctx context.Context, msg *Message) error {
	if !msg.shouldSendDisposition() {
		return nil
	}
	return r.messageDisposition(ctx, msg, &encoding.StateAccepted{})
}

// Reject notifies the server that the message is invalid.
//
// Rejection error is optional.
func (r *Receiver) RejectMessage(ctx context.Context, msg *Message, e *Error) error {
	if !msg.shouldSendDisposition() {
		return nil
	}
	return r.messageDisposition(ctx, msg, &encoding.StateRejected{Error: e})
}

// Release releases the message back to the server. The message
// may be redelivered to this or another consumer.
func (r *Receiver) ReleaseMessage(ctx context.Context, msg *Message) error {
	if !msg.shouldSendDisposition() {
		return nil
	}
	return r.messageDisposition(ctx, msg, &encoding.StateReleased{})
}

// Modify notifies the server that the message was not acted upon
// and should be modifed.
//
// deliveryFailed indicates that the server must consider this and
// unsuccessful delivery attempt and increment the delivery count.
//
// undeliverableHere indicates that the server must not redeliver
// the message to this link.
//
// messageAnnotations is an optional annotation map to be merged
// with the existing message annotations, overwriting existing keys
// if necessary.
func (r *Receiver) ModifyMessage(ctx context.Context, msg *Message, deliveryFailed, undeliverableHere bool, messageAnnotations Annotations) error {
	if !msg.shouldSendDisposition() {
		return nil
	}
	return r.messageDisposition(ctx,
		msg, &encoding.StateModified{
			DeliveryFailed:     deliveryFailed,
			UndeliverableHere:  undeliverableHere,
			MessageAnnotations: messageAnnotations,
		})
}

// Address returns the link's address.
func (r *Receiver) Address() string {
	if r.link.Source == nil {
		return ""
	}
	return r.link.Source.Address
}

// LinkName returns associated link name or an empty string if link is not defined.
func (r *Receiver) LinkName() string {
	return r.link.Key.name
}

// LinkSourceFilterValue retrieves the specified link source filter value or nil if it doesn't exist.
func (r *Receiver) LinkSourceFilterValue(name string) interface{} {
	if r.link.Source == nil {
		return nil
	}
	filter, ok := r.link.Source.Filter[encoding.Symbol(name)]
	if !ok {
		return nil
	}
	return filter.Value
}

// Close closes the Receiver and AMQP link.
//
// If ctx expires while waiting for servers response, ctx.Err() will be returned.
// The session will continue to wait for the response until the Session or Client
// is closed.
func (r *Receiver) Close(ctx context.Context) error {
	return r.link.Close(ctx)
}

func (r *Receiver) dispositionBatcher() {
	// batch operations:
	// Keep track of the first and last delivery ID, incrementing as
	// Accept() is called. After last-first == batchSize, send disposition.
	// If Reject()/Release() is called, send one disposition for previously
	// accepted, and one for the rejected/released message. If messages are
	// accepted out of order, send any existing batch and the current message.
	var (
		batchSize    = r.maxCredit
		batchStarted bool
		first        uint32
		last         uint32
	)

	// create an unstarted timer
	batchTimer := time.NewTimer(1 * time.Minute)
	batchTimer.Stop()
	defer batchTimer.Stop()

	for {
		select {
		case msgDis := <-r.dispositions:

			// not accepted or batch out of order
			_, isAccept := msgDis.state.(*encoding.StateAccepted)
			if !isAccept || (batchStarted && last+1 != msgDis.id) {
				// send the current batch, if any
				if batchStarted {
					lastCopy := last
					err := r.sendDisposition(first, &lastCopy, &encoding.StateAccepted{})
					if err != nil {
						r.inFlight.remove(first, &lastCopy, err)
					}
					batchStarted = false
				}

				// send the current message
				err := r.sendDisposition(msgDis.id, nil, msgDis.state)
				if err != nil {
					r.inFlight.remove(msgDis.id, nil, err)
				}
				continue
			}

			if batchStarted {
				// increment last
				last++
			} else {
				// start new batch
				batchStarted = true
				first = msgDis.id
				last = msgDis.id
				batchTimer.Reset(r.batchMaxAge)
			}

			// send batch if current size == batchSize
			if last-first+1 >= batchSize {
				lastCopy := last
				err := r.sendDisposition(first, &lastCopy, &encoding.StateAccepted{})
				if err != nil {
					r.inFlight.remove(first, &lastCopy, err)
				}
				batchStarted = false
				if !batchTimer.Stop() {
					<-batchTimer.C // batch timer must be drained if stop returns false
				}
			}

		// maxBatchAge elapsed, send batch
		case <-batchTimer.C:
			lastCopy := last
			err := r.sendDisposition(first, &lastCopy, &encoding.StateAccepted{})
			if err != nil {
				r.inFlight.remove(first, &lastCopy, err)
			}
			batchStarted = false
			batchTimer.Stop()

		case <-r.link.Detached:
			return
		}
	}
}

// sendDisposition sends a disposition frame to the peer
func (r *Receiver) sendDisposition(first uint32, last *uint32, state encoding.DeliveryState) error {
	fr := &frames.PerformDisposition{
		Role:    encoding.RoleReceiver,
		First:   first,
		Last:    last,
		Settled: r.link.ReceiverSettleMode == nil || *r.link.ReceiverSettleMode == ModeFirst,
		State:   state,
	}

	debug(1, "TX (sendDisposition): %s", fr)
	return r.link.Session.txFrame(fr, nil)
}

func (r *Receiver) messageDisposition(ctx context.Context, msg *Message, state encoding.DeliveryState) error {
	var wait chan error
	if r.link.ReceiverSettleMode != nil && *r.link.ReceiverSettleMode == ModeSecond {
		debug(3, "RX (messageDisposition): add %d to inflight", msg.deliveryID)
		wait = r.inFlight.add(msg.deliveryID)
	}

	if r.batching {
		r.dispositions <- messageDisposition{id: msg.deliveryID, state: state}
	} else {
		err := r.sendDisposition(msg.deliveryID, nil, state)
		if err != nil {
			return err
		}
	}

	if wait == nil {
		return nil
	}

	select {
	case err := <-wait:
		// we've received confirmation of disposition
		r.link.DeleteUnsettled(msg)
		msg.settled = true
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// inFlight tracks in-flight message dispositions allowing receivers
// to block waiting for the server to respond when an appropriate
// settlement mode is configured.
type inFlight struct {
	mu sync.RWMutex
	m  map[uint32]chan error
}

func (f *inFlight) add(id uint32) chan error {
	wait := make(chan error, 1)

	f.mu.Lock()
	if f.m == nil {
		f.m = map[uint32]chan error{id: wait}
	} else {
		f.m[id] = wait
	}
	f.mu.Unlock()

	return wait
}

func (f *inFlight) remove(first uint32, last *uint32, err error) {
	f.mu.Lock()

	if f.m == nil {
		f.mu.Unlock()
		return
	}

	ll := first
	if last != nil {
		ll = *last
	}

	for i := first; i <= ll; i++ {
		wait, ok := f.m[i]
		if ok {
			wait <- err
			delete(f.m, i)
		}
	}

	f.mu.Unlock()
}

func (f *inFlight) clear(err error) {
	f.mu.Lock()
	for id, wait := range f.m {
		wait <- err
		delete(f.m, id)
	}
	f.mu.Unlock()
}

func (f *inFlight) len() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return len(f.m)
}
