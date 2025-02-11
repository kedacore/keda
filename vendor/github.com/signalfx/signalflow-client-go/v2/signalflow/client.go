// Copyright Splunk Inc.
// SPDX-License-Identifier: Apache-2.0

package signalflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/signalfx/signalflow-client-go/v2/signalflow/messages"
)

// Client for SignalFlow via websockets (SSE is not currently supported).
type Client struct {
	// Access token for the org
	token                  string
	userAgent              string
	defaultMetadataTimeout time.Duration
	nextChannelNum         int64
	conn                   *wsConn
	readTimeout            time.Duration
	// How long to wait for writes to the websocket to finish
	writeTimeout   time.Duration
	streamURL      *url.URL
	onError        OnErrorFunc
	channelsByName map[string]chan messages.Message

	// These are the lower-level WebSocket level channels for byte messages
	outgoingTextMsgs   chan *outgoingMessage
	incomingTextMsgs   chan []byte
	incomingBinaryMsgs chan []byte
	connectedCh        chan struct{}

	isClosed atomic.Bool
	sync.Mutex
	cancel context.CancelFunc
}

type clientMessageRequest struct {
	msg      interface{}
	resultCh chan error
}

// ClientParam is the common type of configuration functions for the SignalFlow client
type ClientParam func(*Client) error

// StreamURL lets you set the full URL to the stream endpoint, including the
// path.
func StreamURL(streamEndpoint string) ClientParam {
	return func(c *Client) error {
		var err error
		c.streamURL, err = url.Parse(streamEndpoint)
		return err
	}
}

// StreamURLForRealm can be used to configure the websocket url for a specific
// SignalFx realm.
func StreamURLForRealm(realm string) ClientParam {
	return func(c *Client) error {
		var err error
		c.streamURL, err = url.Parse(fmt.Sprintf("wss://stream.%s.signalfx.com/v2/signalflow", realm))
		return err
	}
}

// AccessToken can be used to provide a SignalFx organization access token or
// user access token to the SignalFlow client.
func AccessToken(token string) ClientParam {
	return func(c *Client) error {
		c.token = token
		return nil
	}
}

// UserAgent allows setting the `userAgent` field when authenticating to
// SignalFlow.  This can be useful for accounting how many jobs are started
// from each client.
func UserAgent(userAgent string) ClientParam {
	return func(c *Client) error {
		c.userAgent = userAgent
		return nil
	}
}

// ReadTimeout sets the duration to wait between messages that come on the
// websocket.  If the resolution of the job is very low, this should be
// increased.
func ReadTimeout(timeout time.Duration) ClientParam {
	return func(c *Client) error {
		if timeout <= 0 {
			return errors.New("ReadTimeout cannot be <= 0")
		}
		c.readTimeout = timeout
		return nil
	}
}

// WriteTimeout sets the maximum duration to wait to send a single message when
// writing messages to the SignalFlow server over the WebSocket connection.
func WriteTimeout(timeout time.Duration) ClientParam {
	return func(c *Client) error {
		if timeout <= 0 {
			return errors.New("WriteTimeout cannot be <= 0")
		}
		c.writeTimeout = timeout
		return nil
	}
}

type OnErrorFunc func(err error)

func OnError(f OnErrorFunc) ClientParam {
	return func(c *Client) error {
		c.onError = f
		return nil
	}
}

// NewClient makes a new SignalFlow client that will immediately try and
// connect to the SignalFlow backend.
func NewClient(options ...ClientParam) (*Client, error) {
	c := &Client{
		streamURL: &url.URL{
			Scheme: "wss",
			Host:   "stream.us0.signalfx.com",
			Path:   "/v2/signalflow",
		},
		readTimeout:    1 * time.Minute,
		writeTimeout:   5 * time.Second,
		channelsByName: make(map[string]chan messages.Message),

		outgoingTextMsgs:   make(chan *outgoingMessage),
		incomingTextMsgs:   make(chan []byte),
		incomingBinaryMsgs: make(chan []byte),
		connectedCh:        make(chan struct{}),
	}

	for i := range options {
		if err := options[i](c); err != nil {
			return nil, err
		}
	}

	c.conn = &wsConn{
		StreamURL:          c.streamURL,
		OutgoingTextMsgs:   c.outgoingTextMsgs,
		IncomingTextMsgs:   c.incomingTextMsgs,
		IncomingBinaryMsgs: c.incomingBinaryMsgs,
		ConnectedCh:        c.connectedCh,
		ConnectTimeout:     10 * time.Second,
		ReadTimeout:        c.readTimeout,
		WriteTimeout:       c.writeTimeout,
		OnError:            c.onError,
		PostDisconnectCallback: func() {
			c.closeRegisteredChannels()
		},
		PostConnectMessage: func() []byte {
			bytes, err := c.makeAuthRequest()
			if err != nil {
				c.sendErrIfWanted(fmt.Errorf("failed to send auth: %w", err))
				return nil
			}
			return bytes
		},
	}

	var ctx context.Context
	ctx, c.cancel = context.WithCancel(context.Background())

	go c.conn.Run(ctx)
	go c.run(ctx)

	return c, nil
}

func (c *Client) newUniqueChannelName() string {
	name := fmt.Sprintf("ch-%d", atomic.AddInt64(&c.nextChannelNum, 1))
	return name
}

func (c *Client) sendErrIfWanted(err error) {
	if c.onError != nil {
		c.onError(err)
	}
}

