package scalers

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/dynamodbstreams"
	"github.com/aws/aws-sdk-go-v2/service/dynamodbstreams/types"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"reflect"
	"testing"
)

const (
	testAWSDynamoDBStreamsRoleArn          = "none"
	testAWSDynamoDBStreamsAccessKeyID      = "none"
	testAWSDynamoDBStreamsSecretAccessKey  = "none"
	testAWSDynamoDBStreamsSessionToken     = "none"
	testAWSDynamoDBStreamsRegion           = "ap-northeast-1"
	testAWSDynamoDBStreamsEndpoint         = "http://localhost:4566"
	testAWSDynamoDBStreamsArnForSmallTable = "smallstreamarn"
	testAWSDynamoDBStreamsArnForBigTable   = "bigstreamarn"
	testAWSDynamoDBStreamsErrorArn         = "errorarn"
	testAWSDynamoDBSmallTable              = "smalltable" // table with 5 shards
	testAWSDynamoDBBigTable                = "bigtable"   // table with 105 shards
	testAWSDynamoDBErrorTable              = "errortable"
	testAWSDynamoDBInvalidTable            = "invalidtable"
)

var testAwsDynamoDBStreamAuthentication = map[string]string{
	"awsAccessKeyId":     testAWSDynamoDBStreamsAccessKeyID,
	"awsSecretAccessKey": testAWSDynamoDBStreamsSecretAccessKey,
}

func generateTestDynamoDBStreamShards(shardNum int64) []types.Shard {
	var shards []types.Shard
	for i := 0; i < int(shardNum); i++ {
		shards = append(shards, types.Shard{})
	}
	return shards
}

type parseAwsDynamoDBStreamsMetadataTestData struct {
	metadata    map[string]string
	expected    *awsDynamoDBStreamsMetadata
	authParams  map[string]string
	isError     bool
	comment     string
	scalerIndex int
}

type awsDynamoDBStreamsMetricIdentifier struct {
	metadataTestData *parseAwsDynamoDBStreamsMetadataTestData
	scalerIndex      int
	name             string
}

type mockAwsDynamoDBStreams struct {
}

func (m *mockAwsDynamoDBStreams) DescribeStream(_ context.Context, input *dynamodbstreams.DescribeStreamInput, _ ...func(*dynamodbstreams.Options)) (*dynamodbstreams.DescribeStreamOutput, error) {
	switch *input.StreamArn {
	case testAWSDynamoDBStreamsErrorArn:
		return nil, errors.New("Error dynamodbstream DescribeStream")
	case testAWSDynamoDBStreamsArnForBigTable:
		if input.ExclusiveStartShardId != nil {
			return &dynamodbstreams.DescribeStreamOutput{
				StreamDescription: &types.StreamDescription{
					Shards: generateTestDynamoDBStreamShards(5),
				}}, nil
		}
		lastShardID := "testid"
		return &dynamodbstreams.DescribeStreamOutput{
			StreamDescription: &types.StreamDescription{
				Shards:               generateTestDynamoDBStreamShards(100),
				LastEvaluatedShardId: &lastShardID,
			}}, nil
	default:
		return &dynamodbstreams.DescribeStreamOutput{
			StreamDescription: &types.StreamDescription{
				Shards: generateTestDynamoDBStreamShards(5),
			}}, nil
	}
}

type mockAwsDynamoDB struct {
}

func (m *mockAwsDynamoDB) DescribeTable(_ context.Context, input *dynamodb.DescribeTableInput, _ ...func(*dynamodb.Options)) (*dynamodb.DescribeTableOutput, error) {
	switch *input.TableName {
	case testAWSDynamoDBInvalidTable:
		return nil, fmt.Errorf("DynamoDB Stream Arn is invalid")
	case testAWSDynamoDBErrorTable:
		return &dynamodb.DescribeTableOutput{
			Table: &dynamodbTypes.TableDescription{
				LatestStreamArn: aws.String(testAWSDynamoDBStreamsErrorArn),
			},
		}, nil
	case testAWSDynamoDBBigTable:
		return &dynamodb.DescribeTableOutput{
			Table: &dynamodbTypes.TableDescription{
				LatestStreamArn: aws.String(testAWSDynamoDBStreamsArnForBigTable),
			},
		}, nil
	default:
		return &dynamodb.DescribeTableOutput{
			Table: &dynamodbTypes.TableDescription{
				LatestStreamArn: aws.String(testAWSDynamoDBStreamsArnForSmallTable),
			},
		}, nil
	}
}

