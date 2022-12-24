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
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"math"
	"time"
)

// Slice provides read only access to a VPack value
type Slice []byte

// SliceFromHex creates a Slice by decoding the given hex string into a Slice.
// If decoding fails, nil is returned.
func SliceFromHex(v string) Slice {
	if bytes, err := hex.DecodeString(v); err != nil {
		return nil
	} else {
		return Slice(bytes)
	}
}

// String returns a HEX representation of the slice.
func (s Slice) String() string {
	return hex.EncodeToString(s)
}

// JSONString converts the contents of the slice to JSON.
func (s Slice) JSONString(options ...DumperOptions) (string, error) {
	buf := &bytes.Buffer{}
	var opt *DumperOptions
	if len(options) > 0 {
		opt = &options[0]
	}
	d := NewDumper(buf, opt)
	if err := d.Append(s); err != nil {
		return "", WithStack(err)
	}
	return buf.String(), nil
}

// head returns the first element of the slice or 0 if the slice is empty.
func (s Slice) head() byte {
	if len(s) > 0 {
		return s[0]
	}
	return 0
}

// ByteSize returns the total byte size for the slice, including the head byte
func (s Slice) ByteSize() (ValueLength, error) {
	h := s.head()
	// check if the type has a fixed length first
	l := fixedTypeLengths[h]
	if l != 0 {
		// return fixed length
		return ValueLength(l), nil
	}

	// types with dynamic lengths need special treatment:
	switch s.Type() {
	case Array, Object:
		if h == 0x13 || h == 0x14 {
			// compact Array or Object
			return readVariableValueLength(s, 1, false), nil
		}

		vpackAssert(h > 0x00 && h <= 0x0e)
		return ValueLength(readIntegerNonEmpty(s[1:], widthMap[h])), nil

	case String:
		vpackAssert(h == 0xbf)
		// long UTF-8 String
		return ValueLength(1 + 8 + readIntegerFixed(s[1:], 8)), nil

	case Binary:
		vpackAssert(h >= 0xc0 && h <= 0xc7)
		return ValueLength(1 + ValueLength(h) - 0xbf + ValueLength(readIntegerNonEmpty(s[1:], uint(h)-0xbf))), nil

	case BCD:
		if h <= 0xcf {
			// positive BCD
			vpackAssert(h >= 0xc8 && h < 0xcf)
			return ValueLength(1 + ValueLength(h) - 0xc7 + ValueLength(readIntegerNonEmpty(s[1:], uint(h)-0xc7))), nil
		}

		// negative BCD
		vpackAssert(h >= 0xd0 && h < 0xd7)
		return ValueLength(1 + ValueLength(h) - 0xcf + ValueLength(readIntegerNonEmpty(s[1:], uint(h)-0xcf))), nil

	case Custom:
		vpackAssert(h >= 0xf4)
		switch h {
		case 0xf4, 0xf5, 0xf6:
			return ValueLength(2 + readIntegerFixed(s[1:], 1)), nil
		case 0xf7, 0xf8, 0xf9:
			return ValueLength(3 + readIntegerFixed(s[1:], 2)), nil
		case 0xfa, 0xfb, 0xfc:
			return ValueLength(5 + readIntegerFixed(s[1:], 4)), nil
		case 0xfd, 0xfe, 0xff:
			return ValueLength(9 + readIntegerFixed(s[1:], 8)), nil
		}
	}

	return 0, WithStack(InternalError)
}

// Next returns the Slice that directly follows the given slice.
// Same as s[s.ByteSize:]
func (s Slice) Next() (Slice, error) {
	size, err := s.ByteSize()
	if err != nil {
		return nil, WithStack(err)
	}
	return Slice(s[size:]), nil
}

// GetBool returns a boolean value from the slice.
// Returns an error if slice is not of type Bool.
func (s Slice) GetBool() (bool, error) {
	if err := s.AssertType(Bool); err != nil {
		return false, WithStack(err)
	}
	return s.IsTrue(), nil
}

