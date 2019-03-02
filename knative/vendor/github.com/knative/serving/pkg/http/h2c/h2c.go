/*
Copyright 2018 The Knative Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package h2c

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func NewServer(addr string, h http.Handler) *http.Server {
	h1s := &http.Server{
		Addr:    addr,
		Handler: h2c.NewHandler(h, &http2.Server{}),
	}

	return h1s
}

func ListenAndServe(addr string, h http.Handler) error {
	s := NewServer(addr, h)
	return s.ListenAndServe()
}

// NewTransport will reroute all https traffic to http. This is
// to explicitly allow h2c (http2 without TLS) transport.
// See https://github.com/golang/go/issues/14141 for more details.
var DefaultTransport http.RoundTripper = &http2.Transport{
	AllowHTTP: true,
	DialTLS: func(netw, addr string, cfg *tls.Config) (net.Conn, error) {
		d := &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}

		return d.Dial(netw, addr)
	},
}
