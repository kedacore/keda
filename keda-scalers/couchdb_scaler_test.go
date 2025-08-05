package scalers

import (
	"context"
	"testing"

	_ "github.com/go-kivik/couchdb/v3"
	"github.com/go-kivik/kivik/v3"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
)

var testCouchDBResolvedEnv = map[string]string{
	"CouchDB_CONN_STR": "http://admin:YeFvQno9LylIm5MDgwcV@localhost:5984/",
	"CouchDB_PASSWORD": "YeFvQno9LylIm5MDgwcV",
}

type parseCouchDBMetadataTestData struct {
	name        string
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
		name:        "no metadata",
		metadata:    map[string]string{},
		authParams:  map[string]string{},
		resolvedEnv: testCouchDBResolvedEnv,
		raisesError: true,
	},
	// connectionStringFromEnv
	{
		name:        "with connectionStringFromEnv",
		metadata:    map[string]string{"query": `{ "selector": { "feet": { "$gt": 0 } }, "fields": ["_id", "feet", "greeting"] }`, "queryValue": "1", "connectionStringFromEnv": "CouchDB_CONN_STR", "dbName": "animals"},
		authParams:  map[string]string{},
		resolvedEnv: testCouchDBResolvedEnv,
		raisesError: false,
	},
	// with metric name
	{
		name:        "with metric name",
		metadata:    map[string]string{"query": `{ "selector": { "feet": { "$gt": 0 } }, "fields": ["_id", "feet", "greeting"] }`, "queryValue": "1", "connectionStringFromEnv": "CouchDB_CONN_STR", "dbName": "animals"},
		authParams:  map[string]string{},
		resolvedEnv: testCouchDBResolvedEnv,
		raisesError: false,
	},
	// from trigger auth
	{
		name:        "from trigger auth",
		metadata:    map[string]string{"query": `{ "selector": { "feet": { "$gt": 0 } }, "fields": ["_id", "feet", "greeting"] }`, "queryValue": "1"},
		authParams:  map[string]string{"dbName": "animals", "host": "localhost", "port": "5984", "username": "admin", "password": "YeFvQno9LylIm5MDgwcV"},
		resolvedEnv: testCouchDBResolvedEnv,
		raisesError: false,
	},
	// wrong activationQueryValue
	{
		name:        "wrong activationQueryValue",
		metadata:    map[string]string{"query": `{ "selector": { "feet": { "$gt": 0 } }, "fields": ["_id", "feet", "greeting"] }`, "queryValue": "1", "activationQueryValue": "a", "connectionStringFromEnv": "CouchDB_CONN_STR", "dbName": "animals"},
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
		t.Run(testData.name, func(t *testing.T) {
			_, err := parseCouchDBMetadata(&scalersconfig.ScalerConfig{
				TriggerMetadata: testData.metadata,
				AuthParams:      testData.authParams,
				ResolvedEnv:     testData.resolvedEnv,
			})
			if err != nil && !testData.raisesError {
				t.Errorf("Test case '%s': Expected success but got error: %v", testData.name, err)
			}
			if testData.raisesError && err == nil {
				t.Errorf("Test case '%s': Expected error but got success", testData.name)
			}
		})
	}
}

func TestCouchDBGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range couchDBMetricIdentifiers {
		t.Run(testData.name, func(t *testing.T) {
			meta, err := parseCouchDBMetadata(&scalersconfig.ScalerConfig{
				ResolvedEnv:     testData.metadataTestData.resolvedEnv,
				AuthParams:      testData.metadataTestData.authParams,
				TriggerMetadata: testData.metadataTestData.metadata,
				TriggerIndex:    testData.triggerIndex,
			})
			if err != nil {
				t.Fatal("Could not parse metadata:", err)
			}

			mockCouchDBScaler := couchDBScaler{
				metricType: v2.AverageValueMetricType,
				metadata:   meta,
				client:     &kivik.Client{},
				logger:     logr.Discard(),
			}

			metricSpec := mockCouchDBScaler.GetMetricSpecForScaling(context.Background())
			metricName := metricSpec[0].External.Metric.Name
			if metricName != testData.name {
				t.Errorf("Wrong External metric source name: %s, expected: %s", metricName, testData.name)
			}
		})
	}
}
