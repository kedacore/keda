package scalers

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
	v2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/fallback"
	"github.com/kedacore/keda/v2/pkg/scalers/azure"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

const cosmosDBDotNetV1Leases = `{
  "Documents": [
    {
      "id": "testprocessormyaccount.documents.azure.com_q0oIAA==_q0oIALS6YvQ=..-1FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF",
      "version": 1,
      "LeaseToken": "-1FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF",
      "FeedRange": {"Range": {"min": "", "max": "1FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"}},
      "Owner": "dotnet-host1",
      "ContinuationToken": "\"750\"",
      "properties": {},
      "timestamp": "2026-03-04T23:55:16.5233511Z",
      "Mode": "Incremental Feed"
    },
    {
      "id": "testprocessormyaccount.documents.azure.com_q0oIAA==_q0oIALS6YvQ=..1FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF-FF",
      "version": 1,
      "LeaseToken": "1FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF-FF",
      "FeedRange": {"Range": {"min": "1FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF", "max": "FF"}},
      "Owner": "dotnet-host1",
      "ContinuationToken": "\"320\"",
      "properties": {},
      "Mode": "Incremental Feed"
    }
  ]
}
`

const cosmosDBJavaV1Leases = `{
  "Documents": [
    {
      "id": "testprocessormyaccount.documents.azure.com_q0oIAA==_q0oIALS6YvQ=..-1FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF",
      "LeaseToken": "-1FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF",
      "ContinuationToken": "eyJWIjoxLCJSaWQiOiJxMG9JQUxTNll2UT0iLCJNb2RlIjoiSU5DUkVNRU5UQUwiLCJTdGFydEZyb20iOnsiVHlwZSI6IkJFR0lOTklORyJ9LCJDb250aW51YXRpb24iOnsiViI6MSwiUmlkIjoicTBvSUFMUzZZdlE9IiwiQ29udGludWF0aW9uIjpbeyJ0b2tlbiI6IlwiNTAwXCIiLCJyYW5nZSI6eyJtaW4iOiIiLCJtYXgiOiIxRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRiJ9fV0sIlJhbmdlIjp7Im1pbiI6IiIsIm1heCI6IjFGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGIn19fQ==",
      "timestamp": "2026-03-04T23:50:46.219570110Z",
      "Owner": "java-host1",
      "version": 1,
      "feedRange": {"Range": {"min": "", "max": "1FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"}}
    },
    {
      "id": "testprocessormyaccount.documents.azure.com_q0oIAA==_q0oIALS6YvQ=..1FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF-FF",
      "LeaseToken": "1FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF-FF",
      "ContinuationToken": "eyJWIjoxLCJSaWQiOiJxMG9JQUxTNll2UT0iLCJNb2RlIjoiSU5DUkVNRU5UQUwiLCJTdGFydEZyb20iOnsiVHlwZSI6IkJFR0lOTklORyJ9LCJDb250aW51YXRpb24iOnsiViI6MSwiUmlkIjoicTBvSUFMUzZZdlE9IiwiQ29udGludWF0aW9uIjpbeyJ0b2tlbiI6IlwiMjAwXCIiLCJyYW5nZSI6eyJtaW4iOiIxRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRiIsIm1heCI6IkZGIn19XSwiUmFuZ2UiOnsibWluIjoiMUZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkZGRkYiLCJtYXgiOiJGRiJ9fX0=",
      "timestamp": "2026-03-04T23:50:46.219570110Z",
      "Owner": "java-host2",
      "version": 1,
      "feedRange": {"Range": {"min": "1FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF", "max": "FF"}}
    }
  ]
}
`

var testCosmosDBResolvedEnv = map[string]string{
	"COSMOS_CONNECTION": "AccountEndpoint=https://test.documents.azure.com:443/;AccountKey=dGVzdGtleQ==",
}

func estimateCosmosDBLegacyLag(client *cosmosDBClient) (int64, int64, bool, error) {
	state, err := client.estimateLag(context.Background())
	return state.totalLag, state.legacyActivePartitions, state.splitRecoveryRequired, err
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
		dataEndpoint:        "https://data.documents.example",
		leaseEndpoint:       "https://lease.documents.example",
	}

	for _, endpoint := range []string{
		"https://data.documents.example/dbs/testdb/colls/data/docs",
		"https://lease.documents.example/dbs/leasedb/colls/leases/docs",
	} {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, endpoint, nil)
		require.NoError(t, err)
		err = client.setAuthHeader(req, http.MethodGet, "docs", "resource", "date", "")
		require.NoError(t, err)
		authHeader := req.Header.Get("Authorization")
		// Cosmos DB expects the URL-escaped form of "type=aad&ver=1.0&sig={bearer-token}"
		// on the wire, so the header must start with type%3Daad and include the token
		// returned by the injected TokenCredential unchanged.
		assert.Contains(t, authHeader, "type%3Daad")
		assert.Contains(t, authHeader, "test-token")
	}

	assert.Equal(t, []string{"https://cosmos.example/.default", "https://cosmos.example/.default"}, credential.scopes)
}

