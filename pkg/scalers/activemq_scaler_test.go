package scalers

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

const (
	testInvalidRestAPITemplate = "testInvalidRestAPITemplate"
	defaultTargetQueueSize     = 10
)

type parseActiveMQMetadataTestData struct {
	name       string
	metadata   map[string]string
	authParams map[string]string
	isError    bool
}

type activeMQMetricIdentifier struct {
	metadataTestData *parseActiveMQMetadataTestData
	triggerIndex     int
	name             string
}

// Setting metric identifier mock name
var activeMQMetricIdentifiers = []activeMQMetricIdentifier{
	{&testActiveMQMetadata[1], 0, "s0-activemq-testQueue"},
	{&testActiveMQMetadata[10], 1, "s1-activemq-testQueue"},
}

var testActiveMQMetadata = []parseActiveMQMetadataTestData{
	{
		name:       "nothing passed",
		metadata:   map[string]string{},
		authParams: map[string]string{},
		isError:    true,
	},
	{
		name: "properly formed metadata",
		metadata: map[string]string{
			"managementEndpoint":        "localhost:8161",
			"destinationName":           "testQueue",
			"brokerName":                "localhost",
			"targetQueueSize":           "10",
			"activationTargetQueueSize": "0",
		},
		authParams: map[string]string{
			"username": "testUsername",
			"password": "pass123",
		},
		isError: false,
	},
	{
		name: "no metricName passed, metricName is generated from destinationName",
		metadata: map[string]string{
			"managementEndpoint": "localhost:8161",
			"destinationName":    "testQueue",
			"brokerName":         "localhost",
			"targetQueueSize":    "10",
		},
		authParams: map[string]string{
			"username": "testUsername",
			"password": "pass123",
		},
		isError: false,
	},
	{
		name: "Invalid targetQueueSize using a string",
		metadata: map[string]string{
			"managementEndpoint": "localhost:8161",
			"destinationName":    "testQueue",
			"brokerName":         "localhost",
			"targetQueueSize":    "AA",
			"metricName":         "testMetricName",
		},
		authParams: map[string]string{
			"username": "testUsername",
			"password": "pass123",
		},
		isError: true,
	},
	{
		name: "Invalid activatingTargetQueueSize using a string",
		metadata: map[string]string{
			"managementEndpoint":        "localhost:8161",
			"destinationName":           "testQueue",
			"brokerName":                "localhost",
			"targetQueueSize":           "10",
			"activationTargetQueueSize": "AA",
			"metricName":                "testMetricName",
		},
		authParams: map[string]string{
			"username": "testUsername",
			"password": "pass123",
		},
		isError: true,
	},
	{
		name: "missing management endpoint should fail",
		metadata: map[string]string{
			"destinationName": "testQueue",
			"brokerName":      "localhost",
			"targetQueueSize": "10",
			"metricName":      "testMetricName",
		},
		authParams: map[string]string{
			"username": "testUsername",
			"password": "pass123",
		},
		isError: true,
	},
	{
		name: "missing destination name, should fail",
		metadata: map[string]string{
			"managementEndpoint": "localhost:8161",
			"brokerName":         "localhost",
			"targetQueueSize":    "10",
			"metricName":         "testMetricName",
		},
		authParams: map[string]string{
			"username": "testUsername",
			"password": "pass123",
		},
		isError: true,
	},
	{
		name: "missing broker name, should fail",
		metadata: map[string]string{
			"managementEndpoint": "localhost:8161",
			"destinationName":    "testQueue",
			"targetQueueSize":    "10",
			"metricName":         "testMetricName",
		},
		authParams: map[string]string{
			"username": "testUsername",
			"password": "pass123",
		},
		isError: true,
	},
	{
		name: "missing username, should fail",
		metadata: map[string]string{
			"managementEndpoint": "localhost:8161",
			"destinationName":    "testQueue",
			"brokerName":         "localhost",
			"targetQueueSize":    "10",
			"metricName":         "testMetricName",
		},
		authParams: map[string]string{
			"password": "pass123",
		},
		isError: true,
	},
	{
		name: "missing password, should fail",
		metadata: map[string]string{
			"managementEndpoint": "localhost:8161",
			"destinationName":    "testQueue",
			"brokerName":         "localhost",
			"targetQueueSize":    "10",
			"metricName":         "testMetricName",
		},
		authParams: map[string]string{
			"username": "testUsername",
		},
		isError: true,
	},
	{
		name: "properly formed metadata with restAPITemplate",
		metadata: map[string]string{
			"restAPITemplate": "http://localhost:8161/api/jolokia/read/org.apache.activemq:type=Broker,brokerName=localhost,destinationType=Queue,destinationName=testQueue/QueueSize",
			"targetQueueSize": "10",
			"metricName":      "testMetricName",
		},
		authParams: map[string]string{
			"username": "testUsername",
			"password": "pass123",
		},
		isError: false,
	},
	{
		name: "invalid restAPITemplate, should fail",
		metadata: map[string]string{
			"restAPITemplate": testInvalidRestAPITemplate,
			"targetQueueSize": "10",
			"metricName":      "testMetricName",
		},
		authParams: map[string]string{
			"username": "testUsername",
			"password": "pass123",
		},
		isError: true,
	},
	{
		name: "missing username, should fail",
		metadata: map[string]string{
			"restAPITemplate": "http://localhost:8161/api/jolokia/read/org.apache.activemq:type=Broker,brokerName=localhost,destinationType=Queue,destinationName=testQueue/QueueSize",
			"targetQueueSize": "10",
			"metricName":      "testMetricName",
		},
		authParams: map[string]string{
			"password": "pass123",
		},
		isError: true,
	},
	{
		name: "missing password, should fail",
		metadata: map[string]string{
			"restAPITemplate": "http://localhost:8161/api/jolokia/read/org.apache.activemq:type=Broker,brokerName=localhost,destinationType=Queue,destinationName=testQueue/QueueSize",
			"targetQueueSize": "10",
			"metricName":      "testMetricName",
		},
		authParams: map[string]string{
			"username": "testUsername",
		},
		isError: true,
	},
}

