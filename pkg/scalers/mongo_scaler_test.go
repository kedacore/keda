package scalers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
)

var testMongoDBResolvedEnv = map[string]string{
	"MongoDB_CONN_STR": "mongodb://mongodb0.example.com:27017",
	"MongoDB_PASSWORD": "test@password",
}

type parseMongoDBMetadataTestData struct {
	metadata    map[string]string
	authParams  map[string]string
	resolvedEnv map[string]string
	raisesError bool
}

type mongoDBConnectionStringTestData struct {
	metadataTestData *parseMongoDBMetadataTestData
	connectionString string
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
		metadata:    map[string]string{"query": `{"name":"John"}`, "collection": "demo", "queryValue": "12", "connectionStringFromEnv": "MongoDB_CONN_STR", "dbName": "test"},
		authParams:  map[string]string{},
		resolvedEnv: testMongoDBResolvedEnv,
		raisesError: false,
	},
	// with metric name
	{
		metadata:    map[string]string{"query": `{"name":"John"}`, "collection": "demo", "queryValue": "12", "connectionStringFromEnv": "MongoDB_CONN_STR", "dbName": "test"},
		authParams:  map[string]string{},
		resolvedEnv: testMongoDBResolvedEnv,
		raisesError: false,
	},
	// from passwordFromEnv
	{
		metadata:    map[string]string{"query": `{"name":"John"}`, "collection": "demo", "queryValue": "12", "passwordFromEnv": "MongoDB_PASSWORD"},
		authParams:  map[string]string{"dbName": "test", "host": "localhost", "port": "1234", "username": "sample"},
		resolvedEnv: testMongoDBResolvedEnv,
		raisesError: false,
	},
	// from trigger auth
	{
		metadata:    map[string]string{"query": `{"name":"John"}`, "collection": "demo", "queryValue": "12"},
		authParams:  map[string]string{"dbName": "test", "host": "localhost", "port": "1234", "username": "sample", "password": "sec@ure"},
		resolvedEnv: testMongoDBResolvedEnv,
		raisesError: false,
	},
	// wrong activationQueryValue
	{
		metadata:    map[string]string{"query": `{"name":"John"}`, "collection": "demo", "queryValue": "12", "activationQueryValue": "aa", "connectionStringFromEnv": "Mongo_CONN_STR", "dbName": "test"},
		authParams:  map[string]string{},
		resolvedEnv: testMongoDBResolvedEnv,
		raisesError: true,
	},
}

var mongoDBConnectionStringTestDatas = []mongoDBConnectionStringTestData{
	{metadataTestData: &testMONGODBMetadata[2], connectionString: "mongodb://mongodb0.example.com:27017"},
	{metadataTestData: &testMONGODBMetadata[3], connectionString: "mongodb://sample:test%40password@localhost:1234/test"},
	{metadataTestData: &testMONGODBMetadata[4], connectionString: "mongodb://sample:sec%40ure@localhost:1234/test"},
}

var mongoDBMetricIdentifiers = []mongoDBMetricIdentifier{
	{metadataTestData: &testMONGODBMetadata[2], scalerIndex: 0, name: "s0-mongodb-demo"},
	{metadataTestData: &testMONGODBMetadata[2], scalerIndex: 1, name: "s1-mongodb-demo"},
}

func TestParseMongoDBMetadata(t *testing.T) {
	for _, testData := range testMONGODBMetadata {
		_, _, err := parseMongoDBMetadata(&ScalerConfig{ResolvedEnv: testData.resolvedEnv, TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		if err != nil && !testData.raisesError {
			t.Error("Expected success but got error:", err)
		}
		if err == nil && testData.raisesError {
			t.Error("Expected error but got success")
		}
	}
}

func TestParseMongoDBConnectionString(t *testing.T) {
	for _, testData := range mongoDBConnectionStringTestDatas {
		_, connStr, err := parseMongoDBMetadata(&ScalerConfig{ResolvedEnv: testData.metadataTestData.resolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.metadataTestData.authParams})
		if err != nil {
			t.Error("Expected success but got error:", err)
		}
		assert.Equal(t, testData.connectionString, connStr)
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
			t.Error("Wrong External metric source name:", metricName, "Expected", testData.name)
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
