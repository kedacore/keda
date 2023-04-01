package amqp

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"math"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/Azure/go-amqp/internal/bitmap"
	"github.com/Azure/go-amqp/internal/buffer"
	"github.com/Azure/go-amqp/internal/debug"
	"github.com/Azure/go-amqp/internal/encoding"
	"github.com/Azure/go-amqp/internal/frames"
	"github.com/Azure/go-amqp/internal/shared"
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

	// IdleTimeout specifies the maximum period between
	// receiving frames from the peer.
	//
	// Specify a value less than zero to disable idle timeout.
	//
	// Default: 1 minute (60000000000).
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
	Properties map[string]any

	// SASLType contains the specified SASL authentication mechanism.
	SASLType SASLType

	// TLSConfig sets the tls.Config to be used during
	// TLS negotiation.
	//
	// This option is for advanced usage, in most scenarios
	// providing a URL scheme of "amqps://" is sufficient.
	TLSConfig *tls.Config

	// test hook
	dialer dialer
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
func Dial(ctx context.Context, addr string, opts *ConnOptions) (*Conn, error) {
	deadline, _ := ctx.Deadline()
	c, err := dialConn(deadline, addr, opts)
	if err != nil {
		return nil, err
	}
	err = c.start(deadline)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// NewConn establishes a new AMQP client connection over conn.
// opts: pass nil to accept the default values.
func NewConn(ctx context.Context, conn net.Conn, opts *ConnOptions) (*Conn, error) {
	c, err := newConn(conn, opts)
	if err != nil {
		return nil, err
	}
	deadline, _ := ctx.Deadline()
	err = c.start(deadline)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// Conn is an AMQP connection.
type Conn struct {
	net    net.Conn // underlying connection
	dialer dialer   // used for testing purposes, it allows faking dialing TCP/TLS endpoints

	// TLS
	tlsNegotiation bool        // negotiate TLS
	tlsComplete    bool        // TLS negotiation complete
	tlsConfig      *tls.Config // TLS config, default used if nil (ServerName set to Client.hostname)

	// SASL
	saslHandlers map[encoding.Symbol]stateFunc // map of supported handlers keyed by SASL mechanism, SASL not negotiated if nil
	saslComplete bool                          // SASL negotiation complete; internal *except* for SASL auth methods

	// local settings
	maxFrameSize uint32                  // max frame size to accept
	channelMax   uint16                  // maximum number of channels to allow
	hostname     string                  // hostname of remote server (set explicitly or parsed from URL)
	idleTimeout  time.Duration           // maximum period between receiving frames
	properties   map[encoding.Symbol]any // additional properties sent upon connection open
	containerID  string                  // set explicitly or randomly generated

	// peer settings
	peerIdleTimeout  time.Duration // maximum period between sending frames
	peerMaxFrameSize uint32        // maximum frame size peer will accept

	// conn state
	done    chan struct{} // indicates the connection has terminated
	doneErr error         // contains the error state returned from Close(); DO NOT TOUCH outside of conn.go until done has been closed!

	// connReader and connWriter management
	rxtxExit  chan struct{} // signals connReader and connWriter to exit
	closeOnce sync.Once     // ensures that close() is only called once

	// session tracking
	channels            *bitmap.Bitmap
	sessionsByChannel   map[uint16]*Session
	sessionsByChannelMu sync.RWMutex

	// connReader
	rxBuf  buffer.Buffer // incoming bytes buffer
	rxDone chan struct{} // closed when connReader exits
	rxErr  error         // contains last error reading from c.net; DO NOT TOUCH outside of connReader until rxDone has been closed!

	// connWriter
	txFrame chan frames.Frame // AMQP frames to be sent by connWriter
	txBuf   buffer.Buffer     // buffer for marshaling frames before transmitting
	txDone  chan struct{}     // closed when connWriter exits
	txErr   error             // contains last error writing to c.net; DO NOT TOUCH outside of connWriter until txDone has been closed!
}

// used to abstract the underlying dialer for testing purposes
type dialer interface {
	NetDialerDial(deadline time.Time, c *Conn, host, port string) error
	TLSDialWithDialer(deadline time.Time, c *Conn, host, port string) error
}

// implements the dialer interface
type defaultDialer struct{}

func (defaultDialer) NetDialerDial(deadline time.Time, c *Conn, host, port string) (err error) {
	dialer := &net.Dialer{Deadline: deadline}
	c.net, err = dialer.Dial("tcp", net.JoinHostPort(host, port))
	return
}

func (defaultDialer) TLSDialWithDialer(deadline time.Time, c *Conn, host, port string) (err error) {
	dialer := &net.Dialer{Deadline: deadline}
	c.net, err = tls.DialWithDialer(dialer, "tcp", net.JoinHostPort(host, port), c.tlsConfig)
	return
}

func dialConn(deadline time.Time, addr string, opts *ConnOptions) (*Conn, error) {
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
		err = c.dialer.NetDialerDial(deadline, c, host, port)
	case "amqps", "amqp+ssl":
		c.initTLSConfig()
		c.tlsNegotiation = false
		err = c.dialer.TLSDialWithDialer(deadline, c, host, port)
	default:
		err = fmt.Errorf("unsupported scheme %q", u.Scheme)
	}

	if err != nil {
		return nil, err
	}
	return c, nil
}

func newConn(netConn net.Conn, opts *ConnOptions) (*Conn, error) {
	c := &Conn{
		dialer:            defaultDialer{},
		net:               netConn,
		maxFrameSize:      defaultMaxFrameSize,
		peerMaxFrameSize:  defaultMaxFrameSize,
		channelMax:        defaultMaxSessions - 1, // -1 because channel-max starts at zero
		idleTimeout:       defaultIdleTimeout,
		containerID:       shared.RandString(40),
		done:              make(chan struct{}),
		rxtxExit:          make(chan struct{}),
		rxDone:            make(chan struct{}),
		txFrame:           make(chan frames.Frame),
		txDone:            make(chan struct{}),
		sessionsByChannel: map[uint16]*Session{},
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
	if opts.Properties != nil {
		c.properties = make(map[encoding.Symbol]any)
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

func (c *Conn) initTLSConfig() {
	// create a new config if not already set
	if c.tlsConfig == nil {
		c.tlsConfig = new(tls.Config)
	}

	// TLS config must have ServerName or InsecureSkipVerify set
	if c.tlsConfig.ServerName == "" && !c.tlsConfig.InsecureSkipVerify {
		c.tlsConfig.ServerName = c.hostname
	}
}

// start establishes the connection and begins multiplexing network IO.
// It is an error to call Start() on a connection that's been closed.
func (c *Conn) start(deadline time.Time) error {
	// set connection establishment deadline
	_ = c.net.SetDeadline(deadline)

	// run connection establishment state machine
	for state := c.negotiateProto; state != nil; {
		var err error
		state, err = state()
		// check if err occurred
		if err != nil {
			close(c.txDone) // close here since connWriter hasn't been started yet
			close(c.rxDone)
			_ = c.Close()
			return err
		}
	}

	// remove connection establishment deadline
	_ = c.net.SetDeadline(time.Time{})

	// we can't create the channel bitmap until the connection has been established.
	// this is because our peer can tell us the max channels they support.
	c.channels = bitmap.New(uint32(c.channelMax))

	go c.connWriter()
	go c.connReader()

	return nil
}

// Close closes the connection.
func (c *Conn) Close() error {
	c.close()
	var connErr *ConnError
	if errors.As(c.doneErr, &connErr) && connErr.RemoteErr == nil && connErr.inner == nil {
		// an empty ConnectionError means the connection was closed by the caller
		return nil
	}

	// there was an error during shut-down or connReader/connWriter
	// experienced a terminal error
	return c.doneErr
}

// close is called once, either from Close() or when connReader/connWriter exits
func (c *Conn) close() {
	c.closeOnce.Do(func() {
		defer close(c.done)

		close(c.rxtxExit)

		// wait for writing to stop, allows it to send the final close frame
		<-c.txDone

		closeErr := c.net.Close()

		// check rxDone after closing net, otherwise may block
		// for up to c.idleTimeout
		<-c.rxDone

		if errors.Is(c.rxErr, net.ErrClosed) {
			// this is the expected error when the connection is closed, swallow it
			c.rxErr = nil
		}

		if c.txErr == nil && c.rxErr == nil && closeErr == nil {
			// if there are no errors, it means user initiated close() and we shut down cleanly
			c.doneErr = &ConnError{}
		} else if amqpErr, ok := c.rxErr.(*Error); ok {
			// we experienced a peer-initiated close that contained an Error.  return it
			c.doneErr = &ConnError{RemoteErr: amqpErr}
		} else if c.txErr != nil {
			c.doneErr = &ConnError{inner: c.txErr}
		} else if c.rxErr != nil {
			c.doneErr = &ConnError{inner: c.rxErr}
		} else {
			c.doneErr = &ConnError{inner: closeErr}
		}
	})
}

// NewSession starts a new session on the connection.
//   - ctx controls waiting for the peer to acknowledge the session
//   - opts contains optional values, pass nil to accept the defaults
//
// If the context's deadline expires or is cancelled before the operation
// completes, the application can be left in an unknown state, potentially
// resulting in connection errors.
func (c *Conn) NewSession(ctx context.Context, opts *SessionOptions) (*Session, error) {
	session, err := c.newSession(opts)
	if err != nil {
		return nil, err
	}

	if err := session.begin(ctx); err != nil {
		c.deleteSession(session)
		return nil, err
	}

	return session, nil
}

func (c *Conn) newSession(opts *SessionOptions) (*Session, error) {
	c.sessionsByChannelMu.Lock()
	defer c.sessionsByChannelMu.Unlock()

	// create the next session to allocate
	// note that channel always start at 0
	channel, ok := c.channels.Next()
	if !ok {
		return nil, fmt.Errorf("reached connection channel max (%d)", c.channelMax)
	}
	session := newSession(c, uint16(channel), opts)
	c.sessionsByChannel[session.channel] = session

	return session, nil
}

func (c *Conn) deleteSession(s *Session) {
	c.sessionsByChannelMu.Lock()
	defer c.sessionsByChannelMu.Unlock()

	delete(c.sessionsByChannel, s.channel)
	c.channels.Remove(uint32(s.channel))
}

// connReader reads from the net.Conn, decodes frames, and either handles
// them here as appropriate or sends them to the session.rx channel.
func (c *Conn) connReader() {
	defer func() {
		close(c.rxDone)
		c.close()
	}()

	var sessionsByRemoteChannel = make(map[uint16]*Session)
	var err error
	for {
		if err != nil {
			debug.Log(1, "RX (connReader): terminal error: %v", err)
			c.rxErr = err
			return
		}

		var fr frames.Frame
		fr, err = c.readFrame()
		if err != nil {
			continue
		}

		debug.Log(1, "RX (connReader): %s", fr)

		var (
			session *Session
			ok      bool
		)

		switch body := fr.Body.(type) {
		// Server initiated close.
		case *frames.PerformClose:
			// connWriter will send the close performative ack on its way out.
			// it's a SHOULD though, not a MUST.
			if body.Error == nil {
				return
			}
			err = body.Error
			continue

		// RemoteChannel should be used when frame is Begin
		case *frames.PerformBegin:
			if body.RemoteChannel == nil {
				// since we only support remotely-initiated sessions, this is an error
				// TODO: it would be ideal to not have this kill the connection
				err = fmt.Errorf("%T: nil RemoteChannel", fr.Body)
				continue
			}
			c.sessionsByChannelMu.RLock()
			session, ok = c.sessionsByChannel[*body.RemoteChannel]
			c.sessionsByChannelMu.RUnlock()
			if !ok {
				// this can happen if NewSession() exits due to the context expiring/cancelled
				// before the begin ack is received.
				err = fmt.Errorf("unexpected remote channel number %d", *body.RemoteChannel)
				continue
			}

			session.remoteChannel = fr.Channel
			sessionsByRemoteChannel[fr.Channel] = session

		case *frames.PerformEnd:
			session, ok = sessionsByRemoteChannel[fr.Channel]
			if !ok {
				err = fmt.Errorf("%T: didn't find channel %d in sessionsByRemoteChannel (PerformEnd)", fr.Body, fr.Channel)
				continue
			}
			// we MUST remove the remote channel from our map as soon as we receive
			// the ack (i.e. before passing it on to the session mux) on the session
			// ending since the numbers are recycled.
			delete(sessionsByRemoteChannel, fr.Channel)

		default:
			// pass on performative to the correct session
			session, ok = sessionsByRemoteChannel[fr.Channel]
			if !ok {
				err = fmt.Errorf("%T: didn't find channel %d in sessionsByRemoteChannel", fr.Body, fr.Channel)
				continue
			}
		}

		q := session.rxQ.Acquire()
		q.Enqueue(fr.Body)
		session.rxQ.Release(q)
		debug.Log(2, "RX (connReader): mux frame to session: %s", fr)
	}
}

// readFrame reads a complete frame from c.net.
// it assumes that any read deadline has already been applied.
// used externally by SASL only.
func (c *Conn) readFrame() (frames.Frame, error) {
	switch {
	// Cheaply reuse free buffer space when fully read.
	case c.rxBuf.Len() == 0:
		c.rxBuf.Reset()

	// Prevent excessive/unbounded growth by shifting data to beginning of buffer.
	case int64(c.rxBuf.Size()) > int64(c.maxFrameSize):
		c.rxBuf.Reclaim()
	}

	var (
		currentHeader   frames.Header // keep track of the current header, for frames split across multiple TCP packets
		frameInProgress bool          // true if in the middle of receiving data for currentHeader
	)

	for {
		// need to read more if buf doesn't contain the complete frame
		// or there's not enough in buf to parse the header
		if frameInProgress || c.rxBuf.Len() < frames.HeaderSize {
			// we MUST reset the idle timeout before each read from net.Conn
			if c.idleTimeout > 0 {
				_ = c.net.SetReadDeadline(time.Now().Add(c.idleTimeout))
			}
			err := c.rxBuf.ReadFromOnce(c.net)
			if err != nil {
				return frames.Frame{}, err
			}
		}

		// read more if buf doesn't contain enough to parse the header
		if c.rxBuf.Len() < frames.HeaderSize {
			continue
		}

		// parse the header if a frame isn't in progress
		if !frameInProgress {
			var err error
			currentHeader, err = frames.ParseHeader(&c.rxBuf)
			if err != nil {
				return frames.Frame{}, err
			}
			frameInProgress = true
		}

		// check size is reasonable
		if currentHeader.Size > math.MaxInt32 { // make max size configurable
			return frames.Frame{}, errors.New("payload too large")
		}

		bodySize := int64(currentHeader.Size - frames.HeaderSize)

		// the full frame hasn't been received, keep reading
		if int64(c.rxBuf.Len()) < bodySize {
			continue
		}
		frameInProgress = false

		// check if body is empty (keepalive)
		if bodySize == 0 {
			debug.Log(3, "RX (connReader): received keep-alive frame")
			continue
		}

		// parse the frame
		b, ok := c.rxBuf.Next(bodySize)
		if !ok {
			return frames.Frame{}, fmt.Errorf("buffer EOF; requested bytes: %d, actual size: %d", bodySize, c.rxBuf.Len())
		}

		parsedBody, err := frames.ParseBody(buffer.New(b))
		if err != nil {
			return frames.Frame{}, err
		}

		return frames.Frame{Channel: currentHeader.Channel, Body: parsedBody}, nil
	}
}

func (c *Conn) connWriter() {
	defer func() {
		close(c.txDone)
		c.close()
	}()

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
			debug.Log(1, "TX (connWriter): terminal error: %v", err)
			c.txErr = err
			return
		}

		select {
		// frame write request
		case fr := <-c.txFrame:
			debug.Log(1, "TX (connWriter): %s", fr)
			err = c.writeFrame(fr)
			if err == nil && fr.Done != nil {
				close(fr.Done)
			}

		// keepalive timer
		case <-keepalive:
			debug.Log(3, "TX (connWriter): sending keep-alive frame")
			_, err = c.net.Write(keepaliveFrame)
			// It would be slightly more efficient in terms of network
			// resources to reset the timer each time a frame is sent.
			// However, keepalives are small (8 bytes) and the interval
			// is usually on the order of minutes. It does not seem
			// worth it to add extra operations in the write path to
			// avoid. (To properly reset a timer it needs to be stopped,
			// possibly drained, then reset.)

		// connection complete
		case <-c.rxtxExit:
			// send close performative.  note that the spec says we
			// SHOULD wait for the ack but we don't HAVE to, in order
			// to be resilient to bad actors etc.  so we just send
			// the close performative and exit.
			fr := frames.Frame{
				Type: frames.TypeAMQP,
				Body: &frames.PerformClose{},
			}
			debug.Log(1, "TX (connWriter): %s", fr)
			c.txErr = c.writeFrame(fr)
			return
		}
	}
}

// writeFrame writes a frame to the network.
// used externally by SASL only.
func (c *Conn) writeFrame(fr frames.Frame) error {
	// writeFrame into txBuf
	c.txBuf.Reset()
	err := frames.Write(&c.txBuf, fr)
	if err != nil {
		return err
	}

	// validate the frame isn't exceeding peer's max frame size
	requiredFrameSize := c.txBuf.Len()
	if uint64(requiredFrameSize) > uint64(c.peerMaxFrameSize) {
		return fmt.Errorf("%T frame size %d larger than peer's max frame size %d", fr, requiredFrameSize, c.peerMaxFrameSize)
	}

	// write to network
	n, err := c.net.Write(c.txBuf.Bytes())
	if l := c.txBuf.Len(); n > 0 && n < l && err != nil {
		debug.Log(1, "TX (writeFrame): wrote %d bytes less than len %d: %v", n, l, err)
	}
	return err
}

// writeProtoHeader writes an AMQP protocol header to the
// network
func (c *Conn) writeProtoHeader(pID protoID) error {
	_, err := c.net.Write([]byte{'A', 'M', 'Q', 'P', byte(pID), 1, 0, 0})
	return err
}

// keepaliveFrame is an AMQP frame with no body, used for keepalives
var keepaliveFrame = []byte{0x00, 0x00, 0x00, 0x08, 0x02, 0x00, 0x00, 0x00}

// SendFrame is used by sessions and links to send frames across the network.
func (c *Conn) sendFrame(fr frames.Frame) error {
	select {
	case c.txFrame <- fr:
		debug.Log(2, "TX (Conn): mux frame to connWriter: %s", fr)
		return nil
	case <-c.done:
		return c.doneErr
	}
}

// stateFunc is a state in a state machine.
//
// The state is advanced by returning the next state.
// The state machine concludes when nil is returned.
type stateFunc func() (stateFunc, error)

// negotiateProto determines which proto to negotiate next.
// used externally by SASL only.
func (c *Conn) negotiateProto() (stateFunc, error) {
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
func (c *Conn) exchangeProtoHeader(pID protoID) (stateFunc, error) {
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
func (c *Conn) readProtoHeader() (protoHeader, error) {
	const protoHeaderSize = 8

	// only read from the network once our buffer has been exhausted.
	// TODO: this preserves existing behavior as some tests rely on this
	// implementation detail (it lets you replay a stream of bytes). we
	// might want to consider removing this and fixing the tests as the
	// protocol doesn't actually work this way.
	if c.rxBuf.Len() == 0 {
		for {
			err := c.rxBuf.ReadFromOnce(c.net)
			if err != nil {
				return protoHeader{}, err
			}

			// read more if buf doesn't contain enough to parse the header
			if c.rxBuf.Len() >= protoHeaderSize {
				break
			}
		}
	}

	buf, ok := c.rxBuf.Next(protoHeaderSize)
	if !ok {
		return protoHeader{}, errors.New("invalid protoHeader")
	}
	// bounds check hint to compiler; see golang.org/issue/14808
	_ = buf[protoHeaderSize-1]

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
		return protoHeader{}, fmt.Errorf("unexpected protocol version %d.%d.%d", p.Major, p.Minor, p.Revision)
	}

	return p, nil
}

// startTLS wraps the conn with TLS and returns to Client.negotiateProto
func (c *Conn) startTLS() (stateFunc, error) {
	c.initTLSConfig()

	_ = c.net.SetReadDeadline(time.Time{}) // clear timeout

	// wrap existing net.Conn and perform TLS handshake
	tlsConn := tls.Client(c.net, c.tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		return nil, err
	}

	// swap net.Conn
	c.net = tlsConn
	c.tlsComplete = true

	// go to next protocol
	return c.negotiateProto, nil
}

// openAMQP round trips the AMQP open performative
func (c *Conn) openAMQP() (stateFunc, error) {
	// send open frame
	open := &frames.PerformOpen{
		ContainerID:  c.containerID,
		Hostname:     c.hostname,
		MaxFrameSize: c.maxFrameSize,
		ChannelMax:   c.channelMax,
		IdleTimeout:  c.idleTimeout / 2, // per spec, advertise half our idle timeout
		Properties:   c.properties,
	}
	fr := frames.Frame{
		Type:    frames.TypeAMQP,
		Body:    open,
		Channel: 0,
	}
	debug.Log(1, "TX (openAMQP): %s", fr)
	err := c.writeFrame(fr)
	if err != nil {
		return nil, err
	}

	// get the response
	fr, err = c.readSingleFrame()
	if err != nil {
		return nil, err
	}
	debug.Log(1, "RX (openAMQP): %s", fr)
	o, ok := fr.Body.(*frames.PerformOpen)
	if !ok {
		return nil, fmt.Errorf("openAMQP: unexpected frame type %T", fr.Body)
	}

	// update peer settings
	if o.MaxFrameSize > 0 {
		c.peerMaxFrameSize = o.MaxFrameSize
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
func (c *Conn) negotiateSASL() (stateFunc, error) {
	// read mechanisms frame
	fr, err := c.readSingleFrame()
	if err != nil {
		return nil, err
	}
	debug.Log(1, "RX (negotiateSASL): %s", fr)
	sm, ok := fr.Body.(*frames.SASLMechanisms)
	if !ok {
		return nil, fmt.Errorf("negotiateSASL: unexpected frame type %T", fr.Body)
	}

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
func (c *Conn) saslOutcome() (stateFunc, error) {
	// read outcome frame
	fr, err := c.readSingleFrame()
	if err != nil {
		return nil, err
	}
	debug.Log(1, "RX (saslOutcome): %s", fr)
	so, ok := fr.Body.(*frames.SASLOutcome)
	if !ok {
		return nil, fmt.Errorf("saslOutcome: unexpected frame type %T", fr.Body)
	}

	// check if auth succeeded
	if so.Code != encoding.CodeSASLOK {
		return nil, fmt.Errorf("SASL PLAIN auth failed with code %#00x: %s", so.Code, so.AdditionalData) // implement Stringer for so.Code
	}

	// return to c.negotiateProto
	c.saslComplete = true
	return c.negotiateProto, nil
}

// readSingleFrame is used during connection establishment to read a single frame.
//
// After setup, conn.connReader handles incoming frames.
func (c *Conn) readSingleFrame() (frames.Frame, error) {
	fr, err := c.readFrame()
	if err != nil {
		return frames.Frame{}, err
	}

	return fr, nil
}

type protoHeader struct {
	ProtoID  protoID
	Major    uint8
	Minor    uint8
	Revision uint8
}
