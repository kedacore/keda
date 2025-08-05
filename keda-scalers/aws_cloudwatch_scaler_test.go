package scalers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"

	awsutils "github.com/kedacore/keda/v2/keda-scalers/aws"
	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
)

const (
	testAWSCloudwatchRoleArn         = "none"
	testAWSCloudwatchAccessKeyID     = "none"
	testAWSCloudwatchSecretAccessKey = "none"
	testAWSCloudwatchSessionToken    = "none"
	testAWSCloudwatchErrorMetric     = "Error"
	testAWSCloudwatchNoValueMetric   = "NoValue"
	testAWSCloudwatchEmptyValues     = "EmptyValues"
)

var testAWSCloudwatchResolvedEnv = map[string]string{
	"AWS_ACCESS_KEY":        "none",
	"AWS_SECRET_ACCESS_KEY": "none",
}

var testAWSAuthentication = map[string]string{
	"awsAccessKeyId":     testAWSCloudwatchAccessKeyID,
	"awsSecretAccessKey": testAWSCloudwatchSecretAccessKey,
}

type parseAWSCloudwatchMetadataTestData struct {
	metadata   map[string]string
	authParams map[string]string
	isError    bool
	comment    string
}

type awsCloudwatchMetricIdentifier struct {
	metadataTestData *parseAWSCloudwatchMetadataTestData
	triggerIndex     int
	name             string
}

