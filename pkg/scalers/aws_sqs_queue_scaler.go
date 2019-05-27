package scalers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go/aws/credentials"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	v2beta1 "k8s.io/api/autoscaling/v2beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

const (
	awsSqsQueueMetricName    = "ApproximateNumberOfMessages"
	awsAccessKeyIDEnvVar     = "AWS_ACCESS_KEY_ID"
	awsSecretAccessKeyEnvVar = "AWS_SECRET_ACCESS_KEY"
)

type awsSqsQueueScaler struct {
	metadata *awsSqsQueueMetadata
}

type awsSqsQueueMetadata struct {
	targetQueueLength  int
	queueURL           string
	region             string
	awsAccessKeyID     string
	awsSecretAccessKey string
}

// NewawsSqsQueueScaler creates a new awsSqsQueueScaler
func NewAwsSqsQueueScaler(resolvedEnv, metadata map[string]string) (Scaler, error) {
	meta, err := parseAwsSqsQueueMetadata(metadata, resolvedEnv)
	if err != nil {
		return nil, fmt.Errorf("Error parsing SQS queue metadata: %s", err)
	}

	return &awsSqsQueueScaler{
		metadata: meta,
	}, nil
}

func parseAwsSqsQueueMetadata(metadata, resolvedEnv map[string]string) (*awsSqsQueueMetadata, error) {
	meta := awsSqsQueueMetadata{}
	meta.targetQueueLength = defaultTargetQueueLength

	if val, ok := metadata["queueLength"]; ok && val != "" {
		queueLength, err := strconv.Atoi(val)
		if err != nil {
			log.Errorf("Error parsing SQS queue metadata %s: %s", "queueLength", err)
		} else {
			meta.targetQueueLength = queueLength
		}
	}

	if val, ok := metadata["queueURL"]; ok && val != "" {
		meta.queueURL = val
	} else {
		return nil, fmt.Errorf("no queueURL given")
	}

	if val, ok := metadata["region"]; ok && val != "" {
		meta.region = val
	} else {
		return nil, fmt.Errorf("no region given")
	}

	if val, ok := resolvedEnv["awsAccessKeyID"]; ok && val != "" {
		meta.awsAccessKeyID = val
	} else {
		meta.awsAccessKeyID = awsAccessKeyIDEnvVar
	}

	if val, ok := resolvedEnv["awsSecretAccessKey"]; ok && val != "" {
		meta.awsSecretAccessKey = val
	} else {
		meta.awsSecretAccessKey = awsSecretAccessKeyEnvVar
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
		log.Errorf("Error getting queue length %s", err)
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
	creds := credentials.NewStaticCredentials(s.metadata.awsAccessKeyID, s.metadata.awsSecretAccessKey, "")
	sess := session.New(&aws.Config{
		Region:      aws.String(s.metadata.region),
		Credentials: creds,
	})

	sqsClient := sqs.New(sess)

	input := &sqs.GetQueueAttributesInput{
		AttributeNames: aws.StringSlice([]string{"All"}),
		QueueUrl:       aws.String(s.metadata.queueURL),
	}

	output, err := sqsClient.GetQueueAttributes(input)
	if err != nil {
		return -1, err
	}

	approximateNumberOfMessages, err := strconv.Atoi(*output.Attributes["ApproximateNumberOfMessages"])
	if err != nil {
		return -1, err
	}

	return int32(approximateNumberOfMessages), nil
}
