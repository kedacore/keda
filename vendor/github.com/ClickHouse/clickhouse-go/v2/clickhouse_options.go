package clickhouse

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ClickHouse/ch-go/compress"

	"github.com/ClickHouse/clickhouse-go/v2/lib/churl"
)

type CompressionMethod byte

func (c CompressionMethod) String() string {
	switch c {
	case CompressionNone:
		return "none"
	case CompressionZSTD:
		return "zstd"
	case CompressionLZ4:
		return "lz4"
	case CompressionLZ4HC:
		return "lz4hc"
	case CompressionGZIP:
		return "gzip"
	case CompressionDeflate:
		return "deflate"
	case CompressionBrotli:
		return "br"
	default:
		return ""
	}
}

const (
	CompressionNone    = CompressionMethod(compress.None)
	CompressionLZ4     = CompressionMethod(compress.LZ4)
	CompressionLZ4HC   = CompressionMethod(compress.LZ4HC)
	CompressionZSTD    = CompressionMethod(compress.ZSTD)
	CompressionGZIP    = CompressionMethod(0x95)
	CompressionDeflate = CompressionMethod(0x96)
	CompressionBrotli  = CompressionMethod(0x97)
)

var compressionMap = map[string]CompressionMethod{
	"none":    CompressionNone,
	"zstd":    CompressionZSTD,
	"lz4":     CompressionLZ4,
	"lz4hc":   CompressionLZ4HC,
	"gzip":    CompressionGZIP,
	"deflate": CompressionDeflate,
	"br":      CompressionBrotli,
}

type Auth struct { // has_control_character
	Database string

	Username string
	Password string
}

type Compression struct {
	Method CompressionMethod
	// this only applies to lz4, lz4hc, zlib, and brotli compression algorithms
	Level int
}

type ConnOpenStrategy uint8

const (
	ConnOpenInOrder ConnOpenStrategy = iota
	ConnOpenRoundRobin
	ConnOpenRandom
)

type Protocol int

const (
	Native Protocol = iota
	HTTP
)

func (p Protocol) String() string {
	switch p {
	case Native:
		return "native"
	case HTTP:
		return "http"
	default:
		return ""
	}
}

func ParseDSN(dsn string) (*Options, error) {
	opt := &Options{}
	if err := opt.fromDSN(dsn); err != nil {
		return nil, err
	}
	return opt, nil
}

type Dial func(ctx context.Context, addr string, opt *Options) (DialResult, error)
type DialResult struct {
	conn nativeTransport
}

type HTTPProxy func(*http.Request) (*url.URL, error)

type Options struct {
	Protocol   Protocol
	ClientInfo ClientInfo

	TLS          *tls.Config
	Addr         []string
	Auth         Auth
	DialContext  func(ctx context.Context, addr string) (net.Conn, error)
	DialStrategy func(ctx context.Context, connID int, options *Options, dial Dial) (DialResult, error)

	// Deprecated: Use Logger instead. Debug enables legacy debug logging to stdout.
	// For structured logging with levels, use the Logger field.
	Debug bool

	// Deprecated: Use Logger instead. Debugf provides a custom debug logging function.
	// For structured logging with levels and custom handlers, use the Logger field with
	// a custom slog.Handler.
	Debugf func(format string, v ...any)

	// Logger provides structured logging using Go's standard log/slog package.
	// If nil, no logging occurs (default). To enable logging, provide a configured
	// slog.Logger:
	//
	//   logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
	//       Level: slog.LevelDebug,
	//   }))
	//   opts := &clickhouse.Options{
	//       Logger: logger,
	//   }
	//
	// For backward compatibility, if Debug=true and Debugf is set, those will be used
	// instead of Logger.
	Logger *slog.Logger

	Settings             Settings
	Compression          *Compression
	DialTimeout          time.Duration // default 30 second
	MaxOpenConns         int           // default MaxIdleConns + 5
	MaxIdleConns         int           // default 5
	ConnMaxLifetime      time.Duration // default 1 hour
	ConnOpenStrategy     ConnOpenStrategy
	FreeBufOnConnRelease bool              // drop preserved memory buffer after each query
	HttpHeaders          map[string]string // set additional headers on HTTP requests
	HttpUrlPath          string            // set additional URL path for HTTP requests
	HttpMaxConnsPerHost  int               // MaxConnsPerHost for http.Transport
	BlockBufferSize      uint8             // default 2 - can be overwritten on query
	MaxCompressionBuffer int               // default 10485760 - measured in bytes  i.e.

	// HTTPProxy specifies an HTTP proxy URL to use for requests made by the client.
	HTTPProxyURL *url.URL

	// GetJWT should return a JWT for authentication with ClickHouse Cloud.
	// This is called per connection/request, so you may cache the token in your app if needed.
	// Use this instead of Auth.Username and Auth.Password if you're using JWT auth.
	GetJWT GetJWTFunc

	scheme string

	// ReadTimeout is the maximum duration the client will wait for ClickHouse
	// to respond to a single Read call for bytes over the connection.
	// Can be overridden with context.WithDeadline.
	ReadTimeout time.Duration

	// Set a custom transport for the http client.
	// The default transport configured by the library is passed in as an argument.
	TransportFunc func(*http.Transport) (http.RoundTripper, error)
}

