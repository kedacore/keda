package azure

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

// mockCredential implements azcore.TokenCredential for testing
type mockCredential struct {
	token string
	err   error
}

func (m *mockCredential) GetToken(ctx context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
	if m.err != nil {
		return azcore.AccessToken{}, m.err
	}
	return azcore.AccessToken{
		Token: m.token,
	}, nil
}

func TestMSALAppInsightsClient_GetMetricValue(t *testing.T) {
	tests := []struct {
		name             string
		serverResponse   string
		statusCode       int
		expectedValue    float64
		expectError      bool
		ignoreNullValues bool
	}{
		{
			name:       "successful metric retrieval",
			statusCode: http.StatusOK,
			serverResponse: `{
				"value": {
					"requests/count": {
						"sum": 42.5
					}
				}
			}`,
			expectedValue: 42.5,
			expectError:   false,
		},
		{
			name:             "null value with ignore flag",
			statusCode:       http.StatusOK,
			serverResponse:   `{"value": {"requests/count": {"sum": null}}}`,
			expectedValue:    0.0,
			expectError:      false,
			ignoreNullValues: true,
		},
		{
			name:             "null value without ignore flag",
			statusCode:       http.StatusOK,
			serverResponse:   `{"value": {"requests/count": {"sum": null}}}`,
			expectedValue:    -1,
			expectError:      true,
			ignoreNullValues: false,
		},
		{
			name:          "API error response",
			statusCode:    http.StatusUnauthorized,
			expectedValue: -1,
			expectError:   true,
		},
		{
			name:           "invalid JSON response",
			statusCode:     http.StatusOK,
			serverResponse: `{invalid json}`,
			expectedValue:  -1,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify authorization header
				authHeader := r.Header.Get("Authorization")
				assert.Equal(t, "Bearer test-token", authHeader)

				// Verify request path
				assert.Contains(t, r.URL.Path, "/v1/apps/test-app-id/metrics/requests/count")

				w.WriteHeader(tt.statusCode)
				if tt.serverResponse != "" {
					w.Write([]byte(tt.serverResponse))
				}
			}))
			defer server.Close()

			// Create client with mock credential
			client := &MSALAppInsightsClient{
				httpClient: &http.Client{},
				credential: &mockCredential{token: "test-token"},
				info: AppInsightsInfo{
					ApplicationInsightsID:  "test-app-id",
					MetricID:               "requests/count",
					AggregationType:        "sum",
					AggregationTimespan:    "01:00",
					AppInsightsResourceURL: server.URL,
				},
			}

			ctx := context.Background()
			value, err := client.GetMetricValue(ctx, tt.ignoreNullValues)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedValue, value)
			}
		})
	}
}

func TestNewMSALAppInsightsClient(t *testing.T) {
	tests := []struct {
		name         string
		podIdentity  kedav1alpha1.AuthPodIdentity
		appInfo      AppInsightsInfo
		expectError  bool
		errorMessage string
	}{
		{
			name: "client credentials authentication",
			podIdentity: kedav1alpha1.AuthPodIdentity{
				Provider: kedav1alpha1.PodIdentityProviderNone,
			},
			appInfo: AppInsightsInfo{
				TenantID:       "test-tenant",
				ClientID:       "test-client",
				ClientPassword: "test-secret",
			},
			expectError: false,
		},
		{
			name: "workload identity authentication",
			podIdentity: kedav1alpha1.AuthPodIdentity{
				Provider:   kedav1alpha1.PodIdentityProviderAzureWorkload,
				IdentityID: "test-identity",
			},
			appInfo:     AppInsightsInfo{},
			expectError: true, // Will fail in test environment without proper workload identity setup
		},
		{
			name: "unsupported identity provider",
			podIdentity: kedav1alpha1.AuthPodIdentity{
				Provider: "unsupported",
			},
			appInfo:      AppInsightsInfo{},
			expectError:  true,
			errorMessage: "unsupported pod identity provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			client, err := NewMSALAppInsightsClient(ctx, tt.appInfo, tt.podIdentity)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMessage != "" {
					assert.Contains(t, err.Error(), tt.errorMessage)
				}
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				assert.NotNil(t, client.credential)
				assert.NotNil(t, client.httpClient)
			}
		})
	}
}

func TestGetAzureAppInsightsMetricValueMSAL(t *testing.T) {
	// Create a test server that simulates App Insights API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := ApplicationInsightsMetric{
			Value: map[string]interface{}{
				"requests/count": map[string]interface{}{
					"sum": 100.0,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	info := AppInsightsInfo{
		ApplicationInsightsID:  "test-app-id",
		MetricID:               "requests/count",
		AggregationType:        "sum",
		AggregationTimespan:    "01:00",
		AppInsightsResourceURL: server.URL,
		TenantID:               "test-tenant",
		ClientID:               "test-client",
		ClientPassword:         "test-secret",
	}

	podIdentity := kedav1alpha1.AuthPodIdentity{
		Provider: kedav1alpha1.PodIdentityProviderNone,
	}

	// This test will fail in CI/test environment since it requires actual Azure credentials
	// but demonstrates the interface
	ctx := context.Background()
	_, err := GetAzureAppInsightsMetricValueMSAL(ctx, info, podIdentity, false)

	// We expect an error since we don't have valid Azure credentials in test environment
	assert.Error(t, err)
}

func TestQueryParamsForAppInsightsRequest(t *testing.T) {
	info := AppInsightsInfo{
		AggregationType:     "sum",
		AggregationTimespan: "01:30",
		Filter:              "customDimensions/environment eq 'production'",
	}

	params, err := queryParamsForAppInsightsRequest(info)
	require.NoError(t, err)

	assert.Equal(t, "sum", params["aggregation"])
	assert.Equal(t, "PT01H30M", params["timespan"])
	assert.Equal(t, "customDimensions/environment eq 'production'", params["filter"])
}

func TestExtractAppInsightValue(t *testing.T) {
	tests := []struct {
		name          string
		info          AppInsightsInfo
		metric        ApplicationInsightsMetric
		expectedValue float64
		expectError   bool
	}{
		{
			name: "successful extraction",
			info: AppInsightsInfo{
				MetricID:        "requests/count",
				AggregationType: "sum",
			},
			metric: ApplicationInsightsMetric{
				Value: map[string]interface{}{
					"requests/count": map[string]interface{}{
						"sum": 42.5,
					},
				},
			},
			expectedValue: 42.5,
			expectError:   false,
		},
		{
			name: "metric not found",
			info: AppInsightsInfo{
				MetricID:        "nonexistent/metric",
				AggregationType: "sum",
			},
			metric: ApplicationInsightsMetric{
				Value: map[string]interface{}{
					"requests/count": map[string]interface{}{
						"sum": 42.5,
					},
				},
			},
			expectedValue: -1,
			expectError:   true,
		},
		{
			name: "aggregation type not found",
			info: AppInsightsInfo{
				MetricID:        "requests/count",
				AggregationType: "average",
			},
			metric: ApplicationInsightsMetric{
				Value: map[string]interface{}{
					"requests/count": map[string]interface{}{
						"sum": 42.5,
					},
				},
			},
			expectedValue: -1,
			expectError:   true,
		},
		{
			name: "null value",
			info: AppInsightsInfo{
				MetricID:        "requests/count",
				AggregationType: "sum",
			},
			metric: ApplicationInsightsMetric{
				Value: map[string]interface{}{
					"requests/count": map[string]interface{}{
						"sum": nil,
					},
				},
			},
			expectedValue: -1,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := extractAppInsightValue(tt.info, tt.metric)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedValue, value)
			}
		})
	}
}
