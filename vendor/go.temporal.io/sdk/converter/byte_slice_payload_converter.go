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

package converter

import (
	"encoding/base64"
	"fmt"
	"reflect"

	commonpb "go.temporal.io/api/common/v1"
)

// ByteSlicePayloadConverter pass through []byte to Data field in payload.
type ByteSlicePayloadConverter struct {
}

// NewByteSlicePayloadConverter creates new instance of ByteSlicePayloadConverter.
func NewByteSlicePayloadConverter() *ByteSlicePayloadConverter {
	return &ByteSlicePayloadConverter{}
}

// ToPayload converts single []byte value to payload.
func (c *ByteSlicePayloadConverter) ToPayload(value interface{}) (*commonpb.Payload, error) {
	if valueBytes, isByteSlice := value.([]byte); isByteSlice {
		return newPayload(valueBytes, c), nil
	}

	return nil, nil
}

// FromPayload converts single []byte value from payload.
func (c *ByteSlicePayloadConverter) FromPayload(payload *commonpb.Payload, valuePtr interface{}) error {
	rv := reflect.ValueOf(valuePtr)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("type: %T: %w", valuePtr, ErrValuePtrIsNotPointer)
	}
	v := rv.Elem()
	value := payload.Data
	if v.Kind() == reflect.Interface {
		v.Set(reflect.ValueOf(value))
	} else if v.Kind() == reflect.Slice && v.Type().Elem().Kind() == reflect.Uint8 {
		// Must be a []byte.
		v.SetBytes(value)
	} else {
		return fmt.Errorf("type %T: %w", valuePtr, ErrTypeIsNotByteSlice)
	}
	return nil
}

// ToString converts payload object into human readable string.
func (c *ByteSlicePayloadConverter) ToString(payload *commonpb.Payload) string {
	var byteSlice []byte
	err := c.FromPayload(payload, &byteSlice)
	if err != nil {
		return err.Error()
	}
	return base64.RawStdEncoding.EncodeToString(byteSlice)
}

// Encoding returns MetadataEncodingBinary.
func (c *ByteSlicePayloadConverter) Encoding() string {
	return MetadataEncodingBinary
}
