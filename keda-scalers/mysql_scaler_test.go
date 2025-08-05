package scalers

import (
	"testing"

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
)

var testMySQLResolvedEnv = map[string]string{
	"MYSQL_USERNAME": "test_username",
	"MYSQL_PASSWORD": "pass",
	"MYSQL_CONN_STR": "user@tcp(http://my.mysql.dev:3306)/stats_db",
}

type parseMySQLMetadataTestData struct {
	metadata    map[string]string
	resolvedEnv map[string]string
	authParams  map[string]string
	raisesError bool
}

type mySQLMetricIdentifier struct {
	metadataTestData *parseMySQLMetadataTestData
	triggerIndex     int
	metricName       string
}

var testMySQLMetadata = []parseMySQLMetadataTestData{
	// No metadata
	{
		metadata:    map[string]string{},
		authParams:  map[string]string{},
		resolvedEnv: testMySQLResolvedEnv,
		raisesError: true,
	},
	// connectionString
	{
		metadata:    map[string]string{"query": "query", "queryValue": "12", "connectionStringFromEnv": "MYSQL_CONN_STR"},
		authParams:  map[string]string{},
		resolvedEnv: testMySQLResolvedEnv,
		raisesError: false,
	},
	// Params instead of conn str
	{
		metadata:    map[string]string{"query": "query", "queryValue": "12", "host": "test_host", "port": "test_port", "username": "test_username", "passwordFromEnv": "MYSQL_PASSWORD", "dbName": "test_dbname"},
		authParams:  map[string]string{},
		resolvedEnv: testMySQLResolvedEnv,
		raisesError: false,
	},
	// Params instead of conn str with userFromEnv
	{
		metadata:    map[string]string{"query": "query", "queryValue": "12", "host": "test_host", "port": "test_port", "usernameFromEnv": "MYSQL_USERNAME", "passwordFromEnv": "MYSQL_PASSWORD", "dbName": "test_dbname"},
		authParams:  map[string]string{},
		resolvedEnv: testMySQLResolvedEnv,
		raisesError: false,
	},
	// Params from trigger authentication
	{
		metadata:    map[string]string{"query": "query", "queryValue": "12"},
		authParams:  map[string]string{"host": "test_host", "port": "test_port", "username": "test_username", "password": "MYSQL_PASSWORD", "dbName": "test_dbname"},
		resolvedEnv: testMySQLResolvedEnv,
		raisesError: false,
	},
	// Invalid activationQueryValue
	{
		metadata:    map[string]string{"query": "query", "queryValue": "12", "activationQueryValue": "AA"},
		authParams:  map[string]string{"host": "test_host", "port": "test_port", "username": "test_username", "password": "MYSQL_PASSWORD", "dbName": "test_dbname"},
		resolvedEnv: testMySQLResolvedEnv,
		raisesError: true,
	},
	// No username provided in authParams, metadata, resolvedEnv
	{
		metadata:    map[string]string{"query": "query", "queryValue": "12", "activationQueryValue": "AA"},
		authParams:  map[string]string{"host": "test_host", "port": "test_port", "password": "MYSQL_PASSWORD", "dbName": "test_dbname"},
		resolvedEnv: map[string]string{},
		raisesError: true,
	},
}

var mySQLMetricIdentifiers = []mySQLMetricIdentifier{
	{metadataTestData: &testMySQLMetadata[1], triggerIndex: 0, metricName: "s0-mysql-stats_db"},
	{metadataTestData: &testMySQLMetadata[2], triggerIndex: 1, metricName: "s1-mysql-test_dbname"},
}

func TestParseMySQLMetadata(t *testing.T) {
	for _, testData := range testMySQLMetadata {
		_, err := parseMySQLMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: testMySQLResolvedEnv, TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		if err != nil && !testData.raisesError {
			t.Error("Expected success but got error", err)
		}
		if err == nil && testData.raisesError {
			t.Error("Expected error but got success")
		}
	}
}

func TestMetadataToConnectionStrUseConnStr(t *testing.T) {
	// Use existing ConnStr
	testMeta := map[string]string{"query": "query", "queryValue": "12", "connectionStringFromEnv": "MYSQL_CONN_STR"}
	meta, _ := parseMySQLMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: testMySQLResolvedEnv, TriggerMetadata: testMeta, AuthParams: map[string]string{}})
	connStr := metadataToConnectionStr(meta)
	if connStr != testMySQLResolvedEnv["MYSQL_CONN_STR"] {
		t.Error("Expected success")
	}
}

func TestMetadataToConnectionStrBuildNew(t *testing.T) {
	// Build new ConnStr
	expected := "test_username:pass@tcp(test_host:test_port)/test_dbname"
	testMeta := map[string]string{"query": "query", "queryValue": "12", "host": "test_host", "port": "test_port", "username": "test_username", "passwordFromEnv": "MYSQL_PASSWORD", "dbName": "test_dbname"}
	meta, _ := parseMySQLMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: testMySQLResolvedEnv, TriggerMetadata: testMeta, AuthParams: map[string]string{}})
	connStr := metadataToConnectionStr(meta)
	if connStr != expected {
		t.Errorf("%s != %s", expected, connStr)
	}
}

func TestMySQLGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range mySQLMetricIdentifiers {
		meta, err := parseMySQLMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: testData.metadataTestData.resolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: nil, TriggerIndex: testData.triggerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		if meta.MetricName != testData.metricName {
			t.Error("Wrong External metric source name:", meta.MetricName)
		}
	}
}
