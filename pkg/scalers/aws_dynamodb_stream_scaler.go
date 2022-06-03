package scalers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/dynamodbstreams"
	"github.com/aws/aws-sdk-go/service/dynamodbstreams/dynamodbstreamsiface"
	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	defaultTargetDBStreamShardCount = 2
)

type awsDynamoDBStreamScaler struct {
	metricType     v2beta2.MetricTargetType
	metadata       *awsDynamoDBStreamMetadata
	streamArn      *string
	dbStreamClient dynamodbstreamsiface.DynamoDBStreamsAPI
}

type awsDynamoDBStreamMetadata struct {
	targetShardCount int64
	tableName        string
	awsRegion        string
	awsAuthorization awsAuthorizationMetadata
	scalerIndex      int
}

var dynamodbStreamLog = logf.Log.WithName("aws_dynamodb_stream_scaler")

// NewawsDynamoDBStreamScaler creates a new awsDynamoDBStreamScaler
func NewAwsDynamoDBStreamScaler(ctx context.Context, config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %s", err)
	}

	meta, err := parseAwsDynamoDBStreamMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing dynamodb stream metadata: %s", err)
	}

	dbClient, dbStreamClient := createClientsForDynamoDBStreamScaler(meta)

	streamArn, err := getDynamoDBStreamArn(ctx, dbClient, &meta.tableName)
	if err != nil {
		return nil, fmt.Errorf("error dynamodb stream arn: %s", err)
	}

	return &awsDynamoDBStreamScaler{
		metricType:     metricType,
		metadata:       meta,
		streamArn:      streamArn,
		dbStreamClient: dbStreamClient,
	}, nil
}

func parseAwsDynamoDBStreamMetadata(config *ScalerConfig) (*awsDynamoDBStreamMetadata, error) {
	meta := awsDynamoDBStreamMetadata{}
	meta.targetShardCount = defaultTargetDBStreamShardCount

	if val, ok := config.TriggerMetadata["awsRegion"]; ok && val != "" {
		meta.awsRegion = val
	} else {
		return nil, fmt.Errorf("no awsRegion given")
	}

	if val, ok := config.TriggerMetadata["tableName"]; ok && val != "" {
		meta.tableName = val
	} else {
		return nil, fmt.Errorf("no tableName given")
	}

	if val, ok := config.TriggerMetadata["shardCount"]; ok && val != "" {
		shardCount, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			meta.targetShardCount = defaultTargetDBStreamShardCount
			dynamodbStreamLog.Error(err, "error parsing dyanmodb stream metadata shardCount, using default %n", defaultTargetDBStreamShardCount)
		} else {
			meta.targetShardCount = shardCount
		}
	}

	auth, err := getAwsAuthorization(config.AuthParams, config.TriggerMetadata, config.ResolvedEnv)
	if err != nil {
		return nil, err
	}

	meta.awsAuthorization = auth
	meta.scalerIndex = config.ScalerIndex

	return &meta, nil
}

func createClientsForDynamoDBStreamScaler(metadata *awsDynamoDBStreamMetadata) (*dynamodb.DynamoDB, *dynamodbstreams.DynamoDBStreams) {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(metadata.awsRegion),
	}))

	var dbClient *dynamodb.DynamoDB
	var dbStreamClient *dynamodbstreams.DynamoDBStreams

	if metadata.awsAuthorization.podIdentityOwner {
		creds := credentials.NewStaticCredentials(metadata.awsAuthorization.awsAccessKeyID, metadata.awsAuthorization.awsSecretAccessKey, metadata.awsAuthorization.awsSessionToken)
		if metadata.awsAuthorization.awsRoleArn != "" {
			creds = stscreds.NewCredentials(sess, metadata.awsAuthorization.awsRoleArn)
		}
		dbClient = dynamodb.New(sess, &aws.Config{
			Region:      aws.String(metadata.awsRegion),
			Credentials: creds,
		})
		dbStreamClient = dynamodbstreams.New(sess, &aws.Config{
			Region:      aws.String(metadata.awsRegion),
			Credentials: creds,
		})
	} else {
		dbClient = dynamodb.New(sess, &aws.Config{
			Region: aws.String(metadata.awsRegion),
		})
		dbStreamClient = dynamodbstreams.New(sess, &aws.Config{
			Region: aws.String(metadata.awsRegion),
		})
	}
	return dbClient, dbStreamClient
}

func getDynamoDBStreamArn(ctx context.Context, db dynamodbiface.DynamoDBAPI, tableName *string) (*string, error) {
	tableOutput, err := db.DescribeTableWithContext(ctx, &dynamodb.DescribeTableInput{
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

// IsActive determines if we need to scale from zero
func (s *awsDynamoDBStreamScaler) IsActive(ctx context.Context) (bool, error) {
	count, err := s.GetDynamoDBStreamShardCount(ctx)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *awsDynamoDBStreamScaler) Close(context.Context) error {
	return nil
}

func (s *awsDynamoDBStreamScaler) GetMetricSpecForScaling(context.Context) []v2beta2.MetricSpec {
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, kedautil.NormalizeString(fmt.Sprintf("aws-dynamodb-stream-%s", s.metadata.tableName))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.targetShardCount),
	}
	metricSpec := v2beta2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2beta2.MetricSpec{metricSpec}
}

// GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *awsDynamoDBStreamScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	shardCount, err := s.GetDynamoDBStreamShardCount(ctx)

	if err != nil {
		dynamodbStreamLog.Error(err, "error getting shard count")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(shardCount, resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

// Get DynamoDB Stream Shard Count
func (s *awsDynamoDBStreamScaler) GetDynamoDBStreamShardCount(ctx context.Context) (int64, error) {
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
		des, err := s.dbStreamClient.DescribeStreamWithContext(ctx, &input)
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