func TestCosmosDBAccountKeyPrecedesBearerCredential(t *testing.T) {
	credential := &recordingTokenCredential{token: "test-token"}
	client := &cosmosDBClient{
		credential:          credential,
		cosmosDBResourceURL: "https://cosmos.azure.com",
		dataEndpoint:        "https://test.documents.azure.com",
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
			name: "private cloud rejects overrides",
			triggerMetadata: map[string]string{
				"cloud":                   "Private",
				"cosmosDBResourceURL":     "https://cosmos.private.example",
				"activeDirectoryEndpoint": "https://login.private.example/",
			},
			wantError: "metadata is not supported",
		},
		{
			name: "private cloud requires cosmos resource",
			triggerMetadata: map[string]string{
				"cloud":                   "Private",
				"activeDirectoryEndpoint": "https://login.private.example/",
			},
			wantError: "metadata is not supported",
		},
		{
			name: "private cloud requires authority endpoint",
			triggerMetadata: map[string]string{
				"cloud":               "Private",
				"cosmosDBResourceURL": "https://cosmos.private.example",
			},
			wantError: "metadata is not supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, err := resolveCosmosDBEnvironment(tt.triggerMetadata)
			if tt.wantError != "" {
				assert.ErrorContains(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantResourceURL, env.ResourceIdentifiers.CosmosDB)
			assert.Equal(t, tt.wantAuthorityHost, env.ActiveDirectoryEndpoint)
			assert.Equal(t, tt.wantDisableInstanceDiscovery, env.private)
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
		assert.NoError(t, r.ParseForm())
		assert.Equal(t, "client-secret", r.Form.Get("client_secret"))
		assert.ElementsMatch(t, []string{"https://cosmos.private.example/.default", "openid", "offline_access", "profile"}, strings.Fields(r.Form.Get("scope")))
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
	setCosmosDBEnvironmentFile(t, map[string]interface{}{
		"cosmosDBDNSSuffix":       "documents.private.example",
		"activeDirectoryEndpoint": authorityServer.URL + "/",
		"resourceIdentifiers":     map[string]string{"cosmosDB": "https://cosmos.private.example"},
	})
	env, err := resolveCosmosDBEnvironment(map[string]string{"cloud": "Private"})
	require.NoError(t, err)

	credential, resourceURL, err := newCosmosDBTokenCredential(
		meta,
		env,
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
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		httpClient:       server.Client(),
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

	totalLag, activePartitionCount, _, err := estimateCosmosDBLegacyLag(client)
	assert.NoError(t, err)
	// Only partition 6 has lag; partition 3 is caught up; metadata doc is filtered
	assert.Equal(t, int64(89), totalLag)
	assert.Equal(t, int64(1), activePartitionCount)
}

func TestCosmosDBLeaseParsingJavaFormat(t *testing.T) {
	// Realistic Java SDK lease documents: no version field, no FeedRange/Mode/properties.
	// The scaler must parse these identically.
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		httpClient:       server.Client(),
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

	totalLag, activePartitionCount, _, err := estimateCosmosDBLegacyLag(client)
	assert.NoError(t, err)
	// Both partitions 2 and 5 have lag; lock doc is filtered out
	assert.Equal(t, int64(252), totalLag)
	assert.Equal(t, int64(2), activePartitionCount)
}

func TestCosmosDBLeaseParsingMixedFormats(t *testing.T) {
	// Edge case: lease container might contain docs from both SDKs (e.g. during migration).
	// The scaler should handle this gracefully since it only reads common fields.
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		httpClient:       server.Client(),
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

	totalLag, _, _, err := estimateCosmosDBLegacyLag(client)
	assert.NoError(t, err)
	assert.Equal(t, int64(302), totalLag)
}

func TestCosmosDBLeaseParsingEPKBasedDotNet(t *testing.T) {
	testCosmosDBEPKLeaseFixture(t, cosmosDBDotNetV1Leases, []string{`"750"`, `"320"`})
}

func TestCosmosDBLeaseParsingEPKBasedJava(t *testing.T) {
	testCosmosDBEPKLeaseFixture(t, cosmosDBJavaV1Leases, []string{`"500"`, `"200"`})
}

func TestCosmosDBLagEstimation(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		httpClient:       server.Client(),
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

	totalLag, _, _, err := estimateCosmosDBLegacyLag(client)
	assert.NoError(t, err)
	assert.Equal(t, int64(51), totalLag)
}

func TestCosmosDBLagEstimationEmptyLeases(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"Documents": []map[string]interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &cosmosDBClient{
		httpClient:       server.Client(),
		leaseEndpoint:    server.URL,
		leaseKey:         "dGVzdGtleQ==",
		leaseDatabaseID:  "testdb",
		leaseContainerID: "leases",
		processorName:    "testprocessor",
	}

	totalLag, activePartitionCount, _, err := estimateCosmosDBLegacyLag(client)
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
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		httpClient:       server.Client(),
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

	totalLag, activePartitionCount, _, err := estimateCosmosDBLegacyLag(client)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), totalLag, "backlog behind a never-checkpointed lease must be detected, not silently reported as 0")
	assert.Equal(t, int64(1), activePartitionCount)
}

func TestCosmosDBLagEstimationManyBootstrappingPartitionsCollapseForCap(t *testing.T) {
	// Simulates a container with 3 partitions all bootstrapping simultaneously (never
	// checkpointed), each reporting real backlog. Without dampening, activePartitionCount
	// would be 3, letting the scale-out cap request up to 3 replicas purely from a one-time
	// historical backlog visible on first bootstrap, rather than genuine ongoing accumulation.
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		httpClient:       server.Client(),
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

	totalLag, activePartitionCount, _, err := estimateCosmosDBLegacyLag(client)
	assert.NoError(t, err)
	assert.Equal(t, int64(30), totalLag, "raw backlog across all 3 bootstrapping partitions must still be fully detected")
	assert.Equal(t, int64(1), activePartitionCount, "never-checkpointed partitions must collapse to at most one partition's worth for scale-out capping")
}

func TestCosmosDBLagEstimationMixedCheckpointedAndBootstrappingPartitions(t *testing.T) {
	// 2 partitions already checkpointed with real ongoing lag, plus 3 more partitions
	// simultaneously bootstrapping (never checkpointed). Checkpointed partitions must still
	// count individually toward the scale-out cap; the bootstrapping group counts as only one more.
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		httpClient:       server.Client(),
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

	totalLag, activePartitionCount, _, err := estimateCosmosDBLegacyLag(client)
	assert.NoError(t, err)
	assert.Equal(t, int64(50), totalLag, "all 5 partitions contribute their real lag to the total")
	assert.Equal(t, int64(3), activePartitionCount, "2 checkpointed partitions count individually, plus 1 for the collapsed bootstrapping group")
}

func TestCosmosDBLagEstimationAllPartitionsLagging(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		httpClient:       server.Client(),
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

	totalLag, activePartitionCount, _, err := estimateCosmosDBLegacyLag(client)
	assert.NoError(t, err)
	assert.Equal(t, int64(202), totalLag)
	assert.Equal(t, int64(2), activePartitionCount)
}

func TestCosmosDBLagEstimationRecoversPartitionSplit(t *testing.T) {
	leaseQueryCount := 0
	changeFeedCalls := map[string]int{}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		httpClient:       server.Client(),
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

	totalLag, activePartitionCount, splitRecoveryRequired, err := estimateCosmosDBLegacyLag(client)
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
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
					httpClient:       server.Client(),
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
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		httpClient:       server.Client(),
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

	totalLag, activePartitionCount, splitRecoveryRequired, err := estimateCosmosDBLegacyLag(client)
	assert.ErrorContains(t, err, `status 410 and substatus "1008"`)
	assert.Zero(t, totalLag)
	assert.Zero(t, activePartitionCount)
	assert.False(t, splitRecoveryRequired)
	assert.Equal(t, 1, leaseQueryCount)
}

func TestCosmosDBLagEstimationPartitionError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		httpClient:       server.Client(),
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

	totalLag, activePartitionCount, splitRecoveryRequired, err := estimateCosmosDBLegacyLag(client)
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
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
					httpClient:       server.Client(),
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

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			httpClient:       server.Client(),
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
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			httpClient:       server.Client(),
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
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			httpClient:       server.Client(),
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
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
			httpClient:       server.Client(),
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
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/dbs/testdb/colls/leases/docs" {
			bodyBytes, _ := io.ReadAll(r.Body)
			capturedBody = string(bodyBytes)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"Documents":[]}`))
		}
	}))
	defer server.Close()

	client := &cosmosDBClient{
		httpClient:       server.Client(),
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

func cosmosDBCapacityConfig(endpoint string) *scalersconfig.ScalerConfig {
	return &scalersconfig.ScalerConfig{
		ScalableObjectName:      "cosmos-capacity",
		ScalableObjectNamespace: "default",
		ScalableObjectType:      "ScaledObject",
		TriggerMetadata: map[string]string{
			"connection": fmt.Sprintf("AccountEndpoint=%s;AccountKey=dGVzdGtleQ==", endpoint),
			"databaseId": "testdb", "containerId": "data",
			"leaseDatabaseId": "testdb", "leaseContainerId": "leases",
			"processorName": "testprocessor",
		},
	}
}

func TestCosmosDBCapacityMetadata(t *testing.T) {
	tests := []struct {
		name       string
		capacity   string
		ha         string
		metricType v2.MetricTargetType
		wantError  string
	}{
		{name: "legacy defaults"},
		{name: "legacy Value", metricType: v2.ValueMetricType},
		{name: "capacity", capacity: "4"},
		{name: "HA", capacity: "4", ha: "true"},
		{name: "disabled HA", ha: "false"},
		{name: "capacity Value", capacity: "4", metricType: v2.ValueMetricType, wantError: "requires metricType AverageValue"},
		{name: "zero", capacity: "0", wantError: "must be greater than zero"},
		{name: "negative", capacity: "-1", wantError: "must be greater than zero"},
		{name: "fractional", capacity: "1.5", wantError: "error parsing"},
		{name: "overflow", capacity: "9223372036854775808", wantError: "error parsing"},
		{name: "invalid HA", ha: "sometimes", wantError: "error parsing"},
		{name: "HA without capacity", ha: "true", wantError: "requires maxActiveLeasesPerReplica"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := cosmosDBCapacityConfig("https://test.documents.azure.com")
			config.MetricType = tt.metricType
			if tt.capacity != "" {
				config.TriggerMetadata["maxActiveLeasesPerReplica"] = tt.capacity
			}
			if tt.ha != "" {
				config.TriggerMetadata["enableHighAvailability"] = tt.ha
			}
			scaler, err := NewAzureCosmosDBScaler(config)
			if tt.wantError != "" {
				require.ErrorContains(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, scaler.Close(context.Background())) })
			spec := scaler.GetMetricSpecForScaling(context.Background())
			require.Len(t, spec, 1)
			wantTarget := int64(100)
			if tt.capacity != "" {
				wantTarget = 1
			}
			if tt.metricType == v2.ValueMetricType {
				assert.Equal(t, wantTarget, spec[0].External.Target.Value.Value())
			} else {
				assert.Equal(t, wantTarget, spec[0].External.Target.AverageValue.Value())
			}
		})
	}
	config := cosmosDBCapacityConfig("https://test.documents.azure.com")
	config.TriggerMetadata["maxActiveLeasesPerReplica"] = ""
	_, err := NewAzureCosmosDBScaler(config)
	require.Error(t, err)
}

type cosmosDBCapacityScenario struct {
	activeLeases int
	totalLeases  int
	lagPerLease  int64
	checkpointed bool
}

func newCosmosDBCapacityServer(t *testing.T, scenario cosmosDBCapacityScenario) *httptest.Server {
	t.Helper()
	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dbs/testdb/colls/leases/docs":
			docs := make([]map[string]interface{}, scenario.totalLeases)
			for i := range docs {
				docs[i] = map[string]interface{}{"id": fmt.Sprintf("lease%d", i), "LeaseToken": strconv.Itoa(i)}
				if scenario.checkpointed {
					docs[i]["ContinuationToken"] = `"0"`
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"Documents": docs})
		case "/dbs/testdb/colls/data/docs":
			id, err := strconv.Atoi(r.Header.Get("x-ms-documentdb-partitionkeyrangeid"))
			assert.NoError(t, err)
			if id >= scenario.activeLeases {
				w.WriteHeader(http.StatusNotModified)
				return
			}
			w.Header().Set("x-ms-session-token", fmt.Sprintf("%d:0#%d", id, scenario.lagPerLease))
			_, _ = w.Write([]byte(`{"Documents":[{"id":"item","_lsn":1}]}`))
		default:
			t.Errorf("unexpected request %s", r.URL)
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
}

func TestCosmosDBLeaseCapacityScaling(t *testing.T) {
	tests := []struct {
		name       string
		scenario   cosmosDBCapacityScenario
		capacity   int64
		ha         bool
		activation int64
		wantMetric int64
		wantActive bool
	}{
		{name: "many leases little work", scenario: cosmosDBCapacityScenario{20, 20, 1, true}, capacity: 4, wantMetric: 5, wantActive: true},
		{name: "round up lease count", scenario: cosmosDBCapacityScenario{5, 5, 1, true}, capacity: 4, wantMetric: 2, wantActive: true},
		{name: "lag dominates", scenario: cosmosDBCapacityScenario{10, 10, 70, true}, capacity: 4, wantMetric: 7, wantActive: true},
		{name: "lag rounds up", scenario: cosmosDBCapacityScenario{3, 3, 34, true}, capacity: 4, wantMetric: 2, wantActive: true},
		{name: "nonHA active cap", scenario: cosmosDBCapacityScenario{2, 20, 1000, true}, capacity: 4, wantMetric: 2, wantActive: true},
		{name: "HA total cap", scenario: cosmosDBCapacityScenario{2, 20, 2000, true}, capacity: 4, ha: true, wantMetric: 20, wantActive: true},
		{name: "HA caught up", scenario: cosmosDBCapacityScenario{0, 9, 0, true}, capacity: 4, ha: true, wantMetric: 3, wantActive: true},
		{name: "nonHA caught up", scenario: cosmosDBCapacityScenario{0, 9, 0, true}, capacity: 4},
		{name: "HA empty", capacity: 4, ha: true},
		{name: "nonHA empty", capacity: 4},
		{name: "capacity never checkpointed", scenario: cosmosDBCapacityScenario{20, 20, 1, false}, capacity: 4, wantMetric: 5, wantActive: true},
		{name: "capacity one per replica", scenario: cosmosDBCapacityScenario{6, 6, 1, true}, capacity: 1, wantMetric: 6, wantActive: true},
		{name: "activation threshold", scenario: cosmosDBCapacityScenario{5, 5, 1, true}, capacity: 4, activation: 5, wantMetric: 2},
		{name: "HA bypasses activation", scenario: cosmosDBCapacityScenario{0, 5, 0, true}, capacity: 4, ha: true, activation: 5, wantMetric: 2, wantActive: true},
		{name: "legacy bootstrap cap", scenario: cosmosDBCapacityScenario{20, 20, 100, false}, wantMetric: 100, wantActive: true},
		{name: "legacy checkpointed cap", scenario: cosmosDBCapacityScenario{2, 20, 1000, true}, wantMetric: 200, wantActive: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newCosmosDBCapacityServer(t, tt.scenario)
			defer server.Close()
			client := newCosmosDBEPKTestClient(server)
			state, err := client.estimateLag(context.Background())
			require.NoError(t, err)
			assert.EqualValues(t, tt.scenario.activeLeases, state.activeLeases)
			assert.EqualValues(t, tt.scenario.totalLeases, state.totalLeases)
			scaler := &azureCosmosDBScaler{
				cosmosClient: client,
				metricType:   v2.AverageValueMetricType,
				metadata: &azureCosmosDBMetadata{
					Threshold: 100, ActivationThreshold: tt.activation,
					MaxActiveLeasesPerReplica: tt.capacity, EnableHighAvailability: tt.ha,
				},
				logger: logr.Discard(),
			}
			metrics, active, err := scaler.GetMetricsAndActivity(context.Background(), "test")
			require.NoError(t, err)
			require.Len(t, metrics, 1)
			assert.Equal(t, tt.wantMetric, metrics[0].Value.Value())
			assert.Equal(t, tt.wantActive, active)
		})
	}
}

func TestCosmosDBLeaseCountsAcrossPages(t *testing.T) {
	requests := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dbs/testdb/colls/leases/docs":
			requests++
			switch requests {
			case 1:
				assert.Empty(t, r.Header.Get("x-ms-continuation"))
				w.Header().Set("x-ms-continuation", "page2")
				_, _ = w.Write([]byte(`{"Documents":[{"id":"a","LeaseToken":"0"},{"id":"metadata"}]}`))
			case 2:
				assert.Equal(t, "page2", r.Header.Get("x-ms-continuation"))
				w.Header().Set("x-ms-continuation", "page3")
				_, _ = w.Write([]byte(`{"Documents":[]}`))
			case 3:
				assert.Equal(t, "page3", r.Header.Get("x-ms-continuation"))
				_, _ = w.Write([]byte(`{"Documents":[{"id":"b","LeaseToken":"1","ContinuationToken":"\"5\""},{"id":"c","LeaseToken":"2"}]}`))
			default:
				t.Errorf("unexpected page %d", requests)
				w.WriteHeader(http.StatusBadRequest)
			}
		case "/dbs/testdb/colls/data/docs":
			if r.Header.Get("x-ms-documentdb-partitionkeyrangeid") == "1" {
				w.WriteHeader(http.StatusNotModified)
			} else {
				w.Header().Set("x-ms-session-token", "0:0#1")
				_, _ = w.Write([]byte(`{"Documents":[{"_lsn":1}]}`))
			}
		}
	}))
	defer server.Close()
	state, err := newCosmosDBEPKTestClient(server).estimateLag(context.Background())
	require.NoError(t, err)
	assert.Equal(t, cosmosDBLeaseState{totalLag: 2, activeLeases: 2, totalLeases: 3, legacyActivePartitions: 1}, state)
	assert.Equal(t, 3, requests)
}

func TestCosmosDBCapacityArithmeticLimits(t *testing.T) {
	for _, tt := range []struct{ value, divisor, want int64 }{
		{0, 4, 0}, {4, 4, 1}, {5, 4, 2},
		{math.MaxInt64, 1, math.MaxInt64},
		{math.MaxInt64, 2, math.MaxInt64/2 + 1},
		{math.MaxInt64, math.MaxInt64, 1},
	} {
		assert.Equal(t, tt.want, cosmosDBCeilDivide(tt.value, tt.divisor))
	}
	server := newCosmosDBCapacityServer(t, cosmosDBCapacityScenario{3, 3, math.MaxInt64, true})
	defer server.Close()
	state, err := newCosmosDBEPKTestClient(server).estimateLag(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(math.MaxInt64), state.totalLag)
	assert.Equal(t, int64(3), state.activeLeases)
}

func TestCosmosDBLeaseCapacityFallback(t *testing.T) {
	var fail bool
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if r.URL.Path == "/dbs/testdb/colls/leases/docs" {
			_, _ = w.Write([]byte(`{"Documents":[{"id":"a","LeaseToken":"0"},{"id":"b","LeaseToken":"1"}]}`))
			return
		}
		if r.Header.Get("x-ms-documentdb-partitionkeyrangeid") == "1" {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("x-ms-session-token", "0:0#250")
		_, _ = w.Write([]byte(`{"Documents":[{"_lsn":1}]}`))
	}))
	defer server.Close()
	config := cosmosDBCapacityConfig(cosmosDBTestEndpoint)
	config.TriggerMetadata["maxActiveLeasesPerReplica"] = "1"
	config.TriggerMetadata["enableHighAvailability"] = "true"
	scaler, err := NewAzureCosmosDBScaler(config)
	require.NoError(t, err)
	scaler.(*azureCosmosDBScaler).cosmosClient.httpClient.Transport = cosmosDBTestTransport(server)
	t.Cleanup(func() {
		require.NoError(t, scaler.Close(context.Background()))
	})
	specs := scaler.GetMetricSpecForScaling(context.Background())
	require.Len(t, specs, 1)
	metricName := specs[0].External.Metric.Name
	metrics, active, err := scaler.GetMetricsAndActivity(context.Background(), metricName)
	require.NoError(t, err)
	assert.True(t, active)
	require.Len(t, metrics, 1)
	assert.Equal(t, int64(2), metrics[0].Value.Value())
	fail = true
	metrics, active, metricErr := scaler.GetMetricsAndActivity(context.Background(), metricName)
	require.Error(t, metricErr)
	assert.Empty(t, metrics)
	assert.False(t, active)
	scheme := runtime.NewScheme()
	require.NoError(t, kedav1alpha1.AddToScheme(scheme))
	object := &kedav1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{Name: "cosmos-capacity", Namespace: "default"},
		Spec: kedav1alpha1.ScaledObjectSpec{Fallback: &kedav1alpha1.Fallback{
			FailureThreshold: 1, Replicas: 3, Behavior: kedav1alpha1.FallbackBehaviorStatic,
		}},
	}
	kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(object).WithStatusSubresource(object).Build()
	handler := fallback.ScaledObjectHandler{Ctx: context.Background(), KubeClient: kubeClient, UpdateLock: &sync.RWMutex{}, ScaledObject: object}
	_, fallingBack, err := fallback.GetMetricsWithFallback(handler, metrics, metricErr, metricName, specs[0])
	require.Error(t, err)
	assert.False(t, fallingBack)
	fallbackMetrics, fallingBack, err := fallback.GetMetricsWithFallback(handler, metrics, metricErr, metricName, specs[0])
	require.NoError(t, err)
	assert.True(t, fallingBack)
	require.Len(t, fallbackMetrics, 1)
	assert.Equal(t, int64(3), fallbackMetrics[0].Value.Value())

	fail = false
	metrics, active, err = scaler.GetMetricsAndActivity(context.Background(), metricName)
	require.NoError(t, err)
	assert.True(t, active)
	require.Len(t, metrics, 1)
	assert.Equal(t, int64(2), metrics[0].Value.Value())
}

const cosmosDBTestEPKBoundary = "1FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"

func TestCosmosDBPaginationProgress(t *testing.T) {
	for _, resource := range []string{"leases", "ranges"} {
		for _, repeated := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/repeated=%t", resource, repeated), func(t *testing.T) {
				requests := 0
				server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests++
					if requests > 2 {
						t.Error("pagination requested a third page")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					if requests == 1 {
						assert.Empty(t, r.Header.Get("x-ms-continuation"))
						w.Header().Set("x-ms-continuation", "next-page")
					} else {
						assert.Equal(t, "next-page", r.Header.Get("x-ms-continuation"))
						if repeated {
							w.Header().Set("x-ms-continuation", "next-page")
						}
					}
					id := fmt.Sprint(requests)
					if resource == "leases" {
						assert.Equal(t, "/dbs/testdb/colls/leases/docs", r.URL.Path)
						assert.NoError(t, json.NewEncoder(w).Encode(map[string]interface{}{
							"Documents": []map[string]string{{"id": id, "LeaseToken": id}},
						}))
					} else {
						assert.Equal(t, "/dbs/testdb/colls/data/pkranges", r.URL.Path)
						assert.NoError(t, json.NewEncoder(w).Encode(map[string]interface{}{
							"PartitionKeyRanges": []map[string]string{{
								"id": id, "minInclusive": []string{"", "80"}[requests-1], "maxExclusive": []string{"80", "FF"}[requests-1],
							}},
						}))
					}
				}))
				defer server.Close()
				client := newCosmosDBEPKTestClient(server)
				if resource == "leases" {
					leases, err := client.queryLeases(context.Background())
					if repeated {
						assert.EqualError(t, err, "lease query returned an unchanged continuation token")
						assert.Nil(t, leases)
					} else {
						require.NoError(t, err)
						require.Len(t, leases, 2)
						assert.Equal(t, "1", leases[0].ID)
						assert.Equal(t, "2", leases[1].ID)
					}
				} else {
					ranges, err := client.readPartitionKeyRanges(context.Background())
					if repeated {
						assert.EqualError(t, err, "partition key range read returned an unchanged continuation token")
						assert.Nil(t, ranges)
					} else {
						require.NoError(t, err)
						require.Len(t, ranges, 2)
						assert.Equal(t, "1", ranges[0].ID)
						assert.Equal(t, "2", ranges[1].ID)
					}
				}
				assert.Equal(t, 2, requests)
			})
		}
	}
}

func newCosmosDBEPKTestClient(server *httptest.Server) *cosmosDBClient {
	return &cosmosDBClient{
		httpClient:       server.Client(),
		dataEndpoint:     server.URL,
		dataKey:          "dGVzdGtleQ==",
		leaseEndpoint:    server.URL,
		leaseKey:         "bGVhc2VrZXk=",
		databaseID:       "testdb",
		containerID:      "data",
		leaseDatabaseID:  "testdb",
		leaseContainerID: "leases",
		processorName:    "testprocessor",
		logger:           logr.Discard(),
	}
}

func testCosmosDBEPKLeaseFixture(t *testing.T, fixture string, etags []string) {
	t.Helper()
	// Fixtures follow DocumentServiceLeaseCoreEpk (.NET) and ServiceItemLeaseV1 (Java).
	// Java tokens contain ChangeFeedStateV1 with a nested FeedRangeCompositeContinuationImpl.
	for _, subrange := range []bool{false, true} {
		t.Run(fmt.Sprintf("subrange=%t", subrange), func(t *testing.T) {
			rangeRequests, feedRequests := 0, 0
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/dbs/testdb/colls/leases/docs":
					assert.NoError(t, json.NewEncoder(w).Encode(json.RawMessage(fixture)))
				case "/dbs/testdb/colls/data/pkranges":
					rangeRequests++
					assert.Equal(t, http.MethodGet, r.Method)
					assert.Equal(t, cosmosDBRestAPIVersion, r.Header.Get("x-ms-version"))
					auth, authErr := generateCosmosDBAuthToken(http.MethodGet, "pkranges", "dbs/testdb/colls/data", r.Header.Get("x-ms-date"), "dGVzdGtleQ==")
					assert.NoError(t, authErr)
					assert.Equal(t, auth, r.Header.Get("Authorization"))
					switch {
					case subrange:
						_, _ = w.Write([]byte(`{"PartitionKeyRanges":[{"id":"7","minInclusive":"","maxExclusive":"FF"}]}`))
					case rangeRequests == 1:
						assert.Empty(t, r.Header.Get("x-ms-continuation"))
						w.Header().Set("x-ms-continuation", "next-page")
						assert.NoError(t, json.NewEncoder(w).Encode(map[string]interface{}{
							"PartitionKeyRanges": []map[string]string{{"id": "6", "minInclusive": "", "maxExclusive": cosmosDBTestEPKBoundary}},
						}))
					default:
						assert.Equal(t, "next-page", r.Header.Get("x-ms-continuation"))
						assert.NoError(t, json.NewEncoder(w).Encode(map[string]interface{}{
							"PartitionKeyRanges": []map[string]string{{"id": "9", "minInclusive": cosmosDBTestEPKBoundary, "maxExclusive": "FF"}},
						}))
					}
				case "/dbs/testdb/colls/data/docs":
					index := feedRequests
					feedRequests++
					if !assert.Less(t, index, len(etags)) {
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					assert.Equal(t, etags[index], r.Header.Get("If-None-Match"))
					assert.Equal(t, "Incremental feed", r.Header.Get("A-IM"))
					assert.Equal(t, "1", r.Header.Get("x-ms-max-item-count"))
					if subrange {
						assert.Equal(t, "7", r.Header.Get("x-ms-documentdb-partitionkeyrangeid"))
						assert.Equal(t, "EffectivePartitionKeyRange", r.Header.Get("x-ms-read-key-type"))
						assert.Contains(t, r.Header, http.CanonicalHeaderKey("x-ms-start-epk"))
						assert.Equal(t, []string{"", cosmosDBTestEPKBoundary}[index], r.Header.Get("x-ms-start-epk"))
						assert.Equal(t, []string{cosmosDBTestEPKBoundary, "FF"}[index], r.Header.Get("x-ms-end-epk"))
					} else {
						assert.Equal(t, []string{"6", "9"}[index], r.Header.Get("x-ms-documentdb-partitionkeyrangeid"))
						for _, header := range []string{"x-ms-read-key-type", "x-ms-start-epk", "x-ms-end-epk"} {
							assert.NotContains(t, r.Header, http.CanonicalHeaderKey(header))
						}
					}
					if index == 0 {
						w.Header().Set("x-ms-session-token", "6:0#900")
						_, _ = w.Write([]byte(`{"Documents":[{"id":"doc1","_lsn":751}]}`))
					} else {
						w.WriteHeader(http.StatusNotModified)
					}
				default:
					t.Errorf("unexpected request: %s", r.URL)
					w.WriteHeader(http.StatusBadRequest)
				}
			}))
			defer server.Close()
			lag, count, split, err := estimateCosmosDBLegacyLag(newCosmosDBEPKTestClient(server))
			require.NoError(t, err)
			assert.Equal(t, int64(150), lag)
			assert.Equal(t, int64(1), count)
			assert.False(t, split)
			assert.Equal(t, 2, feedRequests)
			if subrange {
				assert.Equal(t, 1, rangeRequests)
			} else {
				assert.Equal(t, 2, rangeRequests)
			}
		})
	}
}

func TestCosmosDBEPKRangeResolution(t *testing.T) {
	ranges := []cosmosDBPartitionKeyRange{
		{ID: "4", Range: cosmosDBEPKRange{Min: "", Max: "40"}},
		{ID: "5", Range: cosmosDBEPKRange{Min: "40", Max: "80"}},
		{ID: "6", Range: cosmosDBEPKRange{Min: "80", Max: "FF"}},
	}
	tests := []struct {
		name  string
		min   string
		max   string
		id    string
		split bool
	}{
		{name: "exact first", min: "", max: "40", id: "4"},
		{name: "exact adjacent", min: "40", max: "80", id: "5"},
		{name: "exact last", min: "80", max: "FF", id: "6"},
		{name: "interior subrange", min: "50", max: "60", id: "5"},
		{name: "lower subrange", min: "", max: "20", id: "4"},
		{name: "upper subrange", min: "90", max: "FF", id: "6"},
		{name: "full split", min: "", max: "FF", split: true},
		{name: "partial split", min: "30", max: "50", split: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, split, err := resolveCosmosDBEPKRange(cosmosDBEPKRange{Min: tt.min, Max: tt.max}, ranges)
			require.NoError(t, err)
			assert.Equal(t, tt.split, split)
			if tt.split {
				assert.Nil(t, match)
			} else {
				require.NotNil(t, match)
				assert.Equal(t, tt.id, match.ID)
			}
		})
	}
	for _, epkRange := range []cosmosDBEPKRange{{Min: "", Max: "20"}, {Min: "", Max: "60"}, {Min: "60", Max: "FF"}} {
		_, _, err := resolveCosmosDBEPKRange(epkRange, ranges[1:2])
		assert.ErrorContains(t, err, "no physical partition contains")
	}
}

func TestCosmosDBEPKSplitRecovery(t *testing.T) {
	const parent = `{"id":"parent","version":1,"LeaseToken":"-80","FeedRange":{"Range":{"min":"","max":"80"}},"ContinuationToken":"\"100\""}`
	const children = `{"id":"child1","version":1,"LeaseToken":"-40","FeedRange":{"Range":{"min":"","max":"40"}},"ContinuationToken":"\"100\""},
		{"id":"child2","version":1,"LeaseToken":"40-80","FeedRange":{"Range":{"min":"40","max":"80"}},"ContinuationToken":"\"100\""}`
	tests := []struct {
		name         string
		refreshLease bool
		serverSplit  bool
	}{
		{name: "persistent overlap"},
		{name: "children replace stale parent", refreshLease: true},
		{name: "split after range lookup", refreshLease: true, serverSplit: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			leaseRequests, rangeRequests := 0, 0
			feedRequests := map[string]int{}
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/dbs/testdb/colls/leases/docs":
					leaseRequests++
					leases := parent
					if tt.refreshLease && leaseRequests > 1 {
						leases = children
					}
					assert.NoError(t, json.NewEncoder(w).Encode(json.RawMessage(
						fmt.Sprintf(`{"Documents":[%s,{"id":"legacy","LeaseToken":"9","ContinuationToken":"\"200\""}]}`, leases),
					)))
				case "/dbs/testdb/colls/data/pkranges":
					rangeRequests++
					if tt.serverSplit && rangeRequests == 1 {
						_, _ = w.Write([]byte(`{"PartitionKeyRanges":[{"id":"0","minInclusive":"","maxExclusive":"80"},{"id":"9","minInclusive":"80","maxExclusive":"FF"}]}`))
					} else {
						_, _ = w.Write([]byte(`{"PartitionKeyRanges":[{"id":"1","minInclusive":"","maxExclusive":"40"},{"id":"2","minInclusive":"40","maxExclusive":"80"},{"id":"9","minInclusive":"80","maxExclusive":"FF"}]}`))
					}
				case "/dbs/testdb/colls/data/docs":
					id := r.Header.Get("x-ms-documentdb-partitionkeyrangeid")
					feedRequests[id]++
					switch id {
					case "0":
						w.Header().Set(cosmosDBSubStatusHeader, cosmosDBPartitionKeyRangeGoneSubStatus)
						w.WriteHeader(http.StatusGone)
					case "1", "2":
						assert.Equal(t, `"100"`, r.Header.Get("If-None-Match"))
						w.WriteHeader(http.StatusNotModified)
					case "9":
						w.Header().Set("x-ms-session-token", "9:0#500")
						_, _ = w.Write([]byte(`{"Documents":[{"_lsn":201}]}`))
					default:
						t.Errorf("unexpected range ID %q", id)
						w.WriteHeader(http.StatusBadRequest)
					}
				default:
					t.Errorf("unexpected request: %s", r.URL)
					w.WriteHeader(http.StatusBadRequest)
				}
			}))
			defer server.Close()
			lag, count, split, err := estimateCosmosDBLegacyLag(newCosmosDBEPKTestClient(server))
			require.NoError(t, err)
			assert.Equal(t, int64(300), lag)
			assert.Equal(t, int64(1), count)
			assert.Equal(t, !tt.refreshLease, split)
			assert.Equal(t, 2, leaseRequests)
			assert.Equal(t, 2, rangeRequests)
			assert.Equal(t, 2, feedRequests["9"])
			assert.Zero(t, feedRequests["-80"])
			if tt.refreshLease {
				assert.Equal(t, 1, feedRequests["1"])
				assert.Equal(t, 1, feedRequests["2"])
			}
			if tt.serverSplit {
				assert.Equal(t, 1, feedRequests["0"])
			}
		})
	}
}

func TestCosmosDBEPKSplitActivatesProcessor(t *testing.T) {
	leaseRequests := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dbs/testdb/colls/leases/docs":
			leaseRequests++
			_, _ = w.Write([]byte(`{"Documents":[{"id":"parent","version":1,"LeaseToken":"-FF","feedRange":{"Range":{"min":"","max":"FF"}},"ContinuationToken":"\"100\""}]}`))
		case "/dbs/testdb/colls/data/pkranges":
			_, _ = w.Write([]byte(`{"PartitionKeyRanges":[{"id":"1","minInclusive":"","maxExclusive":"80"},{"id":"2","minInclusive":"80","maxExclusive":"FF"}]}`))
		default:
			t.Errorf("split parent must not read the feed: %s", r.URL)
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer server.Close()
	scaler := &azureCosmosDBScaler{
		cosmosClient: newCosmosDBEPKTestClient(server),
		metadata:     &azureCosmosDBMetadata{Threshold: 100, ActivationThreshold: 10},
		logger:       logr.Discard(),
	}
	metrics, active, err := scaler.GetMetricsAndActivity(context.Background(), "test-metric")
	require.NoError(t, err)
	require.Len(t, metrics, 1)
	assert.Equal(t, int64(11), metrics[0].Value.Value())
	assert.True(t, active)
	assert.Equal(t, 2, leaseRequests)
}

func TestCosmosDBEPKMalformedFeedRange(t *testing.T) {
	for _, field := range []string{"FeedRange", "feedRange"} {
		for _, feedRange := range []string{
			`null`, `{}`, `[]`, `"invalid"`, `{"min":"","max":"FF"}`, `{"PKRangeId":"0"}`,
			`{"Range":null}`, `{"Range":{"max":"FF"}}`, `{"Range":{"min":null,"max":"FF"}}`,
			`{"Range":{"min":"","max":null}}`, `{"Range":{"min":0,"max":"FF"}}`,
			`{"Range":{"min":"80","max":"40"}}`, `{"Range":{"min":"80","max":"80"}}`,
			`{"Range":{"min":"","max":""}}`, `{"Range":{"min":"","max":"FFFF"}}`,
			`{"Range":{"min":"GG","max":"FF"}}`, `{"Range":{"min":"","max":"ff"}}`,
			`{"Range":{"min":"","max":"FF","isMinInclusive":false}}`,
			`{"Range":{"min":"","max":"FF","isMaxInclusive":true}}`,
		} {
			t.Run(field+"/"+feedRange, func(t *testing.T) {
				server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					assert.Equal(t, "/dbs/testdb/colls/leases/docs", r.URL.Path)
					assert.NoError(t, json.NewEncoder(w).Encode(map[string]interface{}{
						"Documents": []map[string]interface{}{{
							"id": "bad", "version": 1, "LeaseToken": "-FF", field: json.RawMessage(feedRange),
						}},
					}))
				}))
				defer server.Close()
				_, _, _, err := estimateCosmosDBLegacyLag(newCosmosDBEPKTestClient(server))
				assert.ErrorContains(t, err, `invalid FeedRange for lease "bad"`)
			})
		}
	}
	for _, version := range []int{1, 2} {
		lease := leaseDocument{ID: "missing", Version: version, LeaseToken: "-FF"}
		require.Error(t, lease.parseFeedRange())
	}
}

func TestCosmosDBEPKJavaContinuation(t *testing.T) {
	const valid = `{"V":1,"Rid":"rid???","Mode":"INCREMENTAL","StartFrom":{"Type":"BEGINNING"},"Continuation":{"V":1,"Rid":"rid???","Continuation":[{"token":"\"500\"","range":{"min":"","max":"FF"}}],"Range":{"min":"","max":"FF"}}}`
	for _, encoding := range []*base64.Encoding{base64.URLEncoding, base64.StdEncoding, base64.RawURLEncoding, base64.RawStdEncoding} {
		encoded := encoding.EncodeToString([]byte(valid))
		assert.True(t, strings.Contains(encoded, "_") || strings.Contains(encoded, "/"))
		token, err := decodeCosmosDBContinuation(encoded, cosmosDBEPKRange{Min: "20", Max: "40"})
		require.NoError(t, err)
		assert.Equal(t, cosmosDBChangeFeedStart{etag: `"500"`, checkpointed: true}, token)
	}

	for _, token := range []string{"", `"750"`} {
		actual, err := decodeCosmosDBContinuation(token, cosmosDBEPKRange{Min: "", Max: "FF"})
		require.NoError(t, err)
		assert.Equal(t, cosmosDBChangeFeedStart{etag: token, checkpointed: token != ""}, actual)
	}
	tests := []string{
		`{`, `null`, `{}`, `{"V":"1"}`,
		strings.Replace(valid, `"V":1`, `"V":2`, 1),
		strings.ReplaceAll(valid, `"V":1`, `"V":2`),
		strings.Replace(valid, `"INCREMENTAL"`, `"FULL_FIDELITY"`, 1),
		strings.Replace(valid, `"token":"\"500\""`, `"token":""`, 1),
		strings.Replace(valid, `"token":"\"500\""`, `"token":42`, 1),
		strings.Replace(valid, `"max":"FF"`, `"max":"80"`, 1),
		strings.Replace(valid, `"min":""`, `"min":"80"`, 1),
		strings.Replace(valid, `"range":{"min":"","max":"FF"}`, `"range":{}`, 1),
		`{"V":1,"Mode":"INCREMENTAL","Continuation":{"V":1,"Continuation":[]}}`,
		`{"V":1,"Mode":"INCREMENTAL","Continuation":{"V":1,"Continuation":[{},{}]}}`,
		`{"V":1,"Mode":"INCREMENTAL","StartFrom":{"Type":"NOW"}}`,
	}
	for _, startFrom := range []string{
		`null`, `{}`, `{"Type":"UNSUPPORTED"}`, `{"Type":42}`,
		`{"Type":"POINT_IN_TIME"}`, `{"Type":"POINT_IN_TIME","PointInTimeMs":null}`,
		`{"Type":"POINT_IN_TIME","PointInTimeMs":"1767225600000"}`,
		`{"Type":"POINT_IN_TIME","PointInTimeMs":1.5}`,
		`{"Type":"POINT_IN_TIME","PointInTimeMs":9223372036854775807}`,
		`{"Type":"POINT_IN_TIME","PointInTimeMs":-62135596800001}`,
		`{"Type":"LEASE"}`, `{"Type":"LEASE","Etag":""}`, `{"Type":"LEASE","Etag":null}`,
		`{"Type":"LEASE","Etag":"\"100\""}`,
		`{"Type":"LEASE","Etag":"\"100\"","Range":{"min":"","max":"80"}}`,
	} {
		for _, token := range []string{`"\"500\""`, `null`} {
			state := strings.Replace(valid, `{"Type":"BEGINNING"}`, startFrom, 1)
			tests = append(tests, strings.Replace(state, `"token":"\"500\""`, `"token":`+token, 1))
		}
	}
	for _, state := range tests {
		t.Run(state, func(t *testing.T) {
			_, err := decodeCosmosDBContinuation(base64.URLEncoding.EncodeToString([]byte(state)), cosmosDBEPKRange{Min: "", Max: "FF"})
			require.Error(t, err)
		})
	}
	_, err := decodeCosmosDBContinuation("not-base64!", cosmosDBEPKRange{Min: "", Max: "FF"})
	assert.ErrorContains(t, err, "invalid Java change feed continuation encoding")
}

func TestCosmosDBJavaInitialLeaseState(t *testing.T) {
	tests := []struct {
		name              string
		startFrom         string
		token             string
		noContinuation    bool
		wantETag          string
		wantModifiedSince string
		checkpointed      bool
	}{
		{name: "migrated beginning", startFrom: `{"Type":"BEGINNING"}`, token: `null`},
		{name: "migrated now", startFrom: `{"Type":"NOW"}`, token: `null`, wantETag: "*"},
		{name: "migrated point in time", startFrom: `{"Type":"POINT_IN_TIME","PointInTimeMs":1767225600123}`, token: `null`, wantModifiedSince: "Thu, 01 Jan 2026 00:00:00 GMT"},
		{name: "beginning time sentinel", startFrom: `{"Type":"POINT_IN_TIME","PointInTimeMs":-62135596800000}`, token: `null`},
		{name: "migrated configured lease", startFrom: `{"Type":"LEASE","Etag":"\"100\"","Range":{"min":"","max":"FF"}}`, token: `null`, wantETag: `"100"`, checkpointed: true},
		{name: "legacy checkpoint start", startFrom: `{"Type":"LEGACY_CHECKPOINT"}`, token: `null`},
		{name: "initial beginning", startFrom: `{"Type":"BEGINNING"}`, noContinuation: true},
		{name: "initial now", startFrom: `{"Type":"NOW"}`, noContinuation: true, wantETag: "*"},
		{name: "checkpoint replaces now", startFrom: `{"Type":"NOW"}`, token: `"\"500\""`, wantETag: `"500"`, checkpointed: true},
		{name: "checkpoint replaces configured lease", startFrom: `{"Type":"LEASE","Etag":"\"100\"","Range":{"min":"","max":"FF"}}`, token: `"\"500\""`, wantETag: `"500"`, checkpointed: true},
		{name: "checkpoint retains time filter", startFrom: `{"Type":"POINT_IN_TIME","PointInTimeMs":1767225600000}`, token: `"\"500\""`, wantETag: `"500"`, wantModifiedSince: "Thu, 01 Jan 2026 00:00:00 GMT", checkpointed: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docs := make([]map[string]interface{}, 0, 2)
			for _, bounds := range [][2]string{{"", "80"}, {"80", "FF"}} {
				leaseRange := fmt.Sprintf(`{"min":%q,"max":%q}`, bounds[0], bounds[1])
				stateRange := `"Range":` + leaseRange
				if !tt.noContinuation {
					stateRange = fmt.Sprintf(`"Continuation":{"V":1,"Rid":"q0oIALS6YvQ=","Continuation":[{"token":%s,"range":%s}],"Range":%s}`, tt.token, leaseRange, leaseRange)
				}
				state := fmt.Sprintf(`{"V":1,"Rid":"q0oIALS6YvQ=","Mode":"INCREMENTAL","StartFrom":%s,%s}`, tt.startFrom, stateRange)
				token := bounds[0] + "-" + bounds[1]
				docs = append(docs, map[string]interface{}{
					"id":      "testprocessormyaccount.documents.azure.com_q0oIAA==_q0oIALS6YvQ=.." + token,
					"version": 1, "LeaseToken": token, "Owner": "java-host",
					"feedRange":         json.RawMessage(`{"Range":` + leaseRange + `}`),
					"ContinuationToken": base64.URLEncoding.EncodeToString([]byte(state)),
				})
			}
			feedRequests := 0
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/dbs/testdb/colls/leases/docs":
					_ = json.NewEncoder(w).Encode(map[string]interface{}{"Documents": docs})
				case "/dbs/testdb/colls/data/pkranges":
					_, _ = w.Write([]byte(`{"PartitionKeyRanges":[{"id":"6","minInclusive":"","maxExclusive":"80"},{"id":"9","minInclusive":"80","maxExclusive":"FF"}]}`))
				case "/dbs/testdb/colls/data/docs":
					feedRequests++
					assert.Contains(t, []string{"6", "9"}, r.Header.Get("x-ms-documentdb-partitionkeyrangeid"))
					assert.Equal(t, tt.wantETag, r.Header.Get("If-None-Match"))
					assert.Equal(t, tt.wantModifiedSince, r.Header.Get("If-Modified-Since"))
					if tt.wantETag == "" {
						assert.NotContains(t, r.Header, "If-None-Match")
					}
					if tt.wantModifiedSince == "" {
						assert.NotContains(t, r.Header, "If-Modified-Since")
					}
					if tt.wantETag == "*" {
						w.WriteHeader(http.StatusNotModified)
						return
					}
					w.Header().Set("x-ms-session-token", "6:0#200")
					_, _ = w.Write([]byte(`{"Documents":[{"_lsn":1}]}`))
				default:
					t.Errorf("unexpected request: %s", r.URL)
					w.WriteHeader(http.StatusBadRequest)
				}
			}))
			defer server.Close()
			state, err := newCosmosDBEPKTestClient(server).estimateLag(context.Background())
			require.NoError(t, err)
			assert.Equal(t, int64(2), state.totalLeases)
			wantCap := int64(1)
			if tt.checkpointed {
				wantCap = 2
			}
			if tt.wantETag == "*" {
				wantCap = 0
				assert.Zero(t, state.activeLeases)
				assert.Zero(t, state.totalLag)
			} else {
				assert.Equal(t, int64(2), state.activeLeases)
				assert.Equal(t, int64(400), state.totalLag)
			}
			assert.Equal(t, wantCap, state.legacyActivePartitions)

			for _, capacity := range []string{"", "1"} {
				config := cosmosDBCapacityConfig(cosmosDBTestEndpoint)
				if capacity != "" {
					config.TriggerMetadata["maxActiveLeasesPerReplica"] = capacity
				}
				scaler, err := NewAzureCosmosDBScaler(config)
				require.NoError(t, err)
				scaler.(*azureCosmosDBScaler).cosmosClient.httpClient.Transport = cosmosDBTestTransport(server)
				spec := scaler.GetMetricSpecForScaling(context.Background())[0]
				metrics, active, err := scaler.GetMetricsAndActivity(context.Background(), spec.External.Metric.Name)
				require.NoError(t, err)
				assert.Equal(t, tt.wantETag != "*", active)
				require.Len(t, metrics, 1)
				wantMetric := wantCap * 100
				if capacity != "" {
					wantMetric = state.activeLeases
				}
				assert.Equal(t, wantMetric, metrics[0].Value.Value())
				require.NoError(t, scaler.Close(context.Background()))
			}
			assert.Equal(t, 6, feedRequests)
		})
	}
}

func TestCosmosDBLegacyContinuationIsUnchanged(t *testing.T) {
	for _, version := range []string{"", `"version":0,`} {
		t.Run(version, func(t *testing.T) {
			feedRequests := 0
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/dbs/testdb/colls/leases/docs":
					assert.NoError(t, json.NewEncoder(w).Encode(json.RawMessage(
						fmt.Sprintf(`{"Documents":[{"id":"legacy",%s"LeaseToken":"42","ContinuationToken":"opaque-server-token","FeedRange":{"Range":{"min":"","max":"FF"}}}]}`, version),
					)))
				case "/dbs/testdb/colls/data/docs":
					feedRequests++
					assert.Equal(t, "42", r.Header.Get("x-ms-documentdb-partitionkeyrangeid"))
					assert.Equal(t, "opaque-server-token", r.Header.Get("If-None-Match"))
					for _, header := range []string{"x-ms-read-key-type", "x-ms-start-epk", "x-ms-end-epk"} {
						assert.NotContains(t, r.Header, http.CanonicalHeaderKey(header))
					}
					w.WriteHeader(http.StatusNotModified)
				default:
					t.Errorf("legacy lease must not query physical ranges: %s", r.URL)
					w.WriteHeader(http.StatusBadRequest)
				}
			}))
			defer server.Close()
			lag, count, split, err := estimateCosmosDBLegacyLag(newCosmosDBEPKTestClient(server))
			require.NoError(t, err)
			assert.Zero(t, lag)
			assert.Zero(t, count)
			assert.False(t, split)
			assert.Equal(t, 1, feedRequests)
		})
	}
}

func TestCosmosDBEPKRangeReadErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "authorization", status: http.StatusForbidden},
		{name: "throttling", status: http.StatusTooManyRequests},
		{name: "invalid JSON", status: http.StatusOK, body: `{`},
		{name: "empty ranges", status: http.StatusOK, body: `{"PartitionKeyRanges":[]}`},
		{name: "missing min", status: http.StatusOK, body: `{"PartitionKeyRanges":[{"id":"0","maxExclusive":"FF"}]}`},
		{name: "missing ID", status: http.StatusOK, body: `{"PartitionKeyRanges":[{"minInclusive":"","maxExclusive":"FF"}]}`},
		{name: "reversed bounds", status: http.StatusOK, body: `{"PartitionKeyRanges":[{"id":"0","minInclusive":"FF","maxExclusive":""}]}`},
		{name: "missing overlap", status: http.StatusOK, body: `{"PartitionKeyRanges":[{"id":"0","minInclusive":"80","maxExclusive":"FF"}]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/dbs/testdb/colls/leases/docs":
					_, _ = w.Write([]byte(`{"Documents":[{"id":"epk","version":1,"LeaseToken":"-80","FeedRange":{"Range":{"min":"","max":"80"}}}]}`))
				case "/dbs/testdb/colls/data/pkranges":
					w.WriteHeader(tt.status)
					if tt.body == "{" {
						_, err := w.Write([]byte("{"))
						assert.NoError(t, err)
					} else if tt.body != "" {
						assert.NoError(t, json.NewEncoder(w).Encode(json.RawMessage(tt.body)))
					}
				default:
					t.Errorf("unexpected feed request: %s", r.URL)
					w.WriteHeader(http.StatusBadRequest)
				}
			}))
			defer server.Close()
			lag, count, split, err := estimateCosmosDBLegacyLag(newCosmosDBEPKTestClient(server))
			require.Error(t, err)
			assert.Zero(t, lag)
			assert.Zero(t, count)
			assert.False(t, split)
		})
	}
}

