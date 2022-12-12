/*
 * Copyright (c) "Neo4j"
 * Neo4j Sweden AB [https://neo4j.com]
 *
 * This file is part of Neo4j.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package packstream

import (
	"encoding/binary"
	"fmt"
	"math"
)

const (
	PackedUndef = iota // Undefined must be zero!
	PackedInt
	PackedFloat
	PackedStr
	PackedStruct
	PackedByteArray
	PackedArray
	PackedMap
	PackedNil
	PackedTrue
	PackedFalse
)

type Unpacker struct {
	buf  []byte
	off  uint32
	len  uint32
	mrk  marker
	Err  error
	Curr int // Packed type
}

func (u *Unpacker) Reset(buf []byte) {
	u.buf = buf
	u.off = 0
	u.len = uint32(len(buf))
	u.Err = nil
	u.mrk.typ = PackedUndef
	u.Curr = PackedUndef
}

func (u *Unpacker) setErr(err error) {
	if u.Err == nil {
		u.Err = err
	}
}

func (u *Unpacker) Next() {
	i := u.pop()
	u.mrk = markers[i]
	u.Curr = u.mrk.typ
}

func (u *Unpacker) Len() uint32 {
	if u.mrk.numlenbytes == 0 {
		return uint32(u.mrk.shortlen)
	}
	return u.readlen(uint32(u.mrk.numlenbytes))
}

func (u *Unpacker) Int() int64 {
	n := u.mrk.numlenbytes
	if n == 0 {
		return int64(u.mrk.shortlen)
	}

	end := u.off + uint32(n)
	if end > u.len {
		u.setErr(&IoError{})
		return 0
	}
	i := int64(0)
	switch n {
	case 1:
		i = int64(int8(u.buf[u.off]))
	case 2:
		i = int64(int16(binary.BigEndian.Uint16(u.buf[u.off:])))
	case 4:
		i = int64(int32(binary.BigEndian.Uint32(u.buf[u.off:])))
	case 8:
		i = int64(binary.BigEndian.Uint64(u.buf[u.off:]))
	default:
		u.setErr(&UnpackError{msg: fmt.Sprintf("Illegal int length: %d", n)})
		return 0
	}
	u.off = end
	return i
}

func (u *Unpacker) Float() float64 {
	buf := u.read(8)
	if u.Err != nil {
		return math.NaN()
	}
	return math.Float64frombits(binary.BigEndian.Uint64(buf))
}

func (u *Unpacker) StructTag() byte {
	return u.pop()
}

func (u *Unpacker) String() string {
	n := uint32(u.mrk.numlenbytes)
	if n == 0 {
		n = uint32(u.mrk.shortlen)
	} else {
		n = u.readlen(n)
	}
	return string(u.read(n))
}

func (u *Unpacker) Bool() bool {
	switch u.Curr {
	case PackedTrue:
		return true
	case PackedFalse:
		return false
	default:
		u.setErr(&UnpackError{msg: "Illegal value for bool"})
		return false
	}
}

func (u *Unpacker) ByteArray() []byte {
	n := u.Len()
	buf := u.read(n)
	out := make([]byte, n)
	copy(out, buf)
	return out
}

func (u *Unpacker) pop() byte {
	if u.off < u.len {
		x := u.buf[u.off]
		u.off += 1
		return x
	}
	u.setErr(&IoError{})
	return 0
}

func (u *Unpacker) read(n uint32) []byte {
	start := u.off
	end := u.off + n
	if end > u.len {
		u.setErr(&IoError{})
		return []byte{}
	}
	u.off = end
	return u.buf[start:end]
}

func (u *Unpacker) readlen(n uint32) uint32 {
	end := u.off + n
	if end > u.len {
		u.setErr(&IoError{})
		return 0
	}
	l := uint32(0)
	switch n {
	case 1:
		l = uint32(u.buf[u.off])
	case 2:
		l = uint32(binary.BigEndian.Uint16(u.buf[u.off:]))
	case 4:
		l = uint32(binary.BigEndian.Uint32(u.buf[u.off:]))
	default:
		u.setErr(&UnpackError{msg: fmt.Sprintf("Illegal length: %d (%d)", n, u.Curr)})
	}
	u.off = end
	return l
}

type marker struct {
	typ         int
	shortlen    int8
	numlenbytes byte
}

var markers [0x100]marker

func init() {
	i := 0
	// Tiny int
	for ; i < 0x80; i++ {
		markers[i] = marker{typ: PackedInt, shortlen: int8(i)}
	}
	// Tiny string
	for ; i < 0x90; i++ {
		markers[i] = marker{typ: PackedStr, shortlen: int8(i - 0x80)}
	}
	// Tiny array
	for ; i < 0xa0; i++ {
		markers[i] = marker{typ: PackedArray, shortlen: int8(i - 0x90)}
	}
	// Tiny map
	for ; i < 0xb0; i++ {
		markers[i] = marker{typ: PackedMap, shortlen: int8(i - 0xa0)}
	}
	// Struct
	for ; i < 0xc0; i++ {
		markers[i] = marker{typ: PackedStruct, shortlen: int8(i - 0xb0)}
	}

	markers[0xc0] = marker{typ: PackedNil}
	markers[0xc1] = marker{typ: PackedFloat, numlenbytes: 8}
	markers[0xc2] = marker{typ: PackedFalse}
	markers[0xc3] = marker{typ: PackedTrue}

	markers[0xc8] = marker{typ: PackedInt, numlenbytes: 1}
	markers[0xc9] = marker{typ: PackedInt, numlenbytes: 2}
	markers[0xca] = marker{typ: PackedInt, numlenbytes: 4}
	markers[0xcb] = marker{typ: PackedInt, numlenbytes: 8}

	markers[0xcc] = marker{typ: PackedByteArray, numlenbytes: 1}
	markers[0xcd] = marker{typ: PackedByteArray, numlenbytes: 2}
	markers[0xce] = marker{typ: PackedByteArray, numlenbytes: 4}

	markers[0xd0] = marker{typ: PackedStr, numlenbytes: 1}
	markers[0xd1] = marker{typ: PackedStr, numlenbytes: 2}
	markers[0xd2] = marker{typ: PackedStr, numlenbytes: 4}

	markers[0xd4] = marker{typ: PackedArray, numlenbytes: 1}
	markers[0xd5] = marker{typ: PackedArray, numlenbytes: 2}
	markers[0xd6] = marker{typ: PackedArray, numlenbytes: 4}

	markers[0xd8] = marker{typ: PackedMap, numlenbytes: 1}
	markers[0xd9] = marker{typ: PackedMap, numlenbytes: 2}
	markers[0xda] = marker{typ: PackedMap, numlenbytes: 4}

	for i = 0xf0; i < 0x100; i++ {
		markers[i] = marker{typ: PackedInt, shortlen: int8(i - 0x100)}
	}
}