// GetDouble returns a Double value from the slice.
// Returns an error if slice is not of type Double.
func (s Slice) GetDouble() (float64, error) {
	if err := s.AssertType(Double); err != nil {
		return 0.0, WithStack(err)
	}
	bits := binary.LittleEndian.Uint64(s[1:])
	return math.Float64frombits(bits), nil
}

// GetInt returns a Int value from the slice.
// Returns an error if slice is not of type Int.
func (s Slice) GetInt() (int64, error) {
	h := s.head()

	if h >= 0x20 && h <= 0x27 {
		// Int  T
		v := readIntegerNonEmpty(s[1:], uint(h)-0x1f)
		if h == 0x27 {
			return toInt64(v), nil
		} else {
			vv := int64(v)
			shift := int64(1) << ((h-0x1f)*8 - 1)
			if vv < shift {
				return vv, nil
			} else {
				return vv - (shift << 1), nil
			}
		}
	}

	if h >= 0x28 && h <= 0x2f {
		// UInt
		v, err := s.GetUInt()
		if err != nil {
			return 0, WithStack(err)
		}
		if v > math.MaxInt64 {
			return 0, WithStack(NumberOutOfRangeError)
		}
		return int64(v), nil
	}

	if h >= 0x30 && h <= 0x3f {
		// SmallInt
		return s.GetSmallInt()
	}

	return 0, WithStack(InvalidTypeError{"Expecting type Int"})
}

// GetUInt returns a UInt value from the slice.
// Returns an error if slice is not of type UInt.
func (s Slice) GetUInt() (uint64, error) {
	h := s.head()

	if h == 0x28 {
		// single byte integer
		return uint64(s[1]), nil
	}

	if h >= 0x29 && h <= 0x2f {
		// UInt
		return readIntegerNonEmpty(s[1:], uint(h)-0x27), nil
	}

	if h >= 0x20 && h <= 0x27 {
		// Int
		v, err := s.GetInt()
		if err != nil {
			return 0, WithStack(err)
		}
		if v < 0 {
			return 0, WithStack(NumberOutOfRangeError)
		}
		return uint64(v), nil
	}

	if h >= 0x30 && h <= 0x39 {
		// Smallint >= 0
		return uint64(h - 0x30), nil
	}

	if h >= 0x3a && h <= 0x3f {
		// Smallint < 0
		return 0, WithStack(NumberOutOfRangeError)
	}

	return 0, WithStack(InvalidTypeError{"Expecting type UInt"})
}

// GetSmallInt returns a SmallInt value from the slice.
// Returns an error if slice is not of type SmallInt.
func (s Slice) GetSmallInt() (int64, error) {
	h := s.head()

	if h >= 0x30 && h <= 0x39 {
		// Smallint >= 0
		return int64(h - 0x30), nil
	}

	if h >= 0x3a && h <= 0x3f {
		// Smallint < 0
		return int64(h-0x3a) - 6, nil
	}

	if (h >= 0x20 && h <= 0x27) || (h >= 0x28 && h <= 0x2f) {
		// Int and UInt
		// we'll leave it to the compiler to detect the two ranges above are
		// adjacent
		return s.GetInt()
	}

	return 0, InvalidTypeError{"Expecting type SmallInt"}
}

// GetUTCDate return the value for an UTCDate object
func (s Slice) GetUTCDate() (time.Time, error) {
	if !s.IsUTCDate() {
		return time.Time{}, InvalidTypeError{"Expecting type UTCDate"}
	}
	v := toInt64(readIntegerFixed(s[1:], 8)) // milliseconds since epoch
	sec := v / 1000
	nsec := (v % 1000) * 1000000
	return time.Unix(sec, nsec).UTC(), nil
}

