package scalers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
)

type parsePostgreSQLMetadataTestData struct {
	metadata map[string]string
}

var testPostgreSQLMetdata = []parsePostgreSQLMetadataTestData{
	// connection with username and password
	{metadata: map[string]string{"query": "test_query", "targetQueryValue": "5", "connectionFromEnv": "test_connection_string"}},
	// connection with username
	{metadata: map[string]string{"query": "test_query", "targetQueryValue": "5", "connectionFromEnv": "test_connection_string2"}},
	// connection with activationTargetQueryValue
	{metadata: map[string]string{"query": "test_query", "targetQueryValue": "5", "activationTargetQueryValue": "3", "connectionFromEnv": "test_connection_string2"}},
	// connection without username and password
	{metadata: map[string]string{"query": "test_query", "targetQueryValue": "5", "connection": "postgresql://localhost:5432"}},
	// connection with password + metricname
	{metadata: map[string]string{"query": "test_query", "targetQueryValue": "5", "connection": "postgresql://username:password@localhost:5432", "metricName": "scaler_sql_data2"}},
	// dbName
	{metadata: map[string]string{"query": "test_query", "targetQueryValue": "5", "host": "test_host", "port": "test_port", "userName": "test_user_name", "dbName": "test_db_name", "sslmode": "test_ssl_mode"}},
	// dbName + metricName
	{metadata: map[string]string{"query": "test_query", "targetQueryValue": "5", "host": "test_host", "port": "test_port", "userName": "test_user_name", "dbName": "test_db_name", "sslmode": "test_ssl_mode", "metricName": "scaler_sql_data"}},
}

type postgreSQLMetricIdentifier struct {
	metadataTestData *parsePostgreSQLMetadataTestData
	resolvedEnv      map[string]string
	authParam        map[string]string
	scaleIndex       int
	name             string
}

var postgreSQLMetricIdentifiers = []postgreSQLMetricIdentifier{
	{&testPostgreSQLMetdata[0], map[string]string{"test_connection_string": "postgresql://localhost:5432"}, nil, 0, "s0-postgresql"},
	{&testPostgreSQLMetdata[1], map[string]string{"test_connection_string2": "postgresql://test@localhost"}, nil, 1, "s1-postgresql"},
}

func TestPosgresSQLGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range postgreSQLMetricIdentifiers {
		meta, err := parsePostgreSQLMetadata(&ScalerConfig{ResolvedEnv: testData.resolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.authParam, ScalerIndex: testData.scaleIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockPostgresSQLScaler := postgreSQLScaler{"", meta, nil, logr.Discard()}

		metricSpec := mockPostgresSQLScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

type postgreSQLConnectionStringTestData struct {
	metadata         map[string]string
	resolvedEnv      map[string]string
	authParam        map[string]string
	connectionString string
}

var testPostgreSQLConnectionstring = []postgreSQLConnectionStringTestData{
	// from environment
	{metadata: map[string]string{"query": "test_query", "targetQueryValue": "5", "connectionFromEnv": "CONNECTION_ENV"}, resolvedEnv: map[string]string{"CONNECTION_ENV": "test_connection_from_env"}, connectionString: "test_connection_from_env"},
	// from authentication
	{metadata: map[string]string{"query": "test_query", "targetQueryValue": "5"}, authParam: map[string]string{"connection": "test_connection_from_auth"}, connectionString: "test_connection_from_auth"},
	// from meta
	{metadata: map[string]string{"query": "test_query", "targetQueryValue": "5", "host": "localhost", "port": "1234", "dbName": "testDb", "userName": "user", "sslmode": "required"}, connectionString: "host=localhost port=1234 user=user dbname=testDb sslmode=required password="},
}

func TestPosgresSQLConnectionStringGeneration(t *testing.T) {
	for _, testData := range testPostgreSQLConnectionstring {
		meta, err := parsePostgreSQLMetadata(&ScalerConfig{ResolvedEnv: testData.resolvedEnv, TriggerMetadata: testData.metadata, AuthParams: testData.authParam, ScalerIndex: 0})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}

		if meta.connection != testData.connectionString {
			t.Errorf("Error generating connectionString, expected '%s' and get '%s'", testData.connectionString, meta.connection)
		}
	}
}

type parsePostgresMetadataTestData struct {
	metadata    map[string]string
	authParams  map[string]string
	resolvedEnv map[string]string
	raisesError bool
}

var testPostgresResolvedEnv = map[string]string{
	"POSTGRE_PASSWORD": "pass",
	"POSTGRE_CONN_STR": "test_conn_str",
}

var testPostgresMetadata = []parsePostgresMetadataTestData{
	// No metadata
	{
		metadata:    map[string]string{},
		authParams:  map[string]string{},
		resolvedEnv: map[string]string{},
		raisesError: true,
	},
	// connectionString
	{
		metadata:    map[string]string{"query": "query", "targetQueryValue": "12", "connectionFromEnv": "POSTGRE_CONN_STR"},
		authParams:  map[string]string{},
		resolvedEnv: testPostgresResolvedEnv,
		raisesError: false,
	},
	// Params instead of conn str
	{
		metadata:    map[string]string{"query": "query", "targetQueryValue": "12", "host": "test_host", "port": "test_port", "userName": "test_username", "passwordFromEnv": "POSTGRE_PASSWORD", "dbName": "test_dbname", "sslmode": "require"},
		authParams:  map[string]string{},
		resolvedEnv: testPostgresResolvedEnv,
		raisesError: false,
	},
	// Params from trigger authentication
	{
		metadata:    map[string]string{"query": "query", "targetQueryValue": "12"},
		authParams:  map[string]string{"host": "test_host", "port": "test_port", "userName": "test_username", "password": "POSTGRE_PASSWORD", "dbName": "test_dbname", "sslmode": "disable"},
		resolvedEnv: testPostgresResolvedEnv,
		raisesError: false,
	},
}

func TestParsePosgresSQLMetadata(t *testing.T) {
	for _, testData := range testPostgresMetadata {
		_, err := parsePostgreSQLMetadata(&ScalerConfig{ResolvedEnv: testData.resolvedEnv, TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		if err != nil && !testData.raisesError {
			t.Error("Expected success but got error", err)
		}
		if err == nil && testData.raisesError {
			t.Error("Expected error but got success")
		}
	}
}
