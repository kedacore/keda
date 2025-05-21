package amqp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/Azure/go-amqp/internal/buffer"
	"github.com/Azure/go-amqp/internal/debug"
	"github.com/Azure/go-amqp/internal/encoding"
	"github.com/Azure/go-amqp/internal/frames"
	"github.com/Azure/go-amqp/internal/queue"
)

// Default link options
const (
	defaultLinkCredit = 1
)

// Receiver receives messages on a single AMQP link.
type Receiver struct {
	l link

	// message receiving
	receiverReady chan struct{}          // receiver sends on this when mux is paused to indicate it can handle more messages
	messagesQ     *queue.Holder[Message] // used to send completed messages to receiver
	txDisposition chan frameBodyEnvelope // used to funnel disposition frames through the mux

	// NOTE: this will need to be retooled if/when we need to support resuming links.
	// at present, this is only used for debug tracing purposes so it's safe to change it to a count.
	unsettledMessages int32 // count of unsettled messages for this receiver; MUST be atomically accessed

	msgBuf buffer.Buffer // buffered bytes for current message
	more   bool          // if true, buf contains a partial message
	msg    Message       // current message being decoded

	settlementCount   uint32     // the count of settled messages
	settlementCountMu sync.Mutex // must be held when accessing settlementCount

	autoSendFlow bool     // automatically send flow frames as credit becomes available
	inFlight     inFlight // used to track message disposition when rcv-settle-mode == second
	creditor     creditor // manages credits via calls to IssueCredit/DrainCredit
}

// IssueCredit adds credits to be requested in the next flow request.
// Attempting to issue more credit than the receiver's max credit as
// specified in ReceiverOptions.MaxCredit will result in an error.
func (r *Receiver) IssueCredit(credit uint32) error {
	if r.autoSendFlow {
		return errors.New("issueCredit can only be used with receiver links using manual credit management")
	}

	if err := r.creditor.IssueCredit(credit); err != nil {
		return err
	}

	// cause mux() to check our flow conditions.
	select {
	case r.receiverReady <- struct{}{}:
	default:
	}

	return nil
}

// DrainCreditOptions contains any optional values for the Receiver.DrainCredit method.
type DrainCreditOptions struct {
	// for future expansion
}

// DrainCredit sets the drain flag on the next outbound FLOW frame and blocks until
// the corresponding FLOW frame is received. While a drain is in progress, messages
// can continue to arrive. After a drain completes, the Receiver will have
// zero active credits. To begin receiving again, call IssueCredit() to add active credits
// to your Receiver.
//
// You may only have a single Drain operation active, at a time.
//
// If the context passed to DrainCredit expires or is cancelled then the receiver's
// issued credits should be considered ambiguous.
//
// Returns nil if the drain has completed, error otherwise.
//
// NOTE: The behavior of drain is optional, as per the AMQP spec. Check with your individual
// broker's documentation for implementation details.
func (r *Receiver) DrainCredit(ctx context.Context, _ *DrainCreditOptions) error {
	if r.autoSendFlow {
		return errors.New("drain can only be used with receiver links using manual credit management")
	}

	return r.creditor.Drain(ctx, r)
}

// Prefetched returns the next message that is stored in the Receiver's
// prefetch cache. It does NOT wait for the remote sender to send messages
// and returns immediately if the prefetch cache is empty. To receive from the
// prefetch and wait for messages from the remote Sender use `Receive`.
//
// Once a message is received, and if the sender is configured in any mode other
// than SenderSettleModeSettled, you *must* take an action on the message by calling
// one of the following: AcceptMessage, RejectMessage, ReleaseMessage, ModifyMessage.
func (r *Receiver) Prefetched() *Message {
	select {
	case r.receiverReady <- struct{}{}:
	default:
	}

	// non-blocking receive to ensure buffered messages are
	// delivered regardless of whether the link has been closed.
	q := r.messagesQ.Acquire()
	msg := q.Dequeue()
	r.messagesQ.Release(q)

	if msg == nil {
		return nil
	}

	debug.Log(3, "RX (Receiver %p): prefetched delivery ID %d", r, msg.deliveryID)

	if msg.settled {
		r.onSettlement(1)
	}

	return msg
}

// ReceiveOptions contains any optional values for the Receiver.Receive method.
type ReceiveOptions struct {
	// for future expansion
}

