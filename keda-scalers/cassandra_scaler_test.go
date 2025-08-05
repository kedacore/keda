package scalers

import (
	"context"
	"os"
	"testing"

	"github.com/go-logr/logr"
	"github.com/gocql/gocql"
	"github.com/stretchr/testify/assert"
	v2 "k8s.io/api/autoscaling/v2"

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
)

type parseCassandraMetadataTestData struct {
	name       string
	metadata   map[string]string
	authParams map[string]string
	isError    bool
}

type parseCassandraTLSTestData struct {
	name       string
	authParams map[string]string
	isError    bool
	tlsEnabled bool
}

type cassandraMetricIdentifier struct {
	name             string
	metadataTestData *parseCassandraMetadataTestData
	triggerIndex     int
	metricName       string
}

var testCassandraMetadata = []parseCassandraMetadataTestData{
	{
		name:       "nothing passed",
		metadata:   map[string]string{},
		authParams: map[string]string{},
		isError:    true,
	},
	{
		name: "everything passed verbatim",
		metadata: map[string]string{
			"query":            "SELECT COUNT(*) FROM test_keyspace.test_table;",
			"targetQueryValue": "1",
			"username":         "cassandra",
			"port":             "9042",
			"clusterIPAddress": "cassandra.test",
			"keyspace":         "test_keyspace",
		},
		authParams: map[string]string{"password": "Y2Fzc2FuZHJhCg=="},
		isError:    false,
	},
	{
		name: "metricName from keyspace",
		metadata: map[string]string{
			"query":            "SELECT COUNT(*) FROM test_keyspace.test_table;",
			"targetQueryValue": "1",
			"username":         "cassandra",
			"clusterIPAddress": "cassandra.test:9042",
			"keyspace":         "test_keyspace",
		},
		authParams: map[string]string{"password": "Y2Fzc2FuZHJhCg=="},
		isError:    false,
	},
	{
		name: "no query",
		metadata: map[string]string{
			"targetQueryValue": "1",
			"username":         "cassandra",
			"clusterIPAddress": "cassandra.test:9042",
			"keyspace":         "test_keyspace",
		},
		authParams: map[string]string{"password": "Y2Fzc2FuZHJhCg=="},
		isError:    true,
	},
	{
		name: "no targetQueryValue",
		metadata: map[string]string{
			"query":            "SELECT COUNT(*) FROM test_keyspace.test_table;",
			"username":         "cassandra",
			"clusterIPAddress": "cassandra.test:9042",
			"keyspace":         "test_keyspace",
		},
		authParams: map[string]string{"password": "Y2Fzc2FuZHJhCg=="},
		isError:    true,
	},
	{
		name: "no username",
		metadata: map[string]string{
			"query":            "SELECT COUNT(*) FROM test_keyspace.test_table;",
			"targetQueryValue": "1",
			"clusterIPAddress": "cassandra.test:9042",
			"keyspace":         "test_keyspace",
		},
		authParams: map[string]string{"password": "Y2Fzc2FuZHJhCg=="},
		isError:    true,
	},
	{
		name: "no port",
		metadata: map[string]string{
			"query":            "SELECT COUNT(*) FROM test_keyspace.test_table;",
			"targetQueryValue": "1",
			"username":         "cassandra",
			"clusterIPAddress": "cassandra.test",
			"keyspace":         "test_keyspace",
		},
		authParams: map[string]string{"password": "Y2Fzc2FuZHJhCg=="},
		isError:    true,
	},
	{
		name: "no clusterIPAddress",
		metadata: map[string]string{
			"query":            "SELECT COUNT(*) FROM test_keyspace.test_table;",
			"targetQueryValue": "1",
			"username":         "cassandra",
			"port":             "9042",
			"keyspace":         "test_keyspace",
		},
		authParams: map[string]string{"password": "Y2Fzc2FuZHJhCg=="},
		isError:    true,
	},
	{
		name: "no keyspace",
		metadata: map[string]string{
			"query":            "SELECT COUNT(*) FROM test_keyspace.test_table;",
			"targetQueryValue": "1",
			"username":         "cassandra",
			"clusterIPAddress": "cassandra.test:9042",
		},
		authParams: map[string]string{"password": "Y2Fzc2FuZHJhCg=="},
		isError:    true,
	},
	{
		name: "no password",
		metadata: map[string]string{
			"query":            "SELECT COUNT(*) FROM test_keyspace.test_table;",
			"targetQueryValue": "1",
			"username":         "cassandra",
			"clusterIPAddress": "cassandra.test:9042",
			"keyspace":         "test_keyspace",
		},
		authParams: map[string]string{},
		isError:    true,
	},
	{
		name: "with https prefix",
		metadata: map[string]string{
			"query":            "SELECT COUNT(*) FROM test_keyspace.test_table;",
			"targetQueryValue": "1",
			"username":         "cassandra",
			"port":             "9042",
			"clusterIPAddress": "https://cassandra.test",
			"keyspace":         "test_keyspace",
		},
		authParams: map[string]string{"password": "Y2Fzc2FuZHJhCg=="},
		isError:    false,
	},
}

