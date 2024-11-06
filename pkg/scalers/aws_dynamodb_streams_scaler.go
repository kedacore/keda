package scalers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodbstreams"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	awsutils "github.com/kedacore/keda/v2/pkg/scalers/aws"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	defaultTargetDBStreamsShardCount           = 2
	defaultActivationTargetDBStreamsShardCount = 0
)

type awsDynamoDBStreamsScaler struct {
	metricType            v2.MetricTargetType
	metadata              *awsDynamoDBStreamsMetadata
	streamArn             *string
	dbStreamWrapperClient DynamodbStreamWrapperClient
	logger                logr.Logger
}

type awsDynamoDBStreamsMetadata struct {
	targetShardCount           int64
	activationTargetShardCount int64
	tableName                  string
	awsRegion                  string
	awsEndpoint                string
	awsAuthorization           awsutils.AuthorizationMetadata
	triggerIndex               int
}

// NewAwsDynamoDBStreamsScaler creates a new awsDynamoDBStreamsScaler
func NewAwsDynamoDBStreamsScaler(ctx context.Context, config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "aws_dynamodb_streams_scaler")

	meta, err := parseAwsDynamoDBStreamsMetadata(config, logger)
	if err != nil {
		return nil, fmt.Errorf("error parsing dynamodb stream metadata: %w", err)
	}

	dbClient, dbStreamClient, err := createClientsForDynamoDBStreamsScaler(ctx, meta)
	if err != nil {
		return nil, fmt.Errorf("error when creating dynamodbstream client: %w", err)
	}
	streamArn, err := getDynamoDBStreamsArn(ctx, dbClient, &meta.tableName)
	if err != nil {
		return nil, fmt.Errorf("error dynamodb stream arn: %w", err)
	}

	return &awsDynamoDBStreamsScaler{
		metricType: metricType,
		metadata:   meta,
		streamArn:  streamArn,
		dbStreamWrapperClient: &dynamodbStreamWrapperClient{
			dbStreamClient: dbStreamClient,
		},
		logger: logger,
	}, nil
}

func parseAwsDynamoDBStreamsMetadata(config *scalersconfig.ScalerConfig, logger logr.Logger) (*awsDynamoDBStreamsMetadata, error) {
	meta := awsDynamoDBStreamsMetadata{}
	meta.targetShardCount = defaultTargetDBStreamsShardCount

	if val, ok := config.TriggerMetadata["awsRegion"]; ok && val != "" {
		meta.awsRegion = val
	} else {
		return nil, fmt.Errorf("no awsRegion given")
	}

	if val, ok := config.TriggerMetadata["awsEndpoint"]; ok {
		meta.awsEndpoint = val
	}

	if val, ok := config.TriggerMetadata["tableName"]; ok && val != "" {
		meta.tableName = val
	} else {
		return nil, fmt.Errorf("no tableName given")
	}

	if val, ok := config.TriggerMetadata["shardCount"]; ok && val != "" {
		shardCount, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			meta.targetShardCount = defaultTargetDBStreamsShardCount
			logger.Error(err, "error parsing dyanmodb stream metadata shardCount, using default %n", defaultTargetDBStreamsShardCount)
		} else {
			meta.targetShardCount = shardCount
		}
	}
	if val, ok := config.TriggerMetadata["activationShardCount"]; ok && val != "" {
		shardCount, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			meta.activationTargetShardCount = defaultActivationTargetDBStreamsShardCount
			logger.Error(err, "error parsing dyanmodb stream metadata activationTargetShardCount, using default %n", defaultActivationTargetDBStreamsShardCount)
		} else {
			meta.activationTargetShardCount = shardCount
		}
	}

	auth, err := awsutils.GetAwsAuthorization(config.TriggerUniqueKey, config.PodIdentity, config.TriggerMetadata, config.AuthParams, config.ResolvedEnv)
	if err != nil {
		return nil, err
	}

	meta.awsAuthorization = auth
	meta.triggerIndex = config.TriggerIndex

	return &meta, nil
}

