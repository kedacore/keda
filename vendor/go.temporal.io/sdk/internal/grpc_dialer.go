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
	"sync/atomic"
	"time"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/internal/common/metrics"
	"go.temporal.io/sdk/internal/common/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type (
	// dialParameters are passed to GRPCDialer and must be used to create gRPC connection.
	dialParameters struct {
		HostPort              string
		UserConnectionOptions ConnectionOptions
		RequiredInterceptors  []grpc.UnaryClientInterceptor
		DefaultServiceConfig  string
	}
)

const (
	// LocalHostPort is a default host:port for worker and client to connect to.
	//
	// Exposed as: [go.temporal.io/sdk/client.DefaultHostPort]
	LocalHostPort = "localhost:7233"

	// defaultServiceConfig is a default gRPC connection service config which enables DNS round-robin between IPs.
	defaultServiceConfig = `{"loadBalancingConfig": [{"round_robin":{}}]}`

	// minConnectTimeout is the minimum amount of time we are willing to give a connection to complete.
	minConnectTimeout = 20 * time.Second

	// attemptSuffix is a suffix added to the metric name for individual call attempts made to the server, which includes retries.
	attemptSuffix = "_attempt"

	// mb is a number of bytes in a megabyte
	mb = 1024 * 1024

	// defaultMaxPayloadSize is a maximum size of the payload that grpc client would allow.
	defaultMaxPayloadSize = 128 * mb

	// defaultKeepAliveTime is the keep alive time if one is not specified.
	defaultKeepAliveTime = 30 * time.Second

	// defaultKeepAliveTimeout is the keep alive timeout if one is not specified.
	defaultKeepAliveTimeout = 15 * time.Second

	// temporalNamespaceHeaderKey is the header key that should contain the target namespace of the request.
	temporalNamespaceHeaderKey = "temporal-namespace"
)

func dial(params dialParameters) (*grpc.ClientConn, error) {
	var securityOptions []grpc.DialOption
	if params.UserConnectionOptions.TLS != nil {
		securityOptions = []grpc.DialOption{
			grpc.WithTransportCredentials(credentials.NewTLS(params.UserConnectionOptions.TLS)),
		}
	} else {
		securityOptions = []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithAuthority(params.UserConnectionOptions.Authority),
		}
	}

	maxPayloadSize := defaultMaxPayloadSize
	if params.UserConnectionOptions.MaxPayloadSize != 0 {
		maxPayloadSize = params.UserConnectionOptions.MaxPayloadSize
	}

	// gRPC maintains connection pool inside grpc.ClientConn.
	// This connection pool has auto reconnect feature.
	// If connection goes down, gRPC will try to reconnect using exponential backoff strategy:
	// https://github.com/grpc/grpc/blob/master/doc/connection-backoff.md.
	// Default MaxDelay is 120 seconds which is too high.
	// Setting it to retryPollOperationMaxInterval here will correlate with poll reconnect interval.
	var cp = grpc.ConnectParams{
		Backoff:           backoff.DefaultConfig,
		MinConnectTimeout: minConnectTimeout,
	}
	cp.Backoff.BaseDelay = retryPollOperationInitialInterval
	cp.Backoff.MaxDelay = retryPollOperationMaxInterval
	opts := []grpc.DialOption{
		grpc.WithChainUnaryInterceptor(params.RequiredInterceptors...),
		grpc.WithDefaultServiceConfig(params.DefaultServiceConfig),
		grpc.WithConnectParams(cp),
	}

	opts = append(opts, securityOptions...)
	opts = append(opts, grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(maxPayloadSize)))
	opts = append(opts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxPayloadSize)))

	if !params.UserConnectionOptions.DisableKeepAliveCheck {
		// gRPC utilizes keep alive mechanism to detect dead connections in case if server didn't close them
		// gracefully. Client would ping the server periodically and expect replies withing the specified timeout.
		// Learn more by reading https://github.com/grpc/grpc/blob/master/doc/keepalive.md
		keepAliveTime := params.UserConnectionOptions.KeepAliveTime
		if keepAliveTime == 0 {
			keepAliveTime = defaultKeepAliveTime
		}
		keepAliveTimeout := params.UserConnectionOptions.KeepAliveTimeout
		if keepAliveTimeout == 0 {
			keepAliveTimeout = defaultKeepAliveTimeout
		}
		var kap = keepalive.ClientParameters{
			Time:                keepAliveTime,
			Timeout:             keepAliveTimeout,
			PermitWithoutStream: !params.UserConnectionOptions.DisableKeepAlivePermitWithoutStream,
		}
		opts = append(opts, grpc.WithKeepaliveParams(kap))
	}

	// Append any user-supplied options
	opts = append(opts, params.UserConnectionOptions.DialOptions...)

	return grpc.NewClient(params.HostPort, opts...)
}

func requiredInterceptors(
	clientOptions *ClientOptions,
	excludeInternalFromRetry *atomic.Bool,
) []grpc.UnaryClientInterceptor {
	interceptors := []grpc.UnaryClientInterceptor{
		errorInterceptor,
		// Report aggregated metrics for the call, this is done outside of the retry loop.
		metrics.NewGRPCInterceptor(clientOptions.MetricsHandler, "", clientOptions.DisableErrorCodeMetricTags),
		// By default the grpc retry interceptor *is disabled*, preventing accidental use of retries.
		// We add call options for retry configuration based on the values present in the context.
		retry.NewRetryOptionsInterceptor(excludeInternalFromRetry),
		// Performs retries *IF* retry options are set for the call.
		grpc_retry.UnaryClientInterceptor(),
		// Report metrics for every call made to the server.
		metrics.NewGRPCInterceptor(clientOptions.MetricsHandler, attemptSuffix, clientOptions.DisableErrorCodeMetricTags),
	}
	if clientOptions.HeadersProvider != nil {
		interceptors = append(interceptors, headersProviderInterceptor(clientOptions.HeadersProvider))
	}
	if clientOptions.TrafficController != nil {
		interceptors = append(interceptors, trafficControllerInterceptor(clientOptions.TrafficController))
	}
	// Add credentials interceptor. This is intentionally added after headers
	// provider to overwrite anything set there.
	if clientOptions.Credentials != nil {
		if interceptor := clientOptions.Credentials.gRPCInterceptor(); interceptor != nil {
			interceptors = append(interceptors, interceptor)
		}
	}
	// Add namespace provider interceptor
	interceptors = append(interceptors, namespaceProviderInterceptor())
	return interceptors
}

func namespaceProviderInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if nsReq, ok := req.(interface{ GetNamespace() string }); ok {
			// Only add namespace if it doesn't already exist
			if md, _ := metadata.FromOutgoingContext(ctx); len(md.Get(temporalNamespaceHeaderKey)) == 0 {
				ctx = metadata.AppendToOutgoingContext(ctx, temporalNamespaceHeaderKey, nsReq.GetNamespace())
			}
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func trafficControllerInterceptor(controller TrafficController) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		err := controller.CheckCallAllowed(ctx, method, req, reply)
		// Break execution chain and return an error without sending actual request to the server.
		if err != nil {
			return err
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func headersProviderInterceptor(headersProvider HeadersProvider) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		headers, err := headersProvider.GetHeaders(ctx)
		if err != nil {
			return err
		}
		for k, v := range headers {
			ctx = metadata.AppendToOutgoingContext(ctx, k, v)
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func errorInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	err := invoker(ctx, method, req, reply, cc, opts...)
	err = serviceerror.FromStatus(status.Convert(err))
	return err
}
