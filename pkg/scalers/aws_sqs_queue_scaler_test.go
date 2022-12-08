package scalers

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
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

var testAWSSQSEmptyResolvedEnv = map[string]string{}

var testAWSSQSResolvedEnv = map[string]string{
	"QUEUE_URL": testAWSSQSProperQueueURL,
}

var testAWSSQSAuthentication = map[string]string{
	"awsAccessKeyId":     testAWSSQSAccessKeyID,
	"awsSecretAccessKey": testAWSSQSSecretAccessKey,
}

type parseAWSSQSMetadataTestData struct {
	metadata    map[string]string
	authParams  map[string]string
	resolvedEnv map[string]string
	isError     bool
	comment     string
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
		testAWSSQSEmptyResolvedEnv,
		true,
		"metadata empty"},
	{map[string]string{
		"queueURL":    testAWSSQSProperQueueURL,
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		false,
		"properly formed queue and region"},
	{map[string]string{
		"queueURL":    testAWSSQSProperQueueURL,
		"queueLength": "1",
		"awsRegion":   "eu-west-1",
		"awsEndpoint": "http://localhost:4566"},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		false,
		"properly formed queue and region with custom endpoint"},
	{map[string]string{
		"queueURL":    testAWSSQSImproperQueueURL1,
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		true,
		"improperly formed queue, missing queueName"},
	{map[string]string{
		"queueURL":    testAWSSQSImproperQueueURL2,
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		true,
		"improperly formed queue, missing path"},
	{map[string]string{
		"queueURL":    testAWSSQSProperQueueURL,
		"queueLength": "1",
		"awsRegion":   ""},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		true,
		"properly formed queue, empty region"},
	{map[string]string{
		"queueURL":    testAWSSQSProperQueueURL,
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		false,
		"properly formed queue, integer queueLength"},
	{map[string]string{
		"queueURL":    testAWSSQSProperQueueURL,
		"queueLength": "a",
		"awsRegion":   "eu-west-1"},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		false,
		"properly formed queue, invalid queueLength"},
	{map[string]string{
		"queueURL":              testAWSSQSProperQueueURL,
		"queueLength":           "1",
		"activationQueueLength": "1",
		"awsRegion":             "eu-west-1"},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		false,
		"properly formed queue, integer activationQueueLength"},
	{map[string]string{
		"queueURL":              testAWSSQSProperQueueURL,
		"queueLength":           "1",
		"activationQueueLength": "a",
		"awsRegion":             "eu-west-1"},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		false,
		"properly formed queue, invalid activationQueueLength"},
	{map[string]string{
		"queueURL":    testAWSSQSProperQueueURL,
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		map[string]string{
			"awsAccessKeyId":     testAWSSQSAccessKeyID,
			"awsSecretAccessKey": testAWSSQSSecretAccessKey,
		},
		testAWSSQSEmptyResolvedEnv,
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
		testAWSSQSEmptyResolvedEnv,
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
		testAWSSQSEmptyResolvedEnv,
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
		testAWSSQSEmptyResolvedEnv,
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
		testAWSSQSEmptyResolvedEnv,
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
		testAWSSQSEmptyResolvedEnv,
		true,
		"with AWS static credentials from TriggerAuthentication, missing Secret Access Key"},
	{map[string]string{
		"queueURL":    testAWSSQSProperQueueURL,
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		map[string]string{
			"awsRoleArn": testAWSSQSRoleArn,
		},
		testAWSSQSEmptyResolvedEnv,
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
		testAWSSQSEmptyResolvedEnv,
		false,
		"with AWS Role assigned on KEDA operator itself"},
	{map[string]string{
		"queueURL":    testAWSSimpleQueueURL,
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		false,
		"properly formed queue and region"},
	{map[string]string{
		"queueURL":        testAWSSimpleQueueURL,
		"queueLength":     "1",
		"awsRegion":       "eu-west-1",
		"scaleOnInFlight": "false"},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		false,
		"properly formed queue and region"},
	{map[string]string{
		"queueURL":        testAWSSimpleQueueURL,
		"queueLength":     "1",
		"awsRegion":       "eu-west-1",
		"scaleOnInFlight": "true"},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		false,
		"properly formed queue and region"},
	{map[string]string{
		"queueURLFromEnv": "QUEUE_URL",
		"queueLength":     "1",
		"awsRegion":       "eu-west-1"},
		testAWSSQSAuthentication,
		testAWSSQSResolvedEnv,
		false,
		"properly formed queue loaded from env"},
	{map[string]string{
		"queueLength": "1",
		"awsRegion":   "eu-west-1"},
		testAWSSQSAuthentication,
		testAWSSQSEmptyResolvedEnv,
		true,
		"missing queue url from both queueURL and queueURLFromEnv"},
	{map[string]string{
		"queueURLFromEnv": "QUEUE_URL",
		"queueLength":     "1",
		"awsRegion":       "eu-west-1"},
		testAWSSQSAuthentication,
		map[string]string{
			"QUEUE_URL": "",
		},
		true,
		"empty QUEUE_URL env value"},
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
		_, err := parseAwsSqsQueueMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, ResolvedEnv: testData.resolvedEnv, AuthParams: testData.authParams}, logr.Discard())
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
		meta, err := parseAwsSqsQueueMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ResolvedEnv: testData.metadataTestData.resolvedEnv, AuthParams: testData.metadataTestData.authParams, ScalerIndex: testData.scalerIndex}, logr.Discard())
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockAWSSQSScaler := awsSqsQueueScaler{"", meta, &mockSqs{}, logr.Discard()}

		metricSpec := mockAWSSQSScaler.GetMetricSpecForScaling(ctx)
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func TestAWSSQSScalerGetMetrics(t *testing.T) {
	for _, meta := range awsSQSGetMetricTestData {
		scaler := awsSqsQueueScaler{"", meta, &mockSqs{}, logr.Discard()}
		value, _, err := scaler.GetMetricsAndActivity(context.Background(), "MetricName")
		switch meta.queueURL {
		case testAWSSQSErrorQueueURL:
			assert.Error(t, err, "expect error because of sqs api error")
		case testAWSSQSBadDataQueueURL:
			assert.Error(t, err, "expect error because of bad data return from sqs")
		default:
			if meta.scaleOnInFlight {
				assert.EqualValues(t, int64(300.0), value[0].Value.Value())
			} else {
				assert.EqualValues(t, int64(200.0), value[0].Value.Value())
			}
		}
	}
}
