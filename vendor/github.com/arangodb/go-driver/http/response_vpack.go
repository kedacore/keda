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
	"fmt"
	"net/http"

	velocypack "github.com/arangodb/go-velocypack"

	driver "github.com/arangodb/go-driver"
)

// httpVPackResponse implements driver.Response for standard golang Velocypack encoded http responses.
type httpVPackResponse struct {
	resp        *http.Response
	rawResponse []byte
	slice       velocypack.Slice
	bodyArray   []driver.Response
}

// StatusCode returns an HTTP compatible status code of the response.
func (r *httpVPackResponse) StatusCode() int {
	return r.resp.StatusCode
}

// Endpoint returns the endpoint that handled the request.
func (r *httpVPackResponse) Endpoint() string {
	u := *r.resp.Request.URL
	u.Path = ""
	u.RawQuery = ""
	return u.String()
}

// CheckStatus checks if the status of the response equals to one of the given status codes.
// If so, nil is returned.
// If not, an attempt is made to parse an error response in the body and an error is returned.
func (r *httpVPackResponse) CheckStatus(validStatusCodes ...int) error {
	for _, x := range validStatusCodes {
		if x == r.resp.StatusCode {
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
		Code:         r.resp.StatusCode,
		ErrorMessage: fmt.Sprintf("Unexpected status code %d", r.resp.StatusCode),
	}
}

// Header returns the value of a response header with given key.
// If no such header is found, an empty string is returned.
func (r *httpVPackResponse) Header(key string) string {
	return r.resp.Header.Get(key)
}

// ParseBody performs protocol specific unmarshalling of the response data into the given result.
// If the given field is non-empty, the contents of that field will be parsed into the given result.
func (r *httpVPackResponse) ParseBody(field string, result interface{}) error {
	slice, err := r.getSlice()
	if err != nil {
		return driver.WithStack(err)
	}
	if field != "" {
		var err error
		slice, err = slice.Get(field)
		if err != nil {
			return driver.WithStack(err)
		}
		if slice.IsNone() {
			// Field not found
			return nil
		}
	}
	if result != nil {
		if err := velocypack.Unmarshal(slice, result); err != nil {
			return driver.WithStack(err)
		}
	}
	return nil
}

// ParseArrayBody performs protocol specific unmarshalling of the response array data into individual response objects.
// This can only be used for requests that return an array of objects.
func (r *httpVPackResponse) ParseArrayBody() ([]driver.Response, error) {
	if r.bodyArray == nil {
		slice, err := r.getSlice()
		if err != nil {
			return nil, driver.WithStack(err)
		}
		l, err := slice.Length()
		if err != nil {
			return nil, driver.WithStack(err)
		}

		bodyArray := make([]driver.Response, 0, l)
		it, err := velocypack.NewArrayIterator(slice)
		if err != nil {
			return nil, driver.WithStack(err)
		}
		for it.IsValid() {
			v, err := it.Value()
			if err != nil {
				return nil, driver.WithStack(err)
			}
			bodyArray = append(bodyArray, &httpVPackResponseElement{slice: v})
			it.Next()
		}
		r.bodyArray = bodyArray
	}

	return r.bodyArray, nil
}

// getSlice reads the slice from the response if needed.
func (r *httpVPackResponse) getSlice() (velocypack.Slice, error) {
	if r.slice == nil {
		r.slice = velocypack.Slice(r.rawResponse)
		//fmt.Println(r.slice)
	}
	return r.slice, nil
}