var testAWSCloudwatchMetadata = []parseAWSCloudwatchMetadataTestData{
	{map[string]string{}, testAWSAuthentication, true, "Empty structures"},
	// properly formed cloudwatch query and awsRegion
	{
		map[string]string{
			"namespace":                   "AWS/SQS",
			"dimensionName":               "QueueName",
			"dimensionValue":              "keda",
			"metricName":                  "ApproximateNumberOfMessagesVisible",
			"targetMetricValue":           "2",
			"activationTargetMetricValue": "0",
			"minMetricValue":              "0",
			"awsRegion":                   "eu-west-1",
		},
		testAWSAuthentication,
		false,
		"properly formed cloudwatch query and awsRegion",
	},
	// properly formed cloudwatch expression query and awsRegion
	{
		map[string]string{
			"namespace":                   "AWS/SQS",
			"expression":                  "SELECT MIN(MessageCount) FROM \"AWS/AmazonMQ\" WHERE Broker = 'production' and Queue = 'worker'",
			"metricName":                  "ApproximateNumberOfMessagesVisible",
			"targetMetricValue":           "2",
			"activationTargetMetricValue": "0",
			"minMetricValue":              "0",
			"awsRegion":                   "eu-west-1",
		},
		testAWSAuthentication,
		false,
		"properly formed cloudwatch expression query and awsRegion",
	},
	// Properly formed cloudwatch query with optional parameters
	{
		map[string]string{
			"namespace":                   "AWS/SQS",
			"dimensionName":               "QueueName",
			"dimensionValue":              "keda",
			"metricName":                  "ApproximateNumberOfMessagesVisible",
			"targetMetricValue":           "2",
			"activationTargetMetricValue": "0",
			"minMetricValue":              "0",
			"metricCollectionTime":        "300",
			"metricStat":                  "Average",
			"metricStatPeriod":            "300",
			"awsRegion":                   "eu-west-1",
			"awsEndpoint":                 "http://localhost:4566",
		},
		testAWSAuthentication, false,
		"Properly formed cloudwatch query with optional parameters",
	},
	// properly formed cloudwatch query but Region is empty
	{
		map[string]string{
			"namespace":                   "AWS/SQS",
			"dimensionName":               "QueueName",
			"dimensionValue":              "keda",
			"metricName":                  "ApproximateNumberOfMessagesVisible",
			"targetMetricValue":           "2",
			"activationTargetMetricValue": "0",
			"minMetricValue":              "0",
			"awsRegion":                   "",
		},
		testAWSAuthentication,
		true,
		"properly formed cloudwatch query but Region is empty",
	},
	// Missing namespace
	{
		map[string]string{
			"dimensionName":     "QueueName",
			"dimensionValue":    "keda",
			"metricName":        "ApproximateNumberOfMessagesVisible",
			"targetMetricValue": "2",
			"minMetricValue":    "0",
			"awsRegion":         "eu-west-1",
		},
		testAWSAuthentication,
		true,
		"Missing namespace",
	},
	// Missing dimensionName
	{
		map[string]string{
			"namespace":         "AWS/SQS",
			"dimensionValue":    "keda",
			"metricName":        "ApproximateNumberOfMessagesVisible",
			"targetMetricValue": "2",
			"minMetricValue":    "0",
			"awsRegion":         "eu-west-1",
		},
		testAWSAuthentication,
		true,
		"Missing dimensionName",
	},
	// Missing dimensionValue
	{
		map[string]string{
			"namespace":         "AWS/SQS",
			"dimensionName":     "QueueName",
			"metricName":        "ApproximateNumberOfMessagesVisible",
			"targetMetricValue": "2",
			"minMetricValue":    "0",
			"awsRegion":         "eu-west-1",
		},
		testAWSAuthentication,
		true,
		"Missing dimensionValue",
	},
	// Missing metricName
	{
		map[string]string{
			"namespace":         "AWS/SQS",
			"dimensionName":     "QueueName",
			"dimensionValue":    "keda",
			"targetMetricValue": "2",
			"minMetricValue":    "0",
			"awsRegion":         "eu-west-1",
		},
		testAWSAuthentication,
		true,
		"Missing metricName",
	},
	// with static "aws_credentials" from TriggerAuthentication
	{
		map[string]string{
			"namespace":            "AWS/SQS",
			"dimensionName":        "QueueName",
			"dimensionValue":       "keda",
			"metricName":           "ApproximateNumberOfMessagesVisible",
			"targetMetricValue":    "2",
			"minMetricValue":       "0",
			"metricCollectionTime": "300",
			"metricStat":           "Average",
			"metricStatPeriod":     "300",
			"awsRegion":            "eu-west-1",
		},
		map[string]string{
			"awsAccessKeyId":     testAWSCloudwatchAccessKeyID,
			"awsSecretAccessKey": testAWSCloudwatchSecretAccessKey,
		},
		false,
		"with AWS Credentials from TriggerAuthentication",
	},
	// with temporary "aws_credentials" from TriggerAuthentication
	{
		map[string]string{
			"namespace":            "AWS/SQS",
			"dimensionName":        "QueueName",
			"dimensionValue":       "keda",
			"metricName":           "ApproximateNumberOfMessagesVisible",
			"targetMetricValue":    "2",
			"minMetricValue":       "0",
			"metricCollectionTime": "300",
			"metricStat":           "Average",
			"metricStatPeriod":     "300",
			"awsRegion":            "eu-west-1",
		},
		map[string]string{
			"awsAccessKeyId":     testAWSCloudwatchAccessKeyID,
			"awsSecretAccessKey": testAWSCloudwatchSecretAccessKey,
			"awsSessionToken":    testAWSCloudwatchSessionToken,
		},
		false,
		"with AWS Credentials from TriggerAuthentication",
	},
	// with "aws_role" from TriggerAuthentication
	{
		map[string]string{
			"namespace":            "AWS/SQS",
			"dimensionName":        "QueueName",
			"dimensionValue":       "keda",
			"metricName":           "ApproximateNumberOfMessagesVisible",
			"targetMetricValue":    "2",
			"minMetricValue":       "0",
			"metricCollectionTime": "300",
			"metricStat":           "Average",
			"metricStatPeriod":     "300",
			"awsRegion":            "eu-west-1",
		},
		map[string]string{
			"awsRoleArn": testAWSCloudwatchRoleArn,
		},
		false,
		"with AWS Role from TriggerAuthentication",
	},
	{
		map[string]string{
			"namespace":            "AWS/SQS",
			"dimensionName":        "QueueName",
			"dimensionValue":       "keda",
			"metricName":           "ApproximateNumberOfMessagesVisible",
			"targetMetricValue":    "2",
			"minMetricValue":       "0",
			"metricCollectionTime": "300",
			"metricStat":           "Average",
			"metricStatPeriod":     "300",
			"awsRegion":            "eu-west-1",
			"identityOwner":        "operator",
		},
		map[string]string{},
		false,
		"with AWS Role assigned on KEDA operator itself",
	},
	{
		map[string]string{
			"namespace":            "AWS/SQS",
			"dimensionName":        "QueueName",
			"dimensionValue":       "keda",
			"metricName":           "ApproximateNumberOfMessagesVisible",
			"targetMetricValue":    "2",
			"minMetricValue":       "0",
			"metricCollectionTime": "a",
			"metricStat":           "Average",
			"metricStatPeriod":     "300",
			"awsRegion":            "eu-west-1",
			"identityOwner":        "operator",
		},
		map[string]string{},
		true,
		"if metricCollectionTime assigned with a string, need to be a number",
	},
	{
		map[string]string{
			"namespace":            "AWS/SQS",
			"dimensionName":        "QueueName",
			"dimensionValue":       "keda",
			"metricName":           "ApproximateNumberOfMessagesVisible",
			"targetMetricValue":    "2",
			"minMetricValue":       "0",
			"metricCollectionTime": "300",
			"metricStat":           "Average",
			"metricStatPeriod":     "a",
			"awsRegion":            "eu-west-1",
			"identityOwner":        "operator",
		},
		map[string]string{},
		true,
		"if metricStatPeriod assigned with a string, need to be a number",
	},
	{
		map[string]string{
			"namespace":         "AWS/SQS",
			"dimensionName":     "QueueName",
			"dimensionValue":    "keda",
			"metricName":        "ApproximateNumberOfMessagesVisible",
			"targetMetricValue": "2",
			"minMetricValue":    "0",
			"metricStat":        "Average",
			"metricStatPeriod":  "300",
			"awsRegion":         "eu-west-1",
		},
		testAWSAuthentication, false,
		"Missing metricCollectionTime not generate error because will get the default value",
	},
	{
		map[string]string{
			"namespace":            "AWS/SQS",
			"dimensionName":        "QueueName",
			"dimensionValue":       "keda",
			"metricName":           "ApproximateNumberOfMessagesVisible",
			"targetMetricValue":    "2",
			"minMetricValue":       "0",
			"metricCollectionTime": "300",
			"metricStatPeriod":     "300",
			"awsRegion":            "eu-west-1",
		},
		testAWSAuthentication, false,
		"Missing metricStat not generate error because will get the default value",
	},
	{
		map[string]string{
			"namespace":            "AWS/SQS",
			"dimensionName":        "QueueName",
			"dimensionValue":       "keda",
			"metricName":           "ApproximateNumberOfMessagesVisible",
			"targetMetricValue":    "2",
			"minMetricValue":       "0",
			"metricCollectionTime": "300",
			"metricStat":           "Average",
			"awsRegion":            "eu-west-1",
		},
		testAWSAuthentication, false,
		"Missing metricStatPeriod not generate error because will get the default value",
	},
	{
		map[string]string{
			"namespace":           "AWS/SQS",
			"dimensionName":       "QueueName",
			"dimensionValue":      "keda",
			"metricName":          "ApproximateNumberOfMessagesVisible",
			"targetMetricValue":   "2",
			"minMetricValue":      "0",
			"metricStat":          "Average",
			"metricUnit":          "Count",
			"metricEndTimeOffset": "60",
			"awsRegion":           "eu-west-1",
		},
		testAWSAuthentication, false,
		"set a supported metricUnit",
	},
	{
		map[string]string{
			"namespace":            "AWS/SQS",
			"dimensionName":        "QueueName",
			"dimensionValue":       "keda",
			"metricName":           "ApproximateNumberOfMessagesVisible",
			"targetMetricValue":    "2",
			"minMetricValue":       "0",
			"metricCollectionTime": "300",
			"metricStat":           "SomeStat",
			"awsRegion":            "eu-west-1",
		},
		testAWSAuthentication, true,
		"metricStat is not supported",
	},
	{
		map[string]string{
			"namespace":            "AWS/SQS",
			"dimensionName":        "QueueName",
			"dimensionValue":       "keda",
			"metricName":           "ApproximateNumberOfMessagesVisible",
			"targetMetricValue":    "2",
			"minMetricValue":       "0",
			"metricStatPeriod":     "300",
			"metricCollectionTime": "100",
			"metricStat":           "Average",
			"awsRegion":            "eu-west-1",
		},
		testAWSAuthentication, true,
		"metricCollectionTime smaller than metricStatPeriod",
	},
	{
		map[string]string{
			"namespace":         "AWS/SQS",
			"dimensionName":     "QueueName",
			"dimensionValue":    "keda",
			"metricName":        "ApproximateNumberOfMessagesVisible",
			"targetMetricValue": "2",
			"minMetricValue":    "0",
			"metricStatPeriod":  "250",
			"metricStat":        "Average",
			"awsRegion":         "eu-west-1",
		},
		testAWSAuthentication, true,
		"unsupported metricStatPeriod",
	},
	{
		map[string]string{
			"namespace":         "AWS/SQS",
			"dimensionName":     "QueueName",
			"dimensionValue":    "keda",
			"metricName":        "ApproximateNumberOfMessagesVisible",
			"targetMetricValue": "2",
			"minMetricValue":    "0",
			"metricStatPeriod":  "25",
			"metricStat":        "Average",
			"awsRegion":         "eu-west-1",
		},
		testAWSAuthentication, true,
		"unsupported metricStatPeriod",
	},
	{
		map[string]string{
			"namespace":         "AWS/SQS",
			"dimensionName":     "QueueName",
			"dimensionValue":    "keda",
			"metricName":        "ApproximateNumberOfMessagesVisible",
			"targetMetricValue": "2",
			"minMetricValue":    "0",
			"metricStatPeriod":  "25",
			"metricStat":        "Average",
			"metricUnit":        "Hour",
			"awsRegion":         "eu-west-1",
		},
		testAWSAuthentication, true,
		"unsupported metricUnit",
	},
	// test ignoreNullValues is false
	{
		map[string]string{
			"namespace":         "AWS/SQS",
			"dimensionName":     "QueueName",
			"dimensionValue":    "keda",
			"metricName":        "ApproximateNumberOfMessagesVisible",
			"targetMetricValue": "2",
			"minMetricValue":    "0",
			"metricStatPeriod":  "60",
			"metricStat":        "Average",
			"ignoreNullValues":  "false",
			"awsRegion":         "eu-west-1",
		},
		testAWSAuthentication, false,
		"with ignoreNullValues set to false",
	},
	// test ignoreNullValues is true
	{
		map[string]string{
			"namespace":         "AWS/SQS",
			"dimensionName":     "QueueName",
			"dimensionValue":    "keda",
			"metricName":        "ApproximateNumberOfMessagesVisible",
			"targetMetricValue": "2",
			"minMetricValue":    "0",
			"metricStatPeriod":  "60",
			"metricStat":        "Average",
			"ignoreNullValues":  "true",
			"awsRegion":         "eu-west-1",
		},
		testAWSAuthentication, false,
		"with ignoreNullValues set to true",
	},
	// test ignoreNullValues is incorrect
	{
		map[string]string{
			"namespace":         "AWS/SQS",
			"dimensionName":     "QueueName",
			"dimensionValue":    "keda",
			"metricName":        "ApproximateNumberOfMessagesVisible",
			"targetMetricValue": "2",
			"minMetricValue":    "0",
			"metricStatPeriod":  "60",
			"metricStat":        "Average",
			"ignoreNullValues":  "maybe",
			"awsRegion":         "eu-west-1",
		},
		testAWSAuthentication, true,
		"unsupported value for ignoreNullValues",
	},
	// test case for multiple dimensions with valid separator
	{
		map[string]string{
			"namespace":         "AWS/SQS",
			"dimensionName":     "QueueName;Region",
			"dimensionValue":    "queue1;us-west-2",
			"metricName":        "ApproximateNumberOfMessagesVisible",
			"targetMetricValue": "5",
			"minMetricValue":    "1",
			"metricStat":        "Average",
			"awsRegion":         "us-west-2",
		},
		testAWSAuthentication,
		false,
		"Multiple dimensions with valid separator",
	},
}

