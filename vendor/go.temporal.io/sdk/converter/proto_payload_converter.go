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

	gogoproto "github.com/gogo/protobuf/proto"
	commonpb "go.temporal.io/api/common/v1"
	"google.golang.org/protobuf/proto"
)

// ProtoPayloadConverter converts proto objects to protobuf binary format.
type ProtoPayloadConverter struct {
	options ProtoPayloadConverterOptions
}

// ProtoPayloadConverterOptions represents options for `NewProtoPayloadConverterWithOptions`.
type ProtoPayloadConverterOptions struct {
	// ExcludeProtobufMessageTypes prevents the message type (`my.package.MyMessage`)
	// from being included in the Payload.
	ExcludeProtobufMessageTypes bool
}

// NewProtoPayloadConverter creates new instance of `ProtoPayloadConverter``.
func NewProtoPayloadConverter() *ProtoPayloadConverter {
	return &ProtoPayloadConverter{}
}

// NewProtoPayloadConverterWithOptions creates new instance of `ProtoPayloadConverter` with the provided options.
func NewProtoPayloadConverterWithOptions(options ProtoPayloadConverterOptions) *ProtoPayloadConverter {
	return &ProtoPayloadConverter{
		options: options,
	}
}

// ToPayload converts single proto value to payload.
func (c *ProtoPayloadConverter) ToPayload(value interface{}) (*commonpb.Payload, error) {
	// Proto golang structs might be generated with 4 different protoc plugin versions:
	//   1. github.com/golang/protobuf - ~v1.3.5 is the most recent pre-APIv2 version of APIv1.
	//   2. github.com/golang/protobuf - ^v1.4.0 is a version of APIv1 implemented in terms of APIv2.
	//   3. google.golang.org/protobuf - ^v1.20.0 is APIv2.
	//   4. github.com/gogo/protobuf - any version.
	// Case 1 is not supported.
	// Cases 2 and 3 implements proto.Message and are the same in this context.
	// Case 4 implements gogoproto.Message.
	// It is important to check for proto.Message first because cases 2 and 3 also implements gogoproto.Message.

	builtPointer := false
	for {
		if valueProto, ok := value.(proto.Message); ok {
			byteSlice, err := proto.Marshal(valueProto)
			if err != nil {
				return nil, fmt.Errorf("%w: %v", ErrUnableToEncode, err)
			}
			return newProtoPayload(byteSlice, c, string(valueProto.ProtoReflect().Descriptor().FullName())), nil
		}
		if valueGogoProto, ok := value.(gogoproto.Message); ok {
			data, err := gogoproto.Marshal(valueGogoProto)
			if err != nil {
				return nil, fmt.Errorf("%w: %v", ErrUnableToEncode, err)
			}
			return newProtoPayload(data, c, gogoproto.MessageName(valueGogoProto)), nil
		}
		if builtPointer {
			break
		}
		value = pointerTo(value).Interface()
		builtPointer = true
	}

	return nil, nil
}

// FromPayload converts single proto value from payload.
func (c *ProtoPayloadConverter) FromPayload(payload *commonpb.Payload, valuePtr interface{}) error {
	originalValue := reflect.ValueOf(valuePtr)
	if originalValue.Kind() != reflect.Ptr {
		return fmt.Errorf("type: %T: %w", valuePtr, ErrValuePtrIsNotPointer)
	}

	originalValue = originalValue.Elem()
	if !originalValue.CanSet() {
		return fmt.Errorf("type: %T: %w", valuePtr, ErrUnableToSetValue)
	}

	if originalValue.Kind() == reflect.Interface {
		return fmt.Errorf("value type: %s: %w", originalValue.Type().String(), ErrValuePtrMustConcreteType)
	}

	value := originalValue
	// If original value is of value type (i.e. commonpb.WorkflowType), create a pointer to it.
	if originalValue.Kind() != reflect.Ptr {
		value = pointerTo(originalValue.Interface())
	}

	protoValue := value.Interface() // protoValue is for sure of pointer type (i.e. *commonpb.WorkflowType).
	gogoProtoMessage, isGogoProtoMessage := protoValue.(gogoproto.Message)
	protoMessage, isProtoMessage := protoValue.(proto.Message)
	if !isGogoProtoMessage && !isProtoMessage {
		return fmt.Errorf("type: %T: %w", protoValue, ErrTypeNotImplementProtoMessage)
	}

	// If original value is nil, create new instance.
	if originalValue.Kind() == reflect.Ptr && originalValue.IsNil() {
		value = newOfSameType(originalValue)
		protoValue = value.Interface()
		if isProtoMessage {
			protoMessage = protoValue.(proto.Message) // type assertion must always succeed
		} else if isGogoProtoMessage {
			gogoProtoMessage = protoValue.(gogoproto.Message) // type assertion must always succeed
		}
	}

	var err error
	if isProtoMessage {
		err = proto.Unmarshal(payload.GetData(), protoMessage)
	} else if isGogoProtoMessage {
		err = gogoproto.Unmarshal(payload.GetData(), gogoProtoMessage)
	}
	// If original value wasn't a pointer then set value back to where valuePtr points to.
	if originalValue.Kind() != reflect.Ptr {
		originalValue.Set(value.Elem())
	}

	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnableToDecode, err)
	}

	return nil
}

// ToString converts payload object into human readable string.
func (c *ProtoPayloadConverter) ToString(payload *commonpb.Payload) string {
	// We can't do anything better here.
	return base64.RawStdEncoding.EncodeToString(payload.GetData())
}

// Encoding returns MetadataEncodingProto.
func (c *ProtoPayloadConverter) Encoding() string {
	return MetadataEncodingProto
}

func (c *ProtoPayloadConverter) ExcludeProtobufMessageTypes() bool {
	return c.options.ExcludeProtobufMessageTypes
}
