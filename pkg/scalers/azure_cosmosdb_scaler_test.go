package scalers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v2 "k8s.io/api/autoscaling/v2"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

var testCosmosDBResolvedEnv = map[string]string{
	"COSMOS_CONNECTION": "AccountEndpoint=https://test.documents.azure.com:443/;AccountKey=dGVzdGtleQ==",
}

type recordingTokenCredential struct {
	scopes []string
	token  string
}

func (c *recordingTokenCredential) GetToken(_ context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
	c.scopes = append(c.scopes, options.Scopes...)
	return azcore.AccessToken{
		Token:     c.token,
		ExpiresOn: time.Now().Add(time.Hour),
	}, nil
}

type parseCosmosDBMetadataTestData struct {
	name        string
	metadata    map[string]string
	isError     bool
	resolvedEnv map[string]string
	authParams  map[string]string
	podIdentity kedav1alpha1.PodIdentityProvider
}

type cosmosDBMetricIdentifier struct {
	name             string
	metadataTestData *parseCosmosDBMetadataTestData
	triggerIndex     int
	metricName       string
}

var testCosmosDBMetadata = []parseCosmosDBMetadataTestData{
	{
		name:        "nothing passed",
		metadata:    map[string]string{},
		isError:     true,
		resolvedEnv: testCosmosDBResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: "",
	},
	{
		name: "properly formed with connection string",
		metadata: map[string]string{
			"connectionFromEnv": "COSMOS_CONNECTION",
			"databaseId":        "testdb",
			"containerId":       "testcontainer",
			"leaseDatabaseId":   "testdb",
			"leaseContainerId":  "leases",
			"processorName":     "testprocessor",
		},
		isError:     false,
		resolvedEnv: testCosmosDBResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: "",
	},
	{
		name: "missing database id",
		metadata: map[string]string{
			"connectionFromEnv": "COSMOS_CONNECTION",
			"containerId":       "testcontainer",
			"leaseDatabaseId":   "testdb",
			"leaseContainerId":  "leases",
			"processorName":     "testprocessor",
		},
		isError:     true,
		resolvedEnv: testCosmosDBResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: "",
	},
	{
		name: "missing container id",
		metadata: map[string]string{
			"connectionFromEnv": "COSMOS_CONNECTION",
			"databaseId":        "testdb",
			"leaseDatabaseId":   "testdb",
			"leaseContainerId":  "leases",
			"processorName":     "testprocessor",
		},
		isError:     true,
		resolvedEnv: testCosmosDBResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: "",
	},
	{
		name: "missing lease database id",
		metadata: map[string]string{
			"connectionFromEnv": "COSMOS_CONNECTION",
			"databaseId":        "testdb",
			"containerId":       "testcontainer",
			"leaseContainerId":  "leases",
			"processorName":     "testprocessor",
		},
		isError:     true,
		resolvedEnv: testCosmosDBResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: "",
	},
	{
		name: "missing lease container id",
		metadata: map[string]string{
			"connectionFromEnv": "COSMOS_CONNECTION",
			"databaseId":        "testdb",
			"containerId":       "testcontainer",
			"leaseDatabaseId":   "testdb",
			"processorName":     "testprocessor",
		},
		isError:     true,
		resolvedEnv: testCosmosDBResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: "",
	},
	{
		name: "missing processor name",
		metadata: map[string]string{
			"connectionFromEnv": "COSMOS_CONNECTION",
			"databaseId":        "testdb",
			"containerId":       "testcontainer",
			"leaseDatabaseId":   "testdb",
			"leaseContainerId":  "leases",
		},
		isError:     true,
		resolvedEnv: testCosmosDBResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: "",
	},
	{
		name: "missing connection and key",
		metadata: map[string]string{
			"databaseId":       "testdb",
			"containerId":      "testcontainer",
			"leaseDatabaseId":  "testdb",
			"leaseContainerId": "leases",
			"processorName":    "testprocessor",
		},
		isError:     true,
		resolvedEnv: map[string]string{},
		authParams:  map[string]string{},
		podIdentity: "",
	},
	{
		name: "connection from authParams",
		metadata: map[string]string{
			"databaseId":       "testdb",
			"containerId":      "testcontainer",
			"leaseDatabaseId":  "testdb",
			"leaseContainerId": "leases",
			"processorName":    "testprocessor",
		},
		isError:     false,
		resolvedEnv: map[string]string{},
		authParams: map[string]string{
			"connection": "AccountEndpoint=https://test.documents.azure.com:443/;AccountKey=dGVzdGtleQ==",
		},
		podIdentity: kedav1alpha1.PodIdentityProviderNone,
	},
	{
		name: "endpoint with key",
		metadata: map[string]string{
			"endpoint":         "https://test.documents.azure.com:443/",
			"databaseId":       "testdb",
			"containerId":      "testcontainer",
			"leaseDatabaseId":  "testdb",
			"leaseContainerId": "leases",
			"processorName":    "testprocessor",
		},
		isError:     false,
		resolvedEnv: map[string]string{},
		authParams: map[string]string{
			"cosmosDBKey": "dGVzdGtleQ==",
		},
		podIdentity: "",
	},
	{
		name: "podIdentity azure-workload with endpoint",
		metadata: map[string]string{
			"endpoint":         "https://test.documents.azure.com:443/",
			"databaseId":       "testdb",
			"containerId":      "testcontainer",
			"leaseDatabaseId":  "testdb",
			"leaseContainerId": "leases",
			"processorName":    "testprocessor",
		},
		isError:     false,
		resolvedEnv: map[string]string{},
		authParams: map[string]string{
			"cosmosDBKey": "dGVzdGtleQ==",
		},
		podIdentity: kedav1alpha1.PodIdentityProviderAzureWorkload,
	},
	{
		name: "podIdentity azure-workload without endpoint or connection",
		metadata: map[string]string{
			"databaseId":       "testdb",
			"containerId":      "testcontainer",
			"leaseDatabaseId":  "testdb",
			"leaseContainerId": "leases",
			"processorName":    "testprocessor",
		},
		isError:     true,
		resolvedEnv: map[string]string{},
		authParams:  map[string]string{},
		podIdentity: kedav1alpha1.PodIdentityProviderAzureWorkload,
	},
	{
		name: "separate lease connection",
		metadata: map[string]string{
			"connectionFromEnv":      "COSMOS_CONNECTION",
			"leaseConnectionFromEnv": "COSMOS_CONNECTION",
			"databaseId":             "testdb",
			"containerId":            "testcontainer",
			"leaseDatabaseId":        "testdb",
			"leaseContainerId":       "leases",
			"processorName":          "testprocessor",
		},
		isError:     false,
		resolvedEnv: testCosmosDBResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: "",
	},
	{
		name: "invalid changeFeedLagThreshold",
		metadata: map[string]string{
			"connectionFromEnv":      "COSMOS_CONNECTION",
			"databaseId":             "testdb",
			"containerId":            "testcontainer",
			"leaseDatabaseId":        "testdb",
			"leaseContainerId":       "leases",
			"processorName":          "testprocessor",
			"changeFeedLagThreshold": "invalid",
		},
		isError:     true,
		resolvedEnv: testCosmosDBResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: "",
	},
	{
		name: "invalid activationChangeFeedLagThreshold",
		metadata: map[string]string{
			"connectionFromEnv":                "COSMOS_CONNECTION",
			"databaseId":                       "testdb",
			"containerId":                      "testcontainer",
			"leaseDatabaseId":                  "testdb",
			"leaseContainerId":                 "leases",
			"processorName":                    "testprocessor",
			"activationChangeFeedLagThreshold": "invalid",
		},
		isError:     true,
		resolvedEnv: testCosmosDBResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: "",
	},
	{
		name: "endpoint with service principal",
		metadata: map[string]string{
			"endpoint":         "https://data.documents.azure.com:443/",
			"leaseEndpoint":    "https://lease.documents.azure.com:443/",
			"databaseId":       "testdb",
			"containerId":      "testcontainer",
			"leaseDatabaseId":  "leasedb",
			"leaseContainerId": "leases",
			"processorName":    "testprocessor",
		},
		isError:     false,
		resolvedEnv: map[string]string{},
		authParams: map[string]string{
			"tenantId":     "tenant-id",
			"clientId":     "client-id",
			"clientSecret": "client-secret",
		},
		podIdentity: kedav1alpha1.PodIdentityProviderNone,
	},
	{
		name: "partial service principal",
		metadata: map[string]string{
			"endpoint":         "https://test.documents.azure.com:443/",
			"databaseId":       "testdb",
			"containerId":      "testcontainer",
			"leaseDatabaseId":  "testdb",
			"leaseContainerId": "leases",
			"processorName":    "testprocessor",
		},
		isError:     true,
		resolvedEnv: map[string]string{},
		authParams: map[string]string{
			"tenantId": "tenant-id",
			"clientId": "client-id",
		},
		podIdentity: kedav1alpha1.PodIdentityProviderNone,
	},
	{
		name: "separate lease endpoint requires authentication",
		metadata: map[string]string{
			"connectionFromEnv": "COSMOS_CONNECTION",
			"leaseEndpoint":     "https://lease.documents.azure.com:443/",
			"databaseId":        "testdb",
			"containerId":       "testcontainer",
			"leaseDatabaseId":   "leasedb",
			"leaseContainerId":  "leases",
			"processorName":     "testprocessor",
		},
		isError:     true,
		resolvedEnv: testCosmosDBResolvedEnv,
		authParams:  map[string]string{},
		podIdentity: kedav1alpha1.PodIdentityProviderNone,
	},
	{
		name: "account key ignores partial service principal",
		metadata: map[string]string{
			"endpoint":         "https://test.documents.azure.com:443/",
			"databaseId":       "testdb",
			"containerId":      "testcontainer",
			"leaseDatabaseId":  "testdb",
			"leaseContainerId": "leases",
			"processorName":    "testprocessor",
		},
		isError:     false,
		resolvedEnv: map[string]string{},
		authParams: map[string]string{
			"cosmosDBKey": "dGVzdGtleQ==",
			"tenantId":    "tenant-id",
		},
		podIdentity: kedav1alpha1.PodIdentityProviderNone,
	},
	{
		name: "workload identity takes precedence over partial service principal",
		metadata: map[string]string{
			"endpoint":         "https://test.documents.azure.com:443/",
			"databaseId":       "testdb",
			"containerId":      "testcontainer",
			"leaseDatabaseId":  "testdb",
			"leaseContainerId": "leases",
			"processorName":    "testprocessor",
		},
		isError:     false,
		resolvedEnv: map[string]string{},
		authParams: map[string]string{
			"tenantId": "ignored-tenant-id",
		},
		podIdentity: kedav1alpha1.PodIdentityProviderAzureWorkload,
	},
}

