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

// request contains logic to make polling HTTP requests against an endpoint with optional host spoofing.

package test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/knative/pkg/test/logging"
	"github.com/knative/pkg/test/spoof"
)

// MatchesAny is a NOP matcher. This is useful for polling until a 200 is returned.
func MatchesAny(_ *spoof.Response) (bool, error) {
	return true, nil
}

// Retrying modifies a ResponseChecker to retry certain response codes.
func Retrying(rc spoof.ResponseChecker, codes ...int) spoof.ResponseChecker {
	return func(resp *spoof.Response) (bool, error) {
		for _, code := range codes {
			if resp.StatusCode == code {
				// Returning (false, nil) causes SpoofingClient.Poll to retry.
				// sc.logger.Infof("Retrying for code %v", resp.StatusCode)
				return false, nil
			}
		}

		// If we didn't match any retryable codes, invoke the ResponseChecker that we wrapped.
		return rc(resp)
	}
}

// IsOneOfStatusCodes checks that the response code is equal to the given one.
func IsOneOfStatusCodes(codes ...int) spoof.ResponseChecker {
	return func(resp *spoof.Response) (bool, error) {
		for _, code := range codes {
			if resp.StatusCode == code {
				return true, nil
			}
		}

		return true, fmt.Errorf("status = %d, want one of: %v, body = %s", resp.StatusCode, codes, string(resp.Body))
	}
}

// IsStatusOK checks that the response code is a 200.
func IsStatusOK(resp *spoof.Response) (bool, error) {
	return IsOneOfStatusCodes(http.StatusOK)(resp)
}

// MatchesBody checks that the *first* response body matches the "expected" body, otherwise failing.
func MatchesBody(expected string) spoof.ResponseChecker {
	return func(resp *spoof.Response) (bool, error) {
		if !strings.Contains(string(resp.Body), expected) {
			// Returning (true, err) causes SpoofingClient.Poll to fail.
			return true, fmt.Errorf("body mismatch: got %q, want %q", string(resp.Body), expected)
		}

		return true, nil
	}
}

// EventuallyMatchesBody checks that the response body *eventually* matches the expected body.
// TODO(#1178): Delete me. We don't want to need this; we should be waiting for an appropriate Status instead.
func EventuallyMatchesBody(expected string) spoof.ResponseChecker {
	return func(resp *spoof.Response) (bool, error) {
		if !strings.Contains(string(resp.Body), expected) {
			// Returning (false, nil) causes SpoofingClient.Poll to retry.
			return false, nil
		}

		return true, nil
	}
}

// MatchesAllOf combines multiple ResponseCheckers to one ResponseChecker with a logical AND. The
// checkers are executed in order. The first function to trigger an error or a retry will short-circuit
// the other functions (they will not be executed).
//
// This is useful for combining a body with a status check like:
// MatchesAllOf(IsStatusOK, MatchesBody("test"))
//
// The MatchesBody check will only be executed after the IsStatusOK has passed.
func MatchesAllOf(checkers ...spoof.ResponseChecker) spoof.ResponseChecker {
	return func(resp *spoof.Response) (bool, error) {
		for _, checker := range checkers {
			done, err := checker(resp)
			if err != nil || !done {
				return done, err
			}
		}
		return true, nil
	}
}

// WaitForEndpointState will poll an endpoint until inState indicates the state is achieved,
// or default timeout is reached.
// If resolvableDomain is false, it will use kubeClientset to look up the ingress and spoof
// the domain in the request headers, otherwise it will make the request directly to domain.
// desc will be used to name the metric that is emitted to track how long it took for the
// domain to get into the state checked by inState.  Commas in `desc` must be escaped.
func WaitForEndpointState(kubeClient *KubeClient, logf logging.FormatLogger, domain string, inState spoof.ResponseChecker, desc string, resolvable bool) (*spoof.Response, error) {
	return WaitForEndpointStateWithTimeout(kubeClient, logf, domain, inState, desc, resolvable, spoof.RequestTimeout)
}

// WaitForEndpointStateWithTimeout will poll an endpoint until inState indicates the state is achieved
// or the provided timeout is achieved.
// If resolvableDomain is false, it will use kubeClientset to look up the ingress and spoof
// the domain in the request headers, otherwise it will make the request directly to domain.
// desc will be used to name the metric that is emitted to track how long it took for the
// domain to get into the state checked by inState.  Commas in `desc` must be escaped.
func WaitForEndpointStateWithTimeout(
	kubeClient *KubeClient, logf logging.FormatLogger, domain string, inState spoof.ResponseChecker,
	desc string, resolvable bool, timeout time.Duration) (*spoof.Response, error) {
	defer logging.GetEmitableSpan(context.Background(), fmt.Sprintf("WaitForEndpointState/%s", desc)).End()

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s", domain), nil)
	if err != nil {
		return nil, err
	}

	client, err := NewSpoofingClient(kubeClient, logf, domain, resolvable)
	if err != nil {
		return nil, err
	}
	client.RequestTimeout = timeout

	return client.Poll(req, inState)
}
