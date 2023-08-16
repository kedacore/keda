package scalers

import (
	"errors"
	"testing"
)

type mssqlTestData struct {
	// test inputs
	metadata    map[string]string
	resolvedEnv map[string]string
	authParams  map[string]string

	// expected outputs
	expectedMetricName       string
	expectedConnectionString string
	expectedError            error
}

var testInputs = []mssqlTestData{
	// direct connection string input
	{
		metadata:                 map[string]string{"query": "SELECT 1", "targetValue": "1"},
		resolvedEnv:              map[string]string{},
		authParams:               map[string]string{"connectionString": "sqlserver://localhost"},
		expectedConnectionString: "sqlserver://localhost",
	},
	// direct connection string input with activationTargetValue
	{
		metadata:                 map[string]string{"query": "SELECT 1", "targetValue": "1", "activationTargetValue": "20"},
		resolvedEnv:              map[string]string{},
		authParams:               map[string]string{"connectionString": "sqlserver://localhost"},
		expectedConnectionString: "sqlserver://localhost",
	},
	// direct connection string input, OLEDB format
	{
		metadata:                 map[string]string{"query": "SELECT 1", "targetValue": "1"},
		resolvedEnv:              map[string]string{},
		authParams:               map[string]string{"connectionString": "Server=example.database.windows.net;port=1433;Database=AdventureWorks;Persist Security Info=False;User ID=user1;Password=Password#1;MultipleActiveResultSets=False;Encrypt=True;TrustServerCertificate=False;Connection Timeout=30;"},
		expectedConnectionString: "Server=example.database.windows.net;port=1433;Database=AdventureWorks;Persist Security Info=False;User ID=user1;Password=Password#1;MultipleActiveResultSets=False;Encrypt=True;TrustServerCertificate=False;Connection Timeout=30;",
	},
	// connection string input via environment variables
	{
		metadata:                 map[string]string{"query": "SELECT 1", "targetValue": "1", "connectionStringFromEnv": "test_connection_string"},
		resolvedEnv:              map[string]string{"test_connection_string": "sqlserver://localhost?database=AdventureWorks"},
		authParams:               map[string]string{},
		expectedConnectionString: "sqlserver://localhost?database=AdventureWorks",
	},
	// connection string generated from minimal required metadata
	{
		metadata:                 map[string]string{"query": "SELECT 1", "targetValue": "1", "host": "127.0.0.1"},
		resolvedEnv:              map[string]string{},
		authParams:               map[string]string{},
		expectedMetricName:       "mssql",
		expectedConnectionString: "sqlserver://127.0.0.1",
	},
	// connection string generated from full metadata
	{
		metadata:                 map[string]string{"query": "SELECT 1", "targetValue": "1", "host": "example.database.windows.net", "username": "user1", "passwordFromEnv": "test_password", "port": "1433", "database": "AdventureWorks"},
		resolvedEnv:              map[string]string{"test_password": "Password#1"},
		authParams:               map[string]string{},
		expectedConnectionString: "sqlserver://user1:Password%231@example.database.windows.net:1433?database=AdventureWorks",
	},
	// variation of previous: no port, password from authParams, metricName from database name
	{
		metadata:                 map[string]string{"query": "SELECT 1", "targetValue": "1", "host": "example.database.windows.net", "username": "user2", "database": "AdventureWorks"},
		resolvedEnv:              map[string]string{},
		authParams:               map[string]string{"password": "Password#2"},
		expectedMetricName:       "mssql",
		expectedConnectionString: "sqlserver://user2:Password%232@example.database.windows.net?database=AdventureWorks",
	},
	// connection string generated from full authParams
	{
		metadata:                 map[string]string{"query": "SELECT 1", "targetValue": "1"},
		resolvedEnv:              map[string]string{},
		authParams:               map[string]string{"password": "Password#2", "host": "example.database.windows.net", "username": "user2", "database": "AdventureWorks", "port": "1433"},
		expectedMetricName:       "mssql",
		expectedConnectionString: "sqlserver://user2:Password%232@example.database.windows.net:1433?database=AdventureWorks",
	},
	// variation of previous: no database name, metricName from host
	{
		metadata:                 map[string]string{"query": "SELECT 1", "targetValue": "1", "host": "example.database.windows.net", "username": "user3"},
		resolvedEnv:              map[string]string{},
		authParams:               map[string]string{"password": "Password#3"},
		expectedMetricName:       "mssql",
		expectedConnectionString: "sqlserver://user3:Password%233@example.database.windows.net",
	},
	// Error: missing query
	{
		metadata:      map[string]string{"targetValue": "1"},
		resolvedEnv:   map[string]string{},
		authParams:    map[string]string{"connectionString": "sqlserver://localhost"},
		expectedError: ErrMsSQLNoQuery,
	},
	// Error: missing targetValue
	{
		metadata:      map[string]string{"query": "SELECT 1"},
		resolvedEnv:   map[string]string{},
		authParams:    map[string]string{"connectionString": "sqlserver://localhost"},
		expectedError: ErrMsSQLNoTargetValue,
	},
	// Error: missing host
	{
		metadata:      map[string]string{"query": "SELECT 1", "targetValue": "1"},
		resolvedEnv:   map[string]string{},
		authParams:    map[string]string{},
		expectedError: ErrScalerConfigMissingField,
	},
}

func TestMSSQLMetadataParsing(t *testing.T) {
	for _, testData := range testInputs {
		var config = ScalerConfig{
			ResolvedEnv:     testData.resolvedEnv,
			TriggerMetadata: testData.metadata,
			AuthParams:      testData.authParams,
		}

		outputMetadata, err := parseMSSQLMetadata(&config)
		if err != nil {
			if testData.expectedError == nil {
				t.Errorf("Unexpected error parsing input metadata: %v", err)
			} else if !errors.Is(err, testData.expectedError) {
				t.Errorf("Expected error '%v' but got '%v'", testData.expectedError, err)
			}

			continue
		}

		expectedQuery := "SELECT 1"
		if outputMetadata.query != expectedQuery {
			t.Errorf("Wrong query. Expected '%s' but got '%s'", expectedQuery, outputMetadata.query)
		}

		var expectedTargetValue float64 = 1
		if outputMetadata.targetValue != expectedTargetValue {
			t.Errorf("Wrong targetValue. Expected %f but got %f", expectedTargetValue, outputMetadata.targetValue)
		}

		outputConnectionString := getMSSQLConnectionString(outputMetadata)
		if testData.expectedConnectionString != outputConnectionString {
			t.Errorf("Wrong connection string. Expected '%s' but got '%s'", testData.expectedConnectionString, outputConnectionString)
		}

		if testData.expectedMetricName != "" && testData.expectedMetricName != outputMetadata.metricName {
			t.Errorf("Wrong metric name. Expected '%s' but got '%s'", testData.expectedMetricName, outputMetadata.metricName)
		}
	}
}
