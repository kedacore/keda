package scalers

import (
	"testing"
)

var testAWSCloudwatchRoleArn = "none"

var testAWSCloudwatchAccessKeyID = "none"
var testAWSCloudwatchSecretAccessKey = "none"

var testAWSCloudwatchResolvedEnv = map[string]string{
	"AWS_ACCESS_KEY":        "none",
	"AWS_SECRET_ACCESS_KEY": "none",
}

var testAWSAuthentication = map[string]string{
	"awsAccessKeyId":     testAWSCloudwatchAccessKeyID,
	"awsSecretAccessKey": testAWSCloudwatchSecretAccessKey,
}

type parseAWSCloudwatchMetadataTestData struct {
	metadata   map[string]string
	authParams map[string]string
	isError    bool
	comment    string
}

type awsCloudwatchMetricIdentifier struct {
	metadataTestData *parseAWSCloudwatchMetadataTestData
	name             string
}

var testAWSCloudwatchMetadata = []parseAWSCloudwatchMetadataTestData{
	{map[string]string{}, testAWSAuthentication, true, "Empty structures"},
	// properly formed cloudwatch query and awsRegion
	{map[string]string{
		"namespace":         "AWS/SQS",
		"dimensionName":     "QueueName",
		"dimensionValue":    "keda",
		"metricName":        "ApproximateNumberOfMessagesVisible",
		"targetMetricValue": "2",
		"minMetricValue":    "0",
		"awsRegion":         "eu-west-1"},
		testAWSAuthentication,
		false,
		"properly formed cloudwatch query and awsRegion"},
	// Properly formed cloudwatch query with optional parameters
	{map[string]string{
		"namespace":            "AWS/SQS",
		"dimensionName":        "QueueName",
		"dimensionValue":       "keda",
		"metricName":           "ApproximateNumberOfMessagesVisible",
		"targetMetricValue":    "2",
		"minMetricValue":       "0",
		"metricCollectionTime": "300",
		"metricStat":           "Average",
		"metricStatPeriod":     "300",
		"awsRegion":            "eu-west-1"},
		testAWSAuthentication, false,
		"Properly formed cloudwatch query with optional parameters"},
	// properly formed cloudwatch query but Region is empty
	{map[string]string{
		"namespace":         "AWS/SQS",
		"dimensionName":     "QueueName",
		"dimensionValue":    "keda",
		"metricName":        "ApproximateNumberOfMessagesVisible",
		"targetMetricValue": "2",
		"minMetricValue":    "0",
		"awsRegion":         ""},
		testAWSAuthentication,
		true,
		"properly formed cloudwatch query but Region is empty"},
	// Missing namespace
	{map[string]string{"dimensionName": "QueueName",
		"dimensionValue":    "keda",
		"metricName":        "ApproximateNumberOfMessagesVisible",
		"targetMetricValue": "2",
		"minMetricValue":    "0",
		"awsRegion":         "eu-west-1"},
		testAWSAuthentication,
		true,
		"Missing namespace"},
	// Missing dimensionName
	{map[string]string{
		"dimensionName":     "QueueName",
		"dimensionValue":    "keda",
		"metricName":        "ApproximateNumberOfMessagesVisible",
		"targetMetricValue": "2",
		"minMetricValue":    "0",
		"awsRegion":         "eu-west-1"},
		testAWSAuthentication,
		true,
		"Missing dimensionName"},
	// Missing dimensionValue
	{map[string]string{
		"namespace":         "AWS/SQS",
		"dimensionName":     "QueueName",
		"metricName":        "ApproximateNumberOfMessagesVisible",
		"targetMetricValue": "2",
		"minMetricValue":    "0",
		"awsRegion":         "eu-west-1"},
		testAWSAuthentication,
		true,
		"Missing dimensionValue"},
	// Missing metricName
	{map[string]string{
		"namespace":         "AWS/SQS",
		"dimensionName":     "QueueName",
		"dimensionValue":    "keda",
		"targetMetricValue": "2",
		"minMetricValue":    "0",
		"awsRegion":         "eu-west-1"},
		testAWSAuthentication,
		true,
		"Missing metricName"},
	// with "aws_credentials" from TriggerAuthentication
	{map[string]string{
		"namespace":            "AWS/SQS",
		"dimensionName":        "QueueName",
		"dimensionValue":       "keda",
		"metricName":           "ApproximateNumberOfMessagesVisible",
		"targetMetricValue":    "2",
		"minMetricValue":       "0",
		"metricCollectionTime": "300",
		"metricStat":           "Average",
		"metricStatPeriod":     "300",
		"awsRegion":            "eu-west-1"},
		map[string]string{
			"awsAccessKeyId":     testAWSCloudwatchAccessKeyID,
			"awsSecretAccessKey": testAWSCloudwatchSecretAccessKey,
		},
		false,
		"with AWS Credentials from TriggerAuthentication"},
	// with "aws_role" from TriggerAuthentication
	{map[string]string{
		"namespace":            "AWS/SQS",
		"dimensionName":        "QueueName",
		"dimensionValue":       "keda",
		"metricName":           "ApproximateNumberOfMessagesVisible",
		"targetMetricValue":    "2",
		"minMetricValue":       "0",
		"metricCollectionTime": "300",
		"metricStat":           "Average",
		"metricStatPeriod":     "300",
		"awsRegion":            "eu-west-1"},
		map[string]string{
			"awsRoleArn": testAWSCloudwatchRoleArn,
		},
		false,
		"with AWS Role from TriggerAuthentication"},
	{map[string]string{
		"namespace":            "AWS/SQS",
		"dimensionName":        "QueueName",
		"dimensionValue":       "keda",
		"metricName":           "ApproximateNumberOfMessagesVisible",
		"targetMetricValue":    "2",
		"minMetricValue":       "0",
		"metricCollectionTime": "300",
		"metricStat":           "Average",
		"metricStatPeriod":     "300",
		"awsRegion":            "eu-west-1",
		"identityOwner":        "operator"},
		map[string]string{},
		false,
		"with AWS Role assigned on KEDA operator itself"},
	{map[string]string{
		"namespace":            "AWS/SQS",
		"dimensionName":        "QueueName",
		"dimensionValue":       "keda",
		"metricName":           "ApproximateNumberOfMessagesVisible",
		"targetMetricValue":    "2",
		"minMetricValue":       "0",
		"metricCollectionTime": "a",
		"metricStat":           "Average",
		"metricStatPeriod":     "300",
		"awsRegion":            "eu-west-1",
		"identityOwner":        "operator"},
		map[string]string{},
		true,
		"if metricCollectionTime assigned with a string, need to be a number"},
	{map[string]string{
		"namespace":            "AWS/SQS",
		"dimensionName":        "QueueName",
		"dimensionValue":       "keda",
		"metricName":           "ApproximateNumberOfMessagesVisible",
		"targetMetricValue":    "2",
		"minMetricValue":       "0",
		"metricCollectionTime": "300",
		"metricStat":           "Average",
		"metricStatPeriod":     "a",
		"awsRegion":            "eu-west-1",
		"identityOwner":        "operator"},
		map[string]string{},
		true,
		"if metricStatPeriod assigned with a string, need to be a number"},
	{map[string]string{
		"namespace":         "AWS/SQS",
		"dimensionName":     "QueueName",
		"dimensionValue":    "keda",
		"metricName":        "ApproximateNumberOfMessagesVisible",
		"targetMetricValue": "2",
		"minMetricValue":    "0",
		"metricStat":        "Average",
		"metricStatPeriod":  "300",
		"awsRegion":         "eu-west-1"},
		testAWSAuthentication, false,
		"Missing metricCollectionTime not generate error because will get the default value"},
	{map[string]string{
		"namespace":            "AWS/SQS",
		"dimensionName":        "QueueName",
		"dimensionValue":       "keda",
		"metricName":           "ApproximateNumberOfMessagesVisible",
		"targetMetricValue":    "2",
		"minMetricValue":       "0",
		"metricCollectionTime": "300",
		"metricStatPeriod":     "300",
		"awsRegion":            "eu-west-1"},
		testAWSAuthentication, false,
		"Missing metricStat not generate error because will get the default value"},
	{map[string]string{
		"namespace":            "AWS/SQS",
		"dimensionName":        "QueueName",
		"dimensionValue":       "keda",
		"metricName":           "ApproximateNumberOfMessagesVisible",
		"targetMetricValue":    "2",
		"minMetricValue":       "0",
		"metricCollectionTime": "300",
		"metricStat":           "Average",
		"awsRegion":            "eu-west-1"},
		testAWSAuthentication, false,
		"Missing metricStatPeriod not generate error because will get the default value"},
}

var awsCloudwatchMetricIdentifiers = []awsCloudwatchMetricIdentifier{
	{&testAWSCloudwatchMetadata[1], "aws-cloudwatch-AWS-SQS-QueueName-keda"},
}

func TestCloudwatchParseMetadata(t *testing.T) {
	for _, testData := range testAWSCloudwatchMetadata {
		_, err := parseAwsCloudwatchMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, ResolvedEnv: testAWSCloudwatchResolvedEnv, AuthParams: testData.authParams})
		if err != nil && !testData.isError {
			t.Errorf("%s: Expected success but got error %s", testData.comment, err)
		}
		if testData.isError && err == nil {
			t.Errorf("%s: Expected error but got success", testData.comment)
		}
	}
}

func TestAWSCloudwatchGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range awsCloudwatchMetricIdentifiers {
		meta, err := parseAwsCloudwatchMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ResolvedEnv: testAWSCloudwatchResolvedEnv, AuthParams: testData.metadataTestData.authParams})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockAWSCloudwatchScaler := awsCloudwatchScaler{meta}

		metricSpec := mockAWSCloudwatchScaler.GetMetricSpecForScaling()
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
