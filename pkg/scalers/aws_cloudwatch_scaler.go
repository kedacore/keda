package scalers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	awsutils "github.com/kedacore/keda/v2/pkg/scalers/aws"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
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
	cwClient   cloudwatch.GetMetricDataAPIClient
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

	awsRegion   string
	awsEndpoint string

	awsAuthorization awsutils.AuthorizationMetadata

	triggerIndex int
}

// NewAwsCloudwatchScaler creates a new awsCloudwatchScaler
func NewAwsCloudwatchScaler(ctx context.Context, config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, err := parseAwsCloudwatchMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing cloudwatch metadata: %w", err)
	}

	cloudwatchClient, err := createCloudwatchClient(ctx, meta)
	if err != nil {
		return nil, fmt.Errorf("error creating cloudwatch client: %w", err)
	}
	return &awsCloudwatchScaler{
		metricType: metricType,
		metadata:   meta,
		cwClient:   cloudwatchClient,
		logger:     InitializeLogger(config, "aws_cloudwatch_scaler"),
	}, nil
}

func getIntMetadataValue(metadata map[string]string, key string, required bool, defaultValue int64) (int64, error) {
	if val, ok := metadata[key]; ok && val != "" {
		value, err := strconv.Atoi(val)
		if err != nil {
			return 0, fmt.Errorf("error parsing %s metadata: %w", key, err)
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
			return 0, fmt.Errorf("error parsing %s metadata: %w", key, err)
		}
		return value, nil
	}

	if required {
		return 0, fmt.Errorf("metadata %s not given", key)
	}

	return defaultValue, nil
}

func createCloudwatchClient(ctx context.Context, metadata *awsCloudwatchMetadata) (*cloudwatch.Client, error) {
	cfg, err := awsutils.GetAwsConfig(ctx, metadata.awsRegion, metadata.awsAuthorization)

	if err != nil {
		return nil, err
	}
	return cloudwatch.NewFromConfig(*cfg, func(options *cloudwatch.Options) {
		if metadata.awsEndpoint != "" {
			options.BaseEndpoint = aws.String(metadata.awsEndpoint)
		}
	}), nil
}

func parseAwsCloudwatchMetadata(config *scalersconfig.ScalerConfig) (*awsCloudwatchMetadata, error) {
	var err error
	meta := awsCloudwatchMetadata{}

	if config.TriggerMetadata["expression"] != "" {
		if val, ok := config.TriggerMetadata["expression"]; ok && val != "" {
			meta.expression = val
		} else {
			return nil, fmt.Errorf("expression not given")
		}
	} else {
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

		meta.metricUnit = config.TriggerMetadata["metricUnit"]
		if err = checkMetricUnit(meta.metricUnit); err != nil {
			return nil, err
		}
	}

	targetMetricValue, err := getFloatMetadataValue(config.TriggerMetadata, "targetMetricValue", true, 0)
	if err != nil {
		return nil, err
	}
	meta.targetMetricValue = targetMetricValue

	activationTargetMetricValue, err := getFloatMetadataValue(config.TriggerMetadata, "activationTargetMetricValue", false, 0)
	if err != nil {
		return nil, err
	}
	meta.activationTargetMetricValue = activationTargetMetricValue

	minMetricValue, err := getFloatMetadataValue(config.TriggerMetadata, "minMetricValue", true, 0)
	if err != nil {
		return nil, err
	}
	meta.minMetricValue = minMetricValue

	meta.metricStat = defaultMetricStat
	if val, ok := config.TriggerMetadata["metricStat"]; ok && val != "" {
		meta.metricStat = val
	}
	if err = checkMetricStat(meta.metricStat); err != nil {
		return nil, err
	}

	metricStatPeriod, err := getIntMetadataValue(config.TriggerMetadata, "metricStatPeriod", false, defaultMetricStatPeriod)
	if err != nil {
		return nil, err
	}
	meta.metricStatPeriod = metricStatPeriod

	if err = checkMetricStatPeriod(meta.metricStatPeriod); err != nil {
		return nil, err
	}

	metricCollectionTime, err := getIntMetadataValue(config.TriggerMetadata, "metricCollectionTime", false, defaultMetricCollectionTime)
	if err != nil {
		return nil, err
	}
	meta.metricCollectionTime = metricCollectionTime

	if meta.metricCollectionTime < 0 || meta.metricCollectionTime%meta.metricStatPeriod != 0 {
		return nil, fmt.Errorf("metricCollectionTime must be greater than 0 and a multiple of metricStatPeriod(%d), %d is given", meta.metricStatPeriod, meta.metricCollectionTime)
	}

	metricEndTimeOffset, err := getIntMetadataValue(config.TriggerMetadata, "metricEndTimeOffset", false, defaultMetricEndTimeOffset)
	if err != nil {
		return nil, err
	}
	meta.metricEndTimeOffset = metricEndTimeOffset

	if val, ok := config.TriggerMetadata["awsRegion"]; ok && val != "" {
		meta.awsRegion = val
	} else {
		return nil, fmt.Errorf("no awsRegion given")
	}

	if val, ok := config.TriggerMetadata["awsEndpoint"]; ok {
		meta.awsEndpoint = val
	}

	awsAuthorization, err := awsutils.GetAwsAuthorization(config.TriggerUniqueKey, config.PodIdentity, config.TriggerMetadata, config.AuthParams, config.ResolvedEnv)
	if err != nil {
		return nil, err
	}
	meta.awsAuthorization = awsAuthorization

	meta.triggerIndex = config.TriggerIndex

	return &meta, nil
}

