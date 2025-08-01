package temporalproto

import (
	"google.golang.org/protobuf/proto"

	"go.temporal.io/api/internal/protojson"
)

// CustomJSONMarshalOptions is a configurable JSON format marshaler that supports
// both TYPE_PREFIXED_SCREAMING_SNAKE enums as well as camelCase enums.
type CustomJSONMarshalOptions struct {
	// Metadata is used for storing request metadata, such as whether shorthand
	// payloads are disabled
	Metadata map[string]interface{}

	// Indent specifies the set of indentation characters to use in a multiline
	// formatted output such that every entry is preceded by Indent and
	// terminated by a newline. If non-empty, then Multiline is treated as true.
	// Indent can only be composed of space or tab characters.
	Indent string
}

// Marshal marshals the given [proto.Message] in the JSON format using options in
// MarshalOptions. Do not depend on the output being stable. It may change over
// time across different versions of the program.
func (o CustomJSONMarshalOptions) Marshal(m proto.Message) ([]byte, error) {
	return protojson.MarshalOptions{
		Indent:   o.Indent,
		Metadata: o.Metadata,
	}.Marshal(m)
}
