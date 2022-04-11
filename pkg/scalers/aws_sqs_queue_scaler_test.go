package scalers

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	testAWSSQSRoleArn         = "none"
	testAWSSQSAccessKeyID     = "none"
	testAWSSQSSecretAccessKey = "none"
	testAWSSQSSessionToken    = "none"

	testAWSSQSProperQueueURL    = "https://sqs.eu-west-1.amazonaws.com/account_id/DeleteArtifactQ"
	testAWSSQSImproperQueueURL1 = "https://sqs.eu-west-1.amazonaws.com/account_id"
	testAWSSQSImproperQueueURL2 = "https://sqs.eu-west-1.amazonaws.com"
	testAWSSimpleQueueURL       = "my-queue"

	testAWSSQSErrorQueueURL   = "https://sqs.eu-west-1.amazonaws.com/account_id/Error"
	testAWSSQSBadDataQueueURL = "https://sqs.eu-west-1.amazonaws.com/account_id/BadData"
)

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

type awsSQSMetricIdentifier struct {
	metadataTestData *parseAWSSQSMetadataTestData
	scalerIndex      int
	name             string
}

type mockSqs struct {
	sqsiface.SQSAPI
}

func (m *mockSqs) GetQueueAttributes(input *sqs.GetQueueAttributesInput) (*sqs.GetQueueAttributesOutput, error) {
	switch *input.QueueUrl {
	case testAWSSQSErrorQueueURL:
		return nil, errors.New("some error")
	case testAWSSQSBadDataQueueURL:
		return &sqs.GetQueueAttributesOutput{
			Attributes: map[string]*string{
				"ApproximateNumberOfMessages":           aws.String("NotInt"),
				"ApproximateNumberOfMessagesNotVisible": aws.String("NotInt"),
			},
		}, nil
	}

	return &sqs.GetQueueAttributesOutput{
		Attributes: map[string]*string{
			"ApproximateNumberOfMessages":           aws.String("200"),
			"ApproximateNumberOfMessagesNotVisible": aws.String("100"),
		},
	}, nil
}

