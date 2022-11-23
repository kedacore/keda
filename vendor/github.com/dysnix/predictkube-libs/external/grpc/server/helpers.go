package server

import (
	"context"
	"net"
	"strings"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	_ "google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"

	"github.com/dysnix/predictkube-libs/external/configs"
	_ "github.com/dysnix/predictkube-libs/external/grpc/zstd_compressor"
)

const (
	errClosing = "use of closed network connection"

	DefaultMaxMsgSize = 2 << 20 // 2Mb
)

func CheckNetErrClosing(err error) error {
	if err != nil {
		if e, ok := err.(net.Error); ok && strings.Contains(e.Error(), errClosing) {
			// This was a 'use of closed network connection'
			return nil
		}

		return err
	}

	return nil
}

func SetGrpcServerOptions(conf *configs.GRPC, baseConf *configs.Base, internalInterceptors ...grpc.UnaryServerInterceptor) (options []grpc.ServerOption, err error) {
	unaryInterceptors := make([]grpc.UnaryServerInterceptor, 0)
	streamInterceptors := make([]grpc.StreamServerInterceptor, 0)

	if conf.Conn.Timeout > 0 {
		options = append(options, grpc.ConnectionTimeout(conf.Conn.Timeout))
	}

	if conf.Keepalive != nil {
		options = append(options, grpc.KeepaliveParams(
			keepalive.ServerParameters{
				Time:    conf.Keepalive.Time,
				Timeout: conf.Keepalive.Timeout,
			},
		))

		if conf.Keepalive.EnforcementPolicy != nil {
			options = append(options, grpc.KeepaliveEnforcementPolicy(
				keepalive.EnforcementPolicy{
					MinTime:             conf.Keepalive.EnforcementPolicy.MinTime,
					PermitWithoutStream: conf.Keepalive.EnforcementPolicy.PermitWithoutStream,
				},
			))
		}
	}

	if conf.Conn.ReadBufferSize > 0 {
		options = append(options, grpc.ReadBufferSize(int(conf.Conn.ReadBufferSize)))
	}

	if conf.Conn.WriteBufferSize > 0 {
		options = append(options, grpc.WriteBufferSize(int(conf.Conn.WriteBufferSize)))
	}

	if conf.Conn.MaxMessageSize > 0 {
		options = append(options, grpc.MaxRecvMsgSize(int(conf.Conn.MaxMessageSize)))
		options = append(options, grpc.MaxSendMsgSize(int(conf.Conn.MaxMessageSize)))
	} else {
		options = append(options, grpc.MaxRecvMsgSize(DefaultMaxMsgSize))
		options = append(options, grpc.MaxSendMsgSize(DefaultMaxMsgSize))
	}

	unaryInterceptors = append(unaryInterceptors,
		PanicServerInterceptor(func(ctx context.Context, err error, params ...interface{}) error {
			//TODO:? can be any other logic...
			return status.Errorf(codes.Unknown, "panic triggered: %v", err)
		}),
	)

	// TODO: implement all needed interceptors...

	if baseConf.Monitoring.Enabled {
		unaryInterceptors = append(unaryInterceptors, grpc_prometheus.UnaryServerInterceptor)
		streamInterceptors = append(streamInterceptors, grpc_prometheus.StreamServerInterceptor)
	}

	if len(internalInterceptors) > 0 {
		unaryInterceptors = append(unaryInterceptors, internalInterceptors...)
	}

	options = append(options,
		grpc_middleware.WithUnaryServerChain(unaryInterceptors...),
		grpc_middleware.WithStreamServerChain(streamInterceptors...),
	)

	return options, nil
}
