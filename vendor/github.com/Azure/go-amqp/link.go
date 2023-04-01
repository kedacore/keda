package amqp

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/Azure/go-amqp/internal/debug"
	"github.com/Azure/go-amqp/internal/encoding"
	"github.com/Azure/go-amqp/internal/frames"
	"github.com/Azure/go-amqp/internal/queue"
	"github.com/Azure/go-amqp/internal/shared"
)

// linkKey uniquely identifies a link on a connection by name and direction.
//
// A link can be identified uniquely by the ordered tuple
//
//	(source-container-id, target-container-id, name)
//
// On a single connection the container ID pairs can be abbreviated
// to a boolean flag indicating the direction of the link.
type linkKey struct {
	name string
	role encoding.Role // Local role: sender/receiver
}

// link contains the common state and methods for sending and receiving links
type link struct {
	key          linkKey // Name and direction
	handle       uint32  // our handle
	remoteHandle uint32  // remote's handle
	dynamicAddr  bool    // request a dynamic link address from the server

	// frames destined for this link are added to this queue by Session.muxFrameToLink
	rxQ *queue.Holder[frames.FrameBody]

	// used for gracefully closing link
	close      chan struct{} // signals a link's mux to shut down; DO NOT use this to check if a link has terminated, use done instead
	forceClose chan struct{} // used for forcibly terminate a link if Close() times out/is cancelled
	closeOnce  *sync.Once    // closeOnce protects close from being closed multiple times

	done     chan struct{} // closed when the link has terminated (mux exited); DO NOT wait on this from within a link's mux() as it will never trigger!
	doneErr  error         // contains the mux error state; ONLY written to by the mux and MUST only be read from after done is closed!
	closeErr error         // contains the error state returned from closeLink(); ONLY closeLink() reads/writes this!

	session    *Session                // parent session
	source     *frames.Source          // used for Receiver links
	target     *frames.Target          // used for Sender links
	properties map[encoding.Symbol]any // additional properties sent upon link attach

	// "The delivery-count is initialized by the sender when a link endpoint is created,
	// and is incremented whenever a message is sent. Only the sender MAY independently
	// modify this field. The receiver's value is calculated based on the last known
	// value from the sender and any subsequent messages received on the link. Note that,
	// despite its name, the delivery-count is not a count but a sequence number
	// initialized at an arbitrary point by the sender."
	deliveryCount uint32

	// The current maximum number of messages that can be handled at the receiver endpoint of the link. Only the receiver endpoint
	// can independently set this value. The sender endpoint sets this to the last known value seen from the receiver.
	linkCredit uint32

	senderSettleMode   *SenderSettleMode
	receiverSettleMode *ReceiverSettleMode
	maxMessageSize     uint64

	closeInProgress bool // indicates that the detach performative has been sent
}

func newLink(s *Session, r encoding.Role) link {
	l := link{
		key:        linkKey{shared.RandString(40), r},
		session:    s,
		close:      make(chan struct{}),
		forceClose: make(chan struct{}),
		closeOnce:  &sync.Once{},
		done:       make(chan struct{}),
	}

	// set the segment size relative to respective window
	var segmentSize int
	if r == encoding.RoleReceiver {
		segmentSize = int(s.incomingWindow)
	} else {
		segmentSize = int(s.outgoingWindow)
	}

	l.rxQ = queue.NewHolder(queue.New[frames.FrameBody](segmentSize))
	return l
}

// waitForFrame waits for an incoming frame to be queued.
// it returns the next frame from the queue, or an error.
// the error is either from the context or session.doneErr.
// not meant for consumption outside of link.go.
func (l *link) waitForFrame(ctx context.Context) (frames.FrameBody, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-l.session.done:
		// session has terminated, no need to deallocate in this case
		return nil, l.session.doneErr
	case q := <-l.rxQ.Wait():
		// frame received
		fr := q.Dequeue()
		l.rxQ.Release(q)
		return *fr, nil
	}
}

// attach sends the Attach performative to establish the link with its parent session.
// this is automatically called by the new*Link constructors.
func (l *link) attach(ctx context.Context, beforeAttach func(*frames.PerformAttach), afterAttach func(*frames.PerformAttach)) error {
	if err := l.session.allocateHandle(l); err != nil {
		return err
	}

	attach := &frames.PerformAttach{
		Name:               l.key.name,
		Handle:             l.handle,
		ReceiverSettleMode: l.receiverSettleMode,
		SenderSettleMode:   l.senderSettleMode,
		MaxMessageSize:     l.maxMessageSize,
		Source:             l.source,
		Target:             l.target,
		Properties:         l.properties,
	}

	// link-specific configuration of the attach frame
	beforeAttach(attach)

	_ = l.session.txFrame(attach, nil)

	// wait for response
	fr, err := l.waitForFrame(ctx)
	if err != nil {
		l.session.deallocateHandle(l)
		return err
	}

	resp, ok := fr.(*frames.PerformAttach)
	if !ok {
		debug.Log(1, "RX (link): unexpected attach response frame %T", fr)
		if err := l.session.conn.Close(); err != nil {
			return err
		}
		return &ConnError{inner: fmt.Errorf("unexpected attach response: %#v", fr)}
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
		fr, err := l.waitForFrame(ctx)
		if err != nil {
			l.session.deallocateHandle(l)
			return err
		}

		detach, ok := fr.(*frames.PerformDetach)
		if !ok {
			if err := l.session.conn.Close(); err != nil {
				return err
			}
			return &ConnError{inner: fmt.Errorf("unexpected frame while waiting for detach: %#v", fr)}
		}

		// send return detach
		fr = &frames.PerformDetach{
			Handle: l.handle,
			Closed: true,
		}
		_ = l.session.txFrame(fr, nil)

		if detach.Error == nil {
			return fmt.Errorf("received detach with no error specified")
		}
		return detach.Error
	}

	if l.maxMessageSize == 0 || resp.MaxMessageSize < l.maxMessageSize {
		l.maxMessageSize = resp.MaxMessageSize
	}

	// link-specific configuration post attach
	afterAttach(resp)

	if err := l.setSettleModes(resp); err != nil {
		// close the link as there's a mismatch on requested/supported settlement modes
		dr := &frames.PerformDetach{
			Handle: l.handle,
			Closed: true,
		}
		_ = l.session.txFrame(dr, nil)
		return err
	}

	return nil
}

