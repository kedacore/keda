package proto

import (
	"encoding/binary"
	"math"
)

// Int128 represents Int128 type.
type Int128 struct {
	Low  uint64 // first 64 bits
	High uint64 // last 64 bits
}

// Int value of Int128.
//
// Returns math.MaxInt if High is set.
func (i Int128) Int() int {
	switch i.High {
	case 0, math.MaxUint64:
		return int(i.Low)
	default:
		return math.MaxInt
	}
}

// UInt64 value of Int128.
func (i Int128) UInt64() uint64 {
	switch i.High {
	case 0, math.MaxUint64:
		return uint64(int(i.Low))
	default:
		return math.MaxUint64
	}
}

// Int128FromInt creates new Int128 from int.
func Int128FromInt(v int) Int128 {
	var hi uint64
	if v < 0 {
		hi = math.MaxUint64
	}
	return Int128{
		High: hi,
		Low:  uint64(v),
	}
}

// Int128FromUInt64 creates new Int128 from uint64.
func Int128FromUInt64(v uint64) Int128 {
	return Int128(UInt128FromUInt64(v))
}

// UInt128 represents UInt128 type.
type UInt128 struct {
	Low  uint64 // first 64 bits
	High uint64 // last 64 bits
}

// UInt64 returns UInt64 value of UInt128.
func (i UInt128) UInt64() uint64 {
	if i.High > 0 {
		return math.MaxUint64
	}
	return i.Low
}

// Int returns Int value of UInt128.
func (i UInt128) Int() int {
	return int(i.UInt64())
}

// UInt128FromInt creates new UInt128 from int.
func UInt128FromInt(v int) UInt128 {
	return UInt128(Int128FromInt(v))
}

// UInt128FromUInt64 creates new UInt128 from uint64.
func UInt128FromUInt64(v uint64) UInt128 {
	return UInt128{Low: v}
}

func binUInt128(b []byte) UInt128 {
	_ = b[:128/8] // bounds check hint to compiler; see golang.org/issue/14808
	return UInt128{
		Low:  binary.LittleEndian.Uint64(b[0 : 64/8]),
		High: binary.LittleEndian.Uint64(b[64/8 : 128/8]),
	}
}

func binPutUInt128(b []byte, v UInt128) {
	binary.LittleEndian.PutUint64(b[64/8:128/8], v.High)
	binary.LittleEndian.PutUint64(b[0:64/8], v.Low)
}
