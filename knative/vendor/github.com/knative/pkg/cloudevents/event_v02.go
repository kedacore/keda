/*
Copyright 2019 The Knative Authors

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
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	// V02CloudEventsVersion is the version of the CloudEvents spec targeted
	// by this library.
	V02CloudEventsVersion = "0.2"

	// required attributes
	fieldSpecVersion  = "specversion"
	fieldID           = "id"
	fieldType         = "type"
	fieldSource       = "source"
	fieldTime         = "time"
	fieldSchemaURL    = "schemaurl"
	fieldContentType  = "contenttype"
	headerContentType = "Content-Type"
)

// V02EventContext represents the non-data attributes of a CloudEvents v0.2
// event.
type V02EventContext struct {
	// The version of the CloudEvents specification used by the event.
	SpecVersion string `json:"specversion"`
	// The type of the occurrence which has happened.
	Type string `json:"type"`
	// A URI describing the event producer.
	Source string `json:"source"`
	// ID of the event; must be non-empty and unique within the scope of the producer.
	ID string `json:"id"`
	// Timestamp when the event happened.
	Time time.Time `json:"time,omitempty"`
	// A link to the schema that the `data` attribute adheres to.
	SchemaURL string `json:"schemaurl,omitempty"`
	// A MIME (RFC2046) string describing the media type of `data`.
	// TODO: Should an empty string assume `application/json`, `application/octet-stream`, or auto-detect the content?
	ContentType string `json:"contenttype,omitempty"`
	// Additional extension metadata beyond the base spec.
	Extensions map[string]interface{} `json:"-,omitempty"`
}

// AsV01 implements the ContextTranslator interface.
func (ec V02EventContext) AsV01() V01EventContext {
	ret := V01EventContext{
		CloudEventsVersion: V01CloudEventsVersion,
		EventID:            ec.ID,
		EventTime:          ec.Time,
		EventType:          ec.Type,
		SchemaURL:          ec.SchemaURL,
		ContentType:        ec.ContentType,
		Source:             ec.Source,
		Extensions:         make(map[string]interface{}),
	}
	for k, v := range ec.Extensions {
		// eventTypeVersion was retired in v0.2
		if strings.EqualFold(k, "eventTypeVersion") {
			etv, ok := v.(string)
			if ok {
				ret.EventTypeVersion = etv
			}
			continue
		}
		ret.Extensions[k] = v
	}
	return ret
}

// AsV02 implements the ContextTranslator interface.
func (ec V02EventContext) AsV02() V02EventContext {
	return ec
}

// AsHeaders implements the BinarySender interface.
func (ec V02EventContext) AsHeaders() (http.Header, error) {
	h := http.Header{}
	h.Set("CE-"+fieldSpecVersion, ec.SpecVersion)
	h.Set("CE-"+fieldType, ec.Type)
	h.Set("CE-"+fieldSource, ec.Source)
	h.Set("CE-"+fieldID, ec.ID)
	if ec.SpecVersion == "" {
		h.Set("CE-"+fieldSpecVersion, V02CloudEventsVersion)
	}
	if !ec.Time.IsZero() {
		h.Set("CE-"+fieldTime, ec.Time.Format(time.RFC3339Nano))
	}
	if ec.SchemaURL != "" {
		h.Set("CE-"+fieldSchemaURL, ec.SchemaURL)
	}
	if ec.ContentType != "" {
		h.Set(headerContentType, ec.ContentType)
	}
	for k, v := range ec.Extensions {
		// Per spec, map-valued extensions are converted to a list of headers as:
		// CE-attrib-key
		if mapVal, ok := v.(map[string]interface{}); ok {
			for subkey, subval := range mapVal {
				encoded, err := json.Marshal(subval)
				if err != nil {
					return nil, err
				}
				h.Set("CE-"+k+"-"+subkey, string(encoded))
			}
			continue
		}
		encoded, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		h.Set("CE-"+k, string(encoded))
	}

	return h, nil
}

// FromHeaders implements the BinaryLoader interface.
func (ec *V02EventContext) FromHeaders(in http.Header) error {
	missingField := func(name string) error {
		if in.Get("CE-"+name) == "" {
			return fmt.Errorf("Missing field %q in %v: %q", "CE-"+name, in, in.Get("CE-"+name))
		}
		return nil
	}
	err := anyError(
		missingField(fieldSpecVersion),
		missingField(fieldID),
		missingField(fieldType),
		missingField(fieldSource),
	)
	if err != nil {
		return err
	}
	data := V02EventContext{
		ContentType: in.Get(headerContentType),
		Extensions:  make(map[string]interface{}),
	}
	// Extensions and top-level fields are mixed under "CE-" headers.
	// Extract them all here rather than trying to clear fields in headers.
	for k, v := range in {
		if strings.EqualFold(k[:len("CE-")], "CE-") {
			key, value := strings.ToLower(string(k[len("CE-"):])), v[0]
			switch key {
			case fieldSpecVersion:
				data.SpecVersion = value
			case fieldType:
				data.Type = value
			case fieldSource:
				data.Source = value
			case fieldID:
				data.ID = value
			case fieldSchemaURL:
				data.SchemaURL = value
			case fieldTime:
				if data.Time, err = time.Parse(time.RFC3339Nano, value); err != nil {
					return err
				}
			default:
				var tmp interface{}
				if err = json.Unmarshal([]byte(value), &tmp); err != nil {
					tmp = value
				}
				// Per spec, map-valued extensions are converted to a list of headers as:
				// CE-attrib-key. This is where things get a bit crazy... see
				// https://github.com/cloudevents/spec/issues/367 for additional notes.
				if strings.Contains(key, "-") {
					items := strings.SplitN(key, "-", 2)
					key, subkey := items[0], items[1]
					if _, ok := data.Extensions[key]; !ok {
						data.Extensions[key] = make(map[string]interface{})
					}
					if submap, ok := data.Extensions[key].(map[string]interface{}); ok {
						submap[subkey] = tmp
					}
				} else {
					data.Extensions[key] = tmp
				}
			}
		}
	}
	*ec = data
	return nil
}

// AsJSON implementsn the StructuredSender interface.
func (ec V02EventContext) AsJSON() (map[string]json.RawMessage, error) {
	ret := make(map[string]json.RawMessage)
	err := anyError(
		encodeKey(ret, fieldSpecVersion, ec.SpecVersion),
		encodeKey(ret, fieldType, ec.Type),
		encodeKey(ret, fieldSource, ec.Source),
		encodeKey(ret, fieldID, ec.ID),
		encodeKey(ret, fieldTime, ec.Time),
		encodeKey(ret, fieldSchemaURL, ec.SchemaURL),
		encodeKey(ret, fieldContentType, ec.ContentType),
	)
	if err != nil {
		return nil, err
	}
	for k, v := range ec.Extensions {
		if err = encodeKey(ret, k, v); err != nil {
			return nil, err
		}
	}
	return ret, nil
}

// DataContentType implements the StructuredSender interface.
func (ec V02EventContext) DataContentType() string {
	return ec.ContentType
}

// FromJSON implements the StructuredLoader interface.
func (ec *V02EventContext) FromJSON(in map[string]json.RawMessage) error {
	data := V02EventContext{
		SpecVersion: extractKey(in, fieldSpecVersion),
		Type:        extractKey(in, fieldType),
		Source:      extractKey(in, fieldSource),
		ID:          extractKey(in, fieldID),
		Extensions:  make(map[string]interface{}),
	}
	var err error
	if timeStr := extractKey(in, fieldTime); timeStr != "" {
		if data.Time, err = time.Parse(time.RFC3339Nano, timeStr); err != nil {
			return err
		}
	}
	extractKeyTo(in, fieldSchemaURL, &data.SchemaURL)
	extractKeyTo(in, fieldContentType, &data.ContentType)
	// Extract the remaining items from in by converting to JSON and then
	// unpacking into Extensions. This avoids having to do funny type
	// checking/testing in the loop over values.
	extensionsJSON, err := json.Marshal(in)
	if err != nil {
		return err
	}
	err = json.Unmarshal(extensionsJSON, &data.Extensions)
	*ec = data
	return err
}
