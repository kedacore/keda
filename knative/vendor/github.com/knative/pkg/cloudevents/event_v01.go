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
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	// V01CloudEventsVersion is the version of the CloudEvents spec targeted
	// by this library.
	V01CloudEventsVersion = "0.1"

	// v0.1 field names
	fieldCloudEventsVersion = "CloudEventsVersion"
	fieldEventID            = "EventID"
	fieldEventType          = "EventType"
)

// V01EventContext holds standard metadata about an event. See
// https://github.com/cloudevents/spec/blob/v0.1/spec.md#context-attributes for
// details on these fields.
type V01EventContext struct {
	// The version of the CloudEvents specification used by the event.
	CloudEventsVersion string `json:"cloudEventsVersion,omitempty"`
	// ID of the event; must be non-empty and unique within the scope of the producer.
	EventID string `json:"eventID"`
	// Timestamp when the event happened.
	EventTime time.Time `json:"eventTime,omitempty"`
	// Type of occurrence which has happened.
	EventType string `json:"eventType"`
	// The version of the `eventType`; this is producer-specific.
	EventTypeVersion string `json:"eventTypeVersion,omitempty"`
	// A link to the schema that the `data` attribute adheres to.
	SchemaURL string `json:"schemaURL,omitempty"`
	// A MIME (RFC 2046) string describing the media type of `data`.
	// TODO: Should an empty string assume `application/json`, or auto-detect the content?
	ContentType string `json:"contentType,omitempty"`
	// A URI describing the event producer.
	Source string `json:"source"`
	// Additional metadata without a well-defined structure.
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// AsV01 implements the ContextTranslator interface.
func (ec V01EventContext) AsV01() V01EventContext {
	return ec
}

// AsV02 implements the ContextTranslator interface.
func (ec V01EventContext) AsV02() V02EventContext {
	ret := V02EventContext{
		SpecVersion: V02CloudEventsVersion,
		Type:        ec.EventType,
		Source:      ec.Source,
		ID:          ec.EventID,
		Time:        ec.EventTime,
		SchemaURL:   ec.SchemaURL,
		ContentType: ec.ContentType,
		Extensions:  make(map[string]interface{}),
	}
	// eventTypeVersion was retired in v0.2, so put it in an extension.
	if ec.EventTypeVersion != "" {
		ret.Extensions["eventtypeversion"] = ec.EventTypeVersion
	}
	for k, v := range ec.Extensions {
		ret.Extensions[k] = v
	}
	return ret
}

// AsHeaders implements the BinarySender interface.
func (ec V01EventContext) AsHeaders() (http.Header, error) {
	h := http.Header{}
	h.Set("CE-CloudEventsVersion", ec.CloudEventsVersion)
	h.Set("CE-EventID", ec.EventID)
	h.Set("CE-EventType", ec.EventType)
	h.Set("CE-Source", ec.Source)
	if ec.CloudEventsVersion == "" {
		h.Set("CE-CloudEventsVersion", V01CloudEventsVersion)
	}
	if !ec.EventTime.IsZero() {
		h.Set("CE-EventTime", ec.EventTime.Format(time.RFC3339Nano))
	}
	if ec.EventTypeVersion != "" {
		h.Set("CE-EventTypeVersion", ec.EventTypeVersion)
	}
	if ec.SchemaURL != "" {
		h.Set("CE-SchemaUrl", ec.SchemaURL)
	}
	if ec.ContentType != "" {
		h.Set("Content-Type", ec.ContentType)
	}
	for k, v := range ec.Extensions {
		encoded, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		// Preserve case in v0.1, even though HTTP headers are case-insensitive.
		h["CE-X-"+k] = []string{string(encoded)}
	}
	return h, nil
}

// FromHeaders implements the BinaryLoader interface.
func (ec *V01EventContext) FromHeaders(in http.Header) error {
	missingField := func(name string) error {
		if in.Get("CE-"+name) == "" {
			return fmt.Errorf("Missing field %q in %v: %q", "CE-"+name, in, in.Get("CE-"+name))
		}
		return nil
	}
	if err := anyError(
		missingField("CloudEventsVersion"),
		missingField("EventID"),
		missingField("EventType"),
		missingField("Source")); err != nil {
		return err
	}
	data := V01EventContext{
		CloudEventsVersion: in.Get("CE-CloudEventsVersion"),
		EventID:            in.Get("CE-EventID"),
		EventType:          in.Get("CE-EventType"),
		EventTypeVersion:   in.Get("CE-EventTypeVersion"),
		SchemaURL:          in.Get("CE-SchemaURL"),
		ContentType:        in.Get("Content-Type"),
		Source:             in.Get("CE-Source"),
		Extensions:         make(map[string]interface{}),
	}
	if timeStr := in.Get("CE-EventTime"); timeStr != "" {
		var err error
		if data.EventTime, err = time.Parse(time.RFC3339Nano, timeStr); err != nil {
			return err
		}
	}
	for k, v := range in {
		if strings.EqualFold(k[:len("CE-X-")], "CE-X-") {
			key := k[len("CE-X-"):]
			var tmp interface{}
			if err := json.Unmarshal([]byte(v[0]), &tmp); err == nil {
				data.Extensions[key] = tmp
			} else {
				// If we can't unmarshal the data, treat it as a string.
				data.Extensions[key] = v[0]
			}
		}
	}
	*ec = data
	return nil
}

// AsJSON implements the StructuredSender interface.
func (ec V01EventContext) AsJSON() (map[string]json.RawMessage, error) {
	ret := make(map[string]json.RawMessage)
	err := anyError(
		encodeKey(ret, "cloudEventsVersion", ec.CloudEventsVersion),
		encodeKey(ret, "eventID", ec.EventID),
		encodeKey(ret, "eventTime", ec.EventTime),
		encodeKey(ret, "eventType", ec.EventType),
		encodeKey(ret, "eventTypeVersion", ec.EventTypeVersion),
		encodeKey(ret, "schemaURL", ec.SchemaURL),
		encodeKey(ret, "contentType", ec.ContentType),
		encodeKey(ret, "source", ec.Source),
		encodeKey(ret, "extensions", ec.Extensions))
	return ret, err
}

// DataContentType implements the StructuredSender interface.
func (ec V01EventContext) DataContentType() string {
	return ec.ContentType
}

// FromJSON implements the StructuredLoader interface.
func (ec *V01EventContext) FromJSON(in map[string]json.RawMessage) error {
	data := V01EventContext{
		CloudEventsVersion: extractKey(in, "cloudEventsVersion"),
		EventID:            extractKey(in, "eventID"),
		EventType:          extractKey(in, "eventType"),
		Source:             extractKey(in, "source"),
	}
	var err error
	if timeStr := extractKey(in, "eventTime"); timeStr != "" {
		if data.EventTime, err = time.Parse(time.RFC3339Nano, timeStr); err != nil {
			return err
		}
	}
	extractKeyTo(in, "eventTypeVersion", &data.EventTypeVersion)
	extractKeyTo(in, "schemaURL", &data.SchemaURL)
	extractKeyTo(in, "contentType", &data.ContentType)
	if len(in["extensions"]) == 0 {
		in["extensions"] = []byte("{}")
	}
	if err = json.Unmarshal(in["extensions"], &data.Extensions); err != nil {
		return err
	}
	*ec = data
	return nil
}

func encodeKey(out map[string]json.RawMessage, key string, value interface{}) (err error) {
	if s, ok := value.(string); ok && s == "" {
		// Skip empty strings.
		return nil
	}
	out[key], err = json.Marshal(value)
	return
}

func extractKey(in map[string]json.RawMessage, key string) (s string) {
	extractKeyTo(in, key, &s)
	return
}

func extractKeyTo(in map[string]json.RawMessage, key string, out *string) error {
	tmp := in[key]
	delete(in, key)
	return json.Unmarshal(tmp, out)
}
