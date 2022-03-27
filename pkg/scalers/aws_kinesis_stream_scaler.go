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
	"github.com/aws/aws-sdk-go/service/kinesis/kinesisiface"
	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	targetShardCountDefault = 2
)

type awsKinesisStreamScaler struct {
	metricType    v2beta2.MetricTargetType
	metadata      *awsKinesisStreamMetadata
	kinesisClient kinesisiface.KinesisAPI
}

type awsKinesisStreamMetadata struct {
	targetShardCount int64
	streamName       string
	awsRegion        string
	awsAuthorization awsAuthorizationMetadata
	scalerIndex      int
}

var kinesisStreamLog = logf.Log.WithName("aws_kinesis_stream_scaler")

// NewAwsKinesisStreamScaler creates a new awsKinesisStreamScaler
func NewAwsKinesisStreamScaler(config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %s", err)
	}

	meta, err := parseAwsKinesisStreamMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing Kinesis stream metadata: %s", err)
	}

	return &awsKinesisStreamScaler{
		metricType:    metricType,
		metadata:      meta,
		kinesisClient: createKinesisClient(meta),
	}, nil
}

func parseAwsKinesisStreamMetadata(config *ScalerConfig) (*awsKinesisStreamMetadata, error) {
	meta := awsKinesisStreamMetadata{}
	meta.targetShardCount = targetShardCountDefault

	if val, ok := config.TriggerMetadata["shardCount"]; ok && val != "" {
		shardCount, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			meta.targetShardCount = targetShardCountDefault
			kinesisStreamLog.Error(err, "Error parsing Kinesis stream metadata shardCount, using default %n", targetShardCountDefault)
		} else {
			meta.targetShardCount = shardCount
		}
	}

	if val, ok := config.TriggerMetadata["streamName"]; ok && val != "" {
		meta.streamName = val
	} else {
		return nil, fmt.Errorf("no streamName given")
	}

	if val, ok := config.TriggerMetadata["awsRegion"]; ok && val != "" {
		meta.awsRegion = val
	} else {
		return nil, fmt.Errorf("no awsRegion given")
	}

	auth, err := getAwsAuthorization(config.AuthParams, config.TriggerMetadata, config.ResolvedEnv)
	if err != nil {
		return nil, err
	}

	meta.awsAuthorization = auth

	meta.scalerIndex = config.ScalerIndex

	return &meta, nil
}

func createKinesisClient(metadata *awsKinesisStreamMetadata) *kinesis.Kinesis {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(metadata.awsRegion),
	}))

	var kinesisClinent *kinesis.Kinesis
	if metadata.awsAuthorization.podIdentityOwner {
		creds := credentials.NewStaticCredentials(metadata.awsAuthorization.awsAccessKeyID, metadata.awsAuthorization.awsSecretAccessKey, metadata.awsAuthorization.awsSessionToken)

		if metadata.awsAuthorization.awsRoleArn != "" {
			creds = stscreds.NewCredentials(sess, metadata.awsAuthorization.awsRoleArn)
		}

		kinesisClinent = kinesis.New(sess, &aws.Config{
			Region:      aws.String(metadata.awsRegion),
			Credentials: creds,
		})
	} else {
		kinesisClinent = kinesis.New(sess, &aws.Config{
			Region: aws.String(metadata.awsRegion),
		})
	}
	return kinesisClinent
}

// IsActive determines if we need to scale from zero
func (s *awsKinesisStreamScaler) IsActive(ctx context.Context) (bool, error) {
	count, err := s.GetAwsKinesisOpenShardCount()

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (s *awsKinesisStreamScaler) Close(context.Context) error {
	return nil
}

func (s *awsKinesisStreamScaler) GetMetricSpecForScaling(context.Context) []v2beta2.MetricSpec {
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, kedautil.NormalizeString(fmt.Sprintf("aws-kinesis-%s", s.metadata.streamName))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.targetShardCount),
	}
	metricSpec := v2beta2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2beta2.MetricSpec{metricSpec}
}

// GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *awsKinesisStreamScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	shardCount, err := s.GetAwsKinesisOpenShardCount()

	if err != nil {
		kinesisStreamLog.Error(err, "Error getting shard count")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(shardCount, resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

// Get Kinesis open shard count
func (s *awsKinesisStreamScaler) GetAwsKinesisOpenShardCount() (int64, error) {
	input := &kinesis.DescribeStreamSummaryInput{
		StreamName: &s.metadata.streamName,
	}

	output, err := s.kinesisClient.DescribeStreamSummary(input)
	if err != nil {
		return -1, err
	}

	return *output.StreamDescriptionSummary.OpenShardCount, nil
}
