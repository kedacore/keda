package lineprotocol

import (
	"fmt"
	"reflect"
	"strconv"
	"unsafe"
)

// parseIntBytes is a zero-alloc wrapper around strconv.ParseInt.
func parseIntBytes(b []byte, base int, bitSize int) (i int64, err error) {
	return strconv.ParseInt(unsafeBytesToString(b), base, bitSize)
}

// parseUintBytes is a zero-alloc wrapper around strconv.ParseUint.
func parseUintBytes(b []byte, base int, bitSize int) (i uint64, err error) {
	return strconv.ParseUint(unsafeBytesToString(b), base, bitSize)
}

// parseFloatBytes is a zero-alloc wrapper around strconv.ParseFloat.
func parseFloatBytes(b []byte, bitSize int) (float64, error) {
	return strconv.ParseFloat(unsafeBytesToString(b), bitSize)
}

var errInvalidBool = fmt.Errorf("invalid boolean value")

// parseBoolBytes doesn't bother wrapping strconv.ParseBool because
// it's not quite the same, so simple and faster this way.
func parseBoolBytes(s []byte) (byte, error) {
	switch string(s) {
	case "t", "T", "true", "True", "TRUE":
		return 1, nil
	case "f", "F", "false", "False", "FALSE":
		return 0, nil
	}
	return 0, errInvalidBool
}

// unsafeBytesToString converts a []byte to a string without a heap allocation.
//
// It is unsafe, and is intended to prepare input to short-lived functions
// that require strings.
func unsafeBytesToString(data []byte) string {
	dataHeader := *(*reflect.SliceHeader)(unsafe.Pointer(&data))
	return *(*string)(unsafe.Pointer(&reflect.StringHeader{
		Data: dataHeader.Data,
		Len:  dataHeader.Len,
	}))
}