func TestCosmosDBEPKAndLegacyNeverCheckpointed(t *testing.T) {
	for _, version := range []int{0, 1} {
		t.Run(fmt.Sprintf("version=%d", version), func(t *testing.T) {
			rangeRequests, feedRequests := 0, 0
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/dbs/testdb/colls/leases/docs":
					leases := []map[string]interface{}{
						{"id": "first", "version": version, "LeaseToken": "6", "ContinuationToken": nil, "FeedRange": "ignored in v0"},
						{"id": "second", "version": version, "LeaseToken": "9", "ContinuationToken": "", "FeedRange": "ignored in v0"},
					}
					if version == 1 {
						for i, bounds := range [][2]string{{"", "80"}, {"80", "FF"}} {
							leases[i]["LeaseToken"] = bounds[0] + "-" + bounds[1]
							leases[i]["FeedRange"] = map[string]interface{}{"Range": map[string]string{"min": bounds[0], "max": bounds[1]}}
						}
					}
					_ = json.NewEncoder(w).Encode(map[string]interface{}{"Documents": leases})
				case "/dbs/testdb/colls/data/pkranges":
					rangeRequests++
					_, _ = w.Write([]byte(`{"PartitionKeyRanges":[{"id":"6","minInclusive":"","maxExclusive":"80"},{"id":"9","minInclusive":"80","maxExclusive":"FF"}]}`))
				case "/dbs/testdb/colls/data/docs":
					assert.Equal(t, []string{"6", "9"}[feedRequests], r.Header.Get("x-ms-documentdb-partitionkeyrangeid"))
					feedRequests++
					assert.NotContains(t, r.Header, "If-None-Match")
					assert.Empty(t, r.Header.Get("x-ms-read-key-type"))
					w.Header().Set("x-ms-session-token", "6:0#500")
					_, _ = w.Write([]byte(`{"Documents":[{"_lsn":1}]}`))
				default:
					t.Errorf("unexpected request: %s", r.URL)
					w.WriteHeader(http.StatusBadRequest)
				}
			}))
			defer server.Close()
			lag, count, split, err := estimateCosmosDBLegacyLag(newCosmosDBEPKTestClient(server))
			require.NoError(t, err)
			assert.Equal(t, int64(1000), lag)
			assert.Equal(t, int64(1), count)
			assert.False(t, split)
			assert.Equal(t, version, rangeRequests)
			assert.Equal(t, 2, feedRequests)
		})
	}
}

