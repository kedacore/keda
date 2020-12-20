package scalers

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	awsSqsQueueMetricName    = "ApproximateNumberOfMessages"
	targetQueueLengthDefault = 5
)

type awsSqsQueueScaler struct {
	metadata *awsSqsQueueMetadata
}

type awsSqsQueueMetadata struct {
	targetQueueLength int
	queueURL          string
	queueName         string
	awsRegion         string
	awsAuthorization  awsAuthorizationMetadata
}

var sqsQueueLog = logf.Log.WithName("aws_sqs_queue_scaler")

// NewAwsSqsQueueScaler creates a new awsSqsQueueScaler
func NewAwsSqsQueueScaler(config *ScalerConfig) (Scaler, error) {
	meta, err := parseAwsSqsQueueMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing SQS queue metadata: %s", err)
	}

	return &awsSqsQueueScaler{
		metadata: meta,
	}, nil
}

func parseAwsSqsQueueMetadata(config *ScalerConfig) (*awsSqsQueueMetadata, error) {
	meta := awsSqsQueueMetadata{}
	meta.targetQueueLength = defaultTargetQueueLength

	if val, ok := config.TriggerMetadata["queueLength"]; ok && val != "" {
		queueLength, err := strconv.Atoi(val)
		if err != nil {
			meta.targetQueueLength = targetQueueLengthDefault
			sqsQueueLog.Error(err, "Error parsing SQS queue metadata queueLength, using default %n", targetQueueLengthDefault)
		} else {
			meta.targetQueueLength = queueLength
		}
	}

	if val, ok := config.TriggerMetadata["queueURL"]; ok && val != "" {
		meta.queueURL = val
	} else {
		return nil, fmt.Errorf("no queueURL given")
	}

	queueURL, err := url.ParseRequestURI(meta.queueURL)
	if err != nil {
		return nil, fmt.Errorf("queueURL is not a valid URL")
	}

	queueURLPath := queueURL.Path
	queueURLPathParts := strings.Split(queueURLPath, "/")
	if len(queueURLPathParts) != 3 || len(queueURLPathParts[2]) == 0 {
		return nil, fmt.Errorf("cannot get queueName from queueURL")
	}

	meta.queueName = queueURLPathParts[2]

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

func (s *awsSqsQueueScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	targetQueueLengthQty := resource.NewQuantity(int64(s.metadata.targetQueueLength), resource.DecimalSI)
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: kedautil.NormalizeString(fmt.Sprintf("%s-%s", "AWS-SQS-Queue", s.metadata.queueName)),
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: targetQueueLengthQty,
		},
	}
	metricSpec := v2beta2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2beta2.MetricSpec{metricSpec}
}

// GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
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
		AttributeNames: aws.StringSlice([]string{awsSqsQueueMetricName}),
		QueueUrl:       aws.String(s.metadata.queueURL),
	}

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(s.metadata.awsRegion),
	}))

	var sqsClient *sqs.SQS
	if s.metadata.awsAuthorization.podIdentityOwner {
		creds := credentials.NewStaticCredentials(s.metadata.awsAuthorization.awsAccessKeyID, s.metadata.awsAuthorization.awsSecretAccessKey, "")

		if s.metadata.awsAuthorization.awsRoleArn != "" {
			creds = stscreds.NewCredentials(sess, s.metadata.awsAuthorization.awsRoleArn)
		}

		sqsClient = sqs.New(sess, &aws.Config{
			Region:      aws.String(s.metadata.awsRegion),
			Credentials: creds,
		})
	} else {
		sqsClient = sqs.New(sess, &aws.Config{
			Region: aws.String(s.metadata.awsRegion),
		})
	}

	output, err := sqsClient.GetQueueAttributes(input)
	if err != nil {
		return -1, err
	}

	approximateNumberOfMessages, err := strconv.Atoi(*output.Attributes[awsSqsQueueMetricName])
	if err != nil {
		return -1, err
	}

	return int32(approximateNumberOfMessages), nil
}
