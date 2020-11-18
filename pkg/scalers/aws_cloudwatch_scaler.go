package scalers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	defaultMetricCollectionTime = 300
	defaultMetricStat           = "Average"
	defaultMetricStatPeriod     = 300
)

type awsCloudwatchScaler struct {
	metadata *awsCloudwatchMetadata
}

type awsCloudwatchMetadata struct {
	namespace      string
	metricsName    string
	dimensionName  string
	dimensionValue string

	targetMetricValue float64
	minMetricValue    float64

	metricCollectionTime int64
	metricStat           string
	metricStatPeriod     int64

	awsRegion string

	awsAuthorization awsAuthorizationMetadata
}

var cloudwatchLog = logf.Log.WithName("aws_cloudwatch_scaler")

// NewAwsCloudwatchScaler creates a new awsCloudwatchScaler
func NewAwsCloudwatchScaler(config *ScalerConfig) (Scaler, error) {
	meta, err := parseAwsCloudwatchMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing cloudwatch metadata: %s", err)
	}

	return &awsCloudwatchScaler{
		metadata: meta,
	}, nil
}

func parseAwsCloudwatchMetadata(config *ScalerConfig) (*awsCloudwatchMetadata, error) {
	meta := awsCloudwatchMetadata{}
	meta.metricCollectionTime = defaultMetricCollectionTime
	meta.metricStat = defaultMetricStat
	meta.metricStatPeriod = defaultMetricStatPeriod

	if val, ok := config.TriggerMetadata["namespace"]; ok && val != "" {
		meta.namespace = val
	} else {
		return nil, fmt.Errorf("namespace not given")
	}

	if val, ok := config.TriggerMetadata["metricName"]; ok && val != "" {
		meta.metricsName = val
	} else {
		return nil, fmt.Errorf("metric name not given")
	}

	if val, ok := config.TriggerMetadata["dimensionName"]; ok && val != "" {
		meta.dimensionName = val
	} else {
		return nil, fmt.Errorf("dimension name not given")
	}

	if val, ok := config.TriggerMetadata["dimensionValue"]; ok && val != "" {
		meta.dimensionValue = val
	} else {
		return nil, fmt.Errorf("dimension value not given")
	}

	if val, ok := config.TriggerMetadata["targetMetricValue"]; ok && val != "" {
		targetMetricValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			cloudwatchLog.Error(err, "Error parsing targetMetricValue metadata")
		} else {
			meta.targetMetricValue = targetMetricValue
		}
	} else {
		return nil, fmt.Errorf("target Metric Value not given")
	}

	if val, ok := config.TriggerMetadata["minMetricValue"]; ok && val != "" {
		minMetricValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			cloudwatchLog.Error(err, "Error parsing minMetricValue metadata")
		} else {
			meta.minMetricValue = minMetricValue
		}
	} else {
		return nil, fmt.Errorf("min metric value not given")
	}

	if val, ok := config.TriggerMetadata["metricCollectionTime"]; ok && val != "" {
		metricCollectionTime, err := strconv.Atoi(val)
		if err != nil {
			cloudwatchLog.Error(err, "Error parsing metricCollectionTime metadata")
		} else {
			meta.metricCollectionTime = int64(metricCollectionTime)
		}
	}

	if val, ok := config.TriggerMetadata["metricStat"]; ok && val != "" {
		meta.metricStat = val
	}

	if val, ok := config.TriggerMetadata["metricStatPeriod"]; ok && val != "" {
		metricStatPeriod, err := strconv.Atoi(val)
		if err != nil {
			cloudwatchLog.Error(err, "Error parsing metricStatPeriod metadata")
		} else {
			meta.metricStatPeriod = int64(metricStatPeriod)
		}
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

	return &meta, nil
}

func (c *awsCloudwatchScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	metricValue, err := c.GetCloudwatchMetrics()

	if err != nil {
		cloudwatchLog.Error(err, "Error getting metric value")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(metricValue), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

func (c *awsCloudwatchScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	targetMetricValue := resource.NewQuantity(int64(c.metadata.targetMetricValue), resource.DecimalSI)
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: kedautil.NormalizeString(fmt.Sprintf("%s-%s-%s-%s", "aws-cloudwatch", c.metadata.namespace, c.metadata.dimensionName, c.metadata.dimensionValue)),
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: targetMetricValue,
		},
	}
	metricSpec := v2beta2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2beta2.MetricSpec{metricSpec}
}

func (c *awsCloudwatchScaler) IsActive(ctx context.Context) (bool, error) {
	val, err := c.GetCloudwatchMetrics()

	if err != nil {
		return false, err
	}

	return val > c.metadata.minMetricValue, nil
}

func (c *awsCloudwatchScaler) Close() error {
	return nil
}

func (c *awsCloudwatchScaler) GetCloudwatchMetrics() (float64, error) {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(c.metadata.awsRegion),
	}))

	var cloudwatchClient *cloudwatch.CloudWatch
	if c.metadata.awsAuthorization.podIdentityOwner {
		creds := credentials.NewStaticCredentials(c.metadata.awsAuthorization.awsAccessKeyID, c.metadata.awsAuthorization.awsSecretAccessKey, "")

		if c.metadata.awsAuthorization.awsRoleArn != "" {
			creds = stscreds.NewCredentials(sess, c.metadata.awsAuthorization.awsRoleArn)
		}

		cloudwatchClient = cloudwatch.New(sess, &aws.Config{
			Region:      aws.String(c.metadata.awsRegion),
			Credentials: creds,
		})
	} else {
		cloudwatchClient = cloudwatch.New(sess, &aws.Config{
			Region: aws.String(c.metadata.awsRegion),
		})
	}

	input := cloudwatch.GetMetricDataInput{
		StartTime: aws.Time(time.Now().Add(time.Second * -1 * time.Duration(c.metadata.metricCollectionTime))),
		EndTime:   aws.Time(time.Now()),
		MetricDataQueries: []*cloudwatch.MetricDataQuery{
			{
				Id: aws.String("c1"),
				MetricStat: &cloudwatch.MetricStat{
					Metric: &cloudwatch.Metric{
						Namespace: aws.String(c.metadata.namespace),
						Dimensions: []*cloudwatch.Dimension{
							{
								Name:  aws.String(c.metadata.dimensionName),
								Value: aws.String(c.metadata.dimensionValue),
							},
						},
						MetricName: aws.String(c.metadata.metricsName),
					},
					Period: aws.Int64(c.metadata.metricStatPeriod),
					Stat:   aws.String(c.metadata.metricStat),
				},
				ReturnData: aws.Bool(true),
			},
		},
	}

	output, err := cloudwatchClient.GetMetricData(&input)

	if err != nil {
		cloudwatchLog.Error(err, "Failed to get output")
		return -1, err
	}

	cloudwatchLog.V(1).Info("Received Metric Data", "data", output)
	var metricValue float64
	if output.MetricDataResults[0].Values != nil {
		metricValue = *output.MetricDataResults[0].Values[0]
	} else {
		return -1, fmt.Errorf("metric data not received")
	}

	return metricValue, nil
}