const cosmosDBTestEndpoint = "https://test.documents.azure.com"

type cosmosDBTestRoundTripper func(*http.Request) (*http.Response, error)

func (f cosmosDBTestRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func cosmosDBTestTransport(server *httptest.Server) http.RoundTripper {
	return cosmosDBTestRoundTripper(func(req *http.Request) (*http.Response, error) {
		clone := req.Clone(req.Context())
		target, err := url.Parse(server.URL)
		if err != nil {
			return nil, err
		}
		clone.URL.Scheme, clone.URL.Host = target.Scheme, target.Host
		return server.Client().Transport.RoundTrip(clone)
	})
}

func setCosmosDBEnvironmentFile(t *testing.T, contents interface{}) {
	t.Helper()
	data, err := json.Marshal(contents)
	require.NoError(t, err)
	setCosmosDBEnvironmentFileBytes(t, data)
}

func setCosmosDBEnvironmentFileBytes(t *testing.T, data []byte) {
	t.Helper()
	// Keep test fixtures inside the checkout, including when the system temp directory is restricted.
	path := fmt.Sprintf(".cosmosdb-environment-%x.json", sha256.Sum256([]byte(t.Name())))
	require.NoError(t, os.WriteFile(path, data, 0o600))
	t.Cleanup(func() { require.NoError(t, os.Remove(path)) })
	absolute, err := filepath.Abs(path)
	require.NoError(t, err)
	t.Setenv(azure.EnvironmentFilepathName, absolute)
}

func cosmosDBPrivateEnvironment(authority string) map[string]interface{} {
	return map[string]interface{}{
		"cosmosDBDNSSuffix":       "documents.private.example",
		"activeDirectoryEndpoint": authority,
		"resourceIdentifiers":     map[string]string{"cosmosDB": "https://cosmos.private.example"},
	}
}

func TestCosmosDBRejectsUntrustedEndpointsBeforeCredentials(t *testing.T) {
	endpoints := []string{
		"http://test.documents.azure.com",
		"https://attacker.example",
		"https://documents.azure.com",
		"https://evildocuments.azure.com",
		"https://test.documents.azure.com.attacker.example",
		"https://test.documents.azure.cn",
		"https://test.documents.azure.us",
		"https://127.0.0.1", "https://[::1]", "https://2130706433",
		"https://test.documents.azure.com@attacker.example",
		"https://attacker@test.documents.azure.com",
		"https://test.documents.azure.com?target=attacker",
		"https://test.documents.azure.com?",
		"https://test.documents.azure.com#fragment",
		"https://test.documents.azure.com#",
		"https://test.documents.azure.com/path",
		"https://test.documents.azure.com/%2f",
		"https://test.documents.azure.com\\@attacker.example",
		"https:test.documents.azure.com",
		"//test.documents.azure.com",
		"https://test.documents.azure.com.",
		"https://test..documents.azure.com",
		"https://test.documents.azure.com:0",
		"https://test.documents.azure.com:65536",
		"https://test.documents.azure.com:",
	}
	for _, endpoint := range endpoints {
		for _, field := range []string{"endpoint", "leaseEndpoint", "connection", "leaseConnection"} {
			for _, auth := range []string{"key", "service-principal", "workload"} {
				t.Run(field+"/"+auth+"/"+endpoint, func(t *testing.T) {
					meta := &azureCosmosDBMetadata{Endpoint: cosmosDBTestEndpoint, LeaseEndpoint: cosmosDBTestEndpoint}
					identity := kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone}
					switch auth {
					case "key":
						meta.CosmosDBKey, meta.LeaseCosmosDBKey = "dGVzdA==", "bGVhc2U="
					case "service-principal":
						meta.TenantID = "invalid/tenant"
					case "workload":
						identity.Provider = kedav1alpha1.PodIdentityProviderAzureWorkload
					}
					switch field {
					case "endpoint":
						meta.Endpoint = endpoint
					case "leaseEndpoint":
						meta.LeaseEndpoint = endpoint
					case "connection":
						meta.Connection = "AccountEndpoint=" + endpoint + ";AccountKey=dGVzdA=="
					case "leaseConnection":
						meta.LeaseConnection = "AccountEndpoint=" + endpoint + ";AccountKey=dGVzdA=="
					}
					client, err := newCosmosDBClient(meta, nil, identity, logr.Discard(), time.Second)
					require.Error(t, err)
					assert.Nil(t, client)
					assert.Contains(t, err.Error(), "cosmos db")
					assert.NotContains(t, err.Error(), "creating cosmos db credential")
				})
			}
		}
	}
}

