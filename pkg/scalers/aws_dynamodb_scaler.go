package scalers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/go-logr/logr"
	"go.mongodb.org/mongo-driver/bson"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type awsDynamoDBScaler struct {
	metricType v2.MetricTargetType
	metadata   *awsDynamoDBMetadata
	dbClient   dynamodbiface.DynamoDBAPI
	logger     logr.Logger
}

type awsDynamoDBMetadata struct {
	tableName                 string
	awsRegion                 string
	awsEndpoint               string
	keyConditionExpression    string
	expressionAttributeNames  map[string]*string
	expressionAttributeValues map[string]*dynamodb.AttributeValue
	indexName                 string
	targetValue               int64
	activationTargetValue     int64
	awsAuthorization          awsAuthorizationMetadata
	scalerIndex               int
	metricName                string
}

func NewAwsDynamoDBScaler(config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, err := parseAwsDynamoDBMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing DynamoDb metadata: %w", err)
	}

	return &awsDynamoDBScaler{
		metricType: metricType,
		metadata:   meta,
		dbClient:   createDynamoDBClient(meta),
		logger:     InitializeLogger(config, "aws_dynamodb_scaler"),
	}, nil
}

var (
	// ErrAwsDynamoNoTableName is returned when "tableName" is missing from the config.
	ErrAwsDynamoNoTableName = errors.New("no tableName given")

	// ErrAwsDynamoNoAwsRegion is returned when "awsRegion" is missing from the config.
	ErrAwsDynamoNoAwsRegion = errors.New("no awsRegion given")

	// ErrAwsDynamoNoKeyConditionExpression is returned when "keyConditionExpression" is missing from the config.
	ErrAwsDynamoNoKeyConditionExpression = errors.New("no keyConditionExpression given")

	// ErrAwsDynamoEmptyExpressionAttributeNames is returned when "expressionAttributeNames" is empty.
	ErrAwsDynamoEmptyExpressionAttributeNames = errors.New("empty map")

	// ErrAwsDynamoInvalidExpressionAttributeNames is returned when "expressionAttributeNames" is an invalid JSON.
	ErrAwsDynamoInvalidExpressionAttributeNames = errors.New("invalid expressionAttributeNames")

	// ErrAwsDynamoNoExpressionAttributeNames is returned when "expressionAttributeNames" is missing from the config.
	ErrAwsDynamoNoExpressionAttributeNames = errors.New("no expressionAttributeNames given")

	// ErrAwsDynamoInvalidExpressionAttributeValues is returned when "expressionAttributeNames" is missing an invalid JSON.
	ErrAwsDynamoInvalidExpressionAttributeValues = errors.New("invalid expressionAttributeValues")

	// ErrAwsDynamoNoExpressionAttributeValues is returned when "expressionAttributeValues" is missing from the config.
	ErrAwsDynamoNoExpressionAttributeValues = errors.New("no expressionAttributeValues given")

	// ErrAwsDynamoNoTargetValue is returned when "targetValue" is missing from the config.
	ErrAwsDynamoNoTargetValue = errors.New("no targetValue given")
)

