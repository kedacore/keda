package scalers

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/dynamodbstreams"
	"github.com/aws/aws-sdk-go/service/dynamodbstreams/dynamodbstreamsiface"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

const (
	testAWSDynamoDBStreamRoleArn          = "none"
	testAWSDynamoDBStreamAccessKeyID      = "none"
	testAWSDynamoDBStreamSecretAccessKey  = "none"
	testAWSDynamoDBStreamSessionToken     = "none"
	testAWSDynamoDBStreamRegion           = "ap-northeast-1"
	testAWSDynamoDBStreamArnForSmallTable = "smallstreamarn"
	testAWSDynamoDBStreamArnForBigTable   = "bigstreamarn"
	testAWSDynamoDBStreamErrorArn         = "errorarn"
	testAWSDynamoDBSmallTable             = "smalltable" // table with 5 shards
	testAWSDynamoDBBigTable               = "bigtable"   // table with 105 shards
	testAWSDynamoDBErrorTable             = "errortable"
	testAWSDynamoDBInvalidTable           = "invalidtable"
)

var testAwsDynamoDBStreamAuthentication = map[string]string{
	"awsAccessKeyId":     testAWSDynamoDBStreamAccessKeyID,
	"awsSecretAccessKey": testAWSDynamoDBStreamSecretAccessKey,
}

func generateTestDynamoDBStreamShards(shardNum int64) []*dynamodbstreams.Shard {
	var shards []*dynamodbstreams.Shard
	for i := 0; i < int(shardNum); i++ {
		shards = append(shards, &dynamodbstreams.Shard{})
	}
	return shards
}

type parseAwsDynamoDBStreamMetadataTestData struct {
	metadata    map[string]string
	expected    *awsDynamoDBStreamMetadata
	authParams  map[string]string
	isError     bool
	comment     string
	scalerIndex int
}

type awsDynamoDBStreamMetricIdentifier struct {
	metadataTestData *parseAwsDynamoDBStreamMetadataTestData
	scalerIndex      int
	name             string
}

type mockAwsDynamoDBStream struct {
	dynamodbstreamsiface.DynamoDBStreamsAPI
}

func (m *mockAwsDynamoDBStream) DescribeStream(input *dynamodbstreams.DescribeStreamInput) (*dynamodbstreams.DescribeStreamOutput, error) {
	switch *input.StreamArn {
	case testAWSDynamoDBStreamErrorArn:
		return nil, errors.New("Error dynamodbstream DescribeStream")
	case testAWSDynamoDBStreamArnForBigTable:
		if input.ExclusiveStartShardId != nil {
			return &dynamodbstreams.DescribeStreamOutput{
				StreamDescription: &dynamodbstreams.StreamDescription{
					Shards: generateTestDynamoDBStreamShards(5),
				}}, nil
		}
		lastShardID := "testid"
		return &dynamodbstreams.DescribeStreamOutput{
			StreamDescription: &dynamodbstreams.StreamDescription{
				Shards:               generateTestDynamoDBStreamShards(100),
				LastEvaluatedShardId: &lastShardID,
			}}, nil
	default:
		return &dynamodbstreams.DescribeStreamOutput{
			StreamDescription: &dynamodbstreams.StreamDescription{
				Shards: generateTestDynamoDBStreamShards(5),
			}}, nil
	}
}

type mockAwsDynamoDB struct {
	dynamodbiface.DynamoDBAPI
}

func (m *mockAwsDynamoDB) DescribeTable(input *dynamodb.DescribeTableInput) (*dynamodb.DescribeTableOutput, error) {
	switch *input.TableName {
	case testAWSDynamoDBInvalidTable:
		return nil, fmt.Errorf("DynamoDB Stream Arn is invalid")
	case testAWSDynamoDBErrorTable:
		return &dynamodb.DescribeTableOutput{
			Table: &dynamodb.TableDescription{
				LatestStreamArn: aws.String(testAWSDynamoDBStreamErrorArn),
			},
		}, nil
	case testAWSDynamoDBBigTable:
		return &dynamodb.DescribeTableOutput{
			Table: &dynamodb.TableDescription{
				LatestStreamArn: aws.String(testAWSDynamoDBStreamArnForBigTable),
			},
		}, nil
	default:
		return &dynamodb.DescribeTableOutput{
			Table: &dynamodb.TableDescription{
				LatestStreamArn: aws.String(testAWSDynamoDBStreamArnForSmallTable),
			},
		}, nil
	}
}

