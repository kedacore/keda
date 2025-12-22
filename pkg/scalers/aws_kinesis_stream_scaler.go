package scalers

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	awsutils "github.com/kedacore/keda/v2/pkg/scalers/aws"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	targetShardCountDefault           = 2
	activationTargetShardCountDefault = 0
)

type awsKinesisStreamScaler struct {
	metricType           v2.MetricTargetType
	metadata             *awsKinesisStreamMetadata
	kinesisWrapperClient KinesisWrapperClient
	logger               logr.Logger
}

type KinesisWrapperClient interface {
	DescribeStreamSummary(context.Context, *kinesis.DescribeStreamSummaryInput, ...func(*kinesis.Options)) (*kinesis.DescribeStreamSummaryOutput, error)
}

type kinesisWrapperClient struct {
	kinesisClient *kinesis.Client
}

func (w kinesisWrapperClient) DescribeStreamSummary(ctx context.Context, params *kinesis.DescribeStreamSummaryInput, optFns ...func(*kinesis.Options)) (*kinesis.DescribeStreamSummaryOutput, error) {
	return w.kinesisClient.DescribeStreamSummary(ctx, params, optFns...)
}

type awsKinesisStreamMetadata struct {
	TargetShardCount           int64  `keda:"name=shardCount, order=triggerMetadata, default=2"`
	ActivationTargetShardCount int64  `keda:"name=activationShardCount, order=triggerMetadata, default=0"`
	StreamName                 string `keda:"name=streamName, order=triggerMetadata"`
	AwsRegion                  string `keda:"name=awsRegion, order=triggerMetadata;authParams"`
	AwsEndpoint                string `keda:"name=awsEndpoint, order=triggerMetadata, optional"`
	awsAuthorization           awsutils.AuthorizationMetadata
	triggerIndex               int

	IdentityOwner string `keda:"name=identityOwner, order=triggerMetadata, optional"`
}

// NewAwsKinesisStreamScaler creates a new awsKinesisStreamScaler
func NewAwsKinesisStreamScaler(ctx context.Context, config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "aws_kinesis_stream_scaler")

	meta, err := parseAwsKinesisStreamMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing Kinesis stream metadata: %w", err)
	}
	awsKinesisClient, err := createKinesisClient(ctx, meta)
	if err != nil {
		return nil, fmt.Errorf("error creating kinesis client: %w", err)
	}

	return &awsKinesisStreamScaler{
		metricType: metricType,
		metadata:   meta,
		kinesisWrapperClient: &kinesisWrapperClient{
			kinesisClient: awsKinesisClient,
		},
		logger: logger,
	}, nil
}

func parseAwsKinesisStreamMetadata(config *scalersconfig.ScalerConfig) (*awsKinesisStreamMetadata, error) {
	meta := &awsKinesisStreamMetadata{}

	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing Kinesis stream metadata: %w", err)
	}

	auth, err := awsutils.GetAwsAuthorization(config.TriggerUniqueKey, meta.AwsRegion, config.PodIdentity, config.TriggerMetadata, config.AuthParams, config.ResolvedEnv)
	if err != nil {
		return nil, err
	}

	meta.awsAuthorization = auth
	meta.triggerIndex = config.TriggerIndex

	return meta, nil
}

func createKinesisClient(ctx context.Context, metadata *awsKinesisStreamMetadata) (*kinesis.Client, error) {
	cfg, err := awsutils.GetAwsConfig(ctx, metadata.awsAuthorization)
	if err != nil {
		return nil, err
	}
	return kinesis.NewFromConfig(*cfg, func(options *kinesis.Options) {
		if metadata.AwsEndpoint != "" {
			options.BaseEndpoint = aws.String(metadata.AwsEndpoint)
		}
	}), nil
}

func (s *awsKinesisStreamScaler) Close(context.Context) error {
	awsutils.ClearAwsConfig(s.metadata.awsAuthorization)
	return nil
}

func (s *awsKinesisStreamScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("aws-kinesis-%s", s.metadata.StreamName))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.TargetShardCount),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *awsKinesisStreamScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	shardCount, err := s.GetAwsKinesisOpenShardCount(ctx)

	if err != nil {
		s.logger.Error(err, "Error getting shard count")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, float64(shardCount))

	return []external_metrics.ExternalMetricValue{metric}, shardCount > s.metadata.ActivationTargetShardCount, nil
}

// GetAwsKinesisOpenShardCount Get Kinesis open shard count
func (s *awsKinesisStreamScaler) GetAwsKinesisOpenShardCount(ctx context.Context) (int64, error) {
	input := &kinesis.DescribeStreamSummaryInput{
		StreamName: &s.metadata.StreamName,
	}

	output, err := s.kinesisWrapperClient.DescribeStreamSummary(ctx, input)
	if err != nil {
		return -1, err
	}

	return int64(*output.StreamDescriptionSummary.OpenShardCount), nil
}
