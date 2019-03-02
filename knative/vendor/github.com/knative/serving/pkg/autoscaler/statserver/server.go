/*
Copyright 2018 The Knative Authors

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

package statserver

import (
	"bytes"
	"context"
	"encoding/gob"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/knative/serving/pkg/autoscaler"
	"go.uber.org/zap"
)

const closeCodeServiceRestart = 1012 // See https://www.iana.org/assignments/websocket/websocket.xhtml

// Server receives autoscaler statistics over WebSocket and sends them to a channel.
type Server struct {
	addr        string
	wsSrv       http.Server
	servingCh   chan struct{}
	stopCh      chan struct{}
	statsCh     chan<- *autoscaler.StatMessage
	openClients sync.WaitGroup
	logger      *zap.SugaredLogger
}

// New creates a Server which will receive autoscaler statistics and forward them to statsCh until Shutdown is called.
func New(statsServerAddr string, statsCh chan<- *autoscaler.StatMessage, logger *zap.SugaredLogger) *Server {
	svr := Server{
		addr:        statsServerAddr,
		servingCh:   make(chan struct{}),
		stopCh:      make(chan struct{}),
		statsCh:     statsCh,
		openClients: sync.WaitGroup{},
		logger:      logger.Named("stats-websocket-server").With("address", statsServerAddr),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", svr.Handler)
	svr.wsSrv = http.Server{
		Addr:      statsServerAddr,
		Handler:   mux,
		ConnState: svr.onConnStateChange,
	}
	return &svr
}

func (s *Server) onConnStateChange(conn net.Conn, state http.ConnState) {
	if state == http.StateNew {
		tcpConn := conn.(*net.TCPConn)
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(3 * time.Minute)
	}
}

// ListenAndServe listens on the address s.addr and handles incoming connections.
// It blocks until the server fails or Shutdown is called.
// It returns an error or, if Shutdown was called, nil.
func (s *Server) ListenAndServe() error {
	listener, err := s.listen()
	if err != nil {
		return err
	}
	return s.serve(listener)
}

func (s *Server) listen() (net.Listener, error) {
	s.logger.Info("Starting")
	return net.Listen("tcp", s.addr)
}

func (s *Server) serve(l net.Listener) error {
	close(s.servingCh)
	err := s.wsSrv.Serve(l)
	if err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Handler exposes a websocket handler for receiving stats from queue
// sidecar containers.
func (s *Server) Handler(w http.ResponseWriter, r *http.Request) {
	s.logger.Debug("Handle entered")
	var upgrader websocket.Upgrader
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("Error upgrading websocket.", zap.Error(err))
		return
	}

	handlerCh := make(chan struct{})

	s.openClients.Add(1)
	go func() {
		defer s.openClients.Done()
		select {
		case <-s.stopCh:
			// Send a close message to tell the client to immediately reconnect
			s.logger.Debug("Sending close message to client")
			err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(closeCodeServiceRestart, "Restarting"))
			if err != nil {
				s.logger.Errorf("Failed to send close message to client: %#v", err)
			}
			conn.Close()
		case <-handlerCh:
			s.logger.Debug("Handler exit complete")
		}
	}()

	s.logger.Debug("Connection upgraded to WebSocket. Entering receive loop.")

	for {
		messageType, msg, err := conn.ReadMessage()
		if err != nil {
			// We close abnormally, because we're just closing the connection in the client,
			// which is okay. There's no value delaying closure of the connection unnecessarily.
			if websocket.IsCloseError(err, websocket.CloseAbnormalClosure) {
				s.logger.Debug("Handler disconnected")
			} else {
				s.logger.Errorf("Handler exiting on error: %#v", err)
			}
			close(handlerCh)
			return
		}
		if messageType != websocket.BinaryMessage {
			s.logger.Error("Dropping non-binary message.")
			continue
		}
		dec := gob.NewDecoder(bytes.NewBuffer(msg))
		var sm autoscaler.StatMessage
		err = dec.Decode(&sm)
		if err != nil {
			s.logger.Error(err)
			continue
		}

		s.logger.Debugf("Received stat message: %+v", sm)
		// Drop stats from lameducked pods
		if !sm.Stat.LameDuck {
			s.statsCh <- &sm
		}
	}
}

// Shutdown terminates the server gracefully for the given timeout period and then returns.
func (s *Server) Shutdown(timeout time.Duration) {
	<-s.servingCh
	s.logger.Info("Shutting down")
	shutdownStart := time.Now()

	close(s.stopCh)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err := s.wsSrv.Shutdown(ctx)
	if err != nil {
		if err == context.DeadlineExceeded {
			s.logger.Warn("Shutdown timed out")
		} else {
			s.logger.Error("Shutdown failed.", err)
		}
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		s.openClients.Wait()
	}()

	// Wait until all client connections have been closed or any remaining timeout expires.
	select {
	case <-done:
		s.logger.Info("Shutdown complete")
	case <-time.After(shutdownStart.Add(timeout).Sub(time.Now())):
		s.logger.Warn("Shutdown timed out")
	}
	close(s.statsCh)
}
