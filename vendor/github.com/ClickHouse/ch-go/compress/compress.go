// Package compress implements compression support.
package compress

import (
	"fmt"

	"github.com/go-faster/city"
)

//go:generate go run github.com/dmarkham/enumer -transform upper -type Method -output method_enum.go

// Method is compression codec.
type Method byte

// Possible compression methods.
const (
	None Method = iota
	LZ4
	LZ4HC
	ZSTD
	NumMethods int = iota
)

type methodEncoding byte

const (
	encodedNone  methodEncoding = 0x02
	encodedLZ4   methodEncoding = 0x82
	encodedLZ4HC methodEncoding = encodedLZ4
	encodedZSTD  methodEncoding = 0x90
)

var methodTable = map[Method]methodEncoding{
	None:  encodedNone,
	LZ4:   encodedLZ4,
	LZ4HC: encodedLZ4HC,
	ZSTD:  encodedZSTD,
}

// Level for supporting compression codecs.
type Level uint32

// Constants for compression encoding.
//
// See https://go-faster.org/docs/clickhouse/compression for reference.
const (
	checksumSize       = 16
	compressHeaderSize = 1 + 4 + 4
	headerSize         = checksumSize + compressHeaderSize

	// Limiting total data/block size to protect from possible OOM.
	maxDataSize  = 1024 * 1024 * 128 // 128MB
	maxBlockSize = maxDataSize

	hRawSize  = 17
	hDataSize = 21
	hMethod   = 16
)

// CorruptedDataErr means that provided hash mismatch with calculated.
type CorruptedDataErr struct {
	Actual    city.U128
	Reference city.U128
	RawSize   int
	DataSize  int
}

func (c *CorruptedDataErr) Error() string {
	return fmt.Sprintf("corrupted data: %s (actual), %s (reference), compressed size: %d, data size: %d",
		FormatU128(c.Actual), FormatU128(c.Reference), c.RawSize, c.DataSize,
	)
}
