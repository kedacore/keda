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
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"context"
	"database/sql/driver"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/resources"

	"github.com/ClickHouse/ch-go/compress"
	chproto "github.com/ClickHouse/ch-go/proto"
	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
	"github.com/andybalholm/brotli"
	"github.com/pkg/errors"
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

func dialHttp(ctx context.Context, addr string, num int, opt *Options) (*httpConnect, error) {
	var debugf = func(format string, v ...any) {}
	if opt.Debug {
		if opt.Debugf != nil {
			debugf = func(format string, v ...any) {
				opt.Debugf(
					"[clickhouse][conn=%d][%s] "+format,
					append([]interface{}{num, addr}, v...)...,
				)
			}
		} else {
			debugf = log.New(os.Stdout, fmt.Sprintf("[clickhouse][conn=%d][%s]", num, addr), 0).Printf
		}
	}

	if opt.scheme == "" {
		switch opt.Protocol {
		case HTTP:
			opt.scheme = opt.Protocol.String()
			if opt.TLS != nil {
				opt.scheme = fmt.Sprintf("%ss", opt.scheme)
			}
		default:
			return nil, errors.New("invalid interface type for http")
		}
	}
	u := &url.URL{
		Scheme: opt.scheme,
		Host:   addr,
		Path:   opt.HttpUrlPath,
	}

	headers := make(map[string]string)
	for k, v := range opt.HttpHeaders {
		headers[k] = v
	}

	if opt.TLS == nil && len(opt.Auth.Username) > 0 {
		if len(opt.Auth.Password) > 0 {
			u.User = url.UserPassword(opt.Auth.Username, opt.Auth.Password)
		} else {
			u.User = url.User(opt.Auth.Username)
		}
	} else if opt.TLS != nil && len(opt.Auth.Username) > 0 {
		headers["X-ClickHouse-User"] = opt.Auth.Username
		if len(opt.Auth.Password) > 0 {
			headers["X-ClickHouse-Key"] = opt.Auth.Password
			headers["X-ClickHouse-SSL-Certificate-Auth"] = "off"
		} else {
			headers["X-ClickHouse-SSL-Certificate-Auth"] = "on"
		}
	}

	headers["User-Agent"] = opt.ClientInfo.String()

	query := u.Query()
	if len(opt.Auth.Database) > 0 {
		query.Set("database", opt.Auth.Database)
	}

	if opt.Compression == nil {
		opt.Compression = &Compression{
			Method: CompressionNone,
		}
	}

	compressionPool, err := createCompressionPool(opt.Compression)
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
	u.RawQuery = query.Encode()

	httpProxy := http.ProxyFromEnvironment
	if opt.HTTPProxyURL != nil {
		httpProxy = http.ProxyURL(opt.HTTPProxyURL)
	}

	t := &http.Transport{
		Proxy: httpProxy,
		DialContext: (&net.Dialer{
			Timeout: opt.DialTimeout,
		}).DialContext,
		MaxIdleConns:          1,
		IdleConnTimeout:       opt.ConnMaxLifetime,
		ResponseHeaderTimeout: opt.ReadTimeout,
		TLSClientConfig:       opt.TLS,
	}

	if opt.DialContext != nil {
		t.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return opt.DialContext(ctx, addr)
		}
	}

	conn := &httpConnect{
		client: &http.Client{
			Transport: t,
		},
		url:             u,
		buffer:          new(chproto.Buffer),
		compression:     opt.Compression.Method,
		blockCompressor: compress.NewWriter(),
		compressionPool: compressionPool,
		blockBufferSize: opt.BlockBufferSize,
		headers:         headers,
	}
	location, err := conn.readTimeZone(ctx)
	if err != nil {
		return nil, err
	}
	if num == 1 {
		version, err := conn.readVersion(ctx)
		if err != nil {
			return nil, err
		}
		if !resources.ClientMeta.IsSupportedClickHouseVersion(version) {
			debugf("WARNING: version %v of ClickHouse is not supported by this client\n", version)
		}
	}

	return &httpConnect{
		client: &http.Client{
			Transport: t,
		},
		url:             u,
		buffer:          new(chproto.Buffer),
		compression:     opt.Compression.Method,
		blockCompressor: compress.NewWriter(),
		compressionPool: compressionPool,
		location:        location,
		blockBufferSize: opt.BlockBufferSize,
		headers:         headers,
	}, nil
}

