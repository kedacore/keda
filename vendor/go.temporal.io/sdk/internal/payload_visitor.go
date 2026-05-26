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

type PayloadVisitorWithContextHook interface {
	PayloadVisitor
	ContextHook(ctx context.Context, msg proto.Message) (context.Context, error)
}

type compositePayloadVisitor struct {
	visitors []PayloadVisitor
}

var _ PayloadVisitor = (*compositePayloadVisitor)(nil)
var _ PayloadVisitorWithContextHook = (*compositePayloadVisitor)(nil)

func (v *compositePayloadVisitor) Visit(ctx *proxy.VisitPayloadsContext, payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	var err error
	for _, visitor := range v.visitors {
		payloads, err = visitor.Visit(ctx, payloads)
		if err != nil {
			return nil, err
		}
	}
	return payloads, err
}

func (v *compositePayloadVisitor) ContextHook(ctx context.Context, msg proto.Message) (context.Context, error) {
	var err error
	for _, visitor := range v.visitors {
		if hookVisitor, ok := visitor.(PayloadVisitorWithContextHook); ok {
			ctx, err = hookVisitor.ContextHook(ctx, msg)
			if err != nil {
				return nil, err
			}
		}
	}
	return ctx, nil
}

func newCompositePayloadVisitor(visitors ...PayloadVisitor) PayloadVisitor {
	return &compositePayloadVisitor{
		visitors: visitors,
	}
}

// visitProtoPayloads runs visitor over all payloads in msg, skipping search
// attributes. If visitor is nil, msg is unchanged.
func visitProtoPayloads(ctx context.Context, visitor PayloadVisitor, msg proto.Message, concurrencyLimit int) error {
	if visitor == nil {
		return nil
	}
	var hook func(context.Context, proto.Message) (context.Context, error)
	if visitorWithHook, ok := visitor.(PayloadVisitorWithContextHook); ok {
		hook = visitorWithHook.ContextHook
	}
	opts := proxy.VisitPayloadsOptions{
		Visitor:              visitor.Visit,
		SkipSearchAttributes: true,
		ContextHook:          hook,
		ConcurrencyLimit:     concurrencyLimit,
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
