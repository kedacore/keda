package clickhouse

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"context"
	sqldriver "database/sql/driver"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ClickHouse/ch-go/compress"
	chproto "github.com/ClickHouse/ch-go/proto"
	"github.com/andybalholm/brotli"

	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
)

const (
	quotaKeyParamName = "quota_key"
	queryIDParamName  = "query_id"
)

type Pool[T any] struct {
	pool *sync.Pool
}

func NewPool[T any](fn func() T) Pool[T] {
	return Pool[T]{
		pool: &sync.Pool{New: func() any { return fn() }},
	}
}

func (p *Pool[T]) Get() T {
	return p.pool.Get().(T)
}

func (p *Pool[T]) Put(x T) {
	p.pool.Put(x)
}

type HTTPReaderWriter struct {
	reader io.Reader
	writer io.WriteCloser
	err    error
	method CompressionMethod
}

// NewReader will return a reader that will decompress data if needed.
func (rw *HTTPReaderWriter) NewReader(res *http.Response) (io.Reader, error) {
	enc := res.Header.Get("Content-Encoding")
	if !res.Uncompressed && rw.method.String() == enc {
		switch rw.method {
		case CompressionGZIP:
			reader := rw.reader.(*gzip.Reader)
			if err := reader.Reset(res.Body); err != nil {
				return nil, err
			}
			return reader, nil
		case CompressionDeflate:
			reader := rw.reader
			if err := reader.(flate.Resetter).Reset(res.Body, nil); err != nil {
				return nil, err
			}
			return reader, nil
		case CompressionBrotli:
			reader := rw.reader.(*brotli.Reader)
			if err := reader.Reset(res.Body); err != nil {
				return nil, err
			}
			return reader, nil
		}
	}
	return res.Body, nil
}

func (rw *HTTPReaderWriter) reset(pw *io.PipeWriter) io.WriteCloser {
	switch rw.method {
	case CompressionGZIP:
		rw.writer.(*gzip.Writer).Reset(pw)
		return rw.writer
	case CompressionDeflate:
		rw.writer.(*zlib.Writer).Reset(pw)
		return rw.writer
	case CompressionBrotli:
		rw.writer.(*brotli.Writer).Reset(pw)
		return rw.writer
	default:
		return pw
	}
}

// applyOptionsToRequest applies the client Options (such as auth, headers, client info) to the given http.Request
func applyOptionsToRequest(ctx context.Context, req *http.Request, opt *Options) error {
	queryOpt := queryOptions(ctx)

	jwt := queryOpt.jwt
	useJWT := jwt != "" || useJWTAuth(opt)

	switch {
	case opt.TLS != nil && useJWT:
		if jwt == "" {
			var err error
			jwt, err = opt.GetJWT(ctx)
			if err != nil {
				return fmt.Errorf("failed to get JWT: %w", err)
			}
		}

		req.Header.Set("Authorization", "Bearer "+jwt)
	case opt.TLS != nil && len(opt.Auth.Username) > 0:
		req.Header.Set("X-ClickHouse-User", opt.Auth.Username)
		if len(opt.Auth.Password) > 0 {
			req.Header.Set("X-ClickHouse-Key", opt.Auth.Password)
			req.Header.Set("X-ClickHouse-SSL-Certificate-Auth", "off")
		} else {
			req.Header.Set("X-ClickHouse-SSL-Certificate-Auth", "on")
		}
	case opt.TLS == nil && len(opt.Auth.Username) > 0:
		if len(opt.Auth.Password) > 0 {
			req.URL.User = url.UserPassword(opt.Auth.Username, opt.Auth.Password)

		} else {
			req.URL.User = url.User(opt.Auth.Username)
		}
	}

	req.Header.Set("User-Agent", opt.ClientInfo.Append(queryOpt.clientInfo).String())

	for k, v := range opt.HttpHeaders {
		if strings.EqualFold(k, "Host") {
			req.Host = v
		} else {
			req.Header.Set(k, v)
		}
	}

	return nil
}

