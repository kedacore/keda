package scalers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/go-logr/logr"
	"go.mongodb.org/mongo-driver/bson"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	awsutils "github.com/kedacore/keda/v2/pkg/scalers/aws"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type awsDynamoDBScaler struct {
	metricType v2.MetricTargetType
	metadata   *awsDynamoDBMetadata
	dbClient   dynamodb.QueryAPIClient
	logger     logr.Logger
}

type awsDynamoDBMetadata struct {
	tableName                 string
	awsRegion                 string
	awsEndpoint               string
	keyConditionExpression    string
	expressionAttributeNames  map[string]string
	expressionAttributeValues map[string]types.AttributeValue
	indexName                 string
	targetValue               int64
	activationTargetValue     int64
	awsAuthorization          awsutils.AuthorizationMetadata
	scalerIndex               int
	metricName                string
}

func NewAwsDynamoDBScaler(ctx context.Context, config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, err := parseAwsDynamoDBMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing DynamoDb metadata: %w", err)
	}
	dbClient, err := createDynamoDBClient(ctx, meta)
	if err != nil {
		return nil, fmt.Errorf("error when creating dynamodb client: %w", err)
	}
	return &awsDynamoDBScaler{
		metricType: metricType,
		metadata:   meta,
		dbClient:   dbClient,
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
		if config.AsMetricSource {
			meta.targetValue = 0
		} else {
			return nil, ErrAwsDynamoNoTargetValue
		}
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

	auth, err := awsutils.GetAwsAuthorization(config.ScalerUniqueKey, config.PodIdentity, config.TriggerMetadata, config.AuthParams, config.ResolvedEnv)
	if err != nil {
		return nil, err
	}

	meta.awsAuthorization = auth
	meta.scalerIndex = config.ScalerIndex

	meta.metricName = GenerateMetricNameWithIndex(config.ScalerIndex,
		kedautil.NormalizeString(fmt.Sprintf("aws-dynamodb-%s", meta.tableName)))

	return &meta, nil
}

func createDynamoDBClient(ctx context.Context, metadata *awsDynamoDBMetadata) (*dynamodb.Client, error) {
	cfg, err := awsutils.GetAwsConfig(ctx, metadata.awsRegion, metadata.awsAuthorization)
	if err != nil {
		return nil, err
	}

	return dynamodb.NewFromConfig(*cfg, func(options *dynamodb.Options) {
		if metadata.awsEndpoint != "" {
			options.BaseEndpoint = aws.String(metadata.awsEndpoint)
		}
	}), nil
}

func (s *awsDynamoDBScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	metricValue, err := s.GetQueryMetrics(ctx)
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
	awsutils.ClearAwsConfig(s.metadata.awsAuthorization)
	return nil
}

func (s *awsDynamoDBScaler) GetQueryMetrics(ctx context.Context) (float64, error) {
	dimensions := dynamodb.QueryInput{
		TableName:                 aws.String(s.metadata.tableName),
		KeyConditionExpression:    aws.String(s.metadata.keyConditionExpression),
		ExpressionAttributeNames:  s.metadata.expressionAttributeNames,
		ExpressionAttributeValues: s.metadata.expressionAttributeValues,
	}

	if s.metadata.indexName != "" {
		dimensions.IndexName = aws.String(s.metadata.indexName)
	}

	res, err := s.dbClient.Query(ctx, &dimensions)
	if err != nil {
		s.logger.Error(err, "Failed to get output")
		return 0, err
	}

	return float64(res.Count), nil
}

// json2Map convert Json to map[string]string
func json2Map(js string) (m map[string]string, err error) {
	err = bson.UnmarshalExtJSON([]byte(js), true, &m)
	if err != nil {
		return nil, fmt.Errorf("%v: %w", ErrAwsDynamoInvalidExpressionAttributeNames, err)
	}

	if len(m) == 0 {
		return nil, ErrAwsDynamoEmptyExpressionAttributeNames
	}
	return m, err
}

// json2DynamoMap converts Json to map[string]types.AttributeValue
func json2DynamoMap(js string) (map[string]types.AttributeValue, error) {
	var valueMap map[string]interface{}
	err := json.Unmarshal([]byte(js), &valueMap)
	if err != nil {
		return nil, err
	}
	attributeValues := make(map[string]types.AttributeValue)

	// Iterate through the input map and convert values to AttributeValues
	for k, v := range valueMap {
		av, err := attributeValueFromInterface(v)
		if err != nil {
			return nil, err
		}
		attributeValues[k] = av
	}
	return attributeValues, nil
}

func attributeValueFromInterface(value interface{}) (types.AttributeValue, error) {
	var err error
	switch v := value.(type) {
	case map[string]interface{}:
		// Check the nested map to determine the data type
		for dataType, val := range v {
			switch dataType {
			case "S":
				return &types.AttributeValueMemberS{Value: val.(string)}, nil
			case "N":
				switch av := val.(type) {
				case string:
					return &types.AttributeValueMemberN{Value: av}, nil
				default:
					return nil, ErrAwsDynamoInvalidExpressionAttributeValues
				}
			case "BOOL":
				return &types.AttributeValueMemberBOOL{Value: val.(bool)}, nil
			case "B":
				return &types.AttributeValueMemberB{Value: []byte(val.(string))}, nil
			case "L":
				listValues := val.([]interface{})
				list := make([]types.AttributeValue, len(listValues))
				for i, listVal := range listValues {
					list[i], err = attributeValueFromInterface(listVal)
					if err != nil {
						return nil, err
					}
				}
				return &types.AttributeValueMemberL{Value: list}, nil
			case "M":
				mapValues := val.(map[string]interface{})
				m := make(map[string]types.AttributeValue)
				for mapKey, mapVal := range mapValues {
					mapAttr, err := attributeValueFromInterface(mapVal)
					if err != nil {
						return nil, err
					}
					m[mapKey] = mapAttr
				}
				return &types.AttributeValueMemberM{Value: m}, nil
			case "NULL":
				return &types.AttributeValueMemberNULL{Value: true}, nil
			default:
				return nil, fmt.Errorf("unsupported data type for attribute value: %s", dataType)
			}
		}
	default:
		return nil, fmt.Errorf("unsupported data type for attribute value")
	}
	return nil, nil
}
