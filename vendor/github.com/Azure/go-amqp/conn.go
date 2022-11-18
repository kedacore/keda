package amqp

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/Azure/go-amqp/internal/bitmap"
	"github.com/Azure/go-amqp/internal/buffer"
	"github.com/Azure/go-amqp/internal/encoding"
	"github.com/Azure/go-amqp/internal/frames"
)

// Default connection options
const (
	DefaultIdleTimeout  = 1 * time.Minute
	DefaultMaxFrameSize = 65536
	DefaultMaxSessions  = 65536
)

// ConnOption is a function for configuring an AMQP connection.
type ConnOption func(*conn) error

// ConnServerHostname sets the hostname sent in the AMQP
// Open frame and TLS ServerName (if not otherwise set).
//
// This is useful when the AMQP connection will be established
// via a pre-established TLS connection as the server may not
// know which hostname the client is attempting to connect to.
func ConnServerHostname(hostname string) ConnOption {
	return func(c *conn) error {
		c.hostname = hostname
		return nil
	}
}

// ConnTLS toggles TLS negotiation.
//
// Default: false.
func ConnTLS(enable bool) ConnOption {
	return func(c *conn) error {
		c.tlsNegotiation = enable
		return nil
	}
}

// ConnTLSConfig sets the tls.Config to be used during
// TLS negotiation.
//
// This option is for advanced usage, in most scenarios
// providing a URL scheme of "amqps://" or ConnTLS(true)
// is sufficient.
func ConnTLSConfig(tc *tls.Config) ConnOption {
	return func(c *conn) error {
		c.tlsConfig = tc
		c.tlsNegotiation = true
		return nil
	}
}

// ConnIdleTimeout specifies the maximum period between receiving
// frames from the peer.
//
// Resolution is milliseconds. A value of zero indicates no timeout.
// This setting is in addition to TCP keepalives.
//
// Default: 1 minute.
func ConnIdleTimeout(d time.Duration) ConnOption {
	return func(c *conn) error {
		if d < 0 {
			return errors.New("idle timeout cannot be negative")
		}
		c.idleTimeout = d
		return nil
	}
}

// ConnMaxFrameSize sets the maximum frame size that
// the connection will accept.
//
// Must be 512 or greater.
//
// Default: 512.
func ConnMaxFrameSize(n uint32) ConnOption {
	return func(c *conn) error {
		if n < 512 {
			return errors.New("max frame size must be 512 or greater")
		}
		c.maxFrameSize = n
		return nil
	}
}

// ConnConnectTimeout configures how long to wait for the
// server during connection establishment.
//
// Once the connection has been established, ConnIdleTimeout
// applies. If duration is zero, no timeout will be applied.
//
// Default: 0.
func ConnConnectTimeout(d time.Duration) ConnOption {
	return func(c *conn) error { c.connectTimeout = d; return nil }
}

// ConnMaxSessions sets the maximum number of channels.
//
// n must be in the range 1 to 65536.
//
// Default: 65536.
func ConnMaxSessions(n int) ConnOption {
	return func(c *conn) error {
		if n < 1 {
			return errors.New("max sessions cannot be less than 1")
		}
		if n > 65536 {
			return errors.New("max sessions cannot be greater than 65536")
		}
		c.channelMax = uint16(n - 1)
		return nil
	}
}

// ConnProperty sets an entry in the connection properties map sent to the server.
//
// This option can be used multiple times.
func ConnProperty(key, value string) ConnOption {
	return func(c *conn) error {
		if key == "" {
			return errors.New("connection property key must not be empty")
		}
		if c.properties == nil {
			c.properties = make(map[encoding.Symbol]interface{})
		}
		c.properties[encoding.Symbol(key)] = value
		return nil
	}
}

// ConnContainerID sets the container-id to use when opening the connection.
//
// A container ID will be randomly generated if this option is not used.
func ConnContainerID(id string) ConnOption {
	return func(c *conn) error {
		c.containerID = id
		return nil
	}
}

// used to abstract the underlying dialer for testing purposes
type dialer interface {
	NetDialerDial(c *conn, host, port string) error
	TLSDialWithDialer(c *conn, host, port string) error
}

