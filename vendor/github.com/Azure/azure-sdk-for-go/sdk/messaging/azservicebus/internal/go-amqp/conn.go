// Copyright (C) 2017 Kale Blankenship
// Portions Copyright (c) Microsoft Corporation
package amqp

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/go-amqp/internal/bitmap"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/go-amqp/internal/buffer"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/go-amqp/internal/encoding"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/go-amqp/internal/frames"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/go-amqp/internal/log"
)

// Default connection options
const (
	defaultIdleTimeout  = 1 * time.Minute
	defaultMaxFrameSize = 65536
	defaultMaxSessions  = 65536
)

// ConnOptions contains the optional settings for configuring an AMQP connection.
type ConnOptions struct {
	// ContainerID sets the container-id to use when opening the connection.
	//
	// A container ID will be randomly generated if this option is not used.
	ContainerID string

	// HostName sets the hostname sent in the AMQP
	// Open frame and TLS ServerName (if not otherwise set).
	HostName string

	// IdleTimeout specifies the maximum period in milliseconds between
	// receiving frames from the peer.
	//
	// Specify a value less than zero to disable idle timeout.
	//
	// Default: 1 minute.
	IdleTimeout time.Duration

	// MaxFrameSize sets the maximum frame size that
	// the connection will accept.
	//
	// Must be 512 or greater.
	//
	// Default: 512.
	MaxFrameSize uint32

	// MaxSessions sets the maximum number of channels.
	// The value must be greater than zero.
	//
	// Default: 65535.
	MaxSessions uint16

	// Properties sets an entry in the connection properties map sent to the server.
	Properties map[string]interface{}

	// SASLType contains the specified SASL authentication mechanism.
	SASLType SASLType

	// Timeout configures how long to wait for the
	// server during connection establishment.
	//
	// Once the connection has been established, IdleTimeout
	// applies. If duration is zero, no timeout will be applied.
	//
	// Default: 0.
	Timeout time.Duration

	// TLSConfig sets the tls.Config to be used during
	// TLS negotiation.
	//
	// This option is for advanced usage, in most scenarios
	// providing a URL scheme of "amqps://" is sufficient.
	TLSConfig *tls.Config

	// test hook
	dialer dialer
}

// used to abstract the underlying dialer for testing purposes
type dialer interface {
	NetDialerDial(c *conn, host, port string) error
	TLSDialWithDialer(c *conn, host, port string) error
}

// conn is an AMQP connection.
// only exported fields and methods are part of public surface area,
// all others are considered to be internal implementation details.
type conn struct {
	net            net.Conn      // underlying connection
	connectTimeout time.Duration // time to wait for reads/writes during conn establishment
	dialer         dialer        // used for testing purposes, it allows faking dialing TCP/TLS endpoints

	// TLS
	tlsNegotiation bool        // negotiate TLS
	tlsComplete    bool        // TLS negotiation complete
	tlsConfig      *tls.Config // TLS config, default used if nil (ServerName set to Client.hostname)

	// SASL
	saslHandlers map[encoding.Symbol]stateFunc // map of supported handlers keyed by SASL mechanism, SASL not negotiated if nil
	saslComplete bool                          // SASL negotiation complete; internal *except* for SASL auth methods

	// local settings
	maxFrameSize uint32                          // max frame size to accept
	channelMax   uint16                          // maximum number of channels to allow
	hostname     string                          // hostname of remote server (set explicitly or parsed from URL)
	idleTimeout  time.Duration                   // maximum period between receiving frames
	properties   map[encoding.Symbol]interface{} // additional properties sent upon connection open
	containerID  string                          // set explicitly or randomly generated

	// peer settings
	peerIdleTimeout  time.Duration // maximum period between sending frames
	PeerMaxFrameSize uint32        // maximum frame size peer will accept

	// conn state
	errMu sync.Mutex    // mux holds errMu from start until shutdown completes; operations are sequential before mux is started
	err   error         // error to be returned to client
	Done  chan struct{} // indicates the connection is done

	// mux
	NewSession   chan newSessionResp // new Sessions are requested from mux by reading off this channel
	DelSession   chan *Session       // session completion is indicated to mux by sending the Session on this channel
	connErr      chan error          // connReader/Writer notifications of an error
	closeMux     chan struct{}       // indicates that the mux should stop
	closeMuxOnce sync.Once

	// connReader
	rxProto       chan protoHeader  // protoHeaders received by connReader
	rxFrame       chan frames.Frame // AMQP frames received by connReader
	rxDone        chan struct{}
	connReaderRun chan func() // functions to be run by conn reader (set deadline on conn to run)

	// connWriter
	txFrame chan frames.Frame // AMQP frames to be sent by connWriter
	txBuf   buffer.Buffer     // buffer for marshaling frames before transmitting
	txDone  chan struct{}
}

