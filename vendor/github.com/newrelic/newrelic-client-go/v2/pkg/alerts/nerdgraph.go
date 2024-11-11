package alerts

import (
	"context"
	"strings"

	"github.com/newrelic/newrelic-client-go/v2/internal/http"
)

// NerdGraphQueryWithContext works similarly to the default client's NerdGraphQueryWithContext but with a custom error
// value that provides enhanced error messages and context via GraphQLErrorResponse.
func (a *Alerts) NerdGraphQueryWithContext(ctx context.Context, query string, vars map[string]interface{}, respBody interface{}) error {
	req, err := a.client.NewNerdGraphRequest(query, vars, respBody)
	if err != nil {
		return err
	}

	req.WithContext(ctx)
	req.SetErrorValue(&GraphQLErrorResponse{})

	_, err = a.client.Do(req)
	if err != nil {
		return err
	}

	return nil
}

// GraphQLErrorResponse is a special GraphQL response produced by Alerts GraphQL Service and provides additional context
type GraphQLErrorResponse struct {
	http.GraphQLErrorResponse

	Errors []struct {
		Message    string   `json:"message"`
		Path       []string `json:"path"`
		Extensions struct {
			Code             string `json:"code"`
			ErrorClass       string `json:"errorClass"`
			ErrorCode        string `json:"error_code,omitempty"`
			ValidationErrors []struct {
				Name   string `json:"name"`
				Reason string `json:"reason"`
			} `json:"validationErrors"`
		} `json:"extensions"`
	} `json:"errors"`
}

func (r *GraphQLErrorResponse) Error() string {
	if len(r.Errors) > 0 {
		var messages []string

		for _, e := range r.Errors {
			if e.Message != "" {
				messages = append(messages, e.Message)
			}
		}
		return strings.Join(messages, ", ")
	}

	return ""
}

func (r *GraphQLErrorResponse) IsNotFound() bool {
	for _, err := range r.Errors {
		if strings.Contains(err.Message, "Not Found") {
			return true
		}
	}

	return false
}

func (r *GraphQLErrorResponse) New() http.ErrorResponse {
	return &GraphQLErrorResponse{}
}