var testAwsDynamoDBStreamMetadata = []parseAwsDynamoDBStreamMetadataTestData{
	{
		metadata:   map[string]string{},
		authParams: testAWSKinesisAuthentication,
		expected:   &awsDynamoDBStreamMetadata{},
		isError:    true,
		comment:    "metadata empty"},
	{
		metadata: map[string]string{
			"tableName":  testAWSDynamoDBSmallTable,
			"shardCount": "2",
			"awsRegion":  testAWSDynamoDBStreamRegion},
		authParams: testAWSKinesisAuthentication,
		expected: &awsDynamoDBStreamMetadata{
			targetShardCount: 2,
			tableName:        testAWSDynamoDBSmallTable,
			awsRegion:        testAWSDynamoDBStreamRegion,
			awsAuthorization: awsAuthorizationMetadata{
				awsAccessKeyID:     testAWSDynamoDBStreamAccessKeyID,
				awsSecretAccessKey: testAWSDynamoDBStreamSecretAccessKey,
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
			"awsRegion":  testAWSDynamoDBStreamRegion},
		authParams:  testAWSKinesisAuthentication,
		expected:    &awsDynamoDBStreamMetadata{},
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
		expected:    &awsDynamoDBStreamMetadata{},
		isError:     true,
		comment:     "properly formed dynamodb table name, empty region",
		scalerIndex: 2,
	},
	{
		metadata: map[string]string{
			"tableName":  testAWSDynamoDBSmallTable,
			"shardCount": "",
			"awsRegion":  testAWSDynamoDBStreamRegion},
		authParams: testAWSKinesisAuthentication,
		expected: &awsDynamoDBStreamMetadata{
			targetShardCount: defaultTargetDBStreamShardCount,
			tableName:        testAWSDynamoDBSmallTable,
			awsRegion:        testAWSDynamoDBStreamRegion,
			awsAuthorization: awsAuthorizationMetadata{
				awsAccessKeyID:     testAWSDynamoDBStreamAccessKeyID,
				awsSecretAccessKey: testAWSDynamoDBStreamSecretAccessKey,
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
			"awsRegion":  testAWSDynamoDBStreamRegion},
		authParams: testAWSKinesisAuthentication,
		expected: &awsDynamoDBStreamMetadata{
			targetShardCount: defaultTargetDBStreamShardCount,
			tableName:        testAWSDynamoDBSmallTable,
			awsRegion:        testAWSDynamoDBStreamRegion,
			awsAuthorization: awsAuthorizationMetadata{
				awsAccessKeyID:     testAWSDynamoDBStreamAccessKeyID,
				awsSecretAccessKey: testAWSDynamoDBStreamSecretAccessKey,
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
			"awsRegion":  testAWSDynamoDBStreamRegion},
		authParams: map[string]string{
			"awsAccessKeyID":     "",
			"awsSecretAccessKey": testAWSDynamoDBStreamSecretAccessKey,
		},
		expected:    &awsDynamoDBStreamMetadata{},
		isError:     true,
		comment:     "with AWS static credentials from TriggerAuthentication, missing Access Key Id",
		scalerIndex: 5,
	},
	{metadata: map[string]string{
		"tableName":  testAWSDynamoDBSmallTable,
		"shardCount": "2",
		"awsRegion":  testAWSDynamoDBStreamRegion},
		authParams: map[string]string{
			"awsAccessKeyID":     testAWSDynamoDBStreamAccessKeyID,
			"awsSecretAccessKey": "",
		},
		expected:    &awsDynamoDBStreamMetadata{},
		isError:     true,
		comment:     "with AWS static credentials from TriggerAuthentication, missing Secret Access Key",
		scalerIndex: 6,
	},
	{
		metadata: map[string]string{
			"tableName":  testAWSDynamoDBSmallTable,
			"shardCount": "2",
			"awsRegion":  testAWSDynamoDBStreamRegion},
		authParams: map[string]string{
			"awsAccessKeyID":     testAWSDynamoDBStreamAccessKeyID,
			"awsSecretAccessKey": testAWSDynamoDBStreamSecretAccessKey,
			"awsSessionToken":    testAWSDynamoDBStreamSessionToken,
		},
		expected: &awsDynamoDBStreamMetadata{
			targetShardCount: 2,
			tableName:        testAWSDynamoDBSmallTable,
			awsRegion:        testAWSDynamoDBStreamRegion,
			awsAuthorization: awsAuthorizationMetadata{
				awsAccessKeyID:     testAWSDynamoDBStreamAccessKeyID,
				awsSecretAccessKey: testAWSDynamoDBStreamSecretAccessKey,
				awsSessionToken:    testAWSDynamoDBStreamSessionToken,
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
			"awsRegion":  testAWSDynamoDBStreamRegion},
		authParams: map[string]string{
			"awsAccessKeyID":     "",
			"awsSecretAccessKey": testAWSDynamoDBStreamSecretAccessKey,
			"awsSessionToken":    testAWSDynamoDBStreamSessionToken,
		},
		expected:    &awsDynamoDBStreamMetadata{},
		isError:     true,
		comment:     "with AWS temporary credentials from TriggerAuthentication, missing Access Key Id",
		scalerIndex: 5,
	},
	{metadata: map[string]string{
		"tableName":  testAWSDynamoDBSmallTable,
		"shardCount": "2",
		"awsRegion":  testAWSDynamoDBStreamRegion},
		authParams: map[string]string{
			"awsAccessKeyID":     testAWSDynamoDBStreamAccessKeyID,
			"awsSecretAccessKey": "",
			"awsSessionToken":    testAWSDynamoDBStreamSessionToken,
		},
		expected:    &awsDynamoDBStreamMetadata{},
		isError:     true,
		comment:     "with AWS temporary credentials from TriggerAuthentication, missing Secret Access Key",
		scalerIndex: 6,
	},
	{metadata: map[string]string{
		"tableName":  testAWSDynamoDBSmallTable,
		"shardCount": "2",
		"awsRegion":  testAWSDynamoDBStreamRegion},
		authParams: map[string]string{
			"awsRoleArn": testAWSDynamoDBStreamRoleArn,
		},
		expected: &awsDynamoDBStreamMetadata{
			targetShardCount: 2,
			tableName:        testAWSDynamoDBSmallTable,
			awsRegion:        testAWSDynamoDBStreamRegion,
			awsAuthorization: awsAuthorizationMetadata{
				awsRoleArn:       testAWSDynamoDBStreamRoleArn,
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
		"awsRegion":     testAWSDynamoDBStreamRegion,
		"identityOwner": "operator"},
		authParams: map[string]string{},
		expected: &awsDynamoDBStreamMetadata{
			targetShardCount: 2,
			tableName:        testAWSDynamoDBSmallTable,
			awsRegion:        testAWSDynamoDBStreamRegion,
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

var awsDynamoDBStreamMetricIdentifiers = []awsDynamoDBStreamMetricIdentifier{
	{&testAwsDynamoDBStreamMetadata[1], 0, fmt.Sprintf("s0-aws-dynamodb-stream-%s", testAWSDynamoDBSmallTable)},
	{&testAwsDynamoDBStreamMetadata[1], 1, fmt.Sprintf("s1-aws-dynamodb-stream-%s", testAWSDynamoDBSmallTable)},
}

var awsDynamoDBStreamGetMetricTestData = []*awsDynamoDBStreamMetadata{
	{tableName: testAWSDynamoDBBigTable},
	{tableName: testAWSDynamoDBSmallTable},
	{tableName: testAWSDynamoDBErrorTable},
	{tableName: testAWSDynamoDBInvalidTable},
}

func TestParseAwsDynamoDBStreamMetadata(t *testing.T) {
	for _, testData := range testAwsDynamoDBStreamMetadata {
		result, err := parseAwsDynamoDBStreamMetadata(&ScalerConfig{TriggerMetadata: testData.metadata, ResolvedEnv: testAwsDynamoDBStreamAuthentication, AuthParams: testData.authParams, ScalerIndex: testData.scalerIndex})
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

func TestAwsDynamoDBStreamGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range awsDynamoDBStreamMetricIdentifiers {
		ctx := context.Background()
		meta, err := parseAwsDynamoDBStreamMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ResolvedEnv: testAwsDynamoDBStreamAuthentication, AuthParams: testData.metadataTestData.authParams, ScalerIndex: testData.scalerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		streamArn, err := getDynamoDBStreamArn(&mockAwsDynamoDB{}, &meta.tableName)
		if err != nil {
			t.Fatal("Could not get dynamodb stream arn:", err)
		}
		mockAwsDynamoDBStreamScaler := awsDynamoDBStreamScaler{"", meta, streamArn, &mockAwsDynamoDBStream{}}
		metricSpec := mockAwsDynamoDBStreamScaler.GetMetricSpecForScaling(ctx)
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func TestAwsDynamoDBStreamScalerGetMetrics(t *testing.T) {
	var selector labels.Selector
	for _, meta := range awsDynamoDBStreamGetMetricTestData {
		var value []external_metrics.ExternalMetricValue
		var err error
		var streamArn *string
		streamArn, err = getDynamoDBStreamArn(&mockAwsDynamoDB{}, &meta.tableName)
		if err == nil {
			scaler := awsDynamoDBStreamScaler{"", meta, streamArn, &mockAwsDynamoDBStream{}}
			value, err = scaler.GetMetrics(context.Background(), "MetricName", selector)
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

func TestAwsDynamoDBStreamScalerIsActive(t *testing.T) {
	for _, meta := range awsDynamoDBStreamGetMetricTestData {
		var value bool
		var err error
		var streamArn *string
		streamArn, err = getDynamoDBStreamArn(&mockAwsDynamoDB{}, &meta.tableName)
		if err == nil {
			scaler := awsDynamoDBStreamScaler{"", meta, streamArn, &mockAwsDynamoDBStream{}}
			value, err = scaler.IsActive(context.Background())
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
