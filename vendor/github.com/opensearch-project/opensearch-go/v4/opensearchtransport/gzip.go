// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.

package opensearchtransport

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"sync"
)

type gzipCompressor struct {
	gzipWriterPool *sync.Pool
	bufferPool     *sync.Pool
}

// newGzipCompressor returns a new gzipCompressor that uses a sync.Pool to reuse gzip.Writers.
func newGzipCompressor() *gzipCompressor {
	gzipWriterPool := sync.Pool{
		New: func() any {
			return gzip.NewWriter(io.Discard)
		},
	}

	bufferPool := sync.Pool{
		New: func() any {
			return new(bytes.Buffer)
		},
	}

	return &gzipCompressor{
		gzipWriterPool: &gzipWriterPool,
		bufferPool:     &bufferPool,
	}
}

func (pg *gzipCompressor) compress(rc io.ReadCloser) (*bytes.Buffer, error) {
	writer := pg.gzipWriterPool.Get().(*gzip.Writer)
	defer pg.gzipWriterPool.Put(writer)

	buf := pg.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	writer.Reset(buf)

	if _, err := io.Copy(writer, rc); err != nil {
		return nil, fmt.Errorf("failed to compress request body: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to compress request body (during close): %w", err)
	}
	return buf, nil
}

func (pg *gzipCompressor) collectBuffer(buf *bytes.Buffer) {
	pg.bufferPool.Put(buf)
}