var awsCloudwatchMetricIdentifiers = []awsCloudwatchMetricIdentifier{
	{&testAWSCloudwatchMetadata[1], 0, "s0-aws-cloudwatch"},
	{&testAWSCloudwatchMetadata[1], 3, "s3-aws-cloudwatch"},
	{&testAWSCloudwatchMetadata[2], 5, "s5-aws-cloudwatch"},
}

var awsCloudwatchGetMetricTestData = []awsCloudwatchMetadata{
	{
		Namespace:            "Custom",
		MetricsName:          "HasData",
		DimensionName:        []string{"DIM"},
		DimensionValue:       []string{"DIM_VALUE"},
		TargetMetricValue:    100,
		MinMetricValue:       0,
		MetricCollectionTime: 60,
		MetricStat:           "Average",
		MetricUnit:           "SampleCount",
		MetricStatPeriod:     60,
		MetricEndTimeOffset:  60,
		AwsRegion:            "us-west-2",
		awsAuthorization:     awsutils.AuthorizationMetadata{PodIdentityOwner: false},
		triggerIndex:         0,
	},
	{
		Namespace:            "Custom",
		MetricsName:          "HasDataNoUnit",
		DimensionName:        []string{"DIM"},
		DimensionValue:       []string{"DIM_VALUE"},
		TargetMetricValue:    100,
		MinMetricValue:       0,
		MetricCollectionTime: 60,
		MetricStat:           "Average",
		MetricUnit:           "",
		MetricStatPeriod:     60,
		MetricEndTimeOffset:  60,
		AwsRegion:            "us-west-2",
		awsAuthorization:     awsutils.AuthorizationMetadata{PodIdentityOwner: false},
		triggerIndex:         0,
	},
	{
		Namespace:            "Custom",
		MetricsName:          "HasDataFromExpression",
		Expression:           "SELECT MIN(MessageCount) FROM \"AWS/AmazonMQ\" WHERE Broker = 'production' and Queue = 'worker'",
		TargetMetricValue:    100,
		MinMetricValue:       0,
		MetricCollectionTime: 60,
		MetricStat:           "Average",
		MetricUnit:           "SampleCount",
		MetricStatPeriod:     60,
		MetricEndTimeOffset:  60,
		AwsRegion:            "us-west-2",
		awsAuthorization:     awsutils.AuthorizationMetadata{PodIdentityOwner: false},
		triggerIndex:         0,
	},
	// Test for metric with no data, no error expected as we are ignoring null values
	{
		Namespace:            "Custom",
		MetricsName:          testAWSCloudwatchErrorMetric,
		DimensionName:        []string{"DIM"},
		DimensionValue:       []string{"DIM_VALUE"},
		TargetMetricValue:    100,
		MinMetricValue:       0,
		MetricCollectionTime: 60,
		MetricStat:           "Average",
		MetricUnit:           "",
		MetricStatPeriod:     60,
		MetricEndTimeOffset:  60,
		AwsRegion:            "us-west-2",
		awsAuthorization:     awsutils.AuthorizationMetadata{PodIdentityOwner: false},
		triggerIndex:         0,
	},
	// Test for metric with no data, and the scaler errors when metric data values are empty
	{
		Namespace:            "Custom",
		MetricsName:          testAWSCloudwatchNoValueMetric,
		DimensionName:        []string{"DIM"},
		DimensionValue:       []string{"DIM_VALUE"},
		TargetMetricValue:    100,
		MinMetricValue:       0,
		MetricCollectionTime: 60,
		MetricStat:           "Average",
		MetricUnit:           "",
		MetricStatPeriod:     60,
		MetricEndTimeOffset:  60,
		AwsRegion:            "us-west-2",
		awsAuthorization:     awsutils.AuthorizationMetadata{PodIdentityOwner: false},
		triggerIndex:         0,
	},
	{
		Namespace:            "Custom",
		MetricsName:          "HasDataFromExpression",
		Expression:           "SELECT MIN(MessageCount) FROM \"AWS/AmazonMQ\" WHERE Broker = 'production' and Queue = 'worker'",
		TargetMetricValue:    100,
		MinMetricValue:       0,
		MetricCollectionTime: 60,
		MetricStat:           "Average",
		MetricUnit:           "SampleCount",
		MetricStatPeriod:     60,
		MetricEndTimeOffset:  60,
		AwsRegion:            "us-west-2",
		awsAuthorization:     awsutils.AuthorizationMetadata{PodIdentityOwner: false},
		triggerIndex:         0,
	},
}

