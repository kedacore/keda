package converter

import (
	commonpb "go.temporal.io/api/common/v1"
)

// PayloadConverter is an interface to convert a single payload.
type PayloadConverter interface {
	// ToPayload converts a single value to payload. It should return nil if the
	// PayloadConverter can not convert the passed value (i.e. type is unknown).
	ToPayload(value interface{}) (*commonpb.Payload, error)
	// FromPayload converts single value from payload. valuePtr should be a
	// reference to a variable of a type corresponding to the payload
	// encoding. Otherwise it should return error.
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
