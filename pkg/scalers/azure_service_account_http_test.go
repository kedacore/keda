package scalers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

func TestAzureServiceAccountHTTPClientsRejectCredentialRedirects(t *testing.T) {
	t.Setenv("AZURE_FEDERATED_TOKEN_FILE", "/unused-operator-token")
	cases := []struct {
		name     string
		metadata map[string]string
		auth     map[string]string
		create   func(*scalersconfig.ScalerConfig) (Scaler, error)
		client   func(Scaler) *http.Client
	}{
		{
			name:     "AppInsights",
			metadata: map[string]string{"targetValue": "1", "applicationInsightsId": "app", "metricId": "metric", "metricAggregationTimespan": "01:00", "metricAggregationType": "max", "tenantId": "tenant"},
			create:   NewAzureAppInsightsScaler,
			client:   func(s Scaler) *http.Client { return s.(*azureAppInsightsScaler).httpClient },
		},
		{
			name:     "CosmosDB",
			metadata: map[string]string{"databaseId": "db", "containerId": "container", "leaseDatabaseId": "db", "leaseContainerId": "lease", "processorName": "processor", "endpoint": "https://storage.documents.azure.com"},
			create:   NewAzureCosmosDBScaler,
			client:   func(s Scaler) *http.Client { return s.(*azureCosmosDBScaler).cosmosClient.httpClient },
		},
		{
			name:     "RabbitMQ",
			metadata: map[string]string{"queueName": "queue", "host": "https://rabbitmq.example", "protocol": "http", "mode": "QueueLength", "value": "1"},
			auth:     map[string]string{"workloadIdentityResource": "api://rabbitmq"},
			create:   NewRabbitMQScaler,
			client:   func(s Scaler) *http.Client { return s.(*rabbitMQScaler).httpClient },
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, provider := range []kedav1alpha1.PodIdentityProvider{
				kedav1alpha1.PodIdentityProviderAzureWorkloadServiceAccount,
				kedav1alpha1.PodIdentityProviderAzureWorkload,
			} {
				t.Run(string(provider), func(t *testing.T) {
					config := azureServiceAccountConfig()
					config.PodIdentity.Provider = provider
					if provider == kedav1alpha1.PodIdentityProviderAzureWorkload {
						config.PodIdentity.ServiceAccountName = nil
						config.ServiceAccountTokenProvider = nil
					}
					config.TriggerMetadata, config.AuthParams = tc.metadata, tc.auth
					scaler, err := tc.create(config)
					require.NoError(t, err)
					t.Cleanup(func() { require.NoError(t, scaler.Close(t.Context())) })
					client := tc.client(scaler)
					if provider == kedav1alpha1.PodIdentityProviderAzureWorkload {
						require.Nil(t, client.CheckRedirect)
						return
					}
					for _, target := range []string{
						"https://metrics.example/result",
						"https://other.example/result",
						"https://child.metrics.example/result",
						"https://metrics.example:8443/result",
						"http://metrics.example/result",
					} {
						t.Run(target, func(t *testing.T) {
							requests := 0
							client.Transport = azureServiceAccountRoundTripper(func(request *http.Request) (*http.Response, error) {
								requests++
								require.Equal(t, "Bearer selected-cloud-token", request.Header.Get("Authorization"))
								response := &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("{}")), Request: request}
								if requests == 1 {
									response.StatusCode = http.StatusFound
									response.Header.Set("Location", target)
								}
								return response, nil
							})
							request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://metrics.example/query", nil)
							require.NoError(t, err)
							request.Header.Set("Authorization", "Bearer selected-cloud-token")
							response, err := client.Do(request)
							if target == "https://metrics.example/result" {
								require.NoError(t, err)
								require.NoError(t, response.Body.Close())
								require.Equal(t, 2, requests)
							} else {
								require.Error(t, err)
								require.Equal(t, 1, requests, "redirected endpoint must not receive the credential")
							}
						})
					}
				})
			}
		})
	}
}

func TestAzurePipelinesServiceAccountRejectsRedirectDuringConstruction(t *testing.T) {
	previousTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = previousTransport })
	http.DefaultTransport = azureServiceAccountRoundTripper(func(request *http.Request) (*http.Response, error) {
		if request.URL.Host != "login.microsoftonline.com" {
			return nil, fmt.Errorf("unexpected identity endpoint: %s", request.URL)
		}
		var body string
		switch request.URL.Path {
		case "/22222222-2222-2222-2222-222222222222/v2.0/.well-known/openid-configuration":
			body = `{"token_endpoint":"https://login.microsoftonline.com/22222222-2222-2222-2222-222222222222/oauth2/v2.0/token","issuer":"https://login.microsoftonline.com/22222222-2222-2222-2222-222222222222/v2.0","authorization_endpoint":"https://login.microsoftonline.com/22222222-2222-2222-2222-222222222222/oauth2/v2.0/authorize"}`
		case "/22222222-2222-2222-2222-222222222222/oauth2/v2.0/token":
			body = `{"access_token":"selected-cloud-token","expires_in":3600,"token_type":"Bearer"}`
		default:
			return nil, fmt.Errorf("unexpected identity request: %s", request.URL)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: request}, nil
	})

	for _, sameOrigin := range []bool{true, false} {
		t.Run(fmt.Sprintf("sameOrigin=%t", sameOrigin), func(t *testing.T) {
			receivedToken := atomic.NewString("")
			target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				receivedToken.Store(request.Header.Get("Authorization"))
				_, _ = io.WriteString(w, `{"id":1}`)
			}))
			t.Cleanup(target.Close)
			source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				if request.URL.Path == "/result" {
					receivedToken.Store(request.Header.Get("Authorization"))
					_, _ = io.WriteString(w, `{"id":1}`)
					return
				}
				location := target.URL + "/result"
				if sameOrigin {
					location = "/result"
				}
				http.Redirect(w, request, location, http.StatusFound)
			}))
			t.Cleanup(source.Close)
			config := azureServiceAccountConfig()
			config.ServiceAccountTokenProvider.GetToken = func(context.Context) (string, error) { return "selected-assertion", nil }
			config.TriggerMetadata = map[string]string{"organizationURL": source.URL + "/org", "poolID": "1", "targetPipelinesQueueLength": "1"}
			scaler, err := NewAzurePipelinesScaler(t.Context(), config)
			if sameOrigin {
				require.NoError(t, err)
				require.Equal(t, "Bearer selected-cloud-token", receivedToken.Load())
				require.NoError(t, scaler.Close(t.Context()))
			} else {
				require.Error(t, err)
				require.Empty(t, receivedToken.Load(), "redirected endpoint must not receive the credential")
			}
		})
	}
}
