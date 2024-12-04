package http

import (
	"fmt"
	"net/http"
	"strings"
)

// ErrorResponse provides an interface for obtaining
// a single error message from an error response object.
type ErrorResponse interface {
	IsNotFound() bool
	IsRetryableError() bool
	IsUnauthorized(resp *http.Response) bool
	IsPaymentRequired(resp *http.Response) bool
	IsDeprecated() bool
	Error() string
	New() ErrorResponse
}

// DefaultErrorResponse represents the default error response from New Relic.
type DefaultErrorResponse struct {
	ErrorDetail ErrorDetail `json:"error"`
}

// ErrorDetail represents a New Relic response error detail.
type ErrorDetail struct {
	Title    string   `json:"title"`
	Messages []string `json:"messages"`
}

func (e *DefaultErrorResponse) Error() string {
	m := e.ErrorDetail.Title
	if len(e.ErrorDetail.Messages) > 0 {
		m = fmt.Sprintf("%s: %s", m, strings.Join(e.ErrorDetail.Messages, ", "))
	}

	return m
}

func (e *DefaultErrorResponse) IsNotFound() bool {
	return false
}

func (e *DefaultErrorResponse) IsRetryableError() bool {
	return false
}

func (e *DefaultErrorResponse) IsDeprecated() bool {
	return false
}

func (e *DefaultErrorResponse) IsPaymentRequired(resp *http.Response) bool {
	return resp.StatusCode == http.StatusPaymentRequired
}

// IsUnauthorized checks a response for a 401 Unauthorize HTTP status code.
func (e *DefaultErrorResponse) IsUnauthorized(resp *http.Response) bool {
	return resp.StatusCode == http.StatusUnauthorized
}

// New creates a new instance of the DefaultErrorResponse struct.
func (e *DefaultErrorResponse) New() ErrorResponse {
	return &DefaultErrorResponse{}
}
