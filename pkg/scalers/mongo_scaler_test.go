package scalers

import (
	"testing"

	"go.mongodb.org/mongo-driver/mongo"
)

var testMongoDBResolvedEnv = map[string]string{
	"MongoDB_CONN_STR": "test_conn_str",
	"MongoDB_PASSWORD": "test",
}

type parseMongoDBMetadataTestData struct {
	metadata    map[string]string
	resolvedEnv map[string]string
	raisesError bool
}

type mongoDBMetricIdentifier struct {
	metadataTestData *parseMongoDBMetadataTestData
	name             string
}

var testMONGODBMetadata = []parseMongoDBMetadataTestData{
	// No metadata
	{
		metadata:    map[string]string{},
		resolvedEnv: testMongoDBResolvedEnv,
		raisesError: true,
	},
	// connectionStringFromEnv
	{
		metadata:    map[string]string{"query": `{"name":"John"}`, "collection": "demo", "queryValue": "12", "connectionStringFromEnv": "Mongo_CONN_STR", "dbName": "test"},
		resolvedEnv: testMongoDBResolvedEnv,
		raisesError: false,
	},
	// with metric name
	{
		metadata:    map[string]string{"query": `{"name":"John"}`, "metricName": "hpa", "collection": "demo", "queryValue": "12", "connectionStringFromEnv": "Mongo_CONN_STR", "dbName": "test"},
		resolvedEnv: testMongoDBResolvedEnv,
		raisesError: false,
	},
}

var mongoDBMetricIdentifiers = []mongoDBMetricIdentifier{
	{metadataTestData: &testMONGODBMetadata[2], name: "mongodb-hpa"},
}

func TestParseMongoDBMetadata(t *testing.T) {
	for _, testData := range testMONGODBMetadata {
		_, _, err := parseMongoDBMetadata(&ScalerConfig{TriggerMetadata: testData.metadata})
		if err != nil && !testData.raisesError {
			t.Error("Expected success but got error:", err)
		}
		if err == nil && testData.raisesError {
			t.Error("Expected error but got success")
		}
	}
}

func TestMongoDBGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range mongoDBMetricIdentifiers {
		meta, _, err := parseMongoDBMetadata(&ScalerConfig{ResolvedEnv: testData.metadataTestData.resolvedEnv, TriggerMetadata: testData.metadataTestData.metadata})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockMongoDBScaler := mongoDBScaler{meta, &mongo.Client{}}

		metricSpec := mockMongoDBScaler.GetMetricSpecForScaling()
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func TestJson2BsonDoc(t *testing.T) {
	var testJSON = `{"name":"carson"}`
	doc, err := json2BsonDoc(testJSON)
	if err != nil {
		t.Error("convert testJson to Bson.Doc err:", err)
	}
	if doc == nil {
		t.Error("the doc is nil")
	}
}
