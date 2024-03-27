// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

// Package messaging contains types used across messaging packages.
package messaging

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/internal/uuid"
)

// CloudEvent represents an event conforming to the CloudEvents 1.0 spec.
// See here for more details: https://github.com/cloudevents/spec/blob/v1.0.2/cloudevents/spec.md
type CloudEvent struct {
	//
	// REQUIRED fields
	//

	// ID identifies the event. Producers MUST ensure that source + id is unique for each distinct event. If a duplicate
	// event is re-sent (e.g. due to a network error) it MAY have the same id. Consumers MAY assume that Events with
	// identical source and id are duplicates.
	ID string

	// Source identifies the context in which an event happened.
	Source string

	// SpecVersion is the version of the CloudEvents specification which the event uses.
	SpecVersion string

	// Type contains a value describing the type of event related to the originating occurrence.
	Type string

	//
	// OPTIONAL fields
	//

	// Data is the payload for the event.
	// * []byte will be serialized and deserialized as []byte.
	// * Any other type will be serialized to a JSON object and deserialized into
	//   a []byte, containing the JSON text.
	//
	// To deserialize into your chosen type:
	//
	//   var yourData *YourType
	//   json.Unmarshal(cloudEvent.Data.([]byte), &yourData)
	//
	Data any

	// DataContentType is the content type of [Data] value (ex: "text/xml")
	DataContentType *string

	// DataSchema identifies the schema that Data adheres to.
	DataSchema *string

	// Extensions are attributes that are serialized as siblings to attributes like Data.
	Extensions map[string]any

	// Subject of the event, in the context of the event producer (identified by Source).
	Subject *string

	// Time represents the time this event occurred.
	Time *time.Time
}

// CloudEventOptions are options for the [NewCloudEvent] function.
type CloudEventOptions struct {
	// DataContentType is the content type of [Data] value (ex: "text/xml")
	DataContentType *string

	// DataSchema identifies the schema that Data adheres to.
	DataSchema *string

	// Extensions are attributes that are serialized as siblings to attributes like Data.
	Extensions map[string]any

	// Subject of the event, in the context of the event producer (identified by Source).
	Subject *string

	// Time represents the time this event occurred.
	// Defaults to time.Now().UTC()
	Time *time.Time
}

// NewCloudEvent creates a CloudEvent.
//   - source - Identifies the context in which an event happened. The combination of id and source must be unique
//     for each distinct event.
//   - eventType - Type of event related to the originating occurrence.
//   - data - data to be added to the event. Can be a []byte, or any JSON serializable type, or nil.
//   - options - additional fields that are not required.
func NewCloudEvent(source string, eventType string, data any, options *CloudEventOptions) (CloudEvent, error) {
	if source == "" {
		return CloudEvent{}, errors.New("source cannot be empty")
	}

	if eventType == "" {
		return CloudEvent{}, errors.New("eventType cannot be empty")
	}

	id, err := uuid.New()

	if err != nil {
		return CloudEvent{}, err
	}

	ce := CloudEvent{
		ID:          id.String(),
		Source:      source,
		SpecVersion: "1.0",
		Type:        eventType,

		// optional but probably always filled in.
		Data: data,
	}

	if options != nil {
		ce.DataContentType = options.DataContentType
		ce.DataSchema = options.DataSchema
		ce.Extensions = options.Extensions
		ce.Subject = options.Subject

		ce.Time = options.Time
	}

	if ce.Time == nil {
		ce.Time = to.Ptr(time.Now().UTC())
	}

	return ce, nil
}

// MarshalJSON implements the json.Marshaler interface for CloudEvent.
func (ce CloudEvent) MarshalJSON() ([]byte, error) {
	m := map[string]any{
		"id":          ce.ID,
		"source":      ce.Source,
		"specversion": ce.SpecVersion,
		"type":        ce.Type,
	}

	if ce.Data != nil {
		bytes, isBytes := ce.Data.([]byte)

		if isBytes {
			m["data_base64"] = base64.StdEncoding.EncodeToString(bytes)
		} else {
			m["data"] = ce.Data
		}
	}

	if ce.DataContentType != nil {
		m["datacontenttype"] = ce.DataContentType
	}

	if ce.DataSchema != nil {
		m["dataschema"] = ce.DataSchema
	}

	for k, v := range ce.Extensions {
		m[k] = v
	}

	if ce.Subject != nil {
		m["subject"] = ce.Subject
	}

	if ce.Time != nil {
		m["time"] = ce.Time.Format(time.RFC3339Nano)
	}

	return json.Marshal(m)
}

func getValue[T any](k string, rawV any, dest *T) error {
	v, ok := rawV.(T)

	if !ok {
		var t T
		return fmt.Errorf("field %q is a %T, but should be %T", k, rawV, t)
	}

	*dest = v
	return nil
}

// UnmarshalJSON implements the json.Unmarshaler interface for CloudEvent.
func (ce *CloudEvent) UnmarshalJSON(data []byte) error {
	var m map[string]json.RawMessage

	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	for k, raw := range m {
		if err := updateFieldFromValue(ce, k, raw); err != nil {
			return fmt.Errorf("failed to deserialize %q: %w", k, err)
		}
	}

	if ce.ID == "" {
		return errors.New("required field 'id' was not present, or was empty")
	}

	if ce.Source == "" {
		return errors.New("required field 'source' was not present, or was empty")
	}

	if ce.SpecVersion == "" {
		return errors.New("required field 'specversion' was not present, or was empty")
	}

	if ce.Type == "" {
		return errors.New("required field 'type' was not present, or was empty")
	}

	return nil
}

func updateFieldFromValue(ce *CloudEvent, k string, raw json.RawMessage) error {
	switch k {
	//
	// required attributes
	//
	case "id":
		return json.Unmarshal(raw, &ce.ID)
	case "source":
		return json.Unmarshal(raw, &ce.Source)
	case "specversion":
		return json.Unmarshal(raw, &ce.SpecVersion)
	case "type":
		return json.Unmarshal(raw, &ce.Type)
	//
	// optional attributes
	//
	case "data":
		// let the user deserialize so they can put it into their own native type.
		ce.Data = []byte(raw)
	case "datacontenttype":
		return json.Unmarshal(raw, &ce.DataContentType)
	case "dataschema":
		return json.Unmarshal(raw, &ce.DataSchema)
	case "data_base64":
		var base64Str string
		if err := json.Unmarshal(raw, &base64Str); err != nil {
			return err
		}

		bytes, err := base64.StdEncoding.DecodeString(base64Str)

		if err != nil {
			return err
		}

		ce.Data = bytes
	case "subject":
		return json.Unmarshal(raw, &ce.Subject)
	case "time":
		var timeStr string
		if err := json.Unmarshal(raw, &timeStr); err != nil {
			return err
		}

		tm, err := time.Parse(time.RFC3339Nano, timeStr)

		if err != nil {
			return err
		}

		ce.Time = &tm
	default:
		//  https: //github.com/cloudevents/spec/blob/v1.0.2/cloudevents/spec.md#extension-context-attributes
		if ce.Extensions == nil {
			ce.Extensions = map[string]any{}
		}

		var v any
		if err := json.Unmarshal(raw, &v); err != nil {
			return err
		}

		ce.Extensions[k] = v
	}

	return nil
}
