package binary

import (
	"unsafe"
)

func unsafeStr2Bytes(str string) []byte {
	return unsafe.Slice(unsafe.StringData(str), len(str))
}
