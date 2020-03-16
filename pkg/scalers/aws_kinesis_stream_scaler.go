package scalers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
	v2beta1 "k8s.io/api/autoscaling/v2beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	awsKinesisStreamMetricName = "ShardCount"
	targetShardCountDefault    = 2
)

type awsKinesisStreamScaler struct {
	metadata *awsKinesisStreamMetadata
}

type awsKinesisStreamMetadata struct {
	targetShardCount int
	streamName       string
	awsRegion        string
	awsAuthorization awsAuthorizationMetadata
}

var kinesisStreamLog = logf.Log.WithName("aws_kinesis_stream_scaler")

// NewAwsKinesisStreamScaler creates a new awsKinesisStreamScaler
func NewAwsKinesisStreamScaler(resolvedEnv, metadata map[string]string, authParams map[string]string) (Scaler, error) {
	meta, err := parseAwsKinesisStreamMetadata(metadata, resolvedEnv, authParams)
	if err != nil {
		return nil, fmt.Errorf("Error parsing Kinesis stream metadata: %s", err)
	}

	return &awsKinesisStreamScaler{
		metadata: meta,
	}, nil
}

func parseAwsKinesisStreamMetadata(metadata, resolvedEnv, authParams map[string]string) (*awsKinesisStreamMetadata, error) {
	meta := awsKinesisStreamMetadata{}
	meta.targetShardCount = targetShardCountDefault

	if val, ok := metadata["shardCount"]; ok && val != "" {
		shardCount, err := strconv.Atoi(val)
		if err != nil {
			meta.targetShardCount = targetShardCountDefault
			kinesisStreamLog.Error(err, "Error parsing Kinesis stream metadata shardCount, using default %n", targetShardCountDefault)
		} else {
			meta.targetShardCount = shardCount
		}
	}

	if val, ok := metadata["streamName"]; ok && val != "" {
		meta.streamName = val
	} else {
		return nil, fmt.Errorf("no streamName given")
	}

	if val, ok := metadata["awsRegion"]; ok && val != "" {
		meta.awsRegion = val
	} else {
		return nil, fmt.Errorf("no awsRegion given")
	}

	auth, err := getAwsAuthorization(authParams, metadata, resolvedEnv)
	if err != nil {
		return nil, err
	}

	meta.awsAuthorization = auth

	return &meta, nil
}

// IsActive determines if we need to scale from zero
func (s *awsKinesisStreamScaler) IsActive(ctx context.Context) (bool, error) {
	count, err := s.GetAwsKinesisOpenShardCount()

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (s *awsKinesisStreamScaler) Close() error {
	return nil
}

func (s *awsKinesisStreamScaler) GetMetricSpecForScaling() []v2beta1.MetricSpec {
	targetShardCountQty := resource.NewQuantity(int64(s.metadata.targetShardCount), resource.DecimalSI)
	externalMetric := &v2beta1.ExternalMetricSource{MetricName: fmt.Sprintf("%s-%s-%s", "AWS-Kinesis-Stream", awsKinesisStreamMetricName, s.metadata.streamName),
		TargetAverageValue: targetShardCountQty}
	metricSpec := v2beta1.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2beta1.MetricSpec{metricSpec}
}

//GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *awsKinesisStreamScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	shardCount, err := s.GetAwsKinesisOpenShardCount()

	if err != nil {
		kinesisStreamLog.Error(err, "Error getting shard count")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(shardCount), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

// Get Kinesis open shard count
func (s *awsKinesisStreamScaler) GetAwsKinesisOpenShardCount() (int64, error) {
	input := &kinesis.DescribeStreamSummaryInput{
		StreamName: &s.metadata.streamName,
	}

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(s.metadata.awsRegion),
	}))

	var kinesisClinent *kinesis.Kinesis
	if s.metadata.awsAuthorization.podIdentityOwner {
		creds := credentials.NewStaticCredentials(s.metadata.awsAuthorization.awsAccessKeyID, s.metadata.awsAuthorization.awsSecretAccessKey, "")

		if s.metadata.awsAuthorization.awsRoleArn != "" {
			creds = stscreds.NewCredentials(sess, s.metadata.awsAuthorization.awsRoleArn)
		}

		kinesisClinent = kinesis.New(sess, &aws.Config{
			Region:      aws.String(s.metadata.awsRegion),
			Credentials: creds,
		})
	} else {
		kinesisClinent = kinesis.New(sess, &aws.Config{
			Region: aws.String(s.metadata.awsRegion),
		})
	}

	output, err := kinesisClinent.DescribeStreamSummary(input)
	if err != nil {
		return -1, err
	}

	return *output.StreamDescriptionSummary.OpenShardCount, nil
}
