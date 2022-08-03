package scalers

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/go-logr/logr"
	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	targetQueueLengthDefault           = 5
	activationTargetQueueLengthDefault = 0
	defaultScaleOnInFlight             = true
)

var awsSqsQueueMetricNames = []string{
	"ApproximateNumberOfMessages",
	"ApproximateNumberOfMessagesNotVisible",
}

type awsSqsQueueScaler struct {
	metricType v2beta2.MetricTargetType
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
	awsAuthorization            awsAuthorizationMetadata
	scalerIndex                 int
	scaleOnInFlight             bool
}

// NewAwsSqsQueueScaler creates a new awsSqsQueueScaler
func NewAwsSqsQueueScaler(config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %s", err)
	}

	logger := InitializeLogger(config, "aws_sqs_queue_scaler")

	meta, err := parseAwsSqsQueueMetadata(config, logger)
	if err != nil {
		return nil, fmt.Errorf("error parsing SQS queue metadata: %s", err)
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

	if !meta.scaleOnInFlight {
		awsSqsQueueMetricNames = []string{
			"ApproximateNumberOfMessages",
		}
	}

	if val, ok := config.TriggerMetadata["queueURL"]; ok && val != "" {
		meta.queueURL = val
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

	auth, err := getAwsAuthorization(config.AuthParams, config.TriggerMetadata, config.ResolvedEnv)
	if err != nil {
		return nil, err
	}

	meta.awsAuthorization = auth

	meta.scalerIndex = config.ScalerIndex

	return &meta, nil
}

func createSqsClient(metadata *awsSqsQueueMetadata) *sqs.SQS {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(metadata.awsRegion),
	}))

	var sqsClient *sqs.SQS
	if metadata.awsAuthorization.podIdentityOwner {
		creds := credentials.NewStaticCredentials(metadata.awsAuthorization.awsAccessKeyID, metadata.awsAuthorization.awsSecretAccessKey, metadata.awsAuthorization.awsSessionToken)

		if metadata.awsAuthorization.awsRoleArn != "" {
			creds = stscreds.NewCredentials(sess, metadata.awsAuthorization.awsRoleArn)
		}

		sqsClient = sqs.New(sess, &aws.Config{
			Region:      aws.String(metadata.awsRegion),
			Credentials: creds,
		})
	} else {
		sqsClient = sqs.New(sess, &aws.Config{
			Region: aws.String(metadata.awsRegion),
		})
	}
	return sqsClient
}

// IsActive determines if we need to scale from zero
func (s *awsSqsQueueScaler) IsActive(ctx context.Context) (bool, error) {
	length, err := s.getAwsSqsQueueLength()

	if err != nil {
		return false, err
	}

	return length > s.metadata.activationTargetQueueLength, nil
}

func (s *awsSqsQueueScaler) Close(context.Context) error {
	return nil
}

func (s *awsSqsQueueScaler) GetMetricSpecForScaling(context.Context) []v2beta2.MetricSpec {
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, kedautil.NormalizeString(fmt.Sprintf("aws-sqs-%s", s.metadata.queueName))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.targetQueueLength),
	}
	metricSpec := v2beta2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2beta2.MetricSpec{metricSpec}
}

// GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *awsSqsQueueScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	queuelen, err := s.getAwsSqsQueueLength()

	if err != nil {
		s.logger.Error(err, "Error getting queue length")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := GenerateMetricInMili(metricName, float64(queuelen))

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

// Get SQS Queue Length
func (s *awsSqsQueueScaler) getAwsSqsQueueLength() (int64, error) {
	input := &sqs.GetQueueAttributesInput{
		AttributeNames: aws.StringSlice(awsSqsQueueMetricNames),
		QueueUrl:       aws.String(s.metadata.queueURL),
	}

	output, err := s.sqsClient.GetQueueAttributes(input)
	if err != nil {
		return -1, err
	}

	var approximateNumberOfMessages int64
	for _, awsSqsQueueMetric := range awsSqsQueueMetricNames {
		metricValue, err := strconv.ParseInt(*output.Attributes[awsSqsQueueMetric], 10, 32)
		if err != nil {
			return -1, err
		}
		approximateNumberOfMessages += metricValue
	}

	return approximateNumberOfMessages, nil
}