type newSessionResp struct {
	session *Session
	err     error
}

// implements the dialer interface
type defaultDialer struct{}

func (defaultDialer) NetDialerDial(c *conn, host, port string) (err error) {
	dialer := &net.Dialer{Timeout: c.connectTimeout}
	c.net, err = dialer.Dial("tcp", net.JoinHostPort(host, port))
	return
}

func (defaultDialer) TLSDialWithDialer(c *conn, host, port string) (err error) {
	dialer := &net.Dialer{Timeout: c.connectTimeout}
	c.net, err = tls.DialWithDialer(dialer, "tcp", net.JoinHostPort(host, port), c.tlsConfig)
	return
}

func dialConn(addr string, opts *ConnOptions) (*conn, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	host, port := u.Hostname(), u.Port()
	if port == "" {
		port = "5672"
		if u.Scheme == "amqps" || u.Scheme == "amqp+ssl" {
			port = "5671"
		}
	}

	var cp ConnOptions
	if opts != nil {
		cp = *opts
	}

	// prepend SASL credentials when the user/pass segment is not empty
	if u.User != nil {
		pass, _ := u.User.Password()
		cp.SASLType = SASLTypePlain(u.User.Username(), pass)
	}

	if cp.HostName == "" {
		cp.HostName = host
	}

	c, err := newConn(nil, &cp)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "amqp", "":
		err = c.dialer.NetDialerDial(c, host, port)
	case "amqps", "amqp+ssl":
		c.initTLSConfig()
		c.tlsNegotiation = false
		err = c.dialer.TLSDialWithDialer(c, host, port)
	default:
		err = fmt.Errorf("unsupported scheme %q", u.Scheme)
	}

	if err != nil {
		return nil, err
	}
	return c, nil
}

func newConn(netConn net.Conn, opts *ConnOptions) (*conn, error) {
	c := &conn{
		dialer:           defaultDialer{},
		net:              netConn,
		maxFrameSize:     defaultMaxFrameSize,
		PeerMaxFrameSize: defaultMaxFrameSize,
		channelMax:       defaultMaxSessions - 1, // -1 because channel-max starts at zero
		idleTimeout:      defaultIdleTimeout,
		containerID:      randString(40),
		Done:             make(chan struct{}),
		connErr:          make(chan error, 2), // buffered to ensure connReader/Writer won't leak
		closeMux:         make(chan struct{}),
		rxProto:          make(chan protoHeader),
		rxFrame:          make(chan frames.Frame),
		rxDone:           make(chan struct{}),
		connReaderRun:    make(chan func(), 1), // buffered to allow queueing function before interrupt
		NewSession:       make(chan newSessionResp),
		DelSession:       make(chan *Session),
		txFrame:          make(chan frames.Frame),
		txDone:           make(chan struct{}),
	}

	// apply options
	if opts == nil {
		opts = &ConnOptions{}
	}

	if opts.ContainerID != "" {
		c.containerID = opts.ContainerID
	}
	if opts.HostName != "" {
		c.hostname = opts.HostName
	}
	if opts.IdleTimeout > 0 {
		c.idleTimeout = opts.IdleTimeout
	} else if opts.IdleTimeout < 0 {
		c.idleTimeout = 0
	}
	if opts.MaxFrameSize > 0 && opts.MaxFrameSize < 512 {
		return nil, fmt.Errorf("invalid MaxFrameSize value %d", opts.MaxFrameSize)
	} else if opts.MaxFrameSize > 512 {
		c.maxFrameSize = opts.MaxFrameSize
	}
	if opts.MaxSessions > 0 {
		c.channelMax = opts.MaxSessions
	}
	if opts.SASLType != nil {
		if err := opts.SASLType(c); err != nil {
			return nil, err
		}
	}
	if opts.Timeout > 0 {
		c.connectTimeout = opts.Timeout
	}
	if opts.Properties != nil {
		c.properties = make(map[encoding.Symbol]interface{})
		for key, val := range opts.Properties {
			c.properties[encoding.Symbol(key)] = val
		}
	}
	if opts.TLSConfig != nil {
		c.tlsConfig = opts.TLSConfig.Clone()
	}
	if opts.dialer != nil {
		c.dialer = opts.dialer
	}

	return c, nil
}