// GetStringUTF8 return the value for a String object as a []byte with UTF-8 values.
// This function is a bit faster than GetString, since the conversion from
// []byte to string needs a memory allocation.
func (s Slice) GetStringUTF8() ([]byte, error) {
	h := s.head()
	if h >= 0x40 && h <= 0xbe {
		// short UTF-8 String
		length := h - 0x40
		result := s[1 : 1+length]
		return result, nil
	}

	if h == 0xbf {
		// long UTF-8 String
		length := readIntegerFixed(s[1:], 8)
		if err := checkOverflow(ValueLength(length)); err != nil {
			return nil, WithStack(err)
		}
		result := s[1+8 : 1+8+length]
		return result, nil
	}

	return nil, InvalidTypeError{"Expecting type String"}
}

// GetString return the value for a String object
// This function is a bit slower than GetStringUTF8, since the conversion from
// []byte to string needs a memory allocation.
func (s Slice) GetString() (string, error) {
	bytes, err := s.GetStringUTF8()
	if err != nil {
		return "", WithStack(err)
	}
	return string(bytes), nil
}

// GetStringLength return the length for a String object
func (s Slice) GetStringLength() (ValueLength, error) {
	h := s.head()
	if h >= 0x40 && h <= 0xbe {
		// short UTF-8 String
		length := h - 0x40
		return ValueLength(length), nil
	}

	if h == 0xbf {
		// long UTF-8 String
		length := readIntegerFixed(s[1:], 8)
		if err := checkOverflow(ValueLength(length)); err != nil {
			return 0, WithStack(err)
		}
		return ValueLength(length), nil
	}

	return 0, InvalidTypeError{"Expecting type String"}
}

// CompareString compares the string value in the slice with the given string.
// s == value -> 0
// s < value -> -1
// s > value -> 1
func (s Slice) CompareString(value string) (int, error) {
	k, err := s.GetStringUTF8()
	if err != nil {
		return 0, WithStack(err)
	}
	return bytes.Compare(k, []byte(value)), nil
}

// IsEqualString compares the string value in the slice with the given string for equivalence.
func (s Slice) IsEqualString(value string) (bool, error) {
	k, err := s.GetStringUTF8()
	if err != nil {
		return false, WithStack(err)
	}
	rc := bytes.Compare(k, []byte(value))
	return rc == 0, nil
}

// GetBinary return the value for a Binary object
func (s Slice) GetBinary() ([]byte, error) {
	if !s.IsBinary() {
		return nil, InvalidTypeError{"Expecting type Binary"}
	}

	h := s.head()
	vpackAssert(h >= 0xc0 && h <= 0xc7)

	lengthSize := uint(h - 0xbf)
	length := readIntegerNonEmpty(s[1:], lengthSize)
	checkOverflow(ValueLength(length))
	return s[1+lengthSize : 1+uint64(lengthSize)+length], nil
}

// GetBinaryLength return the length for a Binary object
func (s Slice) GetBinaryLength() (ValueLength, error) {
	if !s.IsBinary() {
		return 0, InvalidTypeError{"Expecting type Binary"}
	}

	h := s.head()
	vpackAssert(h >= 0xc0 && h <= 0xc7)

	lengthSize := uint(h - 0xbf)
	length := readIntegerNonEmpty(s[1:], lengthSize)
	return ValueLength(length), nil
}

