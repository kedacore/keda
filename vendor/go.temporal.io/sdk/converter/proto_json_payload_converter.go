package converter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"

	gogojsonpb "github.com/gogo/protobuf/jsonpb"
	gogoproto "github.com/gogo/protobuf/proto"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/temporalproto"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// ProtoJSONPayloadConverter converts proto objects to/from JSON.
type ProtoJSONPayloadConverter struct {
	gogoMarshaler                 gogojsonpb.Marshaler
	gogoUnmarshaler               gogojsonpb.Unmarshaler
	protoMarshalOptions           protojson.MarshalOptions
	protoUnmarshalOptions         protojson.UnmarshalOptions
	temporalProtoUnmarshalOptions temporalproto.CustomJSONUnmarshalOptions
	options                       ProtoJSONPayloadConverterOptions
}

// ProtoJSONPayloadConverterOptions represents options for `NewProtoJSONPayloadConverterWithOptions`.
type ProtoJSONPayloadConverterOptions struct {
	// ExcludeProtobufMessageTypes prevents the message type (`my.package.MyMessage`)
	// from being included in the Payload.
	ExcludeProtobufMessageTypes bool

	// AllowUnknownFields will ignore unknown fields when unmarshalling, as opposed to returning an error
	AllowUnknownFields bool

	// UseProtoNames uses proto field name instead of lowerCamelCase name in JSON
	// field names.
	UseProtoNames bool

	// UseEnumNumbers emits enum values as numbers.
	UseEnumNumbers bool

	// EmitUnpopulated specifies whether to emit unpopulated fields.
	EmitUnpopulated bool

	// LegacyTemporalProtoCompat will allow enums serialized as SCREAMING_SNAKE_CASE.
	// Useful for backwards compatibility when migrating a proto message from gogoproto to standard protobuf.
	LegacyTemporalProtoCompat bool
}

var (
	jsonNil, _ = json.Marshal(nil)
)

// NewProtoJSONPayloadConverter creates new instance of `ProtoJSONPayloadConverter`.
func NewProtoJSONPayloadConverter() *ProtoJSONPayloadConverter {
	return &ProtoJSONPayloadConverter{
		gogoMarshaler:                 gogojsonpb.Marshaler{},
		gogoUnmarshaler:               gogojsonpb.Unmarshaler{},
		protoMarshalOptions:           protojson.MarshalOptions{},
		protoUnmarshalOptions:         protojson.UnmarshalOptions{},
		temporalProtoUnmarshalOptions: temporalproto.CustomJSONUnmarshalOptions{},
	}
}

// NewProtoJSONPayloadConverterWithOptions creates new instance of `ProtoJSONPayloadConverter` with the provided options.
func NewProtoJSONPayloadConverterWithOptions(options ProtoJSONPayloadConverterOptions) *ProtoJSONPayloadConverter {
	return &ProtoJSONPayloadConverter{
		gogoMarshaler: gogojsonpb.Marshaler{
			EnumsAsInts:  options.UseEnumNumbers,
			EmitDefaults: options.EmitUnpopulated,
			OrigName:     options.UseProtoNames,
		},
		gogoUnmarshaler: gogojsonpb.Unmarshaler{
			AllowUnknownFields: options.AllowUnknownFields,
		},
		protoMarshalOptions: protojson.MarshalOptions{
			UseProtoNames:   options.UseProtoNames,
			UseEnumNumbers:  options.UseEnumNumbers,
			EmitUnpopulated: options.EmitUnpopulated,
		},
		protoUnmarshalOptions: protojson.UnmarshalOptions{
			DiscardUnknown: options.AllowUnknownFields,
		},
		temporalProtoUnmarshalOptions: temporalproto.CustomJSONUnmarshalOptions{
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
			byteSlice, err := c.protoMarshalOptions.Marshal(valueProto)
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
		if c.options.LegacyTemporalProtoCompat {
			err = c.temporalProtoUnmarshalOptions.Unmarshal(payload.GetData(), protoMessage)
		} else {
			err = c.protoUnmarshalOptions.Unmarshal(payload.GetData(), protoMessage)
		}
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
