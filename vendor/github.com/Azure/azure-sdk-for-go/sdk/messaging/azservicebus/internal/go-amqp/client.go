// Copyright (C) 2017 Kale Blankenship
// Portions Copyright (c) Microsoft Corporation
package amqp

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/go-amqp/internal/encoding"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/go-amqp/internal/frames"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/go-amqp/internal/log"
)

// Client is an AMQP client connection.
type Client struct {
	conn *conn
}

// Dial connects to an AMQP server.
//
// If the addr includes a scheme, it must be "amqp", "amqps", or "amqp+ssl".
// If no port is provided, 5672 will be used for "amqp" and 5671 for "amqps" or "amqp+ssl".
//
// If username and password information is not empty it's used as SASL PLAIN
// credentials, equal to passing ConnSASLPlain option.
//
// opts: pass nil to accept the default values.
func Dial(addr string, opts *ConnOptions) (*Client, error) {
	c, err := dialConn(addr, opts)
	if err != nil {
		return nil, err
	}
	err = c.Start()
	if err != nil {
		return nil, err
	}
	return &Client{conn: c}, nil
}

// New establishes an AMQP client connection over conn.
// opts: pass nil to accept the default values.
func New(conn net.Conn, opts *ConnOptions) (*Client, error) {
	c, err := newConn(conn, opts)
	if err != nil {
		return nil, err
	}
	err = c.Start()
	if err != nil {
		return nil, err
	}
	return &Client{conn: c}, nil
}

// Close disconnects the connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// NewSession opens a new AMQP session to the server.
// Returns ErrConnClosed if the underlying connection has been closed.
// opts: pass nil to accept the default values.
func (c *Client) NewSession(ctx context.Context, opts *SessionOptions) (*Session, error) {
	// get a session allocated by Client.mux
	var sResp newSessionResp
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.conn.Done:
		return nil, c.conn.Err()
	case sResp = <-c.conn.NewSession:
	}

	if sResp.err != nil {
		return nil, sResp.err
	}
	s := sResp.session
	s.init(opts)

	// send Begin to server
	begin := &frames.PerformBegin{
		NextOutgoingID: 0,
		IncomingWindow: s.incomingWindow,
		OutgoingWindow: s.outgoingWindow,
		HandleMax:      s.handleMax,
	}
	log.Debug(1, "TX (NewSession): %s", begin)
	_ = s.txFrame(begin, nil)

	// wait for response
	var fr frames.Frame
	select {
	case <-ctx.Done():
		// TODO: this will leak s
		return nil, ctx.Err()
	case <-c.conn.Done:
		return nil, c.conn.Err()
	case fr = <-s.rx:
	}
	log.Debug(1, "RX (NewSession): %s", fr.Body)

	begin, ok := fr.Body.(*frames.PerformBegin)
	if !ok {
		// this codepath is hard to hit (impossible?).  if the response isn't a PerformBegin and we've not
		// yet seen the remote channel number, the default clause in conn.mux will protect us from that.
		// if we have seen the remote channel number then it's likely the session.mux for that channel will
		// either swallow the frame or blow up in some other way, both causing this call to hang.
		// deallocate session on error.  we can't call
		// s.Close() as the session mux hasn't started yet.
		select {
		case <-ctx.Done():
			// TODO: this will leak s
			return nil, ctx.Err()
		case c.conn.DelSession <- s:
		}
		return nil, fmt.Errorf("unexpected begin response: %+v", fr.Body)
	}

	// start Session multiplexor
	go s.mux(begin)

	return s, nil
}

// SessionOption contains the optional settings for configuring an AMQP session.
type SessionOptions struct {
	// IncomingWindow sets the maximum number of unacknowledged
	// transfer frames the server can send.
	IncomingWindow uint32

	// OutgoingWindow sets the maximum number of unacknowledged
	// transfer frames the client can send.
	OutgoingWindow uint32

	// MaxLinks sets the maximum number of links (Senders/Receivers)
	// allowed on the session.
	//
	// Minimum: 1.
	// Default: 4294967295.
	MaxLinks uint32
}

// lockedRand provides a rand source that is safe for concurrent use.
type lockedRand struct {
	mu  sync.Mutex
	src *rand.Rand
}

func (r *lockedRand) Read(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.src.Read(p)
}

// package scoped rand source to avoid any issues with seeding
// of the global source.
var pkgRand = &lockedRand{
	src: rand.New(rand.NewSource(time.Now().UnixNano())),
}

// randBytes returns a base64 encoded string of n bytes.
func randString(n int) string {
	b := make([]byte, n)
	// from math/rand, cannot fail
	_, _ = pkgRand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// linkKey uniquely identifies a link on a connection by name and direction.
//
// A link can be identified uniquely by the ordered tuple (source-container-id, target-container-id, name)
//
// On a single connection the container ID pairs can be abbreviated
// to a boolean flag indicating the direction of the link.
type linkKey struct {
	name string
	role encoding.Role // Local role: sender/receiver
}

const maxTransferFrameHeader = 66 // determined by calcMaxTransferFrameHeader
