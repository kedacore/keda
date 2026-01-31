package compress

import (
	"encoding/binary"

	"github.com/go-faster/city"
	"github.com/go-faster/errors"
	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4/v4"
)

// Writer encodes compressed blocks.
type Writer struct {
	Data []byte

	lz4  *lz4.Compressor
	zstd *zstd.Encoder
}

// Compress buf into Data.
func (w *Writer) Compress(m Method, buf []byte) error {
	maxSize := lz4.CompressBlockBound(len(buf))
	w.Data = append(w.Data[:0], make([]byte, maxSize+headerSize)...)
	_ = w.Data[:headerSize]
	w.Data[hMethod] = byte(m)

	var n int

	switch m {
	case LZ4:
		compressedSize, err := w.lz4.CompressBlock(buf, w.Data[headerSize:])
		if err != nil {
			return errors.Wrap(err, "block")
		}
		n = compressedSize
	case ZSTD:
		w.Data = w.zstd.EncodeAll(buf, w.Data[:headerSize])
		n = len(w.Data) - headerSize
	case None:
		n = copy(w.Data[headerSize:], buf)
	}

	w.Data = w.Data[:n+headerSize]

	binary.LittleEndian.PutUint32(w.Data[hRawSize:], uint32(n+compressHeaderSize))
	binary.LittleEndian.PutUint32(w.Data[hDataSize:], uint32(len(buf)))
	h := city.CH128(w.Data[hMethod:])
	binary.LittleEndian.PutUint64(w.Data[0:8], h.Low)
	binary.LittleEndian.PutUint64(w.Data[8:16], h.High)

	return nil
}

func NewWriter() *Writer {
	w, err := zstd.NewWriter(nil,
		zstd.WithEncoderLevel(zstd.SpeedDefault),
		zstd.WithEncoderConcurrency(1),
		zstd.WithLowerEncoderMem(true),
	)
	if err != nil {
		panic(err)
	}
	return &Writer{
		lz4:  &lz4.Compressor{},
		zstd: w,
	}
}
