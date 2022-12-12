//
// DISCLAIMER
//
// Copyright 2017 ArangoDB GmbH, Cologne, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright holder is ArangoDB GmbH, Cologne, Germany
//
// Author Ewout Prangsma
//

package http

import (
	"encoding/json"
	"fmt"

	driver "github.com/arangodb/go-driver"
)

// httpJSONResponseElement implements driver.Response for an entry of an array response.
type httpJSONResponseElement struct {
	statusCode *int
	bodyObject map[string]*json.RawMessage
}

// StatusCode returns an HTTP compatible status code of the response.
func (r *httpJSONResponseElement) StatusCode() int {
	if r.statusCode == nil {
		statusCode := 200
		// Look for "error" field
		if errorFieldJSON, found := r.bodyObject["error"]; found {
			var hasError bool
			if err := json.Unmarshal(*errorFieldJSON, &hasError); err == nil && hasError {
				// We have an error, look for code field
				statusCode = 500
				if codeFieldJSON, found := r.bodyObject["code"]; found {
					var code int
					if err := json.Unmarshal(*codeFieldJSON, &code); err == nil {
						statusCode = code
					}
				}
			}
		}
		r.statusCode = &statusCode
	}
	return *r.statusCode
}

// Endpoint returns the endpoint that handled the request.
func (r *httpJSONResponseElement) Endpoint() string {
	return ""
}

// CheckStatus checks if the status of the response equals to one of the given status codes.
// If so, nil is returned.
// If not, an attempt is made to parse an error response in the body and an error is returned.
func (r *httpJSONResponseElement) CheckStatus(validStatusCodes ...int) error {
	statusCode := r.StatusCode()
	for _, x := range validStatusCodes {
		if x == statusCode {
			// Found valid status code
			return nil
		}
	}
	// Invalid status code, try to parse arango error response.
	var aerr driver.ArangoError
	if err := r.ParseBody("", &aerr); err == nil && aerr.HasError {
		// Found correct arango error.
		return aerr
	}

	// We do not have a valid error code, so we can only create one based on the HTTP status code.
	return driver.ArangoError{
		HasError:     true,
		Code:         statusCode,
		ErrorMessage: fmt.Sprintf("Unexpected status code %d", statusCode),
	}
}

// Header returns the value of a response header with given key.
// If no such header is found, an empty string is returned.
func (r *httpJSONResponseElement) Header(key string) string {
	return ""
}

// ParseBody performs protocol specific unmarshalling of the response data into the given result.
// If the given field is non-empty, the contents of that field will be parsed into the given result.
func (r *httpJSONResponseElement) ParseBody(field string, result interface{}) error {
	if result != nil {
		if err := parseBody(r.bodyObject, field, result); err != nil {
			return driver.WithStack(err)
		}
	}
	return nil
}

// ParseArrayBody performs protocol specific unmarshalling of the response array data into individual response objects.
// This can only be used for requests that return an array of objects.
func (r *httpJSONResponseElement) ParseArrayBody() ([]driver.Response, error) {
	return nil, driver.WithStack(driver.InvalidArgumentError{Message: "ParseArrayBody not allowed"})
}
