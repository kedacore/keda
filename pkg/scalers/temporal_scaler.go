package scalers

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-logr/logr"
	deploymentpb "go.temporal.io/api/deployment/v1"
	enumspb "go.temporal.io/api/enums/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
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
	WorkerDeploymentName      string   `keda:"name=workerDeploymentName,      order=triggerMetadata;resolvedEnv, optional"`
	WorkerDeploymentBuildID   string   `keda:"name=workerDeploymentBuildId,   order=triggerMetadata;resolvedEnv, optional"`
	AllActive                 bool     `keda:"name=selectAllActive,           order=triggerMetadata, default=false"`
	Unversioned               bool     `keda:"name=selectUnversioned,         order=triggerMetadata, default=false"`
	APIKey                    string   `keda:"name=apiKey,                    order=authParams;resolvedEnv, optional"`
	MinConnectTimeout         int      `keda:"name=minConnectTimeout,         order=triggerMetadata, default=5"`

	UnsafeSsl     bool   `keda:"name=unsafeSsl,                 order=triggerMetadata, optional"`
	Cert          string `keda:"name=cert,                      order=authParams, optional"`
	Key           string `keda:"name=key,                       order=authParams, optional"`
	KeyPassword   string `keda:"name=keyPassword,               order=authParams, optional"`
	CA            string `keda:"name=ca,                        order=authParams, optional"`
	TLSServerName string `keda:"name=tlsServerName,             order=triggerMetadata, optional"`

	triggerIndex int
}