func TestCosmosDBTrustedCloudsAndInheritedEndpoints(t *testing.T) {
	for _, cloud := range []struct{ name, suffix, authority string }{
		{"", "documents.azure.com", "https://login.microsoftonline.com/"},
		{"AzurePublicCloud", "documents.azure.com", "https://login.microsoftonline.com/"},
		{"AzureUSGovernmentCloud", "documents.azure.us", "https://login.microsoftonline.us/"},
		{"AzureChinaCloud", "documents.azure.cn", "https://login.chinacloudapi.cn/"},
		{"AzureGermanCloud", "documents.microsoftazure.de", "https://login.microsoftonline.de/"},
	} {
		for _, auth := range []string{"key", "service-principal", "workload", "mixed"} {
			t.Run(cloud.name+"/"+auth, func(t *testing.T) {
				t.Setenv("AZURE_CLIENT_ID", "client-id")
				t.Setenv("AZURE_TENANT_ID", "tenant-id")
				t.Setenv("AZURE_FEDERATED_TOKEN_FILE", "unused-token-file")
				meta := &azureCosmosDBMetadata{
					Endpoint: "https://Data." + cloud.suffix + ":443/", LeaseEndpoint: "https://lease." + cloud.suffix,
					TenantID: "tenant-id", ClientID: "client-id", ClientSecret: "client-secret",
				}
				identity := kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderNone}
				if auth == "key" || auth == "mixed" {
					meta.CosmosDBKey = "dGVzdA=="
				}
				if auth == "key" {
					meta.LeaseCosmosDBKey = "bGVhc2U="
				}
				if auth == "workload" {
					identity.Provider = kedav1alpha1.PodIdentityProviderAzureWorkload
				}
				metadata := map[string]string{"cloud": cloud.name}
				client, err := newCosmosDBClient(meta, metadata, identity, logr.Discard(), time.Second)
				require.NoError(t, err)
				require.NotNil(t, client.httpClient.CheckRedirect)
				if auth != "key" {
					assert.Equal(t, "https://cosmos.azure.com", client.cosmosDBResourceURL)
					assert.NotNil(t, client.credential)
				}
				env, err := resolveCosmosDBEnvironment(metadata)
				require.NoError(t, err)
				assert.Equal(t, cloud.authority, env.ActiveDirectoryEndpoint)
				assert.False(t, env.private)
			})
		}
	}
	for _, connection := range []bool{false, true} {
		for _, endpoint := range []string{cosmosDBTestEndpoint, "http://attacker.example", "https://attacker.example"} {
			t.Run(fmt.Sprintf("inherited/%t/%s", connection, endpoint), func(t *testing.T) {
				config := cosmosDBCapacityConfig(endpoint)
				if !connection {
					delete(config.TriggerMetadata, "connection")
					config.TriggerMetadata["endpoint"] = endpoint
					config.AuthParams = map[string]string{"cosmosDBKey": "dGVzdA=="}
				}
				scaler, err := NewAzureCosmosDBScaler(config)
				if endpoint != cosmosDBTestEndpoint {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				client := scaler.(*azureCosmosDBScaler).cosmosClient
				assert.Equal(t, client.dataEndpoint, client.leaseEndpoint)
				assert.Equal(t, client.dataKey, client.leaseKey)
				require.NoError(t, scaler.Close(context.Background()))
			})
		}
	}
}

