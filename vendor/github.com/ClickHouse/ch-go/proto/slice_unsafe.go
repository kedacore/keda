//go:build (amd64 || arm64 || riscv64) && !purego

package proto

import "unsafe"

// slice represents slice header.
//
// Used in optimizations when we can interpret [N]T as [M]byte, where
// M = sizeof(T) * N.
//
// NB: careful with endianness!
type slice struct {
	Data unsafe.Pointer
	Len  uintptr
	Cap  uintptr
}
