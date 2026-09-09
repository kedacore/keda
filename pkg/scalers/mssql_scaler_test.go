package scalers

import (
	"context"
	"database/sql"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type parseMSSQLMetadataTestData struct {
	name                     string
	metadata                 map[string]string
	resolvedEnv              map[string]string
	authParams               map[string]string
	podIdentity              kedav1alpha1.AuthPodIdentity
	expectedError            string
	expectedConnectionString string
	expectedMetricName       string
	expectedDriverName       string
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
		name:        "Workload identity with host and database",
		metadata:    map[string]string{"query": "SELECT 1", "targetValue": "1", "host": "myserver.database.windows.net", "database": "mydb"},
		resolvedEnv: map[string]string{},
		authParams:  map[string]string{},
		podIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAzureWorkload},
	},
	{
		name:        "Workload identity without username/password",
		metadata:    map[string]string{"query": "SELECT 1", "targetValue": "1", "host": "myserver.database.windows.net"},
		resolvedEnv: map[string]string{},
		authParams:  map[string]string{},
		podIdentity: kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAzureWorkload},
	},
	{
		name:          "Workload identity missing host",
		metadata:      map[string]string{"query": "SELECT 1", "targetValue": "1", "database": "mydb"},
		resolvedEnv:   map[string]string{},
		authParams:    map[string]string{},
		podIdentity:   kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAzureWorkload},
		expectedError: "must provide either connectionstring or host",
	},
	{
		name:          "Unsupported pod identity provider",
		metadata:      map[string]string{"query": "SELECT 1", "targetValue": "1", "host": "myserver.database.windows.net"},
		resolvedEnv:   map[string]string{},
		authParams:    map[string]string{},
		podIdentity:   kedav1alpha1.AuthPodIdentity{Provider: "gcp"},
		expectedError: "pod identity gcp is not supported for mssql scaler",
	},
	{
		name:                     "Driver name defaults to sqlserver when not set",
		metadata:                 map[string]string{"query": "SELECT 1", "targetValue": "1", "host": "127.0.0.1"},
		resolvedEnv:              map[string]string{},
		authParams:               map[string]string{},
		expectedConnectionString: "sqlserver://127.0.0.1",
		expectedDriverName:       "sqlserver",
	},
	{
		name:                     "Azure SQL driver from trigger metadata",
		metadata:                 map[string]string{"query": "SELECT 1", "targetValue": "1", "host": "example.database.windows.net", "database": "AdventureWorks", "driverName": "azuresql"},
		resolvedEnv:              map[string]string{},
		authParams:               map[string]string{},
		expectedConnectionString: "sqlserver://example.database.windows.net?database=AdventureWorks",
		expectedDriverName:       "azuresql",
	},
	{
		name:                     "Azure SQL driver from auth params",
		metadata:                 map[string]string{"query": "SELECT 1", "targetValue": "1"},
		resolvedEnv:              map[string]string{},
		authParams:               map[string]string{"driverName": "azuresql", "connectionString": "sqlserver://example.database.windows.net?fedauth=ActiveDirectoryDefault"},
		expectedConnectionString: "sqlserver://example.database.windows.net?fedauth=ActiveDirectoryDefault",
		expectedDriverName:       "azuresql",
	},
	{
		name:          "Error: unknown driver name",
		metadata:      map[string]string{"query": "SELECT 1", "targetValue": "1", "host": "127.0.0.1", "driverName": "postgres"},
		resolvedEnv:   map[string]string{},
		authParams:    map[string]string{},
		expectedError: "parameter \"driverName\" value \"postgres\" must be one of [sqlserver azuresql]",
	},
	{
		name:          "Error: azuresql driver combined with workload identity",
		metadata:      map[string]string{"query": "SELECT 1", "targetValue": "1", "host": "myserver.database.windows.net", "driverName": "azuresql"},
		resolvedEnv:   map[string]string{},
		authParams:    map[string]string{},
		podIdentity:   kedav1alpha1.AuthPodIdentity{Provider: kedav1alpha1.PodIdentityProviderAzureWorkload},
		expectedError: "driverName azuresql cannot be combined with azure-workload pod identity",
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

			meta, _, err := parseMSSQLMetadata(logr.Discard(), config)

			if testData.expectedError != "" {
				assert.EqualError(t, err, testData.expectedError)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, meta)

			if testData.expectedConnectionString != "" {
				connStr := getMSSQLConnectionString(&mssqlScaler{metadata: meta})
				assert.Equal(t, testData.expectedConnectionString, connStr)
			}

			if testData.expectedDriverName != "" {
				assert.Equal(t, testData.expectedDriverName, meta.DriverName)
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

			meta, _, err := parseMSSQLMetadata(logr.Discard(), &scalersconfig.ScalerConfig{
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

func TestMSSQLWorkloadIdentityConnectionString(t *testing.T) {
	meta := &mssqlMetadata{
		Host:     "myserver.database.windows.net",
		Database: "mydb",
	}
	scaler := &mssqlScaler{metadata: meta}
	connStr := getMSSQLConnectionString(scaler)
	assert.Equal(t, "sqlserver://myserver.database.windows.net?database=mydb", connStr)
}

func TestMSSQLWorkloadIdentityConnectionStringNoDatabase(t *testing.T) {
	meta := &mssqlMetadata{
		Host: "myserver.database.windows.net",
	}
	scaler := &mssqlScaler{metadata: meta}
	connStr := getMSSQLConnectionString(scaler)
	assert.Equal(t, "sqlserver://myserver.database.windows.net", connStr)
}

func TestMSSQLRegisteredDrivers(t *testing.T) {
	// Both drivers have to be registered with database/sql before sql.Open can
	// resolve the value of the driverName parameter.
	drivers := sql.Drivers()
	assert.Contains(t, drivers, mssqlDriverSQLServer)
	assert.Contains(t, drivers, mssqlDriverAzureSQL)
}
