package scalers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/gocql/gocql"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type parseCassandraMetadataTestData struct {
	metadata   map[string]string
	isError    bool
	authParams map[string]string
}

type parseCassandraTLSTestData struct {
	authParams map[string]string
	isError    bool
	enableTLS  bool
}

type cassandraMetricIdentifier struct {
	metadataTestData *parseCassandraMetadataTestData
	triggerIndex     int
	name             string
}

// Assuming TLS details in not passed. TLS is set to false by default
var testCassandraMetadata = []parseCassandraMetadataTestData{
	// nothing passed
	{map[string]string{}, true, map[string]string{}},
	// everything is passed in verbatim
	{map[string]string{"query": "SELECT COUNT(*) FROM test_keyspace.test_table;", "targetQueryValue": "1", "username": "cassandra", "port": "9042", "clusterIPAddress": "cassandra.test", "keyspace": "test_keyspace", "TriggerIndex": "0"}, false, map[string]string{"password": "Y2Fzc2FuZHJhCg=="}},
	// metricName is generated from keyspace
	{map[string]string{"query": "SELECT COUNT(*) FROM test_keyspace.test_table;", "targetQueryValue": "1", "username": "cassandra", "clusterIPAddress": "cassandra.test:9042", "keyspace": "test_keyspace", "TriggerIndex": "0"}, false, map[string]string{"password": "Y2Fzc2FuZHJhCg=="}},
	// no query passed
	{map[string]string{"targetQueryValue": "1", "username": "cassandra", "clusterIPAddress": "cassandra.test:9042", "keyspace": "test_keyspace", "TriggerIndex": "0"}, true, map[string]string{"password": "Y2Fzc2FuZHJhCg=="}},
	// no targetQueryValue passed
	{map[string]string{"query": "SELECT COUNT(*) FROM test_keyspace.test_table;", "username": "cassandra", "clusterIPAddress": "cassandra.test:9042", "keyspace": "test_keyspace", "TriggerIndex": "0"}, true, map[string]string{"password": "Y2Fzc2FuZHJhCg=="}},
	// no username passed
	{map[string]string{"query": "SELECT COUNT(*) FROM test_keyspace.test_table;", "targetQueryValue": "1", "clusterIPAddress": "cassandra.test:9042", "keyspace": "test_keyspace", "TriggerIndex": "0"}, true, map[string]string{"password": "Y2Fzc2FuZHJhCg=="}},
	// no port passed
	{map[string]string{"query": "SELECT COUNT(*) FROM test_keyspace.test_table;", "targetQueryValue": "1", "username": "cassandra", "clusterIPAddress": "cassandra.test", "keyspace": "test_keyspace", "TriggerIndex": "0"}, true, map[string]string{"password": "Y2Fzc2FuZHJhCg=="}},
	// no clusterIPAddress passed
	{map[string]string{"query": "SELECT COUNT(*) FROM test_keyspace.test_table;", "targetQueryValue": "1", "username": "cassandra", "port": "9042", "keyspace": "test_keyspace", "TriggerIndex": "0"}, true, map[string]string{"password": "Y2Fzc2FuZHJhCg=="}},
	// no keyspace passed
	{map[string]string{"query": "SELECT COUNT(*) FROM test_keyspace.test_table;", "targetQueryValue": "1", "username": "cassandra", "clusterIPAddress": "cassandra.test:9042", "TriggerIndex": "0"}, true, map[string]string{"password": "Y2Fzc2FuZHJhCg=="}},
	// no password passed
	{map[string]string{"query": "SELECT COUNT(*) FROM test_keyspace.test_table;", "targetQueryValue": "1", "username": "cassandra", "clusterIPAddress": "cassandra.test:9042", "keyspace": "test_keyspace", "TriggerIndex": "0"}, true, map[string]string{}},
	// fix issue[4110] passed
	{map[string]string{"query": "SELECT COUNT(*) FROM test_keyspace.test_table;", "targetQueryValue": "1", "username": "cassandra", "port": "9042", "clusterIPAddress": "https://cassandra.test", "keyspace": "test_keyspace", "TriggerIndex": "0"}, false, map[string]string{"password": "Y2Fzc2FuZHJhCg=="}},
}

var tlsAuthParamsTestData = []parseCassandraTLSTestData{
	// success, TLS cert/key
	{map[string]string{"tls": "enable", "cert": "ceert", "key": "keey", "password": "Y2Fzc2FuZHJhCg=="}, false, true},
	// failure, TLS missing cert
	{map[string]string{"tls": "enable", "key": "keey", "password": "Y2Fzc2FuZHJhCg=="}, true, false},
	// failure, TLS missing key
	{map[string]string{"tls": "enable", "cert": "ceert", "password": "Y2Fzc2FuZHJhCg=="}, true, false},
	// failure, TLS invalid
	{map[string]string{"tls": "yes", "cert": "ceert", "key": "keey", "password": "Y2Fzc2FuZHJhCg=="}, true, false},
}

var cassandraMetricIdentifiers = []cassandraMetricIdentifier{
	{&testCassandraMetadata[1], 0, "s0-cassandra-test_keyspace"},
	{&testCassandraMetadata[2], 1, "s1-cassandra-test_keyspace"},
}

func TestCassandraParseMetadata(t *testing.T) {
	testCaseNum := 1
	for _, testData := range testCassandraMetadata {
		_, err := parseCassandraMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		if err != nil && !testData.isError {
			t.Errorf("Expected success but got error for unit test # %v", testCaseNum)
		}
		if testData.isError && err == nil {
			t.Errorf("Expected error but got success for unit test # %v", testCaseNum)
		}
		testCaseNum++
	}
}

func TestCassandraGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range cassandraMetricIdentifiers {
		meta, err := parseCassandraMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, TriggerIndex: testData.triggerIndex, AuthParams: testData.metadataTestData.authParams})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		cluster := gocql.NewCluster(meta.clusterIPAddress)
		session, _ := cluster.CreateSession()
		mockCassandraScaler := cassandraScaler{"", meta, session, logr.Discard()}

		metricSpec := mockCassandraScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Errorf("Wrong External metric source name: %s, expected: %s", metricName, testData.name)
		}
	}
}

var successMetaData = map[string]string{"query": "SELECT COUNT(*) FROM test_keyspace.test_table;", "targetQueryValue": "1", "username": "cassandra", "clusterIPAddress": "cassandra.test:9042", "keyspace": "test_keyspace", "TriggerIndex": "0"}

func TestParseCassandraTLS(t *testing.T) {
	for _, testData := range tlsAuthParamsTestData {
		meta, err := parseCassandraMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: successMetaData, AuthParams: testData.authParams})

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
		if meta.enableTLS != testData.enableTLS {
			t.Errorf("Expected enableTLS to be set to %v but got %v\n", testData.enableTLS, meta.enableTLS)
		}
		if meta.enableTLS {
			if meta.cert != testData.authParams["cert"] {
				t.Errorf("Expected cert to be set to %v but got %v\n", testData.authParams["cert"], meta.cert)
			}
			if meta.key != testData.authParams["key"] {
				t.Errorf("Expected key to be set to %v but got %v\n", testData.authParams["key"], meta.key)
			}
		}
	}
}
