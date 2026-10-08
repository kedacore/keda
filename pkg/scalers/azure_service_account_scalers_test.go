package scalers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type azureServiceAccountRoundTripper func(*http.Request) (*http.Response, error)

func (f azureServiceAccountRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func azureServiceAccountConfig() *scalersconfig.ScalerConfig {
	return &scalersconfig.ScalerConfig{
		PodIdentity: kedav1alpha1.AuthPodIdentity{
			Provider:           kedav1alpha1.PodIdentityProviderAzureWorkload,
			ServiceAccountName: new("reader"),
			IdentityID:         new("11111111-1111-1111-1111-111111111111"),
			IdentityTenantID:   new("22222222-2222-2222-2222-222222222222"),
		},
		ServiceAccountTokenProvider: &scalersconfig.ServiceAccountTokenProvider{
			Namespace: "tenant", ServiceAccountName: "reader", Audience: "api://AzureADTokenExchange",
			GetToken: func(context.Context) (string, error) {
				return "", errors.New("selected service account token denied")
			},
		},
		GlobalHTTPTimeout: time.Second,
	}
}

func TestAzureServiceAccountClientsDoNotUseOperatorCredentials(t *testing.T) {
	t.Setenv("AZURE_FEDERATED_TOKEN_FILE", filepath.Join(t.TempDir(), "missing-operator-token"))
	t.Setenv("AZURE_CLIENT_ID", "operator-client")
	t.Setenv("AZURE_TENANT_ID", "operator-tenant")
	t.Setenv("AZURE_AUTHORITY_HOST", "https://login.microsoftonline.com/")

	cases := []struct {
		name     string
		metadata map[string]string
		auth     map[string]string
		create   func(*scalersconfig.ScalerConfig) (Scaler, error)
	}{
		{"Blob", map[string]string{"blobContainerName": "container", "accountName": "storage"}, nil, NewAzureBlobScaler},
		{"Queue", map[string]string{"queueName": "queue", "accountName": "storage"}, nil, NewAzureQueueScaler},
		{"EventHub", map[string]string{"eventHubNamespace": "namespace", "eventHubName": "hub", "storageAccountName": "storage"}, nil, NewAzureEventHubScaler},
		{"ServiceBus", map[string]string{"queueName": "queue", "namespace": "namespace"}, nil, func(config *scalersconfig.ScalerConfig) (Scaler, error) {
			scaler, err := NewAzureServiceBusScaler(config)
			if err == nil {
				_, err = scaler.(*azureServiceBusScaler).getServiceBusAdminClient()
			}
			return scaler, err
		}},
		{"Monitor", map[string]string{"resourceURI": "namespace/type/resource", "targetValue": "1", "tenantId": "tenant", "subscriptionId": "subscription", "resourceGroupName": "group", "metricName": "metric", "metricAggregationType": "Average"}, nil, NewAzureMonitorScaler},
		{"LogAnalytics", map[string]string{"workspaceId": "workspace", "query": "query", "threshold": "1"}, nil, NewAzureLogAnalyticsScaler},
		{"AppInsights", map[string]string{"targetValue": "1", "applicationInsightsId": "app", "metricId": "metric", "metricAggregationTimespan": "01:00", "metricAggregationType": "max", "tenantId": "tenant"}, nil, NewAzureAppInsightsScaler},
		{"CosmosDB", map[string]string{"databaseId": "db", "containerId": "container", "leaseDatabaseId": "db", "leaseContainerId": "lease", "processorName": "processor", "endpoint": "https://storage.documents.azure.com"}, nil, NewAzureCosmosDBScaler},
		{"Prometheus", map[string]string{"serverAddress": "https://workspace.prometheus.monitor.azure.com", "query": "up", "threshold": "1"}, nil, NewPrometheusScaler},
		{"RabbitMQ", map[string]string{"queueName": "queue", "host": "https://rabbitmq.example", "protocol": "http", "mode": "QueueLength", "value": "1"}, map[string]string{"workloadIdentityResource": "api://rabbitmq"}, NewRabbitMQScaler},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config := azureServiceAccountConfig()
			config.TriggerMetadata, config.AuthParams = tc.metadata, tc.auth
			scaler, err := tc.create(config)
			require.NoError(t, err)
			require.NoError(t, scaler.Close(t.Context()))

			config.ServiceAccountTokenProvider = nil
			_, err = tc.create(config)
			require.ErrorContains(t, err, "service account token provider")
		})
	}
}

