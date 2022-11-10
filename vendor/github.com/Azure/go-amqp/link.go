package amqp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/Azure/go-amqp/internal/buffer"
	"github.com/Azure/go-amqp/internal/encoding"
	"github.com/Azure/go-amqp/internal/frames"
)

// link is a unidirectional route.
//
// May be used for sending or receiving.
type link struct {
	Key          linkKey                     // Name and direction
	Handle       uint32                      // our handle
	RemoteHandle uint32                      // remote's handle
	dynamicAddr  bool                        // request a dynamic link address from the server
	RX           chan frames.FrameBody       // sessions sends frames for this link on this channel
	Transfers    chan frames.PerformTransfer // sender uses to send transfer frames
	closeOnce    sync.Once                   // closeOnce protects close from being closed multiple times

	// close signals the mux to shutdown. This indicates that `Close()` was called on this link.
	// NOTE: observers outside of link.go *must only* use the Detached channel to check if the link is unavailable.
	// including the close channel will lead to a race condition.
	close chan struct{}

	// detached is closed by mux/muxDetach when the link is fully detached.
	// This will be initiated if the service sends back an error or requests the link detach.
	Detached chan struct{}

	detachErrorMu sync.Mutex                      // protects detachError
	detachError   *Error                          // error to send to remote on detach, set by closeWithError
	Session       *Session                        // parent session
	receiver      *Receiver                       // allows link options to modify Receiver
	Source        *frames.Source                  // used for Receiver links
	Target        *frames.Target                  // used for Sender links
	properties    map[encoding.Symbol]interface{} // additional properties sent upon link attach
	// Indicates whether we should allow detaches on disposition errors or not.
	// Some AMQP servers (like Event Hubs) benefit from keeping the link open on disposition errors
	// (for instance, if you're doing many parallel sends over the same link and you get back a
	// throttling error, which is not fatal)
	detachOnDispositionError bool

	// "The delivery-count is initialized by the sender when a link endpoint is created,
	// and is incremented whenever a message is sent. Only the sender MAY independently
	// modify this field. The receiver's value is calculated based on the last known
	// value from the sender and any subsequent messages received on the link. Note that,
	// despite its name, the delivery-count is not a count but a sequence number
	// initialized at an arbitrary point by the sender."
	deliveryCount      uint32
	linkCredit         uint32 // maximum number of messages allowed between flow updates
	SenderSettleMode   *SenderSettleMode
	ReceiverSettleMode *ReceiverSettleMode
	MaxMessageSize     uint64
	detachReceived     bool
	err                error // err returned on Close()

	// message receiving
	Paused                uint32              // atomically accessed; indicates that all link credits have been used by sender
	ReceiverReady         chan struct{}       // receiver sends on this when mux is paused to indicate it can handle more messages
	Messages              chan Message        // used to send completed messages to receiver
	unsettledMessages     map[string]struct{} // used to keep track of messages being handled downstream
	unsettledMessagesLock sync.RWMutex        // lock to protect concurrent access to unsettledMessages
	buf                   buffer.Buffer       // buffered bytes for current message
	more                  bool                // if true, buf contains a partial message
	msg                   Message             // current message being decoded
}

func newLink(s *Session, r *Receiver, opts []LinkOption) (*link, error) {
	l := &link{
		Key:                      linkKey{randString(40), encoding.Role(r != nil)},
		Session:                  s,
		receiver:                 r,
		close:                    make(chan struct{}),
		Detached:                 make(chan struct{}),
		ReceiverReady:            make(chan struct{}, 1),
		detachOnDispositionError: true,
	}

	// configure options
	for _, o := range opts {
		err := o(l)
		if err != nil {
			return nil, err
		}
	}

	// sending unsettled messages when the receiver is in mode-second is currently
	// broken and causes a hang after sending, so just disallow it for now.
	if r == nil && senderSettleModeValue(l.SenderSettleMode) != ModeSettled && receiverSettleModeValue(l.ReceiverSettleMode) == ModeSecond {
		return nil, errors.New("sender does not support exactly-once guarantee")
	}
	return l, nil
}

