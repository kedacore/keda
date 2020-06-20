package scalers

import (
	"testing"
)

const (
	username = "myUserName"
	password = "myPassword"
)

type parseArtemisMetadataTestData struct {
	metadata map[string]string
	isError  bool
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
}

var testArtemisMetadataWithEmptyAuthParams = []parseArtemisMetadataTestData{
	// nothing passed
	{map[string]string{}, true},
	// Missing missing managementEndpoint should fail
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

func TestArtemisParseMetadata(t *testing.T) {
	for _, testData := range testArtemisMetadata {
		_, err := parseArtemisMetadata(sampleArtemisResolvedEnv, testData.metadata, nil)
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}

	// test with missing auth params should all fail
	for _, testData := range testArtemisMetadataWithEmptyAuthParams {
		_, err := parseArtemisMetadata(sampleArtemisResolvedEnv, testData.metadata, emptyArtemisAuthParams)
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}

	// test with complete auth params should not fail
	for _, testData := range testArtemisMetadataWithAuthParams {
		_, err := parseArtemisMetadata(sampleArtemisResolvedEnv, testData.metadata, artemisAuthParams)
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}