func parseAwsDynamoDBMetadata(config *ScalerConfig) (*awsDynamoDBMetadata, error) {
	meta := awsDynamoDBMetadata{}

	if val, ok := config.TriggerMetadata["tableName"]; ok && val != "" {
		meta.tableName = val
	} else {
		return nil, ErrAwsDynamoNoTableName
	}

	if val, ok := config.TriggerMetadata["awsRegion"]; ok && val != "" {
		meta.awsRegion = val
	} else {
		return nil, ErrAwsDynamoNoAwsRegion
	}

	if val, ok := config.TriggerMetadata["awsEndpoint"]; ok {
		meta.awsEndpoint = val
	}

	if val, ok := config.TriggerMetadata["indexName"]; ok {
		meta.indexName = val
	}

	if val, ok := config.TriggerMetadata["keyConditionExpression"]; ok && val != "" {
		meta.keyConditionExpression = val
	} else {
		return nil, ErrAwsDynamoNoKeyConditionExpression
	}

	if val, ok := config.TriggerMetadata["expressionAttributeNames"]; ok && val != "" {
		names, err := json2Map(val)

		if err != nil {
			return nil, fmt.Errorf("error parsing expressionAttributeNames: %w", err)
		}

		meta.expressionAttributeNames = names
	} else {
		return nil, ErrAwsDynamoNoExpressionAttributeNames
	}

	if val, ok := config.TriggerMetadata["expressionAttributeValues"]; ok && val != "" {
		values, err := json2DynamoMap(val)

		if err != nil {
			return nil, fmt.Errorf("error parsing expressionAttributeValues: %w", err)
		}

		meta.expressionAttributeValues = values
	} else {
		return nil, ErrAwsDynamoNoExpressionAttributeValues
	}

	if val, ok := config.TriggerMetadata["targetValue"]; ok && val != "" {
		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing metadata targetValue: %w", err)
		}

		meta.targetValue = n
	} else {
		return nil, ErrAwsDynamoNoTargetValue
	}

	if val, ok := config.TriggerMetadata["activationTargetValue"]; ok && val != "" {
		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing metadata activationTargetValue: %w", err)
		}

		meta.activationTargetValue = n
	} else {
		meta.activationTargetValue = 0
	}

	auth, err := getAwsAuthorization(config.AuthParams, config.TriggerMetadata, config.ResolvedEnv)
	if err != nil {
		return nil, err
	}

	meta.awsAuthorization = auth
	meta.scalerIndex = config.ScalerIndex

	meta.metricName = GenerateMetricNameWithIndex(config.ScalerIndex,
		kedautil.NormalizeString(fmt.Sprintf("aws-dynamodb-%s", meta.tableName)))

	return &meta, nil
}

func createDynamoDBClient(metadata *awsDynamoDBMetadata) *dynamodb.DynamoDB {
	sess, config := getAwsConfig(metadata.awsRegion,
		metadata.awsEndpoint,
		metadata.awsAuthorization)

	return dynamodb.New(sess, config)
}

func (s *awsDynamoDBScaler) GetMetricsAndActivity(_ context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	metricValue, err := s.GetQueryMetrics()
	if err != nil {
		s.logger.Error(err, "Error getting metric value")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, metricValue)

	return []external_metrics.ExternalMetricValue{metric}, metricValue > float64(s.metadata.activationTargetValue), nil
}

func (s *awsDynamoDBScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: s.metadata.metricName,
		},
		Target: GetMetricTarget(s.metricType, s.metadata.targetValue),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}

	return []v2.MetricSpec{
		metricSpec,
	}
}

func (s *awsDynamoDBScaler) Close(context.Context) error {
	return nil
}

func (s *awsDynamoDBScaler) GetQueryMetrics() (float64, error) {
	dimensions := dynamodb.QueryInput{
		TableName:                 aws.String(s.metadata.tableName),
		KeyConditionExpression:    aws.String(s.metadata.keyConditionExpression),
		ExpressionAttributeNames:  s.metadata.expressionAttributeNames,
		ExpressionAttributeValues: s.metadata.expressionAttributeValues,
	}

	if s.metadata.indexName != "" {
		dimensions.IndexName = aws.String(s.metadata.indexName)
	}

	res, err := s.dbClient.Query(&dimensions)
	if err != nil {
		s.logger.Error(err, "Failed to get output")
		return 0, err
	}

	return float64(*res.Count), nil
}

// json2Map convert Json to map[string]string
func json2Map(js string) (m map[string]*string, err error) {
	err = bson.UnmarshalExtJSON([]byte(js), true, &m)
	if err != nil {
		return nil, fmt.Errorf("%v: %w", ErrAwsDynamoInvalidExpressionAttributeNames, err)
	}

	if len(m) == 0 {
		return nil, ErrAwsDynamoEmptyExpressionAttributeNames
	}
	return m, err
}

// json2DynamoMap converts Json to map[string]*dynamoDb.AttributeValue
func json2DynamoMap(js string) (m map[string]*dynamodb.AttributeValue, err error) {
	err = json.Unmarshal([]byte(js), &m)

	if err != nil {
		return nil, fmt.Errorf("%v: %w", ErrAwsDynamoInvalidExpressionAttributeValues, err)
	}
	return m, err
}
