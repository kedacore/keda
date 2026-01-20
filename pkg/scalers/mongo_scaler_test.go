package scalers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"go.mongodb.org/mongo-driver/mongo"
	v2 "k8s.io/api/autoscaling/v2"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
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
	triggerIndex     int
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
	// mongodb srv support
	{
		metadata:    map[string]string{"query": `{"name":"John"}`, "collection": "demo", "queryValue": "12"},
		authParams:  map[string]string{"dbName": "test", "scheme": "mongodb+srv", "host": "localhost", "port": "", "username": "sample", "password": "sec@ure"},
		resolvedEnv: testMongoDBResolvedEnv,
		raisesError: false,
	},
	// test float queryValue
	{
		metadata:    map[string]string{"query": `{"name":"John"}`, "collection": "demo", "queryValue": "0.9"},
		authParams:  map[string]string{"dbName": "test", "scheme": "mongodb+srv", "host": "localhost", "port": "", "username": "sample", "password": "sec@ure"},
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
	// TLS enabled with CA only - should succeed
	{
		metadata:    map[string]string{"query": `{"name":"John"}`, "collection": "demo", "queryValue": "12", "connectionStringFromEnv": "MongoDB_CONN_STR", "dbName": "test"},
		authParams:  map[string]string{"tls": "enable", "ca": "cavalue"},
		resolvedEnv: testMongoDBResolvedEnv,
		raisesError: false,
	},
	// TLS enabled with cert and key - should succeed
	{
		metadata:    map[string]string{"query": `{"name":"John"}`, "collection": "demo", "queryValue": "12", "connectionStringFromEnv": "MongoDB_CONN_STR", "dbName": "test"},
		authParams:  map[string]string{"tls": "enable", "cert": "certvalue", "key": "keyvalue"},
		resolvedEnv: testMongoDBResolvedEnv,
		raisesError: false,
	},
	// TLS enabled with cert and key and CA - should succeed
	{
		metadata:    map[string]string{"query": `{"name":"John"}`, "collection": "demo", "queryValue": "12", "connectionStringFromEnv": "MongoDB_CONN_STR", "dbName": "test"},
		authParams:  map[string]string{"tls": "enable", "cert": "certvalue", "key": "keyvalue", "ca": "cavalue"},
		resolvedEnv: testMongoDBResolvedEnv,
		raisesError: false,
	},
	// TLS enabled with cert only - should fail (key required)
	{
		metadata:    map[string]string{"query": `{"name":"John"}`, "collection": "demo", "queryValue": "12", "connectionStringFromEnv": "MongoDB_CONN_STR", "dbName": "test"},
		authParams:  map[string]string{"tls": "enable", "cert": "certvalue"},
		resolvedEnv: testMongoDBResolvedEnv,
		raisesError: true,
	},
	// TLS enabled with key only - should fail (cert required)
	{
		metadata:    map[string]string{"query": `{"name":"John"}`, "collection": "demo", "queryValue": "12", "connectionStringFromEnv": "MongoDB_CONN_STR", "dbName": "test"},
		authParams:  map[string]string{"tls": "enable", "key": "keyvalue"},
		resolvedEnv: testMongoDBResolvedEnv,
		raisesError: true,
	},
	// TLS enabled without any certs - should fail
	{
		metadata:    map[string]string{"query": `{"name":"John"}`, "collection": "demo", "queryValue": "12", "connectionStringFromEnv": "MongoDB_CONN_STR", "dbName": "test"},
		authParams:  map[string]string{"tls": "enable"},
		resolvedEnv: testMongoDBResolvedEnv,
		raisesError: true,
	},
	// TLS disabled (default) - should succeed without certs
	{
		metadata:    map[string]string{"query": `{"name":"John"}`, "collection": "demo", "queryValue": "12", "connectionStringFromEnv": "MongoDB_CONN_STR", "dbName": "test"},
		authParams:  map[string]string{"tls": "disable"},
		resolvedEnv: testMongoDBResolvedEnv,
		raisesError: false,
	},
}

var mongoDBConnectionStringTestDatas = []mongoDBConnectionStringTestData{
	{metadataTestData: &testMONGODBMetadata[2], connectionString: "mongodb://mongodb0.example.com:27017"},
	{metadataTestData: &testMONGODBMetadata[3], connectionString: "mongodb://sample:test%40password@localhost:1234/test"},
	{metadataTestData: &testMONGODBMetadata[4], connectionString: "mongodb://sample:sec%40ure@localhost:1234/test"},
	{metadataTestData: &testMONGODBMetadata[5], connectionString: "mongodb+srv://sample:sec%40ure@localhost/test"},
}

var mongoDBMetricIdentifiers = []mongoDBMetricIdentifier{
	{metadataTestData: &testMONGODBMetadata[2], triggerIndex: 0, name: "s0-mongodb-demo"},
	{metadataTestData: &testMONGODBMetadata[2], triggerIndex: 1, name: "s1-mongodb-demo"},
}

func TestParseMongoDBMetadata(t *testing.T) {
	for _, testData := range testMONGODBMetadata {
		_, err := parseMongoDBMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: testData.resolvedEnv, TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
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
		_, err := parseMongoDBMetadata(&scalersconfig.ScalerConfig{
			ResolvedEnv:     testData.metadataTestData.resolvedEnv,
			TriggerMetadata: testData.metadataTestData.metadata,
			AuthParams:      testData.metadataTestData.authParams,
		})
		if err != nil {
			t.Error("Expected success but got error:", err)
		}
	}
}

func TestMongoDBGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range mongoDBMetricIdentifiers {
		meta, err := parseMongoDBMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: testData.metadataTestData.resolvedEnv, AuthParams: testData.metadataTestData.authParams, TriggerMetadata: testData.metadataTestData.metadata, TriggerIndex: testData.triggerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockMongoDBScaler := mongoDBScaler{metricType: v2.AverageValueMetricType, metadata: meta, client: &mongo.Client{}, logger: logr.Discard()}

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
