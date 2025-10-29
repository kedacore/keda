package scalers

import (
	"context"
	"errors"
	"fmt"
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

type awsCloudwatchScaler struct {
	metricType v2.MetricTargetType
	metadata   *awsCloudwatchMetadata
	cwClient   cloudwatch.GetMetricDataAPIClient
	logger     logr.Logger
}

type awsCloudwatchMetadata struct {
	awsAuthorization awsutils.AuthorizationMetadata

	triggerIndex   int
	Namespace      string   `keda:"name=namespace,      order=triggerMetadata, optional"`
	MetricsName    string   `keda:"name=metricName,     order=triggerMetadata, optional"`
	DimensionName  []string `keda:"name=dimensionName,  order=triggerMetadata, optional, separator=;"`
	DimensionValue []string `keda:"name=dimensionValue, order=triggerMetadata, optional, separator=;"`
	Expression     string   `keda:"name=expression,     order=triggerMetadata, optional"`

	TargetMetricValue           float64 `keda:"name=targetMetricValue,           order=triggerMetadata"`
	ActivationTargetMetricValue float64 `keda:"name=activationTargetMetricValue, order=triggerMetadata, optional"`
	MinMetricValue              float64 `keda:"name=minMetricValue,              order=triggerMetadata"`
	IgnoreNullValues            bool    `keda:"name=ignoreNullValues,            order=triggerMetadata, default=true"`

	MetricCollectionTime int64  `keda:"name=metricCollectionTime, order=triggerMetadata, default=300"`
	MetricStat           string `keda:"name=metricStat,           order=triggerMetadata, default=Average"`
	MetricUnit           string `keda:"name=metricUnit,           order=triggerMetadata, optional"` // Need to check the metric unit
	MetricStatPeriod     int64  `keda:"name=metricStatPeriod,     order=triggerMetadata, default=300"`
	MetricEndTimeOffset  int64  `keda:"name=metricEndTimeOffset,  order=triggerMetadata, default=0"`

	AwsRegion   string `keda:"name=awsRegion,   order=triggerMetadata;authParams"`
	AwsEndpoint string `keda:"name=awsEndpoint, order=triggerMetadata, optional"`

	IdentityOwner string `keda:"name=identityOwner, order=triggerMetadata, optional"`
}

func (a *awsCloudwatchMetadata) Validate() error {
	var err error
	if a.Expression == "" {
		if a.Namespace == "" {
			return errors.New("namespace not given")
		}

		if a.MetricsName == "" {
			return errors.New("metric name not given")
		}

		if a.DimensionName == nil {
			return errors.New("dimension name not given")
		}

		if a.DimensionValue == nil {
			return errors.New("dimension value not given")
		}

		if len(a.DimensionName) != len(a.DimensionValue) {
			return errors.New("dimensionName and dimensionValue are not matching in size")
		}

		if err = checkMetricUnit(a.MetricUnit); err != nil {
			return err
		}
	}

	if err = checkMetricStatPeriod(a.MetricStatPeriod); err != nil {
		return err
	}
	if a.MetricCollectionTime < 0 || a.MetricCollectionTime%a.MetricStatPeriod != 0 {
		return fmt.Errorf("metricCollectionTime must be greater than 0 and a multiple of metricStatPeriod(%d), %d is given", a.MetricStatPeriod, a.MetricCollectionTime)
	}

	return nil
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
	cfg, err := awsutils.GetAwsConfig(ctx, metadata.awsAuthorization)

	if err != nil {
		return nil, err
	}
	return cloudwatch.NewFromConfig(*cfg, func(options *cloudwatch.Options) {
		if metadata.AwsEndpoint != "" {
			options.BaseEndpoint = aws.String(metadata.AwsEndpoint)
		}
	}), nil
}

func parseAwsCloudwatchMetadata(config *scalersconfig.ScalerConfig) (*awsCloudwatchMetadata, error) {
	meta := &awsCloudwatchMetadata{}
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing prometheus metadata: %w", err)
	}

	awsAuthorization, err := awsutils.GetAwsAuthorization(config.TriggerUniqueKey, meta.AwsRegion, config.PodIdentity, config.TriggerMetadata, config.AuthParams, config.ResolvedEnv)
	if err != nil {
		return nil, err
	}
	meta.awsAuthorization = awsAuthorization

	meta.triggerIndex = config.TriggerIndex

	return meta, nil
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

	return []external_metrics.ExternalMetricValue{metric}, metricValue > s.metadata.ActivationTargetMetricValue, nil
}

func (s *awsCloudwatchScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, "aws-cloudwatch"),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.TargetMetricValue),
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

	startTime, endTime := computeQueryWindow(time.Now(), s.metadata.MetricStatPeriod, s.metadata.MetricEndTimeOffset, s.metadata.MetricCollectionTime)

	if s.metadata.Expression != "" {
		input = cloudwatch.GetMetricDataInput{
			StartTime: aws.Time(startTime),
			EndTime:   aws.Time(endTime),
			ScanBy:    types.ScanByTimestampDescending,
			MetricDataQueries: []types.MetricDataQuery{
				{
					Expression: aws.String(s.metadata.Expression),
					Id:         aws.String("q1"),
					Period:     aws.Int32(int32(s.metadata.MetricStatPeriod)),
				},
			},
		}
	} else {
		var dimensions []types.Dimension
		for i := range s.metadata.DimensionName {
			dimensions = append(dimensions, types.Dimension{
				Name:  &s.metadata.DimensionName[i],
				Value: &s.metadata.DimensionValue[i],
			})
		}

		var metricUnit string
		if s.metadata.MetricUnit != "" {
			metricUnit = s.metadata.MetricUnit
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
							Namespace:  aws.String(s.metadata.Namespace),
							Dimensions: dimensions,
							MetricName: aws.String(s.metadata.MetricsName),
						},
						Period: aws.Int32(int32(s.metadata.MetricStatPeriod)),
						Stat:   aws.String(s.metadata.MetricStat),
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

	// If no metric data results or the first result has no values, and ignoreNullValues is false,
	// the scaler should return an error to prevent any further scaling actions.
	if len(output.MetricDataResults) > 0 && len(output.MetricDataResults[0].Values) == 0 && !s.metadata.IgnoreNullValues {
		emptyMetricsErrMsg := "empty metric data received, ignoreNullValues is false, returning error"
		s.logger.Error(nil, emptyMetricsErrMsg)
		return -1, fmt.Errorf("%s", emptyMetricsErrMsg)
	}

	var metricValue float64

	if len(output.MetricDataResults) > 0 && len(output.MetricDataResults[0].Values) > 0 {
		metricValue = output.MetricDataResults[0].Values[0]
	} else {
		s.logger.Info("empty metric data received, returning minMetricValue")
		metricValue = s.metadata.MinMetricValue
	}
	return metricValue, nil
}
