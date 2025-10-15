package serializer

import (
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type (
	// JSONPBEncoder is JSON encoder/decoder for protobuf structs and slices of protobuf structs.
	JSONPBEncoder struct {
		opts protojson.MarshalOptions
	}
)

// NewJSONPBEncoder creates a new JSONPBEncoder.
func NewJSONPBEncoder() JSONPBEncoder {
	return JSONPBEncoder{}
}

// NewJSONPBIndentEncoder creates a new JSONPBEncoder with indent.
func NewJSONPBIndentEncoder(indent string) JSONPBEncoder {
	return JSONPBEncoder{
		opts: protojson.MarshalOptions{
			Indent: indent,
		},
	}
}

// Encode protobuf struct to bytes.
func (e JSONPBEncoder) Encode(pb proto.Message) ([]byte, error) {
	return e.opts.Marshal(pb)
}

// Decode bytes to protobuf struct.
func (e JSONPBEncoder) Decode(data []byte, pb proto.Message) error {
	return protojson.Unmarshal(data, pb)
}
