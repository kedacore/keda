package amqp

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"

	"github.com/Azure/go-amqp/internal/bitmap"
	"github.com/Azure/go-amqp/internal/debug"
	"github.com/Azure/go-amqp/internal/encoding"
	"github.com/Azure/go-amqp/internal/frames"
	"github.com/Azure/go-amqp/internal/queue"
)

// Default session options
const (
	defaultWindow = 5000
)

// SessionOptions contains the optional settings for configuring an AMQP session.
type SessionOptions struct {
	// MaxLinks sets the maximum number of links (Senders/Receivers)
	// allowed on the session.
	//
	// Minimum: 1.
	// Default: 4294967295.
	MaxLinks uint32
}

// Session is an AMQP session.
//
// A session multiplexes Receivers.
type Session struct {
	channel       uint16                 // session's local channel
	remoteChannel uint16                 // session's remote channel, owned by conn.connReader
	conn          *Conn                  // underlying conn
	tx            chan frameBodyEnvelope // non-transfer frames to be sent; session must track disposition
	txTransfer    chan transferEnvelope  // transfer frames to be sent; session must track disposition

	// frames destined for this session are added to this queue by conn.connReader
	rxQ *queue.Holder[frames.FrameBody]

	// flow control
	incomingWindow uint32
	outgoingWindow uint32
	needFlowCount  uint32

	handleMax uint32

	// link management
	linksMu    sync.RWMutex      // used to synchronize link handle allocation
	linksByKey map[linkKey]*link // mapping of name+role link
	handles    *bitmap.Bitmap    // allocated handles

	abandonedLinksMu sync.Mutex
	abandonedLinks   []*link

	// used for gracefully closing session
	close     chan struct{} // closed by calling Close(). it signals that the end performative should be sent
	closeOnce sync.Once

	// part of internal public surface area
	done     chan struct{} // closed when the session has terminated (mux exited); DO NOT wait on this from within Session.mux() as it will never trigger!
	endSent  chan struct{} // closed when the end performative has been sent; once this is closed, links MUST NOT send any frames!
	doneErr  error         // contains the mux error state; ONLY written to by the mux and MUST only be read from after done is closed!
	closeErr error         // contains the error state returned from Close(); ONLY Close() reads/writes this!
}

func newSession(c *Conn, channel uint16, opts *SessionOptions) *Session {
	s := &Session{
		conn:           c,
		channel:        channel,
		tx:             make(chan frameBodyEnvelope),
		txTransfer:     make(chan transferEnvelope),
		incomingWindow: defaultWindow,
		outgoingWindow: defaultWindow,
		handleMax:      math.MaxUint32 - 1,
		linksMu:        sync.RWMutex{},
		linksByKey:     make(map[linkKey]*link),
		close:          make(chan struct{}),
		done:           make(chan struct{}),
		endSent:        make(chan struct{}),
	}

	if opts != nil {
		if opts.MaxLinks != 0 {
			// MaxLinks is the number of total links.
			// handleMax is the max handle ID which starts
			// at zero.  so we decrement by one
			s.handleMax = opts.MaxLinks - 1
		}
	}

	// create handle map after options have been applied
	s.handles = bitmap.New(s.handleMax)

	s.rxQ = queue.NewHolder(queue.New[frames.FrameBody](int(s.incomingWindow)))

	return s
}

// waitForFrame waits for an incoming frame to be queued.
// it returns the next frame from the queue, or an error.
// the error is either from the context or conn.doneErr.
// not meant for consumption outside of session.go.
func (s *Session) waitForFrame(ctx context.Context) (frames.FrameBody, error) {
	var q *queue.Queue[frames.FrameBody]
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-s.conn.done:
		return nil, s.conn.doneErr
	case q = <-s.rxQ.Wait():
		// populated queue
	}

	fr := q.Dequeue()
	s.rxQ.Release(q)

	return *fr, nil
}

