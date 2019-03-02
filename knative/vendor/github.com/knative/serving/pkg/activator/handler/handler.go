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

package handler

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"time"

	"github.com/knative/pkg/websocket"
	"github.com/knative/serving/pkg/activator"
	"github.com/knative/serving/pkg/activator/util"
	pkghttp "github.com/knative/serving/pkg/http"
	"go.uber.org/zap"
)

// ActivationHandler will wait for an active endpoint for a revision
// to be available before proxing the request
type ActivationHandler struct {
	Activator activator.Activator
	Logger    *zap.SugaredLogger
	Transport http.RoundTripper
	Reporter  activator.StatsReporter
	Throttler *activator.Throttler
}

func (a *ActivationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	namespace := pkghttp.LastHeaderValue(r.Header, activator.RevisionHeaderNamespace)
	name := pkghttp.LastHeaderValue(r.Header, activator.RevisionHeaderName)
	start := time.Now()
	revID := activator.RevisionID{namespace, name}

	// ActiveEndpoint() will block until the first endpoint is available.
	ar := a.Activator.ActiveEndpoint(namespace, name)

	if ar.Error != nil {
		msg := fmt.Sprintf("Error getting active endpoint: %v", ar.Error)
		a.Logger.Error(msg)
		http.Error(w, msg, ar.Status)
		return
	}
	target := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%d", ar.Endpoint.FQDN, ar.Endpoint.Port),
	}

	err := a.Throttler.Try(revID, func() {
		attempts, httpStatus := a.proxyRequest(w, r, target)

		// Report the metrics
		duration := time.Since(start)

		a.Reporter.ReportRequestCount(namespace, ar.ServiceName, ar.ConfigurationName, name, httpStatus, attempts, 1.0)
		a.Reporter.ReportResponseTime(namespace, ar.ServiceName, ar.ConfigurationName, name, httpStatus, duration)
	})
	if err != nil {
		if err == activator.ErrActivatorOverload {
			http.Error(w, activator.ErrActivatorOverload.Error(), http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			a.Logger.Errorw("Error processing request in the activator", zap.Error(err))
		}
	}
}

func (a *ActivationHandler) proxyRequest(w http.ResponseWriter, r *http.Request, target *url.URL) (int, int) {
	capture := &statusCapture{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = a.Transport
	proxy.FlushInterval = -1

	attempts := int(1) // one attempt is always needed
	proxy.ModifyResponse = func(r *http.Response) error {
		if numTries := r.Header.Get(activator.RequestCountHTTPHeader); numTries != "" {
			if count, err := strconv.Atoi(numTries); err == nil {
				a.Logger.Infof("got %d attempts", count)
				attempts = count
			} else {
				a.Logger.Errorf("Value in %v header is not a valid integer. Error: %v", activator.RequestCountHTTPHeader, err)
			}
		}

		// We don't return this header to the user. It's only used to transport
		// state in the activator.
		r.Header.Del(activator.RequestCountHTTPHeader)

		return nil
	}
	util.SetupHeaderPruning(proxy)

	proxy.ServeHTTP(capture, r)
	return attempts, capture.statusCode
}

type statusCapture struct {
	http.ResponseWriter
	statusCode int
}

var _ http.Flusher = (*statusCapture)(nil)

func (s *statusCapture) WriteHeader(statusCode int) {
	s.statusCode = statusCode
	s.ResponseWriter.WriteHeader(statusCode)
}

// Hijack calls Hijack() on the wrapped http.ResponseWriter if it implements
// http.Hijacker interface, which is required for net/http/httputil/reverseproxy
// to handle connection upgrade/switching protocol.  Otherwise returns an error.
func (s *statusCapture) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return websocket.HijackIfPossible(s.ResponseWriter)
}

func (s *statusCapture) Flush() {
	s.ResponseWriter.(http.Flusher).Flush()
}
