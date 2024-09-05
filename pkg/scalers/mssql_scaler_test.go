package scalers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type parseMSSQLMetadataTestData struct {
	name                     string
	metadata                 map[string]string
	resolvedEnv              map[string]string
	authParams               map[string]string
	podIdentity              v1alpha1.AuthPodIdentity
	expectedError            string
	expectedConnectionString string
	expectedMetricName       string
}

var testMSSQLMetadata = []parseMSSQLMetadataTestData{
	{
		name:                     "Direct connection string input",
		metadata:                 map[string]string{"query": "SELECT 1", "targetValue": "1"},
		resolvedEnv:              map[string]string{},
		authParams:               map[string]string{"connectionString": "sqlserver://localhost"},
		expectedConnectionString: "sqlserver://localhost",
	},
	{
		name:                     "Direct connection string input with activationTargetValue",
		metadata:                 map[string]string{"query": "SELECT 1", "targetValue": "1", "activationTargetValue": "20"},
		resolvedEnv:              map[string]string{},
		authParams:               map[string]string{"connectionString": "sqlserver://localhost"},
		expectedConnectionString: "sqlserver://localhost",
	},
	{
		name:                     "Direct connection string input, OLEDB format",
		metadata:                 map[string]string{"query": "SELECT 1", "targetValue": "1"},
		resolvedEnv:              map[string]string{},
		authParams:               map[string]string{"connectionString": "Server=example.database.windows.net;port=1433;Database=AdventureWorks;Persist Security Info=False;User ID=user1;Password=Password#1;MultipleActiveResultSets=False;Encrypt=True;TrustServerCertificate=False;Connection Timeout=30;"},
		expectedConnectionString: "Server=example.database.windows.net;port=1433;Database=AdventureWorks;Persist Security Info=False;User ID=user1;Password=Password#1;MultipleActiveResultSets=False;Encrypt=True;TrustServerCertificate=False;Connection Timeout=30;",
	},
	{
		name:                     "Connection string input via environment variables",
		metadata:                 map[string]string{"query": "SELECT 1", "targetValue": "1", "connectionStringFromEnv": "test_connection_string"},
		resolvedEnv:              map[string]string{"test_connection_string": "sqlserver://localhost?database=AdventureWorks"},
		authParams:               map[string]string{},
		expectedConnectionString: "sqlserver://localhost?database=AdventureWorks",
	},
	{
		name:                     "Connection string generated from minimal required metadata",
		metadata:                 map[string]string{"query": "SELECT 1", "targetValue": "1", "host": "127.0.0.1"},
		resolvedEnv:              map[string]string{},
		authParams:               map[string]string{},
		expectedMetricName:       "mssql",
		expectedConnectionString: "sqlserver://127.0.0.1",
	},
	{
		name:                     "Connection string generated from full metadata",
		metadata:                 map[string]string{"query": "SELECT 1", "targetValue": "1", "host": "example.database.windows.net", "username": "user1", "passwordFromEnv": "test_password", "port": "1433", "database": "AdventureWorks"},
		resolvedEnv:              map[string]string{"test_password": "Password#1"},
		authParams:               map[string]string{},
		expectedConnectionString: "sqlserver://user1:Password%231@example.database.windows.net:1433?database=AdventureWorks",
	},
	{
		name:                     "Variation of previous: no port, password from authParams, metricName from database name",
		metadata:                 map[string]string{"query": "SELECT 1", "targetValue": "1", "host": "example.database.windows.net", "username": "user2", "database": "AdventureWorks"},
		resolvedEnv:              map[string]string{},
		authParams:               map[string]string{"password": "Password#2"},
		expectedMetricName:       "mssql",
		expectedConnectionString: "sqlserver://user2:Password%232@example.database.windows.net?database=AdventureWorks",
	},
	{
		name:                     "Connection string generated from full authParams",
		metadata:                 map[string]string{"query": "SELECT 1", "targetValue": "1"},
		resolvedEnv:              map[string]string{},
		authParams:               map[string]string{"password": "Password#2", "host": "example.database.windows.net", "username": "user2", "database": "AdventureWorks", "port": "1433"},
		expectedMetricName:       "mssql",
		expectedConnectionString: "sqlserver://user2:Password%232@example.database.windows.net:1433?database=AdventureWorks",
	},
	{
		name:                     "Variation of previous: no database name, metricName from host",
		metadata:                 map[string]string{"query": "SELECT 1", "targetValue": "1", "host": "example.database.windows.net", "username": "user3"},
		resolvedEnv:              map[string]string{},
		authParams:               map[string]string{"password": "Password#3"},
		expectedMetricName:       "mssql",
		expectedConnectionString: "sqlserver://user3:Password%233@example.database.windows.net",
	},
	{
		name:          "Error: missing query",
		metadata:      map[string]string{"targetValue": "1"},
		resolvedEnv:   map[string]string{},
		authParams:    map[string]string{"connectionString": "sqlserver://localhost"},
		expectedError: "missing required parameter \"query\" in [triggerMetadata]",
	},
	{
		name:          "Error: missing targetValue",
		metadata:      map[string]string{"query": "SELECT 1"},
		resolvedEnv:   map[string]string{},
		authParams:    map[string]string{"connectionString": "sqlserver://localhost"},
		expectedError: "missing required parameter \"targetValue\" in [triggerMetadata]",
	},
	{
		name:          "Error: missing host",
		metadata:      map[string]string{"query": "SELECT 1", "targetValue": "1"},
		resolvedEnv:   map[string]string{},
		authParams:    map[string]string{},
		expectedError: "must provide either connectionstring or host",
	},
	{
		name: "Valid metadata with Azure Workload Identity",
		metadata: map[string]string{
			"query":       "SELECT COUNT(*) FROM table",
			"targetValue": "5",
			"host":        "mssql-server.database.windows.net",
			"port":        "1433",
			"database":    "test-db",
		},
		resolvedEnv: map[string]string{},
		authParams: map[string]string{
			"workloadIdentityResource": "mssql-resource-id",
		},
		podIdentity: v1alpha1.AuthPodIdentity{
			Provider:              v1alpha1.PodIdentityProviderAzureWorkload,
			IdentityID:            kedautil.StringPointer("client-id"),
			IdentityTenantID:      kedautil.StringPointer("tenant-id"),
			IdentityAuthorityHost: kedautil.StringPointer("https://login.microsoftonline.com/"),
		},
		expectedError: "",
	},
	{
		name: "Azure Workload Identity without workloadIdentityResource",
		metadata: map[string]string{
			"query":       "SELECT COUNT(*) FROM table",
			"targetValue": "5",
			"host":        "mssql-server.database.windows.net",
			"port":        "1433",
			"database":    "test-db",
		},
		resolvedEnv: map[string]string{},
		authParams:  map[string]string{},
		podIdentity: v1alpha1.AuthPodIdentity{
			Provider:              v1alpha1.PodIdentityProviderAzureWorkload,
			IdentityID:            kedautil.StringPointer("client-id"),
			IdentityTenantID:      kedautil.StringPointer("tenant-id"),
			IdentityAuthorityHost: kedautil.StringPointer("https://login.microsoftonline.com/"),
		},
		expectedError: "",
	},
}