var cosmosDBMetricIdentifiers = []cosmosDBMetricIdentifier{
	{
		name:             "properly formed metric",
		metadataTestData: &testCosmosDBMetadata[1],
		triggerIndex:     0,
		metricName:       "s0-azure-cosmosdb-leases-testprocessor",
	},
	{
		name:             "endpoint with key metric",
		metadataTestData: &testCosmosDBMetadata[9],
		triggerIndex:     1,
		metricName:       "s1-azure-cosmosdb-leases-testprocessor",
	},
}

func TestCosmosDBParseMetadata(t *testing.T) {
	for _, testData := range testCosmosDBMetadata {
		t.Run(testData.name, func(t *testing.T) {
			config := &scalersconfig.ScalerConfig{
				TriggerMetadata: testData.metadata,
				ResolvedEnv:     testData.resolvedEnv,
				AuthParams:      testData.authParams,
				PodIdentity:     kedav1alpha1.AuthPodIdentity{Provider: testData.podIdentity},
			}

			_, err := parseAzureCosmosDBMetadata(config)
			if err != nil && !testData.isError {
				t.Errorf("Expected success but got error: %v", err)
			}
			if testData.isError && err == nil {
				t.Errorf("Expected error but got success. testData: %v", testData)
			}
		})
	}
}

func TestNewAzureCosmosDBScalerAllowsValueMetricType(t *testing.T) {
	scaler, err := NewAzureCosmosDBScaler(&scalersconfig.ScalerConfig{
		MetricType: v2.ValueMetricType,
		TriggerMetadata: map[string]string{
			"connection":       testCosmosDBResolvedEnv["COSMOS_CONNECTION"],
			"databaseId":       "testdb",
			"containerId":      "testcontainer",
			"leaseDatabaseId":  "testdb",
			"leaseContainerId": "leases",
			"processorName":    "testprocessor",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, v2.ValueMetricType, scaler.(*azureCosmosDBScaler).metricType)
}

func TestCosmosDBServicePrincipalMetadataValidation(t *testing.T) {
	baseMetadata := map[string]string{
		"endpoint":         "https://test.documents.azure.com:443/",
		"databaseId":       "testdb",
		"containerId":      "testcontainer",
		"leaseDatabaseId":  "testdb",
		"leaseContainerId": "leases",
		"processorName":    "testprocessor",
	}
	tests := []struct {
		name       string
		authParams map[string]string
		wantError  string
	}{
		{
			name: "complete credentials",
			authParams: map[string]string{
				"tenantId":     "tenant-id",
				"clientId":     "client-id",
				"clientSecret": "client-secret",
			},
		},
		{
			name: "missing tenant id",
			authParams: map[string]string{
				"clientId":     "client-id",
				"clientSecret": "client-secret",
			},
			wantError: "tenantId, clientId and clientSecret must all be provided",
		},
		{
			name: "missing client id",
			authParams: map[string]string{
				"tenantId":     "tenant-id",
				"clientSecret": "client-secret",
			},
			wantError: "tenantId, clientId and clientSecret must all be provided",
		},
		{
			name: "missing client secret",
			authParams: map[string]string{
				"tenantId": "tenant-id",
				"clientId": "client-id",
			},
			wantError: "tenantId, clientId and clientSecret must all be provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &scalersconfig.ScalerConfig{
				TriggerMetadata: baseMetadata,
				AuthParams:      tt.authParams,
				PodIdentity:     kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
			}

			meta, err := parseAzureCosmosDBMetadata(config)
			if tt.wantError != "" {
				assert.ErrorContains(t, err, tt.wantError)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "tenant-id", meta.TenantID)
			assert.Equal(t, "client-id", meta.ClientID)
			assert.Equal(t, "client-secret", meta.ClientSecret)
		})
	}
}

func TestCosmosDBThresholdValidation(t *testing.T) {
	tests := []struct {
		name                string
		threshold           string
		activationThreshold string
		wantError           string
	}{
		{
			name:      "threshold must be positive",
			threshold: "0",
			wantError: "changeFeedLagThreshold must be greater than zero",
		},
		{
			name:                "activation threshold must not be negative",
			threshold:           "100",
			activationThreshold: "-1",
			wantError:           "activationChangeFeedLagThreshold must be greater than or equal to zero",
		},
		{
			name:                "activation threshold must remain below scaling threshold",
			threshold:           "100",
			activationThreshold: "100",
			wantError:           "activationChangeFeedLagThreshold must be less than changeFeedLagThreshold",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			triggerMetadata := map[string]string{
				"connection":             testCosmosDBResolvedEnv["COSMOS_CONNECTION"],
				"databaseId":             "testdb",
				"containerId":            "testcontainer",
				"leaseDatabaseId":        "testdb",
				"leaseContainerId":       "leases",
				"processorName":          "testprocessor",
				"changeFeedLagThreshold": tt.threshold,
			}
			if tt.activationThreshold != "" {
				triggerMetadata["activationChangeFeedLagThreshold"] = tt.activationThreshold
			}
			config := &scalersconfig.ScalerConfig{
				TriggerMetadata: triggerMetadata,
			}

			_, err := parseAzureCosmosDBMetadata(config)
			assert.ErrorContains(t, err, tt.wantError)
		})
	}
}

func TestCosmosDBCredentialSelection(t *testing.T) {
	t.Run("service principal authenticates separate data and lease endpoints", func(t *testing.T) {
		meta := &azureCosmosDBMetadata{
			Endpoint:         "https://data.documents.azure.com:443/",
			LeaseEndpoint:    "https://lease.documents.azure.com:443/",
			DatabaseID:       "testdb",
			ContainerID:      "data",
			LeaseDatabaseID:  "leasedb",
			LeaseContainerID: "leases",
			ProcessorName:    "processor",
			TenantID:         "tenant-id",
			ClientID:         "client-id",
			ClientSecret:     "client-secret",
		}

		client, err := newCosmosDBClient(meta, map[string]string{}, kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone}, logr.Discard(), time.Second)
		require.NoError(t, err)
		assert.IsType(t, &azidentity.ClientSecretCredential{}, client.credential)
		assert.Empty(t, client.dataKey)
		assert.Empty(t, client.leaseKey)
		assert.Equal(t, "https://cosmos.azure.com", client.cosmosDBResourceURL)
	})

	t.Run("service principal fills only lease authentication when data uses a connection string", func(t *testing.T) {
		meta := &azureCosmosDBMetadata{
			Connection:       testCosmosDBResolvedEnv["COSMOS_CONNECTION"],
			LeaseEndpoint:    "https://lease.documents.azure.com:443/",
			DatabaseID:       "testdb",
			ContainerID:      "data",
			LeaseDatabaseID:  "leasedb",
			LeaseContainerID: "leases",
			ProcessorName:    "processor",
			TenantID:         "tenant-id",
			ClientID:         "client-id",
			ClientSecret:     "client-secret",
		}

		client, err := newCosmosDBClient(meta, map[string]string{}, kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone}, logr.Discard(), time.Second)
		require.NoError(t, err)
		assert.NotEmpty(t, client.dataKey)
		assert.Empty(t, client.leaseKey)
		assert.IsType(t, &azidentity.ClientSecretCredential{}, client.credential)
	})

	t.Run("account keys take precedence and avoid creating an unused credential", func(t *testing.T) {
		meta := &azureCosmosDBMetadata{
			Endpoint:         "https://data.documents.azure.com:443/",
			LeaseEndpoint:    "https://lease.documents.azure.com:443/",
			CosmosDBKey:      "ZGF0YS1rZXk=",
			LeaseCosmosDBKey: "bGVhc2Uta2V5",
			DatabaseID:       "testdb",
			ContainerID:      "data",
			LeaseDatabaseID:  "leasedb",
			LeaseContainerID: "leases",
			ProcessorName:    "processor",
			TenantID:         "tenant-id",
			ClientID:         "client-id",
			ClientSecret:     "client-secret",
		}

		client, err := newCosmosDBClient(meta, map[string]string{}, kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone}, logr.Discard(), time.Second)
		require.NoError(t, err)
		assert.Nil(t, client.credential)
	})
}

