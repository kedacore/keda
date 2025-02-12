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
	"fmt"
	"reflect"

	commonpb "go.temporal.io/api/common/v1"
)

// NilPayloadConverter doesn't set Data field in payload.
type NilPayloadConverter struct {
}

// NewNilPayloadConverter creates new instance of NilPayloadConverter.
func NewNilPayloadConverter() *NilPayloadConverter {
	return &NilPayloadConverter{}
}

// ToPayload converts single nil value to payload.
func (c *NilPayloadConverter) ToPayload(value interface{}) (*commonpb.Payload, error) {
	if isInterfaceNil(value) {
		return newPayload(nil, c), nil
	}
	return nil, nil
}

// FromPayload converts single nil value from payload.
func (c *NilPayloadConverter) FromPayload(_ *commonpb.Payload, valuePtr interface{}) error {
	originalValue := reflect.ValueOf(valuePtr)
	if originalValue.Kind() != reflect.Ptr {
		return fmt.Errorf("type: %T: %w", valuePtr, ErrValuePtrIsNotPointer)
	}

	originalValue = originalValue.Elem()
	if !originalValue.CanSet() {
		return fmt.Errorf("type: %T: %w", valuePtr, ErrUnableToSetValue)
	}

	originalValue.Set(reflect.Zero(originalValue.Type()))
	return nil
}

// ToString converts payload object into human readable string.
func (c *NilPayloadConverter) ToString(*commonpb.Payload) string {
	return "nil"
}

// Encoding returns MetadataEncodingNil.
func (c *NilPayloadConverter) Encoding() string {
	return MetadataEncodingNil
}
