package splunk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		expectErr bool
	}{
		{
			name: "Valid Basic Auth Config",
			config: &Config{
				Username: "fake",
				Password: "fake",
			},
		},
		{
			name: "Valid Bearer + Username Auth Config",
			config: &Config{
				APIToken: "fake",
				Username: "fake",
			},
		},
		{
			name:      "Missing username",
			config:    &Config{},
			expectErr: true,
		},
		{
			name: "Invalid Bearer + Password Auth Config",
			config: &Config{
				APIToken: "fake",
				Password: "fake",
			},
			expectErr: true,
		},
		{
			name: "UnsafeSsl config",
			config: &Config{
				APIToken:  "fake",
				Username:  "fake",
				UnsafeSsl: false,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client, err := NewClient(test.config, &scalersconfig.ScalerConfig{})

			if test.expectErr && err != nil {
				return
			}

			if test.expectErr && err == nil {
				t.Error("Expected error, got nil")
			}

			if test.config.UnsafeSsl && client.Client.Transport == nil {
				t.Error("Expected SSL client config to be set, but was nil")
			}
		})
	}
}

func TestSavedSearch(t *testing.T) {
	tests := []struct {
		name            string
		config          *Config
		expectErr       bool
		metricValue     string
		valueField      string
		response        SearchResponse
		savedSearchName string
		statusCode      int
	}{
		{
			name: "Count - 1",
			config: &Config{
				Username: "admin",
				Password: "password",
			},
			metricValue:     "1",
			valueField:      "count",
			response:        SearchResponse{Result: map[string]string{"count": "1"}},
			savedSearchName: "testsearch1",
			statusCode:      http.StatusOK,
		},
		{
			name: "Count - 100",
			config: &Config{
				Username: "admin2",
				Password: "password2",
			},
			metricValue:     "100",
			valueField:      "count",
			response:        SearchResponse{Result: map[string]string{"count": "100"}},
			savedSearchName: "testsearch2",
			statusCode:      http.StatusOK,
		},
		{
			name: "StatusBadRequest",
			config: &Config{
				Username: "admin",
				Password: "password",
			},
			expectErr:       true,
			response:        SearchResponse{Result: map[string]string{}},
			savedSearchName: "testsearch4",
			statusCode:      http.StatusBadRequest,
		},
		{
			name: "StatusForbidden",
			config: &Config{
				Username: "admin",
				Password: "password",
			},
			expectErr:       true,
			response:        SearchResponse{Result: map[string]string{}},
			savedSearchName: "testsearch5",
			statusCode:      http.StatusForbidden,
		},
		{
			name: "Validate Bearer Token",
			config: &Config{
				APIToken: "sometoken",
				Username: "fake",
			},
			expectErr:       true,
			response:        SearchResponse{Result: map[string]string{}},
			savedSearchName: "testsearch5",
			statusCode:      http.StatusForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedReqPath := fmt.Sprintf(savedSearchPathTemplateStr, test.config.Username)
				if r.URL.Path != fmt.Sprintf(savedSearchPathTemplateStr, test.config.Username) {
					t.Errorf("Expected request path '%s', got: %s", expectedReqPath, r.URL.Path)
				}

				err := r.ParseForm()
				if err != nil {
					t.Errorf("Expected no error parsing form data, but got '%s'", err.Error())
				}

				searchFormData := r.FormValue("search")
				if searchFormData != fmt.Sprintf("savedsearch %s", test.savedSearchName) {
					t.Errorf("Expected form data to be 'savedsearch %s' '%s'", test.savedSearchName, searchFormData)
				}

				q, err := url.ParseQuery(r.URL.RawQuery)
				if err != nil {
					t.Errorf("Expected query parsing err to be nil, got %s", err.Error())
				}

				outputMode := q.Get("output_mode")
				if outputMode != "json" {
					t.Errorf("Expected output_mode query string to be '%s', got: %s", "json", outputMode)
				}

				// Bearer token auth
				if test.config.APIToken != "" {
					actual := r.Header.Get("Authorization")
					expected := fmt.Sprintf("Bearer %s", test.config.APIToken)
					if actual != expected {
						t.Errorf("APIToken is set. Expected Authorization header to be '%s', got: %s", actual, expected)
					}
				} else {
					// Basic auth
					reqUsername, reqPassword, ok := r.BasicAuth()
					if !ok {
						t.Error("Expected basic auth to be set, but was not")
					}
					if test.config.Username != reqUsername {
						t.Errorf("Expected request username to be '%s', got: %s", test.config.Username, reqUsername)
					}
					if test.config.Password != reqPassword {
						t.Errorf("Expected request password to be '%s', got: %s", test.config.Password, reqPassword)
					}
				}

				w.WriteHeader(test.statusCode)
				w.Header().Set("Content-Type", "application/json")
				err = json.NewEncoder(w).Encode(test.response)
				if err != nil {
					http.Error(w, fmt.Sprintf("error building the response, %v", err), http.StatusInternalServerError)
					return
				}
			}))
			defer server.Close()

			test.config.Host = server.URL
			s, err := NewClient(test.config, &scalersconfig.ScalerConfig{})
			if err != nil {
				t.Errorf("Expected err to be nil, got %s", err.Error())
			}

			splunkResponse, err := s.SavedSearch(test.savedSearchName)

			if test.expectErr && err != nil {
				return
			}

			if test.expectErr && err == nil {
				t.Error("Expected error, got nil")
			}

			if err != nil {
				t.Errorf("Expected err to be nil, got %s", err.Error())
			}

			v, ok := splunkResponse.Result[test.valueField]
			if !ok {
				t.Errorf("Expected value field to be %s to exist but did not", test.valueField)
			}

			if v != test.metricValue {
				t.Errorf("Expected metric value to be %s, got %s", test.metricValue, v)
			}
		})
	}
}

func TestToMetric(t *testing.T) {
	tests := []struct {
		name                string
		expectErr           bool
		expectedMetricValue float64
		response            *SearchResponse
		valueField          string
	}{
		{
			name:                "Successful metric conversion - 1",
			expectedMetricValue: 1.000000,
			response: &SearchResponse{
				Result: map[string]string{
					"count": "1",
				},
			},
			valueField: "count",
		},
		{
			name:                "Successful metric conversion - 100",
			expectedMetricValue: 100.000000,
			response: &SearchResponse{
				Result: map[string]string{
					"count": "100",
				},
			},
			valueField: "count",
		},
		{
			name:      "Failed metric type conversion",
			expectErr: true,
			response: &SearchResponse{
				Result: map[string]string{
					"count": "A",
				},
			},
			valueField: "count",
		},
		{
			name:      "Value field not found",
			expectErr: true,
			response: &SearchResponse{
				Result: map[string]string{
					"fake": "1",
				},
			},
			valueField: "count",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			metric, err := test.response.ToMetric(test.valueField)
			if test.expectErr && err != nil {
				return
			}

			if test.expectErr && err == nil {
				t.Error("Expected error, got nil")
			}

			if test.expectedMetricValue != metric {
				t.Errorf("Expected metric value '%f', got: %f", test.expectedMetricValue, metric)
			}
		})
	}
}
