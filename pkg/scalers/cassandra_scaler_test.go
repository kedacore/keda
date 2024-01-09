package scalers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/gocql/gocql"
)

type parseCassandraMetadataTestData struct {
	metadata   map[string]string
	isError    bool
	authParams map[string]string
}

type cassandraMetricIdentifier struct {
	metadataTestData *parseCassandraMetadataTestData
	triggerIndex     int
	name             string
}

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

var cassandraMetricIdentifiers = []cassandraMetricIdentifier{
	{&testCassandraMetadata[1], 0, "s0-cassandra-test_keyspace"},
	{&testCassandraMetadata[2], 1, "s1-cassandra-test_keyspace"},
}

func TestCassandraParseMetadata(t *testing.T) {
	testCaseNum := 1
	for _, testData := range testCassandraMetadata {
		_, err := parseCassandraMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
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
		meta, err := parseCassandraMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, TriggerIndex: testData.triggerIndex, AuthParams: testData.metadataTestData.authParams})
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