func (s *Session) begin(ctx context.Context) error {
	// send Begin to server
	begin := &frames.PerformBegin{
		NextOutgoingID: 0,
		IncomingWindow: s.incomingWindow,
		OutgoingWindow: s.outgoingWindow,
		HandleMax:      s.handleMax,
	}

	if err := s.txFrameAndWait(ctx, begin); err != nil {
		return err
	}

	// wait for response
	fr, err := s.waitForFrame(ctx)
	if err != nil {
		// if we exit before receiving the ack, our caller will clean up the channel.
		// however, it does mean that the peer will now have assigned an outgoing
		// channel ID that's not in use.
		return err
	}

	begin, ok := fr.(*frames.PerformBegin)
	if !ok {
		// this codepath is hard to hit (impossible?).  if the response isn't a PerformBegin and we've not
		// yet seen the remote channel number, the default clause in conn.connReader will protect us from that.
		// if we have seen the remote channel number then it's likely the session.mux for that channel will
		// either swallow the frame or blow up in some other way, both causing this call to hang.
		// deallocate session on error.  we can't call
		// s.Close() as the session mux hasn't started yet.
		debug.Log(1, "RX (Session %p): unexpected begin response frame %T", s, fr)
		s.conn.deleteSession(s)
		if err := s.conn.Close(); err != nil {
			return err
		}
		return &ConnError{inner: fmt.Errorf("unexpected begin response: %#v", fr)}
	}

	// start Session multiplexor
	go s.mux(begin)

	return nil
}

// Close closes the session.
//   - ctx controls waiting for the peer to acknowledge the session is closed
//
// If the context's deadline expires or is cancelled before the operation
// completes, an error is returned.  However, the operation will continue to
// execute in the background. Subsequent calls will return a *SessionError
// that contains the context's error message.
func (s *Session) Close(ctx context.Context) error {
	var ctxErr error
	s.closeOnce.Do(func() {
		close(s.close)

		// once the mux has received the ack'ing end performative, the mux will
		// exit which deletes the session and closes s.done.
		select {
		case <-s.done:
			s.closeErr = s.doneErr

		case <-ctx.Done():
			// notify the caller that the close timed out/was cancelled.
			// the mux will remain running and once the ack is received it will terminate.
			ctxErr = ctx.Err()

			// record that the close timed out/was cancelled.
			// subsequent calls to Close() will return this
			debug.Log(1, "TX (Session %p) channel %d: %v", s, s.channel, ctxErr)
			s.closeErr = &SessionError{inner: ctxErr}
		}
	})

	if ctxErr != nil {
		return ctxErr
	}

	var sessionErr *SessionError
	if errors.As(s.closeErr, &sessionErr) && sessionErr.RemoteErr == nil && sessionErr.inner == nil {
		// an empty SessionError means the session was cleanly closed by the caller
		return nil
	}
	return s.closeErr
}

// txFrame sends a frame to the connWriter.
//   - ctx is used to provide the write deadline
//   - fr is the frame to write to net.Conn
//   - sent is the optional channel that will contain the error if the write fails
func (s *Session) txFrame(ctx context.Context, fr frames.FrameBody, sent chan error) {
	debug.Log(2, "TX (Session %p) mux frame to Conn (%p): %s", s, s.conn, fr)
	s.conn.sendFrame(ctx, frames.Frame{
		Type:    frames.TypeAMQP,
		Channel: s.channel,
		Body:    fr,
	}, sent)
}

// txFrameAndWait sends a frame to the connWriter and waits for the write to complete
//   - ctx is used to provide the write deadline
//   - fr is the frame to write to net.Conn
func (s *Session) txFrameAndWait(ctx context.Context, fr frames.FrameBody) error {
	sent := make(chan error, 1)
	s.txFrame(ctx, fr, sent)

	select {
	case err := <-sent:
		return err
	case <-s.conn.done:
		return s.conn.doneErr
	case <-s.done:
		return s.doneErr
	}
}

// NewReceiver opens a new receiver link on the session.
//   - ctx controls waiting for the peer to create a sending terminus
//   - source is the name of the peer's sending terminus
//   - opts contains optional values, pass nil to accept the defaults
//
// If the context's deadline expires or is cancelled before the operation
// completes, an error is returned. If the Receiver was successfully
// created, it will be cleaned up in future calls to NewReceiver.
func (s *Session) NewReceiver(ctx context.Context, source string, opts *ReceiverOptions) (*Receiver, error) {
	r, err := newReceiver(source, s, opts)
	if err != nil {
		return nil, err
	}
	if err = r.attach(ctx); err != nil {
		return nil, err
	}

	go r.mux(receiverTestHooks{})

	return r, nil
}

// NewSender opens a new sender link on the session.
//   - ctx controls waiting for the peer to create a receiver terminus
//   - target is the name of the peer's receiver terminus
//   - opts contains optional values, pass nil to accept the defaults
//
// If the context's deadline expires or is cancelled before the operation
// completes, an error is returned. If the Sender was successfully
// created, it will be cleaned up in future calls to NewSender.
func (s *Session) NewSender(ctx context.Context, target string, opts *SenderOptions) (*Sender, error) {
	l, err := newSender(target, s, opts)
	if err != nil {
		return nil, err
	}
	if err = l.attach(ctx); err != nil {
		return nil, err
	}

	go l.mux(senderTestHooks{})

	return l, nil
}

