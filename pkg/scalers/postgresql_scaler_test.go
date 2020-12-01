package scalers

import (
	"testing"
)

type parsePostgreSQLMetadataTestData struct {
	metadata map[string]string
}

type postgreSQLMetricIdentifier struct {
	metadataTestData *parsePostgreSQLMetadataTestData
	resolvedEnv      map[string]string
	authParam        map[string]string
	name             string
}

var testPostgreSQLMetdata = []parsePostgreSQLMetadataTestData{
	// connection with username and password
	{metadata: map[string]string{"query": "test_query", "targetQueryValue": "5", "connectionFromEnv": "test_connection_string"}},
	// connection with username
	{metadata: map[string]string{"query": "test_query", "targetQueryValue": "5", "connectionFromEnv": "test_connection_string2"}},
	// connection without username and password
	{metadata: map[string]string{"query": "test_query", "targetQueryValue": "5", "connection": "postgresql://localhost:5432"}},
	// connection with password + metricname
	{metadata: map[string]string{"query": "test_query", "targetQueryValue": "5", "connection": "postgresql://username:password@localhost:5432", "metricName": "scaler_sql_data2"}},
	// dbName
	{metadata: map[string]string{"query": "test_query", "targetQueryValue": "5", "host": "test_host", "port": "test_port", "userName": "test_user_name", "dbName": "test_db_name", "sslmode": "test_ssl_mode"}},
	// dbName + metricName
	{metadata: map[string]string{"query": "test_query", "targetQueryValue": "5", "host": "test_host", "port": "test_port", "userName": "test_user_name", "dbName": "test_db_name", "sslmode": "test_ssl_mode", "metricName": "scaler_sql_data"}},
}

var postgreSQLMetricIdentifiers = []postgreSQLMetricIdentifier{
	{&testPostgreSQLMetdata[0], map[string]string{"test_connection_string": "postgresql://localhost:5432"}, nil, "postgresql-postgresql---localhost-5432"},
	{&testPostgreSQLMetdata[1], map[string]string{"test_connection_string2": "postgresql://test@localhost"}, nil, "postgresql-postgresql---test@localhost"},
	{&testPostgreSQLMetdata[2], nil, map[string]string{"connection": "postgresql://user:password@localhost:5432/dbname"}, "postgresql-postgresql---user-xxx@localhost-5432-dbname"},
	{&testPostgreSQLMetdata[3], nil, map[string]string{"connection": "postgresql://Username123:secret@localhost"}, "postgresql-scaler_sql_data2"},
	{&testPostgreSQLMetdata[4], nil, map[string]string{"connection": "postgresql://user:password@localhost:5432/dbname?app_name=test"}, "postgresql-postgresql---user-xxx@localhost-5432-dbname?app_name=test"},
	{&testPostgreSQLMetdata[5], nil, map[string]string{"connection": "postgresql://Username123:secret@localhost"}, "postgresql-scaler_sql_data"},
}

func TestPosgresSQLGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range postgreSQLMetricIdentifiers {
		meta, err := parsePostgreSQLMetadata(&ScalerConfig{ResolvedEnv: testData.resolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.authParam})
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
