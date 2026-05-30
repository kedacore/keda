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

// Writer appends encoded values to a growing byte slice. The first encoding
// error sets err; all subsequent writes are no-ops. Call Err() once after all
// writes to check.
type Writer struct {
	p   []byte
	err error
}

// NewWriter returns a new Writer with an empty internal buffer.
// Use NewWriterCap instead if the final size is known.
func NewWriter() *Writer {
	return &Writer{}
}

// NewWriterCap returns a Writer with its internal buffer pre-allocated to n
// bytes. Use this when the final size is known in advance to avoid
// reallocations.
func NewWriterCap(n int) *Writer {
	return &Writer{p: make([]byte, 0, n)}
}

func (w *Writer) U8(v uint8) {
	if w.err != nil {
		return
	}
	w.p = append(w.p, v)
}

func (w *Writer) U16(v uint16) {
	if w.err != nil {
		return
	}
	w.p = binary.LittleEndian.AppendUint16(w.p, v)
}

func (w *Writer) U32(v uint32) {
	if w.err != nil {
		return
	}
	w.p = binary.LittleEndian.AppendUint32(w.p, v)
}

func (w *Writer) U64(v uint64) {
	if w.err != nil {
		return
	}
	w.p = binary.LittleEndian.AppendUint64(w.p, v)
}

func (w *Writer) F32(v float32) {
	if w.err != nil {
		return
	}
	w.p = binary.LittleEndian.AppendUint32(w.p, math.Float32bits(v))
}

// Str writes a string with no length prefix. Use U8LenStr or U32LenStr
// instead if the reader expects a length prefix.
func (w *Writer) Str(v string) {
	if w.err != nil {
		return
	}
	w.str(v)
}

// U32LenStr writes a length-prefixed string where the length is a 4-byte
// little-endian unsigned integer.
func (w *Writer) U32LenStr(v string) {
	if w.err != nil {
		return
	}
	w.p = binary.LittleEndian.AppendUint32(w.p, uint32(len(v)))
	w.str(v)
}

// U8LenStr writes a length-prefixed string where the length is a single byte.
// Sets w.err if len(v) exceeds 255.
func (w *Writer) U8LenStr(v string) {
	if w.err != nil {
		return
	}
	if len(v) > math.MaxUint8 {
		_, file, line, _ := runtime.Caller(1)
		w.err = fmt.Errorf("string length %d exceeds 255 (%s:%d)", len(v), file, line)
		return
	}
	w.p = append(w.p, uint8(len(v)))
	w.str(v)
}

func (w *Writer) str(v string) {
	w.p = append(w.p, v...)
}

// Raw appends a raw byte slice with no length prefix.
func (w *Writer) Raw(v []byte) {
	if w.err != nil {
		return
	}
	w.p = append(w.p, v...)
}

// Obj encodes v and appends the result to the buffer.
func (w *Writer) Obj(v encoding.BinaryMarshaler) {
	if w.err != nil {
		return
	}
	b, err := v.MarshalBinary()
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		w.err = fmt.Errorf("%w (%s:%d)", err, file, line)
		return
	}
	w.p = append(w.p, b...)
}

// Bytes returns the accumulated buffer directly. The slice is valid only
// until the next write. The caller must not mutate the returned slice.
func (w *Writer) Bytes() []byte {
	return w.p
}

// Err returns the first error encountered during writing, or nil.
func (w *Writer) Err() error { return w.err }
