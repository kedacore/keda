package proto

import "math"

// Float32ToBFloat16 converts a float32 to BFloat16 (Brain Floating Point) format
// using round-to-nearest-even (banker's rounding) for unbiased results.
//
// BFloat16 is a 16-bit floating point format with:
//   - 1 sign bit
//   - 8 exponent bits (same as Float32)
//   - 7 mantissa bits (Float32 has 23)
//
// This function takes the upper 16 bits of Float32 with proper rounding.
//
// What is banker's rounding?
// It is rounding-to-nearest-even in case of numbers that are exactly in half-way midpoint.
//
// Without rounding-to-nearest-even, we have systematic bias:
//   - Round-up always:   3.5->4, 4.5->5, 5.5->6, 6.5->7 (net +2.0 bias)
//   - Round-down always: 3.5->3, 4.5->4, 5.5->5, 6.5->6 (net -2.0 bias)
//   - Round-to-even:     3.5->4, 4.5->4, 5.5->6, 6.5->6 (net 0 bias)
//
// See: https://en.wikipedia.org/wiki/Rounding#Round_half_to_even
func Float32ToBFloat16(v float32) uint16 {
	bits := math.Float32bits(v)

	// halfway is the threshold for rounding (0x7FFF = 0111111111111111)
	// It represents the midpoint between two representable BFloat16 values
	halfway := uint32(0x7FFF)

	// evenness is the LSB of the upper 16 bits
	// If 1 (odd), we add 1 to roundingBias to round up
	// If 0 (even), we keep roundingBias as halfway (rounds down)
	evenness := (bits >> 16) & 1

	// Apply banker's rounding: always round to the nearest even number
	roundingBias := halfway + evenness
	bits += roundingBias

	// Extract upper 16 bits
	return uint16(bits >> 16)
}

// BFloat16ToFloat32 converts a BFloat16 value back to float32.
// BFloat16 is stored in the upper 16 bits of a float32.
func BFloat16ToFloat32(v uint16) float32 {
	bits := uint32(v) << 16
	return math.Float32frombits(bits)
}
