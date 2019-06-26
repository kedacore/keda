package scalers

import (
	"testing"
)

type parseRabbitMQMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

var testRabbitMQMetadata = []parseRabbitMQMetadataTestData{
	// nothing passed
	{map[string]string{}, true},
	// properly formed metadata
	{map[string]string{"queueLength": "10", "queueName": "sample", "host": "redis://redis"}, false},
	// malformed queueLength
	{map[string]string{"queueLength": "AA", "queueName": "sample", "host": "redis://redis"}, true},
	// missing host
	{map[string]string{"queueLength": "AA", "queueName": "sample"}, true},
	// missing queueName
	{map[string]string{"queueLength": "10", "host": "redis://redis"}, true},
}

func TestRabbitMQParseMetadata(t *testing.T) {
	for _, testData := range testRabbitMQMetadata {
		_, err := parseRabbitMQMetadata(testData.metadata)
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}
