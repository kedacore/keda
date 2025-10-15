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
//
// Exposed as: [go.temporal.io/sdk/interceptor.Header]
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
//
// Exposed as: [go.temporal.io/sdk/interceptor.WorkflowHeader]
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
