package cbor

func float16to32(f uint16) uint32 {
	sign, exp, mant := splitf16(f)
	if exp == 0x1f {
		return sign | 0xff<<23 | mant // infinity/NaN
	}

	if exp == 0 { // subnormal
		if mant == 0 {
			return sign
		}
		return normalize(sign, mant)
	}

	return sign | (exp+127-15)<<23 | mant // rebias exp by the difference between the two
}

func splitf16(f uint16) (sign, exp, mantissa uint32) {
	const smask = 0x1 << 15  // put sign in float32 position
	const emask = 0x1f << 10 // pull exponent as a number (for bias shift)
	const mmask = 0x3ff      // put mantissa in float32 position

	return uint32(f&smask) << 16, uint32(f&emask) >> 10, uint32(f&mmask) << 13
}

// moves a float16 normal into normal float32 space
// to do this we must re-express the float16 mantissa in terms of a normal
// float32 where the hidden bit is 1, e.g.
//
// f16: 0    00000              0001010000 = 0.000101 * 2^(-14), which is equal to
// f32: 0 01101101 01000000000000000000000 =     1.01 * 2^(-18)
//
// this is achieved by shifting the mantissa to the right until the leading bit
// that == 1 reaches position 24, then the number of positions shifted over is
// equal to the offset from the subnormal exponent
func normalize(sign, mant uint32) uint32 {
	exp := uint32(-14 + 127) // f16 subnormal exp, with f32 bias
	for mant&0x800000 == 0 { // repeat until bit 24 ("hidden" mantissa) is 1
		mant <<= 1
		exp-- // tracking the offset
	}
	mant &= 0x7fffff // remask to 23bit
	return sign | exp<<23 | mant
}
