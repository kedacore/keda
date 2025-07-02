package scalers

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	awsutils "github.com/kedacore/keda/v2/pkg/scalers/aws"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type awsSqsQueueScaler struct {
	metricType       v2.MetricTargetType
	metadata         *awsSqsQueueMetadata
	sqsWrapperClient SqsWrapperClient
	logger           logr.Logger
}

type awsSqsQueueMetadata struct {
	TargetQueueLength           int64  `keda:"name=queueLength, order=triggerMetadata, default=5"`
	ActivationTargetQueueLength int64  `keda:"name=activationQueueLength, order=triggerMetadata, default=0"`
	QueueURL                    string `keda:"name=queueURL, order=triggerMetadata;resolvedEnv"`
	queueName                   string
	AwsRegion                   string `keda:"name=awsRegion, order=triggerMetadata;authParams"`
	AwsEndpoint                 string `keda:"name=awsEndpoint, order=triggerMetadata, optional"`
	awsAuthorization            awsutils.AuthorizationMetadata
	triggerIndex                int
	ScaleOnInFlight             bool `keda:"name=scaleOnInFlight, order=triggerMetadata, default=true"`
	ScaleOnDelayed              bool `keda:"name=scaleOnDelayed, order=triggerMetadata, default=false"`
	awsSqsQueueMetricNames      []types.QueueAttributeName
}

// NewAwsSqsQueueScaler creates a new awsSqsQueueScaler
func NewAwsSqsQueueScaler(ctx context.Context, config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "aws_sqs_queue_scaler")

	meta, err := parseAwsSqsQueueMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing SQS queue metadata: %w", err)
	}
	awsSqsClient, err := createSqsClient(ctx, meta)
	if err != nil {
		return nil, fmt.Errorf("error when creating sqs client: %w", err)
	}
	return &awsSqsQueueScaler{
		metricType: metricType,
		metadata:   meta,
		sqsWrapperClient: &sqsWrapperClient{
			sqsClient: awsSqsClient,
		},
		logger: logger,
	}, nil
}

type SqsWrapperClient interface {
	GetQueueAttributes(ctx context.Context, params *sqs.GetQueueAttributesInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueAttributesOutput, error)
}

type sqsWrapperClient struct {
	sqsClient *sqs.Client
}

func (w sqsWrapperClient) GetQueueAttributes(ctx context.Context, params *sqs.GetQueueAttributesInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueAttributesOutput, error) {
	return w.sqsClient.GetQueueAttributes(ctx, params, optFns...)
}

func parseAwsSqsQueueMetadata(config *scalersconfig.ScalerConfig) (*awsSqsQueueMetadata, error) {
	meta := &awsSqsQueueMetadata{}

	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing SQS queue metadata: %w", err)
	}

	meta.awsSqsQueueMetricNames = []types.QueueAttributeName{}
	meta.awsSqsQueueMetricNames = append(meta.awsSqsQueueMetricNames, types.QueueAttributeNameApproximateNumberOfMessages)
	if meta.ScaleOnInFlight {
		meta.awsSqsQueueMetricNames = append(meta.awsSqsQueueMetricNames, types.QueueAttributeNameApproximateNumberOfMessagesNotVisible)
	}
	if meta.ScaleOnDelayed {
		meta.awsSqsQueueMetricNames = append(meta.awsSqsQueueMetricNames, types.QueueAttributeNameApproximateNumberOfMessagesDelayed)
	}

	queueURL, err := url.ParseRequestURI(meta.QueueURL)
	if err != nil {
		// queueURL is not a valid URL, using it as queueName
		meta.queueName = meta.QueueURL
	} else {
		queueURLPath := queueURL.Path
		queueURLPathParts := strings.Split(queueURLPath, "/")
		if len(queueURLPathParts) != 3 || len(queueURLPathParts[2]) == 0 {
			return nil, fmt.Errorf("cannot get queueName from queueURL")
		}

		meta.queueName = queueURLPathParts[2]
	}

	auth, err := awsutils.GetAwsAuthorization(config.TriggerUniqueKey, meta.AwsRegion, config.PodIdentity, config.TriggerMetadata, config.AuthParams, config.ResolvedEnv)
	if err != nil {
		return nil, err
	}

	meta.awsAuthorization = auth

	meta.triggerIndex = config.TriggerIndex

	return meta, nil
}

func createSqsClient(ctx context.Context, metadata *awsSqsQueueMetadata) (*sqs.Client, error) {
	cfg, err := awsutils.GetAwsConfig(ctx, metadata.awsAuthorization)
	if err != nil {
		return nil, err
	}
	return sqs.NewFromConfig(*cfg, func(options *sqs.Options) {
		if metadata.AwsEndpoint != "" {
			options.BaseEndpoint = aws.String(metadata.AwsEndpoint)
		}
	}), nil
}

func (s *awsSqsQueueScaler) Close(context.Context) error {
	awsutils.ClearAwsConfig(s.metadata.awsAuthorization)
	return nil
}

func (s *awsSqsQueueScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(fmt.Sprintf("aws-sqs-%s", s.metadata.queueName))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.TargetQueueLength),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *awsSqsQueueScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	queuelen, err := s.getAwsSqsQueueLength(ctx)

	if err != nil {
		s.logger.Error(err, "Error getting queue length")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, float64(queuelen))

	return []external_metrics.ExternalMetricValue{metric}, queuelen > s.metadata.ActivationTargetQueueLength, nil
}

// Get SQS Queue Length
func (s *awsSqsQueueScaler) getAwsSqsQueueLength(ctx context.Context) (int64, error) {
	input := &sqs.GetQueueAttributesInput{
		AttributeNames: s.metadata.awsSqsQueueMetricNames,
		QueueUrl:       aws.String(s.metadata.QueueURL),
	}

	output, err := s.sqsWrapperClient.GetQueueAttributes(ctx, input)
	if err != nil {
		return -1, err
	}

	return s.processQueueLengthFromSqsQueueAttributesOutput(output)
}

func (s *awsSqsQueueScaler) processQueueLengthFromSqsQueueAttributesOutput(output *sqs.GetQueueAttributesOutput) (int64, error) {
	var approximateNumberOfMessages int64

	for _, awsSqsQueueMetric := range s.metadata.awsSqsQueueMetricNames {
		metricValueString, exists := output.Attributes[string(awsSqsQueueMetric)]
		if !exists {
			return -1, fmt.Errorf("metric %s not found in SQS queue attributes", awsSqsQueueMetric)
		}

		metricValue, err := strconv.ParseInt(metricValueString, 10, 64)
		if err != nil {
			return -1, err
		}

		approximateNumberOfMessages += metricValue
	}

	return approximateNumberOfMessages, nil
}
