package scalers

import (
	"testing"
)

var testAWSSQSResolvedEnv = map[string]string{
	"AWS_ACCESS_KEY":        "none",
	"AWS_SECRET_ACCESS_KEY": "none",
}

type parseAWSSQSMetadataTestData struct {
	metadata map[string]string
	isError  bool
	reason   string
}

var testAWSSQSMetadata = []parseAWSSQSMetadataTestData{
	{map[string]string{}, true, "metadata empty"},
	{map[string]string{"queueURL": "myqueue", "awsRegion": "eu-west-1", "awsAccessKeyID": "AWS_ACCESS_KEY", "awsSecretAccessKey": "AWS_SECRET_ACCESS_KEY"}, false, "properly formed queue and region"},
	{map[string]string{"queueURL": "myqueue", "awsRegion": "", "awsAccessKeyID": "AWS_ACCESS_KEY", "awsSecretAccessKey": "AWS_SECRET_ACCESS_KEY"}, true, "properly formed queue, empty region"},
	{map[string]string{"queueURL": "myqueue", "awsRegion": "", "awsSecretAccessKey": "AWS_SECRET_ACCESS_KEY"}, true, "missing access key"},
}

func TestSQSParseMetadata(t *testing.T) {
	for _, testData := range testAWSSQSMetadata {
		_, err := parseAwsSqsQueueMetadata(testData.metadata, testAWSSQSResolvedEnv)
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Errorf("Expected error because %s but got success, %#v", testData.reason, testData)
		}
	}
}