func TestCosmosDBBearerAuthentication(t *testing.T) {
	credential := &recordingTokenCredential{token: "test-token"}
	client := &cosmosDBClient{
		credential:          credential,
		cosmosDBResourceURL: "https://cosmos.example/",
	}

	for _, endpoint := range []string{
		"https://data.documents.example/dbs/testdb/colls/data/docs",
		"https://lease.documents.example/dbs/leasedb/colls/leases/docs",
	} {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, endpoint, nil)
		require.NoError(t, err)
		err = client.setAuthHeader(req, http.MethodGet, "docs", "resource", "date", "")
		require.NoError(t, err)
		assert.Contains(t, req.Header.Get("Authorization"), "test-token")
	}

	assert.Equal(t, []string{"https://cosmos.example/.default", "https://cosmos.example/.default"}, credential.scopes)
}

func TestCosmosDBAccountKeyPrecedesBearerCredential(t *testing.T) {
	credential := &recordingTokenCredential{token: "test-token"}
	client := &cosmosDBClient{
		credential:          credential,
		cosmosDBResourceURL: "https://cosmos.azure.com",
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://test.documents.azure.com/dbs/testdb", nil)
	require.NoError(t, err)

	err = client.setAuthHeader(req, http.MethodGet, "dbs", "dbs/testdb", "thu, 01 jan 2024 00:00:00 gmt", "dGVzdGtleQ==")
	require.NoError(t, err)
	assert.Contains(t, req.Header.Get("Authorization"), "type%3Dmaster")
	assert.Empty(t, credential.scopes)
}

func TestResolveCosmosDBServicePrincipalCloud(t *testing.T) {
	tests := []struct {
		name                         string
		triggerMetadata              map[string]string
		wantResourceURL              string
		wantAuthorityHost            string
		wantDisableInstanceDiscovery bool
		wantError                    string
	}{
		{
			name:              "public cloud defaults",
			triggerMetadata:   map[string]string{},
			wantResourceURL:   "https://cosmos.azure.com",
			wantAuthorityHost: "https://login.microsoftonline.com/",
		},
		{
			name: "government cloud",
			triggerMetadata: map[string]string{
				"cloud": "AzureUSGovernmentCloud",
			},
			wantResourceURL:   "https://cosmos.azure.com",
			wantAuthorityHost: "https://login.microsoftonline.us/",
		},
		{
			name: "private cloud overrides",
			triggerMetadata: map[string]string{
				"cloud":                   "Private",
				"cosmosDBResourceURL":     "https://cosmos.private.example",
				"activeDirectoryEndpoint": "https://login.private.example/",
			},
			wantResourceURL:              "https://cosmos.private.example",
			wantAuthorityHost:            "https://login.private.example/",
			wantDisableInstanceDiscovery: true,
		},
		{
			name: "private cloud requires cosmos resource",
			triggerMetadata: map[string]string{
				"cloud":                   "Private",
				"activeDirectoryEndpoint": "https://login.private.example/",
			},
			wantError: "cosmosDBResourceURL must be provided",
		},
		{
			name: "private cloud requires authority endpoint",
			triggerMetadata: map[string]string{
				"cloud":               "Private",
				"cosmosDBResourceURL": "https://cosmos.private.example",
			},
			wantError: "activeDirectoryEndpoint must be provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resourceURL, credentialCloud, disableInstanceDiscovery, err := resolveCosmosDBServicePrincipalCloud(tt.triggerMetadata)
			if tt.wantError != "" {
				assert.ErrorContains(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantResourceURL, resourceURL)
			assert.Equal(t, tt.wantAuthorityHost, credentialCloud.ActiveDirectoryAuthorityHost)
			assert.Equal(t, tt.wantDisableInstanceDiscovery, disableInstanceDiscovery)
		})
	}
}

func TestCosmosDBPrivateCloudServicePrincipalTokenAcquisition(t *testing.T) {
	requests := make(chan string, 4)
	var authorityServer *httptest.Server
	authorityServer = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Method + " " + r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, ".well-known/openid-configuration") {
			_ = json.NewEncoder(w).Encode(map[string]string{
				"authorization_endpoint": authorityServer.URL + "/tenant-id/oauth2/v2.0/authorize",
				"token_endpoint":         authorityServer.URL + "/tenant-id/oauth2/v2.0/token",
				"issuer":                 authorityServer.URL + "/tenant-id/v2.0",
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"token_type":   "Bearer",
			"expires_in":   3600,
			"access_token": "private-token",
		})
	}))
	defer authorityServer.Close()

	meta := &azureCosmosDBMetadata{
		TenantID:     "tenant-id",
		ClientID:     "client-id",
		ClientSecret: "client-secret",
	}
	triggerMetadata := map[string]string{
		"cloud":                   "Private",
		"cosmosDBResourceURL":     "https://cosmos.private.example",
		"activeDirectoryEndpoint": authorityServer.URL + "/",
	}

	credential, resourceURL, err := newCosmosDBTokenCredential(
		meta,
		triggerMetadata,
		kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone},
		logr.Discard(),
		authorityServer.Client(),
	)
	require.NoError(t, err)
	token, err := credential.GetToken(context.Background(), policy.TokenRequestOptions{
		Scopes: []string{resourceURL + "/.default"},
	})
	require.NoError(t, err)
	assert.Equal(t, "private-token", token.Token)

	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		select {
		case request := <-requests:
			assert.NotContains(t, request, "discovery/instance")
			if request == "POST /tenant-id/oauth2/v2.0/token" {
				return
			}
		case <-timer.C:
			t.Fatal("service-principal credential did not request a token")
		}
	}
}

func TestCosmosDBGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range cosmosDBMetricIdentifiers {
		t.Run(testData.name, func(t *testing.T) {
			config := &scalersconfig.ScalerConfig{
				TriggerMetadata: testData.metadataTestData.metadata,
				ResolvedEnv:     testData.metadataTestData.resolvedEnv,
				AuthParams:      testData.metadataTestData.authParams,
				PodIdentity:     kedav1alpha1.AuthPodIdentity{Provider: testData.metadataTestData.podIdentity},
				TriggerIndex:    testData.triggerIndex,
			}

			meta, err := parseAzureCosmosDBMetadata(config)
			if err != nil {
				t.Fatal("Could not parse metadata:", err)
			}

			mockScaler := azureCosmosDBScaler{
				metadata:   meta,
				logger:     logr.Discard(),
				metricType: v2.AverageValueMetricType,
			}

			metricSpec := mockScaler.GetMetricSpecForScaling(context.Background())
			metricName := metricSpec[0].External.Metric.Name
			assert.Equal(t, testData.metricName, metricName)
		})
	}
}

func TestCosmosDBConnectionStringParsing(t *testing.T) {
	testCases := []struct {
		name             string
		connectionStr    string
		expectError      bool
		expectedEndpoint string
	}{
		{
			name:             "valid connection string",
			connectionStr:    "AccountEndpoint=https://test.documents.azure.com:443/;AccountKey=dGVzdGtleQ==",
			expectError:      false,
			expectedEndpoint: "https://test.documents.azure.com:443/",
		},
		{
			name:          "missing endpoint",
			connectionStr: "AccountKey=dGVzdGtleQ==",
			expectError:   true,
		},
		{
			name:          "missing key",
			connectionStr: "AccountEndpoint=https://test.documents.azure.com:443/",
			expectError:   true,
		},
		{
			name:          "empty string",
			connectionStr: "",
			expectError:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			endpoint, key, err := parseCosmosDBConnectionString(tc.connectionStr)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedEndpoint, endpoint)
				assert.NotEmpty(t, key)
			}
		})
	}
}

func TestExtractLSNFromSessionToken(t *testing.T) {
	testCases := []struct {
		name        string
		token       string
		expectedLSN string
	}{
		{
			name:        "simple format",
			token:       "0:123",
			expectedLSN: "123",
		},
		{
			name:        "compound format with global LSN",
			token:       "0:1#100#2",
			expectedLSN: "100",
		},
		{
			name:        "two segments",
			token:       "5:42#999",
			expectedLSN: "999",
		},
		{
			name:        "empty token",
			token:       "",
			expectedLSN: "",
		},
		{
			name:        "no colon",
			token:       "justanumber",
			expectedLSN: "justanumber",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			lsn := extractLSNFromSessionToken(tc.token)
			assert.Equal(t, tc.expectedLSN, lsn)
		})
	}
}

