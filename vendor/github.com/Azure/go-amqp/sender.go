package amqp

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"

	"github.com/Azure/go-amqp/internal/buffer"
	"github.com/Azure/go-amqp/internal/debug"
	"github.com/Azure/go-amqp/internal/encoding"
	"github.com/Azure/go-amqp/internal/frames"
)

// Sender sends messages on a single AMQP link.
type Sender struct {
	l         link
	transfers chan transferEnvelope // sender uses to send transfer frames

	mu              sync.Mutex // protects buf and nextDeliveryTag
	buf             buffer.Buffer
	nextDeliveryTag uint64
	rollback        chan struct{}
}

// LinkName() is the name of the link used for this Sender.
func (s *Sender) LinkName() string {
	return s.l.key.name
}

// MaxMessageSize is the maximum size of a single message.
func (s *Sender) MaxMessageSize() uint64 {
	return s.l.maxMessageSize
}

// Properties returns the peer's link properties.
// Returns nil if the peer didn't send any properties.
func (s *Sender) Properties() map[string]any {
	return s.l.peerProperties
}

// SendOptions contains any optional values for the Sender.Send method.
type SendOptions struct {
	// Indicates the message is to be sent as settled when settlement mode is SenderSettleModeMixed.
	// If the settlement mode is SenderSettleModeUnsettled and Settled is true, an error is returned.
	Settled bool
}

// Send sends a Message.
//
// Blocks until the message is sent or an error occurs. If the peer is
// configured for receiver settlement mode second, the call also blocks
// until the peer confirms message settlement.
//
//   - ctx controls waiting for the message to be sent and possibly confirmed
//   - msg is the message to send
//   - opts contains optional values, pass nil to accept the defaults
//
// If the context's deadline expires or is cancelled before the operation
// completes, the message is in an unknown state of transmission.
//
// Send is safe for concurrent use. Since only a single message can be
// sent on a link at a time, this is most useful when settlement confirmation
// has been requested (receiver settle mode is second). In this case,
// additional messages can be sent while the current goroutine is waiting
// for the confirmation.
func (s *Sender) Send(ctx context.Context, msg *Message, opts *SendOptions) error {
	// check if the link is dead.  while it's safe to call s.send
	// in this case, this will avoid some allocations etc.
	select {
	case <-s.l.done:
		return s.l.doneErr
	default:
		// link is still active
	}
	done, err := s.send(ctx, msg, opts)
	if err != nil {
		return err
	}

	// wait for transfer to be confirmed
	select {
	case state := <-done:
		if state, ok := state.(*encoding.StateRejected); ok {
			if state.Error != nil {
				return state.Error
			}
			return errors.New("the peer rejected the message without specifying an error")
		}
		return nil
	case <-s.l.done:
		return s.l.doneErr
	case <-ctx.Done():
		// TODO: if the message is not settled and we never received a disposition, how can we consider the message as sent?
		return ctx.Err()
	}
}

