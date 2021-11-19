package scalers

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

const (
	testInvalidRestAPITemplate = "testInvalidRestAPITemplate"
)

type parseActiveMQMetadataTestData struct {
	metadata   map[string]string
	isError    bool
	authParams map[string]string
}

type activeMQMetricIdentifier struct {
	metadataTestData *parseActiveMQMetadataTestData
	scalerIndex      int
	name             string
}

// Setting metric identifier mock name
var activeMQMetricIdentifiers = []activeMQMetricIdentifier{
	{&testActiveMQMetadata[1], 0, "s0-activemq-testMetricName"},
	{&testActiveMQMetadata[2], 1, "s1-activemq-testQueue"},
}

var testActiveMQMetadata = []parseActiveMQMetadataTestData{
	// Nothing passed
	{map[string]string{}, true, map[string]string{}},
	// Properly formed metadata
	{map[string]string{"managementEndpoint": "localhost:8161", "destinationName": "testQueue", "brokerName": "localhost", "targetQueueSize": "10", "metricName": "testMetricName"}, false, map[string]string{"username": "testUsername", "password": "pass123"}},
	// no metricName passed, metricName is generated from destinationName
	{map[string]string{"managementEndpoint": "localhost:8161", "destinationName": "testQueue", "brokerName": "localhost", "targetQueueSize": "10"}, false, map[string]string{"username": "testUsername", "password": "pass123"}},
	// Invalid targetQueueSize using a string
	{map[string]string{"managementEndpoint": "localhost:8161", "destinationName": "testQueue", "brokerName": "localhost", "targetQueueSize": "AA", "metricName": "testMetricName"}, true, map[string]string{"username": "testUsername", "password": "pass123"}},
	// Missing management endpoint should fail
	{map[string]string{"destinationName": "testQueue", "brokerName": "localhost", "targetQueueSize": "10", "metricName": "testMetricName"}, true, map[string]string{"username": "testUsername", "password": "pass123"}},
	// Missing destination name, should fail
	{map[string]string{"managementEndpoint": "localhost:8161", "brokerName": "localhost", "targetQueueSize": "10", "metricName": "testMetricName"}, true, map[string]string{"username": "testUsername", "password": "pass123"}},
	// Missing broker name, should fail
	{map[string]string{"managementEndpoint": "localhost:8161", "destinationName": "testQueue", "targetQueueSize": "10", "metricName": "testMetricName"}, true, map[string]string{"username": "testUsername", "password": "pass123"}},
	// Missing username, should fail
	{map[string]string{"managementEndpoint": "localhost:8161", "destinationName": "testQueue", "brokerName": "localhost", "targetQueueSize": "10", "metricName": "testMetricName"}, true, map[string]string{"password": "pass123"}},
	// Missing password, should fail
	{map[string]string{"managementEndpoint": "localhost:8161", "destinationName": "testQueue", "brokerName": "localhost", "targetQueueSize": "10", "metricName": "testMetricName"}, true, map[string]string{"username": "testUsername"}},
	// Properly formed metadata with restAPITemplate
	{map[string]string{"restAPITemplate": "http://localhost:8161/api/jolokia/read/org.apache.activemq:type=Broker,brokerName=localhost,destinationType=Queue,destinationName=testQueue/QueueSize", "targetQueueSize": "10", "metricName": "testMetricName"}, false, map[string]string{"username": "testUsername", "password": "pass123"}},
	// Invalid restAPITemplate, should fail
	{map[string]string{"restAPITemplate": testInvalidRestAPITemplate, "targetQueueSize": "10", "metricName": "testMetricName"}, true, map[string]string{"username": "testUsername", "password": "pass123"}},
	// Missing username, should fail
	{map[string]string{"restAPITemplate": "http://localhost:8161/api/jolokia/read/org.apache.activemq:type=Broker,brokerName=localhost,destinationType=Queue,destinationName=testQueue/QueueSize", "targetQueueSize": "10", "metricName": "testMetricName"}, true, map[string]string{"password": "pass123"}},
	// Missing password, should fail
	{map[string]string{"restAPITemplate": "http://localhost:8161/api/jolokia/read/org.apache.activemq:type=Broker,brokerName=localhost,destinationType=Queue,destinationName=testQueue/QueueSize", "targetQueueSize": "10", "metricName": "testMetricName"}, true, map[string]string{"username": "testUsername"}},
}

func TestActiveMQParseMetadata(t *testing.T) {
	for _, testData := range testActiveMQMetadata {
		metadata, err := parseActiveMQMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
		if metadata != nil && metadata.password != "" && metadata.password != testData.authParams["password"] {
			t.Error("Expected password from configuration but found something else: ", metadata.password)
			fmt.Println(testData)
		}
	}
}

var testDefaultTargetQueueSize = []parseActiveMQMetadataTestData{
	{map[string]string{"managementEndpoint": "localhost:8161", "destinationName": "testQueue", "brokerName": "localhost"}, false, map[string]string{"username": "testUsername", "password": "pass123"}},
}

func TestParseDefaultTargetQueueSize(t *testing.T) {
	for _, testData := range testDefaultTargetQueueSize {
		metadata, err := parseActiveMQMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
		switch {
		case err != nil && !testData.isError:
			t.Error("Expected success but got error", err)
		case testData.isError && err == nil:
			t.Error("Expected error but got success")
		case metadata.targetQueueSize != defaultTargetQueueSize:
			t.Error("Expected default targetQueueSize =", defaultTargetQueueSize, "but got", metadata.targetQueueSize)
		}
	}
}

func TestActiveMQGetMetricsSpecForScaling(t *testing.T) {
	for _, testData := range activeMQMetricIdentifiers {
		metadata, err := parseActiveMQMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.metadataTestData.authParams, ScalerIndex: testData.scalerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockActiveMQScaler := activeMQScaler{
			metadata:   metadata,
			httpClient: http.DefaultClient,
		}

		metricSpec := mockActiveMQScaler.GetMetricSpecForScaling(context.Background())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Errorf("Wrong External metric source name: %s, expected: %s", metricName, testData.name)
		}
	}
}
