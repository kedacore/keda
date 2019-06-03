package scalers

import (
	"testing"
)

var testAWSSQSResolvedEnv = map[string]string{
	"awsAccessKeyID":     "none",
	"awsSecretAccessKey": "none",
}

type parseAWSSQSMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

var testAWSSQSMetadata = []parseAWSSQSMetadataTestData{
	{map[string]string{}, true},
	// properly formed queue and region
	{map[string]string{"queueURL": "myqueue", "region": "eu-west-1"}, false},
	// properly formed queue, empty region
	{map[string]string{"queueURL": "myqueue", "region": ""}, true},
}

func TestSQSParseMetadata(t *testing.T) {
	for _, testData := range testAWSSQSMetadata {
		_, err := parseAwsSqsQueueMetadata(testData.metadata, testAWSSQSResolvedEnv)
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}
