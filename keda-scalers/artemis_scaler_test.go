package scalers

import (
	"context"
	"net/http"
	"testing"

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
)

const (
	username = "myUserName"
	password = "myPassword"
)

type parseArtemisMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

type artemisMetricIdentifier struct {
	metadataTestData *parseArtemisMetadataTestData
	triggerIndex     int
	name             string
}

var sampleArtemisResolvedEnv = map[string]string{
	username: "artemis",
	password: "artemis",
}

// A complete valid authParams with username and passwd
var artemisAuthParams = map[string]string{
	"username": "admin",
	"password": "admin",
}

// An invalid authParams without username and passwd
var emptyArtemisAuthParams = map[string]string{
	"username": "",
	"password": "",
}

var testArtemisMetadata = []parseArtemisMetadataTestData{
	// nothing passed
	{map[string]string{}, true},
	// Missing missing managementEndpoint should fail
	{map[string]string{"managementEndpoint": "", "queueName": "queue1", "brokerName": "broker-activemq", "brokerAddress": "address1", "username": "myUserName", "password": "myPassword"}, true},
	// Missing queue name, should fail
	{map[string]string{"managementEndpoint": "localhost:8161", "queueName": "", "brokerName": "broker-activemq", "brokerAddress": "address1", "username": "myUserName", "password": "myPassword"}, true},
	// Missing broker name, should fail
	{map[string]string{"managementEndpoint": "localhost:8161", "queueName": "queue1", "brokerName": "", "brokerAddress": "address1", "username": "myUserName", "password": "myPassword"}, true},
	// Missing broker address, should fail
	{map[string]string{"managementEndpoint": "localhost:8161", "queueName": "queue1", "brokerName": "broker-activemq", "brokerAddress": "", "username": "myUserName", "password": "myPassword"}, true},
	// Missing username, should fail
	{map[string]string{"managementEndpoint": "localhost:8161", "queueName": "queue1", "brokerName": "broker-activemq", "brokerAddress": "test", "username": "", "password": "myPassword"}, true},
	// Missing password, should fail
	{map[string]string{"managementEndpoint": "localhost:8161", "queueName": "queue1", "brokerName": "broker-activemq", "brokerAddress": "test", "username": "myUserName", "password": ""}, true},
	{map[string]string{"managementEndpoint": "localhost:8161", "queueName": "queue1", "brokerName": "broker-activemq", "brokerAddress": "test", "username": "myUserName", "password": "myPassword"}, false},
	{map[string]string{"restApiTemplate": "http://localhost:8161/console/jolokia/read/org.apache.activemq.artemis:broker=\"broker-activemq\",component=addresses,address=\"test\",subcomponent=queues,routing-type=\"anycast\",queue=\"queue1\"/MessageCount", "username": "myUserName", "password": "myPassword"}, false},
	// Missing brokername , should fail
	{map[string]string{"restApiTemplate": "http://localhost:8161/console/jolokia/read/org.apache.activemq.artemis:broker=\"\",component=addresses,address=\"test\",subcomponent=queues,routing-type=\"anycast\",queue=\"queue1\"/MessageCount", "username": "myUserName", "password": "myPassword"}, true},
}

var artemisMetricIdentifiers = []artemisMetricIdentifier{
	{&testArtemisMetadata[7], 0, "s0-artemis-queue1"},
	{&testArtemisMetadata[7], 1, "s1-artemis-queue1"},
}

var testArtemisMetadataWithEmptyAuthParams = []parseArtemisMetadataTestData{
	// nothing passed
	{map[string]string{}, true},
	// Missing managementEndpoint should fail
	{map[string]string{"managementEndpoint": "", "queueName": "queue1", "brokerName": "broker-activemq", "brokerAddress": "address1"}, true},
	// Missing queue name, should fail
	{map[string]string{"managementEndpoint": "localhost:8161", "queueName": "", "brokerName": "broker-activemq", "brokerAddress": "address1"}, true},
	// Missing broker name, should fail
	{map[string]string{"managementEndpoint": "localhost:8161", "queueName": "queue1", "brokerName": "", "brokerAddress": "address1"}, true},
	// Missing broker address, should fail
	{map[string]string{"managementEndpoint": "localhost:8161", "queueName": "queue1", "brokerName": "broker-activemq", "brokerAddress": ""}, true},
	// Missing username or password, should fail
	{map[string]string{"managementEndpoint": "localhost:8161", "queueName": "queue1", "brokerName": "broker-activemq", "brokerAddress": "test"}, true},
}

var testArtemisMetadataWithAuthParams = []parseArtemisMetadataTestData{
	{map[string]string{"managementEndpoint": "localhost:8161", "queueName": "queue1", "brokerName": "broker-activemq", "brokerAddress": "test"}, false},
}

func TestArtemisDefaultCorsHeader(t *testing.T) {
	metadata := map[string]string{"managementEndpoint": "localhost:8161", "queueName": "queue1", "brokerName": "broker-activemq", "brokerAddress": "test", "username": "myUserName", "password": "myPassword"}
	meta, err := parseArtemisMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: sampleArtemisResolvedEnv, TriggerMetadata: metadata, AuthParams: nil})

	if err != nil {
		t.Error("Expected success but got error", err)
	}
	if !(meta.CorsHeader == "http://localhost:8161") {
		t.Errorf("Expected http://localhost:8161 but got %s", meta.CorsHeader)
	}
}

func TestArtemisCorsHeader(t *testing.T) {
	metadata := map[string]string{"managementEndpoint": "localhost:8161", "queueName": "queue1", "brokerName": "broker-activemq", "brokerAddress": "test", "username": "myUserName", "password": "myPassword", "corsHeader": "test"}
	meta, err := parseArtemisMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: sampleArtemisResolvedEnv, TriggerMetadata: metadata, AuthParams: nil})

	if err != nil {
		t.Error("Expected success but got error", err)
	}
	if !(meta.CorsHeader == "test") {
		t.Errorf("Expected test but got %s", meta.CorsHeader)
	}
}

func TestArtemisParseMetadata(t *testing.T) {
	for _, testData := range testArtemisMetadata {
		_, err := parseArtemisMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: sampleArtemisResolvedEnv, TriggerMetadata: testData.metadata, AuthParams: nil})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}

	// test with missing auth params should all fail
	for _, testData := range testArtemisMetadataWithEmptyAuthParams {
		_, err := parseArtemisMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: sampleArtemisResolvedEnv, TriggerMetadata: testData.metadata, AuthParams: emptyArtemisAuthParams})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}

	// test with complete auth params should not fail
	for _, testData := range testArtemisMetadataWithAuthParams {
		_, err := parseArtemisMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: sampleArtemisResolvedEnv, TriggerMetadata: testData.metadata, AuthParams: artemisAuthParams})
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}

func TestArtemisGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range artemisMetricIdentifiers {
		ctx := context.Background()
		meta, err := parseArtemisMetadata(&scalersconfig.ScalerConfig{ResolvedEnv: sampleArtemisResolvedEnv, TriggerMetadata: testData.metadataTestData.metadata, AuthParams: nil, TriggerIndex: testData.triggerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockArtemisScaler := artemisScaler{
			metadata:   meta,
			httpClient: http.DefaultClient,
		}

		metricSpec := mockArtemisScaler.GetMetricSpecForScaling(ctx)
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
