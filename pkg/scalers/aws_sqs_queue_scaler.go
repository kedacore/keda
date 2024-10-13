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

const (
	targetQueueLengthDefault           = 5
	activationTargetQueueLengthDefault = 0
	defaultScaleOnInFlight             = true
	defaultScaleOnDelayed              = false
)

type awsSqsQueueScaler struct {
	metricType       v2.MetricTargetType
	metadata         *awsSqsQueueMetadata
	sqsWrapperClient SqsWrapperClient
	logger           logr.Logger
}

type awsSqsQueueMetadata struct {
	targetQueueLength           int64
	activationTargetQueueLength int64
	queueURL                    string
	queueName                   string
	awsRegion                   string
	awsEndpoint                 string
	awsAuthorization            awsutils.AuthorizationMetadata
	triggerIndex                int
	scaleOnInFlight             bool
	scaleOnDelayed              bool
	awsSqsQueueMetricNames      []types.QueueAttributeName
}

// NewAwsSqsQueueScaler creates a new awsSqsQueueScaler
func NewAwsSqsQueueScaler(ctx context.Context, config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "aws_sqs_queue_scaler")

	meta, err := parseAwsSqsQueueMetadata(config, logger)
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

func parseAwsSqsQueueMetadata(config *scalersconfig.ScalerConfig, logger logr.Logger) (*awsSqsQueueMetadata, error) {
	meta := awsSqsQueueMetadata{}
	meta.targetQueueLength = defaultTargetQueueLength
	meta.scaleOnInFlight = defaultScaleOnInFlight
	meta.scaleOnDelayed = defaultScaleOnDelayed

	if val, ok := config.TriggerMetadata["queueLength"]; ok && val != "" {
		queueLength, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			meta.targetQueueLength = targetQueueLengthDefault
			logger.Error(err, "Error parsing SQS queue metadata queueLength, using default %n", targetQueueLengthDefault)
		} else {
			meta.targetQueueLength = queueLength
		}
	}

	if val, ok := config.TriggerMetadata["activationQueueLength"]; ok && val != "" {
		activationQueueLength, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			meta.activationTargetQueueLength = activationTargetQueueLengthDefault
			logger.Error(err, "Error parsing SQS queue metadata activationQueueLength, using default %n", activationTargetQueueLengthDefault)
		} else {
			meta.activationTargetQueueLength = activationQueueLength
		}
	}

	if val, ok := config.TriggerMetadata["scaleOnDelayed"]; ok && val != "" {
		scaleOnDelayed, err := strconv.ParseBool(val)
		if err != nil {
			meta.scaleOnDelayed = defaultScaleOnDelayed
			logger.Error(err, "Error parsing SQS queue metadata scaleOnDelayed, using default %n", defaultScaleOnDelayed)
		} else {
			meta.scaleOnDelayed = scaleOnDelayed
		}
	}

	if val, ok := config.TriggerMetadata["scaleOnInFlight"]; ok && val != "" {
		scaleOnInFlight, err := strconv.ParseBool(val)
		if err != nil {
			meta.scaleOnInFlight = defaultScaleOnInFlight
			logger.Error(err, "Error parsing SQS queue metadata scaleOnInFlight, using default %n", defaultScaleOnInFlight)
		} else {
			meta.scaleOnInFlight = scaleOnInFlight
		}
	}

	meta.awsSqsQueueMetricNames = []types.QueueAttributeName{}
	meta.awsSqsQueueMetricNames = append(meta.awsSqsQueueMetricNames, types.QueueAttributeNameApproximateNumberOfMessages)
	if meta.scaleOnInFlight {
		meta.awsSqsQueueMetricNames = append(meta.awsSqsQueueMetricNames, types.QueueAttributeNameApproximateNumberOfMessagesNotVisible)
	}
	if meta.scaleOnDelayed {
		meta.awsSqsQueueMetricNames = append(meta.awsSqsQueueMetricNames, types.QueueAttributeNameApproximateNumberOfMessagesDelayed)
	}

	if val, ok := config.TriggerMetadata["queueURL"]; ok && val != "" {
		meta.queueURL = val
	} else if val, ok := config.TriggerMetadata["queueURLFromEnv"]; ok && val != "" {
		if val, ok := config.ResolvedEnv[val]; ok && val != "" {
			meta.queueURL = val
		} else {
			return nil, fmt.Errorf("queueURLFromEnv `%s` env variable value is empty", config.TriggerMetadata["queueURLFromEnv"])
		}
	} else {
		return nil, fmt.Errorf("no queueURL given")
	}

	queueURL, err := url.ParseRequestURI(meta.queueURL)
	if err != nil {
		// queueURL is not a valid URL, using it as queueName
		meta.queueName = meta.queueURL
	} else {
		queueURLPath := queueURL.Path
		queueURLPathParts := strings.Split(queueURLPath, "/")
		if len(queueURLPathParts) != 3 || len(queueURLPathParts[2]) == 0 {
			return nil, fmt.Errorf("cannot get queueName from queueURL")
		}

		meta.queueName = queueURLPathParts[2]
	}

	if val, ok := config.TriggerMetadata["awsRegion"]; ok && val != "" {
		meta.awsRegion = val
	} else {
		return nil, fmt.Errorf("no awsRegion given")
	}

	if val, ok := config.TriggerMetadata["awsEndpoint"]; ok {
		meta.awsEndpoint = val
	}

	auth, err := awsutils.GetAwsAuthorization(config.TriggerUniqueKey, config.PodIdentity, config.TriggerMetadata, config.AuthParams, config.ResolvedEnv)
	if err != nil {
		return nil, err
	}

	meta.awsAuthorization = auth

	meta.triggerIndex = config.TriggerIndex

	return &meta, nil
}

func createSqsClient(ctx context.Context, metadata *awsSqsQueueMetadata) (*sqs.Client, error) {
	cfg, err := awsutils.GetAwsConfig(ctx, metadata.awsRegion, metadata.awsAuthorization)
	if err != nil {
		return nil, err
	}
	return sqs.NewFromConfig(*cfg, func(options *sqs.Options) {
		if metadata.awsEndpoint != "" {
			options.BaseEndpoint = aws.String(metadata.awsEndpoint)
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
		Target: GetMetricTarget(s.metricType, s.metadata.targetQueueLength),
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

	return []external_metrics.ExternalMetricValue{metric}, queuelen > s.metadata.activationTargetQueueLength, nil
}

// Get SQS Queue Length
func (s *awsSqsQueueScaler) getAwsSqsQueueLength(ctx context.Context) (int64, error) {
	input := &sqs.GetQueueAttributesInput{
		AttributeNames: s.metadata.awsSqsQueueMetricNames,
		QueueUrl:       aws.String(s.metadata.queueURL),
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
