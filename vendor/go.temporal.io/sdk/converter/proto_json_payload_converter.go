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
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"

	gogojsonpb "github.com/gogo/protobuf/jsonpb"
	gogoproto "github.com/gogo/protobuf/proto"
	commonpb "go.temporal.io/api/common/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// ProtoJSONPayloadConverter converts proto objects to/from JSON.
type ProtoJSONPayloadConverter struct {
	gogoMarshaler         gogojsonpb.Marshaler
	gogoUnmarshaler       gogojsonpb.Unmarshaler
	protoUnmarshalOptions protojson.UnmarshalOptions
	options               ProtoJSONPayloadConverterOptions
}

// ProtoJSONPayloadConverterOptions represents options for `NewProtoJSONPayloadConverterWithOptions`.
type ProtoJSONPayloadConverterOptions struct {
	// ExcludeProtobufMessageTypes prevents the message type (`my.package.MyMessage`)
	// from being included in the Payload.
	ExcludeProtobufMessageTypes bool

	// AllowUnknownFields will ignore unknown fields when unmarshalling, as opposed to returning an error
	AllowUnknownFields bool
}

var (
	jsonNil, _ = json.Marshal(nil)
)

// NewProtoJSONPayloadConverter creates new instance of `ProtoJSONPayloadConverter`.
func NewProtoJSONPayloadConverter() *ProtoJSONPayloadConverter {
	return &ProtoJSONPayloadConverter{
		gogoMarshaler:         gogojsonpb.Marshaler{},
		gogoUnmarshaler:       gogojsonpb.Unmarshaler{},
		protoUnmarshalOptions: protojson.UnmarshalOptions{},
	}
}

// NewProtoJSONPayloadConverterWithOptions creates new instance of `ProtoJSONPayloadConverter` with the provided options.
func NewProtoJSONPayloadConverterWithOptions(options ProtoJSONPayloadConverterOptions) *ProtoJSONPayloadConverter {
	return &ProtoJSONPayloadConverter{
		gogoMarshaler: gogojsonpb.Marshaler{},
		gogoUnmarshaler: gogojsonpb.Unmarshaler{
			AllowUnknownFields: options.AllowUnknownFields,
		},
		protoUnmarshalOptions: protojson.UnmarshalOptions{
			DiscardUnknown: options.AllowUnknownFields,
		},
		options: options,
	}
}

// ToPayload converts single proto value to payload.
func (c *ProtoJSONPayloadConverter) ToPayload(value interface{}) (*commonpb.Payload, error) {
	// Proto golang structs might be generated with 4 different protoc plugin versions:
	//   1. github.com/golang/protobuf - ~v1.3.5 is the most recent pre-APIv2 version of APIv1.
	//   2. github.com/golang/protobuf - ^v1.4.0 is a version of APIv1 implemented in terms of APIv2.
	//   3. google.golang.org/protobuf - ^v1.20.0 is APIv2.
	//   4. github.com/gogo/protobuf - any version.
	// Case 1 is not supported.
	// Cases 2 and 3 implements proto.Message and are the same in this context.
	// Case 4 implements gogoproto.Message.
	// It is important to check for proto.Message first because cases 2 and 3 also implement gogoproto.Message.

	if isInterfaceNil(value) {
		return newPayload(jsonNil, c), nil
	}

	builtPointer := false
	for {
		if valueProto, ok := value.(proto.Message); ok {
			byteSlice, err := protojson.Marshal(valueProto)
			if err != nil {
				return nil, fmt.Errorf("%w: %v", ErrUnableToEncode, err)
			}
			return newProtoPayload(byteSlice, c, string(valueProto.ProtoReflect().Descriptor().FullName())), nil
		}
		if valueGogoProto, ok := value.(gogoproto.Message); ok {
			var buf bytes.Buffer
			err := c.gogoMarshaler.Marshal(&buf, valueGogoProto)
			if err != nil {
				return nil, fmt.Errorf("%w: %v", ErrUnableToEncode, err)
			}
			return newProtoPayload(buf.Bytes(), c, gogoproto.MessageName(valueGogoProto)), nil
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
func (c *ProtoJSONPayloadConverter) FromPayload(payload *commonpb.Payload, valuePtr interface{}) error {
	originalValue := reflect.ValueOf(valuePtr)
	if originalValue.Kind() != reflect.Ptr {
		return fmt.Errorf("type: %T: %w", valuePtr, ErrValuePtrIsNotPointer)
	}

	originalValue = originalValue.Elem()
	if !originalValue.CanSet() {
		return fmt.Errorf("type: %T: %w", valuePtr, ErrUnableToSetValue)
	}

	if bytes.Equal(payload.GetData(), jsonNil) {
		originalValue.Set(reflect.Zero(originalValue.Type()))
		return nil
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
		err = c.protoUnmarshalOptions.Unmarshal(payload.GetData(), protoMessage)
	} else if isGogoProtoMessage {
		err = c.gogoUnmarshaler.Unmarshal(bytes.NewReader(payload.GetData()), gogoProtoMessage)
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
func (c *ProtoJSONPayloadConverter) ToString(payload *commonpb.Payload) string {
	return string(payload.GetData())
}

// Encoding returns MetadataEncodingProtoJSON.
func (c *ProtoJSONPayloadConverter) Encoding() string {
	return MetadataEncodingProtoJSON
}

func (c *ProtoJSONPayloadConverter) ExcludeProtobufMessageTypes() bool {
	return c.options.ExcludeProtobufMessageTypes
}
