package scalers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
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
	dimensionName  []string
	dimensionValue []string

	targetMetricValue float64
	minMetricValue    float64

	metricCollectionTime int64
	metricStat           string
	metricUnit           string
	metricStatPeriod     int64
	metricEndTimeOffset  int64

	awsRegion string

	awsAuthorization awsAuthorizationMetadata

	scalerIndex int
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
		meta.dimensionName = strings.Split(val, ";")
	} else {
		return nil, fmt.Errorf("dimension name not given")
	}

	if val, ok := config.TriggerMetadata["dimensionValue"]; ok && val != "" {
		meta.dimensionValue = strings.Split(val, ";")
	} else {
		return nil, fmt.Errorf("dimension value not given")
	}

	if len(meta.dimensionName) != len(meta.dimensionValue) {
		return nil, fmt.Errorf("dimensionName and dimensionValue are not matching in size")
	}

	if val, ok := config.TriggerMetadata["targetMetricValue"]; ok && val != "" {
		targetMetricValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing targetMetricValue metadata: %v", err)
		} else {
			meta.targetMetricValue = targetMetricValue
		}
	} else {
		return nil, fmt.Errorf("target Metric Value not given")
	}

	if val, ok := config.TriggerMetadata["minMetricValue"]; ok && val != "" {
		minMetricValue, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing minMetricValue metadata: %v", err)
		} else {
			meta.minMetricValue = minMetricValue
		}
	} else {
		return nil, fmt.Errorf("min metric value not given")
	}

	if val, ok := config.TriggerMetadata["metricStat"]; ok && val != "" {
		if err := checkMetricStat(val); err != nil {
			return nil, err
		}
		meta.metricStat = val
	} else {
		meta.metricStat = defaultMetricStat
	}

	if val, ok := config.TriggerMetadata["metricStatPeriod"]; ok && val != "" {
		metricStatPeriod, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("error parsing metricStatPeriod metadata: %v", err)
		}
		if err := checkMetricStatPeriod(metricStatPeriod); err != nil {
			return nil, err
		} else {
			meta.metricStatPeriod = int64(metricStatPeriod)
		}
	} else {
		meta.metricStatPeriod = defaultMetricStatPeriod
	}

	if val, ok := config.TriggerMetadata["metricCollectionTime"]; ok && val != "" {
		metricCollectionTime, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("error parsing metricCollectionTime metadata: %v", err)
		}
		meta.metricCollectionTime = int64(metricCollectionTime)
	} else {
		meta.metricCollectionTime = defaultMetricCollectionTime
	}

	if meta.metricCollectionTime < 0 || meta.metricCollectionTime%meta.metricStatPeriod != 0 {
		return nil, fmt.Errorf("metricCollectionTime must be greater than 0 and a multiple of metricStatPeriod(%d), %d is given", meta.metricStatPeriod, meta.metricCollectionTime)
	}

	if val, ok := config.TriggerMetadata["metricEndTimeOffset"]; ok && val != "" {
		metricEndTimeOffset, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("error parsing metricEndTimeOffset metadata: %v", err)
		} else {
			meta.metricEndTimeOffset = int64(metricEndTimeOffset)
		}
	}

	if val, ok := config.TriggerMetadata["metricUnit"]; ok && val != "" {
		if err := checkMetricUnit(val); err != nil {
			return nil, err
		}
		meta.metricUnit = val
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

func checkMetricStat(stat string) error {
	for _, s := range cloudwatch.Statistic_Values() {
		if stat == s {
			return nil
		}
	}
	return fmt.Errorf("metricStat '%s' is not one of %v", stat, cloudwatch.Statistic_Values())
}

func checkMetricUnit(unit string) error {
	if unit == "" {
		return nil
	}
	for _, u := range cloudwatch.StandardUnit_Values() {
		if unit == u {
			return nil
		}
	}
	return fmt.Errorf("metricUnit '%s' is not one of %v", unit, cloudwatch.StandardUnit_Values())
}

func checkMetricStatPeriod(period int) error {
	if period < 1 {
		return fmt.Errorf("metricStatPeriod can not be smaller than 1, however, %d is provided", period)
	} else if period <= 60 {
		switch period {
		case 1, 5, 10, 30, 60:
			return nil
		default:
			return fmt.Errorf("metricStatPeriod < 60 has to be one of [1, 5, 10, 30], however, %d is provided", period)
		}
	}

	if period % 60 != 0 {
		return fmt.Errorf("metricStatPeriod >= 60 has to be a multiple of 60, however, %d is provided", period)
	}

	return nil
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

func (c *awsCloudwatchScaler) GetMetricSpecForScaling(context.Context) []v2beta2.MetricSpec {
	targetMetricValue := resource.NewQuantity(int64(c.metadata.targetMetricValue), resource.DecimalSI)
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(c.metadata.scalerIndex, kedautil.NormalizeString(fmt.Sprintf("%s-%s-%s-%s", "aws-cloudwatch", c.metadata.namespace, c.metadata.dimensionName[0], c.metadata.dimensionValue[0]))),
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

func (c *awsCloudwatchScaler) Close(context.Context) error {
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

	dimensions := []*cloudwatch.Dimension{}
	for i := range c.metadata.dimensionName {
		dimensions = append(dimensions, &cloudwatch.Dimension{
			Name:  &c.metadata.dimensionName[i],
			Value: &c.metadata.dimensionValue[i],
		})
	}

	endTime := time.Now().Add(time.Second * -1 * time.Duration(c.metadata.metricEndTimeOffset)).Truncate(time.Duration(c.metadata.metricStatPeriod) * time.Second)
	startTime := endTime.Add(time.Second * -1 * time.Duration(c.metadata.metricCollectionTime))

	var metricUnit *string
	if c.metadata.metricUnit != "" {
		metricUnit = aws.String(c.metadata.metricUnit)
	}

	input := cloudwatch.GetMetricDataInput{
		StartTime: aws.Time(startTime),
		EndTime:   aws.Time(endTime),
		ScanBy:    aws.String(cloudwatch.ScanByTimestampDescending),
		MetricDataQueries: []*cloudwatch.MetricDataQuery{
			{
				Id: aws.String("c1"),
				MetricStat: &cloudwatch.MetricStat{
					Metric: &cloudwatch.Metric{
						Namespace:  aws.String(c.metadata.namespace),
						Dimensions: dimensions,
						MetricName: aws.String(c.metadata.metricsName),
					},
					Period: aws.Int64(c.metadata.metricStatPeriod),
					Stat:   aws.String(c.metadata.metricStat),
					Unit:   metricUnit,
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