func TestExtractItemLSN(t *testing.T) {
	testCases := []struct {
		name        string
		item        string
		expectedLSN int64
		expectError bool
	}{
		{
			name:        "numeric LSN",
			item:        `{"_lsn": 1234}`,
			expectedLSN: 1234,
		},
		{
			name:        "string LSN",
			item:        `{"_lsn": "5678"}`,
			expectedLSN: 5678,
		},
		{
			name:        "missing LSN",
			item:        `{"id": "doc1"}`,
			expectedLSN: 0,
			expectError: true,
		},
		{
			name:        "invalid JSON",
			item:        `not json`,
			expectedLSN: -1,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			lsn, err := extractItemLSN(json.RawMessage(tc.item))
			if tc.expectError {
				assert.True(t, err != nil || lsn <= 0)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedLSN, lsn)
			}
		})
	}
}

func TestCosmosDBAuthTokenGeneration(t *testing.T) {
	token, err := generateCosmosDBAuthToken("get", "docs", "dbs/testdb/colls/testcol", "thu, 01 jan 2024 00:00:00 gmt", "dGVzdGtleQ==")
	assert.NoError(t, err)
	assert.Contains(t, token, "type%3Dmaster%26ver%3D1.0%26sig%3D")
}

func TestCosmosDBAuthTokenGenerationInvalidKey(t *testing.T) {
	_, err := generateCosmosDBAuthToken("get", "docs", "dbs/testdb/colls/testcol", "date", "not-valid-base64!!!")
	assert.Error(t, err)
}

func TestCosmosDBLeaseParsingDotNetFormat(t *testing.T) {
	// Realistic .NET SDK lease documents have: version=0, FeedRange, Mode, properties fields.
	// The scaler must parse LeaseToken and ContinuationToken and ignore the extra fields.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dbs/testdb/colls/leases/docs":
			// Return raw JSON matching actual .NET SDK lease format
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Documents":[
				{
					"id": "host1.documents.azure.com_abc==_def=..6",
					"version": 0,
					"_etag": "\"08000b63-0000-0800-0000-69a8c6640000\"",
					"LeaseToken": "6",
					"FeedRange": {"Range": {"min": "36DB6DB6DB6DB6DB6DB6DB6DB6DB6DB6", "max": "FF"}},
					"Owner": "dotnet-host1",
					"ContinuationToken": "\"511\"",
					"properties": {},
					"timestamp": "2026-03-04T23:55:16.5233511Z",
					"Mode": "Incremental Feed",
					"_rid": "abc123",
					"_self": "dbs/abc/colls/def/docs/ghi",
					"_ts": 1772668516
				},
				{
					"id": "host1.documents.azure.com_abc==_def=..3",
					"version": 0,
					"LeaseToken": "3",
					"FeedRange": {"Range": {"min": "0", "max": "36DB6DB6DB6DB6DB6DB6DB6DB6DB6DB6"}},
					"Owner": "dotnet-host1",
					"ContinuationToken": "\"248\"",
					"properties": {},
					"Mode": "Incremental Feed"
				},
				{
					"id": ".metadata.lease",
					"version": 0,
					"Owner": "",
					"properties": {}
				}
			]}`))
		case "/dbs/testdb/colls/data/docs":
			pkRangeID := r.Header.Get("x-ms-documentdb-partitionkeyrangeid")
			switch pkRangeID {
			case "6":
				// Partition 6 has lag: sessionLSN=600, itemLSN=512, lag=89
				w.Header().Set("x-ms-session-token", "6:0#600")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"Documents":[{"id":"doc1","_lsn":512}]}`))
			case "3":
				// Partition 3 is caught up
				w.Header().Set("x-ms-session-token", "3:0#248")
				w.WriteHeader(http.StatusNotModified)
			}
		}
	}))
	defer server.Close()

	client := &cosmosDBClient{
		httpClient:       &http.Client{},
		dataEndpoint:     server.URL,
		dataKey:          "dGVzdGtleQ==",
		leaseEndpoint:    server.URL,
		leaseKey:         "dGVzdGtleQ==",
		databaseID:       "testdb",
		containerID:      "data",
		leaseDatabaseID:  "testdb",
		leaseContainerID: "leases",
		processorName:    "testprocessor",
	}

	totalLag, activePartitionCount, _, err := client.estimateLag(context.Background())
	assert.NoError(t, err)
	// Only partition 6 has lag; partition 3 is caught up; metadata doc is filtered
	assert.Equal(t, int64(89), totalLag)
	assert.Equal(t, int64(1), activePartitionCount)
}

func TestCosmosDBLeaseParsingJavaFormat(t *testing.T) {
	// Realistic Java SDK lease documents: no version field, no FeedRange/Mode/properties.
	// The scaler must parse these identically.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dbs/testdb/colls/leases/docs":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Documents":[
				{
					"id": "myhost.documents.azure.com_changefeed-estimator_data..2",
					"_etag": "\"0100baf0-0000-0800-0000-69a8c5560000\"",
					"LeaseToken": "2",
					"ContinuationToken": "\"248\"",
					"timestamp": "2026-03-04T23:50:46.219570110Z",
					"Owner": "java-host1",
					"_rid": "5jBSAKD6NqgELTEBAAAAAA==",
					"_ts": 1772668246
				},
				{
					"id": "myhost.documents.azure.com_changefeed-estimator_data..5",
					"LeaseToken": "5",
					"ContinuationToken": "\"100\"",
					"Owner": "java-host2"
				},
				{
					"id": ".lock",
					"_etag": "\"abc\"",
					"Owner": ""
				}
			]}`))
		case "/dbs/testdb/colls/data/docs":
			pkRangeID := r.Header.Get("x-ms-documentdb-partitionkeyrangeid")
			switch pkRangeID {
			case "2":
				// Partition 2 has lag: sessionLSN=400, itemLSN=249, lag=152
				w.Header().Set("x-ms-session-token", "2:0#400")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"Documents":[{"id":"doc1","_lsn":249}]}`))
			case "5":
				// Partition 5 also has lag: sessionLSN=200, itemLSN=101, lag=100
				w.Header().Set("x-ms-session-token", "5:0#200")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"Documents":[{"id":"doc2","_lsn":101}]}`))
			}
		}
	}))
	defer server.Close()

	client := &cosmosDBClient{
		httpClient:       &http.Client{},
		dataEndpoint:     server.URL,
		dataKey:          "dGVzdGtleQ==",
		leaseEndpoint:    server.URL,
		leaseKey:         "dGVzdGtleQ==",
		databaseID:       "testdb",
		containerID:      "data",
		leaseDatabaseID:  "testdb",
		leaseContainerID: "leases",
		processorName:    "testprocessor",
	}

	totalLag, activePartitionCount, _, err := client.estimateLag(context.Background())
	assert.NoError(t, err)
	// Both partitions 2 and 5 have lag; lock doc is filtered out
	assert.Equal(t, int64(252), totalLag)
	assert.Equal(t, int64(2), activePartitionCount)
}

func TestCosmosDBLeaseParsingMixedFormats(t *testing.T) {
	// Edge case: lease container might contain docs from both SDKs (e.g. during migration).
	// The scaler should handle this gracefully since it only reads common fields.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dbs/testdb/colls/leases/docs":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Documents":[
				{
					"id": "dotnet-lease",
					"version": 0,
					"LeaseToken": "0",
					"ContinuationToken": "\"500\"",
					"Owner": "dotnet-host",
					"FeedRange": {"Range": {"min": "0", "max": "80"}},
					"Mode": "Incremental Feed"
				},
				{
					"id": "java-lease",
					"LeaseToken": "1",
					"ContinuationToken": "\"300\"",
					"Owner": "java-host"
				}
			]}`))
		case "/dbs/testdb/colls/data/docs":
			// Both partitions have lag
			w.Header().Set("x-ms-session-token", "0:0#700")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Documents":[{"id":"doc1","_lsn":550}]}`))
		}
	}))
	defer server.Close()

	client := &cosmosDBClient{
		httpClient:       &http.Client{},
		dataEndpoint:     server.URL,
		dataKey:          "dGVzdGtleQ==",
		leaseEndpoint:    server.URL,
		leaseKey:         "dGVzdGtleQ==",
		databaseID:       "testdb",
		containerID:      "data",
		leaseDatabaseID:  "testdb",
		leaseContainerID: "leases",
		processorName:    "testprocessor",
	}

	totalLag, _, _, err := client.estimateLag(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int64(302), totalLag)
}

