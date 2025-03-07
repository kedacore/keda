package sumologic

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		expectErr bool
	}{
		{
			name: "Valid Config",
			config: &Config{
				Host:      "fake",
				AccessID:  "fake",
				AccessKey: "fake",
			},
		},
		{
			name: "Missing Host",
			config: &Config{
				AccessID:  "fake",
				AccessKey: "fake",
			},
			expectErr: true,
		},
		{
			name: "Missing AccessID",
			config: &Config{
				Host:      "fake",
				AccessKey: "fake",
			},
			expectErr: true,
		},
		{
			name: "Missing AccessKey",
			config: &Config{
				Host:     "fake",
				AccessID: "fake",
			},
			expectErr: true,
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

			if !test.expectErr && err != nil {
				t.Errorf("Expected no error, got %s", err.Error())
			}

			if !test.expectErr && client == nil {
				t.Error("Expected client to be non-nil")
			}
		})
	}
}

func TestGetLogSearchResult(t *testing.T) {
	tests := []struct {
		name              string
		config            *Config
		query             string
		timerange         time.Duration
		tz                string
		expectErr         bool
		createJobResponse SearchJobResponse
		jobStatusResponse SearchJobStatus
		recordsResponse   RecordsResponse
		statusCode        int
	}{
		{
			name: "Successful Log Search",
			config: &Config{
				Host:      "fake",
				AccessID:  "fake",
				AccessKey: "fake",
				UnsafeSsl: true,
			},
			query:     "test query | count as result",
			timerange: 10,
			tz:        "Asia/Kolkata",
			createJobResponse: SearchJobResponse{
				ID: "fake",
			},
			jobStatusResponse: SearchJobStatus{
				State:       "DONE GATHERING RESULTS",
				RecordCount: 1,
			},
			recordsResponse: RecordsResponse{
				Records: []struct {
					Map map[string]string `json:"map"`
				}{
					{
						Map: map[string]string{"result": "189"},
					},
				},
			},
			statusCode: http.StatusOK,
		},
		{
			name: "Failed Log Search",
			config: &Config{
				Host:      "fake",
				AccessID:  "fake",
				AccessKey: "fake",
				UnsafeSsl: true,
			},
			query:     "test query",
			timerange: 10,
			tz:        "UTC",
			createJobResponse: SearchJobResponse{
				ID: "fake",
			},
			jobStatusResponse: SearchJobStatus{
				State:       "CANCELLED",
				RecordCount: 0,
			},
			expectErr:  true,
			statusCode: http.StatusOK,
		},
		{
			name: "Non-Aggregate Query",
			config: &Config{
				Host:      "fake",
				AccessID:  "fake",
				AccessKey: "fake",
				UnsafeSsl: true,
			},
			query:     "test non-agg query",
			timerange: 10,
			tz:        "UTC",
			createJobResponse: SearchJobResponse{
				ID: "fake",
			},
			jobStatusResponse: SearchJobStatus{
				State:       "DONE GATHERING RESULTS",
				RecordCount: 0,
			},
			expectErr:  true,
			statusCode: http.StatusOK,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.statusCode)
				w.Header().Set("Content-Type", "application/json")

				if r.Method == "POST" && r.URL.Path == "/api/v1/search/jobs" {
					err := json.NewEncoder(w).Encode(test.createJobResponse)

					if err != nil {
						http.Error(w, fmt.Sprintf("error building the response, %v", err), http.StatusInternalServerError)
						return
					}
				} else if r.Method == "GET" && r.URL.Path == fmt.Sprintf("/api/v1/search/jobs/%s", test.createJobResponse.ID) {
					err := json.NewEncoder(w).Encode(test.jobStatusResponse)

					if err != nil {
						http.Error(w, fmt.Sprintf("error building the response, %v", err), http.StatusInternalServerError)
						return
					}
				} else if r.Method == "GET" && r.URL.Path == fmt.Sprintf("/api/v1/search/jobs/%s/records", test.createJobResponse.ID) {
					err := json.NewEncoder(w).Encode(test.recordsResponse)

					if err != nil {
						http.Error(w, fmt.Sprintf("error building the response, %v", err), http.StatusInternalServerError)
						return
					}
				} else if r.Method == "DELETE" {
					// do nothing
				} else {
					fmt.Println(r.Method, r.URL.Path)
				}
			}))

			defer server.Close()

			test.config.Host = strings.Replace(server.URL, "https://", "", 1)
			client, err := NewClient(test.config, &scalersconfig.ScalerConfig{
				GlobalHTTPTimeout: 10 * time.Second,
			})
			if err != nil {
				t.Fatalf("Expected no error, got 111 %s", err.Error())
			}

			records, err := client.GetLogSearchResult(test.query, test.timerange, test.tz)

			if test.expectErr && err != nil {
				return
			}

			if test.expectErr && err == nil {
				t.Error("Expected error, got nil")
			}

			if !test.expectErr && err != nil {
				t.Errorf("Expected no error, got %s", err.Error())
			}

			if !test.expectErr && len(records) == 0 {
				t.Error("Expected records to be non-empty")
			}
		})
	}
}