func (o *Options) fromDSN(in string) error {
	dsn, err := churl.Parse(in)
	if err != nil {
		return err
	}

	if dsn.Host == "" {
		return errors.New("parse dsn address failed")
	}

	if o.Settings == nil {
		o.Settings = make(Settings)
	}
	if dsn.User != nil {
		o.Auth.Username = dsn.User.Username()
		o.Auth.Password, _ = dsn.User.Password()
	}
	o.Addr = append(o.Addr, strings.Split(dsn.Host, ",")...)
	var (
		secure        bool
		params        = dsn.Query()
		skipVerify    bool
		tlsServerName string
	)
	o.Auth.Database = strings.TrimPrefix(dsn.Path, "/")

	for v := range params {
		switch v {
		case "debug":
			o.Debug, _ = strconv.ParseBool(params.Get(v))
		case "compress":
			if on, _ := strconv.ParseBool(params.Get(v)); on {
				if o.Compression == nil {
					o.Compression = &Compression{}
				}

				o.Compression.Method = CompressionLZ4
				continue
			}
			if compressMethod, ok := compressionMap[params.Get(v)]; ok {
				if o.Compression == nil {
					o.Compression = &Compression{
						// default for now same as Clickhouse - https://clickhouse.com/docs/en/operations/settings/settings#settings-http_zlib_compression_level
						Level: 3,
					}
				}

				o.Compression.Method = compressMethod
			}
		case "compress_level":
			level, err := strconv.ParseInt(params.Get(v), 10, 8)
			if err != nil {
				return fmt.Errorf("compress_level invalid value: %w", err)
			}

			if o.Compression == nil {
				o.Compression = &Compression{
					// a level alone doesn't enable compression
					Method: CompressionNone,
					Level:  int(level),
				}
				continue
			}

			o.Compression.Level = int(level)
		case "max_compression_buffer":
			max, err := strconv.Atoi(params.Get(v))
			if err != nil {
				return fmt.Errorf("max_compression_buffer invalid value: %w", err)
			}
			o.MaxCompressionBuffer = max
		case "dial_timeout":
			duration, err := time.ParseDuration(params.Get(v))
			if err != nil {
				return fmt.Errorf("clickhouse [dsn parse]: dial timeout: %s", err)
			}
			o.DialTimeout = duration
		case "block_buffer_size":
			if blockBufferSize, err := strconv.ParseUint(params.Get(v), 10, 8); err == nil {
				if blockBufferSize <= 0 {
					return fmt.Errorf("block_buffer_size must be greater than 0")
				}
				o.BlockBufferSize = uint8(blockBufferSize)
			} else {
				return err
			}
		case "read_timeout":
			duration, err := time.ParseDuration(params.Get(v))
			if err != nil {
				return fmt.Errorf("clickhouse [dsn parse]:read timeout: %s", err)
			}
			o.ReadTimeout = duration
		case "secure":
			secureParam := params.Get(v)
			if secureParam == "" {
				secure = true
			} else {
				secure, err = strconv.ParseBool(secureParam)
				if err != nil {
					return fmt.Errorf("clickhouse [dsn parse]:secure: %s", err)
				}
			}
		case "skip_verify":
			skipVerifyParam := params.Get(v)
			if skipVerifyParam == "" {
				skipVerify = true
			} else {
				skipVerify, err = strconv.ParseBool(skipVerifyParam)
				if err != nil {
					return fmt.Errorf("clickhouse [dsn parse]:verify: %s", err)
				}
			}
		case "tls_server_name":
			tlsServerName = strings.TrimSpace(params.Get(v))
			if tlsServerName == "" {
				return fmt.Errorf("clickhouse [dsn parse]: tls_server_name must not be empty")
			}
		case "connection_open_strategy":
			switch params.Get(v) {
			case "in_order":
				o.ConnOpenStrategy = ConnOpenInOrder
			case "round_robin":
				o.ConnOpenStrategy = ConnOpenRoundRobin
			case "random":
				o.ConnOpenStrategy = ConnOpenRandom
			}
		case "max_open_conns":
			maxOpenConns, err := strconv.Atoi(params.Get(v))
			if err != nil {
				return fmt.Errorf("max_open_conns invalid value: %w", err)
			}
			o.MaxOpenConns = maxOpenConns
		case "max_idle_conns":
			maxIdleConns, err := strconv.Atoi(params.Get(v))
			if err != nil {
				return fmt.Errorf("max_idle_conns invalid value: %w", err)
			}
			o.MaxIdleConns = maxIdleConns
		case "conn_max_lifetime":
			connMaxLifetime, err := time.ParseDuration(params.Get(v))
			if err != nil {
				return fmt.Errorf("conn_max_lifetime invalid value: %w", err)
			}
			o.ConnMaxLifetime = connMaxLifetime
		case "username":
			o.Auth.Username = params.Get(v)
		case "password":
			o.Auth.Password = params.Get(v)
		case "database":
			o.Auth.Database = params.Get(v)
		case "client_info_product":
			chunks := strings.Split(params.Get(v), ",")

			for _, chunk := range chunks {
				name, version, _ := strings.Cut(chunk, "/")

				o.ClientInfo.Products = append(o.ClientInfo.Products, struct{ Name, Version string }{
					name,
					version,
				})
			}
		case "http_proxy":
			proxyURL, err := url.Parse(params.Get(v))
			if err != nil {
				return fmt.Errorf("clickhouse [dsn parse]: http_proxy: %s", err)
			}
			o.HTTPProxyURL = proxyURL
		case "http_path":
			path := params.Get(v)
			if path != "" && !strings.HasPrefix(path, "/") {
				path = "/" + path
			}
			o.HttpUrlPath = path
		default:
			switch p := strings.ToLower(params.Get(v)); p {
			case "true":
				o.Settings[v] = int(1)
			case "false":
				o.Settings[v] = int(0)
			default:
				if n, err := strconv.Atoi(p); err == nil {
					o.Settings[v] = n
				} else {
					o.Settings[v] = p
				}
			}
		}
	}
	if tlsServerName != "" && !secure {
		return fmt.Errorf("clickhouse [dsn parse]: tls_server_name requires secure=true")
	}
	if secure {
		o.TLS = &tls.Config{
			InsecureSkipVerify: skipVerify,
			ServerName:         tlsServerName,
		}
	}
	o.scheme = dsn.Scheme
	switch dsn.Scheme {
	case "http":
		if secure {
			return fmt.Errorf("clickhouse [dsn parse]: http with TLS specify")
		}
		o.Protocol = HTTP
	case "https":
		if !secure {
			return fmt.Errorf("clickhouse [dsn parse]: https without TLS")
		}
		o.Protocol = HTTP
	default:
		o.Protocol = Native
	}
	return nil
}

