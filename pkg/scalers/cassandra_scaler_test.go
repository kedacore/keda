package scalers

import (
	"errors"
	"fmt"
	"testing"

	"github.com/gocql/gocql"
)

type cassandraTestData struct {
	// test inputs
	metadata   map[string]string
	authParams map[string]string

	// expected outputs
	expectedMetricName       string
	expectedConsistency      gocql.Consistency
	expectedProtocolVersion  int
	expectedClusterIPAddress string
	expectedError            error
}

var testCassandraInputs = []cassandraTestData{
	// metricName written
	{
		metadata:           map[string]string{"query": "SELECT COUNT(*) FROM test_keyspace.test_table;", "targetQueryValue": "1", "username": "cassandra", "clusterIPAddress": "cassandra.test:9042", "keyspace": "test_keyspace", "metricName": "myMetric"},
		authParams:         map[string]string{"password": "Y2Fzc2FuZHJhCg=="},
		expectedMetricName: "cassandra-myMetric",
	},

	// keyspace written, no metricName
	{
		metadata:           map[string]string{"query": "SELECT COUNT(*) FROM test_keyspace.test_table;", "targetQueryValue": "1", "username": "cassandra", "clusterIPAddress": "cassandra.test:9042", "keyspace": "test_keyspace"},
		authParams:         map[string]string{"password": "Y2Fzc2FuZHJhCg=="},
		expectedMetricName: "cassandra-test_keyspace",
	},

	// metricName and keyspace written
	{
		metadata:           map[string]string{"query": "SELECT COUNT(*) FROM test_keyspace.test_table;", "targetQueryValue": "1", "username": "cassandra", "clusterIPAddress": "cassandra.test:9042", "metricName": "myMetric", "keyspace": "test_keyspace"},
		authParams:         map[string]string{"password": "Y2Fzc2FuZHJhCg=="},
		expectedMetricName: "cassandra-myMetric",
	},

	// consistency and protocolVersion not written
	{
		metadata:                map[string]string{"query": "SELECT COUNT(*) FROM test_keyspace.test_table;", "targetQueryValue": "1", "username": "cassandra", "clusterIPAddress": "cassandra.test:9042", "keyspace": "test_keyspace"},
		authParams:              map[string]string{"password": "Y2Fzc2FuZHJhCg=="},
		expectedConsistency:     gocql.One,
		expectedProtocolVersion: 4,
	},

	// Error: keyspace not written
	{
		metadata:      map[string]string{"query": "SELECT COUNT(*) FROM test_keyspace.test_table;", "targetQueryValue": "1", "username": "cassandra", "clusterIPAddress": "cassandra.test:9042", "metricName": "myMetric"},
		authParams:    map[string]string{"password": "Y2Fzc2FuZHJhCg=="},
		expectedError: errors.New("no keyspace given"),
	},

	// Error: missing query
	{
		metadata:      map[string]string{"targetQueryValue": "1", "username": "cassandra", "clusterIPAddress": "cassandra.test:9042", "keyspace": "test_keyspace"},
		authParams:    map[string]string{"password": "Y2Fzc2FuZHJhCg=="},
		expectedError: errors.New("no query given"),
	},

	// Error: missing targetQueryValue
	{
		metadata:      map[string]string{"query": "SELECT COUNT(*) FROM test_keyspace.test_table;", "username": "cassandra", "clusterIPAddress": "cassandra.test:9042", "keyspace": "test_keyspace"},
		authParams:    map[string]string{"password": "Y2Fzc2FuZHJhCg=="},
		expectedError: errors.New("no targetQueryValue given"),
	},

	// Error: missing username
	{
		metadata:      map[string]string{"query": "SELECT COUNT(*) FROM test_keyspace.test_table;", "targetQueryValue": "1", "clusterIPAddress": "cassandra.test:9042", "keyspace": "test_keyspace"},
		authParams:    map[string]string{"password": "Y2Fzc2FuZHJhCg=="},
		expectedError: errors.New("no username given"),
	},

	// Error: missing clusterIPAddress
	{
		metadata:      map[string]string{"query": "SELECT COUNT(*) FROM test_keyspace.test_table;", "targetQueryValue": "1", "username": "cassandra", "keyspace": "test_keyspace"},
		authParams:    map[string]string{"password": "Y2Fzc2FuZHJhCg=="},
		expectedError: errors.New("no cluster IP address given"),
	},

	// Error: missing port
	{
		metadata:      map[string]string{"query": "SELECT COUNT(*) FROM test_keyspace.test_table;", "clusterIPAddress": "cassandra.test", "targetQueryValue": "1", "username": "cassandra", "keyspace": "test_keyspace"},
		authParams:    map[string]string{"password": "Y2Fzc2FuZHJhCg=="},
		expectedError: errors.New("no port given"),
	},

	// Error: missing password
	{
		metadata:      map[string]string{"query": "SELECT COUNT(*) FROM test_keyspace.test_table;", "targetQueryValue": "1", "username": "cassandra", "clusterIPAddress": "cassandra.test:9042", "keyspace": "test_keyspace"},
		authParams:    map[string]string{},
		expectedError: errors.New("no password given"),
	},
}

func TestParseCassandraMetadata(t *testing.T) {
	for _, testData := range testCassandraInputs {
		var config = ScalerConfig{
			TriggerMetadata: testData.metadata,
			AuthParams:      testData.authParams,
		}

		outputMetadata, err := ParseCassandraMetadata(&config)
		fmt.Printf("Expected error '%v'\n", testData.expectedError)
		fmt.Printf("Got error '%v'\n", err)
		if err != nil {
			if testData.expectedError == nil {
				t.Errorf("Unexpected error parsing input metadata: %v", err)
			} else if testData.expectedError.Error() != err.Error() {
				t.Errorf("Expected error '%v' but got '%v'", testData.expectedError, err)
			}

			continue
		}

		expectedQuery := "SELECT COUNT(*) FROM test_keyspace.test_table;"
		if outputMetadata.query != expectedQuery {
			t.Errorf("Wrong query. Expected '%s' but got '%s'", expectedQuery, outputMetadata.query)
		}

		expectedTargetQueryValue := 1
		if outputMetadata.targetQueryValue != expectedTargetQueryValue {
			t.Errorf("Wrong targetQueryValue. Expected %d but got %d", expectedTargetQueryValue, outputMetadata.targetQueryValue)
		}

		expectedConsistency := gocql.One
		if testData.expectedConsistency != 0 && testData.expectedConsistency != outputMetadata.consistency {
			t.Errorf("Wrong consistency. Expected %d but got %d", expectedConsistency, outputMetadata.consistency)
		}

		expectedProtocolVersion := 4
		if testData.expectedProtocolVersion != 0 && testData.expectedProtocolVersion != outputMetadata.protocolVersion {
			t.Errorf("Wrong protocol version. Expected %d but got %d", expectedProtocolVersion, outputMetadata.protocolVersion)
		}

		expectedClusterIPAddress := "cassandra.test:9042"
		if testData.expectedClusterIPAddress != "" && testData.expectedClusterIPAddress != outputMetadata.clusterIPAddress {
			t.Errorf("Wrong clusterIPAddress. Expected %s but got %s", expectedClusterIPAddress, outputMetadata.clusterIPAddress)
		}

		if testData.expectedMetricName != "" && testData.expectedMetricName != outputMetadata.metricName {
			t.Errorf("Wrong metric name. Expected '%s' but got '%s'", testData.expectedMetricName, outputMetadata.metricName)
		}
	}
}
