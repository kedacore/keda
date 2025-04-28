package scalers

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-logr/logr"
	sdk "go.temporal.io/sdk/client"
	sdklog "go.temporal.io/sdk/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
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
	Endpoint                  string   `keda:"name=endpoint,                  order=triggerMetadata;resolvedEnv"`
	Namespace                 string   `keda:"name=namespace,                 order=triggerMetadata;resolvedEnv, default=default"`
	ActivationTargetQueueSize int64    `keda:"name=activationTargetQueueSize, order=triggerMetadata, default=0"`
	TargetQueueSize           int64    `keda:"name=targetQueueSize,           order=triggerMetadata, default=5"`
	TaskQueue                 string   `keda:"name=taskQueue,                 order=triggerMetadata;resolvedEnv"`
	QueueTypes                []string `keda:"name=queueTypes,                order=triggerMetadata, optional"`
	BuildID                   string   `keda:"name=buildId,                   order=triggerMetadata;resolvedEnv, optional"`
	AllActive                 bool     `keda:"name=selectAllActive,           order=triggerMetadata, default=false"`
	Unversioned               bool     `keda:"name=selectUnversioned,         order=triggerMetadata, default=false"`
	APIKey                    string   `keda:"name=apiKey,                    order=authParams;resolvedEnv, optional"`
	MinConnectTimeout         int      `keda:"name=minConnectTimeout,         order=triggerMetadata, default=5"`

	UnsafeSsl   bool   `keda:"name=unsafeSsl,                 order=triggerMetadata, optional"`
	Cert        string `keda:"name=cert,                      order=authParams, optional"`
	Key         string `keda:"name=key,                       order=authParams, optional"`
	KeyPassword string `keda:"name=keyPassword,               order=authParams, optional"`
	CA          string `keda:"name=ca,                        order=authParams, optional"`

	triggerIndex int
}

func (a *temporalMetadata) Validate() error {
	if a.TargetQueueSize <= 0 {
		return fmt.Errorf("targetQueueSize must be a positive number")
	}
	if a.ActivationTargetQueueSize < 0 {
		return fmt.Errorf("activationTargetQueueSize must be a positive number")
	}

	if (a.Cert == "") != (a.Key == "") {
		return fmt.Errorf("both cert and key must be provided when using TLS")
	}

	if a.MinConnectTimeout < 0 {
		return fmt.Errorf("minConnectTimeout must be a positive number")
	}

	return nil
}

func NewTemporalScaler(ctx context.Context, config *scalersconfig.ScalerConfig) (Scaler, error) {
	logger := InitializeLogger(config, "temporal_scaler")

	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get scaler metric type: %w", err)
	}

	meta, err := parseTemporalMetadata(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Temporal metadata: %w", err)
	}

	c, err := getTemporalClient(ctx, meta, logger)
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
	metricName := kedautil.NormalizeString(fmt.Sprintf("temporal-%s-%s", s.metadata.Namespace, s.metadata.TaskQueue))
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

	return []external_metrics.ExternalMetricValue{metric}, queueSize > s.metadata.ActivationTargetQueueSize, nil
}

func (s *temporalScaler) getQueueSize(ctx context.Context) (int64, error) {
	var selection *sdk.TaskQueueVersionSelection
	if s.metadata.AllActive || s.metadata.Unversioned || s.metadata.BuildID != "" {
		selection = &sdk.TaskQueueVersionSelection{
			AllActive:   s.metadata.AllActive,
			Unversioned: s.metadata.Unversioned,
			BuildIDs:    []string{s.metadata.BuildID},
		}
	}

	queueType := getQueueTypes(s.metadata.QueueTypes)

	resp, err := s.tcl.DescribeTaskQueueEnhanced(ctx, sdk.DescribeTaskQueueEnhancedOptions{
		TaskQueue:      s.metadata.TaskQueue,
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

	for _, versionInfo := range description.VersionsInfo { //nolint: staticcheck // Tracked by https://github.com/kedacore/keda/issues/6690
		for _, typeInfo := range versionInfo.TypesInfo {
			if typeInfo.Stats != nil {
				count += typeInfo.Stats.ApproximateBacklogCount
			}
		}
	}
	return count
}

func getTemporalClient(ctx context.Context, meta *temporalMetadata, log logr.Logger) (sdk.Client, error) {
	logHandler := logr.ToSlogHandler(log)
	options := sdk.Options{
		HostPort:  meta.Endpoint,
		Namespace: meta.Namespace,
		Logger:    sdklog.NewStructuredLogger(slog.New(logHandler)),
	}

	dialOptions := []grpc.DialOption{
		grpc.WithConnectParams(grpc.ConnectParams{
			MinConnectTimeout: time.Duration(meta.MinConnectTimeout) * time.Second,
		}),
	}

	if meta.APIKey != "" {
		dialOptions = append(dialOptions, grpc.WithUnaryInterceptor(
			func(ctx context.Context, method string, req any, reply any,
				cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
				return invoker(
					metadata.AppendToOutgoingContext(ctx, "temporal-namespace", meta.Namespace),
					method,
					req,
					reply,
					cc,
					opts...,
				)
			},
		))
		options.Credentials = sdk.NewAPIKeyStaticCredentials(meta.APIKey)
		options.ConnectionOptions.TLS = &tls.Config{
			MinVersion: tls.VersionTLS13,
		}
	}

	options.ConnectionOptions = sdk.ConnectionOptions{
		DialOptions: dialOptions,
	}

	if meta.Cert != "" && meta.Key != "" {
		tlsConfig, err := kedautil.NewTLSConfigWithPassword(meta.Cert, meta.Key, meta.KeyPassword, meta.CA, meta.UnsafeSsl)
		if err != nil {
			return nil, err
		}
		options.ConnectionOptions.TLS = tlsConfig
	}

	return sdk.DialContext(ctx, options)
}

func parseTemporalMetadata(config *scalersconfig.ScalerConfig, _ logr.Logger) (*temporalMetadata, error) {
	meta := &temporalMetadata{triggerIndex: config.TriggerIndex}
	if err := config.TypedConfig(meta); err != nil {
		return meta, fmt.Errorf("error parsing temporal metadata: %w", err)
	}

	return meta, nil
}
