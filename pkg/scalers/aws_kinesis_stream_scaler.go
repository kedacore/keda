package scalers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/kinesis/kinesisiface"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	targetShardCountDefault           = 2
	activationTargetShardCountDefault = 0
)

type awsKinesisStreamScaler struct {
	metricType    v2.MetricTargetType
	metadata      *awsKinesisStreamMetadata
	kinesisClient kinesisiface.KinesisAPI
	logger        logr.Logger
}

type awsKinesisStreamMetadata struct {
	targetShardCount           int64
	activationTargetShardCount int64
	streamName                 string
	awsRegion                  string
	awsEndpoint                string
	awsAuthorization           awsAuthorizationMetadata
	scalerIndex                int
}

// NewAwsKinesisStreamScaler creates a new awsKinesisStreamScaler
func NewAwsKinesisStreamScaler(config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "aws_kinesis_stream_scaler")

	meta, err := parseAwsKinesisStreamMetadata(config, logger)
	if err != nil {
		return nil, fmt.Errorf("error parsing Kinesis stream metadata: %w", err)
	}

	return &awsKinesisStreamScaler{
		metricType:    metricType,
		metadata:      meta,
		kinesisClient: createKinesisClient(meta),
		logger:        logger,
	}, nil
}

func parseAwsKinesisStreamMetadata(config *ScalerConfig, logger logr.Logger) (*awsKinesisStreamMetadata, error) {
	meta := awsKinesisStreamMetadata{}
	meta.targetShardCount = targetShardCountDefault

	if val, ok := config.TriggerMetadata["shardCount"]; ok && val != "" {
		shardCount, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			meta.targetShardCount = targetShardCountDefault
			logger.Error(err, "Error parsing Kinesis stream metadata shardCount, using default %n", targetShardCountDefault)
		} else {
			meta.targetShardCount = shardCount
		}
	}

	if val, ok := config.TriggerMetadata["activationShardCount"]; ok && val != "" {
		activationShardCount, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			meta.activationTargetShardCount = activationTargetShardCountDefault
			logger.Error(err, "Error parsing Kinesis stream metadata activationShardCount, using default %n", activationTargetShardCountDefault)
		} else {
			meta.activationTargetShardCount = activationShardCount
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

	if val, ok := config.TriggerMetadata["awsEndpoint"]; ok {
		meta.awsEndpoint = val
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
	sess, config := getAwsConfig(metadata.awsRegion,
		metadata.awsEndpoint,
		metadata.awsAuthorization)

	return kinesis.New(sess, config)
}

func (s *awsKinesisStreamScaler) Close(context.Context) error {
	return nil
}

func (s *awsKinesisStreamScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, kedautil.NormalizeString(fmt.Sprintf("aws-kinesis-%s", s.metadata.streamName))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.targetShardCount),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *awsKinesisStreamScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	shardCount, err := s.GetAwsKinesisOpenShardCount()

	if err != nil {
		s.logger.Error(err, "Error getting shard count")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, float64(shardCount))

	return []external_metrics.ExternalMetricValue{metric}, shardCount > s.metadata.activationTargetShardCount, nil
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