func TestCosmosDBPrivateEnvironmentValidation(t *testing.T) {
	for _, name := range []string{"Private", "private", "AzureStackCloud", "azurestackcloud"} {
		t.Run(name, func(t *testing.T) {
			setCosmosDBEnvironmentFile(t, cosmosDBPrivateEnvironment("https://login.private.example/"))
			env, err := resolveCosmosDBEnvironment(map[string]string{"cloud": name})
			require.NoError(t, err)
			assert.True(t, env.private)
			assert.Equal(t, "https://cosmos.private.example", env.ResourceIdentifiers.CosmosDB)
			assert.Equal(t, "https://login.private.example/", env.ActiveDirectoryEndpoint)
			meta := &azureCosmosDBMetadata{
				Endpoint: "https://data.documents.private.example", LeaseEndpoint: "https://leases.documents.private.example",
				CosmosDBKey: "dGVzdA==", LeaseCosmosDBKey: "bGVhc2U=",
			}
			_, err = newCosmosDBClient(meta, map[string]string{"cloud": name}, kedav1alpha1.AuthPodIdentity{}, logr.Discard(), time.Second)
			require.NoError(t, err)
			for _, parameter := range []string{"cosmosDBResourceURL", "activeDirectoryEndpoint"} {
				for _, override := range []string{"", "https://attacker.example", "https://cosmos.private.example"} {
					_, err := resolveCosmosDBEnvironment(map[string]string{"cloud": name, parameter: override})
					require.ErrorContains(t, err, "metadata is not supported")
				}
			}
		})
	}
	for _, name := range []string{"Private", "AzureStackCloud"} {
		t.Run("missing-file/"+name, func(t *testing.T) {
			t.Setenv(azure.EnvironmentFilepathName, "")
			_, err := resolveCosmosDBEnvironment(map[string]string{"cloud": name})
			require.Error(t, err)
		})
	}
	for _, data := range []string{"{", "null", "{}", `{"resourceIdentifiers":{"cosmosDB":"https://cosmos.example"}}`} {
		t.Run("incomplete/"+data, func(t *testing.T) {
			setCosmosDBEnvironmentFileBytes(t, []byte(data))
			_, err := resolveCosmosDBEnvironment(map[string]string{"cloud": "Private"})
			require.Error(t, err)
		})
	}
	for _, suffix := range []string{"", "N/A", "*.example.com", ".example.com", "example.com.", "https://example.com", "127.0.0.1", "example", "a..example", "example.com:443"} {
		t.Run("suffix/"+suffix, func(t *testing.T) {
			env := cosmosDBPrivateEnvironment("https://login.private.example")
			env["cosmosDBDNSSuffix"] = suffix
			setCosmosDBEnvironmentFile(t, env)
			_, err := resolveCosmosDBEnvironment(map[string]string{"cloud": "Private"})
			require.ErrorContains(t, err, "cosmosDBDNSSuffix")
		})
	}
	for _, field := range []string{"activeDirectoryEndpoint", "resourceIdentifiers"} {
		for _, endpoint := range []string{"", "N/A", "http://login.example", "https://user@login.example", "https://login.example/path", "https://login.example?query", "https://login.example#fragment", "https:login.example"} {
			t.Run(field+"/"+endpoint, func(t *testing.T) {
				env := cosmosDBPrivateEnvironment("https://login.private.example")
				if field == "resourceIdentifiers" {
					env[field] = map[string]string{"cosmosDB": endpoint}
				} else {
					env[field] = endpoint
				}
				setCosmosDBEnvironmentFile(t, env)
				_, err := resolveCosmosDBEnvironment(map[string]string{"cloud": "Private"})
				require.Error(t, err)
			})
		}
	}
}