var tlsAuthParamsTestData = []parseCassandraTLSTestData{
	{
		name: "success with cert/key",
		authParams: map[string]string{
			"tls":      "enable",
			"cert":     "test-cert",
			"key":      "test-key",
			"password": "Y2Fzc2FuZHJhCg==",
		},
		isError:    false,
		tlsEnabled: true,
	},
	{
		name: "failure missing cert",
		authParams: map[string]string{
			"tls":      "enable",
			"key":      "test-key",
			"password": "Y2Fzc2FuZHJhCg==",
		},
		isError:    true,
		tlsEnabled: false,
	},
	{
		name: "failure missing key",
		authParams: map[string]string{
			"tls":      "enable",
			"cert":     "test-cert",
			"password": "Y2Fzc2FuZHJhCg==",
		},
		isError:    true,
		tlsEnabled: false,
	},
	{
		name: "failure invalid tls value",
		authParams: map[string]string{
			"tls":      "yes",
			"cert":     "test-cert",
			"key":      "test-key",
			"password": "Y2Fzc2FuZHJhCg==",
		},
		isError:    true,
		tlsEnabled: false,
	},
}

var cassandraMetricIdentifiers = []cassandraMetricIdentifier{
	{
		name:             "everything passed verbatim",
		metadataTestData: &testCassandraMetadata[1],
		triggerIndex:     0,
		metricName:       "s0-cassandra-test_keyspace",
	},
	{
		name:             "metricName from keyspace",
		metadataTestData: &testCassandraMetadata[2],
		triggerIndex:     1,
		metricName:       "s1-cassandra-test_keyspace",
	},
}

var successMetaData = map[string]string{
	"query":            "SELECT COUNT(*) FROM test_keyspace.test_table;",
	"targetQueryValue": "1",
	"username":         "cassandra",
	"clusterIPAddress": "cassandra.test:9042",
	"keyspace":         "test_keyspace",
}

func TestCassandraParseMetadata(t *testing.T) {
	for _, testData := range testCassandraMetadata {
		t.Run(testData.name, func(t *testing.T) {
			_, err := parseCassandraMetadata(&scalersconfig.ScalerConfig{
				TriggerMetadata: testData.metadata,
				AuthParams:      testData.authParams,
			})
			if err != nil && !testData.isError {
				t.Error("Expected success but got error", err)
			}
			if testData.isError && err == nil {
				t.Error("Expected error but got success")
			}
		})
	}
}

func TestCassandraGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range cassandraMetricIdentifiers {
		t.Run(testData.name, func(t *testing.T) {
			meta, err := parseCassandraMetadata(&scalersconfig.ScalerConfig{
				TriggerMetadata: testData.metadataTestData.metadata,
				TriggerIndex:    testData.triggerIndex,
				AuthParams:      testData.metadataTestData.authParams,
			})
			if err != nil {
				t.Fatal("Could not parse metadata:", err)
			}
			mockCassandraScaler := cassandraScaler{
				metricType: v2.AverageValueMetricType,
				metadata:   meta,
				session:    &gocql.Session{},
				logger:     logr.Discard(),
			}

			metricSpec := mockCassandraScaler.GetMetricSpecForScaling(context.Background())
			metricName := metricSpec[0].External.Metric.Name
			assert.Equal(t, testData.metricName, metricName)
		})
	}
}

func TestParseCassandraTLS(t *testing.T) {
	for _, testData := range tlsAuthParamsTestData {
		t.Run(testData.name, func(t *testing.T) {
			meta, err := parseCassandraMetadata(&scalersconfig.ScalerConfig{
				TriggerMetadata: successMetaData,
				AuthParams:      testData.authParams,
			})

			if testData.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testData.tlsEnabled, meta.TLS == "enable")

				if meta.TLS == "enable" {
					// Verify cert contents
					if testData.authParams["cert"] != "" {
						data, err := os.ReadFile(meta.Cert)
						assert.NoError(t, err)
						assert.Equal(t, testData.authParams["cert"], string(data))
						// Cleanup
						defer os.Remove(meta.Cert)
					}

					// Verify key contents
					if testData.authParams["key"] != "" {
						data, err := os.ReadFile(meta.Key)
						assert.NoError(t, err)
						assert.Equal(t, testData.authParams["key"], string(data))
						// Cleanup
						defer os.Remove(meta.Key)
					}

					// Verify CA contents if present
					if testData.authParams["ca"] != "" {
						data, err := os.ReadFile(meta.CA)
						assert.NoError(t, err)
						assert.Equal(t, testData.authParams["ca"], string(data))
						// Cleanup
						defer os.Remove(meta.CA)
					}
				}
			}
		})
	}
}
