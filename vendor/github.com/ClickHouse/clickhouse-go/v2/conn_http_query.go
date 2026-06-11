package clickhouse

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	chproto "github.com/ClickHouse/ch-go/proto"

	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
)

// capturingReader wraps a reader and captures all data that passes through it
type capturingReader struct {
	reader io.Reader
	buffer bytes.Buffer
}

func (r *capturingReader) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	if n > 0 {
		r.buffer.Write(p[:n])
	}
	return n, err
}

// release is ignored, because http used by std with empty release function
func (h *httpConnect) query(ctx context.Context, release nativeTransportRelease, query string, args ...any) (*rows, error) {
	h.logger.Debug("HTTP query", slog.String("sql", query))
	options := queryOptions(ctx)
	query, err := bindQueryOrAppendParameters(true, &options, query, h.handshake.Timezone, args...)
	if err != nil {
		err = fmt.Errorf("bindQueryOrAppendParameters: %w", err)
		release(h, err)
		return nil, err
	}
	headers := make(map[string]string)
	switch h.compression {
	case CompressionZSTD, CompressionLZ4:
		options.settings["compress"] = "1"
	case CompressionGZIP, CompressionDeflate, CompressionBrotli:
		// request encoding
		headers["Accept-Encoding"] = h.compression.String()
	}

	res, err := h.sendQuery(ctx, query, &options, headers) //nolint:bodyclose // false positive
	if err != nil {
		err = fmt.Errorf("sendQuery: %w", err)
		release(h, err)
		return nil, err
	}

	if res.ContentLength == 0 {
		discardAndClose(res.Body)
		block := proto.NewBlock()
		release(h, nil)
		return &rows{
			block:     block,
			columns:   block.ColumnsNames(),
			structMap: &structMap{},
		}, nil
	}

	rw := h.compressionPool.Get()
	// The HTTPReaderWriter.NewReader will create a reader that will decompress it if needed,
	reader, err := rw.NewReader(res)
	if err != nil {
		err = fmt.Errorf("NewReader: %w", err)
		discardAndClose(res.Body)
		h.compressionPool.Put(rw)
		release(h, err)
		return nil, err
	}

	// Wrap reader with capturing reader to detect exceptions
	capturingRdr := &capturingReader{reader: reader}
	bufferedReader := bufio.NewReader(capturingRdr)
	chReader := chproto.NewReader(bufferedReader)
	block, err := h.readData(chReader, options.userLocation, &capturingRdr.buffer)
	if err != nil && !errors.Is(err, io.EOF) {
		err = fmt.Errorf("readData: %w", err)
		discardAndClose(res.Body)
		h.compressionPool.Put(rw)
		release(h, err)
		return nil, err
	}

	bufferSize := h.blockBufferSize
	if options.blockBufferSize > 0 {
		// allow block buffer size to be overridden per query
		bufferSize = options.blockBufferSize
	}
	var (
		errCh  = make(chan error)
		stream = make(chan *proto.Block, bufferSize)
	)
	go func() {
		for {
			block, err := h.readData(chReader, options.userLocation, &capturingRdr.buffer)
			if err != nil {
				// ch-go wraps EOF errors
				if !errors.Is(err, io.EOF) {
					errCh <- fmt.Errorf("readData stream: %w", err)
				}
				break
			}
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				break
			case stream <- block:
			}
		}
		discardAndClose(res.Body)
		h.compressionPool.Put(rw)
		close(stream)
		close(errCh)
		release(h, nil)
	}()

	if block == nil {
		block = proto.NewBlock()
	}

	return &rows{
		block:     block,
		stream:    stream,
		errors:    errCh,
		columns:   block.ColumnsNames(),
		structMap: &structMap{},
	}, nil
}

func (h *httpConnect) queryRow(ctx context.Context, release nativeTransportRelease, query string, args ...any) *row {
	rows, err := h.query(ctx, release, query, args...)
	if err != nil {
		return &row{
			err: err,
		}
	}

	return &row{
		rows: rows,
	}
}