func TestParseMSSQLMetadata(t *testing.T) {
	for _, testData := range testMSSQLMetadata {
		t.Run(testData.name, func(t *testing.T) {
			config := &scalersconfig.ScalerConfig{
				TriggerMetadata: testData.metadata,
				ResolvedEnv:     testData.resolvedEnv,
				AuthParams:      testData.authParams,
				PodIdentity:     testData.podIdentity,
			}

			meta, err := parseMSSQLMetadata(config)

			if testData.expectedError != "" {
				assert.EqualError(t, err, testData.expectedError)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, meta)

				if testData.podIdentity.Provider == v1alpha1.PodIdentityProviderAzureWorkload {
					if workloadIdentityResource, ok := testData.authParams["workloadIdentityResource"]; ok && workloadIdentityResource != "" {
						// If workloadIdentityResource is provided, all fields should be set
						assert.Equal(t, workloadIdentityResource, meta.WorkloadIdentityResource)
						assert.Equal(t, *testData.podIdentity.IdentityID, meta.WorkloadIdentityClientID)
						assert.Equal(t, *testData.podIdentity.IdentityTenantID, meta.WorkloadIdentityTenantID)
						assert.Equal(t, *testData.podIdentity.IdentityAuthorityHost, meta.WorkloadIdentityAuthorityHost)
					} else {
						// If workloadIdentityResource is not provided, all fields should be empty
						assert.Empty(t, meta.WorkloadIdentityResource)
						assert.Empty(t, meta.WorkloadIdentityClientID)
						assert.Empty(t, meta.WorkloadIdentityTenantID)
						assert.Empty(t, meta.WorkloadIdentityAuthorityHost)
					}
				} else {
					// If not using Azure Workload Identity, all fields should be empty
					assert.Empty(t, meta.WorkloadIdentityResource)
					assert.Empty(t, meta.WorkloadIdentityClientID)
					assert.Empty(t, meta.WorkloadIdentityTenantID)
					assert.Empty(t, meta.WorkloadIdentityAuthorityHost)
				}
			}
		})
	}
}

func TestMSSQLGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range testMSSQLMetadata {
		t.Run(testData.name, func(t *testing.T) {
			if testData.expectedError != "" {
				return
			}

			meta, err := parseMSSQLMetadata(&scalersconfig.ScalerConfig{
				TriggerMetadata: testData.metadata,
				ResolvedEnv:     testData.resolvedEnv,
				AuthParams:      testData.authParams,
				PodIdentity:     testData.podIdentity,
			})

			assert.NoError(t, err)

			mockMSSQLScaler := mssqlScaler{
				metadata: meta,
			}

			metricSpec := mockMSSQLScaler.GetMetricSpecForScaling(context.Background())

			assert.NotNil(t, metricSpec)
			assert.Equal(t, 1, len(metricSpec))
			assert.Contains(t, metricSpec[0].External.Metric.Name, "mssql")
		})
	}
}
