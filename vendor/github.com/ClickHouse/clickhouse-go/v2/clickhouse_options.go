// Licensed to ClickHouse, Inc. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. ClickHouse, Inc. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package clickhouse

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ClickHouse/ch-go/compress"
	"github.com/pkg/errors"
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
	CompressionZSTD    = CompressionMethod(compress.ZSTD)
	CompressionGZIP    = CompressionMethod(0x95)
	CompressionDeflate = CompressionMethod(0x96)
	CompressionBrotli  = CompressionMethod(0x97)
)

var compressionMap = map[string]CompressionMethod{
	"none":    CompressionNone,
	"zstd":    CompressionZSTD,
	"lz4":     CompressionLZ4,
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
	// this only applies to zlib and brotli compression algorithms
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
	conn *connect
}

type HTTPProxy func(*http.Request) (*url.URL, error)

type Options struct {
	Protocol   Protocol
	ClientInfo ClientInfo

	TLS                  *tls.Config
	Addr                 []string
	Auth                 Auth
	DialContext          func(ctx context.Context, addr string) (net.Conn, error)
	DialStrategy         func(ctx context.Context, connID int, options *Options, dial Dial) (DialResult, error)
	Debug                bool
	Debugf               func(format string, v ...any) // only works when Debug is true
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
	BlockBufferSize      uint8             // default 2 - can be overwritten on query
	MaxCompressionBuffer int               // default 10485760 - measured in bytes  i.e.

	// HTTPProxy specifies an HTTP proxy URL to use for requests made by the client.
	HTTPProxyURL *url.URL

	scheme      string
	ReadTimeout time.Duration
}

func (o *Options) fromDSN(in string) error {
	dsn, err := url.Parse(in)
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
		secure     bool
		params     = dsn.Query()
		skipVerify bool
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
				return errors.Wrap(err, "compress_level invalid value")
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
				return errors.Wrap(err, "max_compression_buffer invalid value")
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
				return errors.Wrap(err, "max_open_conns invalid value")
			}
			o.MaxOpenConns = maxOpenConns
		case "max_idle_conns":
			maxIdleConns, err := strconv.Atoi(params.Get(v))
			if err != nil {
				return errors.Wrap(err, "max_idle_conns invalid value")
			}
			o.MaxIdleConns = maxIdleConns
		case "conn_max_lifetime":
			connMaxLifetime, err := time.ParseDuration(params.Get(v))
			if err != nil {
				return errors.Wrap(err, "conn_max_lifetime invalid value")
			}
			o.ConnMaxLifetime = connMaxLifetime
		case "username":
			o.Auth.Username = params.Get(v)
		case "password":
			o.Auth.Password = params.Get(v)
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
	if secure {
		o.TLS = &tls.Config{
			InsecureSkipVerify: skipVerify,
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
	if o.Addr == nil || len(o.Addr) == 0 {
		switch o.Protocol {
		case Native:
			o.Addr = []string{"localhost:9000"}
		case HTTP:
			o.Addr = []string{"localhost:8123"}
		}
	}
	return &o
}
