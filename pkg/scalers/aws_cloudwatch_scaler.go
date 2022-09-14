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
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	defaultMetricCollectionTime = 300
	defaultMetricStat           = "Average"
	defaultMetricStatPeriod     = 300
	defaultMetricEndTimeOffset  = 0
)

type awsCloudwatchScaler struct {
	metricType v2.MetricTargetType
	metadata   *awsCloudwatchMetadata
	cwClient   cloudwatchiface.CloudWatchAPI
	logger     logr.Logger
}

type awsCloudwatchMetadata struct {
	namespace      string
	metricsName    string
	dimensionName  []string
	dimensionValue []string
	expression     string

	targetMetricValue           float64
	activationTargetMetricValue float64
	minMetricValue              float64

	metricCollectionTime int64
	metricStat           string
	metricUnit           string
	metricStatPeriod     int64
	metricEndTimeOffset  int64

	awsRegion string

	awsAuthorization awsAuthorizationMetadata

	scalerIndex int
}

// NewAwsCloudwatchScaler creates a new awsCloudwatchScaler
func NewAwsCloudwatchScaler(config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %s", err)
	}

	meta, err := parseAwsCloudwatchMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing cloudwatch metadata: %s", err)
	}

	return &awsCloudwatchScaler{
		metricType: metricType,
		metadata:   meta,
		cwClient:   createCloudwatchClient(meta),
		logger:     InitializeLogger(config, "aws_cloudwatch_scaler"),
	}, nil
}

func getIntMetadataValue(metadata map[string]string, key string, required bool, defaultValue int64) (int64, error) {
	if val, ok := metadata[key]; ok && val != "" {
		value, err := strconv.Atoi(val)
		if err != nil {
			return 0, fmt.Errorf("error parsing %s metadata: %v", key, err)
		}
		return int64(value), nil
	}

	if required {
		return 0, fmt.Errorf("metadata %s not given", key)
	}

	return defaultValue, nil
}

func getFloatMetadataValue(metadata map[string]string, key string, required bool, defaultValue float64) (float64, error) {
	if val, ok := metadata[key]; ok && val != "" {
		value, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0, fmt.Errorf("error parsing %s metadata: %v", key, err)
		}
		return value, nil
	}

	if required {
		return 0, fmt.Errorf("metadata %s not given", key)
	}

	return defaultValue, nil
}

func createCloudwatchClient(metadata *awsCloudwatchMetadata) *cloudwatch.CloudWatch {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(metadata.awsRegion),
	}))

	var cloudwatchClient *cloudwatch.CloudWatch
	if metadata.awsAuthorization.podIdentityOwner {
		creds := credentials.NewStaticCredentials(metadata.awsAuthorization.awsAccessKeyID, metadata.awsAuthorization.awsSecretAccessKey, metadata.awsAuthorization.awsSessionToken)

		if metadata.awsAuthorization.awsRoleArn != "" {
			creds = stscreds.NewCredentials(sess, metadata.awsAuthorization.awsRoleArn)
		}

		cloudwatchClient = cloudwatch.New(sess, &aws.Config{
			Region:      aws.String(metadata.awsRegion),
			Credentials: creds,
		})
	} else {
		cloudwatchClient = cloudwatch.New(sess, &aws.Config{
			Region: aws.String(metadata.awsRegion),
		})
	}

	return cloudwatchClient
}

