package scalers

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"

	awsutils "github.com/kedacore/keda/v2/keda-scalers/aws"
	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
)

const (
	testAWSDynamoAccessKeyID     = "none"
	testAWSDynamoSecretAccessKey = "none"
	testAWSDynamoErrorTable      = "Error"
	testAWSDynamoNoValueTable    = "NoValue"
	testAWSDynamoIndexTable      = "Index"
)

var testAWSDynamoAuthentication = map[string]string{
	"awsAccessKeyId":     testAWSDynamoAccessKeyID,
	"awsSecretAccessKey": testAWSDynamoSecretAccessKey,
}

type parseDynamoDBMetadataTestData struct {
	name             string
	metadata         map[string]string
	resolvedEnv      map[string]string
	authParams       map[string]string
	expectedMetadata *awsDynamoDBMetadata
	expectedError    error
}

var (
	// ErrAwsDynamoNoTableName is returned when "tableName" is missing from the config.
	ErrAwsDynamoNoTableName = errors.New(`missing required parameter "tableName"`)

	// ErrAwsDynamoNoAwsRegion is returned when "awsRegion" is missing from the config.
	ErrAwsDynamoNoAwsRegion = errors.New(`missing required parameter "awsRegion"`)

	// ErrAwsDynamoNoKeyConditionExpression is returned when "keyConditionExpression" is missing from the config.
	ErrAwsDynamoNoKeyConditionExpression = errors.New(`missing required parameter "keyConditionExpression"`)
)

