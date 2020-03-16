package scalers

import (
	"reflect"
	"testing"
)

const (
	testAWSKinesisRoleArn         = "none"
	testAWSKinesisAccessKeyID     = "none"
	testAWSKinesisSecretAccessKey = "none"
	testAWSKinesisStreamName      = "test"
	testAWSRegion                 = "eu-west-1"
)

var testAWSKinesisResolvedEnv = map[string]string{
	"AWS_ACCESS_KEY":        "none",
	"AWS_SECRET_ACCESS_KEY": "none",
}

var testAWSKinesisAuthentication = map[string]string{
	"awsAccessKeyID":     testAWSKinesisAccessKeyID,
	"awsSecretAccessKey": testAWSKinesisSecretAccessKey,
}

type parseAWSKinesisMetadataTestData struct {
	metadata   map[string]string
	expected   *awsKinesisStreamMetadata
	authParams map[string]string
	isError    bool
	comment    string
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
			"streamName": testAWSKinesisStreamName,
			"shardCount": "2",
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
		},
		isError: false,
		comment: "properly formed stream name and region"},
	{
		metadata: map[string]string{
			"streamName": "",
			"shardCount": "2",
			"awsRegion":  testAWSRegion},
		authParams: testAWSKinesisAuthentication,
		expected:   &awsKinesisStreamMetadata{},
		isError:    true,
		comment:    "missing stream name"},
	{
		metadata: map[string]string{
			"streamName": testAWSKinesisStreamName,
			"shardCount": "2",
			"awsRegion":  ""},
		authParams: testAWSKinesisAuthentication,
		expected:   &awsKinesisStreamMetadata{},
		isError:    true,
		comment:    "properly formed stream name, empty region"},
	{
		metadata: map[string]string{
			"streamName": testAWSKinesisStreamName,
			"shardCount": "",
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
		},
		isError: false,
		comment: "properly formed stream name and region, empty shard count"},
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
		},
		isError: false,
		comment: "properly formed stream name and region, wrong shard count"},

	{
		metadata: map[string]string{
			"streamName": testAWSKinesisStreamName,
			"shardCount": "2",
			"awsRegion":  testAWSRegion},
		authParams: map[string]string{
			"awsAccessKeyID":     "",
			"awsSecretAccessKey": testAWSKinesisSecretAccessKey,
		},
		expected: &awsKinesisStreamMetadata{},
		isError:  true,
		comment:  "with AWS Credentials from TriggerAuthentication, missing Access Key Id"},
	{metadata: map[string]string{
		"streamName": testAWSKinesisStreamName,
		"shardCount": "2",
		"awsRegion":  testAWSRegion},
		authParams: map[string]string{
			"awsAccessKeyID":     testAWSKinesisAccessKeyID,
			"awsSecretAccessKey": "",
		},
		expected: &awsKinesisStreamMetadata{},
		isError:  true,
		comment:  "with AWS Credentials from TriggerAuthentication, missing Secret Access Key"},
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
		},
		isError: false,
		comment: "with AWS Role from TriggerAuthentication"},
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
		},
		isError: false,
		comment: "with AWS Role assigned on KEDA operator itself"},
}

func TestKinesisParseMetadata(t *testing.T) {
	for _, testData := range testAWSKinesisMetadata {
		result, err := parseAwsKinesisStreamMetadata(testData.metadata, testAWSKinesisAuthentication, testData.authParams)
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
