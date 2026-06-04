// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package spanner

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

// ssformat provides encoding utilities for Sortable String Format (ssformat) keys.
// This encoding scheme converts database values into byte sequences that preserve
// lexicographic ordering when compared byte-by-byte.
//
// The encoding supports both ascending (increasing) and descending (decreasing)
// sort orders for all data types.

const (
	// isKey is the bit set in the header byte to indicate this is a key encoding.
	ssIsKey = 0x80

	// Unsigned integers (variable length 1-9 bytes)
	ssTypeUint1           = 0
	ssTypeDecreasingUint1 = 40

	// Signed integers (variable length 1-8 bytes)
	ssTypeNegInt1           = 16
	ssTypePosInt1           = 17
	ssTypeDecreasingNegInt1 = 48
	ssTypeDecreasingPosInt1 = 49

	// Strings
	ssTypeString           = 25
	ssTypeDecreasingString = 57

	// Nullable markers
	ssTypeNullOrderedFirst                = 27
	ssTypeNullableNotNullNullOrderedFirst = 28
	ssTypeNullableNotNullNullOrderedLast  = 59
	ssTypeNullOrderedLast                 = 60

	// Doubles (variable length 1-8 bytes, encoded as transformed int64)
	ssTypeNegDouble1           = 73
	ssTypePosDouble1           = 74
	ssTypeDecreasingNegDouble1 = 89
	ssTypeDecreasingPosDouble1 = 90

	// Escape characters for string/bytes encoding
	ssAscendingZeroEscape byte = 0xF0
	ssAscendingFFEscape   byte = 0x10
	ssSep                 byte = 0x78

	// Tag encoding constants
	ssObjectExistenceTag = 0x7E
	ssMaxFieldTag        = 0xFFFF

	// Offset to make negative timestamp seconds sort correctly
	ssTimestampSecondsOffset = 1 << 63
)

// makePrefixSuccessor returns the smallest possible key that is larger than
// the input key and that does not have the input key as a prefix.
//
// This is done by setting the least significant bit of the last byte.
// Returns nil if the input is empty or nil.
func makePrefixSuccessor(key []byte) []byte {
	if len(key) == 0 {
		return nil
	}
	result := make([]byte, len(key))
	copy(result, key)
	result[len(result)-1] |= 1
	return result
}

// appendCompositeTag appends a composite tag for a table identifier to the buffer.
// Tags must be in the range [1, 65535] and not equal to 0x7E (object existence tag).
//
// Tag encoding uses variable length:
//   - Tags 1-15: 1 byte
//   - Tags 16-4095: 2 bytes
//   - Tags 4096-65535: 3 bytes
func appendCompositeTag(buf []byte, tag int) ([]byte, error) {
	if tag == ssObjectExistenceTag || tag <= 0 || tag > ssMaxFieldTag {
		return buf, fmt.Errorf("ssformat: invalid tag value %d (must be 1-%d, excluding %d)",
			tag, ssMaxFieldTag, ssObjectExistenceTag)
	}

	if tag < 16 {
		// Short tag: encode as (tag << 1)
		return append(buf, byte(tag<<1)), nil
	}

	// Long tag: reserve one bit for the prefix successor bit.
	// Header format: 0 NN TTTTT where NN is num_extra_bytes (01=1, 10=2)
	// and TTTTT are the high bits of the shifted tag.
	shiftedTag := tag << 1

	if shiftedTag < (1 << 13) { // Original tag < 4096
		// 2-byte encoding: num_extra_bytes=1, header byte is 001xxxxx (0x20 | tag_bits)
		return append(buf,
			byte((1<<5)|(shiftedTag>>8)),
			byte(shiftedTag&0xFF),
		), nil
	}

	// 3-byte encoding: num_extra_bytes=2, header byte is 010xxxxx (0x40 | tag_bits)
	return append(buf,
		byte((2<<5)|(shiftedTag>>16)),
		byte((shiftedTag>>8)&0xFF),
		byte(shiftedTag&0xFF),
	), nil
}

// appendNullOrderedFirst appends a NULL marker that sorts before all non-NULL values.
func appendNullOrderedFirst(buf []byte) []byte {
	return append(buf, byte(ssIsKey|ssTypeNullOrderedFirst), 0)
}

// appendNullOrderedLast appends a NULL marker that sorts after all non-NULL values.
func appendNullOrderedLast(buf []byte) []byte {
	return append(buf, byte(ssIsKey|ssTypeNullOrderedLast), 0)
}

