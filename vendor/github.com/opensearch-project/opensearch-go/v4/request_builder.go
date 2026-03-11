// SPDX-License-Identifier: Apache-2.0
//
// The OpenSearch Contributors require contributions made to
// this file be licensed under the Apache-2.0 license or a
// compatible open source license.
//
// Modifications Copyright OpenSearch Contributors. See
// GitHub history for details.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package opensearch

import (
	"io"
	"net/http"
)

const (
	headerContentType = "Content-Type"
)

// BuildRequest is a helper function to build a http.Request
func BuildRequest(method string, path string, body io.Reader, params map[string]string, headers http.Header) (*http.Request, error) {
	//nolint:noctx // ctx gets appended when the requests gets executet
	httpReq, err := http.NewRequest(method, path, body)
	if err != nil {
		return nil, err
	}

	if len(params) > 0 {
		q := httpReq.URL.Query()
		for k, v := range params {
			q.Set(k, v)
		}

		httpReq.URL.RawQuery = q.Encode()
	}

	if body != nil {
		httpReq.Header[headerContentType] = []string{"application/json"}
	}

	if len(headers) > 0 {
		if len(httpReq.Header) == 0 {
			httpReq.Header = headers
		} else {
			for k, vv := range headers {
				for _, v := range vv {
					httpReq.Header.Add(k, v)
				}
			}
		}
	}
	return httpReq, nil
}