// Length return the number of members for an Array or Object object
func (s Slice) Length() (ValueLength, error) {
	if !s.IsArray() && !s.IsObject() {
		return 0, InvalidTypeError{"Expecting type Array or Object"}
	}

	h := s.head()
	if h == 0x01 || h == 0x0a {
		// special case: empty!
		return 0, nil
	}

	if h == 0x13 || h == 0x14 {
		// compact Array or Object
		end := readVariableValueLength(s, 1, false)
		return readVariableValueLength(s, end-1, true), nil
	}

	offsetSize := indexEntrySize(h)
	vpackAssert(offsetSize > 0)
	end := readIntegerNonEmpty(s[1:], offsetSize)

	// find number of items
	if h <= 0x05 { // No offset table or length, need to compute:
		firstSubOffset := s.findDataOffset(h)
		first := s[firstSubOffset:]
		s, err := first.ByteSize()
		if err != nil {
			return 0, WithStack(err)
		}
		if s == 0 {
			return 0, WithStack(InternalError)
		}
		return (ValueLength(end) - firstSubOffset) / s, nil
	} else if offsetSize < 8 {
		return ValueLength(readIntegerNonEmpty(s[offsetSize+1:], offsetSize)), nil
	}

	return ValueLength(readIntegerNonEmpty(s[end-uint64(offsetSize):], offsetSize)), nil
}

// At extracts the array value at the specified index.
func (s Slice) At(index ValueLength) (Slice, error) {
	if !s.IsArray() {
		return nil, InvalidTypeError{"Expecting type Array"}
	}

	if result, err := s.getNth(index); err != nil {
		return nil, WithStack(err)
	} else {
		return result, nil
	}
}

// KeyAt extracts a key from an Object at the specified index.
func (s Slice) KeyAt(index ValueLength, translate ...bool) (Slice, error) {
	if !s.IsObject() {
		return nil, InvalidTypeError{"Expecting type Object"}
	}

	return s.getNthKey(index, optionalBool(translate, true))
}

// ValueAt extracts a value from an Object at the specified index
func (s Slice) ValueAt(index ValueLength) (Slice, error) {
	if !s.IsObject() {
		return nil, InvalidTypeError{"Expecting type Object"}
	}

	key, err := s.getNthKey(index, false)
	if err != nil {
		return nil, WithStack(err)
	}
	byteSize, err := key.ByteSize()
	if err != nil {
		return nil, WithStack(err)
	}
	return Slice(key[byteSize:]), nil
}

func indexEntrySize(head byte) uint {
	vpackAssert(head > 0x00 && head <= 0x12)
	return widthMap[head]
}

// Get looks for the specified attribute path inside an Object
// returns a Slice(ValueType::None) if not found
func (s Slice) Get(attributePath ...string) (Slice, error) {
	result := s
	parent := s
	for _, a := range attributePath {
		var err error
		result, err = parent.get(a)
		if err != nil {
			return nil, WithStack(err)
		}
		if result.IsNone() {
			return result, nil
		}
		parent = result
	}
	return result, nil
}