func (c *conn) initTLSConfig() {
	// create a new config if not already set
	if c.tlsConfig == nil {
		c.tlsConfig = new(tls.Config)
	}

	// TLS config must have ServerName or InsecureSkipVerify set
	if c.tlsConfig.ServerName == "" && !c.tlsConfig.InsecureSkipVerify {
		c.tlsConfig.ServerName = c.hostname
	}
}

// Start establishes the connection and begins multiplexing network IO.
// It is an error to call Start() on a connection that's been closed.
func (c *conn) Start() error {
	// start reader
	go c.connReader()

	// run connection establishment state machine
	for state := c.negotiateProto; state != nil; {
		var err error
		state, err = state()
		// check if err occurred
		if err != nil {
			close(c.txDone) // close here since connWriter hasn't been started yet
			_ = c.Close()
			return err
		}
	}

	// start multiplexor and writer
	go c.mux()
	go c.connWriter()

	return nil
}

// Close closes the connection.
func (c *conn) Close() error {
	c.closeMuxOnce.Do(func() { close(c.closeMux) })
	err := c.Err()
	var connErr *ConnectionError
	if errors.As(err, &connErr) && connErr.inner == nil {
		// an empty ConnectionError means the connection was closed by the caller
		// or as requested by the peer and no error was provided in the close frame.
		return nil
	}
	return err
}

// close should only be called by conn.mux.
func (c *conn) close() {
	close(c.Done) // notify goroutines and blocked functions to exit

	// wait for writing to stop, allows it to send the final close frame
	<-c.txDone

	// reading from connErr in mux can race with closeMux, causing
	// a pending conn read/write error to be lost.  now that the
	// mux has exited, drain any pending error.
	select {
	case err := <-c.connErr:
		c.err = err
	default:
		// no pending read/write error
	}

	err := c.net.Close()
	switch {
	// conn.err already set
	// TODO: err info is lost, log it?
	case c.err != nil:

	// conn.err not set and c.net.Close() returned a non-nil error
	case err != nil:
		c.err = err

	// no errors
	default:
	}

	// check rxDone after closing net, otherwise may block
	// for up to c.idleTimeout
	<-c.rxDone
}

// Err returns the connection's error state after it's been closed.
// Calling this on an open connection will block until the connection is closed.
func (c *conn) Err() error {
	c.errMu.Lock()
	defer c.errMu.Unlock()
	return &ConnectionError{inner: c.err}
}