var dynamoTestCases = []parseDynamoDBMetadataTestData{
	{
		name:          "no tableName given",
		metadata:      map[string]string{},
		authParams:    map[string]string{},
		expectedError: ErrAwsDynamoNoTableName,
	},
	{
		name:          "no awsRegion given",
		metadata:      map[string]string{"tableName": "test"},
		authParams:    map[string]string{},
		expectedError: ErrAwsDynamoNoAwsRegion,
	},
	{
		name: "no keyConditionExpression given",
		metadata: map[string]string{
			"tableName": "test",
			"awsRegion": "eu-west-1",
		},
		authParams:    map[string]string{},
		expectedError: ErrAwsDynamoNoKeyConditionExpression,
	},
	{
		name: "no expressionAttributeNames given",
		metadata: map[string]string{
			"tableName":              "test",
			"awsRegion":              "eu-west-1",
			"keyConditionExpression": "#yr = :yyyy",
		},
		authParams:    map[string]string{},
		expectedError: ErrAwsDynamoNoExpressionAttributeNames,
	},
	{
		name: "no expressionAttributeValues given",
		metadata: map[string]string{
			"tableName":                "test",
			"awsRegion":                "eu-west-1",
			"keyConditionExpression":   "#yr = :yyyy",
			"expressionAttributeNames": "{ \"#yr\" : \"year\" }",
		},
		authParams:    map[string]string{},
		expectedError: ErrAwsDynamoNoExpressionAttributeValues,
	},
	{
		name: "no targetValue given",
		metadata: map[string]string{
			"tableName":                 "test",
			"awsRegion":                 "eu-west-1",
			"keyConditionExpression":    "#yr = :yyyy",
			"expressionAttributeNames":  "{ \"#yr\" : \"year\" }",
			"expressionAttributeValues": "{\":yyyy\": {\"N\": \"1994\"}}",
		},
		authParams:    map[string]string{},
		expectedError: ErrAwsDynamoNoTargetValue,
	},
	{
		name: "invalid targetValue given",
		metadata: map[string]string{
			"tableName":                 "test",
			"awsRegion":                 "eu-west-1",
			"keyConditionExpression":    "#yr = :yyyy",
			"expressionAttributeNames":  "{ \"#yr\" : \"year\" }",
			"expressionAttributeValues": "{\":yyyy\": {\"N\": \"1994\"}}",
			"targetValue":               "no-valid",
		},
		authParams:    map[string]string{},
		expectedError: errors.New(`error parsing DynamoDb metadata: unable to set param "targetValue" value`),
	},
	{
		name: "invalid activationTargetValue given",
		metadata: map[string]string{
			"tableName":                 "test",
			"awsRegion":                 "eu-west-1",
			"keyConditionExpression":    "#yr = :yyyy",
			"expressionAttributeNames":  "{ \"#yr\" : \"year\" }",
			"expressionAttributeValues": "{\":yyyy\": {\"N\": \"1994\"}}",
			"targetValue":               "1",
			"activationTargetValue":     "no-valid",
		},
		authParams:    map[string]string{},
		expectedError: errors.New(`unable to set param "activationTargetValue"`),
	},
	{
		name: "malformed expressionAttributeNames",
		metadata: map[string]string{
			"tableName":                 "test",
			"awsRegion":                 "eu-west-1",
			"keyConditionExpression":    "#yr = :yyyy",
			"expressionAttributeNames":  "{ \"malformed\" }",
			"expressionAttributeValues": "{\":yyyy\": {\"N\": \"1994\"}}",
			"targetValue":               "3",
		},
		authParams:    map[string]string{},
		expectedError: ErrAwsDynamoInvalidExpressionAttributeNames,
	},
	{
		name: "empty expressionAttributeNames",
		metadata: map[string]string{
			"tableName":                 "test",
			"awsRegion":                 "eu-west-1",
			"keyConditionExpression":    "#yr = :yyyy",
			"expressionAttributeNames":  "{}",
			"expressionAttributeValues": "{\":yyyy\": {\"N\": \"1994\"}}",
			"targetValue":               "3",
		},
		authParams:    map[string]string{},
		expectedError: ErrAwsDynamoEmptyExpressionAttributeNames,
	},
	{
		name: "malformed expressionAttributeValues",
		metadata: map[string]string{
			"tableName":                 "test",
			"awsRegion":                 "eu-west-1",
			"keyConditionExpression":    "#yr = :yyyy",
			"expressionAttributeNames":  "{ \"#yr\" : \"year\" }",
			"expressionAttributeValues": "{\":yyyy\": {\"N\": 1994 }}", // This should not be an JSON int.
			"targetValue":               "3",
		},
		authParams:    map[string]string{},
		expectedError: ErrAwsDynamoInvalidExpressionAttributeValues,
	},
	{
		name: "no aws given",
		metadata: map[string]string{
			"tableName":                 "test",
			"awsRegion":                 "eu-west-1",
			"keyConditionExpression":    "#yr = :yyyy",
			"expressionAttributeNames":  "{ \"#yr\" : \"year\" }",
			"expressionAttributeValues": "{\":yyyy\": {\"N\": \"1994\"}}",
			"targetValue":               "3",
		},
		authParams:    map[string]string{},
		expectedError: awsutils.ErrAwsNoAccessKey,
	},
	{
		name: "authentication provided",
		metadata: map[string]string{
			"tableName":                 "test",
			"awsRegion":                 "eu-west-1",
			"keyConditionExpression":    "#yr = :yyyy",
			"expressionAttributeNames":  "{ \"#yr\" : \"year\" }",
			"expressionAttributeValues": "{\":yyyy\": {\"N\": \"1994\"}}",
			"targetValue":               "3",
		},
		authParams:    testAWSDynamoAuthentication,
		expectedError: nil,
		expectedMetadata: &awsDynamoDBMetadata{
			TableName:                 "test",
			AwsRegion:                 "eu-west-1",
			KeyConditionExpression:    "#yr = :yyyy",
			expressionAttributeNames:  map[string]string{"#yr": year},
			expressionAttributeValues: map[string]types.AttributeValue{":yyyy": yearAttr},
			TargetValue:               3,
			triggerIndex:              1,
			metricName:                "s1-aws-dynamodb-test",
			awsAuthorization: awsutils.AuthorizationMetadata{
				AwsAccessKeyID:     "none",
				AwsSecretAccessKey: "none",
				PodIdentityOwner:   true,
				AwsRegion:          testAWSRegion,
			},
		},
	},
	{
		name: "properly formed dynamo name and region with custom endpoint",
		metadata: map[string]string{
			"tableName":                 "test",
			"awsRegion":                 "eu-west-1",
			"awsEndpoint":               "http://localhost:4566",
			"keyConditionExpression":    "#yr = :yyyy",
			"expressionAttributeNames":  "{ \"#yr\" : \"year\" }",
			"expressionAttributeValues": "{\":yyyy\": {\"N\": \"1994\"}}",
			"targetValue":               "3",
		},
		authParams:    testAWSDynamoAuthentication,
		expectedError: nil,
		expectedMetadata: &awsDynamoDBMetadata{
			TableName:                 "test",
			AwsRegion:                 "eu-west-1",
			AwsEndpoint:               "http://localhost:4566",
			KeyConditionExpression:    "#yr = :yyyy",
			expressionAttributeNames:  map[string]string{"#yr": year},
			expressionAttributeValues: map[string]types.AttributeValue{":yyyy": yearAttr},
			TargetValue:               3,
			triggerIndex:              1,
			metricName:                "s1-aws-dynamodb-test",
			awsAuthorization: awsutils.AuthorizationMetadata{
				AwsAccessKeyID:     "none",
				AwsSecretAccessKey: "none",
				PodIdentityOwner:   true,
				AwsRegion:          testAWSRegion,
			},
		},
	},
	{
		name: "properly formed dynamo name and region with activationTargetValue",
		metadata: map[string]string{
			"tableName":                 "test",
			"awsRegion":                 "eu-west-1",
			"keyConditionExpression":    "#yr = :yyyy",
			"expressionAttributeNames":  "{ \"#yr\" : \"year\" }",
			"expressionAttributeValues": "{\":yyyy\": {\"N\": \"1994\"}}",
			"activationTargetValue":     "1",
			"targetValue":               "3",
		},
		authParams:    testAWSDynamoAuthentication,
		expectedError: nil,
		expectedMetadata: &awsDynamoDBMetadata{
			TableName:                 "test",
			AwsRegion:                 "eu-west-1",
			KeyConditionExpression:    "#yr = :yyyy",
			expressionAttributeNames:  map[string]string{"#yr": year},
			expressionAttributeValues: map[string]types.AttributeValue{":yyyy": yearAttr},
			ActivationTargetValue:     1,
			TargetValue:               3,
			triggerIndex:              1,
			metricName:                "s1-aws-dynamodb-test",
			awsAuthorization: awsutils.AuthorizationMetadata{
				AwsAccessKeyID:     "none",
				AwsSecretAccessKey: "none",
				PodIdentityOwner:   true,
				AwsRegion:          testAWSRegion,
			},
		},
	},
	{
		name: "properly formed dynamo name and region with index name",
		metadata: map[string]string{
			"tableName":                 "test",
			"awsRegion":                 "eu-west-1",
			"indexName":                 "test-index",
			"keyConditionExpression":    "#yr = :yyyy",
			"expressionAttributeNames":  "{ \"#yr\" : \"year\" }",
			"expressionAttributeValues": "{\":yyyy\": {\"N\": \"1994\"}}",
			"targetValue":               "3",
		},
		authParams:    testAWSDynamoAuthentication,
		expectedError: nil,
		expectedMetadata: &awsDynamoDBMetadata{
			TableName:                 "test",
			AwsRegion:                 "eu-west-1",
			IndexName:                 "test-index",
			KeyConditionExpression:    "#yr = :yyyy",
			expressionAttributeNames:  map[string]string{"#yr": year},
			expressionAttributeValues: map[string]types.AttributeValue{":yyyy": yearAttr},
			TargetValue:               3,
			triggerIndex:              1,
			metricName:                "s1-aws-dynamodb-test",
			awsAuthorization: awsutils.AuthorizationMetadata{
				AwsAccessKeyID:     "none",
				AwsSecretAccessKey: "none",
				PodIdentityOwner:   true,
				AwsRegion:          testAWSRegion,
			},
		},
	},
}

