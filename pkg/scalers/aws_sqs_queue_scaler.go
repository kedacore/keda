package scalers

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	targetQueueLengthDefault           = 5
	activationTargetQueueLengthDefault = 0
	defaultScaleOnInFlight             = true
	defaultScaleOnDelayed              = false
)

type awsSqsQueueScaler struct {
	metricType v2.MetricTargetType
	metadata   *awsSqsQueueMetadata
	sqsClient  sqsiface.SQSAPI
	logger     logr.Logger
}

type awsSqsQueueMetadata struct {
	targetQueueLength           int64
	activationTargetQueueLength int64
	queueURL                    string
	queueName                   string
	awsRegion                   string
	awsEndpoint                 string
	awsAuthorization            awsAuthorizationMetadata
	scalerIndex                 int
	scaleOnInFlight             bool
	scaleOnDelayed              bool
	awsSqsQueueMetricNames      []string
}

// NewAwsSqsQueueScaler creates a new awsSqsQueueScaler
func NewAwsSqsQueueScaler(config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "aws_sqs_queue_scaler")

	meta, err := parseAwsSqsQueueMetadata(config, logger)
	if err != nil {
		return nil, fmt.Errorf("error parsing SQS queue metadata: %w", err)
	}

	return &awsSqsQueueScaler{
		metricType: metricType,
		metadata:   meta,
		sqsClient:  createSqsClient(meta),
		logger:     logger,
	}, nil
}

func parseAwsSqsQueueMetadata(config *ScalerConfig, logger logr.Logger) (*awsSqsQueueMetadata, error) {
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

	if val, ok := config.TriggerMetadata["scaleOnInFlight"]; ok && val != "" {
		scaleOnInFlight, err := strconv.ParseBool(val)
		if err != nil {
			meta.scaleOnInFlight = defaultScaleOnInFlight
			logger.Error(err, "Error parsing SQS queue metadata scaleOnInFlight, using default %n", defaultScaleOnInFlight)
		} else {
			meta.scaleOnInFlight = scaleOnInFlight
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

	meta.awsSqsQueueMetricNames = []string{"ApproximateNumberOfMessages"}
	if meta.scaleOnInFlight {
		meta.awsSqsQueueMetricNames = append(meta.awsSqsQueueMetricNames, "ApproximateNumberOfMessagesNotVisible")
	}
	if meta.scaleOnDelayed {
		meta.awsSqsQueueMetricNames = append(meta.awsSqsQueueMetricNames, "ApproximateNumberOfMessagesDelayed")
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

	auth, err := getAwsAuthorization(config.AuthParams, config.TriggerMetadata, config.ResolvedEnv)
	if err != nil {
		return nil, err
	}

	meta.awsAuthorization = auth

	meta.scalerIndex = config.ScalerIndex

	return &meta, nil
}

func createSqsClient(metadata *awsSqsQueueMetadata) *sqs.SQS {
	sess, config := getAwsConfig(metadata.awsRegion,
		metadata.awsEndpoint,
		metadata.awsAuthorization)

	return sqs.New(sess, config)
}

func (s *awsSqsQueueScaler) Close(context.Context) error {
	return nil
}

func (s *awsSqsQueueScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, kedautil.NormalizeString(fmt.Sprintf("aws-sqs-%s", s.metadata.queueName))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.targetQueueLength),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *awsSqsQueueScaler) GetMetricsAndActivity(_ context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	queuelen, err := s.getAwsSqsQueueLength()

	if err != nil {
		s.logger.Error(err, "Error getting queue length")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, float64(queuelen))

	return []external_metrics.ExternalMetricValue{metric}, queuelen > s.metadata.activationTargetQueueLength, nil
}

// Get SQS Queue Length
func (s *awsSqsQueueScaler) getAwsSqsQueueLength() (int64, error) {
	input := &sqs.GetQueueAttributesInput{
		AttributeNames: aws.StringSlice(s.metadata.awsSqsQueueMetricNames),
		QueueUrl:       aws.String(s.metadata.queueURL),
	}

	output, err := s.sqsClient.GetQueueAttributes(input)
	if err != nil {
		return -1, err
	}

	var approximateNumberOfMessages int64
	for _, awsSqsQueueMetric := range s.metadata.awsSqsQueueMetricNames {
		metricValue, err := strconv.ParseInt(*output.Attributes[awsSqsQueueMetric], 10, 32)
		if err != nil {
			return -1, err
		}
		approximateNumberOfMessages += metricValue
	}

	return approximateNumberOfMessages, nil
}
