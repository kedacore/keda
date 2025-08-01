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
