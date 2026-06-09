package proto

import (
	"encoding/binary"
	"math"
)

// Int256 is 256-bit signed integer.
type Int256 struct {
	Low  UInt128 // first 128 bits
	High UInt128 // last 128 bits
}

// Int256FromInt creates new Int256 from int.
func Int256FromInt(v int) Int256 {
	var hi UInt128
	lo := UInt128{Low: uint64(v)}
	if v < 0 {
		hi = UInt128{
			Low:  math.MaxUint64,
			High: math.MaxUint64,
		}
		lo.High = math.MaxUint64
	}
	return Int256{
		High: hi,
		Low:  lo,
	}
}

// UInt256 is 256-bit unsigned integer.
type UInt256 struct {
	Low  UInt128 // first 128 bits
	High UInt128 // last 128 bits
}

// UInt256FromInt creates new UInt256 from int.
func UInt256FromInt(v int) UInt256 {
	return UInt256(Int256FromInt(v))
}

// UInt256FromUInt64 creates new UInt256 from uint64.
func UInt256FromUInt64(v uint64) UInt256 {
	return UInt256{Low: UInt128{Low: v}}
}

func binUInt256(b []byte) UInt256 {
	_ = b[:256/8] // bounds check hint to compiler; see golang.org/issue/14808
	// Calling manually because binUInt128 is not inlining.
	return UInt256{
		Low: UInt128{
			Low:  binary.LittleEndian.Uint64(b[0 : 64/8]),
			High: binary.LittleEndian.Uint64(b[64/8 : 128/8]),
		},
		High: UInt128{
			Low:  binary.LittleEndian.Uint64(b[128/8 : 192/8]),
			High: binary.LittleEndian.Uint64(b[192/8 : 256/8]),
		},
	}
}

func binPutUInt256(b []byte, v UInt256) {
	// Calling manually because binPutUInt128 is not inlining.
	binary.LittleEndian.PutUint64(b[192/8:256/8], v.High.High)
	binary.LittleEndian.PutUint64(b[128/8:192/8], v.High.Low)
	binary.LittleEndian.PutUint64(b[64/8:128/8], v.Low.High)
	binary.LittleEndian.PutUint64(b[0:64/8], v.Low.Low)
}
