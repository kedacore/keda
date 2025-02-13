// The MIT License
//
// Copyright (c) 2022 Temporal Technologies Inc.  All rights reserved.
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

package protojson

import "go.temporal.io/api/internal/protojson/json"

// ProtoJSONMaybeMarshaler is implemented by any proto struct that wants to
// customize optional Temporal-specific JSON conversion.
type ProtoJSONMaybeMarshaler interface {
	// MaybeMarshalProtoJSON is for formatting the proto message as JSON. If the
	// "handled" result value is false, "err" are ignored and the default
	// protojson behavior occurs.
	MaybeMarshalProtoJSON(meta map[string]interface{}, enc *json.Encoder) (handled bool, err error)
}

// ProtoJSONMaybeUnmarshaler is implemented by any proto struct that wants to
// customize optional Temporal-specific JSON conversion.
type ProtoJSONMaybeUnmarshaler interface {
	// MaybeUnmarshalProtoJSON is for parsing the given JSON into the proto message.
	// If the "handled" result value is false, "err" is ignored and the default
	// protojson unmarshaling proceeds
	MaybeUnmarshalProtoJSON(meta map[string]interface{}, dec *json.Decoder) (handled bool, err error)
}
