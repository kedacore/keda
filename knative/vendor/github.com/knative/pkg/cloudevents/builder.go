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
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// CloudEventEncoding is used to tell the builder which encoding to select.
// the default is Binary.
type CloudEventEncoding int

const (
	// Binary v0.1
	BinaryV01 CloudEventEncoding = iota
	// Structured v0.1
	StructuredV01
)

// Builder holds settings that do not change over CloudEvents. It is intended
// to represent a builder of only a single CloudEvent type.
type Builder struct {
	// A URI describing the event producer.
	Source string
	// Type of occurrence which has happened.
	EventType string
	// The version of the `eventType`; this is producer-specific.
	EventTypeVersion string
	// A link to the schema that the `data` attribute adheres to.
	SchemaURL string
	// Additional metadata without a well-defined structure.
	Extensions map[string]interface{}

	// Encoding specifies the requested output encoding of the CloudEvent.
	Encoding CloudEventEncoding
}

// Build produces a http request with the constant data embedded in the builder
// merged with the new data provided in the build function. The request will
// send a pre-assembled cloud event to the given target. The target is assumed
// to be a URL with a scheme, ie: "http://localhost:8080"
func (b *Builder) Build(target string, data interface{}, overrides ...SendContext) (*http.Request, error) {
	if len(overrides) > 1 {
		return nil, fmt.Errorf("Build was called with more than one override")
	}

	var overridesV01 *V01EventContext
	if len(overrides) == 1 {
		switch t := overrides[0].(type) {
		case V01EventContext:
			o := overrides[0].(V01EventContext)
			overridesV01 = &o
		default:
			return nil, fmt.Errorf("Build was called with unknown override type %v", t)
		}
	}
	// TODO: when V02 is supported this will have to shuffle a little.
	ctx := b.cloudEventsContextV01(overridesV01)

	if ctx.Source == "" {
		return nil, fmt.Errorf("ctx.Source resolved empty")
	}
	if ctx.EventType == "" {
		return nil, fmt.Errorf("ctx.EventType resolved empty")
	}

	switch b.Encoding {
	case BinaryV01:
		return Binary.NewRequest(target, data, ctx)
	case StructuredV01:
		return Structured.NewRequest(target, data, ctx)
	default:
		return nil, fmt.Errorf("unsupported encoding: %v", b.Encoding)
	}
}

// cloudEventsContext creates a CloudEvent context object, assumes
// application/json as the content type.
func (b *Builder) cloudEventsContextV01(overrides *V01EventContext) V01EventContext {
	ctx := V01EventContext{
		CloudEventsVersion: CloudEventsVersion,
		EventType:          b.EventType,
		EventID:            uuid.New().String(),
		EventTypeVersion:   b.EventTypeVersion,
		SchemaURL:          b.SchemaURL,
		Source:             b.Source,
		ContentType:        "application/json",
		EventTime:          time.Now(),
		Extensions:         b.Extensions,
	}
	if overrides != nil {
		if overrides.Source != "" {
			ctx.Source = overrides.Source
		}
		if overrides.EventID != "" {
			ctx.EventID = overrides.EventID
		}
		if overrides.EventType != "" {
			ctx.EventType = overrides.EventType
		}
		if !overrides.EventTime.IsZero() {
			ctx.EventTime = overrides.EventTime
		}
		if overrides.ContentType != "" {
			ctx.ContentType = overrides.ContentType
		}
		if len(overrides.Extensions) > 0 {
			if ctx.Extensions == nil {
				ctx.Extensions = make(map[string]interface{})
			}
			for k, v := range overrides.Extensions {
				ctx.Extensions[k] = v
			}
		}
	}
	return ctx
}
