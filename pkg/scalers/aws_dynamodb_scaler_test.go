package scalers

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	testAWSDynamoAccessKeyID     = "none"
	testAWSDynamoSecretAccessKey = "none"
	testAWSDynamoErrorTable      = "Error"
	testAWSDynamoNoValueTable    = "NoValue"
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

var dynamoTestCases = []parseDynamoDBMetadataTestData{
	{
		name:          "no tableName given",
		metadata:      map[string]string{},
		authParams:    map[string]string{},
		expectedError: errors.New("no tableName given"),
	},
	{
		name:          "no awsRegion given",
		metadata:      map[string]string{"tableName": "test"},
		authParams:    map[string]string{},
		expectedError: errors.New("no awsRegion given"),
	},
	{
		name: "no keyConditionExpression given",
		metadata: map[string]string{
			"tableName": "test",
			"awsRegion": "eu-west-1",
		},
		authParams:    map[string]string{},
		expectedError: errors.New("no keyConditionExpression given"),
	},
	{
		name: "no expressionAttributeNames given",
		metadata: map[string]string{
			"tableName":              "test",
			"awsRegion":              "eu-west-1",
			"keyConditionExpression": "#yr = :yyyy",
		},
		authParams:    map[string]string{},
		expectedError: errors.New("no expressionAttributeNames given"),
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
		expectedError: errors.New("no expressionAttributeValues given"),
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
		expectedError: errors.New("no targetValue given"),
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
		expectedError: errors.New("error parsing metadata targetValue"),
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
		expectedError: errors.New("error parsing metadata targetValue"),
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
		expectedError: errors.New("error parsing expressionAttributeNames: invalid JSON input: missing colon after key \"malformed\""),
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
		expectedError: errors.New("error parsing expressionAttributeNames: empty map"),
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
		expectedError: errors.New("error parsing expressionAttributeValues: json: cannot unmarshal number into Go struct field AttributeValue.N of type string"),
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
		expectedError: errors.New("awsAccessKeyID not found"),
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
			tableName:                 "test",
			awsRegion:                 "eu-west-1",
			keyConditionExpression:    "#yr = :yyyy",
			expressionAttributeNames:  map[string]*string{"#yr": &year},
			expressionAttributeValues: map[string]*dynamodb.AttributeValue{":yyyy": &yearAttr},
			targetValue:               3,
			scalerIndex:               1,
			metricName:                "s1-aws-dynamodb-test",
			awsAuthorization: awsAuthorizationMetadata{
				awsAccessKeyID:     "none",
				awsSecretAccessKey: "none",
				podIdentityOwner:   true,
			},
		},
	},
}

func TestParseDynamoMetadata(t *testing.T) {
	for _, tc := range dynamoTestCases {
		t.Run(tc.name, func(t *testing.T) {
			metadata, err := parseAwsDynamoDBMetadata(&ScalerConfig{
				TriggerMetadata: tc.metadata,
				AuthParams:      tc.authParams,
				ResolvedEnv:     tc.resolvedEnv,
				ScalerIndex:     1,
			})
			if tc.expectedError != nil {
				assert.Contains(t, err.Error(), tc.expectedError.Error())
			} else {
				assert.NoError(t, err)
				fmt.Println(tc.name)
				assert.Equal(t, tc.expectedMetadata, metadata)
			}
		})
	}
}

type mockDynamoDB struct {
	dynamodbiface.DynamoDBAPI
}

var result int64 = 4
var empty int64

func (c *mockDynamoDB) Query(input *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
	switch *input.TableName {
	case testAWSCloudwatchErrorMetric:
		return nil, errors.New("error")
	case testAWSCloudwatchNoValueMetric:
		return &dynamodb.QueryOutput{
			Count: &empty,
		}, nil
	}

	return &dynamodb.QueryOutput{
		Count: &result,
	}, nil
}

var year = "year"
var target = "1994"
var yearAttr = dynamodb.AttributeValue{N: &target}

var awsDynamoDBGetMetricTestData = []awsDynamoDBMetadata{
	{
		tableName:                 "ValidTable",
		awsRegion:                 "eu-west-1",
		keyConditionExpression:    "#yr = :yyyy",
		expressionAttributeNames:  map[string]*string{"#yr": &year},
		expressionAttributeValues: map[string]*dynamodb.AttributeValue{":yyyy": &yearAttr},
		targetValue:               3,
	},
	{
		tableName:                 testAWSDynamoErrorTable,
		awsRegion:                 "eu-west-1",
		keyConditionExpression:    "#yr = :yyyy",
		expressionAttributeNames:  map[string]*string{"#yr": &year},
		expressionAttributeValues: map[string]*dynamodb.AttributeValue{":yyyy": &yearAttr},
		targetValue:               3,
	},
	{
		tableName:                 testAWSDynamoNoValueTable,
		awsRegion:                 "eu-west-1",
		keyConditionExpression:    "#yr = :yyyy",
		expressionAttributeNames:  map[string]*string{"#yr": &year},
		expressionAttributeValues: map[string]*dynamodb.AttributeValue{":yyyy": &yearAttr},
		targetValue:               3,
	},
}

func TestDynamoGetMetrics(t *testing.T) {
	var selector labels.Selector

	for _, meta := range awsDynamoDBGetMetricTestData {
		t.Run(meta.tableName, func(t *testing.T) {
			scaler := awsDynamoDBScaler{"", &meta, &mockDynamoDB{}, logr.Discard()}

			value, err := scaler.GetMetrics(context.Background(), "aws-dynamodb", selector)
			switch meta.tableName {
			case testAWSDynamoErrorTable:
				assert.Error(t, err, "expect error because of dynamodb api error")
			case testAWSDynamoNoValueTable:
				assert.NoError(t, err, "dont expect error when returning empty result from dynamodb")
			default:
				assert.EqualValues(t, int64(4), value[0].Value.Value())
			}
		})
	}
}

func TestDynamoGetQueryMetrics(t *testing.T) {
	for _, meta := range awsDynamoDBGetMetricTestData {
		t.Run(meta.tableName, func(t *testing.T) {
			scaler := awsDynamoDBScaler{"", &meta, &mockDynamoDB{}, logr.Discard()}

			value, err := scaler.GetQueryMetrics()
			switch meta.tableName {
			case testAWSDynamoErrorTable:
				assert.Error(t, err, "expect error because of dynamodb api error")
			case testAWSDynamoNoValueTable:
				assert.NoError(t, err, "dont expect error when returning empty metric list from cloudwatch")
			default:
				assert.EqualValues(t, int64(4), value)
			}
		})
	}
}

func TestDynamoIsActive(t *testing.T) {
	for _, meta := range awsDynamoDBGetMetricTestData {
		t.Run(meta.tableName, func(t *testing.T) {
			scaler := awsDynamoDBScaler{"", &meta, &mockDynamoDB{}, logr.Discard()}

			value, err := scaler.IsActive(context.Background())
			switch meta.tableName {
			case testAWSDynamoErrorTable:
				assert.Error(t, err, "expect error because of cloudwatch api error")
			case testAWSDynamoNoValueTable:
				assert.NoError(t, err, "dont expect error when returning empty result from dynamodb")
			default:
				assert.EqualValues(t, true, value)
			}
		})
	}
}
