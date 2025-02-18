// The MIT License
//
// Copyright (c) 2022 Temporal Technologies Inc.  All rights reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
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
