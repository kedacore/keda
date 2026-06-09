package city

import "encoding/binary"

// Some primes between 2^63 and 2^64 for various uses.
const (
	k0 uint64 = 0xc3a5c85c97cb3127
	k1 uint64 = 0xb492b66fbe98f273
	k2 uint64 = 0x9ae16a3b2f90404f
)

// Magic numbers for 32-bit hashing. Copied from Murmur3.
const (
	c1 uint32 = 0xcc9e2d51
	c2 uint32 = 0x1b873593
)

func bswap32(x uint32) uint32 {
	return ((x & 0xff000000) >> 24) |
		((x & 0x00ff0000) >> 8) |
		((x & 0x0000ff00) << 8) |
		((x & 0x000000ff) << 24)
}

func fetch32(p []byte) uint32 {
	return binary.LittleEndian.Uint32(p)
}

// A 32-bit to 32-bit integer hash copied from Murmr3.
func fmix(h uint32) uint32 {
	h ^= h >> 16
	h *= 0x85ebca6b
	h ^= h >> 13
	h *= 0xc2b2ae35
	h ^= h >> 16
	return h
}

func rotate32(val uint32, shift int) uint32 {
	// Avoid shifting by 32: doing so yields and undefined result.
	if shift == 0 {
		return val
	}
	return (val >> uint32(shift)) | (val << (32 - uint32(shift)))
}

func mur(a, h uint32) uint32 {
	// Helper from Murmur3 for combining two 32-bit values.
	a *= c1
	a = rotate32(a, 17)
	a *= c2
	h ^= a
	h = rotate32(h, 19)
	return h*5 + 0xe6546b64
}

func hash32Len13to24(s []byte, length int) uint32 {
	a := fetch32(s[(length>>1)-4:])
	b := fetch32(s[4:])
	c := fetch32(s[length-8:])
	d := fetch32(s[length>>1:])
	e := fetch32(s)
	f := fetch32(s[length-4:])
	h := uint32(length)
	return fmix(mur(f, mur(e, mur(d, mur(c, mur(b, mur(a, h)))))))
}

func hash32Len0to4(s []byte, length int) uint32 {
	b := uint32(0)
	c := uint32(9)
	for _, v := range s[:length] {
		b = uint32(int64(b)*int64(c1) + int64(int8(v)))
		c ^= b
	}
	return fmix(mur(b, mur(uint32(length), c)))
}

func hash32Len5to12(s []byte, length int) uint32 {
	a := uint32(length)
	b := uint32(length) * 5
	c := uint32(9)
	d := b
	a += fetch32(s)
	b += fetch32(s[length-4:])
	c += fetch32(s[(length>>1)&4:])
	return fmix(mur(c, mur(b, mur(a, d))))
}

// Hash32 return 32-bit hash.
func Hash32(s []byte) uint32 {
	length := len(s)
	if length <= 4 {
		return hash32Len0to4(s, length)
	}
	if length <= 12 {
		return hash32Len5to12(s, length)
	}
	if length <= 24 {
		return hash32Len13to24(s, length)
	}

	// len > 24
	h := uint32(length)
	g := c1 * uint32(length)
	f := g
	a0 := rotate32(fetch32(s[length-4:])*c1, 17) * c2
	a1 := rotate32(fetch32(s[length-8:])*c1, 17) * c2
	a2 := rotate32(fetch32(s[length-16:])*c1, 17) * c2
	a3 := rotate32(fetch32(s[length-12:])*c1, 17) * c2
	a4 := rotate32(fetch32(s[length-20:])*c1, 17) * c2
	h ^= a0
	h = rotate32(h, 19)
	h = h*5 + 0xe6546b64
	h ^= a2
	h = rotate32(h, 19)
	h = h*5 + 0xe6546b64
	g ^= a1
	g = rotate32(g, 19)
	g = g*5 + 0xe6546b64
	g ^= a3
	g = rotate32(g, 19)
	g = g*5 + 0xe6546b64
	f += a4
	f = rotate32(f, 19)
	f = f*5 + 0xe6546b64
	iters := (length - 1) / 20
	for {
		a0 := rotate32(fetch32(s)*c1, 17) * c2
		a1 := fetch32(s[4:])
		a2 := rotate32(fetch32(s[8:])*c1, 17) * c2
		a3 := rotate32(fetch32(s[12:])*c1, 17) * c2
		a4 := fetch32(s[16:])
		h ^= a0
		h = rotate32(h, 18)
		h = h*5 + 0xe6546b64
		f += a1
		f = rotate32(f, 19)
		f *= c1
		g += a2
		g = rotate32(g, 18)
		g = g*5 + 0xe6546b64
		h ^= a3 + a1
		h = rotate32(h, 19)
		h = h*5 + 0xe6546b64
		g ^= a4
		g = bswap32(g) * 5
		h += a4 * 5
		h = bswap32(h)
		f += a0
		f, h, g = g, f, h
		s = s[20:]

		iters--
		if iters == 0 {
			break
		}
	}
	g = rotate32(g, 11) * c1
	g = rotate32(g, 17) * c1
	f = rotate32(f, 11) * c1
	f = rotate32(f, 17) * c1
	h = rotate32(h+g, 19)
	h = h*5 + 0xe6546b64
	h = rotate32(h, 17) * c1
	h = rotate32(h+f, 19)
	h = h*5 + 0xe6546b64
	h = rotate32(h, 17) * c1
	return h
}
