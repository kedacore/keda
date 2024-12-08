package scalers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	v2 "k8s.io/api/autoscaling/v2"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

func TestParseGitLabRunnerMetadata(t *testing.T) {
	// Create a properly initialized ScalerConfig with valid metadata.
	config := &scalersconfig.ScalerConfig{
		TriggerMetadata: map[string]string{
			"gitlabAPIURL":              "https://gitlab.com",
			"projectID":                 "12345",
			"targetPipelineQueueLength": "5",
		},
		AuthParams: map[string]string{
			"personalAccessToken": "fake-token",
		},
		GlobalHTTPTimeout: 10 * time.Second,
		TriggerIndex:      0,
	}

	// Attempt to parse the metadata.
	meta, err := parseGitLabRunnerMetadata(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Validate the parsed metadata
	if meta.GitLabAPIURL.String() != "https://gitlab.com/api/v4/projects/12345/pipelines?per_page=200&status=waiting_for_resource" {
		t.Errorf("Expected URL to be correctly formed, got %v", meta.GitLabAPIURL.String())
	}

	if meta.ProjectID != "12345" {
		t.Errorf("Expected projectID to be '12345', got %v", meta.ProjectID)
	}

	if meta.TargetPipelineQueueLength != 5 {
		t.Errorf("Expected targetPipelineQueueLength to be 5, got %v", meta.TargetPipelineQueueLength)
	}

	if meta.PersonalAccessToken != "fake-token" {
		t.Errorf("Expected personalAccessToken to be 'fake-token', got %v", meta.PersonalAccessToken)
	}
}

func mustParseURL(rawURL string) *url.URL {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}
	return parsed
}

func TestGitLabRunnerScaler_GetPipelineCount(t *testing.T) {
	testCases := []struct {
		name           string
		responseStatus int
		responseBody   interface{}
		expectedCount  int64
		expectError    bool
	}{
		{
			name:           "Valid response with pipelines",
			responseStatus: http.StatusOK,
			responseBody: []map[string]interface{}{
				{"id": 1},
				{"id": 2},
				{"id": 3},
			},
			expectedCount: 3,
			expectError:   false,
		},
		{
			name:           "Valid response with no pipelines",
			responseStatus: http.StatusOK,
			responseBody:   []map[string]interface{}{},
			expectedCount:  0,
			expectError:    false,
		},
		{
			name:           "Unauthorized response",
			responseStatus: http.StatusUnauthorized,
			responseBody:   map[string]string{"message": "401 Unauthorized"},
			expectedCount:  0,
			expectError:    true,
		},
		{
			name:           "Invalid JSON response",
			responseStatus: http.StatusOK,
			responseBody:   "invalid-json",
			expectedCount:  0,
			expectError:    true,
		},
		{
			name:           "Internal server error",
			responseStatus: http.StatusInternalServerError,
			responseBody:   map[string]string{"message": "500 Internal Server Error"},
			expectedCount:  0,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.responseStatus)
				if err := json.NewEncoder(w).Encode(tc.responseBody); err != nil {
					t.Fatalf("failed to write response: %v", err)
				}
			}))
			defer server.Close()

			meta := &gitlabRunnerMetadata{
				GitLabAPIURL:        mustParseURL(server.URL),
				PersonalAccessToken: "test-token",
			}

			scaler := gitlabRunnerScaler{
				metadata:   meta,
				httpClient: http.DefaultClient,
				logger:     logr.Discard(),
			}

			count, err := scaler.getPipelineCount(context.Background(), server.URL)
			if tc.expectError {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedCount, count)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedCount, count)
			}
		})
	}
}

func TestGitLabRunnerScaler_GetPipelineQueueLength(t *testing.T) {
	totalPipelines := 450 // More than one page
	perPage := 200

	// Create fake pipelines
	createPipelines := func(count int) []map[string]interface{} {
		pipelines := make([]map[string]interface{}, count)
		for i := 0; i < count; i++ {
			pipelines[i] = map[string]interface{}{
				"id": i + 1,
			}
		}
		return pipelines
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pageStr := r.URL.Query().Get("page")
		page, _ := strconv.Atoi(pageStr)
		start := (page - 1) * perPage
		end := start + perPage

		if start >= totalPipelines {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
			return
		}

		if end > totalPipelines {
			end = totalPipelines
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(createPipelines(end - start))
	}))
	defer server.Close()

	meta := &gitlabRunnerMetadata{
		GitLabAPIURL:        mustParseURL(server.URL),
		PersonalAccessToken: "test-token",
	}

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
		logger:     logr.Discard(),
	}

	count, err := scaler.getPipelineQueueLength(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int64(totalPipelines), count)
}

