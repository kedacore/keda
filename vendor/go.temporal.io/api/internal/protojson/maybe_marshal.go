package protojson

import "go.temporal.io/api/internal/protojson/json"

// ProtoJSONMaybeMarshaler is implemented by any proto struct that wants to
// customize optional Temporal-specific JSON conversion.
type ProtoJSONMaybeMarshaler interface {
	// MaybeMarshalProtoJSON is for formatting the proto message as JSON. If the
	// "handled" result value is false, "err" are ignored and the default
	// protojson behavior occurs.
	MaybeMarshalProtoJSON(meta map[string]interface{}, enc *json.Encoder) (handled bool, err error)
}

// ProtoJSONMaybeUnmarshaler is implemented by any proto struct that wants to
// customize optional Temporal-specific JSON conversion.
type ProtoJSONMaybeUnmarshaler interface {
	// MaybeUnmarshalProtoJSON is for parsing the given JSON into the proto message.
	// If the "handled" result value is false, "err" is ignored and the default
	// protojson unmarshaling proceeds
	MaybeUnmarshalProtoJSON(meta map[string]interface{}, dec *json.Decoder) (handled bool, err error)
}
