package cbor

import (
	"encoding/binary"
	"fmt"
	"math"
)

func decode(p []byte) (Value, int, error) {
	if len(p) == 0 {
		return nil, 0, fmt.Errorf("unexpected end of payload")
	}

	switch peekMajor(p) {
	case majorTypeUint:
		return decodeUint(p)
	case majorTypeNegInt:
		return decodeNegInt(p)
	case majorTypeSlice:
		return decodeSlice(p, majorTypeSlice)
	case majorTypeString:
		s, n, err := decodeSlice(p, majorTypeString)
		return String(s), n, err
	case majorTypeList:
		return decodeList(p)
	case majorTypeMap:
		return decodeMap(p)
	case majorTypeTag:
		return decodeTag(p)
	default: // majorType7
		return decodeMajor7(p)
	}
}

func decodeUint(p []byte) (Uint, int, error) {
	i, off, err := decodeArgument(p)
	if err != nil {
		return 0, 0, fmt.Errorf("decode argument: %w", err)
	}

	return Uint(i), off, nil
}

func decodeNegInt(p []byte) (NegInt, int, error) {
	i, off, err := decodeArgument(p)
	if err != nil {
		return 0, 0, fmt.Errorf("decode argument: %w", err)
	}

	return NegInt(i + 1), off, nil
}

// this routine is used for both string and slice major types, the value of
// inner specifies which context we're in (needed for validating subsegments
// inside indefinite encodings)
func decodeSlice(p []byte, inner majorType) (Slice, int, error) {
	minor := peekMinor(p)
	if minor == minorIndefinite {
		return decodeSliceIndefinite(p, inner)
	}

	slen, off, err := decodeArgument(p)
	if err != nil {
		return nil, 0, fmt.Errorf("decode argument: %w", err)
	}

	p = p[off:]
	if uint64(len(p)) < slen {
		return nil, 0, fmt.Errorf("slice len %d greater than remaining buf len", slen)
	}

	return Slice(p[:slen]), off + int(slen), nil
}

func decodeSliceIndefinite(p []byte, inner majorType) (Slice, int, error) {
	p = p[1:]

	s := Slice{}
	for off := 0; len(p) > 0; {
		if p[0] == 0xff {
			return s, off + 2, nil
		}

		if major := peekMajor(p); major != inner {
			return nil, 0, fmt.Errorf("unexpected major type %d in indefinite slice", major)
		}
		if peekMinor(p) == minorIndefinite {
			return nil, 0, fmt.Errorf("nested indefinite slice")
		}

		ss, n, err := decodeSlice(p, inner)
		if err != nil {
			return nil, 0, fmt.Errorf("decode subslice: %w", err)
		}
		p = p[n:]

		s = append(s, ss...)
		off += n
	}
	return nil, 0, fmt.Errorf("expected break marker")
}

func decodeList(p []byte) (List, int, error) {
	minor := peekMinor(p)
	if minor == minorIndefinite {
		return decodeListIndefinite(p)
	}

	alen, off, err := decodeArgument(p)
	if err != nil {
		return nil, 0, fmt.Errorf("decode argument: %w", err)
	}
	p = p[off:]

	l := List{}
	for i := 0; i < int(alen); i++ {
		item, n, err := decode(p)
		if err != nil {
			return nil, 0, fmt.Errorf("decode item: %w", err)
		}
		p = p[n:]

		l = append(l, item)
		off += n
	}

	return l, off, nil
}

func decodeListIndefinite(p []byte) (List, int, error) {
	p = p[1:]

	l := List{}
	for off := 0; len(p) > 0; {
		if p[0] == 0xff {
			return l, off + 2, nil
		}

		item, n, err := decode(p)
		if err != nil {
			return nil, 0, fmt.Errorf("decode item: %w", err)
		}
		p = p[n:]

		l = append(l, item)
		off += n
	}
	return nil, 0, fmt.Errorf("expected break marker")
}

func decodeMap(p []byte) (Map, int, error) {
	minor := peekMinor(p)
	if minor == minorIndefinite {
		return decodeMapIndefinite(p)
	}

	maplen, off, err := decodeArgument(p)
	if err != nil {
		return nil, 0, fmt.Errorf("decode argument: %w", err)
	}
	p = p[off:]

	mp := Map{}
	for i := 0; i < int(maplen); i++ {
		if len(p) == 0 {
			return nil, 0, fmt.Errorf("unexpected end of payload")
		}

		if major := peekMajor(p); major != majorTypeString {
			return nil, 0, fmt.Errorf("unexpected major type %d for map key", major)
		}

		key, kn, err := decodeSlice(p, majorTypeString)
		if err != nil {
			return nil, 0, fmt.Errorf("decode key: %w", err)
		}
		p = p[kn:]

		value, vn, err := decode(p)
		if err != nil {
			return nil, 0, fmt.Errorf("decode value: %w", err)
		}
		p = p[vn:]

		mp[string(key)] = value
		off += kn + vn
	}

	return mp, off, nil
}

