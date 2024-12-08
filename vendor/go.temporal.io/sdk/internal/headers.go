// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package internal

import (
	"context"

	"go.temporal.io/sdk/converter"

	commonpb "go.temporal.io/api/common/v1"
)

// HeaderWriter is an interface to write information to temporal headers
type (
	HeaderWriter interface {
		Set(string, *commonpb.Payload)
	}

	// HeaderReader is an interface to read information from temporal headers
	HeaderReader interface {
		Get(string) (*commonpb.Payload, bool)
		ForEachKey(handler func(string, *commonpb.Payload) error) error
	}

	// ContextPropagator is an interface that determines what information from
	// context to pass along
	ContextPropagator interface {
		// Inject injects information from a Go Context into headers
		Inject(context.Context, HeaderWriter) error

		// Extract extracts context information from headers and returns a context
		// object
		Extract(context.Context, HeaderReader) (context.Context, error)

		// InjectFromWorkflow injects information from workflow context into headers
		InjectFromWorkflow(Context, HeaderWriter) error

		// ExtractToWorkflow extracts context information from headers and returns
		// a workflow context
		ExtractToWorkflow(Context, HeaderReader) (Context, error)
	}

	// ContextAware is an optional interface that can be implemented alongside
	// DataConverter. This interface allows Temporal to pass Workflow/Activity
	// contexts to the DataConverter so that it may tailor it's behaviour.
	//
	// Note that data converters may be called in non-context-aware situations to
	// convert payloads that may not be customized per context. Data converter
	// implementers should not expect or require contextual data be present.
	ContextAware interface {
		WithWorkflowContext(ctx Context) converter.DataConverter
		WithContext(ctx context.Context) converter.DataConverter
	}

	headerReader struct {
		header *commonpb.Header
	}
)

func (hr *headerReader) ForEachKey(handler func(string, *commonpb.Payload) error) error {
	if hr.header == nil {
		return nil
	}
	for key, value := range hr.header.Fields {
		if err := handler(key, value); err != nil {
			return err
		}
	}
	return nil
}

func (hr *headerReader) Get(key string) (*commonpb.Payload, bool) {
	if hr.header == nil {
		return nil, false
	}
	payload, ok := hr.header.Fields[key]
	return payload, ok
}

// NewHeaderReader returns a header reader interface
func NewHeaderReader(header *commonpb.Header) HeaderReader {
	return &headerReader{header: header}
}

type headerWriter struct {
	header *commonpb.Header
}

func (hw *headerWriter) Set(key string, value *commonpb.Payload) {
	if hw.header == nil {
		return
	}
	hw.header.Fields[key] = value
}

// NewHeaderWriter returns a header writer interface
func NewHeaderWriter(header *commonpb.Header) HeaderWriter {
	if header != nil && header.Fields == nil {
		header.Fields = make(map[string]*commonpb.Payload)
	}
	return &headerWriter{header: header}
}

// WithWorkflowContext returns a new DataConverter tailored to the passed Workflow context if
// the DataConverter implements the ContextAware interface. Otherwise the DataConverter is returned
// as-is.
func WithWorkflowContext(ctx Context, dc converter.DataConverter) converter.DataConverter {
	if d, ok := dc.(ContextAware); ok {
		return d.WithWorkflowContext(ctx)
	}
	return dc
}

// WithContext returns a new DataConverter tailored to the passed Workflow/Activity context if
// the DataConverter implements the ContextAware interface. Otherwise the DataConverter is returned
// as-is. This is generally used for Activity context but can be context for a Workflow if we're
// not yet executing the workflow so do not have a workflow.Context.
func WithContext(ctx context.Context, dc converter.DataConverter) converter.DataConverter {
	if d, ok := dc.(ContextAware); ok {
		return d.WithContext(ctx)
	}
	return dc
}
