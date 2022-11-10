// Package synthetics provides a programmatic API for interacting with the New Relic Synthetics product.
package synthetics

import (
	"strings"

	"github.com/newrelic/newrelic-client-go/internal/http"
	"github.com/newrelic/newrelic-client-go/pkg/config"
	"github.com/newrelic/newrelic-client-go/pkg/logging"
)

// Synthetics is used to communicate with the New Relic Synthetics product.
type Synthetics struct {
	client http.Client
	config config.Config
	logger logging.Logger
	pager  http.Pager
}

// ErrorResponse represents an error response from New Relic Synthetics.
type ErrorResponse struct {
	http.DefaultErrorResponse

	Message            string        `json:"error,omitempty"`
	Messages           []ErrorDetail `json:"errors,omitempty"`
	ServerErrorMessage string        `json:"message,omitempty"`
}

// ErrorDetail represents an single error from New Relic Synthetics.
type ErrorDetail struct {
	Message string `json:"error,omitempty"`
}

// Error surfaces an error message from the New Relic Synthetics error response.
func (e *ErrorResponse) Error() string {
	if e.ServerErrorMessage != "" {
		return e.ServerErrorMessage
	}

	if e.Message != "" {
		return e.Message
	}

	if len(e.Messages) > 0 {
		messages := []string{}
		for _, m := range e.Messages {
			messages = append(messages, m.Message)
		}
		return strings.Join(messages, ", ")
	}

	return ""
}

// New creates a new instance of ErrorResponse.
func (e *ErrorResponse) New() http.ErrorResponse {
	return &ErrorResponse{}
}

// New is used to create a new Synthetics client instance.
func New(config config.Config) Synthetics {
	client := http.NewClient(config)
	client.SetAuthStrategy(&http.PersonalAPIKeyCapableV2Authorizer{})
	client.SetErrorValue(&ErrorResponse{})

	pkg := Synthetics{
		client: client,
		config: config,
		logger: config.GetLogger(),
		pager:  &http.LinkHeaderPager{},
	}

	return pkg
}
