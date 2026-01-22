package city

import "encoding/binary"

// much faster than uint64[2]

// U128 is uint128.
type U128 struct {
	Low  uint64 // first 64 bits
	High uint64 // last 64 bits
}

// A subroutine for Hash128(). Returns a decent 128-bit hash for strings
// of any length representable in signed long. Based on City and Mumur.
func cityMurmur(s []byte, seed U128) U128 {
	length := len(s)
	a := seed.Low
	b := seed.High
	c := uint64(0)
	d := uint64(0)
	l := length - 16
	if l <= 0 { // length <= 16
		a = shiftMix(a*k1) * k1
		c = b*k1 + hash0to16(s, length)

		tmp := c
		if length >= 8 {
			tmp = binary.LittleEndian.Uint64(s)
		}
		d = shiftMix(a + tmp)
	} else { // length > 16
		c = hash16(binary.LittleEndian.Uint64(s[length-8:])+k1, a)
		d = hash16(b+uint64(length), c+binary.LittleEndian.Uint64(s[length-16:]))
		a += d
		for {
			a ^= shiftMix(binary.LittleEndian.Uint64(s)*k1) * k1
			a *= k1
			b ^= a
			c ^= shiftMix(binary.LittleEndian.Uint64(s[8:])*k1) * k1
			c *= k1
			d ^= c
			s = s[16:]
			l -= 16
			if l <= 0 {
				break
			}
		}
	}
	a = hash16(a, c)
	b = hash16(d, b)
	return U128{a ^ b, hash16(b, a)}
}

// Hash128Seed return a 128-bit hash with a seed.
func Hash128Seed(s []byte, seed U128) U128 {
	if len(s) < 128 {
		return cityMurmur(s, seed)
	}

	// Saving initial input for tail hashing.
	t := s

	// We expect len >= 128 to be the common case. Keep 56 bytes of state:
	// v, w, x, y and z.
	var v, w U128
	x := seed.Low
	y := seed.High
	z := uint64(len(s)) * k1

	v.Low = rot64(y^k1, 49)*k1 + binary.LittleEndian.Uint64(s)
	v.High = rot64(v.Low, 42)*k1 + binary.LittleEndian.Uint64(s[8:])
	w.Low = rot64(y+z, 35)*k1 + x
	w.High = rot64(x+binary.LittleEndian.Uint64(s[88:]), 53) * k1

	// This is the same inner loop as Hash64(), manually unrolled.
	for len(s) >= 128 {
		// Roll 1.
		x = rot64(x+y+v.Low+binary.LittleEndian.Uint64(s[8:]), 37) * k1
		y = rot64(y+v.High+binary.LittleEndian.Uint64(s[48:]), 42) * k1
		x ^= w.High
		y += v.Low + binary.LittleEndian.Uint64(s[40:])
		z = rot64(z+w.Low, 33) * k1
		v = weakHash32SeedsByte(s, v.High*k1, x+w.Low)
		w = weakHash32SeedsByte(s[32:], z+w.High, y+binary.LittleEndian.Uint64(s[16:]))
		z, x = x, z
		s = s[64:]

		// Roll 2.
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

	x += rot64(v.Low+z, 49) * k0
	y = y*k0 + rot64(w.High, 37)
	z = z*k0 + rot64(w.Low, 27)
	w.Low *= 9
	v.Low *= k0

	// If 0 < length < 128, hash up to 4 chunks of 32 bytes each from the end of s.
	for i := 0; i < len(s); {
		i += 32
		y = rot64(x+y, 42)*k0 + v.High
		w.Low += binary.LittleEndian.Uint64(t[len(t)-i+16:])
		x = x*k0 + w.Low
		z += w.High + binary.LittleEndian.Uint64(t[len(t)-i:])
		w.High += v.Low
		v = weakHash32SeedsByte(t[len(t)-i:], v.Low+z, v.High)
		v.Low *= k0
	}

	// At this point our 56 bytes of state should contain more than
	// enough information for a strong 128-bit hash. We use two different
	// 56-byte-to-8-byte hashes to get a 16-byte final result.
	x = hash16(x, v.Low)
	y = hash16(y+z, w.Low)

	return U128{
		Low:  hash16(x+v.High, w.High) + y,
		High: hash16(x+w.High, y+v.High),
	}
}

// Hash128 returns a 128-bit hash and are tuned for strings of at least
// a few hundred bytes.  Depending on your compiler and hardware,
// it's likely faster than Hash64() on sufficiently long strings.
// It's slower than necessary on shorter strings, but we expect
// that case to be relatively unimportant.
func Hash128(s []byte) U128 {
	if len(s) >= 16 {
		return Hash128Seed(s[16:], U128{
			Low:  binary.LittleEndian.Uint64(s),
			High: binary.LittleEndian.Uint64(s[8:]) + k0},
		)
	}
	return Hash128Seed(s, U128{Low: k0, High: k1})
}
