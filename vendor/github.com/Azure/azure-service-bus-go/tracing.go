package servicebus

//	MIT License
//
//	Copyright (c) Microsoft Corporation. All rights reserved.
//
//	Permission is hereby granted, free of charge, to any person obtaining a copy
//	of this software and associated documentation files (the "Software"), to deal
//	in the Software without restriction, including without limitation the rights
//	to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
//	copies of the Software, and to permit persons to whom the Software is
//	furnished to do so, subject to the following conditions:
//
//	The above copyright notice and this permission notice shall be included in all
//	copies or substantial portions of the Software.
//
//	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
//	IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
//	FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
//	AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
//	LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
//	OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
//	SOFTWARE

import (
	"context"
	"net/http"
	"os"

	"github.com/opentracing/opentracing-go"
	tag "github.com/opentracing/opentracing-go/ext"
)

func (ns *Namespace) startSpanFromContext(ctx context.Context, operationName string, opts ...opentracing.StartSpanOption) (opentracing.Span, context.Context) {
	span, ctx := opentracing.StartSpanFromContext(ctx, operationName, opts...)
	applyComponentInfo(span)
	return span, ctx
}

func (m *Message) startSpanFromContext(ctx context.Context, operationName string, opts ...opentracing.StartSpanOption) (opentracing.Span, context.Context) {
	span, ctx := opentracing.StartSpanFromContext(ctx, operationName, opts...)
	applyComponentInfo(span)
	span.SetTag("amqp.message-id", m.ID)
	if m.SessionID != nil {
		span.SetTag("amqp.message-group-id", *m.SessionID)
	}
	if m.GroupSequence != nil {
		span.SetTag("amqp.message-group-sequence", *m.GroupSequence)
	}
	return span, ctx
}

func (em *entityManager) startSpanFromContext(ctx context.Context, operationName string, opts ...opentracing.StartSpanOption) (opentracing.Span, context.Context) {
	span, ctx := opentracing.StartSpanFromContext(ctx, operationName, opts...)
	applyComponentInfo(span)
	tag.SpanKindRPCClient.Set(span)
	return span, ctx
}

func applyRequestInfo(span opentracing.Span, req *http.Request) {
	tag.HTTPUrl.Set(span, req.URL.String())
	tag.HTTPMethod.Set(span, req.Method)
}

func applyResponseInfo(span opentracing.Span, res *http.Response) {
	if res != nil {
		tag.HTTPStatusCode.Set(span, uint16(res.StatusCode))
	}
}

func (e *entity) startSpanFromContext(ctx context.Context, operationName string, opts ...opentracing.StartSpanOption) (opentracing.Span, context.Context) {
	span, ctx := opentracing.StartSpanFromContext(ctx, operationName, opts...)
	applyComponentInfo(span)
	return span, ctx
}

func (s *Sender) startProducerSpanFromContext(ctx context.Context, operationName string, opts ...opentracing.StartSpanOption) (opentracing.Span, context.Context) {
	span, ctx := opentracing.StartSpanFromContext(ctx, operationName, opts...)
	applyComponentInfo(span)
	tag.SpanKindProducer.Set(span)
	tag.MessageBusDestination.Set(span, s.getFullIdentifier())
	return span, ctx
}

func (r *Receiver) startConsumerSpanFromContext(ctx context.Context, operationName string, opts ...opentracing.StartSpanOption) (opentracing.Span, context.Context) {
	span, ctx := startConsumerSpanFromContext(ctx, operationName, opts...)
	tag.MessageBusDestination.Set(span, r.entityPath)
	return span, ctx
}

func (r *Receiver) startConsumerSpanFromWire(ctx context.Context, operationName string, reference opentracing.SpanContext, opts ...opentracing.StartSpanOption) (opentracing.Span, context.Context) {
	opts = append(opts, opentracing.FollowsFrom(reference))
	span := opentracing.StartSpan(operationName, opts...)
	ctx = opentracing.ContextWithSpan(ctx, span)
	applyComponentInfo(span)
	tag.SpanKindConsumer.Set(span)
	tag.MessageBusDestination.Set(span, r.entityPath)
	return span, ctx
}

func startConsumerSpanFromContext(ctx context.Context, operationName string, opts ...opentracing.StartSpanOption) (opentracing.Span, context.Context) {
	span, ctx := opentracing.StartSpanFromContext(ctx, operationName, opts...)
	applyComponentInfo(span)
	tag.SpanKindConsumer.Set(span)
	return span, ctx
}

func applyComponentInfo(span opentracing.Span) {
	tag.Component.Set(span, "github.com/Azure/azure-service-bus-go")
	span.SetTag("version", Version)
	applyNetworkInfo(span)
}

func applyNetworkInfo(span opentracing.Span) {
	hostname, err := os.Hostname()
	if err == nil {
		tag.PeerHostname.Set(span, hostname)
	}
}