type mockCloudwatch struct {
}

func (m *mockCloudwatch) GetMetricData(_ context.Context, input *cloudwatch.GetMetricDataInput, _ ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error) {
	if input.MetricDataQueries[0].MetricStat != nil {
		switch *input.MetricDataQueries[0].MetricStat.Metric.MetricName {
		case testAWSCloudwatchErrorMetric:
			return nil, errors.New("error")
		case testAWSCloudwatchNoValueMetric:
			return &cloudwatch.GetMetricDataOutput{
				MetricDataResults: []types.MetricDataResult{},
			}, nil
		case testAWSCloudwatchEmptyValues:
			return &cloudwatch.GetMetricDataOutput{
				MetricDataResults: []types.MetricDataResult{
					{
						Values: []float64{},
					},
				},
			}, nil
		}
	}

	return &cloudwatch.GetMetricDataOutput{
		MetricDataResults: []types.MetricDataResult{
			{
				Values: []float64{10},
			},
		},
	}, nil
}

func TestCloudwatchParseMetadata(t *testing.T) {
	for _, testData := range testAWSCloudwatchMetadata {
		_, err := parseAwsCloudwatchMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, ResolvedEnv: testAWSCloudwatchResolvedEnv, AuthParams: testData.authParams})
		if err != nil && !testData.isError {
			t.Errorf("%s: Expected success but got error %s", testData.comment, err)
		}
		if testData.isError && err == nil {
			t.Errorf("%s: Expected error but got success", testData.comment)
		}
	}
}