func TestCosmosDBMetadataCannotSelectAuthorityOrAudience(t *testing.T) {
	for _, cloud := range []string{"", "AzurePublicCloud", "AzureUSGovernmentCloud", "AzureChinaCloud", "Private", "AzureStackCloud"} {
		for _, parameter := range []string{"cosmosDBResourceURL", "activeDirectoryEndpoint"} {
			for _, key := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/key=%t", cloud, parameter, key), func(t *testing.T) {
					meta := &azureCosmosDBMetadata{Endpoint: cosmosDBTestEndpoint, LeaseEndpoint: cosmosDBTestEndpoint}
					if key {
						meta.CosmosDBKey, meta.LeaseCosmosDBKey = "dGVzdA==", "bGVhc2U="
					}
					_, err := newCosmosDBClient(meta, map[string]string{"cloud": cloud, parameter: "https://attacker.example"}, kedav1alpha1.AuthPodIdentity{}, logr.Discard(), time.Second)
					require.ErrorContains(t, err, parameter+" metadata is not supported")
				})
			}
		}
	}
}

func TestCosmosDBAuthOnlyForConfiguredOrigins(t *testing.T) {
	for _, key := range []string{"", "dGVzdA=="} {
		for _, endpoint := range []string{
			"https://test.documents.azure.com/dbs/db", "https://TEST.documents.azure.com:443/dbs/db",
			"https://lease.documents.azure.com/dbs/db", "https://other.documents.azure.com/dbs/db",
			"https://test.documents.azure.com:444/dbs/db", "http://test.documents.azure.com/dbs/db",
			"https://attacker.example/dbs/db", "https://user@test.documents.azure.com/dbs/db",
			"https://test.documents.azure.com.attacker.example/dbs/db", "https:test.documents.azure.com/dbs/db",
			"https://test.documents.azure.com/dbs/db#fragment",
		} {
			t.Run(key+"/"+endpoint, func(t *testing.T) {
				credential := &recordingTokenCredential{token: "secret"}
				client := &cosmosDBClient{
					dataEndpoint: cosmosDBTestEndpoint, leaseEndpoint: "https://lease.documents.azure.com",
					credential: credential, cosmosDBResourceURL: "https://cosmos.azure.com",
				}
				req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, endpoint, nil)
				require.NoError(t, err)
				err = client.setAuthHeader(req, http.MethodGet, "docs", "resource", "date", key)
				allowed := endpoint == "https://test.documents.azure.com/dbs/db" ||
					endpoint == "https://TEST.documents.azure.com:443/dbs/db" || endpoint == "https://lease.documents.azure.com/dbs/db"
				if allowed {
					require.NoError(t, err)
					assert.NotEmpty(t, req.Header.Get("Authorization"))
				} else {
					require.Error(t, err)
					assert.Empty(t, req.Header.Get("Authorization"))
					assert.Empty(t, credential.scopes)
				}
			})
		}
	}
	for _, endpoint := range []string{"", cosmosDBTestEndpoint} {
		credential := &recordingTokenCredential{token: "secret"}
		client := &cosmosDBClient{dataEndpoint: endpoint, credential: credential}
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, cosmosDBTestEndpoint+"/docs", nil)
		require.NoError(t, err)
		req.Host = "attacker.example"
		require.Error(t, client.setAuthHeader(req, http.MethodGet, "docs", "resource", "date", ""))
		assert.Empty(t, credential.scopes)
	}
}

func TestCosmosDBDoesNotFollowRedirects(t *testing.T) {
	for _, status := range []int{301, 302, 303, 307, 308} {
		for _, target := range []string{
			cosmosDBTestEndpoint + "/redirected",
			"https://test.documents.azure.com:444/redirected",
			"https://lease.documents.azure.com/redirected",
			"https://attacker.example/redirected",
			"http://test.documents.azure.com/redirected",
		} {
			for _, key := range []bool{false, true} {
				t.Run(fmt.Sprintf("%d/%s/key=%t", status, target, key), func(t *testing.T) {
					var initial, redirected atomic.Int64
					server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if r.URL.Path == "/redirected" {
							redirected.Inc()
							w.WriteHeader(http.StatusOK)
							return
						}
						initial.Inc()
						assert.NotEmpty(t, r.Header.Get("Authorization"))
						w.Header().Set("Location", target)
						w.WriteHeader(status)
					}))
					defer server.Close()
					meta := &azureCosmosDBMetadata{
						Endpoint: cosmosDBTestEndpoint, LeaseEndpoint: "https://lease.documents.azure.com",
						CosmosDBKey: "dGVzdA==", LeaseCosmosDBKey: "bGVhc2U=",
						DatabaseID: "db", ContainerID: "data", LeaseDatabaseID: "db", LeaseContainerID: "leases",
					}
					client, err := newCosmosDBClient(meta, nil, kedav1alpha1.AuthPodIdentity{}, logr.Discard(), time.Second)
					require.NoError(t, err)
					client.httpClient.Transport = cosmosDBTestTransport(server)
					if !key {
						client.dataKey, client.leaseKey = "", ""
						client.credential = &recordingTokenCredential{token: "secret"}
						client.cosmosDBResourceURL = "https://cosmos.azure.com"
					}
					_, err = client.queryLeases(context.Background())
					require.ErrorContains(t, err, fmt.Sprintf("status %d", status))
					_, err = client.readPartitionKeyRanges(context.Background())
					require.ErrorContains(t, err, fmt.Sprintf("status %d", status))
					_, err = client.readChangeFeed(context.Background(), "0", cosmosDBChangeFeedStart{}, nil)
					require.ErrorContains(t, err, fmt.Sprintf("status %d", status))
					assert.Equal(t, int64(3), initial.Load())
					assert.Zero(t, redirected.Load())
				})
			}
		}
	}
}

func TestCosmosDBCredentialRejectsRedirectsAndDiscoveryTokenOrigin(t *testing.T) {
	for _, stage := range []string{"discovery-redirect", "token-redirect", "discovery-token-origin"} {
		for _, status := range []int{301, 302, 303, 307, 308} {
			t.Run(fmt.Sprintf("%s/%d", stage, status), func(t *testing.T) {
				var attackerRequests, tokenRequests atomic.Int64
				attacker := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					attackerRequests.Inc()
					w.WriteHeader(http.StatusOK)
				}))
				defer attacker.Close()
				var authority *httptest.Server
				authority = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					if strings.Contains(r.URL.Path, ".well-known") && stage != "discovery-redirect" {
						tokenEndpoint := authority.URL + "/tenant-id/oauth2/v2.0/token"
						if stage == "discovery-token-origin" {
							tokenEndpoint = attacker.URL + "/token"
						}
						_ = json.NewEncoder(w).Encode(map[string]string{
							"authorization_endpoint": authority.URL + "/tenant-id/oauth2/v2.0/authorize",
							"token_endpoint":         tokenEndpoint, "issuer": authority.URL + "/tenant-id/v2.0",
						})
						return
					}
					if r.Method == http.MethodPost {
						tokenRequests.Inc()
						assert.NoError(t, r.ParseForm())
						assert.Equal(t, "client-secret", r.Form.Get("client_secret"))
					}
					w.Header().Set("Location", attacker.URL+"/token")
					w.WriteHeader(status)
				}))
				defer authority.Close()
				setCosmosDBEnvironmentFile(t, cosmosDBPrivateEnvironment(authority.URL))
				env, err := resolveCosmosDBEnvironment(map[string]string{"cloud": "Private"})
				require.NoError(t, err)
				credential, resource, err := newCosmosDBTokenCredential(
					&azureCosmosDBMetadata{TenantID: "tenant-id", ClientID: "client-id", ClientSecret: "client-secret"},
					env, kedav1alpha1.AuthPodIdentity{}, logr.Discard(), authority.Client(),
				)
				require.NoError(t, err)
				_, err = credential.GetToken(context.Background(), policy.TokenRequestOptions{Scopes: []string{resource + "/.default"}})
				require.Error(t, err)
				assert.Zero(t, attackerRequests.Load())
				if stage == "token-redirect" {
					assert.Equal(t, int64(1), tokenRequests.Load())
				} else {
					assert.Zero(t, tokenRequests.Load())
				}
			})
		}
	}
}

