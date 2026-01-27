package city

import "encoding/binary"

// A subroutine for CH128(). Returns a decent 128-bit hash for strings
// of any length representable in signed long. Based on City and Mumur.
func chMurmur(s []byte, seed U128) U128 {
	length := len(s)
	a := seed.Low
	b := seed.High
	c := uint64(0)
	d := uint64(0)
	l := length - 16
	if len(s) <= 16 { // length <= 16
		a = shiftMix(a*k1) * k1
		c = b*k1 + ch0to16(s, length)

		if length >= 8 {
			d = shiftMix(a + binary.LittleEndian.Uint64(s))
		} else {
			d = shiftMix(a + c)
		}
	} else { // length > 16
		c = ch16(binary.LittleEndian.Uint64(s[length-8:])+k1, a)
		d = ch16(b+uint64(length), c+binary.LittleEndian.Uint64(s[length-16:]))
		a += d

		{
			a ^= shiftMix(binary.LittleEndian.Uint64(s[0:8:8])*k1) * k1
			a *= k1
			b ^= a
			c ^= shiftMix(binary.LittleEndian.Uint64(s[8:8+8:8+8])*k1) * k1
			c *= k1
			d ^= c
			s = s[16:]
			l -= 16
		}

		if l > 0 {
			for len(s) >= 16 {
				a ^= shiftMix(binary.LittleEndian.Uint64(s[0:8:8])*k1) * k1
				a *= k1
				b ^= a
				c ^= shiftMix(binary.LittleEndian.Uint64(s[8:8+8:8+8])*k1) * k1
				c *= k1
				d ^= c
				s = s[16:]
				l -= 16

				if l <= 0 {
					break
				}
			}
		}
	}
	a = ch16(a, c)
	b = ch16(d, b)
	return U128{a ^ b, ch16(b, a)}
}

// CH128 returns 128-bit ClickHouse CityHash.
func CH128(s []byte) U128 {
	if len(s) >= 16 {
		return CH128Seed(s[16:], U128{
			Low:  binary.LittleEndian.Uint64(s[0:8:8]) ^ k3,
			High: binary.LittleEndian.Uint64(s[8 : 8+8 : 8+8]),
		})
	}
	if len(s) >= 8 {
		l := uint64(len(s))
		return CH128Seed(nil, U128{
			Low:  binary.LittleEndian.Uint64(s) ^ (l * k0),
			High: binary.LittleEndian.Uint64(s[l-8:]) ^ k1,
		})
	}
	return CH128Seed(s, U128{Low: k0, High: k1})
}

// CH128Seed returns 128-bit seeded ClickHouse CityHash.
func CH128Seed(s []byte, seed U128) U128 {
	if len(s) < 128 {
		return chMurmur(s, seed)
	}

	// Saving initial input for tail hashing.
	t := s

	// We expect len >= 128 to be the common case. Keep 56 bytes of state:
	// v, w, x, y and z.
	var v, w U128
	x := seed.Low
	y := seed.High
	z := uint64(len(s)) * k1

	{
		subSlice := (*[96]byte)(s[0:])
		v.Low = rot64(y^k1, 49)*k1 + binary.LittleEndian.Uint64(subSlice[0:])
		v.High = rot64(v.Low, 42)*k1 + binary.LittleEndian.Uint64(subSlice[8:])
		w.Low = rot64(y+z, 35)*k1 + x
		w.High = rot64(x+binary.LittleEndian.Uint64(subSlice[88:]), 53) * k1
	}

	// This is the same inner loop as CH64(), manually unrolled.
	for len(s) >= 128 {
		// Roll 1.
		{
			x = rot64(x+y+v.Low+binary.LittleEndian.Uint64(s[16:16+8:16+8]), 37) * k1
			y = rot64(y+v.High+binary.LittleEndian.Uint64(s[48:48+8:48+8]), 42) * k1

			x ^= w.High
			y ^= v.Low

			z = rot64(z^w.Low, 33)
			v = weakHash32SeedsByte(s, v.High*k1, x+w.Low)
			w = weakHash32SeedsByte(s[32:], z+w.High, y)
			z, x = x, z
		}

		// Roll 2.
		{
			const offset = 64
			x = rot64(x+y+v.Low+binary.LittleEndian.Uint64(s[offset+16:offset+16+8:offset+16+8]), 37) * k1
			y = rot64(y+v.High+binary.LittleEndian.Uint64(s[offset+48:offset+48+8:offset+48+8]), 42) * k1
			x ^= w.High
			y ^= v.Low

			z = rot64(z^w.Low, 33)
			v = weakHash32SeedsByte(s[offset:], v.High*k1, x+w.Low)
			w = weakHash32SeedsByte(s[offset+32:], z+w.High, y)
			z, x = x, z
		}
		s = s[128:]
	}

	y += rot64(w.Low, 37)*k0 + z
	x += rot64(v.Low+z, 49) * k0

	// If 0 < length < 128, hash up to 4 chunks of 32 bytes each from the end of s.
	for i := 0; i < len(s); {
		i += 32
		y = rot64(y-x, 42)*k0 + v.High
		w.Low += binary.LittleEndian.Uint64(t[len(t)-i+16:])
		x = rot64(x, 49)*k0 + w.Low
		w.Low += v.Low
		v = weakHash32SeedsByte(t[len(t)-i:], v.Low, v.High)
	}

	// At this point our 48 bytes of state should contain more than
	// enough information for a strong 128-bit hash.  We use two
	// different 48-byte-to-8-byte hashes to get a 16-byte final result.
	x = ch16(x, v.Low)
	y = ch16(y, w.Low)

	return U128{
		Low:  ch16(x+v.High, w.High) + y,
		High: ch16(x+w.High, y+v.High),
	}
}
