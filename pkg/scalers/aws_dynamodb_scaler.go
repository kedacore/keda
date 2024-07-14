package scalers

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	awsutils "github.com/kedacore/keda/v2/pkg/scalers/aws"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type awsDynamoDBScaler struct {
	metricType v2.MetricTargetType
	metadata   *awsDynamoDBMetadata
	dbClient   dynamodb.QueryAPIClient
	logger     logr.Logger
}

type awsDynamoDBMetadata struct {
	awsAuthorization          awsutils.AuthorizationMetadata
	triggerIndex              int
	metricName                string
	TableName                 string                          `keda:"name=tableName, order=triggerMetadata"`
	AwsRegion                 string                          `keda:"name=awsRegion, order=triggerMetadata"`
	AwsEndpoint               string                          `keda:"name=awsEndpoint, order=triggerMetadata, optional"`
	KeyConditionExpression    string                          `keda:"name=keyConditionExpression, order=triggerMetadata"`
	ExpressionAttributeNames  map[string]string               `keda:"name=expressionAttributeNames, order=triggerMetadata"`
	ExpressionAttributeValues map[string]types.AttributeValue `keda:"name=expressionAttributeValues, order=triggerMetadata"`
	IndexName                 string                          `keda:"name=indexName, order=triggerMetadata, optional"`
	TargetValue               int64                           `keda:"name=targetValue, order=triggerMetadata, optional, default=-1"`
	ActivationTargetValue     int64                           `keda:"name=activationTargetValue, order=triggerMetadata, default=0"`
}

func NewAwsDynamoDBScaler(ctx context.Context, config *scalersconfig.ScalerConfig) (Scaler, error) {
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
	ErrAwsDynamoNoTargetValue = errors.New("no targetValue given")
)

func parseAwsDynamoDBMetadata(config *scalersconfig.ScalerConfig) (*awsDynamoDBMetadata, error) {
	meta := &awsDynamoDBMetadata{}
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing DynamoDb metadata: %w", err)
	}

	if meta.TargetValue == -1 && config.AsMetricSource {
		meta.TargetValue = 0
	} else if meta.TargetValue == -1 && !config.AsMetricSource {
		return nil, ErrAwsDynamoNoTargetValue
	}

	auth, err := awsutils.GetAwsAuthorization(config.TriggerUniqueKey, config.PodIdentity, config.TriggerMetadata, config.AuthParams, config.ResolvedEnv)
	if err != nil {
		return nil, err
	}

	meta.awsAuthorization = auth
	meta.triggerIndex = config.TriggerIndex

	meta.metricName = GenerateMetricNameWithIndex(config.TriggerIndex,
		kedautil.NormalizeString(fmt.Sprintf("aws-dynamodb-%s", meta.TableName)))

	return meta, nil
}

func createDynamoDBClient(ctx context.Context, metadata *awsDynamoDBMetadata) (*dynamodb.Client, error) {
	cfg, err := awsutils.GetAwsConfig(ctx, metadata.AwsRegion, metadata.awsAuthorization)
	if err != nil {
		return nil, err
	}

	return dynamodb.NewFromConfig(*cfg, func(options *dynamodb.Options) {
		if metadata.AwsEndpoint != "" {
			options.BaseEndpoint = aws.String(metadata.AwsEndpoint)
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

	return []external_metrics.ExternalMetricValue{metric}, metricValue > float64(s.metadata.ActivationTargetValue), nil
}

func (s *awsDynamoDBScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: s.metadata.metricName,
		},
		Target: GetMetricTarget(s.metricType, s.metadata.TargetValue),
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
		TableName:                 aws.String(s.metadata.TableName),
		KeyConditionExpression:    aws.String(s.metadata.KeyConditionExpression),
		ExpressionAttributeNames:  s.metadata.ExpressionAttributeNames,
		ExpressionAttributeValues: s.metadata.ExpressionAttributeValues,
	}

	if s.metadata.IndexName != "" {
		dimensions.IndexName = aws.String(s.metadata.IndexName)
	}

	res, err := s.dbClient.Query(ctx, &dimensions)
	if err != nil {
		s.logger.Error(err, "Failed to get output")
		return 0, err
	}

	return float64(res.Count), nil
}