func dialHttp(ctx context.Context, addr string, num int, opt *Options) (*httpConnect, error) {
	// Get base logger and enrich with connection-specific context
	baseLogger := opt.logger()
	logger := prepareConnLogger(baseLogger, num, addr, "http")
	scheme := opt.scheme
	compression := opt.Compression

	if scheme == "" {
		switch opt.Protocol {
		case HTTP:
			scheme = opt.Protocol.String()
			if opt.TLS != nil {
				scheme = fmt.Sprintf("%ss", scheme)
			}
		default:
			return nil, errors.New("invalid interface type for http")
		}
	}
	u := &url.URL{
		Scheme: scheme,
		Host:   addr,
		Path:   opt.HttpUrlPath,
	}

	query := u.Query()
	if len(opt.Auth.Database) > 0 {
		query.Set("database", opt.Auth.Database)
	}

	if compression == nil {
		compression = &Compression{
			Method: CompressionNone,
		}
	}

	compressionPool, err := createCompressionPool(compression)
	if err != nil {
		return nil, err
	}

	for k, v := range opt.Settings {
		if cv, ok := v.(CustomSetting); ok {
			v = cv.Value
		}

		query.Set(k, fmt.Sprint(v))
	}

	query.Set("default_format", "Native")
	query.Set("client_protocol_version", strconv.Itoa(ClientTCPProtocolVersion))
	u.RawQuery = query.Encode()

	rt, err := createHTTPRoundTripper(opt)
	if err != nil {
		return nil, err
	}

	conn := httpConnect{
		id:          num,
		connectedAt: time.Now(),
		released:    false,
		logger:      logger,
		opt:         opt,
		client: &http.Client{
			Transport: rt,
		},
		url:             u,
		revision:        ClientTCPProtocolVersion, // Preflight uses hardcoded revision, may break older versions.
		encodeRevision:  0,                        // Encoding data over HTTP must use 0. client_protocol_version does not apply to inserts.
		buffer:          new(chproto.Buffer),
		compression:     compression.Method,
		blockCompressor: compress.NewWriter(compress.Level(compression.Level), compress.Method(compression.Method)),
		compressionPool: compressionPool,
		blockBufferSize: opt.BlockBufferSize,
	}

	handshake, err := conn.queryHello(ctx, func(nativeTransport, error) {})
	if err != nil {
		return nil, fmt.Errorf("failed to query server hello: %w", err)
	}
	conn.handshake = handshake
	conn.revision = conn.handshake.Revision

	return &conn, nil
}

func createHTTPRoundTripper(opt *Options) (http.RoundTripper, error) {
	httpProxy := http.ProxyFromEnvironment
	if opt.HTTPProxyURL != nil {
		httpProxy = http.ProxyURL(opt.HTTPProxyURL)
	}

	rt := &http.Transport{
		Proxy: httpProxy,
		DialContext: (&net.Dialer{
			Timeout: opt.DialTimeout,
		}).DialContext,
		MaxIdleConns:          1,
		MaxConnsPerHost:       opt.HttpMaxConnsPerHost,
		IdleConnTimeout:       opt.ConnMaxLifetime,
		ResponseHeaderTimeout: opt.ReadTimeout,
		TLSClientConfig:       opt.TLS,
		DisableCompression:    true,
	}

	if opt.DialContext != nil {
		rt.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return opt.DialContext(ctx, addr)
		}
	}

	if opt.TransportFunc == nil {
		return rt, nil
	}

	return opt.TransportFunc(rt)
}

type httpConnect struct {
	id              int
	connectedAt     time.Time
	released        bool
	logger          *slog.Logger
	opt             *Options
	revision        uint64
	encodeRevision  uint64
	url             *url.URL
	client          *http.Client
	buffer          *chproto.Buffer
	compression     CompressionMethod
	blockCompressor *compress.Writer
	compressionPool Pool[HTTPReaderWriter]
	blockBufferSize uint8
	handshake       proto.ServerHandshake
}

func (h *httpConnect) serverVersion() (*ServerVersion, error) {
	return &h.handshake, nil
}

func (h *httpConnect) connID() int {
	return h.id
}

func (h *httpConnect) connectedAtTime() time.Time {
	return h.connectedAt
}

func (h *httpConnect) getLogger() *slog.Logger {
	return h.logger
}

func (h *httpConnect) isReleased() bool {
	return h.released
}

func (h *httpConnect) setReleased(released bool) {
	h.released = released
}

func (h *httpConnect) freeBuffer() {
}

func (h *httpConnect) isBad() bool {
	return h.client == nil
}