// mux is started in it's own goroutine after initial connection establishment.
// It handles muxing of sessions, keepalives, and connection errors.
func (c *conn) mux() {
	var (
		// allocated channels
		channels = bitmap.New(uint32(c.channelMax))

		// create the next session to allocate
		// note that channel always start at 0, and 0 is special and can't be deleted
		nextChannel, _ = channels.Next()
		nextSession    = newSessionResp{session: newSession(c, uint16(nextChannel))}

		// map channels to sessions
		sessionsByChannel       = make(map[uint16]*Session)
		sessionsByRemoteChannel = make(map[uint16]*Session)
	)

	// hold the errMu lock until error or done
	c.errMu.Lock()
	defer c.errMu.Unlock()
	defer c.close() // defer order is important. c.errMu unlock indicates that connection is finally complete

	for {
		// check if last loop returned an error
		if c.err != nil {
			return
		}

		select {
		// error from connReader
		case c.err = <-c.connErr:

		// new frame from connReader
		case fr := <-c.rxFrame:
			var (
				session *Session
				ok      bool
			)

			switch body := fr.Body.(type) {
			// Server initiated close.
			case *frames.PerformClose:
				if body.Error != nil {
					c.err = body.Error
				}
				return

			// RemoteChannel should be used when frame is Begin
			case *frames.PerformBegin:
				if body.RemoteChannel == nil {
					// since we only support remotely-initiated sessions, this is an error
					// TODO: it would be ideal to not have this kill the connection
					c.err = fmt.Errorf("%T: nil RemoteChannel", fr.Body)
					break
				}
				session, ok = sessionsByChannel[*body.RemoteChannel]
				if !ok {
					c.err = fmt.Errorf("unexpected remote channel number %d, expected %d", *body.RemoteChannel, nextChannel)
					break
				}

				session.remoteChannel = fr.Channel
				sessionsByRemoteChannel[fr.Channel] = session

			case *frames.PerformEnd:
				session, ok = sessionsByRemoteChannel[fr.Channel]
				if !ok {
					c.err = fmt.Errorf("%T: didn't find channel %d in sessionsByRemoteChannel (PerformEnd)", fr.Body, fr.Channel)
					break
				}
				// we MUST remove the remote channel from our map as soon as we receive
				// the ack (i.e. before passing it on to the session mux) on the session
				// ending since the numbers are recycled.
				delete(sessionsByRemoteChannel, fr.Channel)

			default:
				// pass on performative to the correct session
				session, ok = sessionsByRemoteChannel[fr.Channel]
				if !ok {
					c.err = fmt.Errorf("%T: didn't find channel %d in sessionsByRemoteChannel", fr.Body, fr.Channel)
				}
			}

			if !ok {
				continue
			}

			select {
			case session.rx <- fr:
			case <-c.closeMux:
				return
			}

		// new session request
		//
		// Continually try to send the next session on the channel,
		// then add it to the sessions map. This allows us to control ID
		// allocation and prevents the need to have shared map. Since new
		// sessions are far less frequent than frames being sent to sessions,
		// this avoids the lock/unlock for session lookup.
		case c.NewSession <- nextSession:
			if nextSession.err != nil {
				continue
			}

			// save session into map
			ch := nextSession.session.channel
			sessionsByChannel[ch] = nextSession.session

			// get next available channel
			next, ok := channels.Next()
			if !ok {
				nextSession = newSessionResp{err: fmt.Errorf("reached connection channel max (%d)", c.channelMax)}
				continue
			}

			// create the next session to send
			nextSession = newSessionResp{session: newSession(c, uint16(next))}

		// session deletion
		case s := <-c.DelSession:
			delete(sessionsByChannel, s.channel)
			channels.Remove(uint32(s.channel))

		// connection is complete
		case <-c.closeMux:
			return
		}
	}
}