func TestParseDynamoMetadata(t *testing.T) {
	for _, tc := range dynamoTestCases {
		t.Run(tc.name, func(t *testing.T) {
			metadata, err := parseAwsDynamoDBMetadata(&scalersconfig.ScalerConfig{
				TriggerMetadata: tc.metadata,
				AuthParams:      tc.authParams,
				ResolvedEnv:     tc.resolvedEnv,
				TriggerIndex:    1,
			})
			if tc.expectedError != nil {
				assert.ErrorContains(t, err, tc.expectedError.Error())
			} else {
				assert.NoError(t, err)
				fmt.Println(tc.name)
				assert.Equal(t, tc.expectedMetadata, metadata)
			}
		})
	}
}

type mockDynamoDB struct {
}

var result int32 = 4
var indexResult int32 = 2
var empty int32

func (c *mockDynamoDB) Query(_ context.Context, input *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	switch *input.TableName {
	case testAWSCloudwatchErrorMetric:
		return nil, errors.New("error")
	case testAWSCloudwatchNoValueMetric:
		return &dynamodb.QueryOutput{
			Count: empty,
		}, nil
	}

	if input.IndexName != nil {
		return &dynamodb.QueryOutput{
			Count: indexResult,
		}, nil
	}

	return &dynamodb.QueryOutput{
		Count: result,
	}, nil
}