// appendNotNullMarkerNullOrderedFirst appends a non-NULL marker when using NULLS FIRST ordering.
func appendNotNullMarkerNullOrderedFirst(buf []byte) []byte {
	return append(buf, byte(ssIsKey|ssTypeNullableNotNullNullOrderedFirst))
}

// appendNotNullMarkerNullOrderedLast appends a non-NULL marker when using NULLS LAST ordering.
func appendNotNullMarkerNullOrderedLast(buf []byte) []byte {
	return append(buf, byte(ssIsKey|ssTypeNullableNotNullNullOrderedLast))
}

// appendUint64Increasing appends an unsigned 64-bit integer value in ascending sort order.
func appendUint64Increasing(buf []byte, val uint64) []byte {
	var payload [9]byte // Max 9 bytes for value payload
	payloadLen := 0

	tempVal := val
	payload[8-payloadLen] = byte((tempVal & 0x7F) << 1) // LSB is prefix-successor bit (0)
	tempVal >>= 7
	payloadLen++

	for tempVal > 0 {
		payload[8-payloadLen] = byte(tempVal & 0xFF)
		tempVal >>= 8
		payloadLen++
	}

	buf = append(buf, byte(ssIsKey|(ssTypeUint1+payloadLen-1)))
	return append(buf, payload[9-payloadLen:]...)
}

// appendUint64Decreasing appends an unsigned 64-bit integer value in descending sort order.
func appendUint64Decreasing(buf []byte, val uint64) []byte {
	var payload [9]byte
	payloadLen := 0
	tempVal := val

	payload[8-payloadLen] = byte((^(tempVal & 0x7F) & 0x7F) << 1)
	tempVal >>= 7
	payloadLen++

	for tempVal > 0 {
		payload[8-payloadLen] = byte(^(tempVal & 0xFF))
		tempVal >>= 8
		payloadLen++
	}

	buf = append(buf, byte(ssIsKey|(ssTypeDecreasingUint1-payloadLen+1)))
	return append(buf, payload[9-payloadLen:]...)
}

// appendIntInternal is the internal implementation for signed integer and double encoding.
func appendIntInternal(buf []byte, val int64, decreasing, isDouble bool) []byte {
	if decreasing {
		val = ^val
	}

	var payload [8]byte // Max 8 bytes for payload
	payloadLen := 0
	tempVal := val

	payload[7-payloadLen] = byte((tempVal & 0x7F) << 1)
	tempVal >>= 7
	payloadLen++

	// For positive numbers, loop until all bits are 0s.
	// For negative numbers, loop until all bits are 1s (sign extension).
	loopEndVal := int64(0)
	if tempVal < 0 {
		loopEndVal = -1
	}
	for tempVal != loopEndVal {
		payload[7-payloadLen] = byte(tempVal & 0xFF)
		tempVal >>= 8
		payloadLen++
	}

	var typeVal int
	if val >= 0 {
		switch {
		case !decreasing && !isDouble:
			typeVal = ssTypePosInt1 + payloadLen - 1
		case !decreasing && isDouble:
			typeVal = ssTypePosDouble1 + payloadLen - 1
		case decreasing && !isDouble:
			typeVal = ssTypeDecreasingPosInt1 + payloadLen - 1
		default: // decreasing && isDouble
			typeVal = ssTypeDecreasingPosDouble1 + payloadLen - 1
		}
	} else {
		switch {
		case !decreasing && !isDouble:
			typeVal = ssTypeNegInt1 - payloadLen + 1
		case !decreasing && isDouble:
			typeVal = ssTypeNegDouble1 - payloadLen + 1
		case decreasing && !isDouble:
			typeVal = ssTypeDecreasingNegInt1 - payloadLen + 1
		default: // decreasing && isDouble
			typeVal = ssTypeDecreasingNegDouble1 - payloadLen + 1
		}
	}

	buf = append(buf, byte(ssIsKey|typeVal))
	return append(buf, payload[8-payloadLen:]...)
}

// appendIntIncreasing appends a signed integer value in ascending sort order.
func appendIntIncreasing(buf []byte, value int64) []byte {
	return appendIntInternal(buf, value, false, false)
}

// appendIntDecreasing appends a signed integer value in descending sort order.
func appendIntDecreasing(buf []byte, value int64) []byte {
	return appendIntInternal(buf, value, true, false)
}

