//go:build !amd64 && !arm64
// +build !amd64,!arm64

package binary

func Str2Bytes(str string, expectedLen int) []byte {
	b := []byte(str)
	if len(str) < expectedLen {
		extended := make([]byte, expectedLen)
		copy(extended, b)
		return extended
	}

	return b
}