func (s *Session) mux(remoteBegin *frames.PerformBegin) {
	defer func() {
		if s.doneErr == nil {
			s.doneErr = &SessionError{}
		} else if connErr := (&ConnError{}); !errors.As(s.doneErr, &connErr) {
			// only wrap non-ConnError error types
			var amqpErr *Error
			if errors.As(s.doneErr, &amqpErr) {
				s.doneErr = &SessionError{RemoteErr: amqpErr}
			} else {
				s.doneErr = &SessionError{inner: s.doneErr}
			}
		}
		// Signal goroutines waiting on the session.
		close(s.done)
	}()

	var (
		links                     = make(map[uint32]*link)  // mapping of remote handles to links
		handlesByDeliveryID       = make(map[uint32]uint32) // mapping of deliveryIDs to handles
		deliveryIDByHandle        = make(map[uint32]uint32) // mapping of handles to latest deliveryID
		handlesByRemoteDeliveryID = make(map[uint32]uint32) // mapping of remote deliveryID to handles

		settlementByDeliveryID = make(map[uint32]chan encoding.DeliveryState)

		nextDeliveryID uint32 // tracks the next delivery ID for outgoing transfers

		// flow control values
		nextOutgoingID       uint32
		nextIncomingID       = remoteBegin.NextOutgoingID
		remoteIncomingWindow = remoteBegin.IncomingWindow
		remoteOutgoingWindow = remoteBegin.OutgoingWindow

		closeInProgress bool // indicates the end performative has been sent
	)

	closeWithError := func(e1 *Error, e2 error) {
		if closeInProgress {
			debug.Log(3, "TX (Session %p): close already pending, discarding %v", s, e1)
			return
		}

		closeInProgress = true
		s.doneErr = e2
		s.txFrame(context.Background(), &frames.PerformEnd{Error: e1}, nil)
		close(s.endSent)
	}

	for {
		txTransfer := s.txTransfer
		// disable txTransfer if flow control windows have been exceeded
		if remoteIncomingWindow == 0 || s.outgoingWindow == 0 {
			debug.Log(1, "TX (Session %p): disabling txTransfer - window exceeded. remoteIncomingWindow: %d outgoingWindow: %d",
				s, remoteIncomingWindow, s.outgoingWindow)
			txTransfer = nil
		}

		tx := s.tx
		closed := s.close
		if closeInProgress {
			// swap out channel so it no longer triggers
			closed = nil

			// once the end performative is sent, we're not allowed to send any frames
			tx = nil
			txTransfer = nil
		}

		// notes on client-side closing session
		// when session is closed, we must keep the mux running until the ack'ing end performative
		// has been received. during this window, the session is allowed to receive frames but cannot
		// send them.
		// client-side close happens either by user calling Session.Close() or due to mux initiated
		// close due to a violation of some invariant (see sending &Error{} to s.close). in the case
		// that both code paths have been triggered, we must be careful to preserve the error that
		// triggered the mux initiated close so it can be surfaced to the caller.

		select {
		// conn has completed, exit
		case <-s.conn.done:
			s.doneErr = s.conn.doneErr
			return

		case <-closed:
			if closeInProgress {
				// a client-side close due to protocol error is in progress
				continue
			}
			// session is being closed by the client
			closeInProgress = true
			s.txFrame(context.Background(), &frames.PerformEnd{}, nil)
			close(s.endSent)

		// incoming frame
		case q := <-s.rxQ.Wait():
			fr := *q.Dequeue()
			s.rxQ.Release(q)
			debug.Log(2, "RX (Session %p): %s", s, fr)

			switch body := fr.(type) {
			// Disposition frames can reference transfers from more than one
			// link. Send this frame to all of them.
			case *frames.PerformDisposition:
				start := body.First
				end := start
				if body.Last != nil {
					end = *body.Last
				}
				for deliveryID := start; deliveryID <= end; deliveryID++ {
					handles := handlesByDeliveryID
					if body.Role == encoding.RoleSender {
						handles = handlesByRemoteDeliveryID
					}

					handle, ok := handles[deliveryID]
					if !ok {
						debug.Log(2, "RX (Session %p): role %s: didn't find deliveryID %d in handles map", s, body.Role, deliveryID)
						continue
					}
					delete(handles, deliveryID)

					if body.Settled && body.Role == encoding.RoleReceiver {
						// check if settlement confirmation was requested, if so
						// confirm by closing channel
						if done, ok := settlementByDeliveryID[deliveryID]; ok {
							delete(settlementByDeliveryID, deliveryID)
							select {
							case done <- body.State:
							default:
							}
							close(done)
						}
					}

					link, ok := links[handle]
					if !ok {
						closeWithError(&Error{
							Condition:   ErrCondUnattachedHandle,
							Description: "received disposition frame referencing a handle that's not in use",
						}, fmt.Errorf("received disposition frame with unknown link handle %d", handle))
						continue
					}

					s.muxFrameToLink(link, fr)
				}
				continue
			case *frames.PerformFlow:
				if body.NextIncomingID == nil {
					// This is a protocol error:
					//       "[...] MUST be set if the peer has received
					//        the begin frame for the session"
					closeWithError(&Error{
						Condition:   ErrCondNotAllowed,
						Description: "next-incoming-id not set after session established",
					}, errors.New("protocol error: received flow without next-incoming-id after session established"))
					continue
				}

				// "When the endpoint receives a flow frame from its peer,
				// it MUST update the next-incoming-id directly from the
				// next-outgoing-id of the frame, and it MUST update the
				// remote-outgoing-window directly from the outgoing-window
				// of the frame."
				nextIncomingID = body.NextOutgoingID
				remoteOutgoingWindow = body.OutgoingWindow

				// "The remote-incoming-window is computed as follows:
				//
				// next-incoming-id(flow) + incoming-window(flow) - next-outgoing-id(endpoint)
				//
				// If the next-incoming-id field of the flow frame is not set, then remote-incoming-window is computed as follows:
				//
				// initial-outgoing-id(endpoint) + incoming-window(flow) - next-outgoing-id(endpoint)"
				remoteIncomingWindow = body.IncomingWindow - nextOutgoingID
				remoteIncomingWindow += *body.NextIncomingID
				debug.Log(3, "RX (Session %p): flow - remoteOutgoingWindow: %d remoteIncomingWindow: %d nextOutgoingID: %d", s, remoteOutgoingWindow, remoteIncomingWindow, nextOutgoingID)

				// Send to link if handle is set
				if body.Handle != nil {
					link, ok := links[*body.Handle]
					if !ok {
						closeWithError(&Error{
							Condition:   ErrCondUnattachedHandle,
							Description: "received flow frame referencing a handle that's not in use",
						}, fmt.Errorf("received flow frame with unknown link handle %d", body.Handle))
						continue
					}

					s.muxFrameToLink(link, fr)
					continue
				}

				if body.Echo && !closeInProgress {
					niID := nextIncomingID
					resp := &frames.PerformFlow{
						NextIncomingID: &niID,
						IncomingWindow: s.incomingWindow,
						NextOutgoingID: nextOutgoingID,
						OutgoingWindow: s.outgoingWindow,
					}
					s.txFrame(context.Background(), resp, nil)
				}

			case *frames.PerformAttach:
				// On Attach response link should be looked up by name, then added
				// to the links map with the remote's handle contained in this
				// attach frame.
				//
				// Note body.Role is the remote peer's role, we reverse for the local key.
				s.linksMu.RLock()
				link, linkOk := s.linksByKey[linkKey{name: body.Name, role: !body.Role}]
				s.linksMu.RUnlock()
				if !linkOk {
					closeWithError(&Error{
						Condition:   ErrCondNotAllowed,
						Description: "received mismatched attach frame",
					}, fmt.Errorf("protocol error: received mismatched attach frame %+v", body))
					continue
				}

				link.remoteHandle = body.Handle
				links[link.remoteHandle] = link

				s.muxFrameToLink(link, fr)

			case *frames.PerformTransfer:
				s.needFlowCount++
				// "Upon receiving a transfer, the receiving endpoint will
				// increment the next-incoming-id to match the implicit
				// transfer-id of the incoming transfer plus one, as well
				// as decrementing the remote-outgoing-window, and MAY
				// (depending on policy) decrement its incoming-window."
				nextIncomingID++
				// don't loop to intmax
				if remoteOutgoingWindow > 0 {
					remoteOutgoingWindow--
				}
				link, ok := links[body.Handle]
				if !ok {
					closeWithError(&Error{
						Condition:   ErrCondUnattachedHandle,
						Description: "received transfer frame referencing a handle that's not in use",
					}, fmt.Errorf("received transfer frame with unknown link handle %d", body.Handle))
					continue
				}

				s.muxFrameToLink(link, fr)

				// if this message is received unsettled and link rcv-settle-mode == second, add to handlesByRemoteDeliveryID
				if !body.Settled && body.DeliveryID != nil && link.receiverSettleMode != nil && *link.receiverSettleMode == ReceiverSettleModeSecond {
					debug.Log(1, "RX (Session %p): adding handle to handlesByRemoteDeliveryID. delivery ID: %d", s, *body.DeliveryID)
					handlesByRemoteDeliveryID[*body.DeliveryID] = body.Handle
				}

				// Update peer's outgoing window if half has been consumed.
				if s.needFlowCount >= s.incomingWindow/2 && !closeInProgress {
					debug.Log(3, "RX (Session %p): channel %d: flow - s.needFlowCount(%d) >= s.incomingWindow(%d)/2\n", s, s.channel, s.needFlowCount, s.incomingWindow)
					s.needFlowCount = 0
					nID := nextIncomingID
					flow := &frames.PerformFlow{
						NextIncomingID: &nID,
						IncomingWindow: s.incomingWindow,
						NextOutgoingID: nextOutgoingID,
						OutgoingWindow: s.outgoingWindow,
					}
					s.txFrame(context.Background(), flow, nil)
				}

			case *frames.PerformDetach:
				link, ok := links[body.Handle]
				if !ok {
					closeWithError(&Error{
						Condition:   ErrCondUnattachedHandle,
						Description: "received detach frame referencing a handle that's not in use",
					}, fmt.Errorf("received detach frame with unknown link handle %d", body.Handle))
					continue
				}
				s.muxFrameToLink(link, fr)

				// we received a detach frame and sent it to the link.
				// this was either the response to a client-side initiated
				// detach or our peer detached us. either way, now that
				// the link has processed the frame it's detached so we
				// are safe to clean up its state.
				delete(links, link.remoteHandle)
				delete(deliveryIDByHandle, link.handle)
				s.deallocateHandle(link)

			case *frames.PerformEnd:
				// there are two possibilities:
				// - this is the ack to a client-side Close()
				// - the peer is ending the session so we must ack

				if closeInProgress {
					return
				}

				// peer detached us with an error, save it and send the ack
				if body.Error != nil {
					s.doneErr = body.Error
				}

				fr := frames.PerformEnd{}
				s.txFrame(context.Background(), &fr, nil)

				// per spec, when end is received, we're no longer allowed to receive frames
				return

			default:
				debug.Log(1, "RX (Session %p): unexpected frame: %s\n", s, body)
				closeWithError(&Error{
					Condition:   ErrCondInternalError,
					Description: "session received unexpected frame",
				}, fmt.Errorf("internal error: unexpected frame %T", body))
			}

		case env := <-txTransfer:
			fr := &env.Frame
			// record current delivery ID
			var deliveryID uint32
			if fr.DeliveryID == &needsDeliveryID {
				deliveryID = nextDeliveryID
				fr.DeliveryID = &deliveryID
				nextDeliveryID++
				deliveryIDByHandle[fr.Handle] = deliveryID

				// add to handleByDeliveryID if not sender-settled
				if !fr.Settled {
					handlesByDeliveryID[deliveryID] = fr.Handle
				}
			} else {
				// if fr.DeliveryID is nil it must have been added
				// to deliveryIDByHandle already
				deliveryID = deliveryIDByHandle[fr.Handle]
			}

			// log after the delivery ID has been assigned
			debug.Log(2, "TX (Session %p): %d, %s", s, s.channel, fr)

			// frame has been sender-settled, remove from map
			if fr.Settled {
				delete(handlesByDeliveryID, deliveryID)
			}

			s.txFrame(env.Ctx, fr, env.Sent)
			if sendErr := <-env.Sent; sendErr != nil {
				s.doneErr = sendErr

				// put the error back as our sender will read from this channel
				env.Sent <- sendErr
				return
			}

			// if not settled, add done chan to map
			if !fr.Settled && fr.Done != nil {
				settlementByDeliveryID[deliveryID] = fr.Done
			} else if fr.Done != nil {
				// sender-settled, close done now that the transfer has been sent
				close(fr.Done)
			}

			// "Upon sending a transfer, the sending endpoint will increment
			// its next-outgoing-id, decrement its remote-incoming-window,
			// and MAY (depending on policy) decrement its outgoing-window."
			nextOutgoingID++
			// don't decrement if we're at 0 or we could loop to int max
			if remoteIncomingWindow != 0 {
				remoteIncomingWindow--
			}

		case env := <-tx:
			fr := env.FrameBody
			debug.Log(2, "TX (Session %p): %d, %s", s, s.channel, fr)
			switch fr := env.FrameBody.(type) {
			case *frames.PerformDisposition:
				if fr.Settled && fr.Role == encoding.RoleSender {
					// sender with a peer that's in mode second; sending confirmation of disposition.
					// disposition frames can reference a range of delivery IDs, although it's highly
					// likely in this case there will only be one.
					start := fr.First
					end := start
					if fr.Last != nil {
						end = *fr.Last
					}
					for deliveryID := start; deliveryID <= end; deliveryID++ {
						// send delivery state to the channel and close it to signal
						// that the delivery has completed.
						if done, ok := settlementByDeliveryID[deliveryID]; ok {
							delete(settlementByDeliveryID, deliveryID)
							select {
							case done <- fr.State:
							default:
							}
							close(done)
						}
					}
				}
				s.txFrame(env.Ctx, fr, env.Sent)
			case *frames.PerformFlow:
				niID := nextIncomingID
				fr.NextIncomingID = &niID
				fr.IncomingWindow = s.incomingWindow
				fr.NextOutgoingID = nextOutgoingID
				fr.OutgoingWindow = s.outgoingWindow
				s.txFrame(context.Background(), fr, env.Sent)
			case *frames.PerformTransfer:
				panic("transfer frames must use txTransfer")
			default:
				s.txFrame(context.Background(), fr, env.Sent)
			}
		}
	}
}

