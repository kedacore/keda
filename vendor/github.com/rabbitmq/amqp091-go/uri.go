// Copyright (c) 2021 VMware, Inc. or its affiliates. All Rights Reserved.
// Copyright (c) 2012-2021, Sean Treadway, SoundCloud Ltd.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package amqp091

import (
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
)

var errURIScheme = errors.New("AMQP scheme must be either 'amqp://' or 'amqps://'")
var errURIWhitespace = errors.New("URI must not contain whitespace")

var schemePorts = map[string]int{
	"amqp":  5672,
	"amqps": 5671,
}

var defaultURI = URI{
	Scheme:   "amqp",
	Host:     "localhost",
	Port:     5672,
	Username: "guest",
	Password: "guest",
	Vhost:    "/",
}

// URI represents a parsed AMQP URI string.
type URI struct {
	Scheme     string
	Host       string
	Port       int
	Username   string
	Password   string
	Vhost      string
	CertFile   string // client TLS auth - path to certificate (PEM)
	CACertFile string // client TLS auth - path to CA certificate (PEM)
	KeyFile    string // client TLS auth - path to private key (PEM)
	ServerName string // client TLS auth - server name
}

// ParseURI attempts to parse the given AMQP URI according to the spec.
// See http://www.rabbitmq.com/uri-spec.html.
//
// Default values for the fields are:
//
//   Scheme: amqp
//   Host: localhost
//   Port: 5672
//   Username: guest
//   Password: guest
//   Vhost: /
//
// Supports TLS query parameters. See https://www.rabbitmq.com/uri-query-parameters.html
//
//   certfile: <path/to/client_cert.pem>
//   keyfile: <path/to/client_key.pem>
//   cacertfile: <path/to/ca.pem>
//   server_name_indication: <server name>
//
// If cacertfile is not provided, system CA certificates will be used.
// Mutual TLS (client auth) will be enabled only in case keyfile AND certfile provided.
//
// If Config.TLSClientConfig is set, TLS parameters from URI will be ignored.
func ParseURI(uri string) (URI, error) {
	builder := defaultURI

	if strings.Contains(uri, " ") {
		return builder, errURIWhitespace
	}

	u, err := url.Parse(uri)
	if err != nil {
		return builder, err
	}

	defaultPort, okScheme := schemePorts[u.Scheme]

	if okScheme {
		builder.Scheme = u.Scheme
	} else {
		return builder, errURIScheme
	}

	host := u.Hostname()
	port := u.Port()

	if host != "" {
		builder.Host = host
	}

	if port != "" {
		port32, err := strconv.ParseInt(port, 10, 32)
		if err != nil {
			return builder, err
		}
		builder.Port = int(port32)
	} else {
		builder.Port = defaultPort
	}

	if u.User != nil {
		builder.Username = u.User.Username()
		if password, ok := u.User.Password(); ok {
			builder.Password = password
		}
	}

	if u.Path != "" {
		if strings.HasPrefix(u.Path, "/") {
			if u.Host == "" && strings.HasPrefix(u.Path, "///") {
				// net/url doesn't handle local context authorities and leaves that up
				// to the scheme handler.  In our case, we translate amqp:/// into the
				// default host and whatever the vhost should be
				if len(u.Path) > 3 {
					builder.Vhost = u.Path[3:]
				}
			} else if len(u.Path) > 1 {
				builder.Vhost = u.Path[1:]
			}
		} else {
			builder.Vhost = u.Path
		}
	}

	// see https://www.rabbitmq.com/uri-query-parameters.html
	params := u.Query()
	builder.CertFile = params.Get("certfile")
	builder.KeyFile = params.Get("keyfile")
	builder.CACertFile = params.Get("cacertfile")
	builder.ServerName = params.Get("server_name_indication")

	return builder, nil
}

// PlainAuth returns a PlainAuth structure based on the parsed URI's
// Username and Password fields.
func (uri URI) PlainAuth() *PlainAuth {
	return &PlainAuth{
		Username: uri.Username,
		Password: uri.Password,
	}
}

// AMQPlainAuth returns a PlainAuth structure based on the parsed URI's
// Username and Password fields.
func (uri URI) AMQPlainAuth() *AMQPlainAuth {
	return &AMQPlainAuth{
		Username: uri.Username,
		Password: uri.Password,
	}
}

func (uri URI) String() string {
	authority, err := url.Parse("")
	if err != nil {
		return err.Error()
	}

	authority.Scheme = uri.Scheme

	if uri.Username != defaultURI.Username || uri.Password != defaultURI.Password {
		authority.User = url.User(uri.Username)

		if uri.Password != defaultURI.Password {
			authority.User = url.UserPassword(uri.Username, uri.Password)
		}
	}

	authority.Host = net.JoinHostPort(uri.Host, strconv.Itoa(uri.Port))

	if defaultPort, found := schemePorts[uri.Scheme]; !found || defaultPort != uri.Port {
		authority.Host = net.JoinHostPort(uri.Host, strconv.Itoa(uri.Port))
	} else {
		// JoinHostPort() automatically add brackets to the host if it's
		// an IPv6 address.
		//
		// If not port is specified, JoinHostPort() return an IP address in the
		// form of "[::1]:", so we use TrimSuffix() to remove the extra ":".
		authority.Host = strings.TrimSuffix(net.JoinHostPort(uri.Host, ""), ":")
	}

	if uri.Vhost != defaultURI.Vhost {
		// Make sure net/url does not double escape, e.g.
		// "%2F" does not become "%252F".
		authority.Path = uri.Vhost
		authority.RawPath = url.QueryEscape(uri.Vhost)
	} else {
		authority.Path = "/"
	}

	return authority.String()
}
