package scalers

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
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

type parseDynamoDbMetadataTestData struct {
	name             string
	metadata         map[string]string
	resolvedEnv      map[string]string
	authParams       map[string]string
	expectedMetadata *awsDynamoDBMetadata
	expectedError    error
}

var dynamoTestCases = []parseDynamoDbMetadataTestData{
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
			"expressionAttributeNames": "\"{ \"#yr\" : \"year\" }\"",
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
			"expressionAttributeValues": "{\":yyyy\": {\"N\": 1994}}",
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
			"expressionAttributeValues": "{\":yyyy\": {\"N\": 1994}}",
			"targetValue":               "no-valid",
		},
		authParams:    map[string]string{},
		expectedError: errors.New("error parsing metadata targetValue"),
	},
	{
		name: "no aws given",
		metadata: map[string]string{
			"tableName":                 "test",
			"awsRegion":                 "eu-west-1",
			"keyConditionExpression":    "#yr = :yyyy",
			"expressionAttributeNames":  "{ \"#yr\" : \"year\" }",
			"expressionAttributeValues": "{\":yyyy\": {\"N\": 1994}}",
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
			"expressionAttributeValues": "{\":yyyy\": {\"N\": 1994}}",
			"targetValue":               "3",
		},
		authParams:    testAWSDynamoAuthentication,
		expectedError: nil,
		expectedMetadata: &awsDynamoDBMetadata{
			tableName:                 "test",
			awsRegion:                 "eu-west-1",
			keyConditionExpression:    "#yr = :yyyy",
			expressionAttributeNames:  "{ \"#yr\" : \"year\" }",
			expressionAttributeValues: "{\":yyyy\": {\"N\": 1994}}",
			targetValue:               3,
			scalerIndex:               0,
			metricName:                "",
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

type mockDynamoDb struct {
	dynamodbiface.DynamoDBAPI
}

var result int64 = 4
var zero int64 = 0

func (c *mockDynamoDb) Query(input *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
	switch *input.TableName {
	case testAWSCloudwatchErrorMetric:
		return nil, errors.New("error")
	case testAWSCloudwatchNoValueMetric:
		return &dynamodb.QueryOutput{
			Count: &zero,
		}, nil
	}

	return &dynamodb.QueryOutput{
		Count: &result,
	}, nil
}

var awsDynamoDBGetMetricTestData = []awsDynamoDBMetadata{
	{
		tableName:                 "ValidTable",
		awsRegion:                 "eu-west-1",
		keyConditionExpression:    "#yr = :yyyy",
		expressionAttributeNames:  "{ \"#yr\" : \"year\" }",
		expressionAttributeValues: "{\":yyyy\": {\"N\": \"1994\"}}",
		targetValue:               3,
	},
	{
		tableName:                 testAWSDynamoErrorTable,
		awsRegion:                 "eu-west-1",
		keyConditionExpression:    "#yr = :yyyy",
		expressionAttributeNames:  "{ \"#yr\" : \"year\" }",
		expressionAttributeValues: "{\":yyyy\": {\"N\": \"1994\"}}",
		targetValue:               3,
	},
	{
		tableName:                 testAWSDynamoNoValueTable,
		awsRegion:                 "eu-west-1",
		keyConditionExpression:    "#yr = :yyyy",
		expressionAttributeNames:  "{ \"#yr\" : \"year\" }",
		expressionAttributeValues: "{\":yyyy\": {\"N\": \"1994\"}}",
		targetValue:               3,
	},
}

func TestDynamoGetMetrics(t *testing.T) {
	var selector labels.Selector

	for _, meta := range awsDynamoDBGetMetricTestData {
		t.Run(meta.tableName, func(t *testing.T) {
			scaler := awsDynamoDbScaler{&meta, &mockDynamoDb{}}

			value, err := scaler.GetMetrics(context.Background(), meta.metricName, selector)
			switch meta.tableName {
			case testAWSDynamoErrorTable:
				assert.Error(t, err, "expect error because of cloudwatch api error")
			case testAWSDynamoNoValueTable:
				assert.NoError(t, err, "dont expect error when returning empty metric list from cloudwatch")
			default:
				assert.EqualValues(t, int64(4), value[0].Value.Value())
			}
		})
	}
}

func TestDynamoGetQueryMetrics(t *testing.T) {
	for _, meta := range awsDynamoDBGetMetricTestData {
		t.Run(meta.tableName, func(t *testing.T) {
			scaler := awsDynamoDbScaler{&meta, &mockDynamoDb{}}

			value, err := scaler.GetQueryMetrics()
			switch meta.tableName {
			case testAWSDynamoErrorTable:
				assert.Error(t, err, "expect error because of cloudwatch api error")
			case testAWSDynamoNoValueTable:
				assert.NoError(t, err, "dont expect error when returning empty metric list from cloudwatch")
			default:
				assert.EqualValues(t, int64(4), value)
			}
		})
	}
}
