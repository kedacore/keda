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
