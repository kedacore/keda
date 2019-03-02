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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

const (
	// Structured implements the JSON structured encoding/decoding
	Structured structured = 0
)

type structured int

// StructuredSender implements an interface for translating an EventContext
// (possibly one of severals versions) to a structured encoding HTTP request.
type StructuredSender interface {
	// AsJSON encodes the object into a map from string to JSON data, which
	// allows additional keys to be encoded later.
	AsJSON() (map[string]json.RawMessage, error)
}

// StructuredLoader implements an interface for translating a structured
// encoding HTTP request or response to a an EventContext (possibly one of
// several versions).
type StructuredLoader interface {
	// FromJSON assumes that the object has already been decoded into a raw map
	// from string to json.RawMessage, because this is needed to extract the
	// CloudEvents version.
	FromJSON(map[string]json.RawMessage) error
}

// FromRequest parses a CloudEvent from structured content encoding.
func (structured) FromRequest(data interface{}, r *http.Request) (LoadContext, error) {
	raw := make(map[string]json.RawMessage)
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return nil, err
	}

	rawData := raw["data"]
	delete(raw, "data")

	var ec LoadContext
	v := ""
	if err := json.Unmarshal(raw["specversion"], &v); err == nil && v == V02CloudEventsVersion {
		ec = &V02EventContext{}
	} else if err := json.Unmarshal(raw["cloudEventsVersion"], &v); err == nil && v == V01CloudEventsVersion {
		ec = &V01EventContext{}
	} else {
		return nil, fmt.Errorf("Could not determine Cloud Events version from payload: %q", data)
	}

	if err := ec.FromJSON(raw); err != nil {
		return nil, err
	}

	contentType := ec.DataContentType()
	if contentType == "" {
		contentType = contentTypeJSON
	}
	var reader io.Reader
	if !isJSONEncoding(contentType) {
		var jsonDecoded string
		if err := json.Unmarshal(rawData, &jsonDecoded); err != nil {
			return nil, fmt.Errorf("Could not JSON decode %q value %q", contentType, rawData)
		}
		reader = strings.NewReader(jsonDecoded)
	} else {
		reader = bytes.NewReader(rawData)
	}
	if err := unmarshalEventData(contentType, reader, data); err != nil {
		return nil, err
	}
	return ec, nil
}

// NewRequest creates an HTTP request for Structured content encoding.
func (structured) NewRequest(urlString string, data interface{}, context SendContext) (*http.Request, error) {
	url, err := url.Parse(urlString)
	if err != nil {
		return nil, err
	}

	fields, err := context.AsJSON()
	if err != nil {
		return nil, err
	}

	// TODO: remove this defaulting?
	contentType := context.DataContentType()
	if contentType == "" {
		contentType = contentTypeJSON
	}

	dataBytes, err := marshalEventData(contentType, data)
	if err != nil {
		return nil, err
	}
	if isJSONEncoding(contentType) {
		fields["data"] = json.RawMessage(dataBytes)
	} else {
		fields["data"], err = json.Marshal(string(dataBytes))
		if err != nil {
			return nil, err
		}
	}

	b, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}

	h := http.Header{}
	h.Set(HeaderContentType, ContentTypeStructuredJSON)
	return &http.Request{
		Method: http.MethodPost,
		URL:    url,
		Header: h,
		Body:   ioutil.NopCloser(bytes.NewReader(b)),
	}, nil
}