var testAWSSQSMetadata = []parseAWSSQSMetadataTestData{
	{map[string]string{},
		testAWSSQSAuthentication,
		true,
		"metadata empty"},
	{map[string]string{
		"queueURL":    testAWSSQSProperQueueURL,
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		testAWSSQSAuthentication,
		false,
		"properly formed queue and region"},
	{map[string]string{
		"queueURL":    testAWSSQSImproperQueueURL1,
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		testAWSSQSAuthentication,
		true,
		"improperly formed queue, missing queueName"},
	{map[string]string{
		"queueURL":    testAWSSQSImproperQueueURL2,
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		testAWSSQSAuthentication,
		true,
		"improperly formed queue, missing path"},
	{map[string]string{
		"queueURL":    testAWSSQSProperQueueURL,
		"queueLength": "1",
		"awsRegion":   ""},
		testAWSSQSAuthentication,
		true,
		"properly formed queue, empty region"},
	{map[string]string{
		"queueURL":    testAWSSQSProperQueueURL,
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		testAWSSQSAuthentication,
		false,
		"properly formed queue, integer queueLength"},
	{map[string]string{
		"queueURL":    testAWSSQSProperQueueURL,
		"queueLength": "a",
		"awsRegion":   "eu-west-1"},
		testAWSSQSAuthentication,
		false,
		"properly formed queue, invalid queueLength"},
	{map[string]string{
		"queueURL":    testAWSSQSProperQueueURL,
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		map[string]string{
			"awsAccessKeyId":     testAWSSQSAccessKeyID,
			"awsSecretAccessKey": testAWSSQSSecretAccessKey,
		},
		false,
		"with AWS static credentials from TriggerAuthentication"},
	{map[string]string{
		"queueURL":    testAWSSQSProperQueueURL,
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		map[string]string{
			"awsAccessKeyId":     testAWSSQSAccessKeyID,
			"awsSecretAccessKey": testAWSSQSSecretAccessKey,
			"awsSessionToken":    testAWSSQSSessionToken,
		},
		false,
		"with AWS temporary credentials from TriggerAuthentication"},
	{map[string]string{
		"queueURL":    testAWSSQSProperQueueURL,
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		map[string]string{
			"awsAccessKeyId":     "",
			"awsSecretAccessKey": testAWSSQSSecretAccessKey,
		},
		true,
		"with AWS static credentials from TriggerAuthentication, missing Access Key Id"},
	{map[string]string{
		"queueURL":    testAWSSQSProperQueueURL,
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		map[string]string{
			"awsAccessKeyId":     testAWSSQSAccessKeyID,
			"awsSecretAccessKey": "",
		},
		true,
		"with AWS temporary credentials from TriggerAuthentication, missing Secret Access Key"},
	{map[string]string{
		"queueURL":    testAWSSQSProperQueueURL,
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		map[string]string{
			"awsAccessKeyId":     "",
			"awsSecretAccessKey": testAWSSQSSecretAccessKey,
			"awsSessionToken":    testAWSSQSSessionToken,
		},
		true,
		"with AWS temporary credentials from TriggerAuthentication, missing Access Key Id"},
	{map[string]string{
		"queueURL":    testAWSSQSProperQueueURL,
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		map[string]string{
			"awsAccessKeyId":     testAWSSQSAccessKeyID,
			"awsSecretAccessKey": "",
			"awsSessionToken":    testAWSSQSSessionToken,
		},
		true,
		"with AWS static credentials from TriggerAuthentication, missing Secret Access Key"},
	{map[string]string{
		"queueURL":    testAWSSQSProperQueueURL,
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		map[string]string{
			"awsRoleArn": testAWSSQSRoleArn,
		},
		false,
		"with AWS Role from TriggerAuthentication"},
	{map[string]string{
		"queueURL":      testAWSSQSProperQueueURL,
		"queueLength":   "1",
		"awsRegion":     "eu-west-1",
		"identityOwner": "operator"},
		map[string]string{
			"awsAccessKeyId":     "",
			"awsSecretAccessKey": "",
		},
		false,
		"with AWS Role assigned on KEDA operator itself"},
	{map[string]string{
		"queueURL":    testAWSSimpleQueueURL,
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		testAWSSQSAuthentication,
		false,
		"properly formed queue and region"},
}

var awsSQSMetricIdentifiers = []awsSQSMetricIdentifier{
	{&testAWSSQSMetadata[1], 0, "s0-aws-sqs-DeleteArtifactQ"},
	{&testAWSSQSMetadata[1], 1, "s1-aws-sqs-DeleteArtifactQ"},
}

var awsSQSGetMetricTestData = []*awsSqsQueueMetadata{
	{queueURL: testAWSSQSProperQueueURL},
	{queueURL: testAWSSQSErrorQueueURL},
	{queueURL: testAWSSQSBadDataQueueURL},
}

func TestSQSParseMetadata(t *testing.T) {
	for _, testData := range testAWSSQSMetadata {
		_, err := parseAwsSqsQueueMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, ResolvedEnv: testAWSSQSAuthentication, AuthParams: testData.authParams})
		if err != nil && !testData.isError {
			t.Errorf("Expected success because %s got error, %s", testData.comment, err)
		}
		if testData.isError && err == nil {
			t.Errorf("Expected error because %s but got success, %#v", testData.comment, testData)
		}
	}
}

func TestAWSSQSGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range awsSQSMetricIdentifiers {
		ctx := context.Background()
		meta, err := parseAwsSqsQueueMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ResolvedEnv: testAWSSQSAuthentication, AuthParams: testData.metadataTestData.authParams, ScalerIndex: testData.scalerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockAWSSQSScaler := awsSqsQueueScaler{"", meta, &mockSqs{}}

		metricSpec := mockAWSSQSScaler.GetMetricSpecForScaling(ctx)
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func TestAWSSQSScalerGetMetrics(t *testing.T) {
	var selector labels.Selector
	for _, meta := range awsSQSGetMetricTestData {
		scaler := awsSqsQueueScaler{"", meta, &mockSqs{}}
		value, err := scaler.GetMetrics(context.Background(), "MetricName", selector)
		switch meta.queueURL {
		case testAWSSQSErrorQueueURL:
			assert.Error(t, err, "expect error because of sqs api error")
		case testAWSSQSBadDataQueueURL:
			assert.Error(t, err, "expect error because of bad data return from sqs")
		default:
			assert.EqualValues(t, int64(300.0), value[0].Value.Value())
		}
	}
}
