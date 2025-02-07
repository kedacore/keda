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
