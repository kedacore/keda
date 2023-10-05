package scalers

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
)

const (
	testAWSKinesisRoleArn         = "none"
	testAWSKinesisAccessKeyID     = "none"
	testAWSKinesisSecretAccessKey = "none"
	testAWSKinesisSessionToken    = "none"
	testAWSKinesisStreamName      = "test"
	testAWSRegion                 = "eu-west-1"
	testAWSEndpoint               = "http://localhost:4566"
	testAWSKinesisErrorStream     = "Error"
)

var testAWSKinesisAuthentication = map[string]string{
	"awsAccessKeyID":     testAWSKinesisAccessKeyID,
	"awsSecretAccessKey": testAWSKinesisSecretAccessKey,
}

type parseAWSKinesisMetadataTestData struct {
	metadata    map[string]string
	expected    *awsKinesisStreamMetadata
	authParams  map[string]string
	isError     bool
	comment     string
	scalerIndex int
}

type awsKinesisMetricIdentifier struct {
	metadataTestData *parseAWSKinesisMetadataTestData
	scalerIndex      int
	name             string
}

type mockKinesis struct {
}

func (m *mockKinesis) DescribeStreamSummary(_ context.Context, input *kinesis.DescribeStreamSummaryInput, _ ...func(*kinesis.Options)) (*kinesis.DescribeStreamSummaryOutput, error) {
	if *input.StreamName == "Error" {
		return nil, errors.New("some error")
	}

	return &kinesis.DescribeStreamSummaryOutput{
		StreamDescriptionSummary: &types.StreamDescriptionSummary{
			OpenShardCount: aws.Int32(100),
		},
	}, nil
}