func createClientsForDynamoDBStreamsScaler(ctx context.Context, metadata *awsDynamoDBStreamsMetadata) (*dynamodb.Client, *dynamodbstreams.Client, error) {
	cfg, err := awsutils.GetAwsConfig(ctx, metadata.awsAuthorization)
	if err != nil {
		return nil, nil, err
	}
	dbClient := dynamodb.NewFromConfig(*cfg, func(options *dynamodb.Options) {
		if metadata.awsEndpoint != "" {
			options.BaseEndpoint = aws.String(metadata.awsEndpoint)
		}
	})
	dbStreamClient := dynamodbstreams.NewFromConfig(*cfg, func(options *dynamodbstreams.Options) {
		if metadata.awsEndpoint != "" {
			options.BaseEndpoint = aws.String(metadata.awsEndpoint)
		}
	})

	return dbClient, dbStreamClient, nil
}

type DynamodbStreamWrapperClient interface {
	DescribeStream(ctx context.Context, params *dynamodbstreams.DescribeStreamInput, optFns ...func(*dynamodbstreams.Options)) (*dynamodbstreams.DescribeStreamOutput, error)
}

type dynamodbStreamWrapperClient struct {
	dbStreamClient *dynamodbstreams.Client
}

func (w dynamodbStreamWrapperClient) DescribeStream(ctx context.Context, params *dynamodbstreams.DescribeStreamInput, optFns ...func(*dynamodbstreams.Options)) (*dynamodbstreams.DescribeStreamOutput, error) {
	return w.dbStreamClient.DescribeStream(ctx, params, optFns...)
}

func getDynamoDBStreamsArn(ctx context.Context, db dynamodb.DescribeTableAPIClient, tableName *string) (*string, error) {
	tableOutput, err := db.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: tableName,
	})
	if err != nil {
		return nil, err
	}
	if tableOutput.Table.LatestStreamArn == nil {
		return nil, fmt.Errorf("dynamodb stream arn for the table %s is empty", *tableName)
	}
	return tableOutput.Table.LatestStreamArn, nil
}

func (s *awsDynamoDBStreamsScaler) Close(_ context.Context) error {
	awsutils.ClearAwsConfig(s.metadata.awsAuthorization)
	return nil
}

func (s *awsDynamoDBStreamsScaler) GetMetricSpecForScaling(_ context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("aws-dynamodb-streams-%s", s.metadata.tableName))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.targetShardCount),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *awsDynamoDBStreamsScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	shardCount, err := s.getDynamoDBStreamShardCount(ctx)

	if err != nil {
		s.logger.Error(err, "error getting shard count")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, float64(shardCount))

	return []external_metrics.ExternalMetricValue{metric}, shardCount > s.metadata.activationTargetShardCount, nil
}

// GetDynamoDBStreamShardCount Get DynamoDB Stream Shard Count
func (s *awsDynamoDBStreamsScaler) getDynamoDBStreamShardCount(ctx context.Context) (int64, error) {
	var shardNum int64
	var lastShardID *string

	input := dynamodbstreams.DescribeStreamInput{
		StreamArn: s.streamArn,
	}
	for {
		if lastShardID != nil {
			// The upper limit of shard num to retrun is 100.
			// ExclusiveStartShardId is the shard ID of the first item that the operation will evaluate.
			input = dynamodbstreams.DescribeStreamInput{
				StreamArn:             s.streamArn,
				ExclusiveStartShardId: lastShardID,
			}
		}
		des, err := s.dbStreamWrapperClient.DescribeStream(ctx, &input)
		if err != nil {
			return -1, err
		}
		shardNum += int64(len(des.StreamDescription.Shards))
		lastShardID = des.StreamDescription.LastEvaluatedShardId
		// If LastEvaluatedShardId is empty, then the "last page" of results has been
		// processed and there is currently no more data to be retrieved.
		if lastShardID == nil {
			break
		}
	}
	return shardNum, nil
}
