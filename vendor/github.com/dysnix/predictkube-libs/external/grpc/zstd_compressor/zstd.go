// Package zstd is a wrapper for using github.com/klauspost/compress/zstd
// with gRPC.
package zstd_compressor

import (
	"bytes"
	"io"
	"io/ioutil"

	"github.com/klauspost/compress/zstd"
	"google.golang.org/grpc/encoding"
)

const Name = "zstd"

type compressor struct {
	encoder *zstd.Encoder
	decoder *zstd.Decoder
}

func init() {
	enc, _ := zstd.NewWriter(nil)
	dec, _ := zstd.NewReader(nil)
	c := &compressor{
		encoder: enc,
		decoder: dec,
	}
	encoding.RegisterCompressor(c)
}

// SetLevel updates the registered compressor to use a particular compression
// level. NOTE: this function must only be called from an init function, and
// is not threadsafe.
func SetLevel(level zstd.EncoderLevel) error {
	c := encoding.GetCompressor(Name).(*compressor)

	enc, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(level))
	if err != nil {
		return err
	}

	c.encoder = enc
	return nil
}

func (c *compressor) Compress(w io.Writer) (io.WriteCloser, error) {
	return &zstdWriteCloser{
		enc:    c.encoder,
		writer: w,
	}, nil
}

type zstdWriteCloser struct {
	enc    *zstd.Encoder
	writer io.Writer    // Compressed data will be written here.
	buf    bytes.Buffer // Buffer uncompressed data here, compress on Close.
}

func (z *zstdWriteCloser) Write(p []byte) (int, error) {
	return z.buf.Write(p)
}

func (z *zstdWriteCloser) Close() error {
	compressed := z.enc.EncodeAll(z.buf.Bytes(), nil)
	_, err := io.Copy(z.writer, bytes.NewReader(compressed))
	return err
}

func (c *compressor) Decompress(r io.Reader) (io.Reader, error) {
	compressed, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	uncompressed, err := c.decoder.DecodeAll(compressed, nil)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(uncompressed), nil
}

func (c *compressor) Name() string {
	return Name
}