// connReader reads from the net.Conn, decodes frames, and passes them
// up via the conn.rxFrame and conn.rxProto channels.
func (c *conn) connReader() {
	defer close(c.rxDone)

	buf := &buffer.Buffer{}

	var (
		negotiating     = true        // true during conn establishment, check for protoHeaders
		currentHeader   frames.Header // keep track of the current header, for frames split across multiple TCP packets
		frameInProgress bool          // true if in the middle of receiving data for currentHeader
	)

	for {
		switch {
		// Cheaply reuse free buffer space when fully read.
		case buf.Len() == 0:
			buf.Reset()

		// Prevent excessive/unbounded growth by shifting data to beginning of buffer.
		case int64(buf.Size()) > int64(c.maxFrameSize):
			buf.Reclaim()
		}

		// need to read more if buf doesn't contain the complete frame
		// or there's not enough in buf to parse the header
		if frameInProgress || buf.Len() < frames.HeaderSize {
			if c.idleTimeout > 0 {
				_ = c.net.SetReadDeadline(time.Now().Add(c.idleTimeout))
			}
			err := buf.ReadFromOnce(c.net)
			if err != nil {
				log.Debug(1, "connReader error: %v", err)
				select {
				// check if error was due to close in progress
				case <-c.Done:
					return

				// if there is a pending connReaderRun function, execute it
				case f := <-c.connReaderRun:
					f()
					continue

				// send error to mux and return
				default:
					c.connErr <- err
					return
				}
			}
		}

		// read more if buf doesn't contain enough to parse the header
		if buf.Len() < frames.HeaderSize {
			continue
		}

		// during negotiation, check for proto frames
		if negotiating && bytes.Equal(buf.Bytes()[:4], []byte{'A', 'M', 'Q', 'P'}) {
			p, err := parseProtoHeader(buf)
			if err != nil {
				c.connErr <- err
				return
			}

			// negotiation is complete once an AMQP proto frame is received
			if p.ProtoID == protoAMQP {
				negotiating = false
			}

			// send proto header
			select {
			case <-c.Done:
				return
			case c.rxProto <- p:
			}

			continue
		}

		// parse the header if a frame isn't in progress
		if !frameInProgress {
			var err error
			currentHeader, err = frames.ParseHeader(buf)
			if err != nil {
				c.connErr <- err
				return
			}
			frameInProgress = true
		}

		// check size is reasonable
		if currentHeader.Size > math.MaxInt32 { // make max size configurable
			c.connErr <- errors.New("payload too large")
			return
		}

		bodySize := int64(currentHeader.Size - frames.HeaderSize)

		// the full frame has been received
		if int64(buf.Len()) < bodySize {
			continue
		}
		frameInProgress = false

		// check if body is empty (keepalive)
		if bodySize == 0 {
			continue
		}

		// parse the frame
		b, ok := buf.Next(bodySize)
		if !ok {
			c.connErr <- fmt.Errorf("buffer EOF; requested bytes: %d, actual size: %d", bodySize, buf.Len())
			return
		}

		parsedBody, err := frames.ParseBody(buffer.New(b))
		if err != nil {
			c.connErr <- err
			return
		}

		// send to mux
		select {
		case <-c.Done:
			return
		case c.rxFrame <- frames.Frame{Channel: currentHeader.Channel, Body: parsedBody}:
		}
	}
}

func (c *conn) connWriter() {
	defer close(c.txDone)

	// disable write timeout
	if c.connectTimeout != 0 {
		c.connectTimeout = 0
		_ = c.net.SetWriteDeadline(time.Time{})
	}

	var (
		// keepalives are sent at a rate of 1/2 idle timeout
		keepaliveInterval = c.peerIdleTimeout / 2
		// 0 disables keepalives
		keepalivesEnabled = keepaliveInterval > 0
		// set if enable, nil if not; nil channels block forever
		keepalive <-chan time.Time
	)

	if keepalivesEnabled {
		ticker := time.NewTicker(keepaliveInterval)
		defer ticker.Stop()
		keepalive = ticker.C
	}

	var err error
	for {
		if err != nil {
			log.Debug(1, "connWriter error: %v", err)
			c.connErr <- err
			return
		}

		select {
		// frame write request
		case fr := <-c.txFrame:
			err = c.writeFrame(fr)
			if err == nil && fr.Done != nil {
				close(fr.Done)
			}

		// keepalive timer
		case <-keepalive:
			log.Debug(3, "sending keep-alive frame")
			_, err = c.net.Write(keepaliveFrame)
			// It would be slightly more efficient in terms of network
			// resources to reset the timer each time a frame is sent.
			// However, keepalives are small (8 bytes) and the interval
			// is usually on the order of minutes. It does not seem
			// worth it to add extra operations in the write path to
			// avoid. (To properly reset a timer it needs to be stopped,
			// possibly drained, then reset.)

		// connection complete
		case <-c.Done:
			// send close
			cls := &frames.PerformClose{}
			log.Debug(1, "TX (connWriter): %s", cls)
			_ = c.writeFrame(frames.Frame{
				Type: frameTypeAMQP,
				Body: cls,
			})
			return
		}
	}
}

