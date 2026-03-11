package cbor

import (
	"fmt"
	"math/big"
	"time"
)

func fmtNegint(v NegInt) string {
	if v == 0 {
		return "-2^64"
	}
	return fmt.Sprintf("-%d", v)
}

// AsInt8 coerces a Value to its int8 representation if possible.
func AsInt8(v Value) (int8, error) {
	const max8 = 0x7f

	switch vv := v.(type) {
	case Uint:
		if vv > max8 {
			return 0, fmt.Errorf("cbor uint %d exceeds max int8 value", vv)
		}
		return int8(vv), nil
	case NegInt:
		if vv > max8+1 || vv == 0 {
			return 0, fmt.Errorf("cbor negint %s exceeds min int8 value", fmtNegint(vv))
		}
		return -int8(vv), nil
	}
	return 0, fmt.Errorf("unexpected value type %T", v)
}

// AsInt16 coerces a Value to its int16 representation if possible.
func AsInt16(v Value) (int16, error) {
	const max16 = 0x7fff

	switch vv := v.(type) {
	case Uint:
		if vv > max16 {
			return 0, fmt.Errorf("cbor uint %d exceeds max int16 value", vv)
		}
		return int16(vv), nil
	case NegInt:
		if vv > max16+1 || vv == 0 {
			return 0, fmt.Errorf("cbor negint %s exceeds min int16 value", fmtNegint(vv))
		}
		return -int16(vv), nil
	}
	return 0, fmt.Errorf("unexpected value type %T", v)
}

// AsInt32 coerces a Value to its int32 representation if possible.
func AsInt32(v Value) (int32, error) {
	const max32 = 0x7fffffff

	switch vv := v.(type) {
	case Uint:
		if vv > max32 {
			return 0, fmt.Errorf("cbor uint %d exceeds max int32 value", vv)
		}
		return int32(vv), nil
	case NegInt:
		if vv > max32+1 || vv == 0 {
			return 0, fmt.Errorf("cbor negint %s exceeds min int32 value", fmtNegint(vv))
		}
		return -int32(vv), nil
	}
	return 0, fmt.Errorf("unexpected value type %T", v)
}

// AsInt64 coerces a Value to its int64 representation if possible.
func AsInt64(v Value) (int64, error) {
	const max64 = 0x7fffffff_ffffffff

	switch vv := v.(type) {
	case Uint:
		if vv > max64 {
			return 0, fmt.Errorf("cbor uint %d exceeds max int64 value", vv)
		}
		return int64(vv), nil
	case NegInt:
		if vv > max64+1 || vv == 0 {
			return 0, fmt.Errorf("cbor negint %s exceeds min int64 value", fmtNegint(vv))
		}
		return -int64(vv), nil
	}
	return 0, fmt.Errorf("unexpected value type %T", v)
}

// AsFloat32 coerces a Value to its float32 representation if possible.
//
// A float32 may be represented by any of the following alternatives:
//   - cbor uint (if within lossless range)
//   - cbor -int (if within lossless range)
func AsFloat32(v Value) (float32, error) {
	const maxLosslessFloat32 = 1 << 24

	switch vv := v.(type) {
	case Float32:
		return float32(vv), nil
	case Uint:
		if vv > maxLosslessFloat32 {
			return 0, fmt.Errorf("cbor uint %d exceeds max lossless float32 value", vv)
		}
		return float32(vv), nil
	case NegInt:
		if vv > maxLosslessFloat32 || vv == 0 {
			return 0, fmt.Errorf("cbor negint %s exceeds min lossless float32 value", fmtNegint(vv))
		}
		return -float32(vv), nil
	}
	return 0, fmt.Errorf("unexpected value type %T", v)
}

// AsFloat64 coerces a Value to its float64 representation if possible.
//
// A float64 may be represented by any of the following alternatives:
//   - float32
//   - cbor uint (if within lossless range)
//   - cbor -int (if within lossless range)
func AsFloat64(v Value) (float64, error) {
	const maxLosslessFloat64 = 1 << 54

	switch vv := v.(type) {
	case Float64:
		return float64(vv), nil
	case Float32:
		return float64(vv), nil
	case Uint:
		if vv > maxLosslessFloat64 {
			return 0, fmt.Errorf("cbor uint %d exceeds max lossless float64 value", vv)
		}
		return float64(vv), nil
	case NegInt:
		if vv > maxLosslessFloat64 || vv == 0 {
			return 0, fmt.Errorf("cbor negint %s exceeds min lossless float64 value", fmtNegint(vv))
		}
		return -float64(vv), nil
	}
	return 0, fmt.Errorf("unexpected value type %T", v)
}

// AsTime coerces a Value to its time.Time representation if possible.
//
// This coercion will check that the given Value is a Tag with the registered
// number (1) for epoch time. The value for time.Time within that tag may be
// derived from any of the following:
//   - float32
//   - float64
//   - cbor uint (within int64 bounds)
//   - cbor -int (within int64 bounds)
//
// Tag number 0 (date-time RFC3339) is not supported.
func AsTime(v Value) (time.Time, error) {
	const tagEpoch = 1

	tag, ok := v.(*Tag)
	if !ok {
		return time.Time{}, fmt.Errorf("unexpected value type %T", v)
	}
	if tag.ID != tagEpoch {
		return time.Time{}, fmt.Errorf("unexpected tag ID %d", tag.ID)
	}

	switch vv := tag.Value.(type) {
	case Float32:
		return time.UnixMilli(int64(vv * 1e3)), nil
	case Float64:
		return time.UnixMilli(int64(vv * 1e3)), nil
	}

	as64, err := AsInt64(tag.Value) // will handle fail on non-int types
	if err != nil {
		return time.Time{}, fmt.Errorf("coerce tag value: %w", err)
	}

	return time.Unix(as64, 0), nil
}

// AsBigInt coerces a Value to its big.Int representation if possible.
//
// A BigInt may be represented by any of the following:
//   - Uint
//   - NegInt
//   - Tag (type 2/3, where tagged value is a Slice)
//   - Nil
func AsBigInt(v Value) (*big.Int, error) {
	switch vv := v.(type) {
	case Uint:
		return new(big.Int).SetUint64(uint64(vv)), nil
	case NegInt:
		i := new(big.Int)
		if vv == 0 {
			i.SetBytes([]byte{1, 0, 0, 0, 0, 0, 0, 0, 0})
		} else {
			i.SetUint64(uint64(vv))
		}
		return i.Neg(i), nil
	case *Tag:
		return asBigIntFromTag(vv)
	case *Nil:
		return nil, nil
	default:
		return nil, fmt.Errorf("unexpected value type %T", v)
	}
}

func asBigIntFromTag(tv *Tag) (*big.Int, error) {
	const tagpos = 2
	const tagneg = 3

	if tv.ID != tagpos && tv.ID != tagneg {
		return nil, fmt.Errorf("unexpected tag ID %d", tv.ID)
	}

	bytes, ok := tv.Value.(Slice)
	if !ok {
		return nil, fmt.Errorf("unexpected tag value type %T", tv.Value)
	}

	i := new(big.Int).SetBytes([]byte(bytes))
	if tv.ID == tagneg {
		i.Sub(big.NewInt(-1), i)
	}

	return i, nil
}