func checkMetricStat(stat string) error {
	for _, s := range types.Statistic("").Values() {
		if s == types.Statistic(stat) {
			return nil
		}
	}
	return fmt.Errorf("metricStat '%s' is not one of %v", stat, types.Statistic("").Values())
}

func checkMetricUnit(unit string) error {
	if unit == "" {
		return nil
	}
	for _, s := range types.StandardUnit("").Values() {
		if s == types.StandardUnit(unit) {
			return nil
		}
	}
	return fmt.Errorf("metricUnit '%s' is not one of %v", unit, types.StandardUnit("").Values())
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

func (s *awsCloudwatchScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	metricValue, err := s.GetCloudwatchMetrics(ctx)

	if err != nil {
		s.logger.Error(err, "Error getting metric value")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, metricValue)

	return []external_metrics.ExternalMetricValue{metric}, metricValue > s.metadata.activationTargetMetricValue, nil
}

func (s *awsCloudwatchScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, "aws-cloudwatch"),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.targetMetricValue),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

func (s *awsCloudwatchScaler) Close(context.Context) error {
	awsutils.ClearAwsConfig(s.metadata.awsAuthorization)
	return nil
}

func (s *awsCloudwatchScaler) GetCloudwatchMetrics(ctx context.Context) (float64, error) {
	var input cloudwatch.GetMetricDataInput

	startTime, endTime := computeQueryWindow(time.Now(), s.metadata.metricStatPeriod, s.metadata.metricEndTimeOffset, s.metadata.metricCollectionTime)

	if s.metadata.expression != "" {
		input = cloudwatch.GetMetricDataInput{
			StartTime: aws.Time(startTime),
			EndTime:   aws.Time(endTime),
			ScanBy:    types.ScanByTimestampDescending,
			MetricDataQueries: []types.MetricDataQuery{
				{
					Expression: aws.String(s.metadata.expression),
					Id:         aws.String("q1"),
					Period:     aws.Int32(int32(s.metadata.metricStatPeriod)),
				},
			},
		}
	} else {
		var dimensions []types.Dimension
		for i := range s.metadata.dimensionName {
			dimensions = append(dimensions, types.Dimension{
				Name:  &s.metadata.dimensionName[i],
				Value: &s.metadata.dimensionValue[i],
			})
		}

		var metricUnit string
		if s.metadata.metricUnit != "" {
			metricUnit = s.metadata.metricUnit
		}

		input = cloudwatch.GetMetricDataInput{
			StartTime: aws.Time(startTime),
			EndTime:   aws.Time(endTime),
			ScanBy:    types.ScanByTimestampDescending,
			MetricDataQueries: []types.MetricDataQuery{
				{
					Id: aws.String("c1"),
					MetricStat: &types.MetricStat{
						Metric: &types.Metric{
							Namespace:  aws.String(s.metadata.namespace),
							Dimensions: dimensions,
							MetricName: aws.String(s.metadata.metricsName),
						},
						Period: aws.Int32(int32(s.metadata.metricStatPeriod)),
						Stat:   aws.String(s.metadata.metricStat),
						Unit:   types.StandardUnit(metricUnit),
					},
					ReturnData: aws.Bool(true),
				},
			},
		}
	}

	output, err := s.cwClient.GetMetricData(ctx, &input)

	if err != nil {
		s.logger.Error(err, "Failed to get output")
		return -1, err
	}

	s.logger.V(1).Info("Received Metric Data", "data", output)
	var metricValue float64
	if len(output.MetricDataResults) > 0 && len(output.MetricDataResults[0].Values) > 0 {
		metricValue = output.MetricDataResults[0].Values[0]
	} else {
		s.logger.Info("empty metric data received, returning minMetricValue")
		metricValue = s.metadata.minMetricValue
	}

	return metricValue, nil
}
