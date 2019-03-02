/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cloudevents

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"reflect"
)

const (
	// ContentTypeStructuredJSON is the content-type for "Structured" encoding
	// where an event envelope is written in JSON and the body is arbitrary
	// data which might be an alternate encoding.
	ContentTypeStructuredJSON = "application/cloudevents+json"

	// ContentTypeBinaryJSON is the content-type for "Binary" encoding where
	// the event context is in HTTP headers and the body is a JSON event data.
	ContentTypeBinaryJSON = "application/json"

	// TODO(inlined) what about charset additions?
	contentTypeJSON = "application/json"
	contentTypeXML  = "application/xml"

	// HeaderContentType is the standard HTTP header "Content-Type"
	HeaderContentType = "Content-Type"

	// CloudEventsVersion is a legacy alias of V01CloudEventsVersion, for compatibility.
	CloudEventsVersion = V01CloudEventsVersion
)

// EventContext is a legacy un-versioned alias, from when we thought that field names would stay the same.
type EventContext = V01EventContext

// HTTPMarshaller implements a scheme for decoding CloudEvents over HTTP.
// Implementations are Binary, Structured, and Any
type HTTPMarshaller interface {
	FromRequest(data interface{}, r *http.Request) (LoadContext, error)
	NewRequest(urlString string, data interface{}, context SendContext) (*http.Request, error)
}

// ContextTranslator provides a set of translation methods between the
// different versions of the CloudEvents spec, which allows programs to
// interoperate with different versions of the CloudEvents spec by
// converting EventContexts to their preferred version.
type ContextTranslator interface {
	// AsV01 provides a translation from whatever the "native" encoding of the
	// CloudEvent was to the equivalent in v0.1 field names, moving fields to or
	// from extensions as necessary.
	AsV01() V01EventContext

	// AsV02 provides a translation from whatever the "native" encoding of the
	// CloudEvent was to the equivalent in v0.2 field names, moving fields to or
	// from extensions as necessary.
	AsV02() V02EventContext

	// DataContentType returns the MIME content type for encoding data, which is
	// needed by both encoding and decoding.
	DataContentType() string
}

// SendContext provides an interface for extracting information from an
// EventContext (the set of non-data event attributes of a CloudEvent).
type SendContext interface {
	ContextTranslator

	StructuredSender
	BinarySender
}

// LoadContext provides an interface for extracting information from an
// EventContext (the set of non-data event attributes of a CloudEvent).
type LoadContext interface {
	ContextTranslator

	StructuredLoader
	BinaryLoader
}

// ContextType is a unified interface for both sending and loading the
// CloudEvent data across versions.
type ContextType interface {
	ContextTranslator

	StructuredSender
	BinarySender

	StructuredLoader
	BinaryLoader
}

func anyError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func require(name string, value string) error {
	if len(value) == 0 {
		return fmt.Errorf("missing required field %q", name)
	}
	return nil
}

// The Cloud-Events spec allows two forms of JSON encoding:
// 1. The overall message (Structured JSON encoding)
// 2. Just the event data, where the context will be in HTTP headers instead
//
// Case #1 actually includes case #2. In structured binary encoding the JSON
// HTTP body itself allows for cross-encoding of the "data" field.
// This method is only intended for checking that inner JSON encoding type.
func isJSONEncoding(encoding string) bool {
	return encoding == contentTypeJSON || encoding == "text/json"
}

func isXMLEncoding(encoding string) bool {
	return encoding == contentTypeXML || encoding == "text/xml"
}

func unmarshalEventData(encoding string, reader io.Reader, data interface{}) error {
	// The Handler tools allow developers to not ask for event data;
	// in this case, just don't unmarshal anything
	if data == nil {
		return nil
	}

	// If someone tried to marshal an event into an io.Reader, just assign our existing reader.
	// (This is used by event.Mux to determine which type to unmarshal as)
	readerPtrType := reflect.TypeOf((*io.Reader)(nil))
	if reflect.TypeOf(data).ConvertibleTo(readerPtrType) {
		reflect.ValueOf(data).Elem().Set(reflect.ValueOf(reader))
		return nil
	}
	if isJSONEncoding(encoding) || encoding == "" {
		return json.NewDecoder(reader).Decode(&data)
	}

	if isXMLEncoding(encoding) {
		return xml.NewDecoder(reader).Decode(&data)
	}

	return fmt.Errorf("Cannot decode content type %q", encoding)
}

func marshalEventData(encoding string, data interface{}) ([]byte, error) {
	var b []byte
	var err error

	if isJSONEncoding(encoding) {
		b, err = json.Marshal(data)
	} else if isXMLEncoding(encoding) {
		b, err = xml.Marshal(data)
	} else {
		err = fmt.Errorf("Cannot encode content type %q", encoding)
	}

	if err != nil {
		return nil, err
	}
	return b, nil
}

// FromRequest parses a CloudEvent from any known encoding.
func FromRequest(data interface{}, r *http.Request) (LoadContext, error) {
	switch r.Header.Get(HeaderContentType) {
	case ContentTypeStructuredJSON:
		return Structured.FromRequest(data, r)
	case ContentTypeBinaryJSON:
		return Binary.FromRequest(data, r)
	default:
		// TODO: assume binary content mode
		// (https://github.com/cloudevents/spec/blob/v0.1/http-transport-binding.md#3-http-message-mapping)
		// and that data is ??? (io.Reader?, byte array?)
		return nil, fmt.Errorf("Cannot handle encoding %q", r.Header.Get("Content-Type"))
	}
}

// NewRequest craetes an HTTP request for Structured content encoding.
func NewRequest(urlString string, data interface{}, context SendContext) (*http.Request, error) {
	return Structured.NewRequest(urlString, data, context)
}

// Opaque key type used to store V01EventContexts in a context.Context
type contextKeyType struct{}

var contextKey = contextKeyType{}

// FromContext loads an V01EventContext from a normal context.Context
func FromContext(ctx context.Context) LoadContext {
	return ctx.Value(contextKey).(LoadContext)
}