// attachLink is used by Receiver and Sender to create new links
func attachLink(s *Session, r *Receiver, opts []LinkOption) (*link, error) {
	l, err := newLink(s, r, opts)
	if err != nil {
		return nil, err
	}

	isReceiver := r != nil

	// buffer rx to linkCredit so that conn.mux won't block
	// attempting to send to a slow reader
	if isReceiver {
		if l.receiver.manualCreditor != nil {
			l.RX = make(chan frames.FrameBody, l.receiver.maxCredit)
		} else {
			l.RX = make(chan frames.FrameBody, l.linkCredit)
		}
	} else {
		l.RX = make(chan frames.FrameBody, 1)
	}

	// request handle from Session.mux
	select {
	case <-s.done:
		return nil, s.err
	case s.allocateHandle <- l:
	}

	// wait for handle allocation
	select {
	case <-s.done:
		return nil, s.err
	case <-l.RX:
	}

	// check for link request error
	if l.err != nil {
		return nil, l.err
	}

	attach := &frames.PerformAttach{
		Name:               l.Key.name,
		Handle:             l.Handle,
		ReceiverSettleMode: l.ReceiverSettleMode,
		SenderSettleMode:   l.SenderSettleMode,
		MaxMessageSize:     l.MaxMessageSize,
		Source:             l.Source,
		Target:             l.Target,
		Properties:         l.properties,
	}

	if isReceiver {
		attach.Role = encoding.RoleReceiver
		if attach.Source == nil {
			attach.Source = new(frames.Source)
		}
		attach.Source.Dynamic = l.dynamicAddr
	} else {
		attach.Role = encoding.RoleSender
		if attach.Target == nil {
			attach.Target = new(frames.Target)
		}
		attach.Target.Dynamic = l.dynamicAddr
	}

	// send Attach frame
	debug(1, "TX (attachLink): %s", attach)
	_ = s.txFrame(attach, nil)

	// wait for response
	var fr frames.FrameBody
	select {
	case <-s.done:
		return nil, s.err
	case fr = <-l.RX:
	}
	debug(3, "RX (attachLink): %s", fr)
	resp, ok := fr.(*frames.PerformAttach)
	if !ok {
		return nil, fmt.Errorf("unexpected attach response: %#v", fr)
	}

	// If the remote encounters an error during the attach it returns an Attach
	// with no Source or Target. The remote then sends a Detach with an error.
	//
	//   Note that if the application chooses not to create a terminus, the session
	//   endpoint will still create a link endpoint and issue an attach indicating
	//   that the link endpoint has no associated local terminus. In this case, the
	//   session endpoint MUST immediately detach the newly created link endpoint.
	//
	// http://docs.oasis-open.org/amqp/core/v1.0/csprd01/amqp-core-transport-v1.0-csprd01.html#doc-idp386144
	if resp.Source == nil && resp.Target == nil {
		// wait for detach
		select {
		case <-s.done:
			return nil, s.err
		case fr = <-l.RX:
		}

		detach, ok := fr.(*frames.PerformDetach)
		if !ok {
			return nil, fmt.Errorf("unexpected frame while waiting for detach: %#v", fr)
		}

		// send return detach
		fr = &frames.PerformDetach{
			Handle: l.Handle,
			Closed: true,
		}
		debug(1, "TX (attachLink): %s", fr)
		_ = s.txFrame(fr, nil)

		if detach.Error == nil {
			return nil, fmt.Errorf("received detach with no error specified")
		}
		return nil, detach.Error
	}

	if l.MaxMessageSize == 0 || resp.MaxMessageSize < l.MaxMessageSize {
		l.MaxMessageSize = resp.MaxMessageSize
	}

	if isReceiver {
		if l.Source == nil {
			l.Source = new(frames.Source)
		}
		// if dynamic address requested, copy assigned name to address
		if l.dynamicAddr && resp.Source != nil {
			l.Source.Address = resp.Source.Address
		}
		// deliveryCount is a sequence number, must initialize to sender's initial sequence number
		l.deliveryCount = resp.InitialDeliveryCount
		// buffer receiver so that link.mux doesn't block
		l.Messages = make(chan Message, l.receiver.maxCredit)
		l.unsettledMessages = map[string]struct{}{}
		// copy the received filter values
		l.Source.Filter = resp.Source.Filter
	} else {
		if l.Target == nil {
			l.Target = new(frames.Target)
		}
		// if dynamic address requested, copy assigned name to address
		if l.dynamicAddr && resp.Target != nil {
			l.Target.Address = resp.Target.Address
		}
		l.Transfers = make(chan frames.PerformTransfer)
	}

	err = l.setSettleModes(resp)
	if err != nil {
		l.muxDetach()
		return nil, err
	}

	go l.mux()

	return l, nil
}