// setSettleModes sets the settlement modes based on the resp frames.PerformAttach.
//
// If a settlement mode has been explicitly set locally and it was not honored by the
// server an error is returned.
func (l *link) setSettleModes(resp *frames.PerformAttach) error {
	var (
		localRecvSettle = receiverSettleModeValue(l.receiverSettleMode)
		respRecvSettle  = receiverSettleModeValue(resp.ReceiverSettleMode)
	)
	if l.receiverSettleMode != nil && localRecvSettle != respRecvSettle {
		return fmt.Errorf("amqp: receiver settlement mode %q requested, received %q from server", l.receiverSettleMode, &respRecvSettle)
	}
	l.receiverSettleMode = &respRecvSettle

	var (
		localSendSettle = senderSettleModeValue(l.senderSettleMode)
		respSendSettle  = senderSettleModeValue(resp.SenderSettleMode)
	)
	if l.senderSettleMode != nil && localSendSettle != respSendSettle {
		return fmt.Errorf("amqp: sender settlement mode %q requested, received %q from server", l.senderSettleMode, &respSendSettle)
	}
	l.senderSettleMode = &respSendSettle

	return nil
}

// muxHandleFrame processes fr based on type.
func (l *link) muxHandleFrame(fr frames.FrameBody) error {
	switch fr := fr.(type) {
	case *frames.PerformDetach:
		if !fr.Closed {
			l.closeWithError(ErrCondNotImplemented, fmt.Sprintf("non-closing detach not supported: %+v", fr))
			return nil
		}

		// there are two possibilities:
		// - this is the ack to a client-side Close()
		// - the peer is closing the link so we must ack

		if l.closeInProgress {
			// if the client-side close was initiated due to an error (l.closeWithError)
			// then l.doneErr will already be set. in this case, return that error instead
			// of an empty LinkError which indicates a clean client-side close.
			if l.doneErr != nil {
				return l.doneErr
			}
			return &LinkError{}
		}

		dr := &frames.PerformDetach{
			Handle: l.handle,
			Closed: true,
		}
		_ = l.session.txFrame(dr, nil)
		return &LinkError{RemoteErr: fr.Error}

	default:
		debug.Log(1, "RX (link): unexpected frame: %s", fr)
		l.closeWithError(ErrCondInternalError, fmt.Sprintf("link received unexpected frame %T", fr))
		return nil
	}
}

// Close closes the Sender and AMQP link.
func (l *link) closeLink(ctx context.Context) error {
	var ctxErr error
	l.closeOnce.Do(func() {
		close(l.close)
		select {
		case <-l.done:
			l.closeErr = l.doneErr
		case <-ctx.Done():
			close(l.forceClose)

			// notify the caller that the close timed out/was cancelled
			ctxErr = ctx.Err()

			// record that the link was forcibly closed.
			// subsequent calls to closeLink() will return this
			l.closeErr = &LinkError{inner: errLinkForciblyClosed}
		}
	})

	if ctxErr != nil {
		return ctxErr
	}

	var linkErr *LinkError
	if errors.As(l.closeErr, &linkErr) && linkErr.RemoteErr == nil && linkErr.inner == nil {
		// an empty LinkError means the link was cleanly closed by the caller
		return nil
	}
	return l.closeErr
}

// closeWithError initiates closing the link with the specified AMQP error.
// the mux must continue to run until the ack'ing detach is received.
// l.doneErr is populated with a &LinkError{} containing an inner error constructed from the specified values
//   - cnd is the AMQP error condition
//   - desc is the error description
func (l *link) closeWithError(cnd ErrCond, desc string) {
	amqpErr := &Error{Condition: cnd, Description: desc}
	if l.closeInProgress {
		debug.Log(3, "TX (link) close error already pending, discarding %v", amqpErr)
		return
	}

	dr := &frames.PerformDetach{
		Handle: l.handle,
		Closed: true,
		Error:  amqpErr,
	}
	l.closeInProgress = true
	l.doneErr = &LinkError{inner: fmt.Errorf("%s: %s", cnd, desc)}
	_ = l.session.txFrame(dr, nil)
}

var errLinkForciblyClosed = errors.New("the link was forcibly closed")