type httpConnect struct {
	url             *url.URL
	client          *http.Client
	location        *time.Location
	buffer          *chproto.Buffer
	compression     CompressionMethod
	blockCompressor *compress.Writer
	compressionPool Pool[HTTPReaderWriter]
	blockBufferSize uint8
	headers         map[string]string
}

func (h *httpConnect) isBad() bool {
	return h.client == nil
}

func (h *httpConnect) readTimeZone(ctx context.Context) (*time.Location, error) {
	rows, err := h.query(Context(ctx, ignoreExternalTables()), func(*connect, error) {}, "SELECT timezone()")
	if err != nil {
		return nil, err
	}

	if !rows.Next() {
		return nil, errors.New("unable to determine server timezone")
	}

	var serverLocation string
	if err := rows.Scan(&serverLocation); err != nil {
		return nil, err
	}

	location, err := time.LoadLocation(serverLocation)
	if err != nil {
		return nil, err
	}
	return location, nil
}

func (h *httpConnect) readVersion(ctx context.Context) (proto.Version, error) {
	rows, err := h.query(Context(ctx, ignoreExternalTables()), func(*connect, error) {}, "SELECT version()")
	if err != nil {
		return proto.Version{}, err
	}

	if !rows.Next() {
		return proto.Version{}, errors.New("unable to determine version")
	}

	var v string
	if err := rows.Scan(&v); err != nil {
		return proto.Version{}, err
	}
	version := proto.ParseVersion(v)
	return version, nil
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
	if err := block.Encode(h.buffer, 0); err != nil {
		return err
	}
	if h.compression == CompressionLZ4 || h.compression == CompressionZSTD {
		// Performing compression. Supported and requires
		data := h.buffer.Buf[start:]
		if err := h.blockCompressor.Compress(compress.Method(h.compression), data); err != nil {
			return errors.Wrap(err, "compress")
		}
		h.buffer.Buf = append(h.buffer.Buf[:start], h.blockCompressor.Data...)
	}
	return nil
}

func (h *httpConnect) readData(reader *chproto.Reader, timezone *time.Location) (*proto.Block, error) {
	location := h.location
	if timezone != nil {
		location = timezone
	}

	block := proto.Block{Timezone: location}
	if h.compression == CompressionLZ4 || h.compression == CompressionZSTD {
		reader.EnableCompression()
		defer reader.DisableCompression()
	}
	if err := block.Decode(reader, 0); err != nil {
		return nil, err
	}
	return &block, nil
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
		err = table.Block().Encode(buf, 0)
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
	headers["Content-Type"] = w.FormDataContentType()
	return h.createRequest(ctx, currentUrl.String(), bytes.NewReader(payload.Bytes()), options, headers)
}

func (h *httpConnect) executeRequest(req *http.Request) (*http.Response, error) {
	if h.client == nil {
		return nil, driver.ErrBadConn
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		msg, err := h.readRawResponse(resp)
		if err != nil {
			return nil, fmt.Errorf("clickhouse [execute]:: %d code: failed to read the response: %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("clickhouse [execute]:: %d code: %s", resp.StatusCode, string(msg))
	}
	return resp, nil
}

func (h *httpConnect) ping(ctx context.Context) error {
	rows, err := h.query(Context(ctx, ignoreExternalTables()), nil, "SELECT 1")
	if err != nil {
		return err
	}
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
