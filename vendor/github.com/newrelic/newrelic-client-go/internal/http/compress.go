package http

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"io"
)

const (
	// CompressorMinimumSize is the required size in bytes that a body exceeds before it is compressed
	CompressorMinimumSize = 150
)

type RequestCompressor interface {
	Compress(r *Request, body []byte) (io.Reader, error)
}

// NoneCompressor does not compress the request at all.
type NoneCompressor struct{}

// CompressRequest returns a reader for the body to the caller
func (c *NoneCompressor) Compress(r *Request, body []byte) (io.Reader, error) {
	buffer := bytes.NewBuffer(body)
	r.DelHeader("Content-Encoding")

	return buffer, nil
}

// GzipCompressor compresses the body with gzip
type GzipCompressor struct{}

// CompressRequest gzips the body, sets the content encoding header, and returns a reader to the caller
func (c *GzipCompressor) Compress(r *Request, body []byte) (io.Reader, error) {
	var err error

	// Adaptive compression
	if len(body) < CompressorMinimumSize {
		buf := bytes.NewBuffer(body)
		r.DelHeader("Content-Encoding")

		return buf, nil
	}

	readBuffer := bufio.NewReader(bytes.NewReader(body))
	buffer := bytes.NewBuffer([]byte{})
	writer := gzip.NewWriter(buffer)

	_, err = readBuffer.WriteTo(writer)
	if err != nil {
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	r.SetHeader("Content-Encoding", "gzip")

	return buffer, nil
}
