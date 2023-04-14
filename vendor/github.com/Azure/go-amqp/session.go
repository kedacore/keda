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
	channel       uint16                       // session's local channel
	remoteChannel uint16                       // session's remote channel, owned by conn.connReader
	conn          *Conn                        // underlying conn
	tx            chan frames.FrameBody        // non-transfer frames to be sent; session must track disposition
	txTransfer    chan *frames.PerformTransfer // transfer frames to be sent; session must track disposition

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

	// used for gracefully closing session
	close      chan struct{}
	forceClose chan struct{}
	closeOnce  sync.Once

	// part of internal public surface area
	done     chan struct{} // closed when the session has terminated (mux exited); DO NOT wait on this from within Session.mux() as it will never trigger!
	doneErr  error         // contains the mux error state; ONLY written to by the mux and MUST only be read from after done is closed!
	closeErr error         // contains the error state returned from Close(); ONLY Close() reads/writes this!
}

func newSession(c *Conn, channel uint16, opts *SessionOptions) *Session {
	s := &Session{
		conn:           c,
		channel:        channel,
		tx:             make(chan frames.FrameBody),
		txTransfer:     make(chan *frames.PerformTransfer),
		incomingWindow: defaultWindow,
		outgoingWindow: defaultWindow,
		handleMax:      math.MaxUint32,
		linksMu:        sync.RWMutex{},
		linksByKey:     make(map[linkKey]*link),
		close:          make(chan struct{}),
		forceClose:     make(chan struct{}),
		done:           make(chan struct{}),
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

	_ = s.txFrame(begin, nil)

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
		debug.Log(1, "RX (Session): unexpected begin response frame %T", fr)
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
// completes, the application can be left in an unknown state, potentially
// resulting in connection errors.
func (s *Session) Close(ctx context.Context) error {
	var ctxErr error
	s.closeOnce.Do(func() {
		close(s.close)
		select {
		case <-s.done:
			s.closeErr = s.doneErr
		case <-ctx.Done():
			close(s.forceClose)

			// notify the caller that the close timed out/was cancelled
			ctxErr = ctx.Err()

			// record that the session was forcibly closed.
			// subsequent calls to Close() will return this
			s.closeErr = &SessionError{inner: errSessionForciblyClosed}
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
// it returns an error if the connection has been closed.
func (s *Session) txFrame(p frames.FrameBody, done chan encoding.DeliveryState) error {
	return s.conn.sendFrame(frames.Frame{
		Type:    frames.TypeAMQP,
		Channel: s.channel,
		Body:    p,
		Done:    done,
	})
}

// NewReceiver opens a new receiver link on the session.
//   - ctx controls waiting for the peer to create a sending terminus
//   - source is the name of the peer's sending terminus
//   - opts contains optional values, pass nil to accept the defaults
//
// If the context's deadline expires or is cancelled before the operation
// completes, the application can be left in an unknown state, potentially
// resulting in connection errors.
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
// completes, the application can be left in an unknown state, potentially
// resulting in connection errors.
func (s *Session) NewSender(ctx context.Context, target string, opts *SenderOptions) (*Sender, error) {
	l, err := newSender(target, s, opts)
	if err != nil {
		return nil, err
	}
	if err = l.attach(ctx); err != nil {
		return nil, err
	}

	return l, nil
}

func (s *Session) mux(remoteBegin *frames.PerformBegin) {
	defer func() {
		s.conn.deleteSession(s)
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
			debug.Log(3, "TX (Session): close already pending, discarding %v", e1)
			return
		}

		closeInProgress = true
		s.doneErr = e2
		_ = s.txFrame(&frames.PerformEnd{Error: e1}, nil)
	}

	for {
		txTransfer := s.txTransfer
		// disable txTransfer if flow control windows have been exceeded
		if remoteIncomingWindow == 0 || s.outgoingWindow == 0 {
			debug.Log(1, "TX (Session): disabling txTransfer - window exceeded. remoteIncomingWindow: %d outgoingWindow: %d",
				remoteIncomingWindow,
				s.outgoingWindow)
			txTransfer = nil
		}

		closed := s.close
		if closeInProgress {
			// swap out channel so it no longer triggers
			closed = nil
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

		case <-s.forceClose:
			// the call to s.Close() timed out waiting for the ack
			s.doneErr = errSessionForciblyClosed
			return

		case <-closed:
			if closeInProgress {
				// a client-side close due to protocol error is in progress
				continue
			}
			// session is being closed by the client
			closeInProgress = true
			fr := frames.PerformEnd{}
			_ = s.txFrame(&fr, nil)

		// incoming frame
		case q := <-s.rxQ.Wait():
			fr := *q.Dequeue()
			s.rxQ.Release(q)
			debug.Log(2, "RX (Session): %s", fr)

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
						debug.Log(2, "RX (Session): role %s: didn't find deliveryID %d in handles map", body.Role, deliveryID)
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
				debug.Log(3, "RX (Session): flow - remoteOutgoingWindow: %d remoteIncomingWindow: %d nextOutgoingID: %d", remoteOutgoingWindow, remoteIncomingWindow, nextOutgoingID)

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

				if body.Echo {
					niID := nextIncomingID
					resp := &frames.PerformFlow{
						NextIncomingID: &niID,
						IncomingWindow: s.incomingWindow,
						NextOutgoingID: nextOutgoingID,
						OutgoingWindow: s.outgoingWindow,
					}
					_ = s.txFrame(resp, nil)
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
					debug.Log(1, "RX (Session): adding handle to handlesByRemoteDeliveryID. delivery ID: %d", *body.DeliveryID)
					handlesByRemoteDeliveryID[*body.DeliveryID] = body.Handle
				}

				// Update peer's outgoing window if half has been consumed.
				if s.needFlowCount >= s.incomingWindow/2 {
					debug.Log(3, "RX (Session): channel %d: flow - s.needFlowCount(%d) >= s.incomingWindow(%d)/2\n", s.channel, s.needFlowCount, s.incomingWindow)
					s.needFlowCount = 0
					nID := nextIncomingID
					flow := &frames.PerformFlow{
						NextIncomingID: &nID,
						IncomingWindow: s.incomingWindow,
						NextOutgoingID: nextOutgoingID,
						OutgoingWindow: s.outgoingWindow,
					}
					_ = s.txFrame(flow, nil)
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
				_ = s.txFrame(&fr, nil)

				// per spec, when end is received, we're no longer allowed to receive frames
				return

			default:
				debug.Log(1, "RX (Session): unexpected frame: %s\n", body)
				closeWithError(&Error{
					Condition:   ErrCondInternalError,
					Description: "session received unexpected frame",
				}, fmt.Errorf("internal error: unexpected frame %T", body))
			}

		case fr := <-txTransfer:
			if closeInProgress {
				// now that the end performative has been sent we're
				// not allowed to send any more frames.
				debug.Log(1, "TX (Session): discarding transfer: %s\n", fr)
				continue
			}

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
			debug.Log(2, "TX (Session): %d, %s", s.channel, fr)

			// frame has been sender-settled, remove from map
			if fr.Settled {
				delete(handlesByDeliveryID, deliveryID)
			}

			// if not settled, add done chan to map
			// and clear from frame so conn doesn't close it.
			if !fr.Settled && fr.Done != nil {
				settlementByDeliveryID[deliveryID] = fr.Done
				fr.Done = nil
			}

			_ = s.txFrame(fr, fr.Done)

			// "Upon sending a transfer, the sending endpoint will increment
			// its next-outgoing-id, decrement its remote-incoming-window,
			// and MAY (depending on policy) decrement its outgoing-window."
			nextOutgoingID++
			// don't decrement if we're at 0 or we could loop to int max
			if remoteIncomingWindow != 0 {
				remoteIncomingWindow--
			}

		case fr := <-s.tx:
			if closeInProgress {
				// now that the end performative has been sent we're
				// not allowed to send any more frames.
				debug.Log(1, "TX (Session): discarding frame: %s\n", fr)
				continue
			}

			debug.Log(2, "TX (Session): %d, %s", s.channel, fr)
			switch fr := fr.(type) {
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
				_ = s.txFrame(fr, nil)
			case *frames.PerformFlow:
				niID := nextIncomingID
				fr.NextIncomingID = &niID
				fr.IncomingWindow = s.incomingWindow
				fr.NextOutgoingID = nextOutgoingID
				fr.OutgoingWindow = s.outgoingWindow
				_ = s.txFrame(fr, nil)
			case *frames.PerformTransfer:
				panic("transfer frames must use txTransfer")
			default:
				_ = s.txFrame(fr, nil)
			}
		}
	}
}

func (s *Session) allocateHandle(l *link) error {
	s.linksMu.Lock()
	defer s.linksMu.Unlock()

	// Check if link name already exists, if so then an error should be returned
	existing := s.linksByKey[l.key]
	if existing != nil {
		return fmt.Errorf("link with name '%v' already exists", l.key.name)
	}

	next, ok := s.handles.Next()
	if !ok {
		// handle numbers are zero-based, report the actual count
		return fmt.Errorf("reached session handle max (%d)", s.handleMax+1)
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

func (s *Session) muxFrameToLink(l *link, fr frames.FrameBody) {
	q := l.rxQ.Acquire()
	q.Enqueue(fr)
	l.rxQ.Release(q)
	debug.Log(2, "RX (Session): mux frame to link: %s, %s", l.key.name, fr)
}

// the address of this var is a sentinel value indicating
// that a transfer frame is in need of a delivery ID
var needsDeliveryID uint32

var errSessionForciblyClosed = errors.New("the session was forcibly closed")
