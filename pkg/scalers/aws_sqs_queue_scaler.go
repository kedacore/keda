package scalers

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go/aws/credentials"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	v2beta1 "k8s.io/api/autoscaling/v2beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	awsSqsQueueMetricName    = "ApproximateNumberOfMessages"
	awsAccessKeyIDEnvVar     = "AWS_ACCESS_KEY_ID"
	awsSecretAccessKeyEnvVar = "AWS_SECRET_ACCESS_KEY"
)

type awsSqsQueueScaler struct {
	metadata  *awsSqsQueueMetadata
	sqsClient *sqs.SQS
}

type awsSqsQueueMetadata struct {
	targetQueueLength  int
	queueURL           string
	awsRegion          string
	awsAccessKeyID     string
	awsSecretAccessKey string
	awsSessionToken    string
}

var sqsQueueLog = logf.Log.WithName("aws_sqs_queue_scaler")

// NewAwsSqsQueueScaler creates a new awsSqsQueueScaler
func NewAwsSqsQueueScaler(resolvedEnv, metadata map[string]string) (Scaler, error) {
	meta, err := parseAwsSqsQueueMetadata(metadata, resolvedEnv)
	if err != nil {
		return nil, fmt.Errorf("Error parsing SQS queue metadata: %s", err)
	}

	sess := session.Must(session.NewSession())
	if sess != nil {
		s, err := session.NewSession(&aws.Config{
			Credentials: credentials.NewStaticCredentials(meta.awsAccessKeyID, meta.awsSecretAccessKey, meta.awsSessionToken),
			Region:      aws.String(meta.awsRegion),
		})

		if err != nil {
			return nil, errors.New("unable to get an AWS session with the default provider chain or provided credentials")
		}

		sess = s
	}

	sqsClient := sqs.New(sess)

	return &awsSqsQueueScaler{
		metadata:  meta,
		sqsClient: sqsClient,
	}, nil
}

func parseAwsSqsQueueMetadata(metadata, resolvedEnv map[string]string) (*awsSqsQueueMetadata, error) {
	meta := awsSqsQueueMetadata{}
	meta.targetQueueLength = defaultTargetQueueLength

	if val, ok := metadata["queueLength"]; ok && val != "" {
		queueLength, err := strconv.Atoi(val)
		if err != nil {
			sqsQueueLog.Error(err, "Error parsing SQS queue metadata queueLength")
		} else {
			meta.targetQueueLength = queueLength
		}
	}

	if val, ok := metadata["queueURL"]; ok && val != "" {
		meta.queueURL = val
	} else {
		return nil, fmt.Errorf("no queueURL given")
	}

	var keyName string
	if keyName = metadata["awsAccessKeyID"]; keyName == "" {
		keyName = awsAccessKeyIDEnvVar
	}
	if val, ok := resolvedEnv[keyName]; ok && val != "" {
		meta.awsAccessKeyID = val
	} else {
		return nil, fmt.Errorf("'%s' doesn't exist in the deployment environment", keyName)
	}

	if keyName = metadata["awsSecretAccessKey"]; keyName == "" {
		keyName = awsSecretAccessKeyEnvVar
	}
	if val, ok := resolvedEnv[keyName]; ok && val != "" {
		meta.awsAccessKeyID = val
	} else {
		return nil, fmt.Errorf("'%s' doesn't exist in the deployment environment", keyName)
	}

	if val, ok := metadata["awsSecretAccessKey"]; ok && val != "" {
		meta.awsSecretAccessKey = val
	}

	if val, ok := metadata["awsSessionToken"]; ok && val != "" {
		meta.awsSessionToken = val
	}

	if val, ok := metadata["awsRegion"]; ok && val != "" {
		meta.awsRegion = val
	} else {
		return nil, fmt.Errorf("no awsRegion given")
	}

	return &meta, nil
}

// IsActive determines if we need to scale from zero
func (s *awsSqsQueueScaler) IsActive(ctx context.Context) (bool, error) {
	length, err := s.GetAwsSqsQueueLength()

	if err != nil {
		return false, err
	}

	return length > 0, nil
}

func (s *awsSqsQueueScaler) Close() error {
	return nil
}

func (s *awsSqsQueueScaler) GetMetricSpecForScaling() []v2beta1.MetricSpec {
	targetQueueLengthQty := resource.NewQuantity(int64(s.metadata.targetQueueLength), resource.DecimalSI)
	externalMetric := &v2beta1.ExternalMetricSource{MetricName: awsSqsQueueMetricName, TargetAverageValue: targetQueueLengthQty}
	metricSpec := v2beta1.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2beta1.MetricSpec{metricSpec}
}

//GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *awsSqsQueueScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	queuelen, err := s.GetAwsSqsQueueLength()

	if err != nil {
		sqsQueueLog.Error(err, "Error getting queue length")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(queuelen), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

// Get SQS Queue Length
func (s *awsSqsQueueScaler) GetAwsSqsQueueLength() (int32, error) {
	input := &sqs.GetQueueAttributesInput{
		AttributeNames: aws.StringSlice([]string{"ApproximateNumberOfMessages"}),
		QueueUrl:       aws.String(s.metadata.queueURL),
	}

	output, err := s.sqsClient.GetQueueAttributes(input)
	if err != nil {
		return -1, err
	}

	approximateNumberOfMessages, err := strconv.Atoi(*output.Attributes["ApproximateNumberOfMessages"])
	if err != nil {
		return -1, err
	}

	return int32(approximateNumberOfMessages), nil
}
