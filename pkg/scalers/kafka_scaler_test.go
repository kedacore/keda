package scalers

import (
	"reflect"
	"testing"
)

type parseKafkaMetadataTestData struct {
	metadata   map[string]string
	isError    bool
	numBrokers int
	brokers    []string
	group      string
	topic      string
}

// A complete valid metadata example for reference
var validMetadata = map[string]string{
	"brokers":   "broker1:9092,broker2:9092",
	"groupName": "my-group",
	"topicName": "my-topic",
}

var parseKafkaMetadataTestDataset = []parseKafkaMetadataTestData{
	{map[string]string{}, true, 0, nil, "", ""},
	{map[string]string{"brokers": "foobar:9092"}, true, 1, []string{"foobar:9092"}, "", ""},
	{map[string]string{"brokers": "foo:9092,bar:9092"}, true, 2, []string{"foo:9092", "bar:9092"}, "", ""},
	{map[string]string{"brokers": "a", "groupName": "my-group"}, true, 1, []string{"a"}, "my-group", ""},
	{validMetadata, false, 2, []string{"broker1:9092", "broker2:9092"}, "my-group", "my-topic"},
}

func TestGetBrokers(t *testing.T) {
	for _, testData := range parseKafkaMetadataTestDataset {
		scaler := &KafkaScaler{
			Metadata: testData.metadata,
		}
		meta, err := scaler.parseKafkaMetadata()
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
		if len(meta.brokers) != testData.numBrokers {
			t.Errorf("Expected %d brokers but got %d\n", testData.numBrokers, len(meta.brokers))
		}
		if !reflect.DeepEqual(testData.brokers, meta.brokers) {
			t.Errorf("Expected %v but got %v\n", testData.brokers, meta.brokers)
		}
		if meta.group != testData.group {
			t.Errorf("Expected group %s but got %s\n", testData.group, meta.group)
		}
		if meta.topic != testData.topic {
			t.Errorf("Expected topic %s but got %s\n", testData.topic, meta.topic)
		}
	}
}