func (h *httpConnect) queryHello(ctx context.Context, release nativeTransportRelease) (proto.ServerHandshake, error) {
	h.logger.Debug("querying server info via HTTP")
	ctx = Context(ctx, ignoreExternalTables())
	query := "SELECT displayName(), version(), revision(), timezone()"
	rows, err := h.query(ctx, release, query)
	if err != nil {
		return proto.ServerHandshake{}, fmt.Errorf("failed to query server hello info: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return proto.ServerHandshake{}, errors.New("no rows returned for server hello query")
	}

	var (
		displayName string
		versionStr  string
		revision    uint32
		timezone    string
	)
	if err := rows.Scan(&displayName, &versionStr, &revision, &timezone); err != nil {
		return proto.ServerHandshake{}, err
	}

	location, err := time.LoadLocation(timezone)
	if err != nil {
		return proto.ServerHandshake{}, fmt.Errorf("failed to load timezone from server hello query: %w", err)
	}

	return proto.ServerHandshake{
		Name:        displayName,
		DisplayName: displayName,
		Revision:    uint64(revision),
		Version:     proto.ParseVersion(versionStr),
		Timezone:    location,
	}, nil
}

func createCompressionPool(compression *Compression) (Pool[HTTPReaderWriter], error) {
	pool := NewPool(func() HTTPReaderWriter {
		switch compression.Method {
		case CompressionGZIP:
			// trick so we can init the reader to something to Reset when we reuse
			writer, err := gzip.NewWriterLevel(io.Discard, compression.Level)
			if err != nil {
				return HTTPReaderWriter{err: err}
			}
			b := new(bytes.Buffer)
			writer.Reset(b)
			writer.Flush()
			writer.Close()
			reader, err := gzip.NewReader(bytes.NewReader(b.Bytes()))
			return HTTPReaderWriter{writer: writer, reader: reader, err: err, method: compression.Method}
		case CompressionDeflate:
			writer, err := zlib.NewWriterLevel(io.Discard, compression.Level)
			if err != nil {
				return HTTPReaderWriter{err: err}
			}
			b := new(bytes.Buffer)
			writer.Reset(b)
			writer.Flush()
			writer.Close()
			reader, err := zlib.NewReader(bytes.NewReader(b.Bytes()))
			if err != nil {
				return HTTPReaderWriter{err: err}
			}
			return HTTPReaderWriter{writer: writer, reader: reader, method: compression.Method}
		case CompressionBrotli:
			writer := brotli.NewWriterLevel(io.Discard, compression.Level)
			b := new(bytes.Buffer)
			writer.Reset(b)
			writer.Flush()
			writer.Close()
			reader := brotli.NewReader(bytes.NewReader(b.Bytes()))
			return HTTPReaderWriter{writer: writer, reader: reader, method: compression.Method}
		default:
			return HTTPReaderWriter{method: CompressionNone}
		}
	})
	err := pool.Get().err
	if err != nil {
		return pool, err
	}
	return pool, nil
}

func (h *httpConnect) writeData(block *proto.Block) error {
	// Saving offset of compressible data
	start := len(h.buffer.Buf)
	if err := block.Encode(h.buffer, h.encodeRevision); err != nil {
		return fmt.Errorf("block encode: %w", err)
	}
	if h.compression == CompressionLZ4 || h.compression == CompressionZSTD {
		// Performing compression. Supported and requires
		data := h.buffer.Buf[start:]
		if err := h.blockCompressor.Compress(data); err != nil {
			return fmt.Errorf("compress: %w", err)
		}
		h.buffer.Buf = append(h.buffer.Buf[:start], h.blockCompressor.Data...)
	}
	return nil
}

func (h *httpConnect) readData(reader *chproto.Reader, timezone *time.Location, captureBuffer *bytes.Buffer) (*proto.Block, error) {
	location := h.handshake.Timezone
	if timezone != nil {
		location = timezone
	}

	serverContext := serverVersionToContext(h.handshake)
	serverContext.Timezone = location
	block := proto.Block{ServerContext: &serverContext}
	if h.compression == CompressionLZ4 || h.compression == CompressionZSTD {
		reader.EnableCompression()
		defer reader.DisableCompression()
	}

	// Try to decode the block
	if err := block.Decode(reader, h.revision); err != nil {
		// Decode failed - check if captured data contains exception marker
		// The decode error typically happens because it tries to read the
		// "__exception__" marker as binary data

		// Exception block size can be up to 16KiB max.
		// https://clickhouse.com/docs/interfaces/http#http_response_codes_caveats
		// NOTE: When exception happens, a dedicated block is allocated for exception and only
		// that exception will be present in the block. So safe to assume whole block size is same
		// exception block max size 16KiB.
		maxSize := int64(16 * 1024)

		// Try to read any remaining data
		lr := &limitedReader{reader: reader, limit: 2 * maxSize} // allocating 2 * maxSize for just in case
		buf := make([]byte, maxSize)
		n, readErr := lr.Read(buf)
		if n > 0 && captureBuffer != nil {
			captureBuffer.Write(buf[:n])
		}
		if readErr != nil {
			h.logger.Error("HTTP read data: decode error while parsing exception block", slog.Any("error", err))
		}

		// Check if the captured data contains the exception marker
		if captureBuffer != nil && bytes.Contains(captureBuffer.Bytes(), []byte("__exception__")) {
			// This is an exception block, parse it
			return nil, parseExceptionFromBytes(captureBuffer.Bytes())
		}

		// Not an exception, return the original decode error
		return nil, fmt.Errorf("block decode for exception: %w", err)
	}
	return &block, nil
}

// limitedReader is a helper to read from chproto.Reader up to a limit
type limitedReader struct {
	reader *chproto.Reader
	limit  int64
	read   int64
}

func (lr *limitedReader) Read(p []byte) (n int, err error) {
	if lr.read >= lr.limit {
		return 0, io.EOF
	}
	if int64(len(p)) > lr.limit-lr.read {
		p = p[0 : lr.limit-lr.read]
	}
	n, err = lr.reader.Read(p)
	lr.read += int64(n)
	return
}

// parseExceptionFromBytes parses ClickHouse exception block
// Format from ClickHouse server (WriteBufferFromHTTPServerResponse.cpp):
//
//	\r\n
//	__exception__
//	\r\n
//	<TAG>  (16 bytes)
//	\r\n
//	<error message>
//	\n
//	<message_length> <TAG>
//	\r\n
//	__exception__
//	\r\n
func parseExceptionFromBytes(data []byte) error {
	const (
		exceptionMarker    = "__exception__"
		exceptionMarkerLen = len(exceptionMarker)
		exceptionTagLen    = 16 // bytes
	)

	dataStr := string(data)

	// Find the first __exception__ marker
	firstMarker := strings.Index(dataStr, exceptionMarker)
	if firstMarker < 0 {
		return fmt.Errorf("exception marker not found")
	}

	// Skip past first __exception__\r\n
	pos := firstMarker + exceptionMarkerLen
	// Skip \r\n after first marker
	if pos+2 < len(dataStr) && dataStr[pos:pos+2] == "\r\n" {
		pos += 2
	}

	// Skip the exception tag (16 bytes) + \r\n
	if pos+exceptionTagLen+2 < len(dataStr) {
		pos += exceptionTagLen + 2 // tag + \r\n
	}

	// Now we're at the start of the error message
	// Find the second __exception__ marker
	secondMarker := strings.Index(dataStr[pos:], exceptionMarker)
	if secondMarker < 0 {
		// If we can't find second marker, just extract what we can
		errorMsg := strings.TrimSpace(dataStr[pos:])
		if len(errorMsg) > 0 {
			return fmt.Errorf("ClickHouse exception: %s", errorMsg)
		}
		return fmt.Errorf("ClickHouse exception occurred but could not parse details")
	}

	// Extract error message between tag and second marker
	// The error message ends with: \n<message_length> <TAG>\r\n__exception__
	errorContent := dataStr[pos : pos+secondMarker]

	// Find the last line which contains "<message_length> <TAG>"
	lines := strings.Split(strings.TrimRight(errorContent, "\r\n"), "\n")
	if len(lines) == 0 {
		return fmt.Errorf("ClickHouse exception occurred but could not parse details")
	}

	// The last line is "<message_length> <TAG>", error message is everything before it
	errorMsg := strings.Join(lines[:len(lines)-1], "\n")
	errorMsg = strings.TrimSpace(errorMsg)

	if len(errorMsg) == 0 {
		return fmt.Errorf("ClickHouse exception occurred but message is empty")
	}

	return fmt.Errorf("ClickHouse exception: %s", errorMsg)
}

func (h *httpConnect) sendStreamQuery(ctx context.Context, r io.Reader, options *QueryOptions, headers map[string]string) (*http.Response, error) {
	req, err := h.createRequest(ctx, h.url.String(), r, options, headers)
	if err != nil {
		return nil, err
	}

	res, err := h.executeRequest(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (h *httpConnect) sendQuery(ctx context.Context, query string, options *QueryOptions, headers map[string]string) (*http.Response, error) {
	req, err := h.prepareRequest(ctx, query, options, headers)
	if err != nil {
		return nil, err
	}

	res, err := h.executeRequest(req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (h *httpConnect) readRawResponse(response *http.Response) (body []byte, err error) {
	rw := h.compressionPool.Get()
	defer h.compressionPool.Put(rw)

	reader, err := rw.NewReader(response)
	if err != nil {
		return nil, err
	}
	if h.compression == CompressionLZ4 || h.compression == CompressionZSTD {
		chReader := chproto.NewReader(reader)
		chReader.EnableCompression()
		reader = chReader
	}

	body, err = io.ReadAll(reader)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	return body, nil
}

func (h *httpConnect) createRequest(ctx context.Context, requestUrl string, reader io.Reader, options *QueryOptions, headers map[string]string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestUrl, reader)
	if err != nil {
		return nil, err
	}

	err = applyOptionsToRequest(ctx, req, h.opt)
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	var query url.Values
	if options != nil {
		query = req.URL.Query()
		if options.queryID != "" {
			query.Set(queryIDParamName, options.queryID)
		}
		if options.quotaKey != "" {
			query.Set(quotaKeyParamName, options.quotaKey)
		}
		for key, value := range options.settings {
			// check that query doesn't change format
			if key == "default_format" {
				continue
			}
			if cv, ok := value.(CustomSetting); ok {
				value = cv.Value
			}
			query.Set(key, fmt.Sprint(value))
		}
		for key, value := range options.parameters {
			query.Set(fmt.Sprintf("param_%s", key), value)
		}
		req.URL.RawQuery = query.Encode()
	}
	return req, nil
}

func (h *httpConnect) prepareRequest(ctx context.Context, query string, options *QueryOptions, headers map[string]string) (*http.Request, error) {
	if options == nil || len(options.external) == 0 {
		return h.createRequest(ctx, h.url.String(), strings.NewReader(query), options, headers)
	}
	return h.createRequestWithExternalTables(ctx, query, options, headers)
}

func (h *httpConnect) createRequestWithExternalTables(ctx context.Context, query string, options *QueryOptions, headers map[string]string) (*http.Request, error) {
	payload := &bytes.Buffer{}
	w := multipart.NewWriter(payload)
	currentUrl := new(url.URL)
	*currentUrl = *h.url
	queryValues := currentUrl.Query()
	buf := &chproto.Buffer{}
	for _, table := range options.external {
		tableName := table.Name()
		queryValues.Set(fmt.Sprintf("%v_format", tableName), "Native")
		queryValues.Set(fmt.Sprintf("%v_structure", tableName), table.Structure())
		partWriter, err := w.CreateFormFile(tableName, "")
		if err != nil {
			return nil, err
		}
		buf.Reset()
		err = table.Block().Encode(buf, h.encodeRevision)
		if err != nil {
			return nil, err
		}
		_, err = partWriter.Write(buf.Buf)
		if err != nil {
			return nil, err
		}
	}
	currentUrl.RawQuery = queryValues.Encode()
	err := w.WriteField("query", query)
	if err != nil {
		return nil, err
	}
	err = w.Close()
	if err != nil {
		return nil, err
	}

	if headers == nil {
		headers = make(map[string]string)
	}
	headers["Content-Type"] = w.FormDataContentType()

	return h.createRequest(ctx, currentUrl.String(), bytes.NewReader(payload.Bytes()), options, headers)
}

func (h *httpConnect) executeRequest(req *http.Request) (*http.Response, error) {
	if h.client == nil {
		return nil, sqldriver.ErrBadConn
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		defer discardAndClose(resp.Body)
		msgBytes, err := h.readRawResponse(resp)
		if err != nil {
			return nil, fmt.Errorf("[HTTP %d] failed to read response: %w", resp.StatusCode, err)
		}

		return nil, fmt.Errorf("[HTTP %d] response body: \"%s\"", resp.StatusCode, string(msgBytes))
	}
	return resp, nil
}

func (h *httpConnect) ping(ctx context.Context) error {
	ctx = Context(ctx, ignoreExternalTables())
	// release func is called by connection pool
	rows, err := h.query(ctx, func(nativeTransport, error) {}, "SELECT 1")
	if err != nil {
		return err
	}
	defer rows.Close()
	column := rows.Columns()
	// check that we got column 1
	if len(column) == 1 && column[0] == "1" {
		return nil
	}
	return errors.New("clickhouse [ping]:: cannot ping clickhouse")
}

func (h *httpConnect) close() error {
	if h.client == nil {
		return nil
	}
	h.client.CloseIdleConnections()
	h.client = nil
	return nil
}

// discardAndClose discards remaining data and closes the reader.
// Intended for freeing HTTP connections for re-use.
func discardAndClose(rc io.ReadCloser) {
	_, _ = io.Copy(io.Discard, rc)
	_ = rc.Close()
}