// Get looks for the specified attribute inside an Object
// returns a Slice(ValueType::None) if not found
func (s Slice) get(attribute string) (Slice, error) {
	if !s.IsObject() {
		return nil, InvalidTypeError{"Expecting Object"}
	}

	h := s.head()
	if h == 0x0a {
		// special case, empty object
		return nil, nil
	}

	if h == 0x14 {
		// compact Object
		value, err := s.getFromCompactObject(attribute)
		return value, WithStack(err)
	}

	offsetSize := indexEntrySize(h)
	vpackAssert(offsetSize > 0)
	end := ValueLength(readIntegerNonEmpty(s[1:], offsetSize))

	// read number of items
	var n ValueLength
	var ieBase ValueLength
	if offsetSize < 8 {
		n = ValueLength(readIntegerNonEmpty(s[1+offsetSize:], offsetSize))
		ieBase = end - n*ValueLength(offsetSize)
	} else {
		n = ValueLength(readIntegerNonEmpty(s[end-ValueLength(offsetSize):], offsetSize))
		ieBase = end - n*ValueLength(offsetSize) - ValueLength(offsetSize)
	}

	if n == 1 {
		// Just one attribute, there is no index table!
		key := Slice(s[s.findDataOffset(h):])

		if key.IsString() {
			if eq, err := key.IsEqualString(attribute); err != nil {
				return nil, WithStack(err)
			} else if eq {
				value, err := key.Next()
				return value, WithStack(err)
			}
			// fall through to returning None Slice below
		} else if key.IsSmallInt() || key.IsUInt() {
			// translate key
			if attributeTranslator == nil {
				return nil, WithStack(NeedAttributeTranslatorError)
			}
			if eq, err := key.translateUnchecked().IsEqualString(attribute); err != nil {
				return nil, WithStack(err)
			} else if eq {
				value, err := key.Next()
				return value, WithStack(err)
			}
		}

		// no match or invalid key type
		return nil, nil
	}

	// only use binary search for attributes if we have at least this many entries
	// otherwise we'll always use the linear search
	const SortedSearchEntriesThreshold = ValueLength(4)

	// bool const isSorted = (h >= 0x0b && h <= 0x0e);
	if n >= SortedSearchEntriesThreshold && (h >= 0x0b && h <= 0x0e) {
		// This means, we have to handle the special case n == 1 only
		// in the linear search!
		switch offsetSize {
		case 1:
			result, err := s.searchObjectKeyBinary(attribute, ieBase, n, 1)
			return result, WithStack(err)
		case 2:
			result, err := s.searchObjectKeyBinary(attribute, ieBase, n, 2)
			return result, WithStack(err)
		case 4:
			result, err := s.searchObjectKeyBinary(attribute, ieBase, n, 4)
			return result, WithStack(err)
		case 8:
			result, err := s.searchObjectKeyBinary(attribute, ieBase, n, 8)
			return result, WithStack(err)
		}
	}

	result, err := s.searchObjectKeyLinear(attribute, ieBase, ValueLength(offsetSize), n)
	return result, WithStack(err)
}

// HasKey returns true if the slice is an object that has a given key path.
func (s Slice) HasKey(keyPath ...string) (bool, error) {
	if result, err := s.Get(keyPath...); err != nil {
		return false, WithStack(err)
	} else {
		return !result.IsNone(), nil
	}
}

func (s Slice) getFromCompactObject(attribute string) (Slice, error) {
	it, err := NewObjectIterator(s)
	if err != nil {
		return nil, WithStack(err)
	}
	for it.IsValid() {
		key, err := it.Key(false)
		if err != nil {
			return nil, WithStack(err)
		}
		k, err := key.makeKey()
		if err != nil {
			return nil, WithStack(err)
		}
		if eq, err := k.IsEqualString(attribute); err != nil {
			return nil, WithStack(err)
		} else if eq {
			value, err := key.Next()
			return value, WithStack(err)
		}

		if err := it.Next(); err != nil {
			return nil, WithStack(err)
		}
	}
	// not found
	return nil, nil
}

func (s Slice) findDataOffset(head byte) ValueLength {
	// Must be called for a nonempty array or object at start():
	vpackAssert(head <= 0x12)
	fsm := firstSubMap[head]
	if fsm <= 2 && s[2] != 0 {
		return 2
	}
	if fsm <= 3 && s[3] != 0 {
		return 3
	}
	if fsm <= 5 && s[5] != 0 {
		return 5
	}
	return 9
}

