// The MIT License
//
// Copyright (c) 2021 Temporal Technologies Inc.  All rights reserved.
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
	"fmt"

	commonpb "go.temporal.io/api/common/v1"
)

type headerKey struct{}

// Header provides Temporal header information from the context for reading or
// writing during specific interceptor calls. See documentation in the
// interceptor package for more details.
func Header(ctx context.Context) map[string]*commonpb.Payload {
	m, _ := ctx.Value(headerKey{}).(map[string]*commonpb.Payload)
	return m
}

func contextWithNewHeader(ctx context.Context) context.Context {
	return context.WithValue(ctx, headerKey{}, map[string]*commonpb.Payload{})
}

func contextWithoutHeader(ctx context.Context) context.Context {
	return context.WithValue(ctx, headerKey{}, nil)
}

func contextWithHeaderPropagated(
	ctx context.Context,
	header *commonpb.Header,
	ctxProps []ContextPropagator,
) (context.Context, error) {
	if header == nil {
		header = &commonpb.Header{}
	}
	if header.Fields == nil {
		header.Fields = map[string]*commonpb.Payload{}
	}
	reader := NewHeaderReader(header)
	for _, ctxProp := range ctxProps {
		var err error
		if ctx, err = ctxProp.Extract(ctx, reader); err != nil {
			return nil, fmt.Errorf("failed propagating header: %w", err)
		}
	}
	return context.WithValue(ctx, headerKey{}, header.Fields), nil
}

func headerPropagated(ctx context.Context, ctxProps []ContextPropagator) (*commonpb.Header, error) {
	header := &commonpb.Header{Fields: Header(ctx)}
	if header.Fields == nil {
		return nil, fmt.Errorf("context missing header")
	}
	writer := NewHeaderWriter(header)
	for _, ctxProp := range ctxProps {
		if err := ctxProp.Inject(ctx, writer); err != nil {
			return nil, fmt.Errorf("failed propagating header: %w", err)
		}
	}
	return header, nil
}

// WorkflowHeader provides Temporal header information from the workflow context
// for reading or writing during specific interceptor calls. See documentation
// in the interceptor package for more details.
func WorkflowHeader(ctx Context) map[string]*commonpb.Payload {
	m, _ := ctx.Value(headerKey{}).(map[string]*commonpb.Payload)
	return m
}

func workflowContextWithNewHeader(ctx Context) Context {
	return WithValue(ctx, headerKey{}, map[string]*commonpb.Payload{})
}

func workflowContextWithoutHeader(ctx Context) Context {
	return WithValue(ctx, headerKey{}, nil)
}

func workflowContextWithHeaderPropagated(
	ctx Context,
	header *commonpb.Header,
	ctxProps []ContextPropagator,
) (Context, error) {
	if header == nil {
		header = &commonpb.Header{}
	}
	if header.Fields == nil {
		header.Fields = map[string]*commonpb.Payload{}
	}
	reader := NewHeaderReader(header)
	for _, ctxProp := range ctxProps {
		var err error
		if ctx, err = ctxProp.ExtractToWorkflow(ctx, reader); err != nil {
			return nil, fmt.Errorf("failed propagating header: %w", err)
		}
	}
	return WithValue(ctx, headerKey{}, header.Fields), nil
}

func workflowHeaderPropagated(ctx Context, ctxProps []ContextPropagator) (*commonpb.Header, error) {
	header := &commonpb.Header{Fields: WorkflowHeader(ctx)}
	if header.Fields == nil {
		return nil, fmt.Errorf("context missing workflow header")
	}
	writer := NewHeaderWriter(header)
	for _, ctxProp := range ctxProps {
		if err := ctxProp.InjectFromWorkflow(ctx, writer); err != nil {
			return nil, fmt.Errorf("failed propagating header: %w", err)
		}
	}
	return header, nil
}