func TestAzureServiceAccountLegacyCallersUseSelectedToken(t *testing.T) {
	t.Setenv("AZURE_FEDERATED_TOKEN_FILE", filepath.Join(t.TempDir(), "missing-operator-token"))
	t.Setenv("AZURE_AUTHORITY_HOST", "https://login.microsoftonline.com/")
	previousTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = previousTransport })
	http.DefaultTransport = azureServiceAccountRoundTripper(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodGet || request.URL.Host != "login.microsoftonline.com" || request.URL.Path != "/22222222-2222-2222-2222-222222222222/v2.0/.well-known/openid-configuration" {
			return nil, fmt.Errorf("unexpected Azure request: %s %s", request.Method, request.URL)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"token_endpoint":"https://login.microsoftonline.com/22222222-2222-2222-2222-222222222222/oauth2/v2.0/token","issuer":"https://login.microsoftonline.com/22222222-2222-2222-2222-222222222222/v2.0","authorization_endpoint":"https://login.microsoftonline.com/22222222-2222-2222-2222-222222222222/oauth2/v2.0/authorize"}`)),
			Request:    request,
		}, nil
	})

	t.Run("Pipelines", func(t *testing.T) {
		config := azureServiceAccountConfig()
		meta := &azurePipelinesMetadata{}
		require.NoError(t, configureAzurePipelinesAuth(logr.Discard(), config, meta))
		_, err := meta.authContext.credential.GetToken(t.Context(), policy.TokenRequestOptions{Scopes: []string{devopsResource}})
		require.ErrorContains(t, err, "selected service account token denied")
	})
	t.Run("MSSQL", func(t *testing.T) {
		config := azureServiceAccountConfig()
		config.TriggerMetadata = map[string]string{"host": "server", "query": "SELECT 1", "targetValue": "1"}
		meta, _, err := parseMSSQLMetadata(logr.Discard(), config)
		require.NoError(t, err)
		_, err = getMSSQLAzureAccessToken(t.Context(), meta, azureDatabaseMSSQLResource)
		require.ErrorContains(t, err, "selected service account token denied")
	})
	t.Run("PostgreSQL", func(t *testing.T) {
		config := azureServiceAccountConfig()
		config.TriggerMetadata = map[string]string{"host": "server", "port": "5432", "userName": "reader", "dbName": "db", "sslmode": "require", "query": "SELECT 1", "targetQueryValue": "1"}
		meta, _, err := parsePostgreSQLMetadata(logr.Discard(), config)
		require.NoError(t, err)
		_, err = meta.azureAuthContext.cred.GetToken(t.Context(), policy.TokenRequestOptions{Scopes: []string{"https://ossrdbms-aad.database.windows.net/.default"}})
		require.ErrorContains(t, err, "selected service account token denied")
	})
	t.Run("DataExplorer", func(t *testing.T) {
		config := azureServiceAccountConfig()
		config.TriggerMetadata = map[string]string{"databaseName": "db", "endpoint": "https://cluster.kusto.windows.net", "query": "query", "threshold": "1"}
		meta, err := parseAzureDataExplorerMetadata(config, logr.Discard())
		require.NoError(t, err)
		require.Same(t, config.ServiceAccountTokenProvider, meta.ServiceAccountTokenProvider)
	})
	t.Run("RabbitMQ", func(t *testing.T) {
		config := azureServiceAccountConfig()
		config.TriggerMetadata = map[string]string{"queueName": "queue", "host": "https://rabbitmq.example", "protocol": "http", "mode": "QueueLength", "value": "1"}
		config.AuthParams = map[string]string{"workloadIdentityResource": "api://rabbitmq"}
		scaler, err := NewRabbitMQScaler(config)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, scaler.Close(t.Context())) })
		_, err = getJSON(t.Context(), scaler.(*rabbitMQScaler), "https://rabbitmq.example/api/queues")
		require.ErrorContains(t, err, "selected service account token denied")
	})
}

func TestAzureServiceAccountRejectsConflictingAuthentication(t *testing.T) {
	t.Run("Pipelines PAT", func(t *testing.T) {
		err := configureAzurePipelinesAuth(logr.Discard(), azureServiceAccountConfig(), &azurePipelinesMetadata{PersonalAccessToken: "pat"})
		require.ErrorContains(t, err, "personalAccessToken cannot be combined")
	})
	t.Run("CosmosDB key", func(t *testing.T) {
		config := azureServiceAccountConfig()
		_, err := newCosmosDBClient(&azureCosmosDBMetadata{CosmosDBKey: "key"}, nil, config.PodIdentity, logr.Discard(), time.Second, config.ServiceAccountTokenProvider)
		require.ErrorContains(t, err, "account keys cannot be combined")
	})
	t.Run("Prometheus basic auth", func(t *testing.T) {
		config := azureServiceAccountConfig()
		config.TriggerMetadata = map[string]string{"serverAddress": "https://prometheus.example", "query": "up", "threshold": "1"}
		config.AuthParams = map[string]string{"authModes": "basic", "username": "user", "password": "password"}
		_, err := NewPrometheusScaler(config)
		require.ErrorContains(t, err, "pod identity cannot be enabled with other auth types")
	})
	t.Run("RabbitMQ missing resource", func(t *testing.T) {
		config := azureServiceAccountConfig()
		config.TriggerMetadata = map[string]string{"queueName": "queue", "host": "https://rabbitmq.example", "protocol": "http", "mode": "QueueLength", "value": "1"}
		_, err := NewRabbitMQScaler(config)
		require.ErrorContains(t, err, "requires Azure workload identity and workloadIdentityResource")
	})
}
