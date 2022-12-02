package scalers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"go.mongodb.org/mongo-driver/mongo"
)

var testMongoDBResolvedEnv = map[string]string{
	"MongoDB_CONN_STR": "test_conn_str",
	"MongoDB_PASSWORD": "test",
}

type parseMongoDBMetadataTestData struct {
	metadata    map[string]string
	authParams  map[string]string
	resolvedEnv map[string]string
	raisesError bool
}

type mongoDBMetricIdentifier struct {
	metadataTestData *parseMongoDBMetadataTestData
	scalerIndex      int
	name             string
}

var testMONGODBMetadata = []parseMongoDBMetadataTestData{
	// No metadata
	{
		metadata:    map[string]string{},
		authParams:  map[string]string{},
		resolvedEnv: testMongoDBResolvedEnv,
		raisesError: true,
	},
	// connectionStringFromEnv
	{
		metadata:    map[string]string{"query": `{"name":"John"}`, "collection": "demo", "queryValue": "12", "connectionStringFromEnv": "Mongo_CONN_STR", "dbName": "test"},
		authParams:  map[string]string{},
		resolvedEnv: testMongoDBResolvedEnv,
		raisesError: false,
	},
	// with metric name
	{
		metadata:    map[string]string{"query": `{"name":"John"}`, "metricName": "hpa", "collection": "demo", "queryValue": "12", "connectionStringFromEnv": "Mongo_CONN_STR", "dbName": "test"},
		authParams:  map[string]string{},
		resolvedEnv: testMongoDBResolvedEnv,
		raisesError: false,
	},
	// from trigger auth
	{
		metadata:    map[string]string{"query": `{"name":"John"}`, "metricName": "hpa", "collection": "demo", "queryValue": "12"},
		authParams:  map[string]string{"dbName": "test", "host": "localshot", "port": "1234", "username": "sample", "password": "secure"},
		resolvedEnv: testMongoDBResolvedEnv,
		raisesError: false,
	},
	// wrong activationQueryValue
	{
		metadata:    map[string]string{"query": `{"name":"John"}`, "metricName": "hpa", "collection": "demo", "queryValue": "12", "activationQueryValue": "aa", "connectionStringFromEnv": "Mongo_CONN_STR", "dbName": "test"},
		authParams:  map[string]string{},
		resolvedEnv: testMongoDBResolvedEnv,
		raisesError: true,
	},
}

var mongoDBMetricIdentifiers = []mongoDBMetricIdentifier{
	{metadataTestData: &testMONGODBMetadata[2], scalerIndex: 0, name: "s0-mongodb-hpa"},
	{metadataTestData: &testMONGODBMetadata[2], scalerIndex: 1, name: "s1-mongodb-hpa"},
}

func TestParseMongoDBMetadata(t *testing.T) {
	for _, testData := range testMONGODBMetadata {
		_, _, err := parseMongoDBMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
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
		meta, _, err := parseMongoDBMetadata(&ScalerConfig{ResolvedEnv: testData.metadataTestData.resolvedEnv, AuthParams: testData.metadataTestData.authParams, TriggerMetadata: testData.metadataTestData.metadata, ScalerIndex: testData.scalerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockMongoDBScaler := mongoDBScaler{"", meta, &mongo.Client{}, logr.Discard()}

		metricSpec := mockMongoDBScaler.GetMetricSpecForScaling(context.Background())
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