var testAWSKinesisMetadata = []parseAWSKinesisMetadataTestData{
	{
		metadata:   map[string]string{},
		authParams: testAWSKinesisAuthentication,
		expected:   &awsKinesisStreamMetadata{},
		isError:    true,
		comment:    "metadata empty"},
	{
		metadata: map[string]string{
			"streamName":           testAWSKinesisStreamName,
			"shardCount":           "2",
			"activationShardCount": "1",
			"awsRegion":            testAWSRegion},
		authParams: testAWSKinesisAuthentication,
		expected: &awsKinesisStreamMetadata{
			targetShardCount:           2,
			activationTargetShardCount: 1,
			streamName:                 testAWSKinesisStreamName,
			awsRegion:                  testAWSRegion,
			awsAuthorization: awsAuthorizationMetadata{
				awsAccessKeyID:     testAWSKinesisAccessKeyID,
				awsSecretAccessKey: testAWSKinesisSecretAccessKey,
				podIdentityOwner:   true,
			},
			scalerIndex: 0,
		},
		isError:     false,
		comment:     "properly formed stream name and region",
		scalerIndex: 0,
	},
	{
		metadata: map[string]string{
			"streamName":           testAWSKinesisStreamName,
			"shardCount":           "2",
			"activationShardCount": "1",
			"awsRegion":            testAWSRegion,
			"awsEndpoint":          testAWSEndpoint},
		authParams: testAWSKinesisAuthentication,
		expected: &awsKinesisStreamMetadata{
			targetShardCount:           2,
			activationTargetShardCount: 1,
			streamName:                 testAWSKinesisStreamName,
			awsRegion:                  testAWSRegion,
			awsEndpoint:                testAWSEndpoint,
			awsAuthorization: awsAuthorizationMetadata{
				awsAccessKeyID:     testAWSKinesisAccessKeyID,
				awsSecretAccessKey: testAWSKinesisSecretAccessKey,
				podIdentityOwner:   true,
			},
			scalerIndex: 0,
		},
		isError:     false,
		comment:     "properly formed stream name and region with custom endpoint",
		scalerIndex: 0,
	},
	{
		metadata: map[string]string{
			"streamName": "",
			"shardCount": "2",
			"awsRegion":  testAWSRegion},
		authParams:  testAWSKinesisAuthentication,
		expected:    &awsKinesisStreamMetadata{},
		isError:     true,
		comment:     "missing stream name",
		scalerIndex: 1,
	},
	{
		metadata: map[string]string{
			"streamName": testAWSKinesisStreamName,
			"shardCount": "2",
			"awsRegion":  ""},
		authParams:  testAWSKinesisAuthentication,
		expected:    &awsKinesisStreamMetadata{},
		isError:     true,
		comment:     "properly formed stream name, empty region",
		scalerIndex: 2,
	},
	{
		metadata: map[string]string{
			"streamName":           testAWSKinesisStreamName,
			"shardCount":           "",
			"activationShardCount": "",
			"awsRegion":            testAWSRegion},
		authParams: testAWSKinesisAuthentication,
		expected: &awsKinesisStreamMetadata{
			targetShardCount:           targetShardCountDefault,
			activationTargetShardCount: activationTargetShardCountDefault,
			streamName:                 testAWSKinesisStreamName,
			awsRegion:                  testAWSRegion,
			awsAuthorization: awsAuthorizationMetadata{
				awsAccessKeyID:     testAWSKinesisAccessKeyID,
				awsSecretAccessKey: testAWSKinesisSecretAccessKey,
				podIdentityOwner:   true,
			},
			scalerIndex: 3,
		},
		isError:     false,
		comment:     "properly formed stream name and region, empty shard count",
		scalerIndex: 3,
	},
	{
		metadata: map[string]string{
			"streamName": testAWSKinesisStreamName,
			"shardCount": "a",
			"awsRegion":  testAWSRegion},
		authParams: testAWSKinesisAuthentication,
		expected: &awsKinesisStreamMetadata{
			targetShardCount: 2,
			streamName:       testAWSKinesisStreamName,
			awsRegion:        testAWSRegion,
			awsAuthorization: awsAuthorizationMetadata{
				awsAccessKeyID:     testAWSKinesisAccessKeyID,
				awsSecretAccessKey: testAWSKinesisSecretAccessKey,
				podIdentityOwner:   true,
			},
			scalerIndex: 4,
		},
		isError:     false,
		comment:     "properly formed stream name and region, wrong shard count",
		scalerIndex: 4,
	},
	{
		metadata: map[string]string{
			"streamName": testAWSKinesisStreamName,
			"shardCount": "2",
			"awsRegion":  testAWSRegion},
		authParams: map[string]string{
			"awsAccessKeyID":     "",
			"awsSecretAccessKey": testAWSKinesisSecretAccessKey,
		},
		expected:    &awsKinesisStreamMetadata{},
		isError:     true,
		comment:     "with AWS static credentials from TriggerAuthentication, missing Access Key Id",
		scalerIndex: 5,
	},
	{metadata: map[string]string{
		"streamName": testAWSKinesisStreamName,
		"shardCount": "2",
		"awsRegion":  testAWSRegion},
		authParams: map[string]string{
			"awsAccessKeyID":     testAWSKinesisAccessKeyID,
			"awsSecretAccessKey": "",
		},
		expected:    &awsKinesisStreamMetadata{},
		isError:     true,
		comment:     "with AWS static credentials from TriggerAuthentication, missing Secret Access Key",
		scalerIndex: 6,
	},
	{
		metadata: map[string]string{
			"streamName": testAWSKinesisStreamName,
			"shardCount": "2",
			"awsRegion":  testAWSRegion},
		authParams: map[string]string{
			"awsAccessKeyID":     testAWSKinesisAccessKeyID,
			"awsSecretAccessKey": testAWSKinesisSecretAccessKey,
			"awsSessionToken":    testAWSKinesisSessionToken,
		},
		expected: &awsKinesisStreamMetadata{
			targetShardCount: 2,
			streamName:       testAWSKinesisStreamName,
			awsRegion:        testAWSRegion,
			awsAuthorization: awsAuthorizationMetadata{
				awsAccessKeyID:     testAWSKinesisAccessKeyID,
				awsSecretAccessKey: testAWSKinesisSecretAccessKey,
				awsSessionToken:    testAWSKinesisSessionToken,
				podIdentityOwner:   true,
			},
			scalerIndex: 5,
		},
		isError:     false,
		comment:     "with AWS temporary credentials from TriggerAuthentication",
		scalerIndex: 5,
	},
	{
		metadata: map[string]string{
			"streamName": testAWSKinesisStreamName,
			"shardCount": "2",
			"awsRegion":  testAWSRegion},
		authParams: map[string]string{
			"awsAccessKeyID":     "",
			"awsSecretAccessKey": testAWSKinesisSecretAccessKey,
			"awsSessionToken":    testAWSKinesisSessionToken,
		},
		expected:    &awsKinesisStreamMetadata{},
		isError:     true,
		comment:     "with AWS temporary credentials from TriggerAuthentication, missing Access Key Id",
		scalerIndex: 5,
	},
	{metadata: map[string]string{
		"streamName": testAWSKinesisStreamName,
		"shardCount": "2",
		"awsRegion":  testAWSRegion},
		authParams: map[string]string{
			"awsAccessKeyID":     testAWSKinesisAccessKeyID,
			"awsSecretAccessKey": "",
			"awsSessionToken":    testAWSKinesisSessionToken,
		},
		expected:    &awsKinesisStreamMetadata{},
		isError:     true,
		comment:     "with AWS temporary credentials from TriggerAuthentication, missing Secret Access Key",
		scalerIndex: 6,
	},
	{metadata: map[string]string{
		"streamName": testAWSKinesisStreamName,
		"shardCount": "2",
		"awsRegion":  testAWSRegion},
		authParams: map[string]string{
			"awsRoleArn": testAWSKinesisRoleArn,
		},
		expected: &awsKinesisStreamMetadata{
			targetShardCount: 2,
			streamName:       testAWSKinesisStreamName,
			awsRegion:        testAWSRegion,
			awsAuthorization: awsAuthorizationMetadata{
				awsRoleArn:       testAWSKinesisRoleArn,
				podIdentityOwner: true,
			},
			scalerIndex: 7,
		},
		isError:     false,
		comment:     "with AWS Role from TriggerAuthentication",
		scalerIndex: 7,
	},
	{metadata: map[string]string{
		"streamName":    testAWSKinesisStreamName,
		"shardCount":    "2",
		"awsRegion":     testAWSRegion,
		"identityOwner": "operator"},
		authParams: map[string]string{},
		expected: &awsKinesisStreamMetadata{
			targetShardCount: 2,
			streamName:       testAWSKinesisStreamName,
			awsRegion:        testAWSRegion,
			awsAuthorization: awsAuthorizationMetadata{
				podIdentityOwner: false,
			},
			scalerIndex: 8,
		},
		isError:     false,
		comment:     "with AWS Role assigned on KEDA operator itself",
		scalerIndex: 8,
	},
}

