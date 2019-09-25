package scalers

import (
	"testing"
)

const (
	host = "myHostSecret"
)

type parseRabbitMQMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

var sampleRabbitMqResolvedEnv = map[string]string{
	host: "none",
}

var testRabbitMQMetadata = []parseRabbitMQMetadataTestData{
	// nothing passed
	{map[string]string{}, true},
	// properly formed metadata
	{map[string]string{"queueLength": "10", "queueName": "sample", "host": host}, false},
	// malformed queueLength
	{map[string]string{"queueLength": "AA", "queueName": "sample", "host": host}, true},
	// missing host
	{map[string]string{"queueLength": "AA", "queueName": "sample"}, true},
	// missing queueName
	{map[string]string{"queueLength": "10", "host": host}, true},
}

func TestRabbitMQParseMetadata(t *testing.T) {
	for _, testData := range testRabbitMQMetadata {
		_, err := parseRabbitMQMetadata(sampleRabbitMqResolvedEnv, testData.metadata)
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}