func (s *Session) allocateHandle(ctx context.Context, l *link) error {
	s.linksMu.Lock()
	defer s.linksMu.Unlock()

	// Check if link name already exists, if so then an error should be returned
	existing := s.linksByKey[l.key]
	if existing != nil {
		return fmt.Errorf("link with name '%v' already exists", l.key.name)
	}

	next, ok := s.handles.Next()
	if !ok {
		if err := s.Close(ctx); err != nil {
			return err
		}
		// handle numbers are zero-based, report the actual count
		return &SessionError{inner: fmt.Errorf("reached session handle max (%d)", s.handleMax+1)}
	}

	l.handle = next         // allocate handle to the link
	s.linksByKey[l.key] = l // add to mapping

	return nil
}

func (s *Session) deallocateHandle(l *link) {
	s.linksMu.Lock()
	defer s.linksMu.Unlock()

	delete(s.linksByKey, l.key)
	s.handles.Remove(l.handle)
}

func (s *Session) abandonLink(l *link) {
	s.abandonedLinksMu.Lock()
	defer s.abandonedLinksMu.Unlock()
	s.abandonedLinks = append(s.abandonedLinks, l)
}

func (s *Session) freeAbandonedLinks(ctx context.Context) error {
	s.abandonedLinksMu.Lock()
	defer s.abandonedLinksMu.Unlock()

	debug.Log(3, "TX (Session %p): cleaning up %d abandoned links", s, len(s.abandonedLinks))

	for _, l := range s.abandonedLinks {
		dr := &frames.PerformDetach{
			Handle: l.handle,
			Closed: true,
		}
		if err := s.txFrameAndWait(ctx, dr); err != nil {
			return err
		}
	}

	s.abandonedLinks = nil
	return nil
}

func (s *Session) muxFrameToLink(l *link, fr frames.FrameBody) {
	q := l.rxQ.Acquire()
	q.Enqueue(fr)
	l.rxQ.Release(q)
	debug.Log(2, "RX (Session %p): mux frame to link (%p): %s, %s", s, l, l.key.name, fr)
}

// transferEnvelope is used by senders to send transfer frames
type transferEnvelope struct {
	Ctx   context.Context
	Frame frames.PerformTransfer

	// Sent is *never* nil as we use this for confirmation of sending
	// NOTE: use a buffered channel of size 1 when populating
	Sent chan error
}

// frameBodyEnvelope is used by senders and receivers to send frames.
type frameBodyEnvelope struct {
	Ctx       context.Context
	FrameBody frames.FrameBody

	// Sent *can* be nil depending on what frame is being sent.
	// e.g. sending a disposition frame frame a receiver's settlement
	// APIs will have a non-nil channel vs sending a flow frame
	// NOTE: use a buffered channel of size 1 when populating
	Sent chan error
}

// the address of this var is a sentinel value indicating
// that a transfer frame is in need of a delivery ID
var needsDeliveryID uint32