func (l *link) addUnsettled(msg *Message) {
	l.unsettledMessagesLock.Lock()
	l.unsettledMessages[string(msg.DeliveryTag)] = struct{}{}
	l.unsettledMessagesLock.Unlock()
}

// DeleteUnsettled removes the message from the map of unsettled messages.
func (l *link) DeleteUnsettled(msg *Message) {
	l.unsettledMessagesLock.Lock()
	delete(l.unsettledMessages, string(msg.DeliveryTag))
	l.unsettledMessagesLock.Unlock()
}

func (l *link) countUnsettled() int {
	l.unsettledMessagesLock.RLock()
	count := len(l.unsettledMessages)
	l.unsettledMessagesLock.RUnlock()
	return count
}

// setSettleModes sets the settlement modes based on the resp frames.PerformAttach.
//
// If a settlement mode has been explicitly set locally and it was not honored by the
// server an error is returned.
func (l *link) setSettleModes(resp *frames.PerformAttach) error {
	var (
		localRecvSettle = receiverSettleModeValue(l.ReceiverSettleMode)
		respRecvSettle  = receiverSettleModeValue(resp.ReceiverSettleMode)
	)
	if l.ReceiverSettleMode != nil && localRecvSettle != respRecvSettle {
		return fmt.Errorf("amqp: receiver settlement mode %q requested, received %q from server", l.ReceiverSettleMode, &respRecvSettle)
	}
	l.ReceiverSettleMode = &respRecvSettle

	var (
		localSendSettle = senderSettleModeValue(l.SenderSettleMode)
		respSendSettle  = senderSettleModeValue(resp.SenderSettleMode)
	)
	if l.SenderSettleMode != nil && localSendSettle != respSendSettle {
		return fmt.Errorf("amqp: sender settlement mode %q requested, received %q from server", l.SenderSettleMode, &respSendSettle)
	}
	l.SenderSettleMode = &respSendSettle

	return nil
}

// doFlow handles the logical 'flow' event for a link.
// For receivers it will send (if needed) an AMQP flow frame, via `muxFlow`. If a fatal error
// occurs it will be set in `l.err` and 'ok' will be false.
// For senders it will indicate if we should try to send any outgoing transfers (the logical
// equivalent of a flow for a sender) by returning true for 'enableOutgoingTransfers'.
func (l *link) doFlow() (ok bool, enableOutgoingTransfers bool) {
	var (
		isReceiver = l.receiver != nil
		isSender   = !isReceiver
	)

	switch {
	// enable outgoing transfers case if sender and credits are available
	case isSender && l.linkCredit > 0:
		debug(1, "Link Mux isSender: credit: %d, deliveryCount: %d, messages: %d, unsettled: %d", l.linkCredit, l.deliveryCount, len(l.Messages), l.countUnsettled())
		return true, true

	case isReceiver && l.receiver.manualCreditor != nil:
		drain, credits := l.receiver.manualCreditor.FlowBits(l.linkCredit)

		if drain || credits > 0 {
			debug(1, "FLOW Link Mux (manual): source: %s, inflight: %d, credit: %d, creditsToAdd: %d, drain: %v, deliveryCount: %d, messages: %d, unsettled: %d, maxCredit : %d, settleMode: %s",
				l.Source.Address, l.receiver.inFlight.len(), l.linkCredit, credits, drain, l.deliveryCount, len(l.Messages), l.countUnsettled(), l.receiver.maxCredit, l.ReceiverSettleMode.String())

			// send a flow frame.
			l.err = l.muxFlow(credits, drain)
		}

	// if receiver && half maxCredits have been processed, send more credits
	case isReceiver && l.linkCredit+uint32(l.countUnsettled()) <= l.receiver.maxCredit/2:
		debug(1, "FLOW Link Mux half: source: %s, inflight: %d, credit: %d, deliveryCount: %d, messages: %d, unsettled: %d, maxCredit : %d, settleMode: %s", l.Source.Address, l.receiver.inFlight.len(), l.linkCredit, l.deliveryCount, len(l.Messages), l.countUnsettled(), l.receiver.maxCredit, l.ReceiverSettleMode.String())

		linkCredit := l.receiver.maxCredit - uint32(l.countUnsettled())
		l.err = l.muxFlow(linkCredit, false)

		if l.err != nil {
			return false, false
		}
		atomic.StoreUint32(&l.Paused, 0)

	case isReceiver && l.linkCredit == 0:
		debug(1, "PAUSE Link Mux pause: inflight: %d, credit: %d, deliveryCount: %d, messages: %d, unsettled: %d, maxCredit : %d, settleMode: %s", l.receiver.inFlight.len(), l.linkCredit, l.deliveryCount, len(l.Messages), l.countUnsettled(), l.receiver.maxCredit, l.ReceiverSettleMode.String())
		atomic.StoreUint32(&l.Paused, 1)
	}

	return true, false
}