// writeFrame writes a frame to the network.
// used externally by SASL only.
func (c *conn) writeFrame(fr frames.Frame) error {
	if c.connectTimeout != 0 {
		_ = c.net.SetWriteDeadline(time.Now().Add(c.connectTimeout))
	}

	// writeFrame into txBuf
	c.txBuf.Reset()
	err := writeFrame(&c.txBuf, fr)
	if err != nil {
		return err
	}

	// validate the frame isn't exceeding peer's max frame size
	requiredFrameSize := c.txBuf.Len()
	if uint64(requiredFrameSize) > uint64(c.PeerMaxFrameSize) {
		return fmt.Errorf("%T frame size %d larger than peer's max frame size %d", fr, requiredFrameSize, c.PeerMaxFrameSize)
	}

	// write to network
	_, err = c.net.Write(c.txBuf.Bytes())
	return err
}

// writeProtoHeader writes an AMQP protocol header to the
// network
func (c *conn) writeProtoHeader(pID protoID) error {
	if c.connectTimeout != 0 {
		_ = c.net.SetWriteDeadline(time.Now().Add(c.connectTimeout))
	}
	_, err := c.net.Write([]byte{'A', 'M', 'Q', 'P', byte(pID), 1, 0, 0})
	return err
}

// keepaliveFrame is an AMQP frame with no body, used for keepalives
var keepaliveFrame = []byte{0x00, 0x00, 0x00, 0x08, 0x02, 0x00, 0x00, 0x00}

// SendFrame is used by sessions and links to send frames across the network.
func (c *conn) SendFrame(fr frames.Frame) error {
	select {
	case c.txFrame <- fr:
		return nil
	case <-c.Done:
		return c.Err()
	}
}

// stateFunc is a state in a state machine.
//
// The state is advanced by returning the next state.
// The state machine concludes when nil is returned.
type stateFunc func() (stateFunc, error)

// negotiateProto determines which proto to negotiate next.
// used externally by SASL only.
func (c *conn) negotiateProto() (stateFunc, error) {
	// in the order each must be negotiated
	switch {
	case c.tlsNegotiation && !c.tlsComplete:
		return c.exchangeProtoHeader(protoTLS)
	case c.saslHandlers != nil && !c.saslComplete:
		return c.exchangeProtoHeader(protoSASL)
	default:
		return c.exchangeProtoHeader(protoAMQP)
	}
}

type protoID uint8

// protocol IDs received in protoHeaders
const (
	protoAMQP protoID = 0x0
	protoTLS  protoID = 0x2
	protoSASL protoID = 0x3
)

// exchangeProtoHeader performs the round trip exchange of protocol
// headers, validation, and returns the protoID specific next state.
func (c *conn) exchangeProtoHeader(pID protoID) (stateFunc, error) {
	// write the proto header
	if err := c.writeProtoHeader(pID); err != nil {
		return nil, err
	}

	// read response header
	p, err := c.readProtoHeader()
	if err != nil {
		return nil, err
	}

	if pID != p.ProtoID {
		return nil, fmt.Errorf("unexpected protocol header %#00x, expected %#00x", p.ProtoID, pID)
	}

	// go to the proto specific state
	switch pID {
	case protoAMQP:
		return c.openAMQP, nil
	case protoTLS:
		return c.startTLS, nil
	case protoSASL:
		return c.negotiateSASL, nil
	default:
		return nil, fmt.Errorf("unknown protocol ID %#02x", p.ProtoID)
	}
}

