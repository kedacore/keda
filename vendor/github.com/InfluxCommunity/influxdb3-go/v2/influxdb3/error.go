/*
 The MIT License

 Permission is hereby granted, free of charge, to any person obtaining a copy
 of this software and associated documentation files (the "Software"), to deal
 in the Software without restriction, including without limitation the rights
 to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 copies of the Software, and to permit persons to whom the Software is
 furnished to do so, subject to the following conditions:

 The above copyright notice and this permission notice shall be included in
 all copies or substantial portions of the Software.

 THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 THE SOFTWARE.
*/

package influxdb3

import (
	"fmt"
	"net/http"
)

// ServerError represents an error returned from an InfluxDB API server.
type ServerError struct {
	// Code holds the Influx error code, or empty if the code is unknown.
	Code string `json:"code"`
	// Message holds the error message.
	Message string `json:"message"`
	// StatusCode holds the HTTP response status code.
	StatusCode int `json:"-"`
	// RetryAfter holds the value of Retry-After header if sent by server, otherwise zero
	RetryAfter int `json:"-"`
	// Headers hold the response headers
	Headers http.Header `json:"headers"`
}

// NewServerError returns new with just a message
func NewServerError(message string) *ServerError {
	return &ServerError{Message: message}
}

// Error implements Error interface
func (e ServerError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return e.Message
}