// appendDoubleInternal is the internal implementation for double encoding.
// The double is transformed to maintain lexicographic ordering:
// negative doubles (bit 63 = 1) are transformed via (MinInt64 - enc).
func appendDoubleInternal(buf []byte, value float64, decreasing bool) []byte {
	enc := int64(math.Float64bits(value))
	if enc < 0 {
		enc = math.MinInt64 - enc
	}
	return appendIntInternal(buf, enc, decreasing, true)
}

// appendDoubleIncreasing appends a double value in ascending sort order.
func appendDoubleIncreasing(buf []byte, value float64) []byte {
	return appendDoubleInternal(buf, value, false)
}

// appendDoubleDecreasing appends a double value in descending sort order.
func appendDoubleDecreasing(buf []byte, value float64) []byte {
	return appendDoubleInternal(buf, value, true)
}

// appendByteSequence is the internal implementation for string and bytes encoding.
func appendByteSequence(buf, data []byte, decreasing bool) []byte {
	if decreasing {
		buf = append(buf, byte(ssIsKey|ssTypeDecreasingString))
	} else {
		buf = append(buf, byte(ssIsKey|ssTypeString))
	}

	for _, b := range data {
		currentByte := b
		if decreasing {
			currentByte = ^b
		}

		switch currentByte {
		case 0x00:
			// Escape sequence for 0x00: write 0x00 followed by 0xF0
			buf = append(buf, 0x00, ssAscendingZeroEscape)
		case 0xFF:
			// Escape sequence for 0xFF: write 0xFF followed by 0x10
			buf = append(buf, 0xFF, ssAscendingFFEscape)
		default:
			buf = append(buf, currentByte)
		}
	}

	// Terminator
	if decreasing {
		buf = append(buf, 0xFF, ssSep)
	} else {
		buf = append(buf, 0x00, ssSep)
	}

	return buf
}

// appendStringIncreasing appends a UTF-8 string value in ascending sort order.
func appendStringIncreasing(buf []byte, value string) []byte {
	return appendByteSequence(buf, []byte(value), false)
}

// appendStringDecreasing appends a UTF-8 string value in descending sort order.
func appendStringDecreasing(buf []byte, value string) []byte {
	return appendByteSequence(buf, []byte(value), true)
}

// appendBytesIncreasing appends a byte slice value in ascending sort order.
func appendBytesIncreasing(buf []byte, value []byte) []byte {
	return appendByteSequence(buf, value, false)
}

// appendBytesDecreasing appends a byte slice value in descending sort order.
func appendBytesDecreasing(buf []byte, value []byte) []byte {
	return appendByteSequence(buf, value, true)
}

// encodeTimestamp encodes a timestamp as 12 bytes: 8 bytes for seconds since epoch
// (with offset to handle negative), 4 bytes for nanoseconds.
func encodeTimestamp(seconds int64, nanos int32) []byte {
	offsetSeconds := uint64(seconds) + ssTimestampSecondsOffset
	buf := make([]byte, 12)
	binary.BigEndian.PutUint64(buf[0:8], offsetSeconds)
	binary.BigEndian.PutUint32(buf[8:12], uint32(nanos))
	return buf
}

// encodeUUID encodes a UUID (128-bit) as 16 bytes in big-endian order.
func encodeUUID(high, low int64) []byte {
	buf := make([]byte, 16)
	binary.BigEndian.PutUint64(buf[0:8], uint64(high))
	binary.BigEndian.PutUint64(buf[8:16], uint64(low))
	return buf
}

// targetRange represents a key range with start and limit boundaries for routing.
type targetRange struct {
	start       []byte
	limit       []byte
	approximate bool
}

// newTargetRange creates a new targetRange with the given boundaries.
func newTargetRange(start, limit []byte, approximate bool) *targetRange {
	return &targetRange{
		start:       start,
		limit:       limit,
		approximate: approximate,
	}
}

// isPoint returns true if this range represents a single key (point lookup).
func (r *targetRange) isPoint() bool {
	return len(r.limit) == 0
}

// mergeFrom merges another targetRange into this one. The resulting range
// will be the union of the two ranges, taking the minimum start key and
// maximum limit key.
func (r *targetRange) mergeFrom(other *targetRange) {
	if bytes.Compare(other.start, r.start) < 0 {
		r.start = other.start
	}
	if other.isPoint() {
		if bytes.Compare(other.start, r.limit) >= 0 {
			r.limit = makePrefixSuccessor(other.start)
		}
	} else if bytes.Compare(other.limit, r.limit) > 0 {
		r.limit = other.limit
	}
	r.approximate = r.approximate || other.approximate
}