func TestAWSCloudwatchScalerMultipleDimensions(t *testing.T) {
	meta := map[string]string{
		"namespace":         "AWS/SQS",
		"dimensionName":     "QueueName;Region",
		"dimensionValue":    "queue1;us-west-2",
		"metricName":        "ApproximateNumberOfMessagesVisible",
		"targetMetricValue": "5",
		"minMetricValue":    "1",
		"metricStat":        "Average",
		"awsRegion":         "us-west-2",
	}

	config := &scalersconfig.ScalerConfig{
		TriggerMetadata: meta,
		ResolvedEnv:     testAWSCloudwatchResolvedEnv,
		AuthParams:      testAWSAuthentication,
	}

	awsMeta, err := parseAwsCloudwatchMetadata(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if assert.Equal(t, 2, len(awsMeta.DimensionName), "Expected two dimension names") &&
		assert.Equal(t, 2, len(awsMeta.DimensionValue), "Expected two dimension values") {
		assert.Equal(t, "QueueName", awsMeta.DimensionName[0], "First dimension name should be QueueName")
		assert.Equal(t, "Region", awsMeta.DimensionName[1], "Second dimension name should be Region")
		assert.Equal(t, "queue1", awsMeta.DimensionValue[0], "First dimension value should be queue1")
		assert.Equal(t, "us-west-2", awsMeta.DimensionValue[1], "Second dimension value should be us-west-2")
	}
}

func TestAWSCloudwatchGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range awsCloudwatchMetricIdentifiers {
		ctx := context.Background()
		meta, err := parseAwsCloudwatchMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, ResolvedEnv: testAWSCloudwatchResolvedEnv, AuthParams: testData.metadataTestData.authParams, TriggerIndex: testData.triggerIndex})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockAWSCloudwatchScaler := awsCloudwatchScaler{"", meta, &mockCloudwatch{}, logr.Discard()}

		metricSpec := mockAWSCloudwatchScaler.GetMetricSpecForScaling(ctx)
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}