var testAwsDynamoDBStreamMetadata = []parseAwsDynamoDBStreamsMetadataTestData{
	{
		metadata:   map[string]string{},
		authParams: testAWSKinesisAuthentication,
		expected:   &awsDynamoDBStreamsMetadata{},
		isError:    true,
		comment:    "metadata empty"},
	{
		metadata: map[string]string{
			"tableName":            testAWSDynamoDBSmallTable,
			"shardCount":           "2",
			"activationShardCount": "1",
			"awsRegion":            testAWSDynamoDBStreamsRegion},
		authParams: testAWSKinesisAuthentication,
		expected: &awsDynamoDBStreamsMetadata{
			targetShardCount:           2,
			activationTargetShardCount: 1,
			tableName:                  testAWSDynamoDBSmallTable,
			awsRegion:                  testAWSDynamoDBStreamsRegion,
			awsAuthorization: awsAuthorizationMetadata{
				awsAccessKeyID:     testAWSDynamoDBStreamsAccessKeyID,
				awsSecretAccessKey: testAWSDynamoDBStreamsSecretAccessKey,
				podIdentityOwner:   true,
			},
			scalerIndex: 0,
		},
		isError:     false,
		comment:     "properly formed dynamodb table name and region",
		scalerIndex: 0,
	},
	{
		metadata: map[string]string{
			"tableName":            testAWSDynamoDBSmallTable,
			"shardCount":           "2",
			"activationShardCount": "1",
			"awsRegion":            testAWSDynamoDBStreamsRegion,
			"awsEndpoint":          testAWSDynamoDBStreamsEndpoint},
		authParams: testAWSKinesisAuthentication,
		expected: &awsDynamoDBStreamsMetadata{
			targetShardCount:           2,
			activationTargetShardCount: 1,
			tableName:                  testAWSDynamoDBSmallTable,
			awsRegion:                  testAWSDynamoDBStreamsRegion,
			awsEndpoint:                testAWSDynamoDBStreamsEndpoint,
			awsAuthorization: awsAuthorizationMetadata{
				awsAccessKeyID:     testAWSDynamoDBStreamsAccessKeyID,
				awsSecretAccessKey: testAWSDynamoDBStreamsSecretAccessKey,
				podIdentityOwner:   true,
			},
			scalerIndex: 0,
		},
		isError:     false,
		comment:     "properly formed dynamodb table name and region",
		scalerIndex: 0,
	},
	{
		metadata: map[string]string{
			"tableName":  "",
			"shardCount": "2",
			"awsRegion":  testAWSDynamoDBStreamsRegion},
		authParams:  testAWSKinesisAuthentication,
		expected:    &awsDynamoDBStreamsMetadata{},
		isError:     true,
		comment:     "missing dynamodb table name",
		scalerIndex: 1,
	},
	{
		metadata: map[string]string{
			"tableName":  testAWSDynamoDBSmallTable,
			"shardCount": "2",
			"awsRegion":  ""},
		authParams:  testAWSKinesisAuthentication,
		expected:    &awsDynamoDBStreamsMetadata{},
		isError:     true,
		comment:     "properly formed dynamodb table name, empty region",
		scalerIndex: 2,
	},
	{
		metadata: map[string]string{
			"tableName":  testAWSDynamoDBSmallTable,
			"shardCount": "",
			"awsRegion":  testAWSDynamoDBStreamsRegion},
		authParams: testAWSKinesisAuthentication,
		expected: &awsDynamoDBStreamsMetadata{
			targetShardCount:           defaultTargetDBStreamsShardCount,
			activationTargetShardCount: defaultActivationTargetDBStreamsShardCount,
			tableName:                  testAWSDynamoDBSmallTable,
			awsRegion:                  testAWSDynamoDBStreamsRegion,
			awsAuthorization: awsAuthorizationMetadata{
				awsAccessKeyID:     testAWSDynamoDBStreamsAccessKeyID,
				awsSecretAccessKey: testAWSDynamoDBStreamsSecretAccessKey,
				podIdentityOwner:   true,
			},
			scalerIndex: 3,
		},
		isError:     false,
		comment:     "properly formed table name and region, empty shard count",
		scalerIndex: 3,
	},
	{
		metadata: map[string]string{
			"tableName":  testAWSDynamoDBSmallTable,
			"shardCount": "a",
			"awsRegion":  testAWSDynamoDBStreamsRegion},
		authParams: testAWSKinesisAuthentication,
		expected: &awsDynamoDBStreamsMetadata{
			targetShardCount: defaultTargetDBStreamsShardCount,
			tableName:        testAWSDynamoDBSmallTable,
			awsRegion:        testAWSDynamoDBStreamsRegion,
			awsAuthorization: awsAuthorizationMetadata{
				awsAccessKeyID:     testAWSDynamoDBStreamsAccessKeyID,
				awsSecretAccessKey: testAWSDynamoDBStreamsSecretAccessKey,
				podIdentityOwner:   true,
			},
			scalerIndex: 4,
		},
		isError:     false,
		comment:     "properly formed table name and region, wrong shard count",
		scalerIndex: 4,
	},
	{
		metadata: map[string]string{
			"tableName":  testAWSDynamoDBSmallTable,
			"shardCount": "2",
			"awsRegion":  testAWSDynamoDBStreamsRegion},
		authParams: map[string]string{
			"awsAccessKeyID":     "",
			"awsSecretAccessKey": testAWSDynamoDBStreamsSecretAccessKey,
		},
		expected:    &awsDynamoDBStreamsMetadata{},
		isError:     true,
		comment:     "with AWS static credentials from TriggerAuthentication, missing Access Key Id",
		scalerIndex: 5,
	},
	{metadata: map[string]string{
		"tableName":  testAWSDynamoDBSmallTable,
		"shardCount": "2",
		"awsRegion":  testAWSDynamoDBStreamsRegion},
		authParams: map[string]string{
			"awsAccessKeyID":     testAWSDynamoDBStreamsAccessKeyID,
			"awsSecretAccessKey": "",
		},
		expected:    &awsDynamoDBStreamsMetadata{},
		isError:     true,
		comment:     "with AWS static credentials from TriggerAuthentication, missing Secret Access Key",
		scalerIndex: 6,
	},
	{
		metadata: map[string]string{
			"tableName":  testAWSDynamoDBSmallTable,
			"shardCount": "2",
			"awsRegion":  testAWSDynamoDBStreamsRegion},
		authParams: map[string]string{
			"awsAccessKeyID":     testAWSDynamoDBStreamsAccessKeyID,
			"awsSecretAccessKey": testAWSDynamoDBStreamsSecretAccessKey,
			"awsSessionToken":    testAWSDynamoDBStreamsSessionToken,
		},
		expected: &awsDynamoDBStreamsMetadata{
			targetShardCount: 2,
			tableName:        testAWSDynamoDBSmallTable,
			awsRegion:        testAWSDynamoDBStreamsRegion,
			awsAuthorization: awsAuthorizationMetadata{
				awsAccessKeyID:     testAWSDynamoDBStreamsAccessKeyID,
				awsSecretAccessKey: testAWSDynamoDBStreamsSecretAccessKey,
				awsSessionToken:    testAWSDynamoDBStreamsSessionToken,
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
			"tableName":  testAWSDynamoDBSmallTable,
			"shardCount": "2",
			"awsRegion":  testAWSDynamoDBStreamsRegion},
		authParams: map[string]string{
			"awsAccessKeyID":     "",
			"awsSecretAccessKey": testAWSDynamoDBStreamsSecretAccessKey,
			"awsSessionToken":    testAWSDynamoDBStreamsSessionToken,
		},
		expected:    &awsDynamoDBStreamsMetadata{},
		isError:     true,
		comment:     "with AWS temporary credentials from TriggerAuthentication, missing Access Key Id",
		scalerIndex: 5,
	},
	{metadata: map[string]string{
		"tableName":  testAWSDynamoDBSmallTable,
		"shardCount": "2",
		"awsRegion":  testAWSDynamoDBStreamsRegion},
		authParams: map[string]string{
			"awsAccessKeyID":     testAWSDynamoDBStreamsAccessKeyID,
			"awsSecretAccessKey": "",
			"awsSessionToken":    testAWSDynamoDBStreamsSessionToken,
		},
		expected:    &awsDynamoDBStreamsMetadata{},
		isError:     true,
		comment:     "with AWS temporary credentials from TriggerAuthentication, missing Secret Access Key",
		scalerIndex: 6,
	},
	{metadata: map[string]string{
		"tableName":  testAWSDynamoDBSmallTable,
		"shardCount": "2",
		"awsRegion":  testAWSDynamoDBStreamsRegion},
		authParams: map[string]string{
			"awsRoleArn": testAWSDynamoDBStreamsRoleArn,
		},
		expected: &awsDynamoDBStreamsMetadata{
			targetShardCount: 2,
			tableName:        testAWSDynamoDBSmallTable,
			awsRegion:        testAWSDynamoDBStreamsRegion,
			awsAuthorization: awsAuthorizationMetadata{
				awsRoleArn:       testAWSDynamoDBStreamsRoleArn,
				podIdentityOwner: true,
			},
			scalerIndex: 7,
		},
		isError:     false,
		comment:     "with AWS Role from TriggerAuthentication",
		scalerIndex: 7,
	},
	{metadata: map[string]string{
		"tableName":     testAWSDynamoDBSmallTable,
		"shardCount":    "2",
		"awsRegion":     testAWSDynamoDBStreamsRegion,
		"identityOwner": "operator"},
		authParams: map[string]string{},
		expected: &awsDynamoDBStreamsMetadata{
			targetShardCount: 2,
			tableName:        testAWSDynamoDBSmallTable,
			awsRegion:        testAWSDynamoDBStreamsRegion,
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

var awsDynamoDBStreamMetricIdentifiers = []awsDynamoDBStreamsMetricIdentifier{
	{&testAwsDynamoDBStreamMetadata[1], 0, fmt.Sprintf("s0-aws-dynamodb-streams-%s", testAWSDynamoDBSmallTable)},
	{&testAwsDynamoDBStreamMetadata[1], 1, fmt.Sprintf("s1-aws-dynamodb-streams-%s", testAWSDynamoDBSmallTable)},
}

var awsDynamoDBStreamsGetMetricTestData = []*awsDynamoDBStreamsMetadata{
	{tableName: testAWSDynamoDBBigTable},
	{tableName: testAWSDynamoDBSmallTable},
	{tableName: testAWSDynamoDBErrorTable},
	{tableName: testAWSDynamoDBInvalidTable},
}

func TestParseAwsDynamoDBStreamsMetadata(t *testing.T) {
	for _, testData := range testAwsDynamoDBStreamMetadata {
		result, err := parseAwsDynamoDBStreamsMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, ResolvedEnv: testAwsDynamoDBStreamAuthentication, AuthParams: testData.authParams, ScalerIndex: testData.scalerIndex}, logr.Discard())
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

func TestAwsDynamoDBStreamsGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range awsDynamoDBStreamMetricIdentifiers {
		ctx := context.Background()
		meta, err := parseAwsDynamoDBStreamsMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ResolvedEnv: testAwsDynamoDBStreamAuthentication, AuthParams: testData.metadataTestData.authParams, ScalerIndex: testData.scalerIndex}, logr.Discard())
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		streamArn, err := getDynamoDBStreamsArn(ctx, &mockAwsDynamoDB{}, &meta.tableName)
		if err != nil {
			t.Fatal("Could not get dynamodb stream arn:", err)
		}
		mockAwsDynamoDBStreamsScaler := awsDynamoDBStreamsScaler{"", meta, streamArn, &mockAwsDynamoDBStreams{}, logr.Discard()}
		metricSpec := mockAwsDynamoDBStreamsScaler.GetMetricSpecForScaling(ctx)
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func TestAwsDynamoDBStreamsScalerGetMetrics(t *testing.T) {
	for _, meta := range awsDynamoDBStreamsGetMetricTestData {
		var value []external_metrics.ExternalMetricValue
		var err error
		var streamArn *string
		ctx := context.Background()
		streamArn, err = getDynamoDBStreamsArn(ctx, &mockAwsDynamoDB{}, &meta.tableName)
		if err == nil {
			scaler := awsDynamoDBStreamsScaler{"", meta, streamArn, &mockAwsDynamoDBStreams{}, logr.Discard()}
			value, _, err = scaler.GetMetricsAndActivity(context.Background(), "MetricName")
		}
		switch meta.tableName {
		case testAWSDynamoDBErrorTable:
			assert.Error(t, err, "expect error because of dynamodb stream api error")
		case testAWSDynamoDBInvalidTable:
			assert.Error(t, err, "expect error because of dynamodb api error")
		case testAWSDynamoDBBigTable:
			assert.EqualValues(t, int64(105), value[0].Value.Value())
		default:
			assert.EqualValues(t, int64(5), value[0].Value.Value())
		}
	}
}

func TestAwsDynamoDBStreamsScalerIsActive(t *testing.T) {
	for _, meta := range awsDynamoDBStreamsGetMetricTestData {
		var value bool
		var err error
		var streamArn *string
		ctx := context.Background()
		streamArn, err = getDynamoDBStreamsArn(ctx, &mockAwsDynamoDB{}, &meta.tableName)
		if err == nil {
			scaler := awsDynamoDBStreamsScaler{"", meta, streamArn, &mockAwsDynamoDBStreams{}, logr.Discard()}
			_, value, err = scaler.GetMetricsAndActivity(context.Background(), "MetricName")
		}
		switch meta.tableName {
		case testAWSDynamoDBErrorTable:
			assert.Error(t, err, "expect error because of dynamodb stream api error")
		case testAWSDynamoDBInvalidTable:
			assert.Error(t, err, "expect error because of dynamodb api error")
		default:
			assert.EqualValues(t, true, value)
		}
	}
}