func TestGitLabRunnerScaler_GetMetricsAndActivity(t *testing.T) {
	testCases := []struct {
		name                      string
		pipelineQueueLength       int64
		targetPipelineQueueLength int64
		expectedMetricValue       int64
		expectedActive            bool
		expectError               bool
	}{
		{
			name:                      "Queue length below target",
			pipelineQueueLength:       2,
			targetPipelineQueueLength: 5,
			expectedMetricValue:       2,
			expectedActive:            false,
			expectError:               false,
		},
		{
			name:                      "Queue length equal to target",
			pipelineQueueLength:       5,
			targetPipelineQueueLength: 5,
			expectedMetricValue:       5,
			expectedActive:            true,
			expectError:               false,
		},
		{
			name:                      "Queue length above target",
			pipelineQueueLength:       10,
			targetPipelineQueueLength: 5,
			expectedMetricValue:       10,
			expectedActive:            true,
			expectError:               false,
		},
		{
			name:                      "Error retrieving queue length",
			pipelineQueueLength:       0,
			targetPipelineQueueLength: 5,
			expectedMetricValue:       0,
			expectedActive:            false,
			expectError:               true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.expectError {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				page := r.URL.Query().Get("page")

				w.WriteHeader(http.StatusOK)

				pipelines := make([]map[string]interface{}, 0, tc.pipelineQueueLength)
				if page == "1" {
					for i := int64(0); i < tc.pipelineQueueLength; i++ {
						pipelines = append(pipelines, map[string]interface{}{
							"id": i + 1,
						})
					}
				}

				_ = json.NewEncoder(w).Encode(pipelines)
			}))
			defer server.Close()

			meta := &gitlabRunnerMetadata{
				GitLabAPIURL:              mustParseURL(server.URL),
				PersonalAccessToken:       "test-token",
				TargetPipelineQueueLength: tc.targetPipelineQueueLength,
				ProjectID:                 "12345",
			}

			scaler := gitlabRunnerScaler{
				metadata:   meta,
				httpClient: http.DefaultClient,
				logger:     logr.Discard(),
			}

			metrics, active, err := scaler.GetMetricsAndActivity(context.Background(), "gitlab-runner-queue-length")
			if tc.expectError {
				assert.Error(t, err)
				assert.Empty(t, metrics, "Expected no metrics")
				assert.False(t, active, "Expected not active")
			} else {
				assert.NoError(t, err)
				assert.Len(t, metrics, 1, "Expected one metric")
				assert.Equal(t, float64(tc.expectedMetricValue), metrics[0].Value.AsApproximateFloat64(), "Expected metric value")
				assert.Equal(t, tc.expectedActive, active, "Expected active")
			}
		})
	}
}

func TestGitLabRunnerScaler_GetMetricSpecForScaling(t *testing.T) {
	meta := &gitlabRunnerMetadata{
		ProjectID:                 "12345",
		TargetPipelineQueueLength: 5,
		TriggerIndex:              0,
	}

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		metricType: v2.AverageValueMetricType,
	}

	metricSpecs := scaler.GetMetricSpecForScaling(context.Background())
	assert.Len(t, metricSpecs, 1)

	metricSpec := metricSpecs[0]
	assert.Equal(t, v2.ExternalMetricSourceType, metricSpec.Type)
	assert.Equal(t, "s0-gitlab-runner-12345", metricSpec.External.Metric.Name)
	assert.Equal(t, int64(5), metricSpec.External.Target.AverageValue.Value())
}

func TestGitLabRunnerScaler_Close(t *testing.T) {
	meta := &gitlabRunnerMetadata{}
	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
	}

	err := scaler.Close(context.Background())
	assert.NoError(t, err)
}

func TestConstructGitlabAPIPipelinesURL(t *testing.T) {
	baseURL := mustParseURL("https://gitlab.example.com")
	projectID := "12345"
	status := "waiting_for_resource"

	expectedURL := "https://gitlab.example.com/api/v4/projects/12345/pipelines?per_page=200&status=waiting_for_resource"

	resultURL := constructGitlabAPIPipelinesURL(*baseURL, projectID, status)
	assert.Equal(t, expectedURL, resultURL.String())
}

func TestPagedURL(t *testing.T) {
	baseURL := mustParseURL("https://gitlab.example.com/api/v4/projects/12345/pipelines?per_page=200&status=waiting_for_resource")
	page := "2"

	expectedURL := "https://gitlab.example.com/api/v4/projects/12345/pipelines?page=2&per_page=200&status=waiting_for_resource"

	resultURL := pagedURL(*baseURL, page)
	assert.Equal(t, expectedURL, resultURL.String())
}

func TestGetPipelineCount_RequestError(t *testing.T) {
	meta := &gitlabRunnerMetadata{
		GitLabAPIURL:        mustParseURL("http://invalid-url"),
		PersonalAccessToken: "test-token",
	}

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
		logger:     logr.Discard(),
	}

	_, err := scaler.getPipelineCount(context.Background(), "http://invalid-url")
	assert.Error(t, err)
}

func TestGetPipelineQueueLength_MaxPagesExceeded(t *testing.T) {
	serverCallCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverCallCount++
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{"id": 1},
		})
	}))
	defer server.Close()

	meta := &gitlabRunnerMetadata{
		GitLabAPIURL:        mustParseURL(server.URL),
		PersonalAccessToken: "test-token",
	}

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
		logger:     logr.Discard(),
	}

	count, err := scaler.getPipelineQueueLength(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int64(maxGitlabAPIPageCount), int64(serverCallCount))
	assert.Equal(t, int64(maxGitlabAPIPageCount), count)
}

func TestGetPipelineQueueLength_RequestError(t *testing.T) {
	meta := &gitlabRunnerMetadata{
		GitLabAPIURL:        mustParseURL("http://invalid-url"),
		PersonalAccessToken: "test-token",
	}

	scaler := gitlabRunnerScaler{
		metadata:   meta,
		httpClient: http.DefaultClient,
		logger:     logr.Discard(),
	}

	_, err := scaler.getPipelineQueueLength(context.Background())
	assert.Error(t, err)
}

func TestNewGitLabRunnerScaler_InvalidMetricType(t *testing.T) {
	config := &scalersconfig.ScalerConfig{
		TriggerMetadata: map[string]string{
			"projectID": "12345",
		},
		AuthParams: map[string]string{
			"personalAccessToken": "test-token",
		},
		MetricType: "InvalidType",
	}

	_, err := NewGitLabRunnerScaler(config)
	assert.Error(t, err)
}