func (l *link) mux() {
	defer l.muxDetach()

Loop:
	for {
		var outgoingTransfers chan frames.PerformTransfer

		ok, enableOutgoingTransfers := l.doFlow()

		if !ok {
			return
		}

		if enableOutgoingTransfers {
			outgoingTransfers = l.Transfers
		}

		select {
		// received frame
		case fr := <-l.RX:
			l.err = l.muxHandleFrame(fr)
			if l.err != nil {
				return
			}

		// send data
		case tr := <-outgoingTransfers:
			debug(3, "TX(link): %s", tr)

			// Ensure the session mux is not blocked
			for {
				select {
				case l.Session.txTransfer <- &tr:
					// decrement link-credit after entire message transferred
					if !tr.More {
						l.deliveryCount++
						l.linkCredit--
						// we are the sender and we keep track of the peer's link credit
						debug(3, "TX(link): key:%s, decremented linkCredit: %d", l.Key.name, l.linkCredit)
					}
					continue Loop
				case fr := <-l.RX:
					l.err = l.muxHandleFrame(fr)
					if l.err != nil {
						return
					}
				case <-l.close:
					l.err = ErrLinkClosed
					return
				case <-l.Session.done:
					l.err = l.Session.err
					return
				}
			}

		case <-l.ReceiverReady:
			continue
		case <-l.close:
			l.err = ErrLinkClosed
			return
		case <-l.Session.done:
			l.err = l.Session.err
			return
		}
	}
}

// muxFlow sends tr to the session mux.
// l.linkCredit will also be updated to `linkCredit`
func (l *link) muxFlow(linkCredit uint32, drain bool) error {
	var (
		deliveryCount = l.deliveryCount
	)

	debug(3, "link.muxFlow(): len(l.Messages):%d - linkCredit: %d - deliveryCount: %d, inFlight: %d", len(l.Messages), linkCredit, deliveryCount, l.receiver.inFlight.len())

	fr := &frames.PerformFlow{
		Handle:        &l.Handle,
		DeliveryCount: &deliveryCount,
		LinkCredit:    &linkCredit, // max number of messages,
		Drain:         drain,
	}
	debug(3, "TX (muxFlow): %s", fr)

	// Update credit. This must happen before entering loop below
	// because incoming messages handled while waiting to transmit
	// flow increment deliveryCount. This causes the credit to become
	// out of sync with the server.

	if !drain {
		// if we're draining we don't want to touch our internal credit - we're not changing it so any issued credits
		// are still valid until drain completes, at which point they will be naturally zeroed.
		l.linkCredit = linkCredit
	}

	// Ensure the session mux is not blocked
	for {
		select {
		case l.Session.tx <- fr:
			return nil
		case fr := <-l.RX:
			err := l.muxHandleFrame(fr)
			if err != nil {
				return err
			}
		case <-l.close:
			return ErrLinkClosed
		case <-l.Session.done:
			return l.Session.err
		}
	}
}