func TestCosmosDBLeaseParsingEPKBasedDotNet(t *testing.T) {
	// .NET SDK EPK-based leases (version=1) use FeedRange with EPK ranges.
	// ContinuationToken is still a quoted LSN for incremental feed mode.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dbs/testdb/colls/leases/docs":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Documents":[
				{
					"id": "host1..epk..0-AA",
					"version": 1,
					"LeaseToken": "0",
					"FeedRange": {"Range": {"min": "", "max": "AA"}},
					"Owner": "dotnet-host1",
					"ContinuationToken": "\"750\"",
					"Mode": "LatestVersion"
				},
				{
					"id": "host1..epk..AA-FF",
					"version": 1,
					"LeaseToken": "1",
					"FeedRange": {"Range": {"min": "AA", "max": "FF"}},
					"Owner": "dotnet-host1",
					"ContinuationToken": "\"320\"",
					"Mode": "LatestVersion"
				}
			]}`))
		case "/dbs/testdb/colls/data/docs":
			pkRangeID := r.Header.Get("x-ms-documentdb-partitionkeyrangeid")
			switch pkRangeID {
			case "0":
				w.Header().Set("x-ms-session-token", "0:0#900")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"Documents":[{"id":"doc1","_lsn":751}]}`))
			case "1":
				w.Header().Set("x-ms-session-token", "1:0#320")
				w.WriteHeader(http.StatusNotModified)
			}
		}
	}))
	defer server.Close()

	client := &cosmosDBClient{
		httpClient:       &http.Client{},
		dataEndpoint:     server.URL,
		dataKey:          "dGVzdGtleQ==",
		leaseEndpoint:    server.URL,
		leaseKey:         "dGVzdGtleQ==",
		databaseID:       "testdb",
		containerID:      "data",
		leaseDatabaseID:  "testdb",
		leaseContainerID: "leases",
		processorName:    "testprocessor",
	}

	totalLag, _, _, err := client.estimateLag(context.Background())
	assert.NoError(t, err)
	// Partition 0 has lag (900-751+1=150), partition 1 is caught up
	assert.Equal(t, int64(150), totalLag)
}

func TestCosmosDBLeaseParsingEPKBasedJava(t *testing.T) {
	// Java SDK EPK-based leases (version=1) may use Base64-encoded ContinuationTokens.
	// The scaler passes ContinuationToken as-is to If-None-Match, and Cosmos DB
	// recognizes its own tokens regardless of encoding.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dbs/testdb/colls/leases/docs":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Documents":[
				{
					"id": "java-epk-lease-0",
					"version": 1,
					"LeaseToken": "0",
					"ContinuationToken": "eyJWIjoiMiIsIlJpZCI6ImFiYz0iLCJDb250aW51YXRpb24iOlt7InRva2VuIjoiXCI1MDBcIiIsInJhbmdlIjp7Im1pbiI6IiIsIm1heCI6IkZGIn19XX0=",
					"Owner": "java-host1",
					"feedRange": {"min": "", "max": "FF"}
				},
				{
					"id": "java-epk-lease-1",
					"version": 1,
					"LeaseToken": "1",
					"ContinuationToken": "eyJWIjoiMiIsIlJpZCI6ImRlZj0iLCJDb250aW51YXRpb24iOlt7InRva2VuIjoiXCIyMDBcIiIsInJhbmdlIjp7Im1pbiI6IkZGIiwibWF4IjoiRkZGRiJ9fV19",
					"Owner": "java-host2",
					"feedRange": {"min": "FF", "max": "FFFF"}
				}
			]}`))
		case "/dbs/testdb/colls/data/docs":
			// Simulate Cosmos DB accepting Base64 continuation tokens and returning results
			w.Header().Set("x-ms-session-token", "0:0#600")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Documents":[{"id":"doc1","_lsn":501}]}`))
		}
	}))
	defer server.Close()

	client := &cosmosDBClient{
		httpClient:       &http.Client{},
		dataEndpoint:     server.URL,
		dataKey:          "dGVzdGtleQ==",
		leaseEndpoint:    server.URL,
		leaseKey:         "dGVzdGtleQ==",
		databaseID:       "testdb",
		containerID:      "data",
		leaseDatabaseID:  "testdb",
		leaseContainerID: "leases",
		processorName:    "testprocessor",
	}

	totalLag, _, _, err := client.estimateLag(context.Background())
	assert.NoError(t, err)
	// Both partitions have lag; Base64 tokens are passed through to the server
	assert.Equal(t, int64(200), totalLag)
}

func TestCosmosDBLagEstimation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dbs/testdb/colls/leases/docs":
			response := map[string]interface{}{
				"Documents": []map[string]interface{}{
					{
						"id":                "lease1",
						"LeaseToken":        "0",
						"ContinuationToken": `"1000"`,
						"Owner":             "testowner",
					},
					{
						"id":                "lease2",
						"LeaseToken":        "1",
						"ContinuationToken": `"2000"`,
						"Owner":             "testowner",
					},
					{
						// Metadata doc - should be filtered out
						"id":    "metadata",
						"Owner": "metadata",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		case "/dbs/testdb/colls/testcontainer/docs":
			pkRangeID := r.Header.Get("x-ms-documentdb-partitionkeyrangeid")

			switch pkRangeID {
			case "0":
				// Partition with lag: sessionLSN=1100, itemLSN=1050, lag=51
				w.Header().Set("x-ms-session-token", "0:0#1100")
				w.Header().Set("Content-Type", "application/json")
				response := map[string]interface{}{
					"Documents": []map[string]interface{}{
						{"id": "item1", "_lsn": 1050},
					},
				}
				_ = json.NewEncoder(w).Encode(response)
			default:
				// Partition without lag (304 Not Modified)
				w.Header().Set("x-ms-session-token", "1:0#2000")
				w.WriteHeader(http.StatusNotModified)
			}
		}
	}))
	defer server.Close()

	client := &cosmosDBClient{
		httpClient:       &http.Client{},
		dataEndpoint:     server.URL,
		dataKey:          "dGVzdGtleQ==",
		leaseEndpoint:    server.URL,
		leaseKey:         "dGVzdGtleQ==",
		databaseID:       "testdb",
		containerID:      "testcontainer",
		leaseDatabaseID:  "testdb",
		leaseContainerID: "leases",
		processorName:    "testprocessor",
	}

	totalLag, _, _, err := client.estimateLag(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int64(51), totalLag)
}

func TestCosmosDBLagEstimationEmptyLeases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"Documents": []map[string]interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &cosmosDBClient{
		httpClient:       &http.Client{},
		leaseEndpoint:    server.URL,
		leaseKey:         "dGVzdGtleQ==",
		leaseDatabaseID:  "testdb",
		leaseContainerID: "leases",
		processorName:    "testprocessor",
	}

	totalLag, activePartitionCount, _, err := client.estimateLag(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int64(0), totalLag)
	assert.Equal(t, int64(0), activePartitionCount)
}

