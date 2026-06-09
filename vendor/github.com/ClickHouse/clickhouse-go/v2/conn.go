package clickhouse

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"syscall"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/column"

	"github.com/ClickHouse/clickhouse-go/v2/resources"

	"github.com/ClickHouse/ch-go/compress"
	chproto "github.com/ClickHouse/ch-go/proto"

	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
)

func dial(ctx context.Context, addr string, num int, opt *Options) (*connect, error) {
	var (
		err  error
		conn net.Conn
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

	// Get base logger and enrich with connection-specific context
	baseLogger := opt.logger()
	logger := prepareConnLogger(baseLogger, num, conn.RemoteAddr().String(), "native")

	var (
		compression CompressionMethod
		compressor  *compress.Writer
	)
	if opt.Compression != nil {
		switch opt.Compression.Method {
		case CompressionLZ4, CompressionLZ4HC, CompressionZSTD, CompressionNone:
			compression = opt.Compression.Method
		default:
			return nil, fmt.Errorf("unsupported compression method for native protocol")
		}

		compressor = compress.NewWriter(compress.Level(opt.Compression.Level), compress.Method(opt.Compression.Method))
	} else {
		compression = CompressionNone
		compressor = compress.NewWriter(compress.LevelZero, compress.None)
	}

	var (
		connect = &connect{
			id:                   num,
			opt:                  opt,
			conn:                 conn,
			logger:               logger,
			buffer:               new(chproto.Buffer),
			reader:               chproto.NewReader(conn),
			revision:             ClientTCPProtocolVersion,
			structMap:            &structMap{},
			compression:          compression,
			connectedAt:          time.Now(),
			compressor:           compressor,
			readTimeout:          opt.ReadTimeout,
			blockBufferSize:      opt.BlockBufferSize,
			maxCompressionBuffer: opt.MaxCompressionBuffer,
		}
	)

	auth := opt.Auth
	if useJWTAuth(opt) {
		jwt, err := opt.GetJWT(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get JWT: %w", err)
		}

		auth.Username = jwtAuthMarker
		auth.Password = jwt
	}

	if err := connect.handshake(auth); err != nil {
		return nil, err
	}

	if connect.revision >= proto.DBMS_MIN_PROTOCOL_VERSION_WITH_ADDENDUM {
		if err := connect.sendAddendum(); err != nil {
			return nil, err
		}
	}

	// warn only on the first connection in the pool
	if num == 1 && !resources.ClientMeta.IsSupportedClickHouseVersion(connect.server.Version) {
		connect.logger.Warn("unsupported clickhouse version",
			slog.String("version", connect.server.Version.String()),
			slog.String("supported_versions", resources.ClientMeta.SupportedVersions()))
	}

	return connect, nil
}

// https://github.com/ClickHouse/ClickHouse/blob/master/src/Client/Connection.cpp
type connect struct {
	id                   int
	opt                  *Options
	conn                 net.Conn
	logger               *slog.Logger
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

func (c *connect) connID() int {
	return c.id
}

func (c *connect) getLogger() *slog.Logger {
	return c.logger
}

func (c *connect) connectedAtTime() time.Time {
	return c.connectedAt
}

func (c *connect) serverVersion() (*ServerVersion, error) {
	return &c.server, nil
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

func (c *connect) isReleased() bool {
	return c.released
}

func (c *connect) setReleased(released bool) {
	c.released = released
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
	c.compressor = nil

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

	c.logger.Debug("query progress",
		slog.Uint64("rows", progress.Rows),
		slog.Uint64("bytes", progress.Bytes),
		slog.Uint64("total_rows", progress.TotalRows))
	return &progress, nil
}

func (c *connect) exception() error {
	var e Exception
	if err := e.Decode(c.reader); err != nil {
		return err
	}

	c.logger.Warn("server exception received",
		slog.String("error", e.Error()),
		slog.Int("code", int(e.Code)))
	return &e
}

func (c *connect) compressBuffer(start int) error {
	if c.compression != CompressionNone && len(c.buffer.Buf) > 0 {
		data := c.buffer.Buf[start:]
		if err := c.compressor.Compress(data); err != nil {
			return fmt.Errorf("compress: %w", err)
		}
		c.buffer.Buf = append(c.buffer.Buf[:start], c.compressor.Data...)
	}
	return nil
}

func (c *connect) sendData(block *proto.Block, name string) error {
	if c.isClosed() {
		err := errors.New("attempted sending on closed connection")
		c.logger.Error("send data failed: connection closed", slog.Any("error", err))
		return err
	}

	c.logger.Debug("sending data block",
		slog.String("compression", c.compression.String()),
		slog.Int("columns", len(block.Columns)),
		slog.Int("rows", block.Rows()))
	c.buffer.PutByte(proto.ClientData)
	c.buffer.PutString(name)

	compressionOffset := len(c.buffer.Buf)

	if err := block.EncodeHeader(c.buffer, c.revision); err != nil {
		return fmt.Errorf("send data: failed to encode block header (conn_id=%d): %w", c.id, err)
	}

	for i := range block.Columns {
		if err := block.EncodeColumn(c.buffer, c.revision, i); err != nil {
			return fmt.Errorf("send data: failed to encode column %d (conn_id=%d): %w", i, c.id, err)
		}
		if len(c.buffer.Buf) >= c.maxCompressionBuffer {
			if err := c.compressBuffer(compressionOffset); err != nil {
				return err
			}
			c.logger.Debug("buffer compressed",
				slog.Int("buffer_bytes", len(c.buffer.Buf)))
			if err := c.flush(); err != nil {
				return fmt.Errorf("send data: failed to flush partial block (conn_id=%d, col=%d): %w", c.id, i, err)
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
			c.logger.Error("connection broken: pipe error",
				slog.Any("error", err),
				slog.Int("block_columns", len(block.Columns)),
				slog.Int("block_rows", block.Rows()))
			c.setClosed()
			return fmt.Errorf("send data: connection broken (EPIPE) to %s (conn_id=%d, block_cols=%d, block_rows=%d): %w",
				c.conn.RemoteAddr(), c.id, len(block.Columns), block.Rows(), err)
		case errors.Is(err, io.EOF):
			c.logger.Error("connection closed unexpectedly",
				slog.Any("error", err),
				slog.Int("block_columns", len(block.Columns)),
				slog.Int("block_rows", block.Rows()))
			c.setClosed()
			return fmt.Errorf("send data: unexpected EOF to %s (conn_id=%d, block_cols=%d, block_rows=%d): %w",
				c.conn.RemoteAddr(), c.id, len(block.Columns), block.Rows(), err)
		default:
			c.logger.Error("send data failed",
				slog.Any("error", err),
				slog.Int("block_columns", len(block.Columns)),
				slog.Int("block_rows", block.Rows()))
			return fmt.Errorf("send data: write error to %s (conn_id=%d, block_cols=%d, block_rows=%d): %w",
				c.conn.RemoteAddr(), c.id, len(block.Columns), block.Rows(), err)
		}
	}

	defer func() {
		c.buffer.Reset()
	}()

	return nil
}

func serverVersionToContext(v ServerVersion) column.ServerContext {
	return column.ServerContext{
		Revision:     v.Revision,
		VersionMajor: v.Version.Major,
		VersionMinor: v.Version.Minor,
		VersionPatch: v.Version.Patch,
		Timezone:     v.Timezone,
	}
}

func (c *connect) readData(ctx context.Context, packet byte, compressible bool) (*proto.Block, error) {
	if c.isClosed() {
		err := errors.New("attempted reading on closed connection")
		c.logger.Error("read data failed: connection closed", slog.Any("error", err))
		return nil, err
	}

	if c.reader == nil {
		err := errors.New("attempted reading on nil reader")
		c.logger.Error("read data failed: nil reader", slog.Any("error", err))
		return nil, err
	}

	if _, err := c.reader.Str(); err != nil {
		c.logger.Error("read data failed: cannot read block name", slog.Any("error", err))
		return nil, fmt.Errorf("read data: failed to read block name from %s (conn_id=%d): %w",
			c.conn.RemoteAddr(), c.id, err)
	}

	if compressible && c.compression != CompressionNone {
		c.reader.EnableCompression()
		defer c.reader.DisableCompression()
	}

	userLocation := queryOptionsUserLocation(ctx)
	location := c.server.Timezone
	if userLocation != nil {
		location = userLocation
	}

	serverContext := serverVersionToContext(c.server)
	serverContext.Timezone = location
	block := proto.Block{ServerContext: &serverContext}
	if err := block.Decode(c.reader, c.revision); err != nil {
		c.logger.Error("read data failed: decode error",
			slog.Any("error", err),
			slog.String("compression", c.compression.String()))
		return nil, fmt.Errorf("read data: failed to decode block from %s (conn_id=%d, compression=%s): %w",
			c.conn.RemoteAddr(), c.id, c.compression, err)
	}

	block.Packet = packet
	c.logger.Debug("data block received",
		slog.String("compression", c.compression.String()),
		slog.Int("columns", len(block.Columns)),
		slog.Int("rows", block.Rows()))
	return &block, nil
}

func (c *connect) freeBuffer() {
	c.buffer = new(chproto.Buffer)
	c.compressor.Data = nil
}

func (c *connect) flush() error {
	if len(c.buffer.Buf) == 0 {
		// Nothing to flush.
		return nil
	}

	n, err := c.conn.Write(c.buffer.Buf)
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}

	if n != len(c.buffer.Buf) {
		return errors.New("wrote less than expected")
	}

	c.buffer.Reset()
	return nil
}

// startReadWriteTimeout applies the configured read timeout to conn.
// If a context deadline is provided, a read and write deadline is set.
// This should be matched with a deferred call to clearReadWriteTimeout.
func (c *connect) startReadWriteTimeout(ctx context.Context) error {
	err := c.conn.SetReadDeadline(time.Now().Add(c.readTimeout))
	if err != nil {
		return err
	}

	// context level deadlines override configured read timeout
	if deadline, ok := ctx.Deadline(); ok {
		return c.conn.SetDeadline(deadline)
	}

	return nil
}

// clearReadWriteTimeout removes the read timeout from conn.
// If a context deadline is provided, the read and write timeout is cleared too.
func (c *connect) clearReadWriteTimeout(ctx context.Context) error {
	err := c.conn.SetReadDeadline(time.Time{})
	if err != nil {
		return err
	}

	// context level deadlines should clear read + write deadlines.
	if _, ok := ctx.Deadline(); ok {
		return c.conn.SetDeadline(time.Time{})
	}

	return nil
}