func TestAWSCloudwatchScalerGetMetrics(t *testing.T) {
	for _, meta := range awsCloudwatchGetMetricTestData {
		mockAWSCloudwatchScaler := awsCloudwatchScaler{"", &meta, &mockCloudwatch{}, logr.Discard()}
		value, _, err := mockAWSCloudwatchScaler.GetMetricsAndActivity(context.Background(), meta.MetricsName)
		switch meta.MetricsName {
		case testAWSCloudwatchErrorMetric:
			assert.Error(t, err, "expect error because of cloudwatch api error")
		case testAWSCloudwatchNoValueMetric:
			assert.NoError(t, err, "dont expect error when returning empty metric list from cloudwatch")
		case testAWSCloudwatchEmptyValues:
			if meta.IgnoreNullValues {
				assert.NoError(t, err, "dont expect error when returning empty metric list from cloudwatch")
			} else {
				assert.Error(t, err, "expect error when returning empty metric list from cloudwatch, because ignoreNullValues is false")
			}
		default:
			assert.EqualValues(t, int64(10.0), value[0].Value.Value())
		}
	}
}

type computeQueryWindowTestArgs struct {
	name                    string
	current                 string
	metricPeriodSec         int64
	metricEndTimeOffsetSec  int64
	metricCollectionTimeSec int64
	expectedStartTime       string
	expectedEndTime         string
}

