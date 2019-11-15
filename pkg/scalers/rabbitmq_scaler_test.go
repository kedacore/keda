package scalers

import (
	"testing"
)

const (
	host = "myHostSecret"
)

type parseRabbitMQMetadataTestData struct {
	metadata   map[string]string
	isError    bool
	authParams map[string]string
}

var sampleRabbitMqResolvedEnv = map[string]string{
	host: "none",
}

var testRabbitMQMetadata = []parseRabbitMQMetadataTestData{
	// nothing passed
	{map[string]string{}, true, map[string]string{}},
	// properly formed metadata
	{map[string]string{"queueLength": "10", "queueName": "sample", "host": host}, false, map[string]string{}},
	// malformed queueLength
	{map[string]string{"queueLength": "AA", "queueName": "sample", "host": host}, true, map[string]string{}},
	// missing host
	{map[string]string{"queueLength": "AA", "queueName": "sample"}, true, map[string]string{}},
	// missing queueName
	{map[string]string{"queueLength": "10", "host": host}, true, map[string]string{}},
	// host defined in authParams
	{map[string]string{"queueLength": "10"}, true, map[string]string{"host": host}},
}

func TestRabbitMQParseMetadata(t *testing.T) {
	for _, testData := range testRabbitMQMetadata {
		_, err := parseRabbitMQMetadata(sampleRabbitMqResolvedEnv, testData.metadata, testData.authParams)
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}
