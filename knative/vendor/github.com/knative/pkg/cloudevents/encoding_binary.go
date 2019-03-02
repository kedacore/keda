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

// TODO(inlined): must add header encoding/decoding

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

const (
	// HeaderCloudEventsVersion is the header for the version of Cloud Events
	// used.
	HeaderCloudEventsVersion = "CE-CloudEventsVersion"

	// HeaderEventID is the header for the unique ID of this event.
	HeaderEventID = "CE-EventID"

	// HeaderEventTime is the OPTIONAL header for the time at which an event
	// occurred.
	HeaderEventTime = "CE-EventTime"

	// HeaderEventType is the header for type of event represented. Value SHOULD
	// be in reverse-dns form.
	HeaderEventType = "CE-EventType"

	// HeaderEventTypeVersion is the OPTIONAL header for the version of the
	// scheme for the event type.
	HeaderEventTypeVersion = "CE-EventTypeVersion"

	// HeaderSchemaURL is the OPTIONAL header for the schema of the event data.
	HeaderSchemaURL = "CE-SchemaURL"

	// HeaderSource is the header for the source which emitted this event.
	HeaderSource = "CE-Source"

	// HeaderExtensionsPrefix is the OPTIONAL header prefix for CloudEvents extensions
	HeaderExtensionsPrefix = "CE-X-"

	// Binary implements Binary encoding/decoding
	Binary binary = 0
)

type binary int

// BinarySender implements an interface for sending an EventContext as
// (possibly one of several versions) as a binary encoding HTTP request.
type BinarySender interface {
	// AsHeaders converts this EventContext to a set of HTTP headers.
	AsHeaders() (http.Header, error)
}

// BinaryLoader implements an interface for translating a binary encoding HTTP
// request or response to a an EventContext (possibly one of several versions).
type BinaryLoader interface {
	// FromHeaders copies data from the supplied HTTP headers into the object.
	// Values will be defaulted if necessary.
	FromHeaders(in http.Header) error
}

// FromRequest parses event data and context from an HTTP request.
func (binary) FromRequest(data interface{}, r *http.Request) (LoadContext, error) {
	var ec LoadContext
	switch {
	case r.Header.Get("CE-SpecVersion") == V02CloudEventsVersion:
		ec = &V02EventContext{}
	case r.Header.Get("CE-CloudEventsVersion") == V01CloudEventsVersion:
		ec = &V01EventContext{}
	default:
		return nil, fmt.Errorf("Could not determine Cloud Events version from header: %+v", r.Header)
	}

	if err := ec.FromHeaders(r.Header); err != nil {
		return nil, err
	}

	if err := unmarshalEventData(ec.DataContentType(), r.Body, data); err != nil {
		return nil, err
	}

	return ec, nil
}

// NewRequest creates an HTTP request for Binary content encoding.
func (t binary) NewRequest(urlString string, data interface{}, context SendContext) (*http.Request, error) {
	url, err := url.Parse(urlString)
	if err != nil {
		return nil, err
	}

	h, err := context.AsHeaders()
	if err != nil {
		return nil, err
	}

	b, err := marshalEventData(h.Get("Content-Type"), data)
	if err != nil {
		return nil, err
	}

	return &http.Request{
		Method: http.MethodPost,
		URL:    url,
		Header: h,
		Body:   ioutil.NopCloser(bytes.NewReader(b)),
	}, nil
}