func TestCosmosDBWorkloadAuthorityIsAdministratorControlled(t *testing.T) {
	env, err := resolveCosmosDBEnvironment(nil)
	require.NoError(t, err)
	t.Setenv("AZURE_CLIENT_ID", "client-id")
	t.Setenv("AZURE_TENANT_ID", "tenant-id")
	t.Setenv("AZURE_FEDERATED_TOKEN_FILE", "unused-token-file")
	identity := kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAzureWorkload}
	for _, authority := range []string{"http://login.example", "https://user@login.example", "https://login.example/path"} {
		t.Run(authority, func(t *testing.T) {
			t.Setenv("AZURE_AUTHORITY_HOST", authority)
			_, _, err := newCosmosDBTokenCredential(&azureCosmosDBMetadata{}, env, identity, logr.Discard(), &http.Client{})
			require.ErrorContains(t, err, "invalid cosmos db credential authority")
		})
	}
	t.Setenv("AZURE_AUTHORITY_HOST", "https://login.microsoftonline.us")
	credential, resource, err := newCosmosDBTokenCredential(&azureCosmosDBMetadata{}, env, identity, logr.Discard(), &http.Client{})
	require.NoError(t, err)
	assert.IsType(t, &azidentity.ChainedTokenCredential{}, credential)
	assert.Equal(t, "https://cosmos.azure.com", resource)
}

func TestCosmosDBKnownCloudServicePrincipalTokenAcquisition(t *testing.T) {
	for _, region := range []string{"", "westus", "TryAutoDetect"} {
		t.Run("region="+region, func(t *testing.T) {
			t.Setenv("AZURE_REGIONAL_AUTHORITY_NAME", region)
			t.Setenv("REGION_NAME", "westus")
			for _, cloud := range []string{"AzurePublicCloud", "AzureUSGovernmentCloud", "AzureChinaCloud", "AzureGermanCloud"} {
				t.Run(cloud, func(t *testing.T) {
					env, err := resolveCosmosDBEnvironment(map[string]string{"cloud": cloud})
					require.NoError(t, err)
					authority := strings.TrimSuffix(env.ActiveDirectoryEndpoint, "/")
					host, err := url.Parse(authority)
					require.NoError(t, err)
					if region != "" {
						regionalHost := host.Host
						if cloud == "AzurePublicCloud" {
							regionalHost = "login.microsoft.com"
						}
						authority = "https://westus." + regionalHost
						host, err = url.Parse(authority)
						require.NoError(t, err)
					}
					var tokenRequests atomic.Int64
					server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						assert.Equal(t, host.Host, r.Host)
						w.Header().Set("Content-Type", "application/json")
						switch {
						case r.URL.Path == "/common/discovery/instance":
							_ = json.NewEncoder(w).Encode(map[string]interface{}{
								"tenant_discovery_endpoint": authority + "/tenant-id/v2.0/.well-known/openid-configuration",
								"metadata": []map[string]interface{}{{
									"preferred_network": host.Host, "preferred_cache": host.Host, "aliases": []string{host.Host},
								}},
							})
						case strings.Contains(r.URL.Path, ".well-known"):
							_ = json.NewEncoder(w).Encode(map[string]string{
								"authorization_endpoint": authority + "/tenant-id/oauth2/v2.0/authorize",
								"token_endpoint":         authority + "/tenant-id/oauth2/v2.0/token",
								"issuer":                 strings.TrimSuffix(env.ActiveDirectoryEndpoint, "/") + "/tenant-id/v2.0",
							})
						case r.URL.Path == "/tenant-id/oauth2/v2.0/token":
							tokenRequests.Inc()
							assert.NoError(t, r.ParseForm())
							assert.Equal(t, "client-secret", r.Form.Get("client_secret"))
							assert.ElementsMatch(t, []string{"https://cosmos.azure.com/.default", "openid", "offline_access", "profile"}, strings.Fields(r.Form.Get("scope")))
							_ = json.NewEncoder(w).Encode(map[string]interface{}{
								"access_token": "cloud-token", "expires_in": 3600, "token_type": "Bearer",
							})
						default:
							t.Errorf("unexpected identity request: %s", r.URL.Path)
							w.WriteHeader(http.StatusBadRequest)
						}
					}))
					defer server.Close()
					credential, resource, err := newCosmosDBTokenCredential(
						&azureCosmosDBMetadata{TenantID: "tenant-id", ClientID: "client-id", ClientSecret: "client-secret"},
						env, kedav1alpha1.AuthPodIdentity{}, logr.Discard(), &http.Client{Transport: cosmosDBTestTransport(server)},
					)
					require.NoError(t, err)
					token, err := credential.GetToken(context.Background(), policy.TokenRequestOptions{Scopes: []string{resource + "/.default"}})
					require.NoError(t, err)
					assert.Equal(t, "cloud-token", token.Token)
					assert.Equal(t, int64(1), tokenRequests.Load())
				})
			}
		})
	}
}

func TestCosmosDBRegionalAuthorityOriginGuard(t *testing.T) {
	for _, region := range []string{"westus", "TryAutoDetect"} {
		for _, endpoint := range []string{
			"https://westus.login.microsoft.com", "https://westus.login.microsoftonline.com",
			"https://eastus.login.microsoft.com", "https://westus.login.microsoft.com.attacker.example",
			"https://westus.attacker.login.microsoft.com", "https://westus.login.microsoft.com:444",
			"http://westus.login.microsoft.com", "https://westus.login.microsoftonline.us",
		} {
			t.Run(region+"/"+endpoint, func(t *testing.T) {
				req, err := http.NewRequest(http.MethodPost, endpoint+"/tenant/oauth2/v2.0/token", strings.NewReader("client_secret=secret"))
				require.NoError(t, err)
				want := endpoint == "https://westus.login.microsoft.com" || endpoint == "https://westus.login.microsoftonline.com" ||
					(region == "TryAutoDetect" && endpoint == "https://eastus.login.microsoft.com")
				assert.Equal(t, want, isCosmosDBRegionalAuthorityRequest(req, "https://login.microsoftonline.com", region))
				req.Host = "attacker.example"
				assert.False(t, isCosmosDBRegionalAuthorityRequest(req, "https://login.microsoftonline.com", region))
			})
		}
	}
	req, err := http.NewRequest(http.MethodPost, "https://westus.login.microsoft.com/token", nil)
	require.NoError(t, err)
	for _, region := range []string{"", "westus.attacker", "WestUS", "-westus", "westus/"} {
		assert.False(t, isCosmosDBRegionalAuthorityRequest(req, "https://login.microsoftonline.com", region))
	}
	assert.False(t, isCosmosDBRegionalAuthorityRequest(req, "https://authority.private.example", "westus"))
}

func TestCosmosDBWorkloadTokenTransport(t *testing.T) {
	for _, redirect := range []bool{false, true} {
		t.Run(fmt.Sprintf("redirect=%t", redirect), func(t *testing.T) {
			t.Setenv("PATH", "")
			t.Setenv("AZURE_CLIENT_ID", "client-id")
			t.Setenv("AZURE_TENANT_ID", "tenant-id")
			tokenFile := fmt.Sprintf(".cosmosdb-assertion-%x.jwt", sha256.Sum256([]byte(t.Name())))
			require.NoError(t, os.WriteFile(tokenFile, []byte("federated-assertion"), 0o600))
			t.Cleanup(func() { require.NoError(t, os.Remove(tokenFile)) })
			t.Setenv("AZURE_FEDERATED_TOKEN_FILE", tokenFile)
			var redirected, tokenRequests atomic.Int64
			var server *httptest.Server
			server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case strings.Contains(r.URL.Path, ".well-known"):
					_ = json.NewEncoder(w).Encode(map[string]string{
						"authorization_endpoint": server.URL + "/tenant-id/oauth2/v2.0/authorize",
						"token_endpoint":         server.URL + "/tenant-id/oauth2/v2.0/token",
						"issuer":                 server.URL + "/tenant-id/v2.0",
					})
				case r.URL.Path == "/tenant-id/oauth2/v2.0/token":
					tokenRequests.Inc()
					assert.NoError(t, r.ParseForm())
					assert.Equal(t, "federated-assertion", r.Form.Get("client_assertion"))
					assert.ElementsMatch(t, []string{"https://cosmos.private.example/.default", "openid", "offline_access", "profile"}, strings.Fields(r.Form.Get("scope")))
					if redirect {
						w.Header().Set("Location", server.URL+"/redirected")
						w.WriteHeader(http.StatusTemporaryRedirect)
						return
					}
					_ = json.NewEncoder(w).Encode(map[string]interface{}{
						"access_token": "workload-token", "expires_in": 3600, "token_type": "Bearer",
					})
				default:
					redirected.Inc()
					w.WriteHeader(http.StatusOK)
				}
			}))
			defer server.Close()
			setCosmosDBEnvironmentFile(t, cosmosDBPrivateEnvironment("https://unused-authority.private.example"))
			t.Setenv("AZURE_AUTHORITY_HOST", server.URL)
			env, err := resolveCosmosDBEnvironment(map[string]string{"cloud": "Private"})
			require.NoError(t, err)
			credential, resource, err := newCosmosDBTokenCredential(
				&azureCosmosDBMetadata{}, env,
				kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAzureWorkload},
				logr.Discard(), server.Client(),
			)
			require.NoError(t, err)
			token, err := credential.GetToken(context.Background(), policy.TokenRequestOptions{Scopes: []string{resource + "/.default"}})
			if redirect {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "workload-token", token.Token)
			}
			assert.Equal(t, int64(1), tokenRequests.Load())
			assert.Zero(t, redirected.Load())
		})
	}
}

func TestCosmosDBCredentialTransportOriginGuard(t *testing.T) {
	for _, tc := range []struct {
		method, endpoint, authorization string
		allowed                         bool
	}{
		{http.MethodPost, "https://login.private.example/token", "", true},
		{http.MethodGet, "https://login.private.example/metadata", "", true},
		{http.MethodGet, "https://login.microsoftonline.com/common/discovery/instance", "", true},
		{http.MethodGet, "https://login.microsoft.com/common/discovery/instance", "", true},
		{http.MethodGet, "https://login.microsoft.com/common/discovery/instance", "secret", false},
		{http.MethodPost, "https://login.microsoft.com/common/discovery/instance", "", false},
		{http.MethodGet, "https://attacker.example/common/discovery/instance", "", false},
		{http.MethodGet, "https://attacker.example/metadata", "", false},
		{http.MethodPost, "https://attacker.example/token", "", false},
		{http.MethodPost, "http://login.private.example/token", "", false},
	} {
		t.Run(tc.method+"/"+tc.endpoint+"/"+tc.authorization, func(t *testing.T) {
			calls := 0
			transport := cosmosDBCredentialTransport{
				authority: "https://login.private.example",
				client: &http.Client{Transport: cosmosDBTestRoundTripper(func(*http.Request) (*http.Response, error) {
					calls++
					return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
				})},
			}
			req, err := http.NewRequestWithContext(context.Background(), tc.method, tc.endpoint, nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", tc.authorization)
			response, err := transport.Do(req)
			if tc.allowed {
				require.NoError(t, err)
				require.NoError(t, response.Body.Close())
				assert.Equal(t, 1, calls)
			} else {
				require.Error(t, err)
				assert.Zero(t, calls)
			}
		})
	}
}
