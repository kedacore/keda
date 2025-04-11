/*
 Copyright 2021 The CloudEvents Authors
 SPDX-License-Identifier: Apache-2.0
*/

package datacodec

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudevents/sdk-go/v2/event/datacodec/json"
	"github.com/cloudevents/sdk-go/v2/event/datacodec/text"
	"github.com/cloudevents/sdk-go/v2/event/datacodec/xml"
)

// Decoder is the expected function signature for decoding `in` to `out`.
// If Event sent the payload as base64, Decoder assumes that `in` is the
// decoded base64 byte array.
type Decoder func(ctx context.Context, in []byte, out interface{}) error

// Encoder is the expected function signature for encoding `in` to bytes.
// Returns an error if the encoder has an issue encoding `in`.
type Encoder func(ctx context.Context, in interface{}) ([]byte, error)

var decoder map[string]Decoder
var encoder map[string]Encoder

// ssDecoder is a map of content-type structured suffixes as defined in
// [Structured Syntax Suffixes](https://www.iana.org/assignments/media-type-structured-suffix/media-type-structured-suffix.xhtml),
// which may be used to match content types such as application/vnd.custom-app+json
var ssDecoder map[string]Decoder

// ssEncoder is a map of content-type structured suffixes similar to ssDecoder.
var ssEncoder map[string]Encoder

func init() {
	decoder = make(map[string]Decoder, 10)
	ssDecoder = make(map[string]Decoder, 10)

	encoder = make(map[string]Encoder, 10)
	ssEncoder = make(map[string]Encoder, 10)

	AddDecoder("", json.Decode)
	AddDecoder("application/json", json.Decode)
	AddDecoder("text/json", json.Decode)
	AddDecoder("application/xml", xml.Decode)
	AddDecoder("text/xml", xml.Decode)
	AddDecoder("text/plain", text.Decode)

	AddStructuredSuffixDecoder("json", json.Decode)
	AddStructuredSuffixDecoder("xml", xml.Decode)

	AddEncoder("", json.Encode)
	AddEncoder("application/json", json.Encode)
	AddEncoder("text/json", json.Encode)
	AddEncoder("application/xml", xml.Encode)
	AddEncoder("text/xml", xml.Encode)
	AddEncoder("text/plain", text.Encode)

	AddStructuredSuffixEncoder("json", json.Encode)
	AddStructuredSuffixEncoder("xml", xml.Encode)
}

// AddDecoder registers a decoder for a given content type. The codecs will use
// these to decode the data payload from a cloudevent.Event object.
func AddDecoder(contentType string, fn Decoder) {
	decoder[contentType] = fn
}

// AddStructuredSuffixDecoder registers a decoder for content-types which match the given structured
// syntax suffix as defined by
// [Structured Syntax Suffixes](https://www.iana.org/assignments/media-type-structured-suffix/media-type-structured-suffix.xhtml).
// This allows users to register custom decoders for non-standard content types which follow the
// structured syntax suffix standard (e.g. application/vnd.custom-app+json).
//
// Suffix should not include the "+" character, and "json" and "xml" are registered by default.
func AddStructuredSuffixDecoder(suffix string, fn Decoder) {
	ssDecoder[suffix] = fn
}

// AddEncoder registers an encoder for a given content type. The codecs will
// use these to encode the data payload for a cloudevent.Event object.
func AddEncoder(contentType string, fn Encoder) {
	encoder[contentType] = fn
}

// AddStructuredSuffixEncoder registers an encoder for content-types which match the given
// structured syntax suffix as defined by
// [Structured Syntax Suffixes](https://www.iana.org/assignments/media-type-structured-suffix/media-type-structured-suffix.xhtml).
// This allows users to register custom encoders for non-standard content types which follow the
// structured syntax suffix standard (e.g. application/vnd.custom-app+json).
//
// Suffix should not include the "+" character, and "json" and "xml" are registered by default.
func AddStructuredSuffixEncoder(suffix string, fn Encoder) {
	ssEncoder[suffix] = fn
}

// Decode looks up and invokes the decoder registered for the given content
// type. An error is returned if no decoder is registered for the given
// content type.
func Decode(ctx context.Context, contentType string, in []byte, out interface{}) error {
	if fn, ok := decoder[contentType]; ok {
		return fn(ctx, in, out)
	}

	if fn, ok := ssDecoder[structuredSuffix(contentType)]; ok {
		return fn(ctx, in, out)
	}

	return fmt.Errorf("[decode] unsupported content type: %q", contentType)
}

// Encode looks up and invokes the encoder registered for the given content
// type. An error is returned if no encoder is registered for the given
// content type.
func Encode(ctx context.Context, contentType string, in interface{}) ([]byte, error) {
	if fn, ok := encoder[contentType]; ok {
		return fn(ctx, in)
	}

	if fn, ok := ssEncoder[structuredSuffix(contentType)]; ok {
		return fn(ctx, in)
	}

	return nil, fmt.Errorf("[encode] unsupported content type: %q", contentType)
}

func structuredSuffix(contentType string) string {
	parts := strings.Split(contentType, "+")
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}

	return ""
}
