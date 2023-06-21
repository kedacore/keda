// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package serializer

import (
	"bytes"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
)

type (
	// JSONPBEncoder is JSON encoder/decoder for protobuf structs and slices of protobuf structs.
	// This is an wrapper on top of jsonpb.Marshaler which supports not only single object serialization
	// but also slices of concrete objects.
	JSONPBEncoder struct {
		marshaler   jsonpb.Marshaler
		unmarshaler jsonpb.Unmarshaler
	}
)

// NewJSONPBEncoder creates a new JSONPBEncoder.
func NewJSONPBEncoder() *JSONPBEncoder {
	return &JSONPBEncoder{
		marshaler:   jsonpb.Marshaler{},
		unmarshaler: jsonpb.Unmarshaler{},
	}
}

// NewJSONPBIndentEncoder creates a new JSONPBEncoder with indent.
func NewJSONPBIndentEncoder(indent string) *JSONPBEncoder {
	return &JSONPBEncoder{
		marshaler:   jsonpb.Marshaler{Indent: indent},
		unmarshaler: jsonpb.Unmarshaler{},
	}
}

// Encode protobuf struct to bytes.
func (e *JSONPBEncoder) Encode(pb proto.Message) ([]byte, error) {
	var buf bytes.Buffer
	err := e.marshaler.Marshal(&buf, pb)
	return buf.Bytes(), err
}

// Decode bytes to protobuf struct.
func (e *JSONPBEncoder) Decode(data []byte, pb proto.Message) error {
	return e.unmarshaler.Unmarshal(bytes.NewReader(data), pb)
}
