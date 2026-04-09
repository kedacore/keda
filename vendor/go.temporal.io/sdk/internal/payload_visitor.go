package internal

import (
	"context"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/proxy"
	"google.golang.org/protobuf/proto"
)

type PayloadVisitor interface {
	Visit(ctx *proxy.VisitPayloadsContext, payloads []*commonpb.Payload) ([]*commonpb.Payload, error)
}

// visitProtoPayloads runs visitor over all payloads in msg, skipping search
// attributes. If visitor is nil, msg is unchanged. An optional ContextHook may
// be supplied to override the context for specific message subtrees during
// traversal (see [proxy.VisitPayloadsOptions.ContextHook]).
func visitProtoPayloads(ctx context.Context, visitor PayloadVisitor, msg proto.Message) error {
	return visitProtoPayloadsWithContextHook(ctx, visitor, msg, nil)
}

func visitProtoPayloadsWithContextHook(ctx context.Context, visitor PayloadVisitor, msg proto.Message, hook func(context.Context, proto.Message) (context.Context, error)) error {
	if visitor == nil {
		return nil
	}
	opts := proxy.VisitPayloadsOptions{
		Visitor:              visitor.Visit,
		SkipSearchAttributes: true,
		ContextHook:          hook,
	}
	return proxy.VisitPayloads(ctx, msg, opts)
}

// visitPayload runs visitor over a single payload. If visitor is nil
// the original payload is returned unchanged.
func visitPayload(ctx context.Context, visitor PayloadVisitor, p *commonpb.Payload) (*commonpb.Payload, error) {
	if visitor == nil {
		return p, nil
	}
	vpc := &proxy.VisitPayloadsContext{Context: ctx}
	visited, err := visitor.Visit(vpc, []*commonpb.Payload{p})
	if err != nil {
		return nil, err
	}
	if len(visited) == 0 {
		return nil, nil
	}
	return visited[0], nil
}
