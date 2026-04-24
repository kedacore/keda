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
	"fmt"
	"io"
	"net/http"
)

const httpStatusCodeThreshold = 299

// Response represents the API response.
type Response struct {
	StatusCode int
	Header     http.Header
	Body       io.ReadCloser
}

// String returns the response as a string.
func (r Response) String() string {
	if r.Body != nil {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			body = []byte(fmt.Sprintf("<error reading response body: %v>", err))
		}
		return fmt.Sprintf("%s %s", r.Status(), body)
	}
	return r.Status()
}

// Status retuens the response status as string.
func (r Response) Status() string {
	status := http.StatusText(r.StatusCode)
	if status == "" {
		status = "<nil>"
	}
	return fmt.Sprintf("[%d %s]", r.StatusCode, status)
}

// IsError returns true when the response status indicates failure.
func (r *Response) IsError() bool {
	return r.StatusCode > httpStatusCodeThreshold
}
