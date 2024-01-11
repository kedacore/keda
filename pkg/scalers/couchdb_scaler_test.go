package scalers

import (
	"context"
	"testing"

	_ "github.com/go-kivik/couchdb/v3"
	"github.com/go-kivik/kivik/v3"
	"github.com/go-logr/logr"
)

var testCouchDBResolvedEnv = map[string]string{
	"CouchDB_CONN_STR": "http://admin:YeFvQno9LylIm5MDgwcV@localhost:5984/",
	"CouchDB_PASSWORD": "YeFvQno9LylIm5MDgwcV",
}

type parseCouchDBMetadataTestData struct {
	metadata    map[string]string
	authParams  map[string]string
	resolvedEnv map[string]string
	raisesError bool
}

type couchDBMetricIdentifier struct {
	metadataTestData *parseCouchDBMetadataTestData
	triggerIndex     int
	name             string
}

var testCOUCHDBMetadata = []parseCouchDBMetadataTestData{
	// No metadata
	{
		metadata:    map[string]string{},
		authParams:  map[string]string{},
		resolvedEnv: testCouchDBResolvedEnv,
		raisesError: true,
	},
	// connectionStringFromEnv
	{
		metadata:    map[string]string{"query": `{ "selector": { "feet": { "$gt": 0 } }, "fields": ["_id", "feet", "greeting"] }`, "queryValue": "1", "connectionStringFromEnv": "CouchDB_CONN_STR", "dbName": "animals"},
		authParams:  map[string]string{},
		resolvedEnv: testCouchDBResolvedEnv,
		raisesError: false,
	},
	// with metric name
	{
		metadata:    map[string]string{"query": `{ "selector": { "feet": { "$gt": 0 } }, "fields": ["_id", "feet", "greeting"] }`, "queryValue": "1", "connectionStringFromEnv": "CouchDB_CONN_STR", "dbName": "animals"},
		authParams:  map[string]string{},
		resolvedEnv: testCouchDBResolvedEnv,
		raisesError: false,
	},
	// from trigger auth
	{
		metadata:    map[string]string{"query": `{ "selector": { "feet": { "$gt": 0 } }, "fields": ["_id", "feet", "greeting"] }`, "queryValue": "1"},
		authParams:  map[string]string{"dbName": "animals", "host": "localhost", "port": "5984", "username": "admin", "password": "YeFvQno9LylIm5MDgwcV"},
		resolvedEnv: testCouchDBResolvedEnv,
		raisesError: false,
	},
	// wrong activationQueryValue
	{
		metadata:    map[string]string{"query": `{ "selector": { "feet": { "$gt": 0 } }, "fields": ["_id", "feet", "greeting"] }`, "queryValue": "1", "activationQueryValue": "1", "connectionStringFromEnv": "CouchDB_CONN_STR", "dbName": "animals"},
		authParams:  map[string]string{},
		resolvedEnv: testCouchDBResolvedEnv,
		raisesError: true,
	},
}

var couchDBMetricIdentifiers = []couchDBMetricIdentifier{
	{metadataTestData: &testCOUCHDBMetadata[2], triggerIndex: 0, name: "s0-coucdb-animals"},
	{metadataTestData: &testCOUCHDBMetadata[2], triggerIndex: 1, name: "s1-coucdb-animals"},
}

func TestParseCouchDBMetadata(t *testing.T) {
	for _, testData := range testCOUCHDBMetadata {
		_, _, err := parseCouchDBMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		if err != nil && !testData.raisesError {
			t.Error("Expected success but got error:", err)
		}
	}
}

func TestCouchDBGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range couchDBMetricIdentifiers {
		meta, _, err := parseCouchDBMetadata(&ScalerConfig{ResolvedEnv: testData.metadataTestData.resolvedEnv, AuthParams: testData.metadataTestData.authParams, TriggerMetadata: testData.metadataTestData.metadata, TriggerIndex: testData.triggerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockCouchDBScaler := couchDBScaler{"", meta, &kivik.Client{}, logr.Discard()}

		metricSpec := mockCouchDBScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
