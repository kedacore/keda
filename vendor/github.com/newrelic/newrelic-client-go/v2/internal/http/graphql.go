package http

import (
	"net/http"
	"strings"
)

type graphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

type graphQLResponse struct {
	Data interface{} `json:"data"`
}

// GraphQLError represents a single error.
type GraphQLError struct {
	Message    string   `json:"message,omitempty"`
	Path       []string `json:"path,omitempty"`
	Extensions struct {
		ErrorClass string `json:"errorClass,omitempty"`
		ErrorCode  string `json:"error_code,omitempty"`
		Code       string `json:"code,omitempty"`
	} `json:"extensions,omitempty"`
}

// GraphQLErrorResponse represents a default error response body.
type GraphQLErrorResponse struct {
	Errors []GraphQLError `json:"errors"`
}

func (r *GraphQLErrorResponse) Error() string {
	if len(r.Errors) > 0 {
		messages := []string{}
		for _, e := range r.Errors {
			if e.Message != "" {
				messages = append(messages, e.Message)
			}
		}
		return strings.Join(messages, ", ")
	}

	return ""
}

// IsNotFound determines if the error is due to a missing resource.
func (r *GraphQLErrorResponse) IsNotFound() bool {
	return false
}

// IsRetryableError determines if the error is due to a server timeout, or another error that we might want to retry.
func (r *GraphQLErrorResponse) IsRetryableError() bool {
	if len(r.Errors) == 0 {
		return false
	}

	for _, err := range r.Errors {
		errorClass := err.Extensions.ErrorClass
		if errorClass == "TIMEOUT" || errorClass == "INTERNAL_SERVER_ERROR" || errorClass == "SERVER_ERROR" {
			return true
		}
	}

	return false
}

// IsUnauthorized checks a NerdGraph response for a 401 Unauthorize HTTP status code,
// then falls back to check the nested extensions error_code field for `BAD_API_KEY`.
func (r *GraphQLErrorResponse) IsUnauthorized(resp *http.Response) bool {
	if len(r.Errors) == 0 {
		return false
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return true
	}

	// Handle invalid or missing API key
	for _, err := range r.Errors {
		if err.Extensions.ErrorCode == "BAD_API_KEY" {
			return true
		}
	}

	return false
}

func (r *GraphQLErrorResponse) IsPaymentRequired(resp *http.Response) bool {
	return resp.StatusCode == http.StatusPaymentRequired
}

// IsDeprecated parses error messages for warnings that a field being used
// is deprecated.  We want to bubble that up, but not stop returning data
//
// Example deprecation message:
//
//	This field is deprecated! Please use `relatedEntities` instead.
func (r *GraphQLErrorResponse) IsDeprecated() bool {
	for _, err := range r.Errors {
		if strings.HasPrefix(err.Message, "This field is deprecated!") {
			return true
		}
	}

	return false
}

// New creates a new instance of GraphQLErrorRepsonse.
func (r *GraphQLErrorResponse) New() ErrorResponse {
	return &GraphQLErrorResponse{}
}
