package scalers

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// Test host URLs for validation
const (
	testValidMQQueueURL   = "https://qmtest.qm2.eu-gb.mq.appdomain.cloud/ibmmq/rest/v2/admin/action/qmgr/QM1/mqsc"
	testInvalidMQQueueURL = "testInvalidURL.com"
)

// Test data struct used for TestIBMMQParseMetadata
type parseIBMMQMetadataTestData struct {
	metadata   map[string]string
	isError    bool
	authParams map[string]string
}

var sampleIBMMQResolvedEnv = map[string]string{
	username: "ibmmquser",
	password: "ibmmqpass",
}

// Test metric identifier with test MQ data and it's name
type IBMMQMetricIdentifier struct {
	metadataTestData *parseIBMMQMetadataTestData
	triggerIndex     int
	name             string
}

// Setting metric identifier mock name
var IBMMQMetricIdentifiers = []IBMMQMetricIdentifier{
	{&testIBMMQMetadata[1], 0, "s0-ibmmq-testQueue"},
	{&testIBMMQMetadata[1], 1, "s1-ibmmq-testQueue"},
}

// Test cases for TestIBMMQParseMetadata test
var testIBMMQMetadata = []parseIBMMQMetadataTestData{
	// Nothing passed
	{map[string]string{}, true, map[string]string{}},
	// Properly formed metadata
	{map[string]string{"host": testValidMQQueueURL, "queueManager": "testQueueManager", "queueName": "testQueue", "queueDepth": "10"}, false, map[string]string{"username": "testUsername", "password": "Pass123"}},
	// Invalid queueDepth using a string
	{map[string]string{"host": testValidMQQueueURL, "queueManager": "testQueueManager", "queueName": "testQueue", "queueDepth": "AA"}, true, map[string]string{"username": "testUsername", "password": "Pass123"}},
	// Invalid activationQueueDepth using a string
	{map[string]string{"host": testValidMQQueueURL, "queueManager": "testQueueManager", "queueName": "testQueue", "queueDepth": "1", "activationQueueDepth": "AA"}, true, map[string]string{"username": "testUsername", "password": "Pass123"}},
	// No host provided
	{map[string]string{"queueManager": "testQueueManager", "queueName": "testQueue", "queueDepth": "10"}, true, map[string]string{"username": "testUsername", "password": "Pass123"}},
	// Missing queueManager
	{map[string]string{"host": testValidMQQueueURL, "queueName": "testQueue", "queueDepth": "10"}, true, map[string]string{"username": "testUsername", "password": "Pass123"}},
	// Missing queueName
	{map[string]string{"host": testValidMQQueueURL, "queueManager": "testQueueManager", "queueDepth": "10"}, true, map[string]string{"username": "testUsername", "password": "Pass123"}},
	// Invalid URL
	{map[string]string{"host": testInvalidMQQueueURL, "queueManager": "testQueueManager", "queueName": "testQueue", "queueDepth": "10"}, true, map[string]string{"username": "testUsername", "password": "Pass123"}},
	// Properly formed authParams
	{map[string]string{"host": testValidMQQueueURL, "queueManager": "testQueueManager", "queueName": "testQueue", "queueDepth": "10"}, false, map[string]string{"username": "testUsername", "password": "Pass123"}},
	// No username provided
	{map[string]string{"host": testValidMQQueueURL, "queueManager": "testQueueManager", "queueName": "testQueue", "queueDepth": "10"}, true, map[string]string{"password": "Pass123"}},
	// No password provided
	{map[string]string{"host": testValidMQQueueURL, "queueManager": "testQueueManager", "queueName": "testQueue", "queueDepth": "10"}, true, map[string]string{"username": "testUsername"}},
}

// Test MQ Connection metadata is parsed correctly
// should error on missing required field
// and verify that the password field is handled correctly.
func TestIBMMQParseMetadata(t *testing.T) {
	for _, testData := range testIBMMQMetadata {
		metadata, err := parseIBMMQMetadata(&ScalerConfig{ResolvedEnv: sampleIBMMQResolvedEnv, TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
			fmt.Println(testData)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
			fmt.Println(testData)
		}
		if metadata != nil && metadata.password != "" && metadata.password != testData.authParams["password"] {
			t.Error("Expected password from configuration but found something else: ", metadata.password)
			fmt.Println(testData)
		}
	}
}

// Test case for TestParseDefaultQueueDepth test
var testDefaultQueueDepth = []parseIBMMQMetadataTestData{
	{map[string]string{"host": testValidMQQueueURL, "queueManager": "testQueueManager", "queueName": "testQueue"}, false, map[string]string{"username": "testUsername", "password": "Pass123"}},
}

// Test that DefaultQueueDepth is set when queueDepth is not provided
func TestParseDefaultQueueDepth(t *testing.T) {
	for _, testData := range testDefaultQueueDepth {
		metadata, err := parseIBMMQMetadata(&ScalerConfig{ResolvedEnv: sampleIBMMQResolvedEnv, TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		switch {
		case err != nil && !testData.isError:
			t.Error("Expected success but got error", err)
		case testData.isError && err == nil:
			t.Error("Expected error but got success")
		case metadata.queueDepth != defaultTargetQueueDepth:
			t.Error("Expected default queueDepth =", defaultTargetQueueDepth, "but got", metadata.queueDepth)
		}
	}
}

// Create a scaler and check if metrics method is available
func TestIBMMQGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range IBMMQMetricIdentifiers {
		metadata, err := parseIBMMQMetadata(&ScalerConfig{ResolvedEnv: sampleIBMMQResolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.metadataTestData.authParams, TriggerIndex: testData.triggerIndex})
		httpTimeout := 100 * time.Millisecond

		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockIBMMQScaler := IBMMQScaler{
			metadata:           metadata,
			defaultHTTPTimeout: httpTimeout,
		}
		metricSpec := mockIBMMQScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name

		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