// readProtoHeader reads a protocol header packet from c.rxProto.
func (c *conn) readProtoHeader() (protoHeader, error) {
	var deadline <-chan time.Time
	if c.connectTimeout != 0 {
		deadline = time.After(c.connectTimeout)
	}
	var p protoHeader
	select {
	case p = <-c.rxProto:
		return p, nil
	case err := <-c.connErr:
		return p, err
	case fr := <-c.rxFrame:
		return p, fmt.Errorf("readProtoHeader: unexpected frame %#v", fr)
	case <-deadline:
		return p, errors.New("amqp: timeout waiting for response")
	}
}

// startTLS wraps the conn with TLS and returns to Client.negotiateProto
func (c *conn) startTLS() (stateFunc, error) {
	c.initTLSConfig()

	// buffered so connReaderRun won't block
	done := make(chan error, 1)

	// this function will be executed by connReader
	c.connReaderRun <- func() {
		defer close(done)
		_ = c.net.SetReadDeadline(time.Time{}) // clear timeout

		// wrap existing net.Conn and perform TLS handshake
		tlsConn := tls.Client(c.net, c.tlsConfig)
		if c.connectTimeout != 0 {
			_ = tlsConn.SetWriteDeadline(time.Now().Add(c.connectTimeout))
		}
		done <- tlsConn.Handshake()
		// TODO: return?

		// swap net.Conn
		c.net = tlsConn
		c.tlsComplete = true
	}

	// set deadline to interrupt connReader
	_ = c.net.SetReadDeadline(time.Time{}.Add(1))

	if err := <-done; err != nil {
		return nil, err
	}

	// go to next protocol
	return c.negotiateProto, nil
}

// openAMQP round trips the AMQP open performative
func (c *conn) openAMQP() (stateFunc, error) {
	// send open frame
	open := &frames.PerformOpen{
		ContainerID:  c.containerID,
		Hostname:     c.hostname,
		MaxFrameSize: c.maxFrameSize,
		ChannelMax:   c.channelMax,
		IdleTimeout:  c.idleTimeout / 2, // per spec, advertise half our idle timeout
		Properties:   c.properties,
	}
	log.Debug(1, "TX (openAMQP): %s", open)
	err := c.writeFrame(frames.Frame{
		Type:    frameTypeAMQP,
		Body:    open,
		Channel: 0,
	})
	if err != nil {
		return nil, err
	}

	// get the response
	fr, err := c.readFrame()
	if err != nil {
		return nil, err
	}
	o, ok := fr.Body.(*frames.PerformOpen)
	if !ok {
		return nil, fmt.Errorf("openAMQP: unexpected frame type %T", fr.Body)
	}
	log.Debug(1, "RX (openAMQP): %s", o)

	// update peer settings
	if o.MaxFrameSize > 0 {
		c.PeerMaxFrameSize = o.MaxFrameSize
	}
	if o.IdleTimeout > 0 {
		// TODO: reject very small idle timeouts
		c.peerIdleTimeout = o.IdleTimeout
	}
	if o.ChannelMax < c.channelMax {
		c.channelMax = o.ChannelMax
	}

	// connection established, exit state machine
	return nil, nil
}

// negotiateSASL returns the SASL handler for the first matched
// mechanism specified by the server
func (c *conn) negotiateSASL() (stateFunc, error) {
	// read mechanisms frame
	fr, err := c.readFrame()
	if err != nil {
		return nil, err
	}
	sm, ok := fr.Body.(*frames.SASLMechanisms)
	if !ok {
		return nil, fmt.Errorf("negotiateSASL: unexpected frame type %T", fr.Body)
	}
	log.Debug(1, "RX (negotiateSASL): %s", sm)

	// return first match in c.saslHandlers based on order received
	for _, mech := range sm.Mechanisms {
		if state, ok := c.saslHandlers[mech]; ok {
			return state, nil
		}
	}

	// no match
	return nil, fmt.Errorf("no supported auth mechanism (%v)", sm.Mechanisms) // TODO: send "auth not supported" frame?
}

