package scalers

import (
	"context"
	"os"
	"testing"
)

const (
	topicName        = "testtopic"
	subscriptionName = "testsubscription"
	queueName        = "testqueue"
)

type parseServiceBusMetadataTestData struct {
	metadata   map[string]string
	isError    bool
	entityType EntityType
}

// not testing connections so it doesn't matter what the resolved env value is for this
var sampleResolvedEnv = map[string]string{
	defaultConnectionSetting: "none",
}

var parseServiceBusMetadataDataset = []parseServiceBusMetadataTestData{
	{map[string]string{}, true, None},
	// properly formed queue
	{map[string]string{"queueName": queueName}, false, Queue},
	// properly formed topic & subscription
	{map[string]string{"topicName": topicName, "subscriptionName": subscriptionName}, false, Subscription},
	// queue and topic specified
	{map[string]string{"queueName": queueName, "topicName": topicName}, true, None},
	// queue and subscription specified
	{map[string]string{"queueName": queueName, "subscriptionName": subscriptionName}, true, None},
	// topic but no subscription specifed
	{map[string]string{"topicName": topicName}, true, None},
	// subscription but no topic specified
	{map[string]string{"subscriptionName": subscriptionName}, true, None},
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

		meta, err := parseAzureServiceBusMetadata(sampleResolvedEnv, testData.metadata)

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