func (l *link) muxReceive(fr frames.PerformTransfer) error {
	if !l.more {
		// this is the first transfer of a message,
		// record the delivery ID, message format,
		// and delivery Tag
		if fr.DeliveryID != nil {
			l.msg.deliveryID = *fr.DeliveryID
		}
		if fr.MessageFormat != nil {
			l.msg.Format = *fr.MessageFormat
		}
		l.msg.DeliveryTag = fr.DeliveryTag

		// these fields are required on first transfer of a message
		if fr.DeliveryID == nil {
			return l.closeWithError(&Error{
				Condition:   ErrorNotAllowed,
				Description: "received message without a delivery-id",
			})
		}
		if fr.MessageFormat == nil {
			return l.closeWithError(&Error{
				Condition:   ErrorNotAllowed,
				Description: "received message without a message-format",
			})
		}
		if fr.DeliveryTag == nil {
			return l.closeWithError(&Error{
				Condition:   ErrorNotAllowed,
				Description: "received message without a delivery-tag",
			})
		}
	} else {
		// this is a continuation of a multipart message
		// some fields may be omitted on continuation transfers,
		// but if they are included they must be consistent
		// with the first.

		if fr.DeliveryID != nil && *fr.DeliveryID != l.msg.deliveryID {
			msg := fmt.Sprintf(
				"received continuation transfer with inconsistent delivery-id: %d != %d",
				*fr.DeliveryID, l.msg.deliveryID,
			)
			return l.closeWithError(&Error{
				Condition:   ErrorNotAllowed,
				Description: msg,
			})
		}
		if fr.MessageFormat != nil && *fr.MessageFormat != l.msg.Format {
			msg := fmt.Sprintf(
				"received continuation transfer with inconsistent message-format: %d != %d",
				*fr.MessageFormat, l.msg.Format,
			)
			return l.closeWithError(&Error{
				Condition:   ErrorNotAllowed,
				Description: msg,
			})
		}
		if fr.DeliveryTag != nil && !bytes.Equal(fr.DeliveryTag, l.msg.DeliveryTag) {
			msg := fmt.Sprintf(
				"received continuation transfer with inconsistent delivery-tag: %q != %q",
				fr.DeliveryTag, l.msg.DeliveryTag,
			)
			return l.closeWithError(&Error{
				Condition:   ErrorNotAllowed,
				Description: msg,
			})
		}
	}

	// discard message if it's been aborted
	if fr.Aborted {
		l.buf.Reset()
		l.msg = Message{}
		l.more = false
		return nil
	}

	// ensure maxMessageSize will not be exceeded
	if l.MaxMessageSize != 0 && uint64(l.buf.Len())+uint64(len(fr.Payload)) > l.MaxMessageSize {
		return l.closeWithError(&Error{
			Condition:   ErrorMessageSizeExceeded,
			Description: fmt.Sprintf("received message larger than max size of %d", l.MaxMessageSize),
		})
	}

	// add the payload the the buffer
	l.buf.Append(fr.Payload)

	// mark as settled if at least one frame is settled
	l.msg.settled = l.msg.settled || fr.Settled

	// save in-progress status
	l.more = fr.More

	if fr.More {
		return nil
	}

	// last frame in message
	err := l.msg.Unmarshal(&l.buf)
	if err != nil {
		return err
	}
	debug(1, "deliveryID %d before push to receiver - deliveryCount : %d - linkCredit: %d, len(messages): %d, len(inflight): %d", l.msg.deliveryID, l.deliveryCount, l.linkCredit, len(l.Messages), l.receiver.inFlight.len())
	// send to receiver
	if receiverSettleModeValue(l.ReceiverSettleMode) == ModeSecond {
		l.addUnsettled(&l.msg)
	}
	select {
	case l.Messages <- l.msg:
		// message received
	case <-l.close:
		// link is being closed
		return l.err
	}

	debug(1, "deliveryID %d after push to receiver - deliveryCount : %d - linkCredit: %d, len(messages): %d, len(inflight): %d", l.msg.deliveryID, l.deliveryCount, l.linkCredit, len(l.Messages), l.receiver.inFlight.len())

	// reset progress
	l.buf.Reset()
	l.msg = Message{}

	// decrement link-credit after entire message received
	l.deliveryCount++
	l.linkCredit--
	debug(1, "deliveryID %d before exit - deliveryCount : %d - linkCredit: %d, len(messages): %d", l.msg.deliveryID, l.deliveryCount, l.linkCredit, len(l.Messages))
	return nil
}

