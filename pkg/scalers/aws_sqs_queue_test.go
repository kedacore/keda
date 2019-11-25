package scalers

import (
	"testing"
)

var testAWSSQSRoleArn = "none"

var testAWSSQSAccessKeyID = "none"
var testAWSSQSSecretAccessKey = "none"

var testAWSSQSResolvedEnv = map[string]string{
	"AWS_ACCESS_KEY":        "none",
	"AWS_SECRET_ACCESS_KEY": "none",
}

var testAWSSQSAuthentication = map[string]string{
	"awsAccessKeyId":     testAWSSQSAccessKeyID,
	"awsSecretAccessKey": testAWSSQSSecretAccessKey,
}

type parseAWSSQSMetadataTestData struct {
	metadata   map[string]string
	authParams map[string]string
	isError    bool
	comment    string
}

var testAWSSQSMetadata = []parseAWSSQSMetadataTestData{
	{map[string]string{},
		testAWSSQSAuthentication,
		true,
		"metadata empty"},
	{map[string]string{
		"queueURL":    "myqueue",
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		testAWSSQSAuthentication,
		false,
		"properly formed queue and region"},
	{map[string]string{
		"queueURL":    "myqueue",
		"queueLength": "1",
		"awsRegion":   ""},
		testAWSSQSAuthentication,
		true,
		"properly formed queue, empty region"},
	{map[string]string{
		"queueURL":    "myqueue",
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		testAWSSQSAuthentication,
		false,
		"properly formed queue, integer queueLength"},
	{map[string]string{
		"queueURL":    "myqueue",
		"queueLength": "a",
		"awsRegion":   "eu-west-1"},
		testAWSSQSAuthentication,
		false,
		"properly formed queue, invalid queueLength"},
	{map[string]string{
		"queueURL":    "myqueue",
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		map[string]string{
			"awsAccessKeyId":     testAWSSQSAccessKeyID,
			"awsSecretAccessKey": testAWSSQSSecretAccessKey,
		},
		false,
		"with AWS Credentials from TriggerAuthentication"},
	{map[string]string{
		"queueURL":    "myqueue",
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		map[string]string{
			"awsAccessKeyId":     "",
			"awsSecretAccessKey": testAWSSQSSecretAccessKey,
		},
		true,
		"with AWS Credentials from TriggerAuthentication, missing Access Key Id"},
	{map[string]string{
		"queueURL":    "myqueue",
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		map[string]string{
			"awsAccessKeyId":     testAWSSQSAccessKeyID,
			"awsSecretAccessKey": "",
		},
		true,
		"with AWS Credentials from TriggerAuthentication, missing Secret Access Key"},
	{map[string]string{
		"queueURL":    "myqueue",
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		map[string]string{
			"awsRoleArn": testAWSSQSRoleArn,
		},
		false,
		"with AWS Role from TriggerAuthentication"},
}

func TestSQSParseMetadata(t *testing.T) {
	for _, testData := range testAWSSQSMetadata {
		_, err := parseAwsSqsQueueMetadata(testData.metadata, testAWSSQSAuthentication, testData.authParams)
		if err != nil && !testData.isError {
			t.Errorf("Expected success because %s got error, %s", testData.comment, err)
		}
		if testData.isError && err == nil {
			t.Errorf("Expected error because %s but got success, %#v", testData.comment, testData)
		}
	}
}
