// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package tcp

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	iggcon "github.com/apache/iggy/foreign/go/contracts"
	ierror "github.com/apache/iggy/foreign/go/errors"
)

type Option func(config *Options)

type Options struct {
	Ctx               context.Context
	ServerAddress     string
	HeartbeatInterval time.Duration
}

func GetDefaultOptions() Options {
	return Options{
		Ctx:               context.Background(),
		ServerAddress:     "127.0.0.1:8090",
		HeartbeatInterval: time.Second * 5,
	}
}

type IggyTcpClient struct {
	conn               *net.TCPConn
	mtx                sync.Mutex
	MessageCompression iggcon.IggyMessageCompression
}

// WithServerAddress Sets the server address for the TCP client.
func WithServerAddress(address string) Option {
	return func(opts *Options) {
		opts.ServerAddress = address
	}
}

// WithContext sets context
func WithContext(ctx context.Context) Option {
	return func(opts *Options) {
		opts.Ctx = ctx
	}
}

func NewIggyTcpClient(options ...Option) (*IggyTcpClient, error) {
	opts := GetDefaultOptions()
	for _, opt := range options {
		if opt != nil {
			opt(&opts)
		}
	}
	addr, err := net.ResolveTCPAddr("tcp", opts.ServerAddress)
	if err != nil {
		return nil, err
	}
	ctx := opts.Ctx
	var d = net.Dialer{
		KeepAlive: -1,
	}
	conn, err := d.DialContext(ctx, "tcp", addr.String())
	if err != nil {
		return nil, err
	}

	client := &IggyTcpClient{
		conn: conn.(*net.TCPConn),
	}

	heartbeatInterval := opts.HeartbeatInterval
	if heartbeatInterval > 0 {
		go func() {
			ticker := time.NewTicker(heartbeatInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if err = client.Ping(); err != nil {
						log.Printf("[WARN] heartbeat failed: %v", err)
					}
				}
			}
		}()
	}

	return client, nil
}

const (
	RequestInitialBytesLength  = 4
	ResponseInitialBytesLength = 8
	MaxStringLength            = 255
	MaxPartitionCount          = 1000
)

func (tms *IggyTcpClient) read(expectedSize int) (int, []byte, error) {
	var totalRead int
	buffer := make([]byte, expectedSize)

	for totalRead < expectedSize {
		readSize := expectedSize - totalRead
		n, err := tms.conn.Read(buffer[totalRead : totalRead+readSize])
		if err != nil {
			return totalRead, buffer[:totalRead], err
		}
		totalRead += n
	}

	return totalRead, buffer, nil
}

func (tms *IggyTcpClient) write(payload []byte) (int, error) {
	var totalWritten int
	for totalWritten < len(payload) {
		n, err := tms.conn.Write(payload[totalWritten:])
		if err != nil {
			return totalWritten, err
		}
		totalWritten += n
	}

	return totalWritten, nil
}

func (tms *IggyTcpClient) sendAndFetchResponse(message []byte, command iggcon.CommandCode) ([]byte, error) {
	tms.mtx.Lock()
	defer tms.mtx.Unlock()

	payload := createPayload(message, command)
	if _, err := tms.write(payload); err != nil {
		return nil, err
	}

	readBytes, buffer, err := tms.read(ResponseInitialBytesLength)
	if err != nil {
		return nil, fmt.Errorf("failed to read response for TCP request: %w", err)
	}

	if readBytes != ResponseInitialBytesLength {
		return nil, fmt.Errorf("received an invalid or empty response: %w", ierror.EmptyResponse{})
	}

	if status := ierror.Code(binary.LittleEndian.Uint32(buffer[0:4])); status != 0 {
		return nil, ierror.FromCode(status)
	}

	length := int(binary.LittleEndian.Uint32(buffer[4:]))
	if length <= 1 {
		return []byte{}, nil
	}

	_, buffer, err = tms.read(length)
	if err != nil {
		return nil, err
	}

	return buffer, nil
}

func createPayload(message []byte, command iggcon.CommandCode) []byte {
	messageLength := len(message) + 4
	messageBytes := make([]byte, RequestInitialBytesLength+messageLength)
	binary.LittleEndian.PutUint32(messageBytes[:4], uint32(messageLength))
	binary.LittleEndian.PutUint32(messageBytes[4:8], uint32(command))
	copy(messageBytes[8:], message)
	return messageBytes
}