// get the offset for the nth member from an Array or Object type
func (s Slice) getNthOffset(index ValueLength) (ValueLength, error) {
	vpackAssert(s.IsArray() || s.IsObject())

	h := s.head()

	if h == 0x13 || h == 0x14 {
		// compact Array or Object
		l, err := s.getNthOffsetFromCompact(index)
		if err != nil {
			return 0, WithStack(err)
		}
		return l, nil
	}

	if h == 0x01 || h == 0x0a {
		// special case: empty Array or empty Object
		return 0, WithStack(IndexOutOfBoundsError)
	}

	offsetSize := indexEntrySize(h)
	end := ValueLength(readIntegerNonEmpty(s[1:], offsetSize))

	dataOffset := ValueLength(0)

	// find the number of items
	var n ValueLength
	if h <= 0x05 { // No offset table or length, need to compute:
		dataOffset = s.findDataOffset(h)
		first := Slice(s[dataOffset:])
		s, err := first.ByteSize()
		if err != nil {
			return 0, WithStack(err)
		}
		if s == 0 {
			return 0, WithStack(InternalError)
		}
		n = (end - dataOffset) / s
	} else if offsetSize < 8 {
		n = ValueLength(readIntegerNonEmpty(s[1+offsetSize:], offsetSize))
	} else {
		n = ValueLength(readIntegerNonEmpty(s[end-ValueLength(offsetSize):], offsetSize))
	}

	if index >= n {
		return 0, WithStack(IndexOutOfBoundsError)
	}

	// empty array case was already covered
	vpackAssert(n > 0)

	if h <= 0x05 || n == 1 {
		// no index table, but all array items have the same length
		// now fetch first item and determine its length
		if dataOffset == 0 {
			dataOffset = s.findDataOffset(h)
		}
		sliceAtDataOffset := Slice(s[dataOffset:])
		sliceAtDataOffsetByteSize, err := sliceAtDataOffset.ByteSize()
		if err != nil {
			return 0, WithStack(err)
		}
		return dataOffset + index*sliceAtDataOffsetByteSize, nil
	}

	offsetSize8Or0 := ValueLength(0)
	if offsetSize == 8 {
		offsetSize8Or0 = 8
	}
	ieBase := end - n*ValueLength(offsetSize) + index*ValueLength(offsetSize) - (offsetSize8Or0)
	return ValueLength(readIntegerNonEmpty(s[ieBase:], offsetSize)), nil
}

// get the offset for the nth member from a compact Array or Object type
func (s Slice) getNthOffsetFromCompact(index ValueLength) (ValueLength, error) {
	end := ValueLength(readVariableValueLength(s, 1, false))
	n := ValueLength(readVariableValueLength(s, end-1, true))
	if index >= n {
		return 0, WithStack(IndexOutOfBoundsError)
	}

	h := s.head()
	offset := ValueLength(1 + getVariableValueLength(end))
	current := ValueLength(0)
	for current != index {
		sliceAtOffset := Slice(s[offset:])
		sliceAtOffsetByteSize, err := sliceAtOffset.ByteSize()
		if err != nil {
			return 0, WithStack(err)
		}
		offset += sliceAtOffsetByteSize
		if h == 0x14 {
			sliceAtOffset := Slice(s[offset:])
			sliceAtOffsetByteSize, err := sliceAtOffset.ByteSize()
			if err != nil {
				return 0, WithStack(err)
			}
			offset += sliceAtOffsetByteSize
		}
		current++
	}
	return offset, nil
}

// extract the nth member from an Array
func (s Slice) getNth(index ValueLength) (Slice, error) {
	vpackAssert(s.IsArray())

	offset, err := s.getNthOffset(index)
	if err != nil {
		return nil, WithStack(err)
	}
	return Slice(s[offset:]), nil
}

// getNthKey extract the nth member from an Object
func (s Slice) getNthKey(index ValueLength, translate bool) (Slice, error) {
	vpackAssert(s.Type() == Object)

	offset, err := s.getNthOffset(index)
	if err != nil {
		return nil, WithStack(err)
	}
	result := Slice(s[offset:])
	if translate {
		result, err = result.makeKey()
		if err != nil {
			return nil, WithStack(err)
		}
	}
	return result, nil
}

// getNthValue extract the nth value from an Object
func (s Slice) getNthValue(index ValueLength) (Slice, error) {
	key, err := s.getNthKey(index, false)
	if err != nil {
		return nil, WithStack(err)
	}
	value, err := key.Next()
	return value, WithStack(err)
}

func (s Slice) makeKey() (Slice, error) {
	if s.IsString() {
		return s, nil
	}
	if s.IsSmallInt() || s.IsUInt() {
		if attributeTranslator == nil {
			return nil, WithStack(NeedAttributeTranslatorError)
		}
		return s.translateUnchecked(), nil
	}

	return nil, InvalidTypeError{"Cannot translate key of this type"}
}