// receive copy of Options, so we don't modify original - so its reusable
func (o Options) setDefaults() *Options {
	if len(o.Auth.Username) == 0 {
		o.Auth.Username = "default"
	}
	if o.DialTimeout == 0 {
		o.DialTimeout = time.Second * 30
	}
	if o.ReadTimeout == 0 {
		o.ReadTimeout = time.Second * time.Duration(300)
	}
	if o.MaxIdleConns <= 0 {
		o.MaxIdleConns = 5
	}
	if o.MaxOpenConns <= 0 {
		o.MaxOpenConns = o.MaxIdleConns + 5
	}
	if o.ConnMaxLifetime == 0 {
		o.ConnMaxLifetime = time.Hour
	}
	if o.BlockBufferSize <= 0 {
		o.BlockBufferSize = 2
	}
	if o.MaxCompressionBuffer <= 0 {
		o.MaxCompressionBuffer = 10485760
	}
	if len(o.Addr) == 0 {
		switch o.Protocol {
		case Native:
			o.Addr = []string{"localhost:9000"}
		case HTTP:
			o.Addr = []string{"localhost:8123"}
		}
	}
	return &o
}

// logger returns the appropriate logger based on the Options configuration.
// Priority order:
// 1. If Debug=true and Debugf is set, use legacy Debugf (backward compatibility)
// 2. If Logger is set, use the provided logger
// 3. Otherwise, use a noop logger (no logging)
func (o *Options) logger() *slog.Logger {
	// Backward compatibility: if legacy Debug/Debugf is set, use it
	if o.Debug && o.Debugf != nil {
		return newDebugfLogger(o.Debugf)
	}

	// If user provided a custom logger, use it
	if o.Logger != nil {
		return o.Logger
	}

	// Default: no logging
	return newNoopLogger()
}