func (a *temporalMetadata) Validate() error {
	if a.TargetQueueSize < 0 {
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

	if a.WorkerDeploymentName != "" || a.WorkerDeploymentBuildID != "" {
		if a.WorkerDeploymentName == "" || a.WorkerDeploymentBuildID == "" {
			return fmt.Errorf("workerDeploymentName and workerDeploymentBuildId must both be set")
		}
		if a.BuildID != "" || a.AllActive || a.Unversioned {
			return fmt.Errorf("workerDeploymentName/workerDeploymentBuildId cannot be combined with buildId, selectAllActive, or selectUnversioned")
		}
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

	if meta.BuildID != "" {
		logger.Info("Warning: buildId is deprecated because Temporal Server will soon stop supporting the deprecated Rules-Based Versioning APIs, use workerDeploymentName and workerDeploymentBuildId instead")
	}
	if meta.AllActive {
		logger.Info("Warning: selectAllActive is deprecated because Temporal Server will soon stop supporting the deprecated Rules-Based Versioning APIs, use workerDeploymentName and workerDeploymentBuildId instead")
	}
	if meta.Unversioned {
		logger.Info("Warning: selectUnversioned is deprecated because Temporal Server will soon stop supporting the deprecated Rules-Based Versioning APIs, use workerDeploymentName and workerDeploymentBuildId instead")
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
	metricName := fmt.Sprintf("temporal-%s-%s", s.metadata.Namespace, s.metadata.TaskQueue)
	if s.metadata.WorkerDeploymentName != "" {
		metricName = fmt.Sprintf("%s-%s-%s", metricName, s.metadata.WorkerDeploymentName, s.metadata.WorkerDeploymentBuildID)
	} else if s.metadata.BuildID != "" {
		metricName = fmt.Sprintf("%s-%s", metricName, s.metadata.BuildID)
	}
	metricName = kedautil.NormalizeString(metricName)

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
	var (
		queueSize int64
		err       error
	)
	switch {
	case s.metadata.WorkerDeploymentName != "":
		queueSize, err = s.getDeploymentBacklogCount(ctx)
	case s.metadata.BuildID != "" || s.metadata.AllActive || s.metadata.Unversioned:
		queueSize, err = s.getQueueSize(ctx)
	default:
		queueSize, err = s.getUnversionedQueueSize(ctx)
	}
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

// getUnversionedQueueSize queries DescribeTaskQueue for each configured queue
// type and returns the summed backlog count. This replaces DescribeTaskQueueEnhanced
// for the unversioned path, which is deprecated and returns the default Build ID's
// backlog when no Versions selector is provided (not the unversioned queue).
func (s *temporalScaler) getUnversionedQueueSize(ctx context.Context) (int64, error) {
	var total int64
	for _, qt := range getQueueTypes(s.metadata.QueueTypes) {
		resp, err := s.tcl.WorkflowService().DescribeTaskQueue(ctx, &workflowservice.DescribeTaskQueueRequest{
			Namespace:     s.metadata.Namespace,
			TaskQueue:     &taskqueuepb.TaskQueue{Name: s.metadata.TaskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
			TaskQueueType: enumspb.TaskQueueType(qt),
			ReportStats:   true,
		})
		if err != nil {
			return 0, fmt.Errorf("failed to describe task queue: %w", err)
		}
		if stats := resp.GetStats(); stats != nil {
			total += stats.GetApproximateBacklogCount()
		}
	}
	return total, nil
}

func (s *temporalScaler) getDeploymentBacklogCount(ctx context.Context) (int64, error) {
	resp, err := s.tcl.WorkflowService().DescribeWorkerDeploymentVersion(ctx,
		&workflowservice.DescribeWorkerDeploymentVersionRequest{
			Namespace: s.metadata.Namespace,
			DeploymentVersion: &deploymentpb.WorkerDeploymentVersion{
				DeploymentName: s.metadata.WorkerDeploymentName,
				BuildId:        s.metadata.WorkerDeploymentBuildID,
			},
			ReportTaskQueueStats: true,
		},
	)
	if err != nil {
		return 0, fmt.Errorf("failed to describe worker deployment version: %w", err)
	}

	return sumDeploymentBacklog(resp.GetVersionTaskQueues(), s.metadata.TaskQueue, s.metadata.QueueTypes), nil
}

// sumDeploymentBacklog sums ApproximateBacklogCount across the provided task
// queues, optionally filtered by taskQueueName and allowedQueueTypes. An empty
// taskQueueName means no name filter; an empty allowedQueueTypes means no type filter.
func sumDeploymentBacklog(tqs []*workflowservice.DescribeWorkerDeploymentVersionResponse_VersionTaskQueue, taskQueueName string, allowedQueueTypes []string) int64 {
	allowedTypes := queueTypeSet(allowedQueueTypes) // nil == allow all

	var totalBacklog int64
	for _, tq := range tqs {
		if allowedTypes != nil && !allowedTypes[tq.GetType()] {
			continue
		}
		if taskQueueName != "" && tq.GetName() != taskQueueName {
			continue
		}
		if stats := tq.GetStats(); stats != nil {
			totalBacklog += stats.GetApproximateBacklogCount()
		}
	}
	return totalBacklog
}

func queueTypeSet(queueTypes []string) map[enumspb.TaskQueueType]bool {
	if len(queueTypes) == 0 {
		return nil
	}
	set := make(map[enumspb.TaskQueueType]bool, len(queueTypes))
	for _, t := range queueTypes {
		switch t {
		case "workflow":
			set[enumspb.TASK_QUEUE_TYPE_WORKFLOW] = true
		case "activity":
			set[enumspb.TASK_QUEUE_TYPE_ACTIVITY] = true
		case "nexus":
			set[enumspb.TASK_QUEUE_TYPE_NEXUS] = true
		}
	}
	return set
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

	dialOptions = append(dialOptions, grpc.WithUnaryInterceptor(
		func(ctx context.Context, method string, req any, reply any,
			cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption,
		) error {
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

	var tlsConfig *tls.Config

	if meta.APIKey != "" {
		options.Credentials = sdk.NewAPIKeyStaticCredentials(meta.APIKey)
		tlsConfig = kedautil.CreateTLSClientConfig(meta.UnsafeSsl)
	}

	if meta.Cert != "" && meta.Key != "" {
		var err error
		tlsConfig, err = kedautil.NewTLSConfigWithPassword(meta.Cert, meta.Key, meta.KeyPassword, meta.CA, meta.UnsafeSsl)
		if err != nil {
			return nil, err
		}
	}

	if tlsConfig != nil && meta.TLSServerName != "" {
		tlsConfig.ServerName = meta.TLSServerName
	}

	options.ConnectionOptions = sdk.ConnectionOptions{
		DialOptions: dialOptions,
		TLS:         tlsConfig,
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
