package city

import "encoding/binary"

func bswap64(x uint64) uint64 {
	return ((x & 0xff00000000000000) >> 56) |
		((x & 0x00ff000000000000) >> 40) |
		((x & 0x0000ff0000000000) >> 24) |
		((x & 0x000000ff00000000) >> 8) |
		((x & 0x00000000ff000000) << 8) |
		((x & 0x0000000000ff0000) << 24) |
		((x & 0x000000000000ff00) << 40) |
		((x & 0x00000000000000ff) << 56)
}

// Bitwise right rotate.
func rot64(val uint64, shift uint) uint64 {
	// Avoid shifting by 64: doing so yields an undefined result.
	if shift == 0 {
		return val
	}
	return (val >> shift) | val<<(64-shift)
}

func shiftMix(val uint64) uint64 {
	return val ^ (val >> 47)
}

func hash128to64(x U128) uint64 {
	const mul = uint64(0x9ddfea08eb382d69)
	a := (x.Low ^ x.High) * mul
	a ^= a >> 47
	b := (x.High ^ a) * mul
	b ^= b >> 47
	b *= mul
	return b
}

func hash16(u, v uint64) uint64 {
	return hash128to64(U128{u, v})
}

func hash16mul(u, v, mul uint64) uint64 {
	// Murmur-inspired hashing.
	a := (u ^ v) * mul
	a ^= a >> 47
	b := (v ^ a) * mul
	b ^= b >> 47
	b *= mul
	return b
}

func hash0to16(s []byte, length int) uint64 {
	if length >= 8 {
		mul := k2 + uint64(length)*2
		a := binary.LittleEndian.Uint64(s) + k2
		b := binary.LittleEndian.Uint64(s[length-8:])
		c := rot64(b, 37)*mul + a
		d := (rot64(a, 25) + b) * mul
		return hash16mul(c, d, mul)
	}
	if length >= 4 {
		mul := k2 + uint64(length)*2
		a := uint64(fetch32(s))
		first := uint64(length) + (a << 3)
		second := uint64(fetch32(s[length-4:]))
		result := hash16mul(
			first,
			second,
			mul)
		return result
	}
	if length > 0 {
		a := s[0]
		b := s[length>>1]
		c := s[length-1]
		y := uint32(a) + (uint32(b) << 8)
		z := uint32(length) + (uint32(c) << 2)
		return shiftMix(uint64(y)*k2^uint64(z)*k0) * k2
	}
	return k2
}

// This probably works well for 16-byte strings as well, but is may be overkill
// in that case
func hash17to32(s []byte, length int) uint64 {
	mul := k2 + uint64(length)*2
	a := binary.LittleEndian.Uint64(s) * k1
	b := binary.LittleEndian.Uint64(s[8:])
	c := binary.LittleEndian.Uint64(s[length-8:]) * mul
	d := binary.LittleEndian.Uint64(s[length-16:]) * k2
	return hash16mul(
		rot64(a+b, 43)+rot64(c, 30)+d,
		a+rot64(b+k2, 18)+c,
		mul,
	)
}

// Return a 16-byte hash for 48 bytes. Quick and dirty.
// callers do best to use "random-looking" values for a and b.
func weakHash32Seeds(w, x, y, z, a, b uint64) U128 {
	a += w
	b = rot64(b+a+z, 21)
	c := a
	a += x
	a += y
	b += rot64(a, 44)
	return U128{a + z, b + c}
}

// Return a 16-byte hash for s[0] ... s[31], a, and b. Quick and dirty.
func weakHash32SeedsByte(s []byte, a, b uint64) U128 {
	_ = s[31]
	return weakHash32Seeds(
		binary.LittleEndian.Uint64(s[0:0+8:0+8]),
		binary.LittleEndian.Uint64(s[8:8+8:8+8]),
		binary.LittleEndian.Uint64(s[16:16+8:16+8]),
		binary.LittleEndian.Uint64(s[24:24+8:24+8]),
		a,
		b,
	)
}

// Return an 8-byte hash for 33 to 64 bytes.
func hash33to64(s []byte, length int) uint64 {
	mul := k2 + uint64(length)*2
	a := binary.LittleEndian.Uint64(s) * k2
	b := binary.LittleEndian.Uint64(s[8:])
	c := binary.LittleEndian.Uint64(s[length-24:])
	d := binary.LittleEndian.Uint64(s[length-32:])
	e := binary.LittleEndian.Uint64(s[16:]) * k2
	f := binary.LittleEndian.Uint64(s[24:]) * 9
	g := binary.LittleEndian.Uint64(s[length-8:])
	h := binary.LittleEndian.Uint64(s[length-16:]) * mul
	u := rot64(a+g, 43) + (rot64(b, 30)+c)*9
	v := ((a + g) ^ d) + f + 1
	w := bswap64((u+v)*mul) + h
	x := rot64(e+f, 42) + c
	y := (bswap64((v+w)*mul) + g) * mul
	z := e + f + c
	a = bswap64((x+z)*mul+y) + b
	b = shiftMix((z+a)*mul+d+h) * mul
	return b + x
}

// nearestMultiple64 returns the nearest multiple of 64 for length of
// provided byte slice.
func nearestMultiple64(b []byte) int {
	return ((len(b)) - 1) & ^63
}

// Hash64 return a 64-bit hash.
func Hash64(s []byte) uint64 {
	length := len(s)
	if length <= 16 {
		return hash0to16(s, length)
	}
	if length <= 32 {
		return hash17to32(s, length)
	}
	if length <= 64 {
		return hash33to64(s, length)
	}

	// For string over 64 bytes we hash the end first, and then as we
	// loop we keep 56 bytes of state: v, w, x, y and z.
	x := binary.LittleEndian.Uint64(s[length-40:])
	y := binary.LittleEndian.Uint64(s[length-16:]) + binary.LittleEndian.Uint64(s[length-56:])
	z := hash16(binary.LittleEndian.Uint64(s[length-48:])+uint64(length), binary.LittleEndian.Uint64(s[length-24:]))
	v := weakHash32SeedsByte(s[length-64:], uint64(length), z)
	w := weakHash32SeedsByte(s[length-32:], y+k1, x)
	x = x*k1 + binary.LittleEndian.Uint64(s)

	// Decrease len to the nearest multiple of 64, and operate on 64-byte chunks.
	s = s[:nearestMultiple64(s)]
	for len(s) > 0 {
		x = rot64(x+y+v.Low+binary.LittleEndian.Uint64(s[8:]), 37) * k1
		y = rot64(y+v.High+binary.LittleEndian.Uint64(s[48:]), 42) * k1
		x ^= w.High
		y += v.Low + binary.LittleEndian.Uint64(s[40:])
		z = rot64(z+w.Low, 33) * k1
		v = weakHash32SeedsByte(s, v.High*k1, x+w.Low)
		w = weakHash32SeedsByte(s[32:], z+w.High, y+binary.LittleEndian.Uint64(s[16:]))

		z, x = x, z
		s = s[64:]
	}

	return hash16(
		hash16(v.Low, w.Low)+shiftMix(y)*k1+z,
		hash16(v.High, w.High)+x,
	)
}

// Hash64WithSeed return a 64-bit hash with a seed.
func Hash64WithSeed(s []byte, seed uint64) uint64 {
	return Hash64WithSeeds(s, k2, seed)
}

// Hash64WithSeeds return a 64-bit hash with two seeds.
func Hash64WithSeeds(s []byte, seed0, seed1 uint64) uint64 {
	return hash16(Hash64(s)-seed0, seed1)
}
