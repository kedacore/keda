/*
Copyright 2019 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package websocket

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"sync"
	"time"

	"go.uber.org/zap"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/gorilla/websocket"
)

var (
	// ErrConnectionNotEstablished is returned by methods that need a connection
	// but no connection is already created.
	ErrConnectionNotEstablished = errors.New("connection has not yet been established")

	// errShuttingDown is returned internally once the shutdown signal has been sent.
	errShuttingDown = errors.New("shutdown in progress")

	// pongTimeout defines the amount of time allowed between two pongs to arrive
	// before the connection is considered broken.
	pongTimeout = 10 * time.Second
)

// RawConnection is an interface defining the methods needed
// from a websocket connection
type rawConnection interface {
	WriteMessage(messageType int, data []byte) error
	NextReader() (int, io.Reader, error)
	Close() error

	SetReadDeadline(deadline time.Time) error
	SetPongHandler(func(string) error)
}

// ManagedConnection represents a websocket connection.
type ManagedConnection struct {
	connection        rawConnection
	connectionFactory func() (rawConnection, error)

	closeChan chan struct{}
	closeOnce sync.Once

	// Used to capture asynchronous processes to be waited
	// on when shutting the connection down.
	processingWg sync.WaitGroup

	// If set, messages will be forwarded to this channel
	messageChan chan []byte

	// This mutex controls access to the connection reference
	// itself.
	connectionLock sync.RWMutex

	// Gorilla's documentation states, that one reader and
	// one writer are allowed concurrently.
	readerLock sync.Mutex
	writerLock sync.Mutex

	// Used for the exponential backoff when connecting
	connectionBackoff wait.Backoff
}

// NewDurableSendingConnection creates a new websocket connection
// that can only send messages to the endpoint it connects to.
// The connection will continuously be kept alive and reconnected
// in case of a loss of connectivity.
func NewDurableSendingConnection(target string, logger *zap.SugaredLogger) *ManagedConnection {
	return NewDurableConnection(target, nil, logger)
}

// NewDurableConnection creates a new websocket connection, that
// passes incoming messages to the given message channel. It can also
// send messages to the endpoint it connects to.
// The connection will continuously be kept alive and reconnected
// in case of a loss of connectivity.
//
// Note: The given channel needs to be drained after calling `Shutdown`
// to not cause any deadlocks. If the channel's buffer is likely to be
// filled, this needs to happen in separate goroutines, i.e.
//
// go func() {conn.Shutdown(); close(messageChan)}
// go func() {for range messageChan {}}
func NewDurableConnection(target string, messageChan chan []byte, logger *zap.SugaredLogger) *ManagedConnection {
	websocketConnectionFactory := func() (rawConnection, error) {
		dialer := &websocket.Dialer{
			HandshakeTimeout: 3 * time.Second,
		}
		conn, _, err := dialer.Dial(target, nil)
		return conn, err
	}

	c := newConnection(websocketConnectionFactory, messageChan)

	// Keep the connection alive asynchronously and reconnect on
	// connection failure.
	c.processingWg.Add(1)
	go func() {
		defer c.processingWg.Done()

		for {
			select {
			default:
				logger.Infof("Connecting to %q", target)
				if err := c.connect(); err != nil {
					logger.Errorw(fmt.Sprintf("Connecting to %q failed", target), zap.Error(err))
					continue
				}
				logger.Infof("Connected to %q", target)
				if err := c.keepalive(); err != nil {
					logger.Errorw(fmt.Sprintf("Connection to %q broke down, reconnecting...", target), zap.Error(err))
				}
				if err := c.closeConnection(); err != nil {
					logger.Errorw("Failed to close the connection after crashing", zap.Error(err))
				}
			case <-c.closeChan:
				logger.Infof("Connection to %q is being shutdown", target)
				return
			}
		}
	}()

	// Keep sending pings 3 times per pongTimeout interval.
	c.processingWg.Add(1)
	go func() {
		defer c.processingWg.Done()

		ticker := time.NewTicker(pongTimeout / 3)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := c.write(websocket.PingMessage, []byte{}); err != nil {
					logger.Errorw("Failed to send ping message", zap.Error(err))
				}
			case <-c.closeChan:
				return
			}
		}
	}()

	return c
}

// newConnection creates a new connection primitive.
func newConnection(connFactory func() (rawConnection, error), messageChan chan []byte) *ManagedConnection {
	conn := &ManagedConnection{
		connectionFactory: connFactory,
		closeChan:         make(chan struct{}),
		messageChan:       messageChan,
		connectionBackoff: wait.Backoff{
			Duration: 100 * time.Millisecond,
			Factor:   1.3,
			Steps:    20,
			Jitter:   0.5,
		},
	}

	return conn
}

// connect tries to establish a websocket connection.
func (c *ManagedConnection) connect() error {
	var err error
	wait.ExponentialBackoff(c.connectionBackoff, func() (bool, error) {
		select {
		default:
			var conn rawConnection
			conn, err = c.connectionFactory()
			if err != nil {
				return false, nil
			}

			// Setting the read deadline will cause NextReader in read
			// to fail if it is exceeded. This deadline is reset each
			// time we receive a pong message so we know the connection
			// is still intact.
			conn.SetReadDeadline(time.Now().Add(pongTimeout))
			conn.SetPongHandler(func(string) error {
				conn.SetReadDeadline(time.Now().Add(pongTimeout))
				return nil
			})

			c.connectionLock.Lock()
			defer c.connectionLock.Unlock()

			c.connection = conn
			return true, nil
		case <-c.closeChan:
			err = errShuttingDown
			return false, err
		}
	})

	return err
}

// keepalive keeps the connection open.
func (c *ManagedConnection) keepalive() error {
	for {
		select {
		default:
			if err := c.read(); err != nil {
				return err
			}
		case <-c.closeChan:
			return errShuttingDown
		}
	}
}

// closeConnection closes the underlying websocket connection.
func (c *ManagedConnection) closeConnection() error {
	c.connectionLock.Lock()
	defer c.connectionLock.Unlock()

	if c.connection != nil {
		err := c.connection.Close()
		c.connection = nil
		return err
	}
	return nil
}

// read reads the next message from the connection.
// If a messageChan is supplied and the current message type is not
// a control message, the message is sent to that channel.
func (c *ManagedConnection) read() error {
	c.connectionLock.RLock()
	defer c.connectionLock.RUnlock()

	if c.connection == nil {
		return ErrConnectionNotEstablished
	}

	c.readerLock.Lock()
	defer c.readerLock.Unlock()

	messageType, reader, err := c.connection.NextReader()
	if err != nil {
		return err
	}

	// Send the message to the channel if its an application level message
	// and if that channel is set.
	// TODO(markusthoemmes): Return the messageType along with the payload.
	if c.messageChan != nil && (messageType == websocket.TextMessage || messageType == websocket.BinaryMessage) {
		if message, _ := ioutil.ReadAll(reader); message != nil {
			c.messageChan <- message
		}
	}

	return nil
}

func (c *ManagedConnection) write(messageType int, body []byte) error {
	c.connectionLock.RLock()
	defer c.connectionLock.RUnlock()

	if c.connection == nil {
		return ErrConnectionNotEstablished
	}

	c.writerLock.Lock()
	defer c.writerLock.Unlock()

	return c.connection.WriteMessage(messageType, body)
}

// Send sends an encodable message over the websocket connection.
func (c *ManagedConnection) Send(msg interface{}) error {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	if err := enc.Encode(msg); err != nil {
		return err
	}

	return c.write(websocket.BinaryMessage, b.Bytes())
}

// Shutdown closes the websocket connection.
func (c *ManagedConnection) Shutdown() error {
	c.closeOnce.Do(func() {
		close(c.closeChan)
	})

	err := c.closeConnection()
	c.processingWg.Wait()
	return err
}
