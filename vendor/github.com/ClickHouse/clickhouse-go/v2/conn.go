// Licensed to ClickHouse, Inc. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. ClickHouse, Inc. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package clickhouse

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/resources"
	"github.com/pkg/errors"

	"github.com/ClickHouse/ch-go/compress"
	chproto "github.com/ClickHouse/ch-go/proto"
	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
)

func dial(ctx context.Context, addr string, num int, opt *Options) (*connect, error) {
	var (
		err    error
		conn   net.Conn
		debugf = func(format string, v ...any) {}
	)

	switch {
	case opt.DialContext != nil:
		conn, err = opt.DialContext(ctx, addr)
	default:
		switch {
		case opt.TLS != nil:
			conn, err = tls.DialWithDialer(&net.Dialer{Timeout: opt.DialTimeout}, "tcp", addr, opt.TLS)
		default:
			conn, err = net.DialTimeout("tcp", addr, opt.DialTimeout)
		}
	}

	if err != nil {
		return nil, err
	}

	if opt.Debug {
		if opt.Debugf != nil {
			debugf = func(format string, v ...any) {
				opt.Debugf(
					"[clickhouse][conn=%d][%s] "+format,
					append([]interface{}{num, conn.RemoteAddr()}, v...)...,
				)
			}
		} else {
			debugf = log.New(os.Stdout, fmt.Sprintf("[clickhouse][conn=%d][%s]", num, conn.RemoteAddr()), 0).Printf
		}
	}

	compression := CompressionNone
	if opt.Compression != nil {
		switch opt.Compression.Method {
		case CompressionLZ4, CompressionZSTD, CompressionNone:
			compression = opt.Compression.Method
		default:
			return nil, fmt.Errorf("unsupported compression method for native protocol")
		}
	}

	var (
		connect = &connect{
			id:                   num,
			opt:                  opt,
			conn:                 conn,
			debugf:               debugf,
			buffer:               new(chproto.Buffer),
			reader:               chproto.NewReader(conn),
			revision:             ClientTCPProtocolVersion,
			structMap:            &structMap{},
			compression:          compression,
			connectedAt:          time.Now(),
			compressor:           compress.NewWriter(),
			readTimeout:          opt.ReadTimeout,
			blockBufferSize:      opt.BlockBufferSize,
			maxCompressionBuffer: opt.MaxCompressionBuffer,
		}
	)

	if err := connect.handshake(opt.Auth.Database, opt.Auth.Username, opt.Auth.Password); err != nil {
		return nil, err
	}

	if connect.revision >= proto.DBMS_MIN_PROTOCOL_VERSION_WITH_ADDENDUM {
		if err := connect.sendAddendum(); err != nil {
			return nil, err
		}
	}

	// warn only on the first connection in the pool
	if num == 1 && !resources.ClientMeta.IsSupportedClickHouseVersion(connect.server.Version) {
		debugf("[handshake] WARNING: version %v of ClickHouse is not supported by this client - client supports %v", connect.server.Version, resources.ClientMeta.SupportedVersions())
	}

	return connect, nil
}

// https://github.com/ClickHouse/ClickHouse/blob/master/src/Client/Connection.cpp
type connect struct {
	id                   int
	opt                  *Options
	conn                 net.Conn
	debugf               func(format string, v ...any)
	server               ServerVersion
	closed               bool
	buffer               *chproto.Buffer
	reader               *chproto.Reader
	released             bool
	revision             uint64
	structMap            *structMap
	compression          CompressionMethod
	connectedAt          time.Time
	compressor           *compress.Writer
	readTimeout          time.Duration
	blockBufferSize      uint8
	maxCompressionBuffer int
	readerMutex          sync.Mutex
	closeMutex           sync.Mutex
}

func (c *connect) settings(querySettings Settings) []proto.Setting {
	settingToProtoSetting := func(k string, v any) proto.Setting {
		isCustom := false
		if cv, ok := v.(CustomSetting); ok {
			v = cv.Value
			isCustom = true
		}

		return proto.Setting{
			Key:       k,
			Value:     v,
			Important: !isCustom,
			Custom:    isCustom,
		}
	}

	settings := make([]proto.Setting, 0, len(c.opt.Settings)+len(querySettings))
	for k, v := range c.opt.Settings {
		settings = append(settings, settingToProtoSetting(k, v))
	}

	for k, v := range querySettings {
		settings = append(settings, settingToProtoSetting(k, v))
	}

	return settings
}

func (c *connect) isBad() bool {
	if c.isClosed() {
		return true
	}

	if time.Since(c.connectedAt) >= c.opt.ConnMaxLifetime {
		return true
	}

	if err := c.connCheck(); err != nil {
		return true
	}

	return false
}

