package scalers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"k8s.io/api/autoscaling/v2beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
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

	awsRegion          string
	awsAccessKeyID     string
	awsSecretAccessKey string
}

var cloudwatchLog = logf.Log.WithName("aws_cloudwatch_scaler")

// NewAwsCloudwatchScaler creates a new awsCloudwatchScaler
func NewAwsCloudwatchScaler(resolvedEnv, metadata map[string]string) (Scaler, error) {
	meta, err := parseAwsCloudwatchMetadata(metadata, resolvedEnv)
	if err != nil {
		return nil, fmt.Errorf("Error parsing Cloudwatch metadata: %s", err)
	}

	return &awsCloudwatchScaler{
		metadata: meta,
	}, nil
}

func parseAwsCloudwatchMetadata(metadata, resolvedEnv map[string]string) (*awsCloudwatchMetadata, error) {
	meta := awsCloudwatchMetadata{}
	meta.metricCollectionTime = defaultMetricCollectionTime
	meta.metricStat = defaultMetricStat
	meta.metricStatPeriod = defaultMetricStatPeriod

	if val, ok := metadata["namespace"]; ok && val != "" {
		meta.namespace = val
	} else {
		return nil, fmt.Errorf("Namespace not given")
	}

	if val, ok := metadata["metricName"]; ok && val != "" {
		meta.metricsName = val
	} else {
		return nil, fmt.Errorf("Metric Name not given")
	}

	if val, ok := metadata["dimensionName"]; ok && val != "" {
		meta.dimensionName = val
	} else {
		return nil, fmt.Errorf("Dimension Name not given")
	}

	if val, ok := metadata["dimensionValue"]; ok && val != "" {
		meta.dimensionValue = val
	} else {
		return nil, fmt.Errorf("Dimension Value not given")
	}

	if val, ok := metadata["targetMetricValue"]; ok && val != "" {
		targetMetricValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			cloudwatchLog.Error(err, "Error parsing targetMetricValue metadata")
		} else {
			meta.targetMetricValue = targetMetricValue
		}
	} else {
		return nil, fmt.Errorf("target Metric Value not given")
	}

	if val, ok := metadata["minMetricValue"]; ok && val != "" {
		minMetricValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			cloudwatchLog.Error(err, "Error parsing minMetricValue metadata")
		} else {
			meta.minMetricValue = minMetricValue
		}
	} else {
		return nil, fmt.Errorf("Min Metric Value not given")
	}

	if val, ok := metadata["metricCollectionTime"]; ok && val != "" {
		metricCollectionTime, err := strconv.Atoi(val)
		if err != nil {
			cloudwatchLog.Error(err, "Error parsing metricCollectionTime metadata")
		} else {
			meta.metricCollectionTime = int64(metricCollectionTime)
		}
	}

	if val, ok := metadata["metricStat"]; ok && val != "" {
		meta.metricStat = val
	}

	if val, ok := metadata["metricStatPeriod"]; ok && val != "" {
		metricStatPeriod, err := strconv.Atoi(val)
		if err != nil {
			cloudwatchLog.Error(err, "Error parsing metricStatPeriod metadata")
		} else {
			meta.metricStatPeriod = int64(metricStatPeriod)
		}
	}

	if val, ok := metadata["awsRegion"]; ok && val != "" {
		meta.awsRegion = val
	} else {
		return nil, fmt.Errorf("no awsRegion given")
	}

	accessIDKey := awsAccessKeyIDEnvVar
	if val, ok := metadata["awsAccessKeyID"]; ok && val != "" {
		accessIDKey = val
	}

	if val, ok := resolvedEnv[accessIDKey]; ok && val != "" {
		meta.awsAccessKeyID = val
	} else {
		return nil, fmt.Errorf("cannot find awsAccessKeyId named %s in pod environment", accessIDKey)
	}

	secretAccessKey := awsSecretAccessKeyEnvVar
	if val, ok := metadata["awsSecretAccessKey"]; ok && val != "" {
		secretAccessKey = val
	}

	if val, ok := resolvedEnv[secretAccessKey]; ok && val != "" {
		meta.awsSecretAccessKey = val
	} else {
		return nil, fmt.Errorf("cannot find awsSecretAccessKey named %s in pod environment", secretAccessKey)
	}

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

func (c *awsCloudwatchScaler) GetMetricSpecForScaling() []v2beta1.MetricSpec {
	targetMetricValue := resource.NewQuantity(int64(c.metadata.targetMetricValue), resource.DecimalSI)
	externalMetric := &v2beta1.ExternalMetricSource{MetricName: fmt.Sprintf("%s-%s-%s", strings.ReplaceAll(c.metadata.namespace, "/", "-"),
		c.metadata.dimensionName, c.metadata.dimensionValue),
		TargetAverageValue: targetMetricValue}
	metricSpec := v2beta1.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2beta1.MetricSpec{metricSpec}
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
	creds := credentials.NewStaticCredentials(c.metadata.awsAccessKeyID, c.metadata.awsSecretAccessKey, "")
	sess := session.New(&aws.Config{
		Region:      aws.String(c.metadata.awsRegion),
		Credentials: creds,
	})

	cloudwatchClient := cloudwatch.New(sess)
	cloudwatchLog.Info(cloudwatch.New(sess).ServiceName)

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
		return -1, fmt.Errorf("Metric Data not received")
	}

	return metricValue, nil
}
