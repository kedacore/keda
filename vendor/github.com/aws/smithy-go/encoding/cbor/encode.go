package cbor

import (
	"encoding/binary"
	"math"
)

func (i Uint) len() int {
	return itoarglen(uint64(i))
}

func (i Uint) encode(p []byte) int {
	return encodeArg(majorTypeUint, uint64(i), p)
}

func (i NegInt) len() int {
	return itoarglen(uint64(i) - 1)
}

func (i NegInt) encode(p []byte) int {
	return encodeArg(majorTypeNegInt, uint64(i-1), p)
}

func (s Slice) len() int {
	return itoarglen(len(s)) + len(s)
}

func (s Slice) encode(p []byte) int {
	off := encodeArg(majorTypeSlice, len(s), p)
	copy(p[off:], []byte(s))
	return off + len(s)
}

func (s String) len() int {
	return itoarglen(len(s)) + len(s)
}

func (s String) encode(p []byte) int {
	off := encodeArg(majorTypeString, len(s), p)
	copy(p[off:], []byte(s))
	return off + len(s)
}

func (l List) len() int {
	total := itoarglen(len(l))
	for _, v := range l {
		total += v.len()
	}
	return total
}

func (l List) encode(p []byte) int {
	off := encodeArg(majorTypeList, len(l), p)
	for _, v := range l {
		off += v.encode(p[off:])
	}
	return off
}

func (m Map) len() int {
	total := itoarglen(len(m))
	for k, v := range m {
		total += String(k).len() + v.len()
	}
	return total
}

func (m Map) encode(p []byte) int {
	off := encodeArg(majorTypeMap, len(m), p)
	for k, v := range m {
		off += String(k).encode(p[off:])
		off += v.encode(p[off:])
	}
	return off
}

func (t Tag) len() int {
	return itoarglen(t.ID) + t.Value.len()
}

func (t Tag) encode(p []byte) int {
	off := encodeArg(majorTypeTag, t.ID, p)
	return off + t.Value.encode(p[off:])
}

func (b Bool) len() int {
	return 1
}

func (b Bool) encode(p []byte) int {
	if b {
		p[0] = compose(majorType7, major7True)
	} else {
		p[0] = compose(majorType7, major7False)
	}
	return 1
}

func (*Nil) len() int {
	return 1
}

func (*Nil) encode(p []byte) int {
	p[0] = compose(majorType7, major7Nil)
	return 1
}

func (*Undefined) len() int {
	return 1
}

func (*Undefined) encode(p []byte) int {
	p[0] = compose(majorType7, major7Undefined)
	return 1
}

func (f Float32) len() int {
	return 5
}

func (f Float32) encode(p []byte) int {
	p[0] = compose(majorType7, major7Float32)
	binary.BigEndian.PutUint32(p[1:], math.Float32bits(float32(f)))
	return 5
}

func (f Float64) len() int {
	return 9
}

func (f Float64) encode(p []byte) int {
	p[0] = compose(majorType7, major7Float64)
	binary.BigEndian.PutUint64(p[1:], math.Float64bits(float64(f)))
	return 9
}

func compose(major majorType, minor byte) byte {
	return byte(major)<<5 | minor
}

func itoarglen[I int | uint64](v I) int {
	vv := uint64(v)
	if vv < 24 {
		return 1 // type and len in single byte
	} else if vv < 0x100 {
		return 2 // type + 1-byte len
	} else if vv < 0x10000 {
		return 3 // type + 2-byte len
	} else if vv < 0x100000000 {
		return 5 // type + 4-byte len
	}
	return 9 // type + 8-byte len
}

func encodeArg[I int | uint64](t majorType, arg I, p []byte) int {
	aarg := uint64(arg)
	if aarg < 24 {
		p[0] = byte(t)<<5 | byte(aarg)
		return 1
	} else if aarg < 0x100 {
		p[0] = compose(t, minorArg1)
		p[1] = byte(aarg)
		return 2
	} else if aarg < 0x10000 {
		p[0] = compose(t, minorArg2)
		binary.BigEndian.PutUint16(p[1:], uint16(aarg))
		return 3
	} else if aarg < 0x100000000 {
		p[0] = compose(t, minorArg4)
		binary.BigEndian.PutUint32(p[1:], uint32(aarg))
		return 5
	}

	p[0] = compose(t, minorArg8)
	binary.BigEndian.PutUint64(p[1:], uint64(aarg))
	return 9
}

// EncodeRaw encodes opaque CBOR data.
//
// This is used by the encoder for the purpose of embedding document shapes.
// Decode will never return values of this type.
type EncodeRaw []byte

func (v EncodeRaw) len() int { return len(v) }

func (v EncodeRaw) encode(p []byte) int {
	copy(p, v)
	return len(v)
}

// FixedUint encodes fixed-width Uint values.
//
// This is used by the encoder for the purpose of embedding integrals in
// document shapes. Decode will never return values of this type.
type EncodeFixedUint uint64

func (EncodeFixedUint) len() int { return 9 }

func (v EncodeFixedUint) encode(p []byte) int {
	p[0] = compose(majorTypeUint, minorArg8)
	binary.BigEndian.PutUint64(p[1:], uint64(v))
	return 9
}

// FixedUint encodes fixed-width NegInt values.
//
// This is used by the encoder for the purpose of embedding integrals in
// document shapes. Decode will never return values of this type.
type EncodeFixedNegInt uint64

func (EncodeFixedNegInt) len() int { return 9 }

func (v EncodeFixedNegInt) encode(p []byte) int {
	p[0] = compose(majorTypeNegInt, minorArg8)
	binary.BigEndian.PutUint64(p[1:], uint64(v-1))
	return 9
}