// Receive returns the next message from the sender.
// Blocks until a message is received, ctx completes, or an error occurs.
//
// Once a message is received, and if the sender is configured in any mode other
// than SenderSettleModeSettled, you *must* take an action on the message by calling
// one of the following: AcceptMessage, RejectMessage, ReleaseMessage, ModifyMessage.
func (r *Receiver) Receive(ctx context.Context, opts *ReceiveOptions) (*Message, error) {
	if msg := r.Prefetched(); msg != nil {
		return msg, nil
	}

	// wait for the next message
	select {
	case q := <-r.messagesQ.Wait():
		msg := q.Dequeue()
		debug.Assert(msg != nil)
		debug.Log(3, "RX (Receiver %p): received delivery ID %d", r, msg.deliveryID)
		r.messagesQ.Release(q)
		if msg.settled {
			r.onSettlement(1)
		}
		return msg, nil
	case <-r.l.done:
		// if the link receives messages and is then closed between the above call to r.Prefetched()
		// and this select statement, the order of selecting r.messages and r.l.done is undefined.
		// however, once r.l.done is closed the link cannot receive any more messages. so be sure to
		// drain any that might have trickled in within this window.
		if msg := r.Prefetched(); msg != nil {
			return msg, nil
		}
		return nil, r.l.doneErr
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Accept notifies the server that the message has been accepted and does not require redelivery.
//   - ctx controls waiting for the peer to acknowledge the disposition
//   - msg is the message to accept
//
// If the context's deadline expires or is cancelled before the operation
// completes, the message's disposition is in an unknown state.
func (r *Receiver) AcceptMessage(ctx context.Context, msg *Message) error {
	return msg.rcv.messageDisposition(ctx, msg, &encoding.StateAccepted{})
}

// Reject notifies the server that the message is invalid.
//   - ctx controls waiting for the peer to acknowledge the disposition
//   - msg is the message to reject
//   - e is an optional rejection error
//
// If the context's deadline expires or is cancelled before the operation
// completes, the message's disposition is in an unknown state.
func (r *Receiver) RejectMessage(ctx context.Context, msg *Message, e *Error) error {
	return msg.rcv.messageDisposition(ctx, msg, &encoding.StateRejected{Error: e})
}

// Release releases the message back to the server. The message may be redelivered to this or another consumer.
//   - ctx controls waiting for the peer to acknowledge the disposition
//   - msg is the message to release
//
// If the context's deadline expires or is cancelled before the operation
// completes, the message's disposition is in an unknown state.
func (r *Receiver) ReleaseMessage(ctx context.Context, msg *Message) error {
	return msg.rcv.messageDisposition(ctx, msg, &encoding.StateReleased{})
}

// Modify notifies the server that the message was not acted upon and should be modifed.
//   - ctx controls waiting for the peer to acknowledge the disposition
//   - msg is the message to modify
//   - options contains the optional settings to modify
//
// If the context's deadline expires or is cancelled before the operation
// completes, the message's disposition is in an unknown state.
func (r *Receiver) ModifyMessage(ctx context.Context, msg *Message, options *ModifyMessageOptions) error {
	if options == nil {
		options = &ModifyMessageOptions{}
	}
	return msg.rcv.messageDisposition(ctx,
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

// Properties returns the peer's link properties.
// Returns nil if the peer didn't send any properties.
func (r *Receiver) Properties() map[string]any {
	return r.l.peerProperties
}

// Close closes the Receiver and AMQP link.
//   - ctx controls waiting for the peer to acknowledge the close
//
// If the context's deadline expires or is cancelled before the operation
// completes, an error is returned.  However, the operation will continue to
// execute in the background. Subsequent calls will return a *LinkError
// that contains the context's error message.
func (r *Receiver) Close(ctx context.Context) error {
	return r.l.closeLink(ctx)
}

// sendDisposition sends a disposition frame to the peer
func (r *Receiver) sendDisposition(ctx context.Context, first uint32, last *uint32, state encoding.DeliveryState) error {
	fr := &frames.PerformDisposition{
		Role:    encoding.RoleReceiver,
		First:   first,
		Last:    last,
		Settled: r.l.receiverSettleMode == nil || *r.l.receiverSettleMode == ReceiverSettleModeFirst,
		State:   state,
	}

	frameCtx := frameContext{
		Ctx:  ctx,
		Done: make(chan struct{}),
	}

	select {
	case r.txDisposition <- frameBodyEnvelope{FrameCtx: &frameCtx, FrameBody: fr}:
		debug.Log(2, "TX (Receiver %p): mux txDisposition %s", r, fr)
	case <-r.l.done:
		return r.l.doneErr
	}

	select {
	case <-frameCtx.Done:
		return frameCtx.Err
	case <-r.l.done:
		return r.l.doneErr
	}
}

// messageDisposition is called via the *Receiver associated with a *Message.
// this allows messages to be settled across Receiver instances.
// note that only unsettled messsages will have their rcv field set.
func (r *Receiver) messageDisposition(ctx context.Context, msg *Message, state encoding.DeliveryState) error {
	// settling a message that's already settled (sender-settled or otherwise) will have a nil rcv.
	// which means that r will be nil. you MUST NOT dereference r if msg.settled == true
	if msg.settled {
		return nil
	}

	debug.Assert(r != nil)

	// NOTE: we MUST add to the in-flight map before sending the disposition. if not, it's possible
	// to receive the ack'ing disposition frame *before* the in-flight map has been updated which
	// will cause the below <-wait to never trigger.

	var wait chan error
	if r.l.receiverSettleMode != nil && *r.l.receiverSettleMode == ReceiverSettleModeSecond {
		debug.Log(3, "TX (Receiver %p): delivery ID %d is in flight", r, msg.deliveryID)
		wait = r.inFlight.add(msg)
	}

	if err := r.sendDisposition(ctx, msg.deliveryID, nil, state); err != nil {
		return err
	}

	if wait == nil {
		// mode first, there will be no settlement ack
		msg.onSettlement()
		r.deleteUnsettled()
		r.onSettlement(1)
		return nil
	}

	select {
	case err := <-wait:
		// err has three possibilities
		//   - nil, meaning the peer acknowledged the settlement
		//   - an *Error, meaning the peer rejected the message with a provided error
		//   - a non-AMQP error. this comes from calls to inFlight.clear() during mux unwind.
		// only for the first two cases is the message considered settled

		if amqpErr := (&Error{}); err == nil || errors.As(err, &amqpErr) {
			debug.Log(3, "RX (Receiver %p): delivery ID %d has been settled", r, msg.deliveryID)
			// we've received confirmation of disposition
			return err
		}

		debug.Log(3, "RX (Receiver %p): error settling delivery ID %d: %v", r, msg.deliveryID, err)
		return err

	case <-ctx.Done():
		// didn't receive the ack in the time allotted, leave message as unsettled
		// TODO: if the ack arrives later, we need to remove the message from the unsettled map and reclaim the credit
		return ctx.Err()
	}
}

// onSettlement is to be called after message settlement.
//   - count is the number of messages that were settled
func (r *Receiver) onSettlement(count uint32) {
	if !r.autoSendFlow {
		return
	}

	r.settlementCountMu.Lock()
	r.settlementCount += count
	r.settlementCountMu.Unlock()

	select {
	case r.receiverReady <- struct{}{}:
		// woke up
	default:
		// wake pending
	}
}

// increments the count of unsettled messages.
// this is only called from our mux.
func (r *Receiver) addUnsettled() {
	atomic.AddInt32(&r.unsettledMessages, 1)
}

// decrements the count of unsettled messages.
// this is called inside _or_ outside the mux.
// it's called outside when RSM is mode first.
func (r *Receiver) deleteUnsettled() {
	atomic.AddInt32(&r.unsettledMessages, -1)
}

// returns the count of unsettled messages.
// this is only called from our mux for diagnostic purposes.
func (r *Receiver) countUnsettled() int32 {
	return atomic.LoadInt32(&r.unsettledMessages)
}

func newReceiver(source string, session *Session, opts *ReceiverOptions) (*Receiver, error) {
	l := newLink(session, encoding.RoleReceiver)
	l.source = &frames.Source{Address: source}
	l.target = new(frames.Target)
	l.linkCredit = defaultLinkCredit
	r := &Receiver{
		l:             l,
		autoSendFlow:  true,
		receiverReady: make(chan struct{}, 1),
		txDisposition: make(chan frameBodyEnvelope),
	}

	r.messagesQ = queue.NewHolder(queue.New[Message](int(session.incomingWindow)))

	if opts == nil {
		return r, nil
	}

	for _, v := range opts.Capabilities {
		r.l.target.Capabilities = append(r.l.target.Capabilities, encoding.Symbol(v))
	}
	if opts.Credit > 0 {
		r.l.linkCredit = uint32(opts.Credit)
	} else if opts.Credit < 0 {
		r.l.linkCredit = 0
		r.autoSendFlow = false
	}

	if opts.DesiredCapabilities != nil {
		r.l.desiredCapabilities = make([]encoding.Symbol, 0, len(opts.DesiredCapabilities))

		for _, capabilityStr := range opts.DesiredCapabilities {
			r.l.desiredCapabilities = append(r.l.desiredCapabilities, encoding.Symbol(capabilityStr))
		}
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
	for _, v := range opts.SourceCapabilities {
		r.l.source.Capabilities = append(r.l.source.Capabilities, encoding.Symbol(v))
	}
	if opts.SourceDurability != DurabilityNone {
		r.l.source.Durable = opts.SourceDurability
	}
	if opts.SourceExpiryPolicy != ExpiryPolicySessionEnd {
		r.l.source.ExpiryPolicy = opts.SourceExpiryPolicy
	}
	if opts.SourceExpiryTimeout != 0 {
		r.l.source.Timeout = opts.SourceExpiryTimeout
	}
	return r, nil
}

// attach sends the Attach performative to establish the link with its parent session.
// this is automatically called by the new*Link constructors.
func (r *Receiver) attach(ctx context.Context) error {
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
		// copy the received filter values
		if pa.Source != nil {
			r.l.source.Filter = pa.Source.Filter
		}
	}); err != nil {
		return err
	}

	return nil
}

func nopHook() {}

type receiverTestHooks struct {
	MuxStart  func()
	MuxSelect func()
}

func (r *Receiver) mux(hooks receiverTestHooks) {
	if hooks.MuxSelect == nil {
		hooks.MuxSelect = nopHook
	}
	if hooks.MuxStart == nil {
		hooks.MuxStart = nopHook
	}

	defer func() {
		// unblock any in flight message dispositions
		r.inFlight.clear(r.l.doneErr)

		if !r.autoSendFlow {
			// unblock any pending drain requests
			r.creditor.EndDrain()
		}

		close(r.l.done)
	}()

	hooks.MuxStart()

	if r.autoSendFlow {
		r.l.doneErr = r.muxFlow(r.l.linkCredit, false)
	}

	for {
		msgLen := r.messagesQ.Len()

		r.settlementCountMu.Lock()
		// counter that accumulates the settled delivery count.
		// once the threshold has been reached, the counter is
		// reset and a flow frame is sent.
		previousSettlementCount := r.settlementCount
		if previousSettlementCount >= r.l.linkCredit {
			r.settlementCount = 0
		}
		r.settlementCountMu.Unlock()

		// once we have pending credit equal to or greater than our available credit, reclaim it.
		// we do this instead of settlementCount > 0 to prevent flow frames from being too chatty.
		// NOTE: we compare the settlementCount against the current link credit instead of some
		// fixed threshold to ensure credit is reclaimed in cases where the number of unsettled
		// messages remains high for whatever reason.
		if r.autoSendFlow && previousSettlementCount > 0 && previousSettlementCount >= r.l.linkCredit {
			debug.Log(1, "RX (Receiver %p) (auto): source: %q, inflight: %d, linkCredit: %d, deliveryCount: %d, messages: %d, unsettled: %d, settlementCount: %d, settleMode: %s",
				r, r.l.source.Address, r.inFlight.len(), r.l.linkCredit, r.l.deliveryCount, msgLen, r.countUnsettled(), previousSettlementCount, r.l.receiverSettleMode.String())
			r.l.doneErr = r.creditor.IssueCredit(previousSettlementCount)
		} else if r.l.linkCredit == 0 {
			debug.Log(1, "RX (Receiver %p) (pause): source: %q, inflight: %d, linkCredit: %d, deliveryCount: %d, messages: %d, unsettled: %d, settlementCount: %d, settleMode: %s",
				r, r.l.source.Address, r.inFlight.len(), r.l.linkCredit, r.l.deliveryCount, msgLen, r.countUnsettled(), previousSettlementCount, r.l.receiverSettleMode.String())
		}

		if r.l.doneErr != nil {
			return
		}

		drain, credits := r.creditor.FlowBits(r.l.linkCredit)
		if drain || credits > 0 {
			debug.Log(1, "RX (Receiver %p) (flow): source: %q, inflight: %d, curLinkCredit: %d, newLinkCredit: %d, drain: %v, deliveryCount: %d, messages: %d, unsettled: %d, settlementCount: %d, settleMode: %s",
				r, r.l.source.Address, r.inFlight.len(), r.l.linkCredit, credits, drain, r.l.deliveryCount, msgLen, r.countUnsettled(), previousSettlementCount, r.l.receiverSettleMode.String())

			// send a flow frame.
			r.l.doneErr = r.muxFlow(credits, drain)
		}

		if r.l.doneErr != nil {
			return
		}

		txDisposition := r.txDisposition
		closed := r.l.close
		if r.l.closeInProgress {
			// swap out channel so it no longer triggers
			closed = nil

			// disable sending of disposition frames once closing is in progress.
			// this is to prevent races between mux shutdown and clearing of
			// any in-flight dispositions.
			txDisposition = nil
		}

		hooks.MuxSelect()

		select {
		case q := <-r.l.rxQ.Wait():
			// populated queue
			fr := *q.Dequeue()
			r.l.rxQ.Release(q)

			// if muxHandleFrame returns an error it means the mux must terminate.
			// note that in the case of a client-side close due to an error, nil
			// is returned in order to keep the mux running to ack the detach frame.
			if err := r.muxHandleFrame(fr); err != nil {
				r.l.doneErr = err
				return
			}

		case env := <-txDisposition:
			r.l.txFrame(env.FrameCtx, env.FrameBody)

		case <-r.receiverReady:
			continue

		case <-closed:
			if r.l.closeInProgress {
				// a client-side close due to protocol error is in progress
				continue
			}

			// receiver is being closed by the client
			r.l.closeInProgress = true
			fr := &frames.PerformDetach{
				Handle: r.l.outputHandle,
				Closed: true,
			}
			r.l.txFrame(&frameContext{Ctx: context.Background()}, fr)

		case <-r.l.session.done:
			r.l.doneErr = r.l.session.doneErr
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

	fr := &frames.PerformFlow{
		Handle:        &r.l.outputHandle,
		DeliveryCount: &deliveryCount,
		LinkCredit:    &linkCredit, // max number of messages,
		Drain:         drain,
	}

	// Update credit. This must happen before entering loop below
	// because incoming messages handled while waiting to transmit
	// flow increment deliveryCount. This causes the credit to become
	// out of sync with the server.

	if !drain {
		// if we're draining we don't want to touch our internal credit - we're not changing it so any issued credits
		// are still valid until drain completes, at which point they will be naturally zeroed.
		r.l.linkCredit = linkCredit
	}

	select {
	case r.l.session.tx <- frameBodyEnvelope{FrameCtx: &frameContext{Ctx: context.Background()}, FrameBody: fr}:
		debug.Log(2, "TX (Receiver %p): mux frame to Session (%p): %d, %s", r, r.l.session, r.l.session.channel, fr)
		return nil
	case <-r.l.close:
		return nil
	case <-r.l.session.done:
		return r.l.session.doneErr
	}
}

// muxHandleFrame processes fr based on type.
func (r *Receiver) muxHandleFrame(fr frames.FrameBody) error {
	debug.Log(2, "RX (Receiver %p): %s", r, fr)
	switch fr := fr.(type) {
	// message frame
	case *frames.PerformTransfer:
		r.muxReceive(*fr)

	// flow control frame
	case *frames.PerformFlow:
		if !fr.Echo {
			// if the 'drain' flag has been set in the frame sent to the _receiver_ then
			// we signal whomever is waiting (the service has seen and acknowledged our drain)
			if fr.Drain && !r.autoSendFlow {
				r.l.linkCredit = 0 // we have no active credits at this point.
				r.creditor.EndDrain()
			}
			return nil
		}

		var (
			// copy because sent by pointer below; prevent race
			linkCredit    = r.l.linkCredit
			deliveryCount = r.l.deliveryCount
		)

		// send flow
		resp := &frames.PerformFlow{
			Handle:        &r.l.outputHandle,
			DeliveryCount: &deliveryCount,
			LinkCredit:    &linkCredit, // max number of messages
		}

		select {
		case r.l.session.tx <- frameBodyEnvelope{FrameCtx: &frameContext{Ctx: context.Background()}, FrameBody: resp}:
			debug.Log(2, "TX (Receiver %p): mux frame to Session (%p): %d, %s", r, r.l.session, r.l.session.channel, resp)
		case <-r.l.close:
			return nil
		case <-r.l.session.done:
			return r.l.session.doneErr
		}

	case *frames.PerformDisposition:
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
		count := r.inFlight.remove(fr.First, fr.Last, dispositionError, func(msg *Message) {
			r.deleteUnsettled()
			msg.onSettlement()
		})
		r.onSettlement(count)

	default:
		return r.l.muxHandleFrame(fr)
	}

	return nil
}

func (r *Receiver) muxReceive(fr frames.PerformTransfer) {
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
			r.l.closeWithError(ErrCondNotAllowed, "received message without a delivery-id")
			return
		}
		if fr.MessageFormat == nil {
			r.l.closeWithError(ErrCondNotAllowed, "received message without a message-format")
			return
		}
		if fr.DeliveryTag == nil {
			r.l.closeWithError(ErrCondNotAllowed, "received message without a delivery-tag")
			return
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
			r.l.closeWithError(ErrCondNotAllowed, msg)
			return
		}
		if fr.MessageFormat != nil && *fr.MessageFormat != r.msg.Format {
			msg := fmt.Sprintf(
				"received continuation transfer with inconsistent message-format: %d != %d",
				*fr.MessageFormat, r.msg.Format,
			)
			r.l.closeWithError(ErrCondNotAllowed, msg)
			return
		}
		if fr.DeliveryTag != nil && !bytes.Equal(fr.DeliveryTag, r.msg.DeliveryTag) {
			msg := fmt.Sprintf(
				"received continuation transfer with inconsistent delivery-tag: %q != %q",
				fr.DeliveryTag, r.msg.DeliveryTag,
			)
			r.l.closeWithError(ErrCondNotAllowed, msg)
			return
		}
	}

	// discard message if it's been aborted
	if fr.Aborted {
		r.msgBuf.Reset()
		r.msg = Message{}
		r.more = false
		return
	}

	// ensure maxMessageSize will not be exceeded
	if r.l.maxMessageSize != 0 && uint64(r.msgBuf.Len())+uint64(len(fr.Payload)) > r.l.maxMessageSize {
		r.l.closeWithError(ErrCondMessageSizeExceeded, fmt.Sprintf("received message larger than max size of %d", r.l.maxMessageSize))
		return
	}

	// add the payload the the buffer
	r.msgBuf.Append(fr.Payload)

	// mark as settled if at least one frame is settled
	r.msg.settled = r.msg.settled || fr.Settled

	// save in-progress status
	r.more = fr.More

	if fr.More {
		return
	}

	// last frame in message
	err := r.msg.Unmarshal(&r.msgBuf)
	if err != nil {
		r.l.closeWithError(ErrCondInternalError, err.Error())
		return
	}

	// send to receiver
	if !r.msg.settled {
		r.addUnsettled()
		r.msg.rcv = r
		debug.Log(3, "RX (Receiver %p): add unsettled delivery ID %d", r, r.msg.deliveryID)
	}

	q := r.messagesQ.Acquire()
	q.Enqueue(r.msg)
	msgLen := q.Len()
	r.messagesQ.Release(q)

	// reset progress
	r.msgBuf.Reset()
	r.msg = Message{}

	// decrement link-credit after entire message received
	r.l.deliveryCount++
	r.l.linkCredit--
	debug.Log(3, "RX (Receiver %p) link %s - deliveryCount: %d, linkCredit: %d, len(messages): %d", r, r.l.key.name, r.l.deliveryCount, r.l.linkCredit, msgLen)
}

// inFlight tracks in-flight message dispositions allowing receivers
// to block waiting for the server to respond when an appropriate
// settlement mode is configured.
type inFlight struct {
	mu sync.RWMutex
	m  map[uint32]inFlightInfo
}

type inFlightInfo struct {
	wait chan error
	msg  *Message
}

func (f *inFlight) add(msg *Message) chan error {
	wait := make(chan error, 1)

	f.mu.Lock()
	if f.m == nil {
		f.m = make(map[uint32]inFlightInfo)
	}

	f.m[msg.deliveryID] = inFlightInfo{wait: wait, msg: msg}
	f.mu.Unlock()

	return wait
}

func (f *inFlight) remove(first uint32, last *uint32, err error, handler func(*Message)) uint32 {
	f.mu.Lock()

	if f.m == nil {
		f.mu.Unlock()
		return 0
	}

	ll := first
	if last != nil {
		ll = *last
	}

	count := uint32(0)
	for i := first; i <= ll; i++ {
		info, ok := f.m[i]
		if ok {
			handler(info.msg)
			info.wait <- err
			delete(f.m, i)
			count++
		}
	}

	f.mu.Unlock()
	return count
}

func (f *inFlight) clear(err error) {
	f.mu.Lock()
	for id, info := range f.m {
		info.wait <- err
		delete(f.m, id)
	}
	f.mu.Unlock()
}

func (f *inFlight) len() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return len(f.m)
}
