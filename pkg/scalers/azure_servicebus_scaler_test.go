package scalers

import (
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
	{map[string]string{"topicName": topicName, "subscriptionName": subscriptionName}, true, None},
	// subscription but no topic specified
	{map[string]string{"topicName": topicName, "subscriptionName": subscriptionName}, true, None},
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
	// TODO
}
