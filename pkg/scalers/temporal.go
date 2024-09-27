package scalers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-logr/logr"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
	sdk "go.temporal.io/sdk/client"
	sdklog "go.temporal.io/sdk/log"
	"google.golang.org/grpc"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

var (
	temporalDefauleQueueTypes = []sdk.TaskQueueType{
		sdk.TaskQueueTypeActivity,
		sdk.TaskQueueTypeWorkflow,
		sdk.TaskQueueTypeNexus,
	}
)

type temporalScaler struct {
	metricType v2.MetricTargetType
	metadata   *temporalMetadata
	tcl        sdk.Client
	logger     logr.Logger
}

type temporalMetadata struct {
	ActivationLagThreshold int64    `keda:"name=activationTargetQueueSize, order=triggerMetadata, default=0"`
	Endpoint               string   `keda:"name=endpoint,       order=triggerMetadata;resolvedEnv"`
	Namespace              string   `keda:"name=namespace,      order=triggerMetadata, default=default"`
	TargetQueueSize        int64    `keda:"name=targetQueueSize, order=triggerMetadata, default=5"`
	QueueName              string   `keda:"name=queueName,      order=triggerMetadata"`
	QueueTypes             []string `keda:"name=queueTypes,      order=triggerMetadata, optional"`
	BuildIDs               []string `keda:"name=buildIds,      order=triggerMetadata, optional"`
	AllActive              bool     `keda:"name=selectAllActive,      order=triggerMetadata, default=true"`
	Unversioned            bool     `keda:"name=selectUnversioned,    order=triggerMetadata, default=true"`
	ApiKey                 string   `keda:"name=apiKey,         order=authParams;triggerMetadata, optional"`

	triggerIndex int
}

func (a *temporalMetadata) Validate() error {
	if a.TargetQueueSize <= 0 {
		return fmt.Errorf("targetQueueSize must be a positive number")
	}
	if a.ActivationLagThreshold < 0 {
		return fmt.Errorf("activationTargetQueueSize must be a positive number")
	}

	return nil
}

func NewTemporalScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	logger := InitializeLogger(config, "temporal_scaler")

	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get scaler metric type: %w", err)
	}

	meta, err := parseTemporalMetadata(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Temporal metadata: %w", err)
	}

	c, err := getTemporalClient(meta)
	if err != nil {
		return nil, fmt.Errorf("failed to create Temporal client connection: %w", err)
	}

	return &temporalScaler{
		metricType: metricType,
		metadata:   meta,
		tcl:        c,
		logger:     logger,
	}, nil
}

func (s *temporalScaler) Close(_ context.Context) error {
	if s.tcl != nil {
		s.tcl.Close()
	}
	return nil
}

func (s *temporalScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	metricName := kedautil.NormalizeString(fmt.Sprintf("temporal-%s-%s", s.metadata.Namespace, s.metadata.QueueName))
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, metricName),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.TargetQueueSize),
	}

	metricSpec := v2.MetricSpec{
		External: externalMetric,
		Type:     externalMetricType,
	}

	return []v2.MetricSpec{metricSpec}
}

func (s *temporalScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	queueSize, err := s.getQueueSize(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get Temporal queue size: %w", err)
	}

	metric := GenerateMetricInMili(metricName, float64(queueSize))

	return []external_metrics.ExternalMetricValue{metric}, queueSize > s.metadata.ActivationLagThreshold, nil
}

func (s *temporalScaler) getQueueSize(ctx context.Context) (int64, error) {
	var selection *sdk.TaskQueueVersionSelection
	if s.metadata.AllActive || s.metadata.Unversioned || len(s.metadata.BuildIDs) > 0 {
		selection = &sdk.TaskQueueVersionSelection{
			AllActive:   s.metadata.AllActive,
			Unversioned: s.metadata.Unversioned,
			BuildIDs:    s.metadata.BuildIDs,
		}
	}

	queueType := getQueueTypes(s.metadata.QueueTypes)

	resp, err := s.tcl.DescribeTaskQueueEnhanced(ctx, sdk.DescribeTaskQueueEnhancedOptions{
		TaskQueue:      s.metadata.QueueName,
		ReportStats:    true,
		Versions:       selection,
		TaskQueueTypes: queueType,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to get Temporal queue size: %w", err)
	}

	return getCombinedBacklogCount(resp), nil
}

func getQueueTypes(queueTypes []string) []sdk.TaskQueueType {
	var taskQueueTypes []sdk.TaskQueueType
	for _, t := range queueTypes {
		var taskQueueType sdk.TaskQueueType
		switch t {
		case "workflow":
			taskQueueType = sdk.TaskQueueTypeWorkflow
		case "activity":
			taskQueueType = sdk.TaskQueueTypeActivity
		case "nexus":
			taskQueueType = sdk.TaskQueueTypeNexus
		}
		taskQueueTypes = append(taskQueueTypes, taskQueueType)
	}

	if len(taskQueueTypes) == 0 {
		return temporalDefauleQueueTypes
	}
	return taskQueueTypes
}

func getCombinedBacklogCount(description sdk.TaskQueueDescription) int64 {
	var count int64
	for _, versionInfo := range description.VersionsInfo {
		for _, typeInfo := range versionInfo.TypesInfo {
			if typeInfo.Stats != nil {
				count += typeInfo.Stats.ApproximateBacklogCount
			}
		}
	}
	return count
}

func getTemporalClient(meta *temporalMetadata) (sdk.Client, error) {
	return sdk.Dial(sdk.Options{
		HostPort:  meta.Endpoint,
		Namespace: meta.Namespace,
		Logger:    sdklog.NewStructuredLogger(slog.Default()),
		ConnectionOptions: sdk.ConnectionOptions{
			DialOptions: []grpc.DialOption{
				grpc.WithConnectParams(grpc.ConnectParams{
					MinConnectTimeout: 5 * time.Second,
				}),
			},
		},
	})
}

func parseTemporalMetadata(config *scalersconfig.ScalerConfig, _ logr.Logger) (*temporalMetadata, error) {
	meta := &temporalMetadata{triggerIndex: config.TriggerIndex}
	if err := config.TypedConfig(meta); err != nil {
		return meta, fmt.Errorf("error parsing temporal metadata: %w", err)
	}

	return meta, nil
}