// saslOutcome processes the SASL outcome frame and return Client.negotiateProto
// on success.
//
// SASL handlers return this stateFunc when the mechanism specific negotiation
// has completed.
// used externally by SASL only.
func (c *conn) saslOutcome() (stateFunc, error) {
	// read outcome frame
	fr, err := c.readFrame()
	if err != nil {
		return nil, err
	}
	so, ok := fr.Body.(*frames.SASLOutcome)
	if !ok {
		return nil, fmt.Errorf("saslOutcome: unexpected frame type %T", fr.Body)
	}
	log.Debug(1, "RX (saslOutcome): %s", so)

	// check if auth succeeded
	if so.Code != encoding.CodeSASLOK {
		return nil, fmt.Errorf("SASL PLAIN auth failed with code %#00x: %s", so.Code, so.AdditionalData) // implement Stringer for so.Code
	}

	// return to c.negotiateProto
	c.saslComplete = true
	return c.negotiateProto, nil
}

// readFrame is used during connection establishment to read a single frame.
//
// After setup, conn.mux handles incoming frames.
// used externally by SASL only.
func (c *conn) readFrame() (frames.Frame, error) {
	var deadline <-chan time.Time
	if c.connectTimeout != 0 {
		deadline = time.After(c.connectTimeout)
	}

	var fr frames.Frame
	select {
	case fr = <-c.rxFrame:
		return fr, nil
	case err := <-c.connErr:
		return fr, err
	case p := <-c.rxProto:
		return fr, fmt.Errorf("unexpected protocol header %#v", p)
	case <-deadline:
		return fr, errors.New("amqp: timeout waiting for response")
	}
}

type protoHeader struct {
	ProtoID  protoID
	Major    uint8
	Minor    uint8
	Revision uint8
}

// parseProtoHeader reads the proto header from r and returns the results
//
// An error is returned if the protocol is not "AMQP" or if the version is not 1.0.0.
func parseProtoHeader(r *buffer.Buffer) (protoHeader, error) {
	const protoHeaderSize = 8
	buf, ok := r.Next(protoHeaderSize)
	if !ok {
		return protoHeader{}, errors.New("invalid protoHeader")
	}
	_ = buf[7]

	if !bytes.Equal(buf[:4], []byte{'A', 'M', 'Q', 'P'}) {
		return protoHeader{}, fmt.Errorf("unexpected protocol %q", buf[:4])
	}

	p := protoHeader{
		ProtoID:  protoID(buf[4]),
		Major:    buf[5],
		Minor:    buf[6],
		Revision: buf[7],
	}

	if p.Major != 1 || p.Minor != 0 || p.Revision != 0 {
		return p, fmt.Errorf("unexpected protocol version %d.%d.%d", p.Major, p.Minor, p.Revision)
	}
	return p, nil
}

// writesFrame encodes fr into buf.
// split out from conn.WriteFrame for testing purposes.
func writeFrame(buf *buffer.Buffer, fr frames.Frame) error {
	// write header
	buf.Append([]byte{
		0, 0, 0, 0, // size, overwrite later
		2,       // doff, see frameHeader.DataOffset comment
		fr.Type, // frame type
	})
	buf.AppendUint16(fr.Channel) // channel

	// write AMQP frame body
	err := encoding.Marshal(buf, fr.Body)
	if err != nil {
		return err
	}

	// validate size
	if uint(buf.Len()) > math.MaxUint32 {
		return errors.New("frame too large")
	}

	// retrieve raw bytes
	bufBytes := buf.Bytes()

	// write correct size
	binary.BigEndian.PutUint32(bufBytes, uint32(len(bufBytes)))
	return nil
}