var awsKinesisMetricIdentifiers = []awsKinesisMetricIdentifier{
	{&testAWSKinesisMetadata[1], 0, "s0-aws-kinesis-test"},
	{&testAWSKinesisMetadata[1], 1, "s1-aws-kinesis-test"},
}

var awsKinesisGetMetricTestData = []*awsKinesisStreamMetadata{
	{streamName: "Good"},
	{streamName: testAWSKinesisErrorStream},
}

func TestKinesisParseMetadata(t *testing.T) {
	for _, testData := range testAWSKinesisMetadata {
		result, err := parseAwsKinesisStreamMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, ResolvedEnv: testAWSKinesisAuthentication, AuthParams: testData.authParams, ScalerIndex: testData.scalerIndex}, logr.Discard())
		if err != nil && !testData.isError {
			t.Errorf("Expected success because %s got error, %s", testData.comment, err)
		}
		if testData.isError && err == nil {
			t.Errorf("Expected error because %s but got success, %#v", testData.comment, testData)
		}

		if !testData.isError && !reflect.DeepEqual(testData.expected, result) {
			t.Fatalf("Expected %#v but got %+#v", testData.expected, result)
		}
	}
}

func TestAWSKinesisGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range awsKinesisMetricIdentifiers {
		ctx := context.Background()
		meta, err := parseAwsKinesisStreamMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ResolvedEnv: testAWSKinesisAuthentication, AuthParams: testData.metadataTestData.authParams, ScalerIndex: testData.scalerIndex}, logr.Discard())
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockAWSKinesisStreamScaler := awsKinesisStreamScaler{"", meta, &mockKinesis{}, logr.Discard()}

		metricSpec := mockAWSKinesisStreamScaler.GetMetricSpecForScaling(ctx)
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func TestAWSKinesisStreamScalerGetMetrics(t *testing.T) {
	for _, meta := range awsKinesisGetMetricTestData {
		scaler := awsKinesisStreamScaler{"", meta, &mockKinesis{}, logr.Discard()}
		value, _, err := scaler.GetMetricsAndActivity(context.Background(), "MetricName")
		switch meta.streamName {
		case testAWSKinesisErrorStream:
			assert.Error(t, err, "expect error because of kinesis api error")
		default:
			assert.EqualValues(t, int64(100.0), value[0].Value.Value())
		}
	}
}