// TestCosmosDBLagEstimationNeverCheckpointedLease covers a lease document that exists
// (LeaseToken set) but has never checkpointed yet - a real, common state immediately after
// CreateLeaseIfNotExistAsync, before the processor has completed a single change feed read.
// Such a lease must NOT be treated the same as a metadata/lock document (which has neither
// field): its backlog must still be detected by reading the change feed from the beginning
// (no If-None-Match sent), exactly like a fresh customer's very first deployment. Filtering
// it out here would make that backlog permanently invisible to the scaler.
func TestCosmosDBLagEstimationNeverCheckpointedLease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dbs/testdb/colls/leases/docs":
			response := map[string]interface{}{
				"Documents": []map[string]interface{}{
					{
						"id":                "lease1",
						"LeaseToken":        "0",
						"ContinuationToken": nil,
						"Owner":             "testowner",
					},
					{
						// Metadata doc - has neither field, must still be filtered out
						"id":    "metadata",
						"Owner": "metadata",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		case "/dbs/testdb/colls/testcontainer/docs":
			assert.Empty(t, r.Header.Get("If-None-Match"), "a never-checkpointed lease must read from the beginning")
			w.Header().Set("x-ms-session-token", "0:0#10")
			w.Header().Set("Content-Type", "application/json")
			response := map[string]interface{}{
				"Documents": []map[string]interface{}{
					{"id": "item1", "_lsn": 1},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	client := &cosmosDBClient{
		httpClient:       &http.Client{},
		dataEndpoint:     server.URL,
		dataKey:          "dGVzdGtleQ==",
		leaseEndpoint:    server.URL,
		leaseKey:         "dGVzdGtleQ==",
		databaseID:       "testdb",
		containerID:      "testcontainer",
		leaseDatabaseID:  "testdb",
		leaseContainerID: "leases",
		processorName:    "testprocessor",
	}

	totalLag, activePartitionCount, _, err := client.estimateLag(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int64(10), totalLag, "backlog behind a never-checkpointed lease must be detected, not silently reported as 0")
	assert.Equal(t, int64(1), activePartitionCount)
}

func TestCosmosDBLagEstimationManyBootstrappingPartitionsCollapseForCap(t *testing.T) {
	// Simulates a container with 3 partitions all bootstrapping simultaneously (never
	// checkpointed), each reporting real backlog. Without dampening, activePartitionCount
	// would be 3, letting the scale-out cap request up to 3 replicas purely from a one-time
	// historical backlog visible on first bootstrap, rather than genuine ongoing accumulation.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dbs/testdb/colls/leases/docs":
			response := map[string]interface{}{
				"Documents": []map[string]interface{}{
					{"id": "lease0", "LeaseToken": "0", "ContinuationToken": nil, "Owner": "owner0"},
					{"id": "lease1", "LeaseToken": "1", "ContinuationToken": nil, "Owner": "owner1"},
					{"id": "lease2", "LeaseToken": "2", "ContinuationToken": nil, "Owner": "owner2"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		case "/dbs/testdb/colls/testcontainer/docs":
			partitionID := r.Header.Get("x-ms-documentdb-partitionkeyrangeid")
			assert.Empty(t, r.Header.Get("If-None-Match"), "partition %s: a never-checkpointed lease must read from the beginning", partitionID)
			w.Header().Set("x-ms-session-token", "0:0#10")
			w.Header().Set("Content-Type", "application/json")
			response := map[string]interface{}{
				"Documents": []map[string]interface{}{
					{"id": "item-" + partitionID, "_lsn": 1},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	client := &cosmosDBClient{
		httpClient:       &http.Client{},
		dataEndpoint:     server.URL,
		dataKey:          "dGVzdGtleQ==",
		leaseEndpoint:    server.URL,
		leaseKey:         "dGVzdGtleQ==",
		databaseID:       "testdb",
		containerID:      "testcontainer",
		leaseDatabaseID:  "testdb",
		leaseContainerID: "leases",
		processorName:    "testprocessor",
	}

	totalLag, activePartitionCount, _, err := client.estimateLag(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int64(30), totalLag, "raw backlog across all 3 bootstrapping partitions must still be fully detected")
	assert.Equal(t, int64(1), activePartitionCount, "never-checkpointed partitions must collapse to at most one partition's worth for scale-out capping")
}

func TestCosmosDBLagEstimationMixedCheckpointedAndBootstrappingPartitions(t *testing.T) {
	// 2 partitions already checkpointed with real ongoing lag, plus 3 more partitions
	// simultaneously bootstrapping (never checkpointed). Checkpointed partitions must still
	// count individually toward the scale-out cap; the bootstrapping group counts as only one more.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dbs/testdb/colls/leases/docs":
			response := map[string]interface{}{
				"Documents": []map[string]interface{}{
					{"id": "lease0", "LeaseToken": "0", "ContinuationToken": `"100"`, "Owner": "owner0"},
					{"id": "lease1", "LeaseToken": "1", "ContinuationToken": `"100"`, "Owner": "owner1"},
					{"id": "lease2", "LeaseToken": "2", "ContinuationToken": nil, "Owner": "owner2"},
					{"id": "lease3", "LeaseToken": "3", "ContinuationToken": nil, "Owner": "owner3"},
					{"id": "lease4", "LeaseToken": "4", "ContinuationToken": nil, "Owner": "owner4"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		case "/dbs/testdb/colls/testcontainer/docs":
			partitionID := r.Header.Get("x-ms-documentdb-partitionkeyrangeid")
			w.Header().Set("x-ms-session-token", "0:0#10")
			w.Header().Set("Content-Type", "application/json")
			response := map[string]interface{}{
				"Documents": []map[string]interface{}{
					{"id": "item-" + partitionID, "_lsn": 1},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	client := &cosmosDBClient{
		httpClient:       &http.Client{},
		dataEndpoint:     server.URL,
		dataKey:          "dGVzdGtleQ==",
		leaseEndpoint:    server.URL,
		leaseKey:         "dGVzdGtleQ==",
		databaseID:       "testdb",
		containerID:      "testcontainer",
		leaseDatabaseID:  "testdb",
		leaseContainerID: "leases",
		processorName:    "testprocessor",
	}

	totalLag, activePartitionCount, _, err := client.estimateLag(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int64(50), totalLag, "all 5 partitions contribute their real lag to the total")
	assert.Equal(t, int64(3), activePartitionCount, "2 checkpointed partitions count individually, plus 1 for the collapsed bootstrapping group")
}

func TestCosmosDBLagEstimationAllPartitionsLagging(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dbs/testdb/colls/leases/docs":
			response := map[string]interface{}{
				"Documents": []map[string]interface{}{
					{"id": "lease1", "LeaseToken": "0", "ContinuationToken": `"100"`, "Owner": "owner1"},
					{"id": "lease2", "LeaseToken": "1", "ContinuationToken": `"200"`, "Owner": "owner2"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		case "/dbs/testdb/colls/testcontainer/docs":
			// Both partitions have lag
			w.Header().Set("x-ms-session-token", "0:0#500")
			w.Header().Set("Content-Type", "application/json")
			response := map[string]interface{}{
				"Documents": []map[string]interface{}{
					{"id": "item1", "_lsn": 400},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	client := &cosmosDBClient{
		httpClient:       &http.Client{},
		dataEndpoint:     server.URL,
		dataKey:          "dGVzdGtleQ==",
		leaseEndpoint:    server.URL,
		leaseKey:         "dGVzdGtleQ==",
		databaseID:       "testdb",
		containerID:      "testcontainer",
		leaseDatabaseID:  "testdb",
		leaseContainerID: "leases",
		processorName:    "testprocessor",
	}

	totalLag, activePartitionCount, _, err := client.estimateLag(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int64(202), totalLag)
	assert.Equal(t, int64(2), activePartitionCount)
}

func TestCosmosDBLagEstimationRecoversPartitionSplit(t *testing.T) {
	leaseQueryCount := 0
	changeFeedCalls := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dbs/testdb/colls/leases/docs":
			leaseQueryCount++
			leases := []map[string]interface{}{
				{"id": "parent", "LeaseToken": "0", "ContinuationToken": `"100"`, "Owner": "owner1"},
			}
			if leaseQueryCount > 1 {
				leases = []map[string]interface{}{
					{"id": "child1", "LeaseToken": "1", "ContinuationToken": `"100"`, "Owner": "owner1"},
					{"id": "child2", "LeaseToken": "2", "ContinuationToken": `"100"`, "Owner": "owner2"},
				}
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"Documents": leases})
		case "/dbs/testdb/colls/testcontainer/docs":
			partitionID := r.Header.Get("x-ms-documentdb-partitionkeyrangeid")
			changeFeedCalls[partitionID]++
			switch partitionID {
			case "0":
				w.Header().Set(cosmosDBSubStatusHeader, cosmosDBPartitionKeyRangeGoneSubStatus)
				w.WriteHeader(http.StatusGone)
			case "1":
				w.Header().Set("x-ms-session-token", "1:0#150")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"Documents":[{"id":"item1","_lsn":100}]}`))
			case "2":
				w.WriteHeader(http.StatusNotModified)
			}
		}
	}))
	defer server.Close()

	client := &cosmosDBClient{
		httpClient:       &http.Client{},
		dataEndpoint:     server.URL,
		dataKey:          "dGVzdGtleQ==",
		leaseEndpoint:    server.URL,
		leaseKey:         "dGVzdGtleQ==",
		databaseID:       "testdb",
		containerID:      "testcontainer",
		leaseDatabaseID:  "testdb",
		leaseContainerID: "leases",
		processorName:    "testprocessor",
	}

	totalLag, activePartitionCount, splitRecoveryRequired, err := client.estimateLag(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int64(51), totalLag)
	assert.Equal(t, int64(1), activePartitionCount)
	assert.False(t, splitRecoveryRequired)
	assert.Equal(t, 2, leaseQueryCount)
	assert.Equal(t, map[string]int{"0": 1, "1": 1, "2": 1}, changeFeedCalls)
}

func TestCosmosDBPersistentPartitionSplitRecoverySignal(t *testing.T) {
	tests := []struct {
		name                string
		activationThreshold int64
		includeLaggingLease bool
		expectedMetric      int64
	}{
		{
			name:           "default activation threshold",
			expectedMetric: 1,
		},
		{
			name:                "nonzero activation threshold",
			activationThreshold: 10,
			expectedMetric:      11,
		},
		{
			name:                "split does not broaden active partition cap",
			includeLaggingLease: true,
			expectedMetric:      100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			leaseQueryCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/dbs/testdb/colls/leases/docs":
					leaseQueryCount++
					leases := []map[string]string{
						{"id": "parent", "LeaseToken": "0", "ContinuationToken": `"100"`, "Owner": "owner1"},
					}
					if tt.includeLaggingLease {
						leases = append(leases, map[string]string{
							"id": "sibling", "LeaseToken": "1", "ContinuationToken": `"100"`, "Owner": "owner2",
						})
					}
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(map[string]interface{}{"Documents": leases})
				case "/dbs/testdb/colls/testcontainer/docs":
					if r.Header.Get("x-ms-documentdb-partitionkeyrangeid") == "0" {
						w.Header().Set(cosmosDBSubStatusHeader, cosmosDBPartitionKeyRangeGoneSubStatus)
						w.WriteHeader(http.StatusGone)
						return
					}
					w.Header().Set("x-ms-session-token", "1:0#1000")
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"Documents":[{"id":"item1","_lsn":1}]}`))
				}
			}))
			defer server.Close()

			scaler := &azureCosmosDBScaler{
				metricType: v2.AverageValueMetricType,
				metadata: &azureCosmosDBMetadata{
					Threshold:           100,
					ActivationThreshold: tt.activationThreshold,
				},
				cosmosClient: &cosmosDBClient{
					httpClient:       &http.Client{},
					dataEndpoint:     server.URL,
					dataKey:          "dGVzdGtleQ==",
					leaseEndpoint:    server.URL,
					leaseKey:         "dGVzdGtleQ==",
					databaseID:       "testdb",
					containerID:      "testcontainer",
					leaseDatabaseID:  "testdb",
					leaseContainerID: "leases",
					processorName:    "testprocessor",
				},
				logger: logr.Discard(),
			}

			metrics, isActive, err := scaler.GetMetricsAndActivity(context.Background(), "test-metric")
			require.NoError(t, err)
			assert.True(t, isActive)
			require.Len(t, metrics, 1)
			assert.Equal(t, tt.expectedMetric, metrics[0].Value.Value())
			assert.Equal(t, 2, leaseQueryCount)
		})
	}
}

func TestCosmosDBUnrelatedGoneResponseIsError(t *testing.T) {
	leaseQueryCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dbs/testdb/colls/leases/docs":
			leaseQueryCount++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Documents":[{"id":"lease1","LeaseToken":"0","ContinuationToken":"\"100\"","Owner":"owner1"}]}`))
		case "/dbs/testdb/colls/testcontainer/docs":
			w.Header().Set(cosmosDBSubStatusHeader, "1008")
			w.WriteHeader(http.StatusGone)
		}
	}))
	defer server.Close()

	client := &cosmosDBClient{
		httpClient:       &http.Client{},
		dataEndpoint:     server.URL,
		dataKey:          "dGVzdGtleQ==",
		leaseEndpoint:    server.URL,
		leaseKey:         "dGVzdGtleQ==",
		databaseID:       "testdb",
		containerID:      "testcontainer",
		leaseDatabaseID:  "testdb",
		leaseContainerID: "leases",
		processorName:    "testprocessor",
	}

	totalLag, activePartitionCount, splitRecoveryRequired, err := client.estimateLag(context.Background())
	assert.ErrorContains(t, err, `status 410 and substatus "1008"`)
	assert.Zero(t, totalLag)
	assert.Zero(t, activePartitionCount)
	assert.False(t, splitRecoveryRequired)
	assert.Equal(t, 1, leaseQueryCount)
}

func TestCosmosDBLagEstimationPartitionError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dbs/testdb/colls/leases/docs":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Documents":[
				{"id":"lease1","LeaseToken":"0","ContinuationToken":"\"100\"","Owner":"owner1"},
				{"id":"lease2","LeaseToken":"1","ContinuationToken":"\"200\"","Owner":"owner2"}
			]}`))
		case "/dbs/testdb/colls/testcontainer/docs":
			if r.Header.Get("x-ms-documentdb-partitionkeyrangeid") == "0" {
				w.Header().Set("x-ms-session-token", "0:0#200")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"Documents":[{"id":"item1","_lsn":101}]}`))
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	client := &cosmosDBClient{
		httpClient:       &http.Client{},
		dataEndpoint:     server.URL,
		dataKey:          "dGVzdGtleQ==",
		leaseEndpoint:    server.URL,
		leaseKey:         "dGVzdGtleQ==",
		databaseID:       "testdb",
		containerID:      "testcontainer",
		leaseDatabaseID:  "testdb",
		leaseContainerID: "leases",
		processorName:    "testprocessor",
	}

	totalLag, activePartitionCount, splitRecoveryRequired, err := client.estimateLag(context.Background())
	assert.ErrorContains(t, err, "error estimating lag: error reading change feed for partition 1")
	assert.Zero(t, totalLag)
	assert.Zero(t, activePartitionCount)
	assert.False(t, splitRecoveryRequired)
}

func TestCosmosDBGetMetricsAndActivityCapsLagByActivePartitions(t *testing.T) {
	tests := []struct {
		name           string
		partitionLags  []int64
		expectedMetric int64
		expectedActive bool
	}{
		{
			name:           "one active partition among several",
			partitionLags:  []int64{1000, 0, 0},
			expectedMetric: 100,
			expectedActive: true,
		},
		{
			name:           "multiple active partitions",
			partitionLags:  []int64{101, 100, 0},
			expectedMetric: 200,
			expectedActive: true,
		},
		{
			name:           "no active partitions",
			partitionLags:  []int64{0, 0, 0},
			expectedMetric: 0,
			expectedActive: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/dbs/testdb/colls/leases/docs":
					leases := make([]map[string]string, 0, len(tt.partitionLags))
					for i := range tt.partitionLags {
						partitionID := strconv.Itoa(i)
						leases = append(leases, map[string]string{
							"id":                "lease" + partitionID,
							"LeaseToken":        partitionID,
							"ContinuationToken": `"100"`,
							"Owner":             "owner" + partitionID,
						})
					}
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(map[string]interface{}{"Documents": leases})
				case "/dbs/testdb/colls/testcontainer/docs":
					partitionID, err := strconv.Atoi(r.Header.Get("x-ms-documentdb-partitionkeyrangeid"))
					assert.NoError(t, err)
					lag := tt.partitionLags[partitionID]
					if lag == 0 {
						w.WriteHeader(http.StatusNotModified)
						return
					}
					w.Header().Set("x-ms-session-token", strconv.Itoa(partitionID)+":0#"+strconv.FormatInt(lag, 10))
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"Documents":[{"id":"item1","_lsn":1}]}`))
				}
			}))
			defer server.Close()

			scaler := &azureCosmosDBScaler{
				metricType: v2.AverageValueMetricType,
				metadata: &azureCosmosDBMetadata{
					DatabaseID:          "testdb",
					ContainerID:         "testcontainer",
					LeaseDatabaseID:     "testdb",
					LeaseContainerID:    "leases",
					ProcessorName:       "testprocessor",
					Threshold:           100,
					ActivationThreshold: 0,
				},
				cosmosClient: &cosmosDBClient{
					httpClient:       &http.Client{},
					dataEndpoint:     server.URL,
					dataKey:          "dGVzdGtleQ==",
					leaseEndpoint:    server.URL,
					leaseKey:         "dGVzdGtleQ==",
					databaseID:       "testdb",
					containerID:      "testcontainer",
					leaseDatabaseID:  "testdb",
					leaseContainerID: "leases",
					processorName:    "testprocessor",
				},
				logger: logr.Discard(),
			}

			metrics, isActive, err := scaler.GetMetricsAndActivity(context.Background(), "test-metric")
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedActive, isActive)
			if assert.Len(t, metrics, 1) {
				assert.Equal(t, tt.expectedMetric, metrics[0].Value.Value())
			}
		})
	}
}

