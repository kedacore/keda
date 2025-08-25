package scalers

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
)

type parseArangoDBMetadataTestData struct {
	metadata    map[string]string
	authParams  map[string]string
	raisesError bool
}

var testArangoDBMetadata = []parseArangoDBMetadataTestData{
	// No metadata
	{
		metadata:    map[string]string{},
		authParams:  map[string]string{},
		raisesError: true,
	},
	// missing query
	{
		metadata:    map[string]string{"endpoints": "https://localhost:8529", "collection": "demo", "queryValue": "12", "dbName": "test"},
		authParams:  map[string]string{},
		raisesError: true,
	},
	// with metric name
	{
		metadata:    map[string]string{"endpoints": "https://localhost:8529", "query": `FOR t IN testCollection FILTER t.cook_time == '3 hours' RETURN t`, "collection": "demo", "queryValue": "12", "dbName": "test"},
		authParams:  map[string]string{},
		raisesError: false,
	},
	// from trigger auth
	{
		metadata:    map[string]string{"endpoints": "https://localhost:8529", "query": `FOR t IN testCollection FILTER t.cook_time == '3 hours' RETURN t`, "collection": "demo", "queryValue": "12"},
		authParams:  map[string]string{"dbName": "test", "username": "sample", "password": "secure"},
		raisesError: false,
	},
	// wrong activationQueryValue
	{
		metadata:    map[string]string{"endpoints": "https://localhost:8529", "query": `FOR t IN testCollection FILTER t.cook_time == '3 hours' RETURN t`, "collection": "demo", "queryValue": "12", "activationQueryValue": "aa", "dbName": "test"},
		authParams:  map[string]string{},
		raisesError: true,
	},
}

type arangoDBAuthMetadataTestData struct {
	metadata    map[string]string
	authParams  map[string]string
	raisesError bool
}

var testArangoDBAuthMetadata = []arangoDBAuthMetadataTestData{
	// success bearer default
	{map[string]string{"endpoints": "https://http://34.162.13.192:8529,https://34.162.13.193:8529", "collection": "demo", "query": "FOR d IN myCollection RETURN d", "queryValue": "1", "dbName": "testdb", "authModes": "bearer"}, map[string]string{"bearerToken": "dummy-token"}, false},
	// fail bearerAuth with no token
	{map[string]string{"endpoints": "https://http://34.162.13.192:8529,https://34.162.13.193:8529", "collection": "demo", "query": "FOR d IN myCollection RETURN d", "queryValue": "1", "dbName": "testdb", "authModes": "bearer"}, map[string]string{}, true},
	// success basicAuth
	{map[string]string{"endpoints": "https://http://34.162.13.192:8529,https://34.162.13.193:8529", "collection": "demo", "query": "FOR d IN myCollection RETURN d", "queryValue": "1", "dbName": "testdb", "authModes": "basic"}, map[string]string{"username": "user", "password": "pass"}, false},
	// fail basicAuth with no username
	{map[string]string{"endpoints": "https://http://34.162.13.192:8529,https://34.162.13.193:8529", "collection": "demo", "query": "FOR d IN myCollection RETURN d", "queryValue": "1", "dbName": "testdb", "authModes": "basic"}, map[string]string{}, true},
	// success basicAuth with no password
	{map[string]string{"endpoints": "https://http://34.162.13.192:8529,https://34.162.13.193:8529", "collection": "demo", "query": "FOR d IN myCollection RETURN d", "queryValue": "1", "dbName": "testdb", "authModes": "basic"}, map[string]string{"username": "user"}, false},
}

type arangoDBMetricIdentifier struct {
	metadataTestData *parseArangoDBMetadataTestData
	triggerIndex     int
	name             string
}

var arangoDBMetricIdentifiers = []arangoDBMetricIdentifier{
	{metadataTestData: &testArangoDBMetadata[2], triggerIndex: 0, name: "s0-arangodb"},
	{metadataTestData: &testArangoDBMetadata[2], triggerIndex: 1, name: "s1-arangodb"},
}

func TestParseArangoDBMetadata(t *testing.T) {
	for idx, testData := range testArangoDBMetadata {
		_, err := parseArangoDBMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		if err != nil && !testData.raisesError {
			t.Errorf("Test %v: expected success but got error: %s", idx, err)
		}
		if err == nil && testData.raisesError {
			t.Error("Expected error but got success")
		}
	}
}

func TestArangoDBScalerAuthParams(t *testing.T) {
	for _, testData := range testArangoDBAuthMetadata {
		meta, err := parseArangoDBMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})

		if err != nil && !testData.raisesError {
			t.Error("Expected success but got error", err)
		}
		if testData.raisesError && err == nil {
			t.Error("Expected error but got success")
		}

		if err == nil {
			if meta.ArangoDBAuth.EnabledBasicAuth() && !strings.Contains(testData.metadata["authModes"], "basic") {
				t.Error("wrong auth mode detected")
			}
		}
	}
}

func TestArangoDBGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range arangoDBMetricIdentifiers {
		meta, err := parseArangoDBMetadata(&scalersconfig.ScalerConfig{
			AuthParams:      testData.metadataTestData.authParams,
			TriggerMetadata: testData.metadataTestData.metadata,
			TriggerIndex:    testData.triggerIndex,
		})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockArangoDBScaler := arangoDBScaler{"", meta, nil, logr.Discard()}

		metricSpec := mockArangoDBScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
