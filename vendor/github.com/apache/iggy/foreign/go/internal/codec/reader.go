// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package codec

import (
	"encoding"
	"encoding/binary"
	"fmt"
	"math"
	"runtime"
)

// Reader is a cursor over a byte slice. The first out-of-bounds read sets err;
// all subsequent reads are no-ops. Call Err() once after all reads to check.
type Reader struct {
	p   []byte
	pos int
	err error
}

func NewReader(p []byte) *Reader {
	return &Reader{p: p}
}

// overrun sets r.err to a descriptive error message including the caller's
// file and line number.
func (r *Reader) overrun(need int) {
	_, file, line, _ := runtime.Caller(2)
	r.err = fmt.Errorf(
		"reader: need %d bytes at offset %d, only %d remaining (%s:%d)",
		need, r.pos, len(r.p)-r.pos, file, line)
}

func (r *Reader) U8() uint8 {
	if r.err != nil {
		return 0
	}
	if r.pos+1 > len(r.p) {
		r.overrun(1)
		return 0
	}
	v := r.p[r.pos]
	r.pos++
	return v
}

func (r *Reader) U16() uint16 {
	if r.err != nil {
		return 0
	}
	if r.pos+2 > len(r.p) {
		r.overrun(2)
		return 0
	}
	v := binary.LittleEndian.Uint16(r.p[r.pos : r.pos+2])
	r.pos += 2
	return v
}

func (r *Reader) U32() uint32 {
	if r.err != nil {
		return 0
	}
	if r.pos+4 > len(r.p) {
		r.overrun(4)
		return 0
	}
	v := binary.LittleEndian.Uint32(r.p[r.pos : r.pos+4])
	r.pos += 4
	return v
}

func (r *Reader) U64() uint64 {
	if r.err != nil {
		return 0
	}
	if r.pos+8 > len(r.p) {
		r.overrun(8)
		return 0
	}
	v := binary.LittleEndian.Uint64(r.p[r.pos : r.pos+8])
	r.pos += 8
	return v
}

func (r *Reader) F32() float32 {
	if r.err != nil {
		return 0
	}
	if r.pos+4 > len(r.p) {
		r.overrun(4)
		return 0
	}
	v := math.Float32frombits(binary.LittleEndian.Uint32(r.p[r.pos : r.pos+4]))
	r.pos += 4
	return v
}

// str reads exactly n bytes and returns a copy as a string.
func (r *Reader) str(n int) string {
	v := string(r.p[r.pos : r.pos+n])
	r.pos += n
	return v
}

// raw reads exactly n bytes and returns a copy.
func (r *Reader) raw(n int) []byte {
	v := make([]byte, n)
	copy(v, r.p[r.pos:r.pos+n])
	r.pos += n
	return v
}

// Raw reads exactly n bytes and returns a copy.
func (r *Reader) Raw(n int) []byte {
	if r.err != nil {
		return nil
	}
	if r.pos+n > len(r.p) {
		r.overrun(n)
		return nil
	}
	return r.raw(n)
}

// Str reads exactly n bytes and returns a copy as a string. Use U8LenStr or
// U32LenStr instead if the data is length-prefixed:
//
//	[length: 1 byte][data: N bytes]   → U8LenStr
//	[length: 4 bytes][data: N bytes]  → U32LenStr
func (r *Reader) Str(n int) string {
	if r.err != nil {
		return ""
	}
	if r.pos+n > len(r.p) {
		r.overrun(n)
		return ""
	}
	return r.str(n)
}

// U32LenStr reads a length-prefixed string where the length is a 4-byte
// little-endian unsigned integer.
func (r *Reader) U32LenStr() string {
	if r.err != nil {
		return ""
	}
	if r.pos+4 > len(r.p) {
		r.overrun(4)
		return ""
	}
	n := int(binary.LittleEndian.Uint32(r.p[r.pos : r.pos+4]))
	r.pos += 4
	if r.pos+n > len(r.p) {
		r.overrun(n)
		return ""
	}
	return r.str(n)
}

// U8LenStr reads a length-prefixed string where the length is a single byte.
func (r *Reader) U8LenStr() string {
	if r.err != nil {
		return ""
	}
	if r.pos+1 > len(r.p) {
		r.overrun(1)
		return ""
	}
	n := int(r.p[r.pos])
	r.pos++
	if r.pos+n > len(r.p) {
		r.overrun(n)
		return ""
	}
	return r.str(n)
}

// Obj reads n bytes and decodes them into v.
func (r *Reader) Obj(n int, v encoding.BinaryUnmarshaler) {
	if r.err != nil {
		return
	}
	if r.pos+n > len(r.p) {
		r.overrun(n)
		return
	}
	err := v.UnmarshalBinary(r.raw(n))
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		r.err = fmt.Errorf("%w (%s:%d)", err, file, line)
		return
	}
}

// Remaining returns the number of unread bytes.
func (r *Reader) Remaining() int {
	return len(r.p) - r.pos
}

// Err returns the first error encountered during reading, or nil.
func (r *Reader) Err() error { return r.err }
