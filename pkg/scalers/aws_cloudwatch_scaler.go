package scalers

import (
	"context"
	"fmt"
	"reflect"
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
	defaultMetricCollectionTime int64 = 300
	defaultMetricStat                 = "Average"
	defaultMetricStatPeriod     int64 = 300
	defaultMetricEndTimeOffset  int64 = 0
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

// getCloudWatchExpression lifts the metric query expression from the metadata, returning either the expression string,
// an empty string, or an error if the expression was not parseable.
func getCloudWatchExpression(config *scalersconfig.ScalerConfig) (string, error) {
	expression, err := getParameterFromConfigV2(config, "expression", reflect.TypeOf(""), IsOptional(true), WithDefaultVal(""), UseMetadata(true))
	if err != nil {
		return "", fmt.Errorf("error parsing expression: %w", err)
	}
	return expression.(string), nil
}

// parseAwsCloudwatchMetadata parses the metric query parameters from the metadata, returning the namespace, metric name,
// metric unit, dimension names, dimension values, or error if the metadata was not parseable.
func getCloudWatchMetricQuery(config *scalersconfig.ScalerConfig) (string, string, string, []string, []string, error) {
	// namespace isa required parameter, that is a single string
	namespaceUntyped, err := getParameterFromConfigV2(config, "namespace", reflect.TypeOf(""), IsOptional(false), UseMetadata(true))
	if err != nil {
		return "", "", "", nil, nil, fmt.Errorf("error parsing namespace: %w", err)
	}
	namespace := namespaceUntyped.(string)

	// metricName is a required parameter, that is a single string
	metricsNameUntyped, err := getParameterFromConfigV2(config, "metricName", reflect.TypeOf(""), IsOptional(false), UseMetadata(true))
	if err != nil {
		return "", "", "", nil, nil, fmt.Errorf("error parsing metric name: %w", err)
	}
	metricsName := metricsNameUntyped.(string)

	// metricUnit is an optional parameter, if not provided, it will default to an empty string
	// if provided, it must be one of the valid metric units
	metricUnitUntyped, err := getParameterFromConfigV2(config, "metricUnit", reflect.TypeOf(""), IsOptional(true), UseMetadata(true), WithDefaultVal(""))
	if err != nil {
		return "", "", "", nil, nil, fmt.Errorf("error parsing metric unit: %w", err)
	}
	metricUnit := metricUnitUntyped.(string)
	if err := checkMetricUnit(metricUnit); err != nil {
		return "", "", "", nil, nil, err
	}

	// dimensionName is a required parameter, which is a comma separated string
	dimensionNameUntyped, err := getParameterFromConfigV2(config, "dimensionName", reflect.TypeOf(""), IsOptional(false), UseMetadata(true))
	if err != nil {
		return "", "", "", nil, nil, fmt.Errorf("error parsing dimension name: %w", err)
	}
	dimensionNameString := dimensionNameUntyped.(string)
	dimensionName := strings.Split(dimensionNameString, ",")

	// dimensionValue is a required parameter, which is a comma separated string
	dimensionValueUntyped, err := getParameterFromConfigV2(config, "dimensionValue", reflect.TypeOf(""), IsOptional(false), UseMetadata(true))
	if err != nil {
		return "", "", "", nil, nil, fmt.Errorf("error parsing dimension value: %w", err)
	}
	dimensionValueString := dimensionValueUntyped.(string)
	dimensionValue := strings.Split(dimensionValueString, ",")

	// dimensionName and dimensionValue must be the same length to parse into a valid GetMetricDataInput type
	if len(dimensionName) != len(dimensionValue) {
		return "", "", "", nil, nil, fmt.Errorf("dimension name and value must be the same length")
	}

	return namespace, metricsName, metricUnit, dimensionName, dimensionValue, nil
}

// parseAwsCloudwatchMetadata parses the input for the scaler, and returns the metadata for the scaler
func parseAwsCloudwatchMetadata(config *scalersconfig.ScalerConfig) (*awsCloudwatchMetadata, error) {
	var err error
	meta := awsCloudwatchMetadata{}

	// try to get the expression first, either an expression or a metric query is required
	meta.expression, err = getCloudWatchExpression(config)
	if err != nil {
		return nil, err
	}

	// if the expression is empty, try to get the metric query. The parameters in the query are now
	// required, as the expression is not present.
	if meta.expression == "" {
		meta.namespace, meta.metricsName, meta.metricUnit, meta.dimensionName, meta.dimensionValue, err = getCloudWatchMetricQuery(config)
		if err != nil {
			return nil, err
		}
	}

	// targetMetricValue is a required parameter, that is a float
	targetMetricValue, err := getParameterFromConfigV2(config, "targetMetricValue", reflect.TypeOf(float64(0)), IsOptional(false), UseMetadata(true))
	if err != nil {
		return nil, err
	}
	meta.targetMetricValue = targetMetricValue.(float64)

	// activationTargetMetricValue is an optional parameter, that is a float, defaults to 0
	activationTargetMetricValue, err := getParameterFromConfigV2(config, "activationTargetMetricValue", reflect.TypeOf(float64(0)), IsOptional(true), UseMetadata(true), WithDefaultVal(float64(0)))
	if err != nil {
		return nil, err
	}
	meta.activationTargetMetricValue = activationTargetMetricValue.(float64)

	// minMetricValue is an optional parameter, that is a float, defaults to 0
	minMetricValue, err := getParameterFromConfigV2(config, "minMetricValue", reflect.TypeOf(float64(0)), IsOptional(true), UseMetadata(true), WithDefaultVal(float64(0)))
	if err != nil {
		return nil, err
	}
	meta.minMetricValue = minMetricValue.(float64)

	// metricStat is an optional parameter, that is a string, defaults to the average statistic
	metricStat, err := getParameterFromConfigV2(config, "metricStat", reflect.TypeOf(""), IsOptional(true), UseMetadata(true), WithDefaultVal(defaultMetricStat))
	if err != nil {
		return nil, err
	}
	meta.metricStat = metricStat.(string)

	// ensure metricStat is a valid statistic
	if err = checkMetricStat(meta.metricStat); err != nil {
		return nil, err
	}

	// metricStatPeriod is an optional parameter, that is an integer, defaults to 60
	metricStatPeriod, err := getParameterFromConfigV2(config, "metricStatPeriod", reflect.TypeOf(int64(0)), IsOptional(true), UseMetadata(true), WithDefaultVal(defaultMetricStatPeriod))
	if err != nil {
		return nil, err
	}
	meta.metricStatPeriod = metricStatPeriod.(int64)

	// ensure metricStatPeriod is a valid period
	if err = checkMetricStatPeriod(meta.metricStatPeriod); err != nil {
		return nil, err
	}

	// metricCollectionTime is an optional parameter, that is an integer, defaults to 300
	metricCollectionTime, err := getParameterFromConfigV2(config, "metricCollectionTime", reflect.TypeOf(int64(0)), IsOptional(true), UseMetadata(true), WithDefaultVal(defaultMetricCollectionTime))
	if err != nil {
		return nil, err
	}
	meta.metricCollectionTime = metricCollectionTime.(int64)

	// metricCollectionTime must be greater than 0 and a multiple of metricStatPeriod
	if meta.metricCollectionTime < 0 || meta.metricCollectionTime%meta.metricStatPeriod != 0 {
		return nil, fmt.Errorf("metricCollectionTime must be greater than 0 and a multiple of metricStatPeriod(%d), %d is given", meta.metricStatPeriod, meta.metricCollectionTime)
	}

	// metricEndTimeOffset is an optional parameter, that is an integer, defaults to 0
	metricEndTimeOffset, err := getParameterFromConfigV2(config, "metricEndTimeOffset", reflect.TypeOf(int64(0)), IsOptional(true), UseMetadata(true), WithDefaultVal(defaultMetricEndTimeOffset))
	if err != nil {
		return nil, err
	}
	meta.metricEndTimeOffset = metricEndTimeOffset.(int64)

	// awsRegion is a required parameter, that is a string
	awsRegion, err := getParameterFromConfigV2(config, "awsRegion", reflect.TypeOf(""), IsOptional(false), UseMetadata(true))
	if err != nil {
		return nil, err
	}
	meta.awsRegion = awsRegion.(string)

	// awsEndpoint is an optional parameter, that is a string
	awsEndpoint, err := getParameterFromConfigV2(config, "awsEndpoint", reflect.TypeOf(""), IsOptional(true), UseMetadata(true))
	if err != nil {
		return nil, err
	}
	meta.awsEndpoint = awsEndpoint.(string)

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