// perform a linear search for the specified attribute inside an Object
func (s Slice) searchObjectKeyLinear(attribute string, ieBase, offsetSize, n ValueLength) (Slice, error) {
	useTranslator := attributeTranslator != nil

	for index := ValueLength(0); index < n; index++ {
		offset := ValueLength(ieBase + index*offsetSize)
		key := Slice(s[readIntegerNonEmpty(s[offset:], uint(offsetSize)):])

		if key.IsString() {
			if eq, err := key.IsEqualString(attribute); err != nil {
				return nil, WithStack(err)
			} else if !eq {
				continue
			}
		} else if key.IsSmallInt() || key.IsUInt() {
			// translate key
			if !useTranslator {
				// no attribute translator
				return nil, WithStack(NeedAttributeTranslatorError)
			}
			if eq, err := key.translateUnchecked().IsEqualString(attribute); err != nil {
				return nil, WithStack(err)
			} else if !eq {
				continue
			}
		} else {
			// invalid key type
			return nil, nil
		}

		// key is identical. now return value
		value, err := key.Next()
		return value, WithStack(err)
	}

	// nothing found
	return nil, nil
}

// perform a binary search for the specified attribute inside an Object
//template<ValueLength offsetSize>
func (s Slice) searchObjectKeyBinary(attribute string, ieBase ValueLength, n ValueLength, offsetSize ValueLength) (Slice, error) {
	useTranslator := attributeTranslator != nil
	vpackAssert(n > 0)

	l := ValueLength(0)
	r := ValueLength(n - 1)
	index := ValueLength(r / 2)

	for {
		offset := ValueLength(ieBase + index*offsetSize)
		key := Slice(s[readIntegerFixed(s[offset:], uint(offsetSize)):])

		var res int
		var err error
		if key.IsString() {
			res, err = key.CompareString(attribute)
			if err != nil {
				return nil, WithStack(err)
			}
		} else if key.IsSmallInt() || key.IsUInt() {
			// translate key
			if !useTranslator {
				// no attribute translator
				return nil, WithStack(NeedAttributeTranslatorError)
			}
			res, err = key.translateUnchecked().CompareString(attribute)
			if err != nil {
				return nil, WithStack(err)
			}
		} else {
			// invalid key
			return nil, nil
		}

		if res == 0 {
			// found. now return a Slice pointing at the value
			keySize, err := key.ByteSize()
			if err != nil {
				return nil, WithStack(err)
			}
			return Slice(key[keySize:]), nil
		}

		if res > 0 {
			if index == 0 {
				return nil, nil
			}
			r = index - 1
		} else {
			l = index + 1
		}
		if r < l {
			return nil, nil
		}

		// determine new midpoint
		index = l + ((r - l) / 2)
	}
}

// translates an integer key into a string
func (s Slice) translate() (Slice, error) {
	if !s.IsSmallInt() && !s.IsUInt() {
		return nil, WithStack(InvalidTypeError{"Cannot translate key of this type"})
	}
	if attributeTranslator == nil {
		return nil, WithStack(NeedAttributeTranslatorError)
	}
	return s.translateUnchecked(), nil
}

// return the value for a UInt object, without checks!
// returns 0 for invalid values/types
func (s Slice) getUIntUnchecked() uint64 {
	h := s.head()
	if h >= 0x28 && h <= 0x2f {
		// UInt
		return readIntegerNonEmpty(s[1:], uint(h-0x27))
	}

	if h >= 0x30 && h <= 0x39 {
		// Smallint >= 0
		return uint64(h - 0x30)
	}
	return 0
}

// translates an integer key into a string, without checks
func (s Slice) translateUnchecked() Slice {
	id := s.getUIntUnchecked()
	key := attributeTranslator.IDToString(id)
	if key == "" {
		return nil
	}
	return StringSlice(key)
}