// send is separated from Send so that the mutex unlock can be deferred without
// locking the transfer confirmation that happens in Send.
func (s *Sender) send(ctx context.Context, msg *Message, opts *SendOptions) (chan encoding.DeliveryState, error) {
	const (
		maxDeliveryTagLength   = 32
		maxTransferFrameHeader = 66 // determined by calcMaxTransferFrameHeader
	)
	if len(msg.DeliveryTag) > maxDeliveryTagLength {
		return nil, &Error{
			Condition:   ErrCondMessageSizeExceeded,
			Description: fmt.Sprintf("delivery tag is over the allowed %v bytes, len: %v", maxDeliveryTagLength, len(msg.DeliveryTag)),
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.buf.Reset()
	err := msg.Marshal(&s.buf)
	if err != nil {
		return nil, err
	}

	if s.l.maxMessageSize != 0 && uint64(s.buf.Len()) > s.l.maxMessageSize {
		return nil, &Error{
			Condition:   ErrCondMessageSizeExceeded,
			Description: fmt.Sprintf("encoded message size exceeds max of %d", s.l.maxMessageSize),
		}
	}

	senderSettled := senderSettleModeValue(s.l.senderSettleMode) == SenderSettleModeSettled
	if opts != nil {
		if opts.Settled && senderSettleModeValue(s.l.senderSettleMode) == SenderSettleModeUnsettled {
			return nil, errors.New("can't send message as settled when sender settlement mode is unsettled")
		} else if opts.Settled {
			senderSettled = true
		}
	}

	var (
		maxPayloadSize = int64(s.l.session.conn.peerMaxFrameSize) - maxTransferFrameHeader
	)

	deliveryTag := msg.DeliveryTag
	if len(deliveryTag) == 0 {
		// use uint64 encoded as []byte as deliveryTag
		deliveryTag = make([]byte, 8)
		binary.BigEndian.PutUint64(deliveryTag, s.nextDeliveryTag)
		s.nextDeliveryTag++
	}

	fr := frames.PerformTransfer{
		Handle:        s.l.outputHandle,
		DeliveryID:    &needsDeliveryID,
		DeliveryTag:   deliveryTag,
		MessageFormat: &msg.Format,
		More:          s.buf.Len() > 0,
	}

	for fr.More {
		buf, _ := s.buf.Next(maxPayloadSize)
		fr.Payload = append([]byte(nil), buf...)
		fr.More = s.buf.Len() > 0
		if !fr.More {
			// SSM=settled: overrides RSM; no acks.
			// SSM=unsettled: sender should wait for receiver to ack
			// RSM=first: receiver considers it settled immediately, but must still send ack (SSM=unsettled only)
			// RSM=second: receiver sends ack and waits for return ack from sender (SSM=unsettled only)

			// mark final transfer as settled when sender mode is settled
			fr.Settled = senderSettled

			// set done on last frame
			fr.Done = make(chan encoding.DeliveryState, 1)
		}

		// NOTE: we MUST send a copy of fr here since we modify it post send

		frameCtx := frameContext{
			Ctx:  ctx,
			Done: make(chan struct{}),
		}

		select {
		case s.transfers <- transferEnvelope{FrameCtx: &frameCtx, InputHandle: s.l.inputHandle, Frame: fr}:
			// frame was sent to our mux
		case <-s.l.done:
			return nil, s.l.doneErr
		case <-ctx.Done():
			return nil, &Error{Condition: ErrCondTransferLimitExceeded, Description: fmt.Sprintf("credit limit exceeded for sending link %s", s.l.key.name)}
		}

		select {
		case <-frameCtx.Done:
			if frameCtx.Err != nil {
				if !fr.More {
					select {
					case s.rollback <- struct{}{}:
						// the write never happened so signal the mux to roll back the delivery count and link credit
					case <-s.l.close:
						// the link is going down
					}
				}
				return nil, frameCtx.Err
			}
			// frame was written to the network
		case <-s.l.done:
			return nil, s.l.doneErr
		}

		// clear values that are only required on first message
		fr.DeliveryID = nil
		fr.DeliveryTag = nil
		fr.MessageFormat = nil
	}

	return fr.Done, nil
}

// Address returns the link's address.
func (s *Sender) Address() string {
	if s.l.target == nil {
		return ""
	}
	return s.l.target.Address
}

// Close closes the Sender and AMQP link.
//   - ctx controls waiting for the peer to acknowledge the close
//
// If the context's deadline expires or is cancelled before the operation
// completes, an error is returned.  However, the operation will continue to
// execute in the background. Subsequent calls will return a *LinkError
// that contains the context's error message.
func (s *Sender) Close(ctx context.Context) error {
	return s.l.closeLink(ctx)
}

// newSendingLink creates a new sending link and attaches it to the session
func newSender(target string, session *Session, opts *SenderOptions) (*Sender, error) {
	l := newLink(session, encoding.RoleSender)
	l.target = &frames.Target{Address: target}
	l.source = new(frames.Source)
	s := &Sender{
		l:        l,
		rollback: make(chan struct{}),
	}

	if opts == nil {
		return s, nil
	}

	for _, v := range opts.Capabilities {
		s.l.source.Capabilities = append(s.l.source.Capabilities, encoding.Symbol(v))
	}
	if opts.Durability > DurabilityUnsettledState {
		return nil, fmt.Errorf("invalid Durability %d", opts.Durability)
	}
	s.l.source.Durable = opts.Durability
	if opts.DynamicAddress {
		s.l.target.Address = ""
		s.l.dynamicAddr = opts.DynamicAddress
	}
	if opts.ExpiryPolicy != "" {
		if err := encoding.ValidateExpiryPolicy(opts.ExpiryPolicy); err != nil {
			return nil, err
		}
		s.l.source.ExpiryPolicy = opts.ExpiryPolicy
	}
	s.l.source.Timeout = opts.ExpiryTimeout
	if opts.Name != "" {
		s.l.key.name = opts.Name
	}
	if opts.Properties != nil {
		s.l.properties = make(map[encoding.Symbol]any)
		for k, v := range opts.Properties {
			if k == "" {
				return nil, errors.New("link property key must not be empty")
			}
			s.l.properties[encoding.Symbol(k)] = v
		}
	}
	if opts.RequestedReceiverSettleMode != nil {
		if rsm := *opts.RequestedReceiverSettleMode; rsm > ReceiverSettleModeSecond {
			return nil, fmt.Errorf("invalid RequestedReceiverSettleMode %d", rsm)
		}
		s.l.receiverSettleMode = opts.RequestedReceiverSettleMode
	}
	if opts.SettlementMode != nil {
		if ssm := *opts.SettlementMode; ssm > SenderSettleModeMixed {
			return nil, fmt.Errorf("invalid SettlementMode %d", ssm)
		}
		s.l.senderSettleMode = opts.SettlementMode
	}
	s.l.source.Address = opts.SourceAddress
	for _, v := range opts.TargetCapabilities {
		s.l.target.Capabilities = append(s.l.target.Capabilities, encoding.Symbol(v))
	}
	if opts.TargetDurability != DurabilityNone {
		s.l.target.Durable = opts.TargetDurability
	}
	if opts.TargetExpiryPolicy != ExpiryPolicySessionEnd {
		s.l.target.ExpiryPolicy = opts.TargetExpiryPolicy
	}
	if opts.TargetExpiryTimeout != 0 {
		s.l.target.Timeout = opts.TargetExpiryTimeout
	}
	return s, nil
}

func (s *Sender) attach(ctx context.Context) error {
	if err := s.l.attach(ctx, func(pa *frames.PerformAttach) {
		pa.Role = encoding.RoleSender
		if pa.Target == nil {
			pa.Target = new(frames.Target)
		}
		pa.Target.Dynamic = s.l.dynamicAddr
	}, func(pa *frames.PerformAttach) {
		if s.l.target == nil {
			s.l.target = new(frames.Target)
		}

		// if dynamic address requested, copy assigned name to address
		if s.l.dynamicAddr && pa.Target != nil {
			s.l.target.Address = pa.Target.Address
		}
	}); err != nil {
		return err
	}

	s.transfers = make(chan transferEnvelope)

	return nil
}

type senderTestHooks struct {
	MuxSelect   func()
	MuxTransfer func()
}

func (s *Sender) mux(hooks senderTestHooks) {
	if hooks.MuxSelect == nil {
		hooks.MuxSelect = nopHook
	}
	if hooks.MuxTransfer == nil {
		hooks.MuxTransfer = nopHook
	}

	defer func() {
		close(s.l.done)
	}()

Loop:
	for {
		var outgoingTransfers chan transferEnvelope
		if s.l.linkCredit > 0 {
			debug.Log(1, "TX (Sender %p) (enable): target: %q, link credit: %d, deliveryCount: %d", s, s.l.target.Address, s.l.linkCredit, s.l.deliveryCount)
			outgoingTransfers = s.transfers
		} else {
			debug.Log(1, "TX (Sender %p) (pause): target: %q, link credit: %d, deliveryCount: %d", s, s.l.target.Address, s.l.linkCredit, s.l.deliveryCount)
		}

		closed := s.l.close
		if s.l.closeInProgress {
			// swap out channel so it no longer triggers
			closed = nil

			// disable sending once closing is in progress.
			// this prevents races with mux shutdown and
			// the peer sending disposition frames.
			outgoingTransfers = nil
		}

		hooks.MuxSelect()

		select {
		// received frame
		case q := <-s.l.rxQ.Wait():
			// populated queue
			fr := *q.Dequeue()
			s.l.rxQ.Release(q)

			// if muxHandleFrame returns an error it means the mux must terminate.
			// note that in the case of a client-side close due to an error, nil
			// is returned in order to keep the mux running to ack the detach frame.
			if err := s.muxHandleFrame(fr); err != nil {
				s.l.doneErr = err
				return
			}

		// send data
		case env := <-outgoingTransfers:
			hooks.MuxTransfer()
			select {
			case s.l.session.txTransfer <- env:
				debug.Log(2, "TX (Sender %p): mux transfer to Session: %d, %s", s, s.l.session.channel, env.Frame)
				// decrement link-credit after entire message transferred
				if !env.Frame.More {
					s.l.deliveryCount++
					s.l.linkCredit--
					// we are the sender and we keep track of the peer's link credit
					debug.Log(3, "TX (Sender %p): link: %s, link credit: %d", s, s.l.key.name, s.l.linkCredit)
				}
				continue Loop
			case <-s.l.close:
				continue Loop
			case <-s.l.session.done:
				continue Loop
			}

		case <-closed:
			if s.l.closeInProgress {
				// a client-side close due to protocol error is in progress
				continue
			}

			// sender is being closed by the client
			s.l.closeInProgress = true
			fr := &frames.PerformDetach{
				Handle: s.l.outputHandle,
				Closed: true,
			}
			s.l.txFrame(&frameContext{Ctx: context.Background()}, fr)

		case <-s.l.session.done:
			s.l.doneErr = s.l.session.doneErr
			return

		case <-s.rollback:
			s.l.deliveryCount--
			s.l.linkCredit++
			debug.Log(3, "TX (Sender %p): rollback link: %s, link credit: %d", s, s.l.key.name, s.l.linkCredit)
		}
	}
}

// muxHandleFrame processes fr based on type.
// depending on the peer's RSM, it might return a disposition frame for sending
func (s *Sender) muxHandleFrame(fr frames.FrameBody) error {
	debug.Log(2, "RX (Sender %p): %s", s, fr)
	switch fr := fr.(type) {
	// flow control frame
	case *frames.PerformFlow:
		// the sender's link-credit variable MUST be set according to this formula when flow information is given by the receiver:
		// link-credit(snd) := delivery-count(rcv) + link-credit(rcv) - delivery-count(snd)
		linkCredit := *fr.LinkCredit - s.l.deliveryCount
		if fr.DeliveryCount != nil {
			// DeliveryCount can be nil if the receiver hasn't processed
			// the attach. That shouldn't be the case here, but it's
			// what ActiveMQ does.
			linkCredit += *fr.DeliveryCount
		}

		s.l.linkCredit = linkCredit

		if !fr.Echo {
			return nil
		}

		var (
			// copy because sent by pointer below; prevent race
			deliveryCount = s.l.deliveryCount
		)

		// send flow
		resp := &frames.PerformFlow{
			Handle:        &s.l.outputHandle,
			DeliveryCount: &deliveryCount,
			LinkCredit:    &linkCredit, // max number of messages
		}

		select {
		case s.l.session.tx <- frameBodyEnvelope{FrameCtx: &frameContext{Ctx: context.Background()}, FrameBody: resp}:
			debug.Log(2, "TX (Sender %p): mux frame to Session (%p): %d, %s", s, s.l.session, s.l.session.channel, resp)
		case <-s.l.close:
			return nil
		case <-s.l.session.done:
			return s.l.session.doneErr
		}

	case *frames.PerformDisposition:
		if fr.Settled {
			return nil
		}

		// peer is in mode second, so we must send confirmation of disposition.
		// NOTE: the ack must be sent through the session so it can close out
		// the in-flight disposition.
		dr := &frames.PerformDisposition{
			Role:    encoding.RoleSender,
			First:   fr.First,
			Last:    fr.Last,
			Settled: true,
		}

		select {
		case s.l.session.tx <- frameBodyEnvelope{FrameCtx: &frameContext{Ctx: context.Background()}, FrameBody: dr}:
			debug.Log(2, "TX (Sender %p): mux frame to Session (%p): %d, %s", s, s.l.session, s.l.session.channel, dr)
		case <-s.l.close:
			return nil
		case <-s.l.session.done:
			return s.l.session.doneErr
		}

		return nil

	default:
		return s.l.muxHandleFrame(fr)
	}

	return nil
}
