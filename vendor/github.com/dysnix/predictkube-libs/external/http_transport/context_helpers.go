package http_transport

import (
	"context"

	"github.com/valyala/fasthttp"
)

type keyCtx string

func AddToContext(ctx context.Context, key string, data interface{}) context.Context {
	if req, ok := ctx.(*fasthttp.RequestCtx); ok {
		req.SetUserValue(key, data)
		return req
	}

	return context.WithValue(ctx, keyCtx(key), data)
}

func GetFromContext(ctx context.Context, key string) interface{} {
	if req, ok := ctx.(*fasthttp.RequestCtx); ok {
		return req.UserValue(key)
	}

	return ctx.Value(keyCtx(key))
}
