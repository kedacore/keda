package city

import "encoding/binary"

// Ref:
// https://github.com/xzkostyan/python-cityhash/commit/f4091154ff2c6c0de11d5d6673b5007fdd6355ad

const k3 uint64 = 0xc949d7c7509e6557

func ch16(u, v uint64) uint64 {
	return hash128to64(U128{u, v})
}

// Return an 8-byte hash for 33 to 64 bytes.
func ch33to64(s []byte, length int) uint64 {
	z := binary.LittleEndian.Uint64(s[24:])
	a := binary.LittleEndian.Uint64(s) + (uint64(length)+binary.LittleEndian.Uint64(s[length-16:]))*k0
	b := rot64(a+z, 52)
	c := rot64(a, 37)

	a += binary.LittleEndian.Uint64(s[8:])
	c += rot64(a, 7)
	a += binary.LittleEndian.Uint64(s[16:])

	vf := a + z
	vs := b + rot64(a, 31) + c

	a = binary.LittleEndian.Uint64(s[16:]) + binary.LittleEndian.Uint64(s[length-32:])
	z = binary.LittleEndian.Uint64(s[length-8:])
	b = rot64(a+z, 52)
	c = rot64(a, 37)
	a += binary.LittleEndian.Uint64(s[length-24:])
	c += rot64(a, 7)
	a += binary.LittleEndian.Uint64(s[length-16:])

	wf := a + z
	ws := b + rot64(a, 31) + c
	r := shiftMix((vf+ws)*k2 + (wf+vs)*k0)
	return shiftMix(r*k0+vs) * k2
}

func ch17to32(s []byte, length int) uint64 {
	a := binary.LittleEndian.Uint64(s) * k1
	b := binary.LittleEndian.Uint64(s[8:])
	c := binary.LittleEndian.Uint64(s[length-8:]) * k2
	d := binary.LittleEndian.Uint64(s[length-16:]) * k0
	return hash16(
		rot64(a-b, 43)+rot64(c, 30)+d,
		a+rot64(b^k3, 20)-c+uint64(length),
	)
}

func ch0to16(s []byte, length int) uint64 {
	if length > 8 {
		a := binary.LittleEndian.Uint64(s)
		b := binary.LittleEndian.Uint64(s[length-8:])
		return ch16(a, rot64(b+uint64(length), uint(length))) ^ b
	}
	if length >= 4 {
		a := uint64(fetch32(s))
		return ch16(uint64(length)+(a<<3), uint64(fetch32(s[length-4:])))
	}
	if length > 0 {
		a := s[0]
		b := s[length>>1]
		c := s[length-1]
		y := uint32(a) + (uint32(b) << 8)
		z := uint32(length) + (uint32(c) << 2)
		return shiftMix(uint64(y)*k2^uint64(z)*k3) * k2
	}
	return k2
}

// CH64 returns ClickHouse version of Hash64.
func CH64(s []byte) uint64 {
	length := len(s)
	if length <= 16 {
		return ch0to16(s, length)
	}
	if length <= 32 {
		return ch17to32(s, length)
	}
	if length <= 64 {
		return ch33to64(s, length)
	}

	x := binary.LittleEndian.Uint64(s)
	y := binary.LittleEndian.Uint64(s[length-16:]) ^ k1
	z := binary.LittleEndian.Uint64(s[length-56:]) ^ k0

	v := weakHash32SeedsByte(s[length-64:], uint64(length), y)
	w := weakHash32SeedsByte(s[length-32:], uint64(length)*k1, k0)
	z += shiftMix(v.High) * k1
	x = rot64(z+x, 39) * k1
	y = rot64(y, 33) * k1

	// Decrease len to the nearest multiple of 64, and operate on 64-byte chunks.
	s = s[:nearestMultiple64(s)]
	for len(s) > 0 {
		x = rot64(x+y+v.Low+binary.LittleEndian.Uint64(s[16:]), 37) * k1
		y = rot64(y+v.High+binary.LittleEndian.Uint64(s[48:]), 42) * k1

		x ^= w.High
		y ^= v.Low

		z = rot64(z^w.Low, 33)
		v = weakHash32SeedsByte(s, v.High*k1, x+w.Low)
		w = weakHash32SeedsByte(s[32:], z+w.High, y)

		z, x = x, z
		s = s[64:]
	}

	return ch16(
		ch16(v.Low, w.Low)+shiftMix(y)*k1+z,
		ch16(v.High, w.High)+x,
	)
}
