package scalers

import (
	"testing"
)

var testMySQLResolvedEnv = map[string]string{
	"MYSQL_PASSWORD": "pass",
	"MYSQL_CONN_STR": "test_conn_str",
}

type parseMySQLMetadataTestData struct {
	metadata    map[string]string
	raisesError bool
}

type mySQLMetricIdentifier struct {
	metadataTestData *parseMySQLMetadataTestData
	name             string
}

var testMySQLMetadata = []parseMySQLMetadataTestData{
	// No metadata
	{map[string]string{}, true},
	// connectionString
	{map[string]string{"query": "query", "queryValue": "12", "connectionString": "test_value"}, false},
	// Params instead of conn str
	{map[string]string{"query": "query", "queryValue": "12", "host": "test_host", "port": "test_port", "username": "test_username", "password": "test_password", "dbName": "test_dbname"}, false},
}

var mySQLMetricIdentifiers = []mySQLMetricIdentifier{
	{&testMySQLMetadata[1], "mysql-test_value"},
	{&testMySQLMetadata[2], "mysql-test_dbname"},
}

func TestParseMySQLMetadata(t *testing.T) {
	for _, testData := range testMySQLMetadata {
		_, err := parseMySQLMetadata(testMySQLResolvedEnv, testData.metadata, map[string]string{})
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
	meta, _ := parseMySQLMetadata(testMySQLResolvedEnv, testMeta, map[string]string{})
	connStr := metadataToConnectionStr(meta)
	if connStr != testMySQLResolvedEnv["MYSQL_CONN_STR"] {
		t.Error("Expected success")
	}
}

func TestMetadataToConnectionStrBuildNew(t *testing.T) {
	// Build new ConnStr
	expected := "test_username:pass@tcp(test_host:test_port)/test_dbname"
	testMeta := map[string]string{"query": "query", "queryValue": "12", "host": "test_host", "port": "test_port", "username": "test_username", "passwordFromEnv": "MYSQL_PASSWORD", "dbName": "test_dbname"}
	meta, _ := parseMySQLMetadata(testMySQLResolvedEnv, testMeta, map[string]string{})
	connStr := metadataToConnectionStr(meta)
	if connStr != expected {
		t.Errorf("%s != %s", expected, connStr)
	}
}

func TestMySQLGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range mySQLMetricIdentifiers {
		meta, err := parseMySQLMetadata(map[string]string{"test_value": "test_value"}, testData.metadataTestData.metadata, nil)
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockMySQLScaler := mySQLScaler{meta, nil}

		metricSpec := mockMySQLScaler.GetMetricSpecForScaling()
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