var awsCloudwatchComputeQueryWindowTestData = []computeQueryWindowTestArgs{
	{
		name:                    "normal",
		current:                 "2021-11-07T15:04:05.999Z",
		metricPeriodSec:         60,
		metricEndTimeOffsetSec:  0,
		metricCollectionTimeSec: 60,
		expectedStartTime:       "2021-11-07T15:03:00Z",
		expectedEndTime:         "2021-11-07T15:04:00Z",
	},
	{
		name:                    "normal with offset",
		current:                 "2021-11-07T15:04:05.999Z",
		metricPeriodSec:         60,
		metricEndTimeOffsetSec:  30,
		metricCollectionTimeSec: 60,
		expectedStartTime:       "2021-11-07T15:02:00Z",
		expectedEndTime:         "2021-11-07T15:03:00Z",
	},
}

func TestComputeQueryWindow(t *testing.T) {
	for _, testData := range awsCloudwatchComputeQueryWindowTestData {
		current, err := time.Parse(time.RFC3339Nano, testData.current)
		if err != nil {
			t.Errorf("unexpected input datetime format: %v", err)
		}
		startTime, endTime := computeQueryWindow(current, testData.metricPeriodSec, testData.metricEndTimeOffsetSec, testData.metricCollectionTimeSec)
		assert.Equal(t, testData.expectedStartTime, startTime.UTC().Format(time.RFC3339Nano), "unexpected startTime", "name", testData.name)
		assert.Equal(t, testData.expectedEndTime, endTime.UTC().Format(time.RFC3339Nano), "unexpected endTime", "name", testData.name)
	}
}