// DrainCredit will cause a flow frame with 'drain' set to true when
// the next flow frame is sent in 'mux()'.
// Applicable only when manual credit management has been enabled.
func (l *link) DrainCredit(ctx context.Context) error {
	if l.receiver == nil || l.receiver.manualCreditor == nil {
		return errors.New("drain can only be used with receiver links using manual credit management")
	}

	// cause mux() to check our flow conditions.
	select {
	case l.ReceiverReady <- struct{}{}:
	default:
	}

	return l.receiver.manualCreditor.Drain(ctx, l)
}

// IssueCredit requests additional credits be issued for this link.
// Applicable only when manual credit management has been enabled.
func (l *link) IssueCredit(credit uint32) error {
	if l.receiver == nil || l.receiver.manualCreditor == nil {
		return errors.New("issueCredit can only be used with receiver links using manual credit management")
	}

	if err := l.receiver.manualCreditor.IssueCredit(credit); err != nil {
		return err
	}

	// cause mux() to check our flow conditions.
	select {
	case l.ReceiverReady <- struct{}{}:
	default:
	}

	return nil
}

func (l *link) detachOnRejectDisp() bool {
	// only detach on rejection when no RSM was requested or in ModeFirst.
	// if the receiver is in ModeSecond, it will send an explicit rejection disposition
	// that we'll have to ack. so in that case, we don't treat it as a link error.
	if l.detachOnDispositionError && (l.receiver == nil && (l.ReceiverSettleMode == nil || *l.ReceiverSettleMode == ModeFirst)) {
		return true
	}
	return false
}

// muxHandleFrame processes fr based on type.
func (l *link) muxHandleFrame(fr frames.FrameBody) error {
	var (
		isSender = l.receiver == nil
	)

	switch fr := fr.(type) {
	// message frame
	case *frames.PerformTransfer:
		debug(3, "RX (muxHandleFrame): %s", fr)
		if isSender {
			// Senders should never receive transfer frames, but handle it just in case.
			return l.closeWithError(&Error{
				Condition:   ErrorNotAllowed,
				Description: "sender cannot process transfer frame",
			})
		}

		return l.muxReceive(*fr)

	// flow control frame
	case *frames.PerformFlow:
		debug(3, "RX (muxHandleFrame): %s", fr)
		if isSender {
			linkCredit := *fr.LinkCredit - l.deliveryCount
			if fr.DeliveryCount != nil {
				// DeliveryCount can be nil if the receiver hasn't processed
				// the attach. That shouldn't be the case here, but it's
				// what ActiveMQ does.
				linkCredit += *fr.DeliveryCount
			}
			l.linkCredit = linkCredit
		}

		if !fr.Echo {
			// if the 'drain' flag has been set in the frame sent to the _receiver_ then
			// we signal whomever is waiting (the service has seen and acknowledged our drain)
			if fr.Drain && l.receiver.manualCreditor != nil {
				l.linkCredit = 0 // we have no active credits at this point.
				l.receiver.manualCreditor.EndDrain()
			}
			return nil
		}

		var (
			// copy because sent by pointer below; prevent race
			linkCredit    = l.linkCredit
			deliveryCount = l.deliveryCount
		)

		// send flow
		resp := &frames.PerformFlow{
			Handle:        &l.Handle,
			DeliveryCount: &deliveryCount,
			LinkCredit:    &linkCredit, // max number of messages
		}
		debug(1, "TX (muxHandleFrame): %s", resp)
		_ = l.Session.txFrame(resp, nil)

	// remote side is closing links
	case *frames.PerformDetach:
		debug(1, "RX (muxHandleFrame): %s", fr)
		// don't currently support link detach and reattach
		if !fr.Closed {
			return fmt.Errorf("non-closing detach not supported: %+v", fr)
		}

		// set detach received and close link
		l.detachReceived = true

		return &DetachError{fr.Error}

	case *frames.PerformDisposition:
		debug(3, "RX (muxHandleFrame): %s", fr)

		// Unblock receivers waiting for message disposition
		if l.receiver != nil {
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
			l.receiver.inFlight.remove(fr.First, fr.Last, dispositionError)
		}

		// If sending async and a message is rejected, cause a link error.
		//
		// This isn't ideal, but there isn't a clear better way to handle it.
		if fr, ok := fr.State.(*encoding.StateRejected); ok && l.detachOnRejectDisp() {
			return &DetachError{fr.Error}
		}

		if fr.Settled {
			return nil
		}

		resp := &frames.PerformDisposition{
			Role:    encoding.RoleSender,
			First:   fr.First,
			Last:    fr.Last,
			Settled: true,
		}
		debug(1, "TX (muxHandleFrame): %s", resp)
		_ = l.Session.txFrame(resp, nil)

	default:
		// TODO: evaluate
		debug(1, "muxHandleFrame: unexpected frame: %s\n", fr)
	}

	return nil
}