func decodeMapIndefinite(p []byte) (Map, int, error) {
	p = p[1:]

	mp := Map{}
	for off := 0; len(p) > 0; {
		if p[0] == 0xff {
			return mp, off + 2, nil
		}

		if major := peekMajor(p); major != majorTypeString {
			return nil, 0, fmt.Errorf("unexpected major type %d for map key", major)
		}

		key, kn, err := decodeSlice(p, majorTypeString)
		if err != nil {
			return nil, 0, fmt.Errorf("decode key: %w", err)
		}
		p = p[kn:]

		value, vn, err := decode(p)
		if err != nil {
			return nil, 0, fmt.Errorf("decode value: %w", err)
		}
		p = p[vn:]

		mp[string(key)] = value
		off += kn + vn
	}
	return nil, 0, fmt.Errorf("expected break marker")
}

func decodeTag(p []byte) (*Tag, int, error) {
	id, off, err := decodeArgument(p)
	if err != nil {
		return nil, 0, fmt.Errorf("decode argument: %w", err)
	}
	p = p[off:]

	v, n, err := decode(p)
	if err != nil {
		return nil, 0, fmt.Errorf("decode value: %w", err)
	}

	return &Tag{ID: id, Value: v}, off + n, nil
}

func decodeMajor7(p []byte) (Value, int, error) {
	switch m := peekMinor(p); m {
	case major7True, major7False:
		return Bool(m == major7True), 1, nil
	case major7Nil:
		return &Nil{}, 1, nil
	case major7Undefined:
		return &Undefined{}, 1, nil
	case major7Float16:
		if len(p) < 3 {
			return nil, 0, fmt.Errorf("incomplete float16 at end of buf")
		}
		b := binary.BigEndian.Uint16(p[1:])
		return Float32(math.Float32frombits(float16to32(b))), 3, nil
	case major7Float32:
		if len(p) < 5 {
			return nil, 0, fmt.Errorf("incomplete float32 at end of buf")
		}
		b := binary.BigEndian.Uint32(p[1:])
		return Float32(math.Float32frombits(b)), 5, nil
	case major7Float64:
		if len(p) < 9 {
			return nil, 0, fmt.Errorf("incomplete float64 at end of buf")
		}
		b := binary.BigEndian.Uint64(p[1:])
		return Float64(math.Float64frombits(b)), 9, nil
	default:
		return nil, 0, fmt.Errorf("unexpected minor value %d", m)
	}
}

func peekMajor(p []byte) majorType {
	return majorType(p[0] & maskMajor >> 5)
}

func peekMinor(p []byte) byte {
	return p[0] & maskMinor
}

// pulls the next argument out of the buffer
//
// expects one of the sized arguments and will error otherwise - callers that
// need to check for the indefinite flag must do so externally
func decodeArgument(p []byte) (uint64, int, error) {
	minor := peekMinor(p)
	if minor < minorArg1 {
		return uint64(minor), 1, nil
	}

	switch minor {
	case minorArg1, minorArg2, minorArg4, minorArg8:
		argLen := mtol(minor)
		if len(p) < argLen+1 {
			return 0, 0, fmt.Errorf("arg len %d greater than remaining buf len", argLen)
		}
		return readArgument(p[1:], argLen), argLen + 1, nil
	default:
		return 0, 0, fmt.Errorf("unexpected minor value %d", minor)
	}
}

// minor value to arg len in bytes, assumes minor was checked to be in [24,27]
func mtol(minor byte) int {
	if minor == minorArg1 {
		return 1
	} else if minor == minorArg2 {
		return 2
	} else if minor == minorArg4 {
		return 4
	}
	return 8
}

func readArgument(p []byte, len int) uint64 {
	if len == 1 {
		return uint64(p[0])
	} else if len == 2 {
		return uint64(binary.BigEndian.Uint16(p))
	} else if len == 4 {
		return uint64(binary.BigEndian.Uint32(p))
	}
	return uint64(binary.BigEndian.Uint64(p))
}