func connDialer(d dialer) ConnOption {
	return func(c *conn) error {
		c.dialer = d
		return nil
	}
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
	err   error         // error to be returned to client; internal *except* for SASL auth methods
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

func dialConn(addr string, opts ...ConnOption) (*conn, error) {
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

	// prepend SASL credentials when the user/pass segment is not empty
	if u.User != nil {
		pass, _ := u.User.Password()
		opts = append([]ConnOption{
			ConnSASLPlain(u.User.Username(), pass),
		}, opts...)
	}

	// append default options so user specified can overwrite
	opts = append([]ConnOption{
		connDialer(defaultDialer{}),
		ConnServerHostname(host),
	}, opts...)

	c, err := newConn(nil, opts...)
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

func newConn(netConn net.Conn, opts ...ConnOption) (*conn, error) {
	c := &conn{
		net:              netConn,
		maxFrameSize:     DefaultMaxFrameSize,
		PeerMaxFrameSize: DefaultMaxFrameSize,
		channelMax:       DefaultMaxSessions - 1, // -1 because channel-max starts at zero
		idleTimeout:      DefaultIdleTimeout,
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
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
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
		state = state()
	}

	// check if err occurred
	if c.err != nil {
		close(c.txDone) // close here since connWriter hasn't been started yet
		_ = c.Close()
		return c.err
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
	if err == ErrConnClosed {
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
	case c.err != nil:

	// conn.err not set and c.net.Close() returned a non-nil error
	case err != nil:
		c.err = err

	// no errors
	default:
		c.err = ErrConnClosed
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
	return c.err
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
				} else {
					c.err = ErrConnClosed
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
					c.err = fmt.Errorf("%T: didn't find channel %d in sessionsByRemoteChannel", fr.Body, fr.Channel)
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
				debug(1, "connReader error: %v", err)
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
			c.connErr <- io.EOF
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
			debug(3, "sending keep-alive frame")
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
			debug(1, "TX (connWriter): %s", cls)
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
type stateFunc func() stateFunc

// negotiateProto determines which proto to negotiate next.
// used externally by SASL only.
func (c *conn) negotiateProto() stateFunc {
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
func (c *conn) exchangeProtoHeader(pID protoID) stateFunc {
	// write the proto header
	c.err = c.writeProtoHeader(pID)
	if c.err != nil {
		return nil
	}

	// read response header
	p, err := c.readProtoHeader()
	if err != nil {
		c.err = err
		return nil
	}

	if pID != p.ProtoID {
		c.err = fmt.Errorf("unexpected protocol header %#00x, expected %#00x", p.ProtoID, pID)
		return nil
	}

	// go to the proto specific state
	switch pID {
	case protoAMQP:
		return c.openAMQP
	case protoTLS:
		return c.startTLS
	case protoSASL:
		return c.negotiateSASL
	default:
		c.err = fmt.Errorf("unknown protocol ID %#02x", p.ProtoID)
		return nil
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
		return p, ErrTimeout
	}
}

// startTLS wraps the conn with TLS and returns to Client.negotiateProto
func (c *conn) startTLS() stateFunc {
	c.initTLSConfig()

	done := make(chan struct{})

	// this function will be executed by connReader
	c.connReaderRun <- func() {
		_ = c.net.SetReadDeadline(time.Time{}) // clear timeout

		// wrap existing net.Conn and perform TLS handshake
		tlsConn := tls.Client(c.net, c.tlsConfig)
		if c.connectTimeout != 0 {
			_ = tlsConn.SetWriteDeadline(time.Now().Add(c.connectTimeout))
		}
		c.err = tlsConn.Handshake()

		// swap net.Conn
		c.net = tlsConn
		c.tlsComplete = true

		close(done)
	}

	// set deadline to interrupt connReader
	_ = c.net.SetReadDeadline(time.Time{}.Add(1))

	<-done

	if c.err != nil {
		return nil
	}

	// go to next protocol
	return c.negotiateProto
}

// openAMQP round trips the AMQP open performative
func (c *conn) openAMQP() stateFunc {
	// send open frame
	open := &frames.PerformOpen{
		ContainerID:  c.containerID,
		Hostname:     c.hostname,
		MaxFrameSize: c.maxFrameSize,
		ChannelMax:   c.channelMax,
		IdleTimeout:  c.idleTimeout / 2, // per spec, advertise half our idle timeout
		Properties:   c.properties,
	}
	debug(1, "TX (openAMQP): %s", open)
	c.err = c.writeFrame(frames.Frame{
		Type:    frameTypeAMQP,
		Body:    open,
		Channel: 0,
	})
	if c.err != nil {
		return nil
	}

	// get the response
	fr, err := c.readFrame()
	if err != nil {
		c.err = err
		return nil
	}
	o, ok := fr.Body.(*frames.PerformOpen)
	if !ok {
		c.err = fmt.Errorf("openAMQP: unexpected frame type %T", fr.Body)
		return nil
	}
	debug(1, "RX (openAMQP): %s", o)

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
	return nil
}

// negotiateSASL returns the SASL handler for the first matched
// mechanism specified by the server
func (c *conn) negotiateSASL() stateFunc {
	// read mechanisms frame
	fr, err := c.readFrame()
	if err != nil {
		c.err = err
		return nil
	}
	sm, ok := fr.Body.(*frames.SASLMechanisms)
	if !ok {
		c.err = fmt.Errorf("negotiateSASL: unexpected frame type %T", fr.Body)
		return nil
	}
	debug(1, "RX (negotiateSASL): %s", sm)

	// return first match in c.saslHandlers based on order received
	for _, mech := range sm.Mechanisms {
		if state, ok := c.saslHandlers[mech]; ok {
			return state
		}
	}

	// no match
	c.err = fmt.Errorf("no supported auth mechanism (%v)", sm.Mechanisms) // TODO: send "auth not supported" frame?
	return nil
}

// saslOutcome processes the SASL outcome frame and return Client.negotiateProto
// on success.
//
// SASL handlers return this stateFunc when the mechanism specific negotiation
// has completed.
// used externally by SASL only.
func (c *conn) saslOutcome() stateFunc {
	// read outcome frame
	fr, err := c.readFrame()
	if err != nil {
		c.err = err
		return nil
	}
	so, ok := fr.Body.(*frames.SASLOutcome)
	if !ok {
		c.err = fmt.Errorf("saslOutcome: unexpected frame type %T", fr.Body)
		return nil
	}
	debug(1, "RX (saslOutcome): %s", so)

	// check if auth succeeded
	if so.Code != encoding.CodeSASLOK {
		c.err = fmt.Errorf("SASL PLAIN auth failed with code %#00x: %s", so.Code, so.AdditionalData) // implement Stringer for so.Code
		return nil
	}

	// return to c.negotiateProto
	c.saslComplete = true
	return c.negotiateProto
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
		return fr, ErrTimeout
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
