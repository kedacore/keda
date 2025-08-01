package temporalproto

import (
	"go.temporal.io/api/internal/protojson"
	"google.golang.org/protobuf/proto"
)

// CustomJSONUnmarshalOptions is a configurable JSON format marshaler that supports
// both TYPE_PREFIXED_SCREAMING_SNAKE enums as well as camelCase enums.
type CustomJSONUnmarshalOptions struct {
	// Metadata is used for storing request metadata, such as whether shorthand
	// payloads are disabled
	Metadata map[string]interface{}

	// If DiscardUnknown is set, unknown fields and enum name values are ignored.
	DiscardUnknown bool
}

// Unmarshal reads the given []byte and populates the given [proto.Message]
// using options in the UnmarshalOptions object.
// It will clear the message first before setting the fields.
// If it returns an error, the given message may be partially set.
// The provided message must be mutable (e.g., a non-nil pointer to a message).
// This is different from the official protojson unmarshaling code in that it
// supports unmarshaling our shorthand payload format as well as both camelCase
// and SCREAMING_SNAKE_CASE JSON enums
func (o CustomJSONUnmarshalOptions) Unmarshal(b []byte, m proto.Message) error {
	return protojson.UnmarshalOptions{
		Metadata:       o.Metadata,
		DiscardUnknown: o.DiscardUnknown,
	}.Unmarshal(b, m)
}
