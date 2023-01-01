package amqp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Azure/go-amqp/internal/buffer"
	"github.com/Azure/go-amqp/internal/debug"
	"github.com/Azure/go-amqp/internal/encoding"
	"github.com/Azure/go-amqp/internal/frames"
	"github.com/Azure/go-amqp/internal/shared"
)

// Default link options
const (
	defaultLinkCredit      = 1
	defaultLinkBatching    = false
	defaultLinkBatchMaxAge = 5 * time.Second
)

type messageDisposition struct {
	id    uint32
	state encoding.DeliveryState
}

// Receiver receives messages on a single AMQP link.
type Receiver struct {
	l link
	// message receiving
	receiverReady         chan struct{}       // receiver sends on this when mux is paused to indicate it can handle more messages
	messages              chan Message        // used to send completed messages to receiver
	unsettledMessages     map[string]struct{} // used to keep track of messages being handled downstream
	unsettledMessagesLock sync.RWMutex        // lock to protect concurrent access to unsettledMessages
	msgBuf                buffer.Buffer       // buffered bytes for current message
	more                  bool                // if true, buf contains a partial message
	msg                   Message             // current message being decoded

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
	if r.manualCreditor == nil {
		return errors.New("issueCredit can only be used with receiver links using manual credit management")
	}

	if err := r.manualCreditor.IssueCredit(credit); err != nil {
		return err
	}

	// cause mux() to check our flow conditions.
	select {
	case r.receiverReady <- struct{}{}:
	default:
	}

	return nil
}

// DrainCredit sets the drain flag on the next flow frame and
// waits for the drain to be acknowledged.
func (r *Receiver) DrainCredit(ctx context.Context) error {
	if r.manualCreditor == nil {
		return errors.New("drain can only be used with receiver links using manual credit management")
	}

	// cause mux() to check our flow conditions.
	select {
	case r.receiverReady <- struct{}{}:
	default:
	}

	return r.manualCreditor.Drain(ctx, r)
}

// Prefetched returns the next message that is stored in the Receiver's
// prefetch cache. It does NOT wait for the remote sender to send messages
// and returns immediately if the prefetch cache is empty. To receive from the
// prefetch and wait for messages from the remote Sender use `Receive`.
//
// When using ReceiverSettleModeSecond, you *must* take an action on the message by calling
// one of the following: AcceptMessage, RejectMessage, ReleaseMessage, ModifyMessage.
// When using ReceiverSettleModeFirst, the message is spontaneously Accepted at reception.
func (r *Receiver) Prefetched() *Message {
	select {
	case r.receiverReady <- struct{}{}:
	default:
	}

	// non-blocking receive to ensure buffered messages are
	// delivered regardless of whether the link has been closed.
	select {
	case msg := <-r.messages:
		debug.Log(3, "Receive() non blocking %d", msg.deliveryID)
		msg.rcvr = r
		return &msg
	default:
		// done draining messages
		return nil
	}
}

