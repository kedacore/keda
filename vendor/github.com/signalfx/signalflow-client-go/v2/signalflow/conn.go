// Copyright Splunk Inc.
// SPDX-License-Identifier: Apache-2.0

package signalflow

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"time"

	"github.com/gorilla/websocket"
)

// How long to wait between connections in case of a bad connection.
var reconnectDelay = 5 * time.Second

type wsConn struct {
	StreamURL *url.URL

	OutgoingTextMsgs   chan *outgoingMessage
	IncomingTextMsgs   chan []byte
	IncomingBinaryMsgs chan []byte
	ConnectedCh        chan struct{}

	ConnectTimeout         time.Duration
	ReadTimeout            time.Duration
	WriteTimeout           time.Duration
	OnError                OnErrorFunc
	PostDisconnectCallback func()
	PostConnectMessage     func() []byte
}

type outgoingMessage struct {
	bytes    []byte
	resultCh chan error
}

// Run keeps the connection alive and puts all incoming messages into a channel
// as needed.
func (c *wsConn) Run(ctx context.Context) {
	var conn *websocket.Conn
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()

	for {
		if conn != nil {
			conn.Close()
			time.Sleep(reconnectDelay)
		}
		// This will get run on before the first connection as well.
		if c.PostDisconnectCallback != nil {
			c.PostDisconnectCallback()
		}

		if ctx.Err() != nil {
			return
		}

		dialCtx, cancel := context.WithTimeout(ctx, c.ConnectTimeout)
		var err error
		conn, err = c.connect(dialCtx)
		cancel()
		if err != nil {
			c.sendErrIfWanted(fmt.Errorf("Error connecting to SignalFlow websocket: %w", err))
			continue
		}

		err = c.postConnect(conn)
		if err != nil {
			c.sendErrIfWanted(fmt.Errorf("Error setting up SignalFlow websocket: %w", err))
			continue
		}

		err = c.readAndWriteMessages(conn)
		if err == nil {
			return
		}
		c.sendErrIfWanted(fmt.Errorf("Error in SignalFlow websocket: %w", err))
	}
}

type messageWithType struct {
	bytes   []byte
	msgType int
}

func (c *wsConn) readAndWriteMessages(conn *websocket.Conn) error {
	readMessageCh := make(chan messageWithType)
	readErrCh := make(chan error)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for {
			bytes, typ, err := readNextMessage(conn, c.ReadTimeout)
			if err != nil {
				select {
				case readErrCh <- err:
				case <-ctx.Done():
				}
				return
			}
			readMessageCh <- messageWithType{
				bytes:   bytes,
				msgType: typ,
			}
		}
	}()

	for {
		select {
		case msg, ok := <-readMessageCh:
			if !ok {
				return nil
			}
			if msg.msgType == websocket.TextMessage {
				c.IncomingTextMsgs <- msg.bytes
			} else {
				c.IncomingBinaryMsgs <- msg.bytes
			}
		case err := <-readErrCh:
			return err
		case msg, ok := <-c.OutgoingTextMsgs:
			if !ok {
				return nil
			}
			err := c.writeMessage(conn, msg.bytes)
			msg.resultCh <- err
			if err != nil {
				return err
			}
		}
	}
}

func (c *wsConn) sendErrIfWanted(err error) {
	if c.OnError != nil {
		c.OnError(err)
	}
}

func (c *wsConn) Close() {
	close(c.IncomingTextMsgs)
	close(c.IncomingBinaryMsgs)
}

func (c *wsConn) connect(ctx context.Context) (*websocket.Conn, error) {
	connectURL := *c.StreamURL
	connectURL.Path = path.Join(c.StreamURL.Path, "connect")
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, connectURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("could not connect Signalflow websocket: %w", err)
	}
	return conn, nil
}

func (c *wsConn) postConnect(conn *websocket.Conn) error {
	if c.PostConnectMessage != nil {
		msg := c.PostConnectMessage()
		if msg != nil {
			return c.writeMessage(conn, msg)
		}
	}
	return nil
}

func readNextMessage(conn *websocket.Conn, timeout time.Duration) (data []byte, msgType int, err error) {
	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return nil, 0, fmt.Errorf("could not set read timeout in SignalFlow client: %w", err)
	}

	typ, bytes, err := conn.ReadMessage()
	if err != nil {
		return nil, 0, err
	}
	return bytes, typ, nil
}

func (c *wsConn) writeMessage(conn *websocket.Conn, msgBytes []byte) error {
	err := conn.SetWriteDeadline(time.Now().Add(c.WriteTimeout))
	if err != nil {
		return fmt.Errorf("could not set write timeout for SignalFlow request: %w", err)
	}

	err = conn.WriteMessage(websocket.TextMessage, msgBytes)
	if err != nil {
		return err
	}
	return nil
}
