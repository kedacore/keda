package scalers

import (
	"testing"
)

var testMySQLResolvedEnv = map[string]string{
	"MYSQL_PASSWORD": "pass",
	"MYSQL_CONN_STR": "test_conn_str",
}

type parseMySQLMetadataTestData struct {
	metdadata   map[string]string
	raisesError bool
}

var testMySQLMetdata = []parseMySQLMetadataTestData{
	// No metadata
	{metdadata: map[string]string{}, raisesError: true},
	// connectionString
	{metdadata: map[string]string{"query": "query", "queryValue": "12", "connectionString": "test_value"}, raisesError: false},
	// Params instead of conn str
	{metdadata: map[string]string{"query": "query", "queryValue": "12", "host": "test_host", "port": "test_port", "username": "test_username", "password": "test_password", "dbName": "test_dbname"}, raisesError: false},
}

func TestParseMySQLMetadata(t *testing.T) {
	for _, testData := range testMySQLMetdata {
		_, err := parseMySQLMetadata(testMySQLResolvedEnv, testData.metdadata, map[string]string{})
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
	testMeta := map[string]string{"query": "query", "queryValue": "12", "connectionString": "MYSQL_CONN_STR"}
	meta, _ := parseMySQLMetadata(testMySQLResolvedEnv, testMeta, map[string]string{})
	connStr := metadataToConnectionStr(meta)
	if connStr != testMySQLResolvedEnv["MYSQL_CONN_STR"] {
		t.Error("Expected success")
	}
}

func TestMetadataToConnectionStrBuildNew(t *testing.T) {
	// Build new ConnStr
	expected := "test_username:pass@tcp(test_host:test_port)/test_dbname"
	testMeta := map[string]string{"query": "query", "queryValue": "12", "host": "test_host", "port": "test_port", "username": "test_username", "password": "MYSQL_PASSWORD", "dbName": "test_dbname"}
	meta, _ := parseMySQLMetadata(testMySQLResolvedEnv, testMeta, map[string]string{})
	connStr := metadataToConnectionStr(meta)
	if connStr != expected {
		t.Errorf("%s != %s", expected, connStr)
	}
}
