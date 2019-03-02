/*
Copyright 2019 The Knative Authors

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

// spoof contains logic to make polling HTTP requests against an endpoint with optional host spoofing.

package spoof

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/knative/pkg/test/logging"
	"github.com/knative/pkg/test/zipkin"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/plugin/ochttp/propagation/b3"
	"go.opencensus.io/trace"
)

const (
	requestInterval = 1 * time.Second
	// RequestTimeout is the default timeout for the polling requests.
	RequestTimeout = 5 * time.Minute
	// TODO(tcnghia): These probably shouldn't be hard-coded here?
	istioIngressNamespace = "istio-system"
	istioIngressName      = "istio-ingressgateway"
	// Name of the temporary HTTP header that is added to http.Request to indicate that
	// it is a SpoofClient.Poll request. This header is removed before making call to backend.
	pollReqHeader = "X-Kn-Poll-Request-Do-Not-Trace"
)

// Response is a stripped down subset of http.Response. The is primarily useful
// for ResponseCheckers to inspect the response body without consuming it.
// Notably, Body is a byte slice instead of an io.ReadCloser.
type Response struct {
	Status     string
	StatusCode int
	Header     http.Header
	Body       []byte
}

// Interface defines the actions that can be performed by the spoofing client.
type Interface interface {
	Do(*http.Request) (*Response, error)
	Poll(*http.Request, ResponseChecker) (*Response, error)
}

// https://medium.com/stupid-gopher-tricks/ensuring-go-interface-satisfaction-at-compile-time-1ed158e8fa17
var _ Interface = (*SpoofingClient)(nil)

// ResponseChecker is used to determine when SpoofinClient.Poll is done polling.
// This allows you to predicate wait.PollImmediate on the request's http.Response.
//
// See the apimachinery wait package:
// https://github.com/kubernetes/apimachinery/blob/cf7ae2f57dabc02a3d215f15ca61ae1446f3be8f/pkg/util/wait/wait.go#L172
type ResponseChecker func(resp *Response) (done bool, err error)

// SpoofingClient is a minimal HTTP client wrapper that spoofs the domain of requests
// for non-resolvable domains.
type SpoofingClient struct {
	Client          *http.Client
	RequestInterval time.Duration
	RequestTimeout  time.Duration

	endpoint string
	domain   string

	logf logging.FormatLogger
}

// New returns a SpoofingClient that rewrites requests if the target domain is not `resolveable`.
// It does this by looking up the ingress at construction time, so reusing a client will not
// follow the ingress if it moves (or if there are multiple ingresses).
//
// If that's a problem, see test/request.go#WaitForEndpointState for oneshot spoofing.
func New(kubeClientset *kubernetes.Clientset, logf logging.FormatLogger, domain string, resolvable bool, endpointOverride string) (*SpoofingClient, error) {
	sc := SpoofingClient{
		Client:          &http.Client{Transport: &ochttp.Transport{Propagation: &b3.HTTPFormat{}}}, // Using ochttp Transport required for zipkin-tracing
		RequestInterval: requestInterval,
		RequestTimeout:  RequestTimeout,
		logf:            logf,
	}

	if !resolvable {
		e := &endpointOverride
		if endpointOverride == "" {
			var err error
			// If the domain that the Route controller is configured to assign to Route.Status.Domain
			// (the domainSuffix) is not resolvable, we need to retrieve the IP of the endpoint and
			// spoof the Host in our requests.
			e, err = GetServiceEndpoint(kubeClientset)
			if err != nil {
				return nil, err
			}
		}

		sc.endpoint = *e
		sc.domain = domain
	} else {
		// If the domain is resolvable, we can use it directly when we make requests.
		sc.endpoint = domain
	}

	return &sc, nil
}

// GetServiceEndpoint gets the endpoint IP or hostname to use for the service.
func GetServiceEndpoint(kubeClientset *kubernetes.Clientset) (*string, error) {
	ingressName := istioIngressName
	if gatewayOverride := os.Getenv("GATEWAY_OVERRIDE"); gatewayOverride != "" {
		ingressName = gatewayOverride
	}
	ingressNamespace := istioIngressNamespace
	if gatewayNsOverride := os.Getenv("GATEWAY_NAMESPACE_OVERRIDE"); gatewayNsOverride != "" {
		ingressNamespace = gatewayNsOverride
	}

	ingress, err := kubeClientset.CoreV1().Services(ingressNamespace).Get(ingressName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	endpoint, err := endpointFromService(ingress)
	if err != nil {
		return nil, err
	}
	return &endpoint, nil
}

// endpointFromService extracts the endpoint from the service's ingress.
func endpointFromService(svc *v1.Service) (string, error) {
	ingresses := svc.Status.LoadBalancer.Ingress
	if len(ingresses) != 1 {
		return "", fmt.Errorf("Expected exactly one ingress load balancer, instead had %d: %v", len(ingresses), ingresses)
	}
	itu := ingresses[0]

	switch {
	case itu.IP != "":
		return itu.IP, nil
	case itu.Hostname != "":
		return itu.Hostname, nil
	default:
		return "", fmt.Errorf("Expected ingress loadbalancer IP or hostname for %s to be set, instead was empty", svc.Name)
	}
}

// Do dispatches to the underlying http.Client.Do, spoofing domains as needed
// and transforming the http.Response into a spoof.Response.
// Each response is augmented with "ZipkinTraceID" header that identifies the zipkin trace corresponding to the request.
func (sc *SpoofingClient) Do(req *http.Request) (*Response, error) {
	// Controls the Host header, for spoofing.
	if sc.domain != "" {
		req.Host = sc.domain
	}

	// Controls the actual resolution.
	if sc.endpoint != "" {
		req.URL.Host = sc.endpoint
	}

	// Starting span to capture zipkin trace.
	traceContext, span := trace.StartSpan(req.Context(), "SpoofingClient-Trace")
	defer span.End()

	// Check to see if the call to this method is coming from a Poll call.
	logZipkinTrace := true
	if req.Header.Get(pollReqHeader) != "" {
		req.Header.Del(pollReqHeader)
		logZipkinTrace = false
	}
	resp, err := sc.Client.Do(req.WithContext(traceContext))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	resp.Header.Add(zipkin.ZipkinTraceIDHeader, span.SpanContext().TraceID.String())
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	spoofResp := &Response{
		Status:     resp.Status,
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		Body:       body,
	}

	if logZipkinTrace {
		sc.logZipkinTrace(spoofResp)
	}

	return spoofResp, nil
}

// Poll executes an http request until it satisfies the inState condition or encounters an error.
func (sc *SpoofingClient) Poll(req *http.Request, inState ResponseChecker) (*Response, error) {
	var (
		resp *Response
		err  error
	)

	err = wait.PollImmediate(sc.RequestInterval, sc.RequestTimeout, func() (bool, error) {
		// As we may do multiple Do calls as part of a single Poll we add this temporary header
		// to the request to indicate to Do method not to log Zipkin trace, instead it is
		// handled by this method itself.
		req.Header.Add(pollReqHeader, "True")
		resp, err = sc.Do(req)
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Timeout() {
				sc.logf("Retrying %s for TCP timeout %v", req.URL.String(), err)
				return false, nil
			}
			return true, err
		}

		return inState(resp)
	})

	if resp != nil {
		sc.logZipkinTrace(resp)
	}

	return resp, err
}

// logZipkinTrace provides support to log Zipkin Trace for param: spoofResponse
// We only log Zipkin trace for HTTP server errors i.e for HTTP status codes between 500 to 600
func (sc *SpoofingClient) logZipkinTrace(spoofResp *Response) {
	if !zipkin.ZipkinTracingEnabled || spoofResp.StatusCode < http.StatusInternalServerError || spoofResp.StatusCode >= 600 {
		return
	}

	traceID := spoofResp.Header.Get(zipkin.ZipkinTraceIDHeader)
	if err := zipkin.CheckZipkinPortAvailability(); err == nil {
		sc.logf("port-forwarding for Zipkin is not-setup. Failing Zipkin Trace retrieval")
		return
	}

	sc.logf("Logging Zipkin Trace: %s", traceID)

	zipkinTraceEndpoint := zipkin.ZipkinTraceEndpoint + traceID
	// Sleep to ensure all traces are correctly pushed on the backend.
	time.Sleep(5 * time.Second)
	resp, err := http.Get(zipkinTraceEndpoint)
	if err != nil {
		sc.logf("Error retrieving Zipkin trace: %v", err)
		return
	}
	defer resp.Body.Close()

	trace, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		sc.logf("Error reading Zipkin trace response: %v", err)
		return
	}

	var prettyJSON bytes.Buffer
	if error := json.Indent(&prettyJSON, trace, "", "\t"); error != nil {
		sc.logf("JSON Parser Error while trying for Pretty-Format: %v, Original Response: %s", error, string(trace))
		return
	}
	sc.logf("%s", prettyJSON.String())
}