// Writes all messages from a single goroutine since that is required by
// websocket library.
func (c *Client) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-c.incomingTextMsgs:
			err := c.handleMessage(msg, websocket.TextMessage)
			if err != nil {
				c.sendErrIfWanted(fmt.Errorf("error handling SignalFlow text message: %w", err))
			}
		case msg := <-c.incomingBinaryMsgs:
			err := c.handleMessage(msg, websocket.BinaryMessage)
			if err != nil {
				c.sendErrIfWanted(fmt.Errorf("error handling SignalFlow binary message: %w", err))
			}
		}
	}
}

func (c *Client) sendMessage(ctx context.Context, message interface{}) error {
	msgBytes, err := c.serializeMessage(message)
	if err != nil {
		return err
	}

	resultCh := make(chan error, 1)
	select {
	case c.outgoingTextMsgs <- &outgoingMessage{
		bytes:    msgBytes,
		resultCh: resultCh,
	}:
		return <-resultCh
	case <-ctx.Done():
		close(resultCh)
		return ctx.Err()
	}
}

func (c *Client) serializeMessage(message interface{}) ([]byte, error) {
	msgBytes, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("could not marshal SignalFlow request: %w", err)
	}
	return msgBytes, nil
}

func (c *Client) handleMessage(msgBytes []byte, msgTyp int) error {
	message, err := messages.ParseMessage(msgBytes, msgTyp == websocket.TextMessage)
	if err != nil {
		return fmt.Errorf("could not parse SignalFlow message: %w", err)
	}

	if cm, ok := message.(messages.ChannelMessage); ok {
		channelName := cm.Channel()
		c.Lock()
		channel, ok := c.channelsByName[channelName]
		if !ok {
			// The channel should have existed before, but now doesn't,
			// probably because it was closed.
			return nil
		} else if channelName == "" {
			c.acceptMessage(message)
			return nil
		}
		channel <- message
		c.Unlock()
	} else {
		return c.acceptMessage(message)
	}
	return nil
}

// acceptMessages accepts non-channel specific messages.  The only one that I
// know of is the authenticated response.
func (c *Client) acceptMessage(message messages.Message) error {
	if _, ok := message.(*messages.AuthenticatedMessage); ok {
		return nil
	} else if msg, ok := message.(*messages.BaseJSONMessage); ok {
		data := msg.RawData()
		if data != nil && data["event"] == "KEEP_ALIVE" {
			// Ignore keep alive messages
			return nil
		}
	}

	return fmt.Errorf("unknown SignalFlow message received: %v", message)
}

// Sends the authenticate message but does not wait for a response.
func (c *Client) makeAuthRequest() ([]byte, error) {
	return c.serializeMessage(&AuthRequest{
		Token:     c.token,
		UserAgent: c.userAgent,
	})
}

// Execute a SignalFlow job and return a channel upon which informational
// messages and data will flow.
// See https://dev.splunk.com/observability/docs/signalflow/messages/websocket_request_messages#Execute-a-computation
func (c *Client) Execute(ctx context.Context, req *ExecuteRequest) (*Computation, error) {
	if req.Channel == "" {
		req.Channel = c.newUniqueChannelName()
	}

	err := c.sendMessage(ctx, req)
	if err != nil {
		return nil, err
	}

	return newComputation(c.registerChannel(req.Channel), req.Channel, c), nil
}

// Detach from a computation but keep it running.  See
// https://dev.splunk.com/observability/docs/signalflow/messages/websocket_request_messages#Detach-from-a-computation.
func (c *Client) Detach(ctx context.Context, req *DetachRequest) error {
	// We are assuming that the detach request will always come from the same
	// client that started it with the Execute method above, and thus the
	// connection is still active (i.e. we don't need to call ensureInitialized
	// here).  If the websocket connection does drop, all jobs started by that
	// connection get detached/stopped automatically.
	return c.sendMessage(ctx, req)
}

// Stop sends a job stop request message to the backend.  It does not wait for
// jobs to actually be stopped.
// See https://dev.splunk.com/observability/docs/signalflow/messages/websocket_request_messages#Stop-a-computation
func (c *Client) Stop(ctx context.Context, req *StopRequest) error {
	// We are assuming that the stop request will always come from the same
	// client that started it with the Execute method above, and thus the
	// connection is still active (i.e. we don't need to call ensureInitialized
	// here).  If the websocket connection does drop, all jobs started by that
	// connection get stopped automatically.
	return c.sendMessage(ctx, req)
}

func (c *Client) registerChannel(name string) chan messages.Message {
	ch := make(chan messages.Message)

	c.Lock()
	c.channelsByName[name] = ch
	c.Unlock()

	return ch
}

func (c *Client) closeRegisteredChannels() {
	c.Lock()
	for _, ch := range c.channelsByName {
		close(ch)
	}
	c.channelsByName = map[string]chan messages.Message{}
	c.Unlock()
}

// Close the client and shutdown any ongoing connections and goroutines.  The client cannot be
// reused after Close. Calling any of the client methods after Close() is undefined and will likely
// result in a panic.
func (c *Client) Close() {
	if c.isClosed.Load() {
		panic("cannot close client more than once")
	}
	c.isClosed.Store(true)

	c.cancel()
	c.closeRegisteredChannels()

DRAIN:
	for {
		select {
		case outMsg := <-c.outgoingTextMsgs:
			outMsg.resultCh <- io.EOF
		default:
			break DRAIN
		}
	}
	close(c.outgoingTextMsgs)
}
