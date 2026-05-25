/*
Copyright 2024 The KEDA Authors

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

package aws

import (
	"net/http"

	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
)

// NewHTTPClient returns an *http.Client configured for use with AWS SDK clients.
// Each call allocates a fresh *http.Transport so that callers have independent
// connection pools with independent lifecycles — this is required because each
// scaler calls CloseIdleConnections on its own client at teardown.
//
// We deliberately use a raw *http.Transport (via GetTransport) rather than the
// SDK's BuildableClient.Freeze(), because Freeze wraps the transport in
// suppressBadHTTPRedirectTransport which does not implement CloseIdleConnections.
// That would make http.Client.CloseIdleConnections() a no-op and leak TCP
// connections across scaler restarts.
//
// To satisfy the AWS SDK's expectation that custom HTTP clients do not follow
// 301/302 redirects (see aws.HTTPClient interface docs and
// https://github.com/aws/aws-sdk-go-v2/blob/v1.41.7/aws/transport/http/client.go#L283-L318),
// we apply limitRedirect as the CheckRedirect policy. This mirrors the SDK's
// own limitedRedirect behavior.
func NewHTTPClient() *http.Client {
	return &http.Client{
		Transport:     awshttp.NewBuildableClient().GetTransport(),
		CheckRedirect: limitRedirect,
	}
}

// limitRedirect replicates the AWS SDK's redirect policy:
//   - Only 307/308 are followed (they preserve the HTTP method).
//   - X-Amz-Security-Token is stripped on cross-host redirects to prevent
//     credential leakage when awsEndpoint is user-supplied.
//   - All other redirects (301, 302, etc.) are suppressed.
//
// Reference: https://github.com/aws/aws-sdk-go-v2/blob/v1.41.7/aws/transport/http/client.go#L291-L318
func limitRedirect(req *http.Request, via []*http.Request) error {
	resp := req.Response
	switch resp.StatusCode {
	case 307, 308:
		// len(via) > 0 is always true when CheckRedirect is called, but we keep
		// the guard to mirror the SDK implementation and as defensive bounds-checking.
		// See: https://github.com/aws/aws-sdk-go-v2/blob/v1.41.7/aws/transport/http/client.go#L307
		if len(via) > 0 {
			last := via[len(via)-1]
			if last.URL.Host != req.URL.Host {
				req.Header.Del("X-Amz-Security-Token")
			}
		}
		return nil
	}
	return http.ErrUseLastResponse
}
