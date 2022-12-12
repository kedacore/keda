//
// DISCLAIMER
//
// Copyright 2017 ArangoDB GmbH, Cologne, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright holder is ArangoDB GmbH, Cologne, Germany
//
// Author Ewout Prangsma
//

package velocypack

// builderBuffer is a byte slice used for building slices.
type builderBuffer []byte

const (
	minGrowDelta = 128         // Minimum amount of extra bytes to add to a buffer when growing
	maxGrowDelta = 1024 * 1024 // Maximum amount of extra bytes to add to a buffer when growing
)

// IsEmpty returns 0 if there are no values in the buffer.
func (b builderBuffer) IsEmpty() bool {
	l := len(b)
	return l == 0
}

// Len returns the length of the buffer.
func (b builderBuffer) Len() ValueLength {
	l := len(b)
	return ValueLength(l)
}

// Bytes returns the bytes written to the buffer.
// The returned slice is only valid until the next modification.
func (b *builderBuffer) Bytes() []byte {
	return *b
}

// WriteByte appends a single byte to the buffer.
func (b *builderBuffer) WriteByte(v byte) {
	off := len(*b)
	b.growCapacity(1)
	*b = (*b)[:off+1]
	(*b)[off] = v
}

// WriteBytes appends a series of identical bytes to the buffer.
func (b *builderBuffer) WriteBytes(v byte, count uint) {
	if count == 0 {
		return
	}
	off := uint(len(*b))
	b.growCapacity(count)
	*b = (*b)[:off+count]
	for i := uint(0); i < count; i++ {
		(*b)[off+i] = v
	}
}

// Write appends a series of bytes to the buffer.
func (b *builderBuffer) Write(v []byte) {
	l := uint(len(v))
	if l > 0 {
		off := uint(len(*b))
		b.growCapacity(l)
		*b = (*b)[:off+l]
		copy((*b)[off:], v)
	}
}

// ReserveSpace ensures that at least n bytes can be added to the buffer without allocating new memory.
func (b *builderBuffer) ReserveSpace(n uint) {
	if n > 0 {
		b.growCapacity(n)
	}
}

// Shrink reduces the length of the buffer by n elements (removing the last elements).
func (b *builderBuffer) Shrink(n uint) {
	if n > 0 {
		newLen := uint(len(*b)) - n
		if newLen < 0 {
			newLen = 0
		}
		*b = (*b)[:newLen]
	}
}

// Grow adds n elements to the buffer, returning a slice where the added elements start.
func (b *builderBuffer) Grow(n uint) []byte {
	l := uint(len(*b))
	if n > 0 {
		b.growCapacity(n)
		*b = (*b)[:l+n]
	}
	return (*b)[l:]
}

// growCapacity ensures that there is enough capacity in the buffer to add n elements.
func (b *builderBuffer) growCapacity(n uint) {
	_b := *b
	curLen := uint(len(_b))
	curCap := uint(cap(_b))
	newCap := curLen + n
	if newCap <= curCap {
		// No need to do anything
		return
	}
	// Increase the capacity
	extra := newCap // Grow a bit more to avoid copying all the time
	if extra < minGrowDelta {
		extra = minGrowDelta
	} else if extra > maxGrowDelta {
		extra = maxGrowDelta
	}
	newBuffer := make(builderBuffer, curLen, newCap+extra)
	copy(newBuffer, _b)
	*b = newBuffer
}
