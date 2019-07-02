package scalers

import (
	"testing"
)

var testAWSCloudwatchResolvedEnv = map[string]string{
	"AWS_ACCESS_KEY":        "none",
	"AWS_SECRET_ACCESS_KEY": "none",
}

type parseAWSCloudwatchMetadataTestData struct {
	metadata map[string]string
	isError  bool
}

var testAWSCloudwatchMetadata = []parseAWSCloudwatchMetadataTestData{
	{map[string]string{}, true},
	// properly formed cloudwatch query and region
	{map[string]string{"namespace": "AWS/SQS", "dimensionName": "QueueName", "dimensionValue": "keda",
		"metricName": "ApproximateNumberOfMessagesVisible", "targetMetricValue": "2", "minMetricValue": "0",
		"region": "eu-west-1", "awsAccessKeyID": "AWS_ACCESS_KEY", "awsSecretAccessKey": "AWS_SECRET_ACCESS_KEY"}, false},
		// Properly formed cloudwatch query with optional parameters
	{map[string]string{"namespace": "AWS/SQS", "dimensionName": "QueueName", "dimensionValue": "keda",
		"metricName": "ApproximateNumberOfMessagesVisible", "targetMetricValue": "2", "minMetricValue": "0",
		"metricCollectionTime": "300", "metricStat": "Average", "metricStatPeriod": "300",
		"region": "eu-west-1", "awsAccessKeyID": "AWS_ACCESS_KEY", "awsSecretAccessKey": "AWS_SECRET_ACCESS_KEY"}, false},
		// properly formed cloudwatch query but Region is empty
	{map[string]string{"namespace": "AWS/SQS", "dimensionName": "QueueName", "dimensionValue": "keda",
		"metricName": "ApproximateNumberOfMessagesVisible", "targetMetricValue": "2", "minMetricValue": "0",
		"region": "", "awsAccessKeyID": "AWS_ACCESS_KEY", "awsSecretAccessKey": "AWS_SECRET_ACCESS_KEY"}, true},
		// Missing namespace
	{map[string]string{"dimensionName": "QueueName", "dimensionValue": "keda",
		"metricName": "ApproximateNumberOfMessagesVisible", "targetMetricValue": "2", "minMetricValue": "0",
		"region": "eu-west-1", "awsAccessKeyID": "AWS_ACCESS_KEY", "awsSecretAccessKey": "AWS_SECRET_ACCESS_KEY"}, true},
		// Missing dimensionName
	{map[string]string{"dimensionName": "QueueName", "dimensionValue": "keda",
		"metricName": "ApproximateNumberOfMessagesVisible", "targetMetricValue": "2", "minMetricValue": "0",
		"region": "eu-west-1", "awsAccessKeyID": "AWS_ACCESS_KEY", "awsSecretAccessKey": "AWS_SECRET_ACCESS_KEY"}, true},
		// Missing dimensionValue
	{map[string]string{"namespace": "AWS/SQS", "dimensionName": "QueueName",
		"metricName": "ApproximateNumberOfMessagesVisible", "targetMetricValue": "2", "minMetricValue": "0",
		"region": "eu-west-1", "awsAccessKeyID": "AWS_ACCESS_KEY", "awsSecretAccessKey": "AWS_SECRET_ACCESS_KEY"}, true},
		// Missing metricName
	{map[string]string{"namespace": "AWS/SQS", "dimensionName": "QueueName", "dimensionValue": "keda",
		"targetMetricValue": "2", "minMetricValue": "0",
		"region": "eu-west-1", "awsAccessKeyID": "AWS_ACCESS_KEY", "awsSecretAccessKey": "AWS_SECRET_ACCESS_KEY"}, true},
}

func TestCloudwatchParseMetadata(t *testing.T) {
	for _, testData := range testAWSCloudwatchMetadata {
		_, err := parseAwsCloudwatchMetadata(testData.metadata, testAWSCloudwatchResolvedEnv)
		if err != nil && !testData.isError {
			t.Error("Expected success but got error", err)
		}
		if testData.isError && err == nil {
			t.Error("Expected error but got success")
		}
	}
}
