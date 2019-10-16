package scalers

import (
	"context"
	"os"
	"testing"
)

const (
	topicName         = "testtopic"
	subscriptionName  = "testsubscription"
	queueName         = "testqueue"
	connectionSetting = "none"
)

type parseServiceBusMetadataTestData struct {
	metadata   map[string]string
	isError    bool
	entityType EntityType
	authParams map[string]string
}

// not testing connections so it doesn't matter what the resolved env value is for this
var sampleResolvedEnv = map[string]string{
	connectionSetting: "none",
}

var parseServiceBusMetadataDataset = []parseServiceBusMetadataTestData{
	{map[string]string{}, true, None, map[string]string{}},
	// properly formed queue
	{map[string]string{"queueName": queueName, "connection": connectionSetting}, false, Queue, map[string]string{}},
	// properly formed topic & subscription
	{map[string]string{"topicName": topicName, "subscriptionName": subscriptionName, "connection": connectionSetting}, false, Subscription, map[string]string{}},
	// queue and topic specified
	{map[string]string{"queueName": queueName, "topicName": topicName, "connection": connectionSetting}, true, None, map[string]string{}},
	// queue and subscription specified
	{map[string]string{"queueName": queueName, "subscriptionName": subscriptionName, "connection": connectionSetting}, true, None, map[string]string{}},
	// topic but no subscription specified
	{map[string]string{"topicName": topicName, "connection": connectionSetting}, true, None, map[string]string{}},
	// subscription but no topic specified
	{map[string]string{"subscriptionName": subscriptionName, "connection": connectionSetting}, true, None, map[string]string{}},
	// connection not set
	{map[string]string{"queueName": queueName}, true, Queue, map[string]string{}},
	// connection set in auth params
	{map[string]string{"queueName": queueName}, false, Queue, map[string]string{"connection": connectionSetting}},
}

var getServiceBusLengthTestScalers = []azureServiceBusScaler{
	{metadata: &azureServiceBusMetadata{
		entityType: Queue,
		queueName:  queueName,
	}},
	{metadata: &azureServiceBusMetadata{
		entityType:       Subscription,
		topicName:        topicName,
		subscriptionName: subscriptionName,
	}},
}

func TestParseServiceBusMetadata(t *testing.T) {
	for _, testData := range parseServiceBusMetadataDataset {

		meta, err := parseAzureServiceBusMetadata(sampleResolvedEnv, testData.metadata, testData.authParams)

		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
		if meta != nil && meta.entityType != testData.entityType {
			t.Errorf("Expected entity type %v but got %v\n", testData.entityType, meta.entityType)
		}
	}
}

func TestGetServiceBusLength(t *testing.T) {

	t.Log("This test will use the environment variable SERVICEBUS_CONNECTION_STRING if it is set")
	t.Log("If set, it will connect to the servicebus namespace specified by the connection string & check:")
	t.Logf("\tQueue '%s' has 1 message\n", queueName)
	t.Logf("\tTopic '%s' with subscription '%s' has 1 message\n", topicName, subscriptionName)

	connection_string := os.Getenv("SERVICEBUS_CONNECTION_STRING")

	for _, scaler := range getServiceBusLengthTestScalers {
		if connection_string != "" {
			// Can actually test that numbers return
			scaler.metadata.connection = connection_string
			length, err := scaler.GetAzureServiceBusLength(context.TODO())

			if err != nil {
				t.Errorf("Expected success but got error: %s", err)
			}

			if length != 1 {
				t.Errorf("Expected 1 message, got %d", length)
			}

		} else {
			// Just test error message
			length, err := scaler.GetAzureServiceBusLength(context.TODO())

			if length != -1 || err == nil {
				t.Errorf("Expected error but got success")
			}
		}
	}
}