func (c *connect) isClosed() bool {
	c.closeMutex.Lock()
	defer c.closeMutex.Unlock()

	return c.closed
}

func (c *connect) setClosed() {
	c.closeMutex.Lock()
	defer c.closeMutex.Unlock()

	c.closed = true
}

func (c *connect) close() error {
	c.closeMutex.Lock()
	if c.closed {
		c.closeMutex.Unlock()
		return nil
	}
	c.closed = true
	c.closeMutex.Unlock()

	if err := c.conn.Close(); err != nil {
		return err
	}

	c.buffer = nil

	c.readerMutex.Lock()
	c.reader = nil
	c.readerMutex.Unlock()

	return nil
}

func (c *connect) progress() (*Progress, error) {
	var progress proto.Progress
	if err := progress.Decode(c.reader, c.revision); err != nil {
		return nil, err
	}

	c.debugf("[progress] %s", &progress)
	return &progress, nil
}

func (c *connect) exception() error {
	var e Exception
	if err := e.Decode(c.reader); err != nil {
		return err
	}

	c.debugf("[exception] %s", e.Error())
	return &e
}

func (c *connect) compressBuffer(start int) error {
	if c.compression != CompressionNone && len(c.buffer.Buf) > 0 {
		data := c.buffer.Buf[start:]
		if err := c.compressor.Compress(compress.Method(c.compression), data); err != nil {
			return errors.Wrap(err, "compress")
		}
		c.buffer.Buf = append(c.buffer.Buf[:start], c.compressor.Data...)
	}
	return nil
}

func (c *connect) sendData(block *proto.Block, name string) error {
	if c.isClosed() {
		err := errors.New("attempted sending on closed connection")
		c.debugf("[send data] err: %v", err)
		return err
	}

	c.debugf("[send data] compression=%q", c.compression)
	c.buffer.PutByte(proto.ClientData)
	c.buffer.PutString(name)

	compressionOffset := len(c.buffer.Buf)

	if err := block.EncodeHeader(c.buffer, c.revision); err != nil {
		return err
	}

	for i := range block.Columns {
		if err := block.EncodeColumn(c.buffer, c.revision, i); err != nil {
			return err
		}
		if len(c.buffer.Buf) >= c.maxCompressionBuffer {
			if err := c.compressBuffer(compressionOffset); err != nil {
				return err
			}
			c.debugf("[buff compress] buffer size: %d", len(c.buffer.Buf))
			if err := c.flush(); err != nil {
				return err
			}
			compressionOffset = 0
		}
	}

	if err := c.compressBuffer(compressionOffset); err != nil {
		return err
	}

	if err := c.flush(); err != nil {
		switch {
		case errors.Is(err, syscall.EPIPE):
			c.debugf("[send data] pipe is broken, closing connection")
			c.setClosed()
		case errors.Is(err, io.EOF):
			c.debugf("[send data] unexpected EOF, closing connection")
			c.setClosed()
		default:
			c.debugf("[send data] unexpected error: %v", err)
		}
		return err
	}

	defer func() {
		c.buffer.Reset()
	}()

	return nil
}

func (c *connect) readData(ctx context.Context, packet byte, compressible bool) (*proto.Block, error) {
	if c.isClosed() {
		err := errors.New("attempted reading on closed connection")
		c.debugf("[read data] err: %v", err)
		return nil, err
	}

	if c.reader == nil {
		err := errors.New("attempted reading on nil reader")
		c.debugf("[read data] err: %v", err)
		return nil, err
	}

	if _, err := c.reader.Str(); err != nil {
		c.debugf("[read data] str error: %v", err)
		return nil, err
	}

	if compressible && c.compression != CompressionNone {
		c.reader.EnableCompression()
		defer c.reader.DisableCompression()
	}

	opts := queryOptions(ctx)
	location := c.server.Timezone
	if opts.userLocation != nil {
		location = opts.userLocation
	}

	block := proto.Block{Timezone: location}
	if err := block.Decode(c.reader, c.revision); err != nil {
		c.debugf("[read data] decode error: %v", err)
		return nil, err
	}

	block.Packet = packet
	c.debugf("[read data] compression=%q. block: columns=%d, rows=%d", c.compression, len(block.Columns), block.Rows())
	return &block, nil
}

func (c *connect) flush() error {
	if len(c.buffer.Buf) == 0 {
		// Nothing to flush.
		return nil
	}

	n, err := c.conn.Write(c.buffer.Buf)
	if err != nil {
		return errors.Wrap(err, "write")
	}

	if n != len(c.buffer.Buf) {
		return errors.New("wrote less than expected")
	}

	c.buffer.Reset()
	return nil
}