// Receive returns the next message from the sender.
//
// Blocks until a message is received, ctx completes, or an error occurs.
// When using ReceiverSettleModeSecond, you *must* take an action on the message by calling
// one of the following: AcceptMessage, RejectMessage, ReleaseMessage, ModifyMessage.
// When using ReceiverSettleModeFirst, the message is spontaneously Accepted at reception.
func (r *Receiver) Receive(ctx context.Context) (*Message, error) {
	if msg := r.Prefetched(); msg != nil {
		return msg, nil
	}

	// wait for the next message
	select {
	case msg := <-r.messages:
		debug.Log(3, "Receive() blocking %d", msg.deliveryID)
		msg.rcvr = r
		return &msg, nil
	case <-r.l.detached:
		return nil, r.l.err
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

// Modify notifies the server that the message was not acted upon and should be modifed.
func (r *Receiver) ModifyMessage(ctx context.Context, msg *Message, options *ModifyMessageOptions) error {
	if !msg.shouldSendDisposition() {
		return nil
	}
	if options == nil {
		options = &ModifyMessageOptions{}
	}
	return r.messageDisposition(ctx,
		msg, &encoding.StateModified{
			DeliveryFailed:     options.DeliveryFailed,
			UndeliverableHere:  options.UndeliverableHere,
			MessageAnnotations: options.Annotations,
		})
}

// ModifyMessageOptions contains the optional parameters to ModifyMessage.
type ModifyMessageOptions struct {
	// DeliveryFailed indicates that the server must consider this an
	// unsuccessful delivery attempt and increment the delivery count.
	DeliveryFailed bool

	// UndeliverableHere indicates that the server must not redeliver
	// the message to this link.
	UndeliverableHere bool

	// Annotations is an optional annotation map to be merged
	// with the existing message annotations, overwriting existing keys
	// if necessary.
	Annotations Annotations
}

// Address returns the link's address.
func (r *Receiver) Address() string {
	if r.l.source == nil {
		return ""
	}
	return r.l.source.Address
}

// LinkName returns associated link name or an empty string if link is not defined.
func (r *Receiver) LinkName() string {
	return r.l.key.name
}

// LinkSourceFilterValue retrieves the specified link source filter value or nil if it doesn't exist.
func (r *Receiver) LinkSourceFilterValue(name string) any {
	if r.l.source == nil {
		return nil
	}
	filter, ok := r.l.source.Filter[encoding.Symbol(name)]
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
	return r.l.closeLink(ctx)
}

// returns the error passed in
func (r *Receiver) closeWithError(de *Error) error {
	r.l.closeOnce.Do(func() {
		r.l.detachErrorMu.Lock()
		r.l.detachError = de
		r.l.detachErrorMu.Unlock()
		close(r.l.close)
	})
	return &DetachError{inner: de}
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

		case <-r.l.detached:
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
		Settled: r.l.receiverSettleMode == nil || *r.l.receiverSettleMode == ReceiverSettleModeFirst,
		State:   state,
	}

	select {
	case <-r.l.detached:
		return r.l.err
	default:
		debug.Log(1, "TX (sendDisposition): %s", fr)
		return r.l.session.txFrame(fr, nil)
	}
}

func (r *Receiver) messageDisposition(ctx context.Context, msg *Message, state encoding.DeliveryState) error {
	var wait chan error
	if r.l.receiverSettleMode != nil && *r.l.receiverSettleMode == ReceiverSettleModeSecond {
		debug.Log(3, "RX (messageDisposition): add %d to inflight", msg.deliveryID)
		wait = r.inFlight.add(msg.deliveryID)
	}

	if r.batching {
		select {
		case r.dispositions <- messageDisposition{id: msg.deliveryID, state: state}:
		case <-r.l.detached:
			return r.l.err
		}
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
		r.deleteUnsettled(msg)
		msg.settled = true
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *Receiver) addUnsettled(msg *Message) {
	r.unsettledMessagesLock.Lock()
	r.unsettledMessages[string(msg.DeliveryTag)] = struct{}{}
	r.unsettledMessagesLock.Unlock()
}

func (r *Receiver) deleteUnsettled(msg *Message) {
	r.unsettledMessagesLock.Lock()
	delete(r.unsettledMessages, string(msg.DeliveryTag))
	r.unsettledMessagesLock.Unlock()
}

func (r *Receiver) countUnsettled() int {
	r.unsettledMessagesLock.RLock()
	count := len(r.unsettledMessages)
	r.unsettledMessagesLock.RUnlock()
	return count
}

func newReceiver(source string, session *Session, opts *ReceiverOptions) (*Receiver, error) {
	r := &Receiver{
		l: link{
			key:      linkKey{shared.RandString(40), encoding.RoleReceiver},
			session:  session,
			close:    make(chan struct{}),
			detached: make(chan struct{}),
			source:   &frames.Source{Address: source},
			target:   new(frames.Target),
		},
		receiverReady: make(chan struct{}, 1),
		batching:      defaultLinkBatching,
		batchMaxAge:   defaultLinkBatchMaxAge,
		maxCredit:     defaultLinkCredit,
	}

	if opts == nil {
		return r, nil
	}

	r.batching = opts.Batching
	if opts.BatchMaxAge > 0 {
		r.batchMaxAge = opts.BatchMaxAge
	}
	for _, v := range opts.Capabilities {
		r.l.target.Capabilities = append(r.l.target.Capabilities, encoding.Symbol(v))
	}
	if opts.Credit > 0 {
		r.maxCredit = opts.Credit
	}
	if opts.Durability > DurabilityUnsettledState {
		return nil, fmt.Errorf("invalid Durability %d", opts.Durability)
	}
	r.l.target.Durable = opts.Durability
	if opts.DynamicAddress {
		r.l.source.Address = ""
		r.l.dynamicAddr = opts.DynamicAddress
	}
	if opts.ExpiryPolicy != "" {
		if err := encoding.ValidateExpiryPolicy(opts.ExpiryPolicy); err != nil {
			return nil, err
		}
		r.l.target.ExpiryPolicy = opts.ExpiryPolicy
	}
	r.l.target.Timeout = opts.ExpiryTimeout
	if opts.Filters != nil {
		r.l.source.Filter = make(encoding.Filter)
		for _, f := range opts.Filters {
			f(r.l.source.Filter)
		}
	}
	if opts.ManualCredits {
		r.manualCreditor = &manualCreditor{}
	}
	if opts.MaxMessageSize > 0 {
		r.l.maxMessageSize = opts.MaxMessageSize
	}
	if opts.Name != "" {
		r.l.key.name = opts.Name
	}
	if opts.Properties != nil {
		r.l.properties = make(map[encoding.Symbol]any)
		for k, v := range opts.Properties {
			if k == "" {
				return nil, errors.New("link property key must not be empty")
			}
			r.l.properties[encoding.Symbol(k)] = v
		}
	}
	if opts.RequestedSenderSettleMode != nil {
		if rsm := *opts.RequestedSenderSettleMode; rsm > SenderSettleModeMixed {
			return nil, fmt.Errorf("invalid RequestedSenderSettleMode %d", rsm)
		}
		r.l.senderSettleMode = opts.RequestedSenderSettleMode
	}
	if opts.SettlementMode != nil {
		if rsm := *opts.SettlementMode; rsm > ReceiverSettleModeSecond {
			return nil, fmt.Errorf("invalid SettlementMode %d", rsm)
		}
		r.l.receiverSettleMode = opts.SettlementMode
	}
	r.l.target.Address = opts.TargetAddress
	for _, v := range opts.SenderCapabilities {
		r.l.source.Capabilities = append(r.l.source.Capabilities, encoding.Symbol(v))
	}
	if opts.SenderDurability != DurabilityNone {
		r.l.source.Durable = opts.SenderDurability
	}
	if opts.SenderExpiryPolicy != ExpiryPolicySessionEnd {
		r.l.source.ExpiryPolicy = opts.SenderExpiryPolicy
	}
	if opts.SenderExpiryTimeout != 0 {
		r.l.source.Timeout = opts.SenderExpiryTimeout
	}
	return r, nil
}

// attach sends the Attach performative to establish the link with its parent session.
// this is automatically called by the new*Link constructors.
func (r *Receiver) attach(ctx context.Context) error {
	// buffer rx to linkCredit so that conn.mux won't block
	// attempting to send to a slow reader
	if r.manualCreditor != nil {
		r.l.rx = make(chan frames.FrameBody, r.maxCredit)
	} else {
		r.l.rx = make(chan frames.FrameBody, r.l.linkCredit)
	}

	if err := r.l.attach(ctx, func(pa *frames.PerformAttach) {
		pa.Role = encoding.RoleReceiver
		if pa.Source == nil {
			pa.Source = new(frames.Source)
		}
		pa.Source.Dynamic = r.l.dynamicAddr
	}, func(pa *frames.PerformAttach) {
		if r.l.source == nil {
			r.l.source = new(frames.Source)
		}
		// if dynamic address requested, copy assigned name to address
		if r.l.dynamicAddr && pa.Source != nil {
			r.l.source.Address = pa.Source.Address
		}
		// deliveryCount is a sequence number, must initialize to sender's initial sequence number
		r.l.deliveryCount = pa.InitialDeliveryCount
		// buffer receiver so that link.mux doesn't block
		r.messages = make(chan Message, r.maxCredit)
		r.unsettledMessages = map[string]struct{}{}
		// copy the received filter values
		if pa.Source != nil {
			r.l.source.Filter = pa.Source.Filter
		}
	}); err != nil {
		return err
	}

	go r.mux()

	return nil
}

func (r *Receiver) mux() {
	defer r.l.muxDetach(context.Background(), func() {
		// unblock any in flight message dispositions
		r.inFlight.clear(r.l.err)

		// unblock any pending drain requests
		if r.manualCreditor != nil {
			r.manualCreditor.EndDrain()
		}
	}, func(fr frames.PerformTransfer) {
		_ = r.muxReceive(fr)
	})

	for {
		switch {
		case r.manualCreditor != nil:
			drain, credits := r.manualCreditor.FlowBits(r.l.linkCredit)

			if drain || credits > 0 {
				debug.Log(1, "receiver (manual): source: %s, inflight: %d, credit: %d, creditsToAdd: %d, drain: %v, deliveryCount: %d, messages: %d, unsettled: %d, maxCredit : %d, settleMode: %s",
					r.l.source.Address, r.inFlight.len(), r.l.linkCredit, credits, drain, r.l.deliveryCount, len(r.messages), r.countUnsettled(), r.maxCredit, r.l.receiverSettleMode.String())

				// send a flow frame.
				r.l.err = r.muxFlow(credits, drain)
			}

		// if receiver && half maxCredits have been processed, send more credits
		case r.l.linkCredit+uint32(r.countUnsettled()) <= r.maxCredit/2:
			debug.Log(1, "receiver (half): source: %s, inflight: %d, credit: %d, deliveryCount: %d, messages: %d, unsettled: %d, maxCredit : %d, settleMode: %s", r.l.source.Address, r.inFlight.len(), r.l.linkCredit, r.l.deliveryCount, len(r.messages), r.countUnsettled(), r.maxCredit, r.l.receiverSettleMode.String())

			linkCredit := r.maxCredit - uint32(r.countUnsettled())
			r.l.err = r.muxFlow(linkCredit, false)

		case r.l.linkCredit == 0:
			debug.Log(1, "receiver (pause): inflight: %d, credit: %d, deliveryCount: %d, messages: %d, unsettled: %d, maxCredit : %d, settleMode: %s", r.inFlight.len(), r.l.linkCredit, r.l.deliveryCount, len(r.messages), r.countUnsettled(), r.maxCredit, r.l.receiverSettleMode.String())
		}

		if r.l.err != nil {
			return
		}

		select {
		// received frame
		case fr := <-r.l.rx:
			r.l.err = r.muxHandleFrame(fr)
			if r.l.err != nil {
				return
			}

		case <-r.receiverReady:
			continue
		case <-r.l.close:
			r.l.err = &DetachError{}
			return
		case <-r.l.session.done:
			r.l.err = r.l.session.err
			return
		}
	}
}

// muxFlow sends tr to the session mux.
// l.linkCredit will also be updated to `linkCredit`
func (r *Receiver) muxFlow(linkCredit uint32, drain bool) error {
	var (
		deliveryCount = r.l.deliveryCount
	)

	debug.Log(3, "muxFlow: len(l.Messages):%d - linkCredit: %d - deliveryCount: %d, inFlight: %d", len(r.messages), linkCredit, deliveryCount, r.inFlight.len())

	fr := &frames.PerformFlow{
		Handle:        &r.l.handle,
		DeliveryCount: &deliveryCount,
		LinkCredit:    &linkCredit, // max number of messages,
		Drain:         drain,
	}
	debug.Log(3, "TX (muxFlow): %s", fr)

	// Update credit. This must happen before entering loop below
	// because incoming messages handled while waiting to transmit
	// flow increment deliveryCount. This causes the credit to become
	// out of sync with the server.

	if !drain {
		// if we're draining we don't want to touch our internal credit - we're not changing it so any issued credits
		// are still valid until drain completes, at which point they will be naturally zeroed.
		r.l.linkCredit = linkCredit
	}

	// Ensure the session mux is not blocked
	for {
		select {
		case r.l.session.tx <- fr:
			return nil
		case fr := <-r.l.rx:
			err := r.muxHandleFrame(fr)
			if err != nil {
				return err
			}
		case <-r.l.close:
			return &DetachError{}
		case <-r.l.session.done:
			return r.l.session.err
		}
	}
}

// muxHandleFrame processes fr based on type.
func (r *Receiver) muxHandleFrame(fr frames.FrameBody) error {
	switch fr := fr.(type) {
	// message frame
	case *frames.PerformTransfer:
		return r.muxReceive(*fr)

	// flow control frame
	case *frames.PerformFlow:
		debug.Log(3, "RX (receiver): %s", fr)
		if !fr.Echo {
			// if the 'drain' flag has been set in the frame sent to the _receiver_ then
			// we signal whomever is waiting (the service has seen and acknowledged our drain)
			if fr.Drain && r.manualCreditor != nil {
				r.l.linkCredit = 0 // we have no active credits at this point.
				r.manualCreditor.EndDrain()
			}
			return nil
		}

		var (
			// copy because sent by pointer below; prevent race
			linkCredit    = r.l.linkCredit
			deliveryCount = r.l.deliveryCount
		)

		// send flow
		// TODO: missing Available and session info
		resp := &frames.PerformFlow{
			Handle:        &r.l.handle,
			DeliveryCount: &deliveryCount,
			LinkCredit:    &linkCredit, // max number of messages
		}
		debug.Log(1, "TX (receiver): %s", resp)
		_ = r.l.session.txFrame(resp, nil)

	case *frames.PerformDisposition:
		debug.Log(3, "RX (receiver): %s", fr)

		// Unblock receivers waiting for message disposition
		// bubble disposition error up to the receiver
		var dispositionError error
		if state, ok := fr.State.(*encoding.StateRejected); ok {
			// state.Error isn't required to be filled out. For instance if you dead letter a message
			// you will get a rejected response that doesn't contain an error.
			if state.Error != nil {
				dispositionError = state.Error
			}
		}
		// removal from the in-flight map will also remove the message from the unsettled map
		r.inFlight.remove(fr.First, fr.Last, dispositionError)

	default:
		return r.l.muxHandleFrame(fr)
	}

	return nil
}

func (r *Receiver) muxReceive(fr frames.PerformTransfer) error {
	if !r.more {
		// this is the first transfer of a message,
		// record the delivery ID, message format,
		// and delivery Tag
		if fr.DeliveryID != nil {
			r.msg.deliveryID = *fr.DeliveryID
		}
		if fr.MessageFormat != nil {
			r.msg.Format = *fr.MessageFormat
		}
		r.msg.DeliveryTag = fr.DeliveryTag

		// these fields are required on first transfer of a message
		if fr.DeliveryID == nil {
			return r.closeWithError(&Error{
				Condition:   ErrCondNotAllowed,
				Description: "received message without a delivery-id",
			})
		}
		if fr.MessageFormat == nil {
			return r.closeWithError(&Error{
				Condition:   ErrCondNotAllowed,
				Description: "received message without a message-format",
			})
		}
		if fr.DeliveryTag == nil {
			return r.closeWithError(&Error{
				Condition:   ErrCondNotAllowed,
				Description: "received message without a delivery-tag",
			})
		}
	} else {
		// this is a continuation of a multipart message
		// some fields may be omitted on continuation transfers,
		// but if they are included they must be consistent
		// with the first.

		if fr.DeliveryID != nil && *fr.DeliveryID != r.msg.deliveryID {
			msg := fmt.Sprintf(
				"received continuation transfer with inconsistent delivery-id: %d != %d",
				*fr.DeliveryID, r.msg.deliveryID,
			)
			return r.closeWithError(&Error{
				Condition:   ErrCondNotAllowed,
				Description: msg,
			})
		}
		if fr.MessageFormat != nil && *fr.MessageFormat != r.msg.Format {
			msg := fmt.Sprintf(
				"received continuation transfer with inconsistent message-format: %d != %d",
				*fr.MessageFormat, r.msg.Format,
			)
			return r.closeWithError(&Error{
				Condition:   ErrCondNotAllowed,
				Description: msg,
			})
		}
		if fr.DeliveryTag != nil && !bytes.Equal(fr.DeliveryTag, r.msg.DeliveryTag) {
			msg := fmt.Sprintf(
				"received continuation transfer with inconsistent delivery-tag: %q != %q",
				fr.DeliveryTag, r.msg.DeliveryTag,
			)
			return r.closeWithError(&Error{
				Condition:   ErrCondNotAllowed,
				Description: msg,
			})
		}
	}

	// discard message if it's been aborted
	if fr.Aborted {
		r.msgBuf.Reset()
		r.msg = Message{}
		r.more = false
		return nil
	}

	// ensure maxMessageSize will not be exceeded
	if r.l.maxMessageSize != 0 && uint64(r.msgBuf.Len())+uint64(len(fr.Payload)) > r.l.maxMessageSize {
		return r.closeWithError(&Error{
			Condition:   ErrCondMessageSizeExceeded,
			Description: fmt.Sprintf("received message larger than max size of %d", r.l.maxMessageSize),
		})
	}

	// add the payload the the buffer
	r.msgBuf.Append(fr.Payload)

	// mark as settled if at least one frame is settled
	r.msg.settled = r.msg.settled || fr.Settled

	// save in-progress status
	r.more = fr.More

	if fr.More {
		return nil
	}

	// last frame in message
	err := r.msg.Unmarshal(&r.msgBuf)
	if err != nil {
		return &DetachError{inner: err}
	}
	debug.Log(1, "deliveryID %d before push to receiver - deliveryCount : %d - linkCredit: %d, len(messages): %d, len(inflight): %d", r.msg.deliveryID, r.l.deliveryCount, r.l.linkCredit, len(r.messages), r.inFlight.len())
	// send to receiver
	if receiverSettleModeValue(r.l.receiverSettleMode) == ReceiverSettleModeSecond {
		r.addUnsettled(&r.msg)
	}
	select {
	case r.messages <- r.msg:
		// message received
	case <-r.l.detached:
		// link has been detached
		return r.l.err
	}

	debug.Log(1, "deliveryID %d after push to receiver - deliveryCount : %d - linkCredit: %d, len(messages): %d, len(inflight): %d", r.msg.deliveryID, r.l.deliveryCount, r.l.linkCredit, len(r.messages), r.inFlight.len())

	// reset progress
	r.msgBuf.Reset()
	r.msg = Message{}

	// decrement link-credit after entire message received
	r.l.deliveryCount++
	r.l.linkCredit--
	debug.Log(1, "deliveryID %d before exit - deliveryCount : %d - linkCredit: %d, len(messages): %d", r.msg.deliveryID, r.l.deliveryCount, r.l.linkCredit, len(r.messages))
	return nil
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