// TestCosmosDBGetMetricsAndActivityNeverCheckpointedPartitionsStayCapped proves why the
// never-checkpointed collapse in estimateOnce must floor activePartitionCount at 1, not 0.
// getChangeFeedTotalLagRelatedToPartitionAmount treats activePartitionCount<=0 as "nothing to
// cap against" and passes totalLag through uncapped - safe in the original code, where that
// case only arose when totalLag was also 0. Collapsing all-never-checkpointed partitions to 0
// would put a nonzero totalLag through that same uncapped escape hatch, defeating the cap
// entirely; flooring at 1 keeps a real (if minimal) cap of exactly 1*threshold in effect.
func TestCosmosDBGetMetricsAndActivityNeverCheckpointedPartitionsStayCapped(t *testing.T) {
	const partitionCount = 5
	const lagPerPartition = 1000 // 5000 raw total - would pass through uncapped if the floor were 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dbs/testdb/colls/leases/docs":
			leases := make([]map[string]interface{}, 0, partitionCount)
			for i := 0; i < partitionCount; i++ {
				partitionID := strconv.Itoa(i)
				leases = append(leases, map[string]interface{}{
					"id":                "lease" + partitionID,
					"LeaseToken":        partitionID,
					"ContinuationToken": nil, // never checkpointed
					"Owner":             "owner" + partitionID,
				})
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"Documents": leases})
		case "/dbs/testdb/colls/testcontainer/docs":
			partitionID := r.Header.Get("x-ms-documentdb-partitionkeyrangeid")
			w.Header().Set("x-ms-session-token", "0:0#"+strconv.Itoa(lagPerPartition))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Documents":[{"id":"item-` + partitionID + `","_lsn":1}]}`))
		}
	}))
	defer server.Close()

	scaler := &azureCosmosDBScaler{
		metricType: v2.AverageValueMetricType,
		metadata: &azureCosmosDBMetadata{
			DatabaseID:          "testdb",
			ContainerID:         "testcontainer",
			LeaseDatabaseID:     "testdb",
			LeaseContainerID:    "leases",
			ProcessorName:       "testprocessor",
			Threshold:           100,
			ActivationThreshold: 0,
		},
		cosmosClient: &cosmosDBClient{
			httpClient:       &http.Client{},
			dataEndpoint:     server.URL,
			dataKey:          "dGVzdGtleQ==",
			leaseEndpoint:    server.URL,
			leaseKey:         "dGVzdGtleQ==",
			databaseID:       "testdb",
			containerID:      "testcontainer",
			leaseDatabaseID:  "testdb",
			leaseContainerID: "leases",
			processorName:    "testprocessor",
		},
		logger: logr.Discard(),
	}

	metrics, isActive, err := scaler.GetMetricsAndActivity(context.Background(), "test-metric")
	assert.NoError(t, err)
	assert.True(t, isActive, "real backlog behind never-checkpointed leases must still activate scale-from-zero")
	if assert.Len(t, metrics, 1) {
		assert.Equal(t, int64(100), metrics[0].Value.Value(),
			"metric must stay capped at 1*threshold, not pass through the raw 5000 total lag uncapped")
	}
}