func parseAwsCloudwatchMetadata(config *ScalerConfig) (*awsCloudwatchMetadata, error) {
	var err error
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

	if config.TriggerMetadata["expression"] != "" {
		if val, ok := config.TriggerMetadata["expression"]; ok && val != "" {
			meta.expression = val
		} else {
			return nil, fmt.Errorf("expression not given")
		}
	} else {
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

		meta.metricUnit = config.TriggerMetadata["metricUnit"]
		if err = checkMetricUnit(meta.metricUnit); err != nil {
			return nil, err
		}
	}

	meta.targetMetricValue, err = getFloatMetadataValue(config.TriggerMetadata, "targetMetricValue", true, 0)
	if err != nil {
		return nil, err
	}

	meta.activationTargetMetricValue, err = getFloatMetadataValue(config.TriggerMetadata, "activationTargetMetricValue", false, 0)
	if err != nil {
		return nil, err
	}

	meta.minMetricValue, err = getFloatMetadataValue(config.TriggerMetadata, "minMetricValue", true, 0)
	if err != nil {
		return nil, err
	}

	meta.metricStat = defaultMetricStat
	if val, ok := config.TriggerMetadata["metricStat"]; ok && val != "" {
		meta.metricStat = val
	}
	if err = checkMetricStat(meta.metricStat); err != nil {
		return nil, err
	}

	meta.metricStatPeriod, err = getIntMetadataValue(config.TriggerMetadata, "metricStatPeriod", false, defaultMetricStatPeriod)
	if err != nil {
		return nil, err
	}

	if err = checkMetricStatPeriod(meta.metricStatPeriod); err != nil {
		return nil, err
	}

	meta.metricCollectionTime, err = getIntMetadataValue(config.TriggerMetadata, "metricCollectionTime", false, defaultMetricCollectionTime)
	if err != nil {
		return nil, err
	}

	if meta.metricCollectionTime < 0 || meta.metricCollectionTime%meta.metricStatPeriod != 0 {
		return nil, fmt.Errorf("metricCollectionTime must be greater than 0 and a multiple of metricStatPeriod(%d), %d is given", meta.metricStatPeriod, meta.metricCollectionTime)
	}

	meta.metricEndTimeOffset, err = getIntMetadataValue(config.TriggerMetadata, "metricEndTimeOffset", false, defaultMetricEndTimeOffset)
	if err != nil {
		return nil, err
	}

	if val, ok := config.TriggerMetadata["awsRegion"]; ok && val != "" {
		meta.awsRegion = val
	} else {
		return nil, fmt.Errorf("no awsRegion given")
	}

	meta.awsAuthorization, err = getAwsAuthorization(config.AuthParams, config.TriggerMetadata, config.ResolvedEnv)
	if err != nil {
		return nil, err
	}

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

func checkMetricStatPeriod(period int64) error {
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

	if period%60 != 0 {
		return fmt.Errorf("metricStatPeriod >= 60 has to be a multiple of 60, however, %d is provided", period)
	}

	return nil
}

func computeQueryWindow(current time.Time, metricPeriodSec, metricEndTimeOffsetSec, metricCollectionTimeSec int64) (startTime, endTime time.Time) {
	endTime = current.Add(time.Second * -1 * time.Duration(metricEndTimeOffsetSec)).Truncate(time.Duration(metricPeriodSec) * time.Second)
	startTime = endTime.Add(time.Second * -1 * time.Duration(metricCollectionTimeSec))
	return
}

func (s *awsCloudwatchScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	metricValue, err := s.GetCloudwatchMetrics()

	if err != nil {
		s.logger.Error(err, "Error getting metric value")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := GenerateMetricInMili(metricName, metricValue)

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

func (s *awsCloudwatchScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	var metricNameSuffix string

	if s.metadata.expression != "" {
		metricNameSuffix = s.metadata.metricsName
	} else {
		metricNameSuffix = s.metadata.dimensionName[0]
	}

	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, kedautil.NormalizeString(fmt.Sprintf("aws-cloudwatch-%s", metricNameSuffix))),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.targetMetricValue),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

func (s *awsCloudwatchScaler) IsActive(ctx context.Context) (bool, error) {
	val, err := s.GetCloudwatchMetrics()

	if err != nil {
		return false, err
	}

	return val > s.metadata.activationTargetMetricValue, nil
}

func (s *awsCloudwatchScaler) Close(context.Context) error {
	return nil
}

func (s *awsCloudwatchScaler) GetCloudwatchMetrics() (float64, error) {
	var input cloudwatch.GetMetricDataInput

	startTime, endTime := computeQueryWindow(time.Now(), s.metadata.metricStatPeriod, s.metadata.metricEndTimeOffset, s.metadata.metricCollectionTime)

	if s.metadata.expression != "" {
		input = cloudwatch.GetMetricDataInput{
			StartTime: aws.Time(startTime),
			EndTime:   aws.Time(endTime),
			ScanBy:    aws.String(cloudwatch.ScanByTimestampDescending),
			MetricDataQueries: []*cloudwatch.MetricDataQuery{
				{
					Expression: aws.String(s.metadata.expression),
					Id:         aws.String("q1"),
					Period:     aws.Int64(s.metadata.metricStatPeriod),
					Label:      aws.String(s.metadata.metricsName),
				},
			},
		}
	} else {
		dimensions := []*cloudwatch.Dimension{}
		for i := range s.metadata.dimensionName {
			dimensions = append(dimensions, &cloudwatch.Dimension{
				Name:  &s.metadata.dimensionName[i],
				Value: &s.metadata.dimensionValue[i],
			})
		}

		var metricUnit *string
		if s.metadata.metricUnit != "" {
			metricUnit = aws.String(s.metadata.metricUnit)
		}

		input = cloudwatch.GetMetricDataInput{
			StartTime: aws.Time(startTime),
			EndTime:   aws.Time(endTime),
			ScanBy:    aws.String(cloudwatch.ScanByTimestampDescending),
			MetricDataQueries: []*cloudwatch.MetricDataQuery{
				{
					Id: aws.String("c1"),
					MetricStat: &cloudwatch.MetricStat{
						Metric: &cloudwatch.Metric{
							Namespace:  aws.String(s.metadata.namespace),
							Dimensions: dimensions,
							MetricName: aws.String(s.metadata.metricsName),
						},
						Period: aws.Int64(s.metadata.metricStatPeriod),
						Stat:   aws.String(s.metadata.metricStat),
						Unit:   metricUnit,
					},
					ReturnData: aws.Bool(true),
				},
			},
		}
	}

	output, err := s.cwClient.GetMetricData(&input)

	if err != nil {
		s.logger.Error(err, "Failed to get output")
		return -1, err
	}

	s.logger.V(1).Info("Received Metric Data", "data", output)
	var metricValue float64
	if len(output.MetricDataResults) > 0 && len(output.MetricDataResults[0].Values) > 0 {
		metricValue = *output.MetricDataResults[0].Values[0]
	} else {
		s.logger.Info("empty metric data received, returning minMetricValue")
		metricValue = s.metadata.minMetricValue
	}

	return metricValue, nil
}
