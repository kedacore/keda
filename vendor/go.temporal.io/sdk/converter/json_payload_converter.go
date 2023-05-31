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
	"encoding/json"
	"fmt"

	commonpb "go.temporal.io/api/common/v1"
)

// JSONPayloadConverter converts to/from JSON.
type JSONPayloadConverter struct {
}

// NewJSONPayloadConverter creates new instance of JSONPayloadConverter.
func NewJSONPayloadConverter() *JSONPayloadConverter {
	return &JSONPayloadConverter{}
}

// ToPayload converts single value to payload.
func (c *JSONPayloadConverter) ToPayload(value interface{}) (*commonpb.Payload, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnableToEncode, err)
	}
	return newPayload(data, c), nil
}

// FromPayload converts single value from payload.
func (c *JSONPayloadConverter) FromPayload(payload *commonpb.Payload, valuePtr interface{}) error {
	err := json.Unmarshal(payload.GetData(), valuePtr)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnableToDecode, err)
	}
	return nil
}

// ToString converts payload object into human readable string.
func (c *JSONPayloadConverter) ToString(payload *commonpb.Payload) string {
	return string(payload.GetData())
}

// Encoding returns MetadataEncodingJSON.
func (c *JSONPayloadConverter) Encoding() string {
	return MetadataEncodingJSON
}
