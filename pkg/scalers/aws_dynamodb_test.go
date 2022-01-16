package scalers

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	testAWSDynamoRoleArn         = "none"
	testAWSDynamoAccessKeyID     = "none"
	testAWSDynamoSecretAccessKey = "none"
	testAWSDynamoErrorMetric     = "Error"
	testAWSDynamoNoValueMetric   = "NoValue"
)

var testAWSDynamoResolvedEnv = map[string]string{
	"AWS_ACCESS_KEY":        "none",
	"AWS_SECRET_ACCESS_KEY": "none",
}

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

type dynamoDbMetricIdentifier struct {
	metadataTestData *parseDynamoDbMetadataTestData
	scalerIndex      int
	name             string
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
