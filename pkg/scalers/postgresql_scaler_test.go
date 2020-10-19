package scalers

import (
	"testing"
)

type parsePostgreSQLMetadataTestData struct {
	metadata map[string]string
}

type postgreSQLMetricIdentifier struct {
	metadataTestData *parsePostgreSQLMetadataTestData
	name             string
}

var testPostgreSQLMetdata = []parsePostgreSQLMetadataTestData{
	// connection
	{metadata: map[string]string{"query": "test_query", "targetQueryValue": "5", "connectionFromEnv": "test_connection_string"}},
	// dbName
	{metadata: map[string]string{"query": "test_query", "targetQueryValue": "5", "host": "test_host", "port": "test_port", "userName": "test_user_name", "dbName": "test_db_name", "sslmode": "test_ssl_mode"}},
}

var postgreSQLMetricIdentifiers = []postgreSQLMetricIdentifier{
	{&testPostgreSQLMetdata[0], "postgresql-test_connection_string"},
	{&testPostgreSQLMetdata[1], "postgresql-test_db_name"},
}

func TestPosgresSQLGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range postgreSQLMetricIdentifiers {
		meta, err := parsePostgreSQLMetadata(&ScalerConfig{ResolvedEnv: map[string]string{"test_connection_string": "test_connection_string"}, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: nil})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockPostgresSQLScaler := postgreSQLScaler{meta, nil}

		metricSpec := mockPostgresSQLScaler.GetMetricSpecForScaling()
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
