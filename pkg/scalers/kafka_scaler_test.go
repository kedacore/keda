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
	"brokerList":    "broker1:9092,broker2:9092",
	"consumerGroup": "my-group",
	"topic":         "my-topic",
}

// A complete valid authParams example for sasl, with username and passwd
var validWithAuthParams = map[string]string{
	"authMode": "sasl_plaintext",
	"username": "admin",
	"passwd":   "admin",
}

// A complete valid authParams example for sasl, without username and passwd
var validWithoutAuthParams = map[string]string{}

var parseKafkaMetadataTestDataset = []parseKafkaMetadataTestData{
	{map[string]string{}, true, 0, nil, "", ""},
	{map[string]string{"brokerList": "foobar:9092"}, true, 1, []string{"foobar:9092"}, "", ""},
	{map[string]string{"brokerList": "foo:9092,bar:9092"}, true, 2, []string{"foo:9092", "bar:9092"}, "", ""},
	{map[string]string{"brokerList": "a", "consumerGroup": "my-group"}, true, 1, []string{"a"}, "my-group", ""},
}

func TestGetBrokers(t *testing.T) {
	for _, testData := range parseKafkaMetadataTestDataset {
		meta, err := parseKafkaMetadata(nil, testData.metadata, validWithAuthParams)

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

		meta, err = parseKafkaMetadata(nil, testData.metadata, validWithoutAuthParams)

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