var year = "year"
var target = "1994"
var yearAttr = &types.AttributeValueMemberN{Value: target}

var awsDynamoDBGetMetricTestData = []awsDynamoDBMetadata{
	{
		TableName:                 "ValidTable",
		AwsRegion:                 "eu-west-1",
		KeyConditionExpression:    "#yr = :yyyy",
		expressionAttributeNames:  map[string]string{"#yr": year},
		expressionAttributeValues: map[string]types.AttributeValue{":yyyy": yearAttr},
		TargetValue:               3,
	},
	{
		TableName:                 testAWSDynamoErrorTable,
		AwsRegion:                 "eu-west-1",
		KeyConditionExpression:    "#yr = :yyyy",
		expressionAttributeNames:  map[string]string{"#yr": year},
		expressionAttributeValues: map[string]types.AttributeValue{":yyyy": yearAttr},
		TargetValue:               3,
	},
	{
		TableName:                 testAWSDynamoNoValueTable,
		AwsRegion:                 "eu-west-1",
		KeyConditionExpression:    "#yr = :yyyy",
		expressionAttributeNames:  map[string]string{"#yr": year},
		expressionAttributeValues: map[string]types.AttributeValue{":yyyy": yearAttr},
		TargetValue:               3,
	},
	{
		TableName:                 testAWSDynamoIndexTable,
		AwsRegion:                 "eu-west-1",
		IndexName:                 "test-index",
		KeyConditionExpression:    "#yr = :yyyy",
		expressionAttributeNames:  map[string]string{"#yr": year},
		expressionAttributeValues: map[string]types.AttributeValue{":yyyy": yearAttr},
		ActivationTargetValue:     3,
		TargetValue:               3,
	},
}

func TestDynamoGetMetrics(t *testing.T) {
	for _, meta := range awsDynamoDBGetMetricTestData {
		t.Run(meta.TableName, func(t *testing.T) {
			scaler := awsDynamoDBScaler{"", &meta, &mockDynamoDB{}, logr.Discard()}

			value, _, err := scaler.GetMetricsAndActivity(context.Background(), "aws-dynamodb")
			switch meta.TableName {
			case testAWSDynamoErrorTable:
				assert.EqualError(t, err, "error", "expect error because of dynamodb api error")
			case testAWSDynamoNoValueTable:
				assert.NoError(t, err, "dont expect error when returning empty result from dynamodb")
			case testAWSDynamoIndexTable:
				assert.EqualValues(t, int64(2), value[0].Value.Value())
			default:
				assert.EqualValues(t, int64(4), value[0].Value.Value())
			}
		})
	}
}

func TestDynamoGetQueryMetrics(t *testing.T) {
	for _, meta := range awsDynamoDBGetMetricTestData {
		t.Run(meta.TableName, func(t *testing.T) {
			scaler := awsDynamoDBScaler{"", &meta, &mockDynamoDB{}, logr.Discard()}

			value, err := scaler.GetQueryMetrics(context.Background())
			switch meta.TableName {
			case testAWSDynamoErrorTable:
				assert.EqualError(t, err, "error", "expect error because of dynamodb api error")
			case testAWSDynamoNoValueTable:
				assert.NoError(t, err, "dont expect error when returning empty result from dynamodb")
			case testAWSDynamoIndexTable:
				assert.EqualValues(t, int64(2), value)
			default:
				assert.EqualValues(t, int64(4), value)
			}
		})
	}
}

func TestDynamoIsActive(t *testing.T) {
	for _, meta := range awsDynamoDBGetMetricTestData {
		t.Run(meta.TableName, func(t *testing.T) {
			scaler := awsDynamoDBScaler{"", &meta, &mockDynamoDB{}, logr.Discard()}

			_, value, err := scaler.GetMetricsAndActivity(context.Background(), "aws-dynamodb")
			switch meta.TableName {
			case testAWSDynamoErrorTable:
				assert.EqualError(t, err, "error", "expect error because of dynamodb api error")
			case testAWSDynamoNoValueTable:
				assert.NoError(t, err, "dont expect error when returning empty result from dynamodb")
			case testAWSDynamoIndexTable:
				assert.EqualValues(t, false, value)
			default:
				assert.EqualValues(t, true, value)
			}
		})
	}
}