func TestActiveMQDefaultCorsHeader(t *testing.T) {
	metadata := map[string]string{"managementEndpoint": "localhost:8161", "destinationName": "queue1", "brokerName": "broker-activemq", "username": "myUserName", "password": "myPassword"}
	meta, err := parseActiveMQMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: metadata, AuthParams: nil})

	if err != nil {
		t.Error("Expected success but got error", err)
	}
	if meta.CorsHeader != "http://localhost:8161" {
		t.Errorf("Expected http://localhost:8161 but got %s", meta.CorsHeader)
	}
}

func TestActiveMQCorsHeader(t *testing.T) {
	metadata := map[string]string{"managementEndpoint": "localhost:8161", "destinationName": "queue1", "brokerName": "broker-activemq", "username": "myUserName", "password": "myPassword", "corsHeader": "test"}
	meta, err := parseActiveMQMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: metadata, AuthParams: nil})

	if err != nil {
		t.Error("Expected success but got error", err)
	}
	if meta.CorsHeader != "test" {
		t.Errorf("Expected test but got %s", meta.CorsHeader)
	}
}

func TestParseActiveMQMetadata(t *testing.T) {
	for _, testData := range testActiveMQMetadata {
		t.Run(testData.name, func(t *testing.T) {
			metadata, err := parseActiveMQMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
			if err != nil && !testData.isError {
				t.Error("Expected success but got error", err)
			}
			if testData.isError && err == nil {
				t.Error("Expected error but got success")
			}
			if metadata != nil && metadata.Password != "" && metadata.Password != testData.authParams["password"] {
				t.Error("Expected password from configuration but found something else: ", metadata.Password)
				fmt.Println(testData)
			}
		})
	}
}

var testDefaultTargetQueueSize = []parseActiveMQMetadataTestData{
	{
		name: "properly formed metadata",
		metadata: map[string]string{
			"managementEndpoint": "localhost:8161",
			"destinationName":    "testQueue",
			"brokerName":         "localhost",
		},
		authParams: map[string]string{
			"username": "testUsername",
			"password": "pass123",
		},
		isError: false,
	},
}

func TestParseDefaultTargetQueueSize(t *testing.T) {
	for _, testData := range testDefaultTargetQueueSize {
		t.Run(testData.name, func(t *testing.T) {
			metadata, err := parseActiveMQMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams})
			switch {
			case err != nil && !testData.isError:
				t.Error("Expected success but got error", err)
			case testData.isError && err == nil:
				t.Error("Expected error but got success")
			case metadata.TargetQueueSize != defaultTargetQueueSize:
				t.Error("Expected default targetQueueSize =", defaultTargetQueueSize, "but got", metadata.TargetQueueSize)
			}
		})
	}
}

func TestActiveMQGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range activeMQMetricIdentifiers {
		ctx := context.Background()
		metadata, err := parseActiveMQMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, AuthParams: testData.metadataTestData.authParams, TriggerIndex: testData.triggerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockActiveMQScaler := activeMQScaler{
			metadata:   metadata,
			httpClient: http.DefaultClient,
		}

		metricSpec := mockActiveMQScaler.GetMetricSpecForScaling(ctx)
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Errorf("Wrong External metric source name: %s, expected: %s", metricName, testData.name)
		}
	}
}

type getMonitoringEndpointTestData struct {
	metadata map[string]string
	expected string
}

var getMonitoringEndpointData = []getMonitoringEndpointTestData{
	{
		expected: "http://localhost:8161/api/jolokia/read/org.apache.activemq:type=Broker,brokerName=localhost,destinationType=Queue,destinationName=testQueue/QueueSize",
		metadata: map[string]string{
			"managementEndpoint": "localhost:8161",
			"destinationName":    "testQueue",
			"brokerName":         "localhost",
			"targetQueueSize":    "10",
		},
	},
	{
		expected: "https://myBrokerHost:8162/api/jolokia/read/org.apache.activemq:type=Broker,brokerName=myBrokerName,destinationType=Queue,destinationName=keda-test/QueueSize",
		metadata: map[string]string{
			"targetQueueSize": "10",
			"restAPITemplate": "https://myBrokerHost:8162/api/jolokia/read/org.apache.activemq:type=Broker,brokerName=myBrokerName,destinationType=Queue,destinationName=keda-test/QueueSize",
		},
	},
}

func TestActiveMQGetMonitoringEndpoint(t *testing.T) {
	authParams := map[string]string{
		"username": "testUsername",
		"password": "pass123",
	}
	for _, testData := range getMonitoringEndpointData {
		metadata, err := parseActiveMQMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: authParams, TriggerIndex: 0})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockActiveMQScaler := activeMQScaler{
			metadata:   metadata,
			httpClient: http.DefaultClient,
		}

		endpoint, err := mockActiveMQScaler.getMonitoringEndpoint()
		if err != nil {
			t.Fatal("Could not get the endpoint:", err)
		}

		if endpoint != testData.expected {
			t.Errorf("Wrong endpoint: %s, expected: %s", endpoint, testData.expected)
		}
	}
}