func TestCosmosDBGetMetricsAndActivity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dbs/testdb/colls/leases/docs":
			response := map[string]interface{}{
				"Documents": []map[string]interface{}{
					{"id": "lease1", "LeaseToken": "0", "ContinuationToken": `"100"`, "Owner": "owner1"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		case "/dbs/testdb/colls/testcontainer/docs":
			w.Header().Set("x-ms-session-token", "0:0#200")
			w.Header().Set("Content-Type", "application/json")
			response := map[string]interface{}{
				"Documents": []map[string]interface{}{
					{"id": "item1", "_lsn": 150},
				},
			}
			_ = json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	scaler := &azureCosmosDBScaler{
		metricType: v2.AverageValueMetricType,
		metadata: &azureCosmosDBMetadata{
			DatabaseID:          "testdb",
			ContainerID:         "testcontainer",
			LeaseDatabaseID:     "testdb",
			LeaseContainerID:    "leases",
			ProcessorName:       "testprocessor",
			Threshold:           1,
			ActivationThreshold: 0,
		},
		cosmosClient: &cosmosDBClient{
			httpClient:       &http.Client{},
			dataEndpoint:     server.URL,
			dataKey:          "dGVzdGtleQ==",
			leaseEndpoint:    server.URL,
			leaseKey:         "dGVzdGtleQ==",
			databaseID:       "testdb",
			containerID:      "testcontainer",
			leaseDatabaseID:  "testdb",
			leaseContainerID: "leases",
			processorName:    "testprocessor",
		},
		logger: logr.Discard(),
	}

	metrics, isActive, err := scaler.GetMetricsAndActivity(context.Background(), "test-metric")
	assert.NoError(t, err)
	assert.True(t, isActive)
	assert.Len(t, metrics, 1)
}

func TestCosmosDBGetMetricsAndActivityNotActive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dbs/testdb/colls/leases/docs":
			response := map[string]interface{}{
				"Documents": []map[string]interface{}{
					{"id": "lease1", "LeaseToken": "0", "ContinuationToken": `"100"`, "Owner": "owner1"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		case "/dbs/testdb/colls/testcontainer/docs":
			// Caught up
			w.Header().Set("x-ms-session-token", "0:0#100")
			w.WriteHeader(http.StatusNotModified)
		}
	}))
	defer server.Close()

	scaler := &azureCosmosDBScaler{
		metricType: v2.AverageValueMetricType,
		metadata: &azureCosmosDBMetadata{
			DatabaseID:          "testdb",
			ContainerID:         "testcontainer",
			LeaseDatabaseID:     "testdb",
			LeaseContainerID:    "leases",
			ProcessorName:       "testprocessor",
			Threshold:           1,
			ActivationThreshold: 0,
		},
		cosmosClient: &cosmosDBClient{
			httpClient:       &http.Client{},
			dataEndpoint:     server.URL,
			dataKey:          "dGVzdGtleQ==",
			leaseEndpoint:    server.URL,
			leaseKey:         "dGVzdGtleQ==",
			databaseID:       "testdb",
			containerID:      "testcontainer",
			leaseDatabaseID:  "testdb",
			leaseContainerID: "leases",
			processorName:    "testprocessor",
		},
		logger: logr.Discard(),
	}

	metrics, isActive, err := scaler.GetMetricsAndActivity(context.Background(), "test-metric")
	assert.NoError(t, err)
	assert.False(t, isActive)
	assert.Len(t, metrics, 1)
}

func TestCosmosDBGetMetricsAndActivityOnError(t *testing.T) {
	// Server that returns 500 for all requests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal server error"}`))
	}))
	defer server.Close()

	scaler := &azureCosmosDBScaler{
		metricType: v2.AverageValueMetricType,
		metadata: &azureCosmosDBMetadata{
			DatabaseID:          "testdb",
			ContainerID:         "testcontainer",
			LeaseDatabaseID:     "testdb",
			LeaseContainerID:    "leases",
			ProcessorName:       "testprocessor",
			Threshold:           100,
			ActivationThreshold: 0,
		},
		cosmosClient: &cosmosDBClient{
			httpClient:       &http.Client{},
			dataEndpoint:     server.URL,
			dataKey:          "dGVzdGtleQ==",
			leaseEndpoint:    server.URL,
			leaseKey:         "dGVzdGtleQ==",
			databaseID:       "testdb",
			containerID:      "testcontainer",
			leaseDatabaseID:  "testdb",
			leaseContainerID: "leases",
			processorName:    "testprocessor",
		},
		logger: logr.Discard(),
	}

	// On error, propagate to KEDA and let fallback spec on ScaledObject handle it
	_, _, err := scaler.GetMetricsAndActivity(context.Background(), "test-metric")
	assert.Error(t, err)
}

func TestCosmosDBLeaseQueryFiltersByProcessorName(t *testing.T) {
	var capturedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/dbs/testdb/colls/leases/docs" {
			bodyBytes, _ := io.ReadAll(r.Body)
			capturedBody = string(bodyBytes)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Documents":[]}`))
		}
	}))
	defer server.Close()

	client := &cosmosDBClient{
		httpClient:       &http.Client{},
		dataEndpoint:     "https://myaccount.documents.azure.com:443/",
		leaseEndpoint:    server.URL,
		leaseKey:         "dGVzdGtleQ==",
		leaseDatabaseID:  "testdb",
		leaseContainerID: "leases",
		processorName:    "myprocessor",
	}

	_, _ = client.queryLeases(context.Background())
	assert.Contains(t, capturedBody, "STARTSWITH")
	assert.Contains(t, capturedBody, "@prefix")

	var captured struct {
		Parameters []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"parameters"`
	}
	require.NoError(t, json.Unmarshal([]byte(capturedBody), &captured))
	require.Len(t, captured.Parameters, 1)
	// The prefix must include the monitored (data) account's short name, matching the real
	// .NET/Java SDK lease id format of {processorName}{accountHost}_{rid}..{partitionId} -
	// processorName alone would let "myprocessor-extended" collide with "myprocessor".
	assert.Equal(t, "myprocessormyaccount.", captured.Parameters[0].Value)
}

func TestCosmosDBAccountShortName(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		expected string
	}{
		{"standard public cloud endpoint", "https://myaccount.documents.azure.com:443/", "myaccount"},
		{"endpoint without port", "https://myaccount.documents.azure.com/", "myaccount"},
		{"sovereign cloud endpoint", "https://myaccount.documents.azure.cn:443/", "myaccount"},
		{"bare hostname, no scheme", "myaccount.documents.azure.com", "myaccount"},
		{"empty endpoint", "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, cosmosDBAccountShortName(tc.endpoint))
		})
	}
}

// TestCosmosDBLeasePrefixAvoidsProcessorNameCollision proves the fix for the collision the
// prefix must avoid: a processor named "app" must not match leases belonging to a differently
// named processor such as "app-extended" that happens to share the same string prefix. This
// uses realistic (unquoted) lease ids in the exact shape the .NET/Java SDKs generate, and
// STARTSWITH's semantics (strings.HasPrefix), rather than a mocked HTTP response - Cosmos DB
// evaluates STARTSWITH server-side, so the scaler's Go code never sees non-matching documents.
func TestCosmosDBLeasePrefixAvoidsProcessorNameCollision(t *testing.T) {
	const dataEndpoint = "https://myaccount.documents.azure.com:443/"
	accountHost := "myaccount.documents.azure.com"

	buildLeaseID := func(processorName string) string {
		return processorName + accountHost + "_dbRid_collRid..0"
	}

	appPrefix := "app" + cosmosDBAccountShortName(dataEndpoint) + "."
	assert.True(t, strings.HasPrefix(buildLeaseID("app"), appPrefix),
		"processor 'app' must match its own lease")
	assert.False(t, strings.HasPrefix(buildLeaseID("app-extended"), appPrefix),
		"processor 'app' must NOT match a lease belonging to 'app-extended'")
}
