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
	commonpb "go.temporal.io/api/common/v1"
)

// PayloadConverter is an interface to convert a single payload.
type PayloadConverter interface {
	// ToPayload converts single value to payload. It should return nil if the PayloadConveter can not convert passed value (i.e. type is unknown).
	ToPayload(value interface{}) (*commonpb.Payload, error)
	// FromPayload converts single value from payload. valuePtr should be a reference to the variable of the type that is corresponding for payload encoding.
	// Otherwise it should return error.
	FromPayload(payload *commonpb.Payload, valuePtr interface{}) error
	// ToString converts payload object into human readable string.
	ToString(*commonpb.Payload) string

	// Encoding returns encoding supported by PayloadConverter.
	Encoding() string
}

type protoPayloadConverterInterface interface {
	PayloadConverter
	ExcludeProtobufMessageTypes() bool
}

func newPayload(data []byte, c PayloadConverter) *commonpb.Payload {
	return &commonpb.Payload{
		Metadata: map[string][]byte{
			MetadataEncoding: []byte(c.Encoding()),
		},
		Data: data,
	}
}

func newProtoPayload(data []byte, c protoPayloadConverterInterface, messageType string) *commonpb.Payload {
	if !c.ExcludeProtobufMessageTypes() {
		return &commonpb.Payload{
			Metadata: map[string][]byte{
				MetadataEncoding:    []byte(c.Encoding()),
				MetadataMessageType: []byte(messageType),
			},
			Data: data,
		}
	}
	return newPayload(data, c)
}