// close closes and requests deletion of the link.
//
// No operations on link are valid after close.
//
// If ctx expires while waiting for servers response, ctx.Err() will be returned.
// The session will continue to wait for the response until the Session or Client
// is closed.
func (l *link) Close(ctx context.Context) error {
	l.closeOnce.Do(func() { close(l.close) })
	select {
	case <-l.Detached:
		// mux exited
	case <-ctx.Done():
		return ctx.Err()
	}
	if l.err == ErrLinkClosed {
		return nil
	}
	return l.err
}

// returns the error passed in
func (l *link) closeWithError(de *Error) error {
	l.closeOnce.Do(func() {
		l.detachErrorMu.Lock()
		l.detachError = de
		l.detachErrorMu.Unlock()
		close(l.close)
	})
	return de
}

func (l *link) muxDetach() {
	defer func() {
		// final cleanup and signaling

		// deallocate handle
	Loop:
		for {
			select {
			case <-l.RX:
				// at this point we shouldn't be receiving any more frames for
				// this link. however, if we do, we need to keep the session mux
				// unblocked else we deadlock.  so just read and discard them.
			case l.Session.deallocateHandle <- l:
				break Loop
			case <-l.Session.done:
				if l.err == nil {
					l.err = l.Session.err
				}
				break Loop
			}
		}

		// unblock any in flight message dispositions
		if l.receiver != nil {
			l.receiver.inFlight.clear(l.err)
		}

		// unblock any pending drain requests
		if l.receiver != nil && l.receiver.manualCreditor != nil {
			l.receiver.manualCreditor.EndDrain()
		}

		// signal that the link mux has exited
		close(l.Detached)
	}()

	// "A peer closes a link by sending the detach frame with the
	// handle for the specified link, and the closed flag set to
	// true. The partner will destroy the corresponding link
	// endpoint, and reply with its own detach frame with the
	// closed flag set to true.
	//
	// Note that one peer MAY send a closing detach while its
	// partner is sending a non-closing detach. In this case,
	// the partner MUST signal that it has closed the link by
	// reattaching and then sending a closing detach."

	l.detachErrorMu.Lock()
	detachError := l.detachError
	l.detachErrorMu.Unlock()

	fr := &frames.PerformDetach{
		Handle: l.Handle,
		Closed: true,
		Error:  detachError,
	}

Loop:
	for {
		select {
		case l.Session.tx <- fr:
			// after sending the detach frame, break the read loop
			break Loop
		case fr := <-l.RX:
			// discard incoming frames to avoid blocking session.mux
			if fr, ok := fr.(*frames.PerformDetach); ok && fr.Closed {
				l.detachReceived = true
			}
		case <-l.Session.done:
			if l.err == nil {
				l.err = l.Session.err
			}
			return
		}
	}

	// don't wait for remote to detach when already
	// received or closing due to error
	if l.detachReceived || detachError != nil {
		return
	}

	for {
		select {
		// read from link until detach with Close == true is received,
		// other frames are discarded.
		case fr := <-l.RX:
			if fr, ok := fr.(*frames.PerformDetach); ok && fr.Closed {
				return
			}

		// connection has ended
		case <-l.Session.done:
			if l.err == nil {
				l.err = l.Session.err
			}
			return
		}
	}
}
