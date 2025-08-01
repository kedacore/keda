package converter

import (
	"encoding/json"
	"fmt"

	commonpb "go.temporal.io/api/common/v1"
)

// JSONPayloadConverter converts to/from JSON.
type JSONPayloadConverter struct {
}

// NewJSONPayloadConverter creates a new instance of JSONPayloadConverter.
func NewJSONPayloadConverter() *JSONPayloadConverter {
	return &JSONPayloadConverter{}
}

// ToPayload converts a single value to a payload.
func (c *JSONPayloadConverter) ToPayload(value interface{}) (*commonpb.Payload, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnableToEncode, err)
	}
	return newPayload(data, c), nil
}

// FromPayload converts a single payload to a value.
func (c *JSONPayloadConverter) FromPayload(payload *commonpb.Payload, valuePtr interface{}) error {
	err := json.Unmarshal(payload.GetData(), valuePtr)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnableToDecode, err)
	}
	return nil
}

// ToString converts a payload object into a human-readable string.
func (c *JSONPayloadConverter) ToString(payload *commonpb.Payload) string {
	return string(payload.GetData())
}

// Encoding returns MetadataEncodingJSON.
func (c *JSONPayloadConverter) Encoding() string {
	return MetadataEncodingJSON
}
