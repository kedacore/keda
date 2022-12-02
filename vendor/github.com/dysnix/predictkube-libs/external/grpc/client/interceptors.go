package client

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/dysnix/predictkube-libs/external/configs"
	grpcC "github.com/dysnix/predictkube-libs/external/grpc"
)

func InjectClientMetadataInterceptor(conf configs.Client) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) (err error) {
		if len(conf.Name) > 0 {
			ctx = metadata.AppendToOutgoingContext(ctx, grpcC.NameKey, conf.Name)
		}

		if len(conf.ClusterID) > 0 {
			ctx = metadata.AppendToOutgoingContext(ctx, grpcC.ClusterIDKey, conf.ClusterID)
		}

		return invoker(metadata.AppendToOutgoingContext(ctx, grpcC.TokenKey, conf.Token), method, req, reply, cc, opts...)
	}
}

func InjectPublicClientMetadataInterceptor(apiKey string) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) (err error) {
		return invoker(metadata.AppendToOutgoingContext(ctx, grpcC.APIKey, apiKey), method, req, reply, cc, opts...)
	}
}

func PanicClientInterceptor(handler func(ctx context.Context, err error, params ...interface{}) error, params ...interface{}) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) (err error) {

		defer func() {
			if r := recover(); r != nil {
				switch errType := r.(type) {
				case error:
					err = handler(ctx, errType, params...)
				case string:
					err = handler(ctx, errors.New(errType), params...)
				}
			}
		}()

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
