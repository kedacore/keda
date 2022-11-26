package configs

import (
	"crypto/tls"

	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

type SignalStopper interface {
	Stop()
}

type SignalStopperWithErr interface {
	Stop() error
}

type SignalCloser interface {
	Close()
}

type SignalCloserWithErr interface {
	Close() error
}

type SingleUseGetter interface {
	SingleEnabled() bool
}

type ServerGetter interface {
	GetServerConfigs() *Single
}

type CacheGetter interface {
	GetCache() *Cache
}

type CacheConfigSetter interface {
	SetCache(configs CacheGetter)
}

type CacheSetters interface {
	CacheConfigSetter
	LoggerSetter
}

type CacheOption func(CacheSetters) error

func SetCache(conf CacheGetter) CacheOption {
	return func(r CacheSetters) error {
		if conf != nil {
			r.SetCache(conf)
		}
		return nil
	}
}

func SetCacheLogger(logger *zap.SugaredLogger) CacheOption {
	return func(r CacheSetters) error {
		if logger != nil {
			r.SetLogger(logger)
		}
		return nil
	}
}

type SingleGetter interface {
	GetBase() *Base
	GetGrpc() *GRPC
	GetClient() *Client
}

type ConfigSetter interface {
	SetConfigs(configs SingleGetter)
}

type LoggerSetter interface {
	SetLogger(logger *zap.SugaredLogger)
}

type GrpcClientConnSetter interface {
	SetGrpcClientConn(conn *grpc.ClientConn)
}

type ServerConfigSetter interface {
	SetConfigs(configs ServerGetter)
}

type Route struct {
	Method string
	fasthttp.RequestHandler
}

type RoutesSetter interface {
	SetRoutes(map[string]map[string]*Route)
}

type MiddlewaresSetter interface {
	SetMiddlewares([]func(fasthttp.RequestHandler) fasthttp.RequestHandler)
}

type ServerSetters interface {
	ServerConfigSetter
	LoggerSetter
	RoutesSetter
	MiddlewaresSetter
}

type ServerOption func(ServerSetters) error

func SetServerConfigs(conf ServerGetter) ServerOption {
	return func(r ServerSetters) error {
		if conf != nil {
			r.SetConfigs(conf)
		}
		return nil
	}
}

func SetServerLogger(logger *zap.SugaredLogger) ServerOption {
	return func(r ServerSetters) error {
		if logger != nil {
			r.SetLogger(logger)
		}
		return nil
	}
}

func SetRoutes(routes map[string]map[string]*Route) ServerOption {
	return func(r ServerSetters) error {
		if routes != nil {
			r.SetRoutes(routes)
		}
		return nil
	}
}

func SetMiddlewares(middlewares []func(fasthttp.RequestHandler) fasthttp.RequestHandler) ServerOption {
	return func(r ServerSetters) error {
		r.SetMiddlewares(middlewares)
		return nil
	}
}

type MainSetters interface {
	ConfigSetter
	LoggerSetter
	GrpcClientConnSetter
}

type Option func(MainSetters) error

func SetConfigs(conf SingleGetter) Option {
	return func(r MainSetters) error {
		if conf != nil {
			r.SetConfigs(conf)
		}
		return nil
	}
}

func SetLogger(logger *zap.SugaredLogger) Option {
	return func(r MainSetters) error {
		if logger != nil {
			r.SetLogger(logger)
		}
		return nil
	}
}

func SetConn(conn *grpc.ClientConn) Option {
	return func(r MainSetters) error {
		if conn != nil && conn.GetState() == connectivity.Ready {
			r.SetGrpcClientConn(conn)
		}
		return nil
	}
}

type TransportGetter interface {
	GetTransportConfigs() *HTTPTransport
}

type TransportConfigSetter interface {
	SetConfigs(configs TransportGetter)
}

type TransportSetters interface {
	TransportConfigSetter
	TLSSetter
}

type TransportOption func(TransportSetters) error

type TLSSetter interface {
	SetTLS(*tls.Config)
}

func SetTransportConfigs(conf TransportGetter) TransportOption {
	return func(r TransportSetters) error {
		if conf != nil {
			r.SetConfigs(conf)
		}
		return nil
	}
}

func SetTLS(conf *tls.Config) TransportOption {
	return func(r TransportSetters) error {
		if conf != nil {
			r.SetTLS(conf)
		}
		return nil
	}
}

type HTTPClientGetter interface {
	GetHTTPClientConfigs() *HTTPClient
}

type HTTPClientConfigSetter interface {
	SetConfigs(configs HTTPClientGetter)
}

type HTTPClientSetters interface {
	HTTPClientConfigSetter
	TLSSetter
}

type HTTPClientOption func(HTTPClientSetters) error

func SetHTTPClientTLS(conf *tls.Config) HTTPClientOption {
	return func(r HTTPClientSetters) error {
		if conf != nil {
			r.SetTLS(conf)
		}
		return nil
	}
}
