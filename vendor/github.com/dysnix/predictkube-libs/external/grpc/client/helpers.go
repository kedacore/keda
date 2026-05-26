package client

import (
	"context"
	"sync"

	_ "github.com/dysnix/predictkube-libs/external/grpc/zstd_compressor"
	_ "google.golang.org/grpc/encoding/gzip"

	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"

	"github.com/dysnix/predictkube-libs/external/configs"
	"github.com/dysnix/predictkube-libs/external/enums"
	"github.com/dysnix/predictkube-libs/external/grpc/zstd_compressor"
)

var (
	clientMetricsOnce sync.Once
	clientMetrics     *grpcprom.ClientMetrics
)

// getClientMetrics returns a process-wide *grpcprom.ClientMetrics registered
// on prometheus.DefaultRegisterer so all gRPC clients in the process share one
// set of collectors and the existing /metrics endpoint exposes them.
func getClientMetrics() *grpcprom.ClientMetrics {
	clientMetricsOnce.Do(func() {
		clientMetrics = grpcprom.NewClientMetrics(grpcprom.WithClientHandlingTimeHistogram())
		prometheus.DefaultRegisterer.MustRegister(clientMetrics)
	})
	return clientMetrics
}

const (
	DefaultMaxMsgSize = 2 << 20 // 2Mb
)

func SetGrpcClientOptions(conf *configs.GRPC, baseConf *configs.Base, internalInterceptors ...grpc.UnaryClientInterceptor) (options []grpc.DialOption, err error) {
	unaryClientInterceptors := make([]grpc.UnaryClientInterceptor, 0)
	streamClientInterceptors := make([]grpc.StreamClientInterceptor, 0)

	if conf.Keepalive != nil {
		options = append(options,
			grpc.WithKeepaliveParams(
				keepalive.ClientParameters{
					Time:                conf.Keepalive.Time,
					Timeout:             conf.Keepalive.Timeout,
					PermitWithoutStream: conf.Keepalive.EnforcementPolicy.PermitWithoutStream,
				},
			))
	}

	switch conf.Compression.Type {
	case enums.Gzip:
		options = append(options, grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name)))
	case enums.Zstd:
		options = append(options, grpc.WithDefaultCallOptions(grpc.UseCompressor(zstd_compressor.Name)))
	}

	if conf.Conn.Timeout > 0 {
		options = append(options, grpc.WithConnectParams(
			grpc.ConnectParams{
				MinConnectTimeout: conf.Conn.Timeout,
			},
		))
	}

	if conf.Conn.ReadBufferSize > 0 {
		options = append(options, grpc.WithReadBufferSize(int(conf.Conn.ReadBufferSize)))
	}

	if conf.Conn.WriteBufferSize > 0 {
		options = append(options, grpc.WithWriteBufferSize(int(conf.Conn.WriteBufferSize)))
	}

	if conf.Conn.MaxMessageSize > 0 {
		options = append(options, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(int(conf.Conn.MaxMessageSize))))
		options = append(options, grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(int(conf.Conn.MaxMessageSize))))
	} else {
		options = append(options, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(DefaultMaxMsgSize)))
		options = append(options, grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(DefaultMaxMsgSize)))
	}

	if conf.Conn.Insecure {
		options = append(options, grpc.WithInsecure())
	}

	// TODO: implement all needed interceptors...

	unaryClientInterceptors = append(unaryClientInterceptors,
		PanicClientInterceptor(func(ctx context.Context, err error, params ...interface{}) error {
			//TODO:? can be any other logic...
			return status.Errorf(codes.Unknown, "panic triggered: %v", err)
		}))

	if baseConf.Monitoring.Enabled {
		m := getClientMetrics()
		unaryClientInterceptors = append(unaryClientInterceptors, m.UnaryClientInterceptor())
		streamClientInterceptors = append(streamClientInterceptors, m.StreamClientInterceptor())
	}

	unaryClientInterceptors = append(unaryClientInterceptors, internalInterceptors...)

	options = append(options,
		grpc.WithChainUnaryInterceptor(unaryClientInterceptors...),
		grpc.WithChainStreamInterceptor(streamClientInterceptors...),
	)

	return options, err
}
