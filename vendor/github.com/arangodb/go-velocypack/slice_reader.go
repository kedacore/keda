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

import (
	"bufio"
	"io"
)

const (
	maxByteSizeBytes = 16
)

// SliceFromReader reads a slice from the given reader.
func SliceFromReader(r io.Reader) (Slice, error) {
	if r, ok := r.(*bufio.Reader); ok {
		// Buffered reader can use faster path.
		return sliceFromBufReader(r)
	}
	hdr := make(Slice, 1, maxByteSizeBytes)
	// Read first byte
	if err := readBytes(hdr, r); err != nil {
		if Cause(err) == io.EOF {
			// Empty slice
			return nil, nil
		}
		return nil, WithStack(err)
	}
	// Lookup first size
	// check if the type has a fixed length first
	l := fixedTypeLengths[hdr[0]]
	if l != 0 {
		// Found fixed length, read it (minus byte already read)
		s := make(Slice, l)
		s[0] = hdr[0]
		if err := readBytes(s[1:], r); err != nil {
			return nil, WithStack(err)
		}
		return s, nil
	}

	readRemaining := func(prefix Slice, l ValueLength) (Slice, error) {
		s := make(Slice, l)
		copy(s, prefix)
		if err := readBytes(s[len(prefix):], r); err != nil {
			return nil, WithStack(err)
		}
		return s, nil
	}

	// types with dynamic lengths need special treatment:
	h := hdr[0]
	switch hdr.Type() {
	case Array, Object:
		if h == 0x13 || h == 0x14 {
			// compact Array or Object
			l, bytes, err := readVariableValueLengthFromReader(r, false)
			if err != nil {
				return nil, WithStack(err)
			}
			return readRemaining(append(hdr, bytes...), l)
		}

		vpackAssert(h > 0x00 && h <= 0x0e)
		l, bytes, err := readIntegerNonEmptyFromReader(r, widthMap[h])
		if err != nil {
			return nil, WithStack(err)
		}
		return readRemaining(append(hdr, bytes...), ValueLength(l))

	case String:
		vpackAssert(h == 0xbf)

		// long UTF-8 String
		l, bytes, err := readIntegerFixedFromReader(r, 8)
		if err != nil {
			return nil, WithStack(err)
		}
		return readRemaining(append(hdr, bytes...), ValueLength(l+1+8))

	case Binary:
		vpackAssert(h >= 0xc0 && h <= 0xc7)
		x, bytes, err := readIntegerNonEmptyFromReader(r, uint(h)-0xbf)
		if err != nil {
			return nil, WithStack(err)
		}
		l := ValueLength(1 + ValueLength(h) - 0xbf + ValueLength(x))
		return readRemaining(append(hdr, bytes...), l)

	case BCD:
		if h <= 0xcf {
			// positive BCD
			vpackAssert(h >= 0xc8 && h < 0xcf)
			x, bytes, err := readIntegerNonEmptyFromReader(r, uint(h)-0xc7)
			if err != nil {
				return nil, WithStack(err)
			}
			l := ValueLength(1 + ValueLength(h) - 0xc7 + ValueLength(x))
			return readRemaining(append(hdr, bytes...), l)
		}

		// negative BCD
		vpackAssert(h >= 0xd0 && h < 0xd7)
		x, bytes, err := readIntegerNonEmptyFromReader(r, uint(h)-0xcf)
		if err != nil {
			return nil, WithStack(err)
		}
		l := ValueLength(1 + ValueLength(h) - 0xcf + ValueLength(x))
		return readRemaining(append(hdr, bytes...), l)

	case Custom:
		vpackAssert(h >= 0xf4)
		switch h {
		case 0xf4, 0xf5, 0xf6:
			x, bytes, err := readIntegerFixedFromReader(r, 1)
			if err != nil {
				return nil, WithStack(err)
			}
			l := ValueLength(2 + x)
			return readRemaining(append(hdr, bytes...), l)
		case 0xf7, 0xf8, 0xf9:
			x, bytes, err := readIntegerFixedFromReader(r, 2)
			if err != nil {
				return nil, WithStack(err)
			}
			l := ValueLength(3 + x)
			return readRemaining(append(hdr, bytes...), l)
		case 0xfa, 0xfb, 0xfc:
			x, bytes, err := readIntegerFixedFromReader(r, 4)
			if err != nil {
				return nil, WithStack(err)
			}
			l := ValueLength(5 + x)
			return readRemaining(append(hdr, bytes...), l)
		case 0xfd, 0xfe, 0xff:
			x, bytes, err := readIntegerFixedFromReader(r, 8)
			if err != nil {
				return nil, WithStack(err)
			}
			l := ValueLength(9 + x)
			return readRemaining(append(hdr, bytes...), l)
		}
	}

	return nil, WithStack(InternalError)
}

// sliceFromBufReader reads a slice from the given buffered reader.
func sliceFromBufReader(r *bufio.Reader) (Slice, error) {
	// ByteSize is always found within first 16 bytes
	hdr, err := r.Peek(maxByteSizeBytes)
	if len(hdr) == 0 && err != nil {
		if Cause(err) == io.EOF {
			// Empty slice
			return nil, nil
		}
		return nil, WithStack(err)
	}
	s := Slice(hdr)
	size, err := s.ByteSize()
	if err != nil {
		return nil, WithStack(err)
	}
	// Now that we know the size, read the entire slice
	buf := make(Slice, size)
	offset := 0
	bytesRead := 0
	for ValueLength(bytesRead) < size {
		n, err := r.Read(buf[offset:])
		bytesRead += n
		offset += n
		if err != nil && ValueLength(bytesRead) < size {
			return nil, WithStack(err)
		}
	}
	return buf, nil
}
