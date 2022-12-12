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
	"fmt"
	"io"
)

// vpackAssert panics if v is false.
func vpackAssert(v bool) {
	if !v {
		panic("VELOCYPACK_ASSERT failed")
	}
}

// readBytes reads bytes from the given reader until the given slice is full.
func readBytes(dst []byte, r io.Reader) error {
	offset := 0
	l := len(dst)
	for {
		n, err := r.Read(dst[offset:])
		offset += n
		l -= n
		if l == 0 {
			// We're done
			return nil
		}
		if err != nil {
			return WithStack(err)
		}
	}
}

// read an unsigned little endian integer value of the
// specified length, starting at the specified byte offset
func readIntegerFixed(start []byte, length uint) uint64 {
	return readIntegerNonEmpty(start, length)
}

// read an unsigned little endian integer value of the
// specified length, starting at the specified byte offset
func readIntegerFixedFromReader(r io.Reader, length uint) (uint64, []byte, error) {
	buf := make([]byte, length)
	if err := readBytes(buf, r); err != nil {
		return 0, nil, WithStack(err)
	}
	return readIntegerFixed(buf, length), buf, nil
}

// read an unsigned little endian integer value of the
// specified length, starting at the specified byte offset
func readIntegerNonEmpty(s []byte, length uint) uint64 {
	x := uint(0)
	v := uint64(0)
	for i := uint(0); i < length; i++ {
		v += uint64(s[i]) << x
		x += 8
	}
	return v
}

// read an unsigned little endian integer value of the
// specified length, starting at the specified byte offset
func readIntegerNonEmptyFromReader(r io.Reader, length uint) (uint64, []byte, error) {
	buf := make([]byte, length)
	if err := readBytes(buf, r); err != nil {
		return 0, nil, WithStack(err)
	}
	return readIntegerNonEmpty(buf, length), buf, nil
}

func toInt64(v uint64) int64 {
	shift2 := uint64(1) << 63
	shift := int64(shift2 - 1)
	if v >= shift2 {
		return (int64(v-shift2) - shift) - 1
	} else {
		return int64(v)
	}
}

func toUInt64(v int64) uint64 {
	// If v is negative, we need to add 2^63 to make it positive,
	// before we can cast it to an uint64_t:
	if v >= 0 {
		return uint64(v)
	}
	shift2 := uint64(1) << 63
	shift := int64(shift2 - 1)
	return uint64((v+shift)+1) + shift2
	//  return v >= 0 ? static_cast<uint64_t>(v)
	//                : static_cast<uint64_t>((v + shift) + 1) + shift2;
	// Note that g++ and clang++ with -O3 compile this away to
	// nothing. Further note that a plain cast from int64_t to
	// uint64_t is not guaranteed to work for negative values!
}

// read a variable length integer in unsigned LEB128 format
func readVariableValueLength(source []byte, offset ValueLength, reverse bool) ValueLength {
	length := ValueLength(0)
	p := uint(0)
	for {
		v := ValueLength(source[offset])
		length += (v & 0x7f) << p
		p += 7
		if reverse {
			offset--
		} else {
			offset++
		}
		if v&0x80 == 0 {
			break
		}
	}
	return length
}

// read a variable length integer in unsigned LEB128 format
func readVariableValueLengthFromReader(r io.Reader, reverse bool) (ValueLength, []byte, error) {
	if reverse {
		return 0, nil, WithStack(fmt.Errorf("reverse is not supported"))
	}
	length := ValueLength(0)
	p := uint(0)
	buf := make([]byte, 1)
	bytes := make([]byte, 0, 8)
	for {
		if n, err := r.Read(buf); n != 1 {
			if err != nil {
				return 0, nil, WithStack(err)
			} else {
				return 0, nil, WithStack(fmt.Errorf("failed to read 1 byte"))
			}
		}
		bytes = append(bytes, buf[0])
		v := ValueLength(buf[0])
		length += (v & 0x7f) << p
		p += 7
		if v&0x80 == 0 {
			break
		}
	}
	return length, bytes, nil
}

// store a variable length integer in unsigned LEB128 format
func storeVariableValueLength(dst []byte, offset, value ValueLength, reverse bool) {
	vpackAssert(value > 0)

	idx := offset
	if reverse {
		for value >= 0x80 {
			dst[idx] = byte(value | 0x80)
			idx--
			value >>= 7
		}
		dst[idx] = byte(value & 0x7f)
	} else {
		for value >= 0x80 {
			dst[idx] = byte(value | 0x80)
			idx++
			value >>= 7
		}
		dst[idx] = byte(value & 0x7f)
	}
}

// optionalBool returns the first arg element if available, otherwise returns defaultValue.
func optionalBool(arg []bool, defaultValue bool) bool {
	if len(arg) == 0 {
		return defaultValue
	}
	return arg[0]
}

// alignAt returns the first number >= value that is aligned at the given alignment.
// alignment must be a power of 2.
func alignAt(value, alignment uint) uint {
	mask := ^(alignment - 1)
	return (value + alignment - 1) & mask
}
