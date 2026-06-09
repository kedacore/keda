package compress

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/go-faster/city"
	"github.com/go-faster/errors"
	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4/v4"
)

// Reader decodes compressed blocks.
type Reader struct {
	reader io.Reader
	data   []byte
	pos    int64
	raw    []byte
	header []byte
	zstd   *zstd.Decoder
}

// FormatU128 formats city.U128 as hex.
func FormatU128(v city.U128) string {
	var buf [16]byte
	binary.LittleEndian.PutUint64(buf[:8], v.Low)
	binary.LittleEndian.PutUint64(buf[8:], v.High)
	return fmt.Sprintf("%x", buf)
}

// readBlock reads next compressed data into raw and decompresses into data.
func (r *Reader) readBlock() error {
	r.pos = 0

	_ = r.header[headerSize-1]
	if _, err := io.ReadFull(r.reader, r.header); err != nil {
		return errors.Wrap(err, "header")
	}

	var (
		rawSize  = int(binary.LittleEndian.Uint32(r.header[hRawSize:])) - compressHeaderSize
		dataSize = int(binary.LittleEndian.Uint32(r.header[hDataSize:]))
	)
	if dataSize < 0 || dataSize > maxDataSize {
		return errors.Errorf("data size should be %d < %d < %d", 0, dataSize, maxDataSize)
	}
	if rawSize < 0 || rawSize > maxBlockSize {
		return errors.Errorf("raw size should be %d < %d < %d", 0, rawSize, maxBlockSize)
	}

	r.data = append(r.data[:0], make([]byte, dataSize)...)
	r.raw = append(r.raw[:0], r.header...)
	r.raw = append(r.raw, make([]byte, rawSize)...)
	_ = r.raw[:rawSize+headerSize-1]

	if _, err := io.ReadFull(r.reader, r.raw[headerSize:]); err != nil {
		return errors.Wrap(err, "read raw")
	}
	hGot := city.U128{
		Low:  binary.LittleEndian.Uint64(r.raw[0:8]),
		High: binary.LittleEndian.Uint64(r.raw[8:16]),
	}
	h := city.CH128(r.raw[hMethod:])
	if hGot != h {
		return errors.Wrap(&CorruptedDataErr{
			Actual:    h,
			Reference: hGot,
			RawSize:   rawSize,
			DataSize:  dataSize,
		}, "mismatch")
	}
	switch m := methodEncoding(r.header[hMethod]); m {
	case encodedLZ4: // == encodedLZ4HC, as decompression is similar for both
		n, err := lz4.UncompressBlock(r.raw[headerSize:], r.data)
		if err != nil {
			return errors.Wrap(err, "uncompress")
		}
		if n != dataSize {
			return errors.Errorf("unexpected uncompressed data size: %d (actual) != %d (got in header)",
				n, dataSize,
			)
		}
	case encodedZSTD:
		if r.zstd == nil {
			// Lazily initializing to prevent spawning goroutines in NewReader.
			// See https://github.com/golang/go/issues/47056#issuecomment-997436820
			zstdReader, err := zstd.NewReader(nil,
				zstd.WithDecoderConcurrency(1),
				zstd.WithDecoderLowmem(true),
			)
			if err != nil {
				return errors.Wrap(err, "zstd")
			}
			r.zstd = zstdReader
		}
		data, err := r.zstd.DecodeAll(r.raw[headerSize:], r.data[:0])
		if err != nil {
			return errors.Wrap(err, "uncompress")
		}
		if len(data) != dataSize {
			return errors.Errorf("unexpected uncompressed data size: %d (actual) != %d (got in header)",
				len(data), dataSize,
			)
		}
		r.data = data
	case encodedNone:
		copy(r.data, r.raw[headerSize:])
	default:
		return errors.Errorf("compression 0x%02x not implemented", m)
	}

	return nil
}

// Read implements io.Reader.
func (r *Reader) Read(p []byte) (n int, err error) {
	if r.pos >= int64(len(r.data)) {
		if err := r.readBlock(); err != nil {
			return 0, errors.Wrap(err, "read next block")
		}
	}
	n = copy(p, r.data[r.pos:])
	r.pos += int64(n)
	return n, nil
}

// NewReader returns new *Reader from r.
func NewReader(r io.Reader) *Reader {
	return &Reader{
		zstd:   nil, // lazily initialized
		reader: r,
		header: make([]byte, headerSize),
	}
}
