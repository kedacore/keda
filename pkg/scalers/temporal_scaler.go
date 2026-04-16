package scalers

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-logr/logr"
	deploymentpb "go.temporal.io/api/deployment/v1"
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

const (
	versioningTypeNone       = "none"
	versioningTypeBuildID    = "build-id"
	versioningTypeDeployment = "deployment"

	queueTypeWorkflow = "workflow"
	queueTypeActivity = "activity"
	queueTypeNexus    = "nexus"
)

var queueTypeMap = map[string]sdk.TaskQueueType{
	queueTypeWorkflow: sdk.TaskQueueTypeWorkflow,
	queueTypeActivity: sdk.TaskQueueTypeActivity,
	queueTypeNexus:    sdk.TaskQueueTypeNexus,
}

var temporalDefaultQueueTypes = []sdk.TaskQueueType{
	sdk.TaskQueueTypeActivity,
	sdk.TaskQueueTypeWorkflow,
	sdk.TaskQueueTypeNexus,
}

// temporalBacklogClient is the subset of sdk.Client used by the scaler,
// extracted as an interface for testability.
type temporalBacklogClient interface {
	DescribeTaskQueueEnhanced(ctx context.Context, options sdk.DescribeTaskQueueEnhancedOptions) (sdk.TaskQueueDescription, error)
	WorkflowService() workflowservice.WorkflowServiceClient
	Close()
}

type temporalScaler struct {
	metricType v2.MetricTargetType
	metadata   *temporalMetadata
	client     temporalBacklogClient
	logger     logr.Logger
}

type temporalMetadata struct {
	// Connection
	Endpoint          string `keda:"name=endpoint,          order=triggerMetadata;resolvedEnv"`
	Namespace         string `keda:"name=namespace,         order=triggerMetadata;resolvedEnv, default=default"`
	MinConnectTimeout int    `keda:"name=minConnectTimeout, order=triggerMetadata, default=5"`

	// Scaling
	TaskQueue                 string `keda:"name=taskQueue,                 order=triggerMetadata;resolvedEnv"`
	TargetQueueSize           int64  `keda:"name=targetQueueSize,           order=triggerMetadata, default=5"`
	ActivationTargetQueueSize int64  `keda:"name=activationTargetQueueSize, order=triggerMetadata, default=0"`

	// Versioning
	WorkerVersioningType string   `keda:"name=workerVersioningType, order=triggerMetadata, optional"`
	BuildID              string   `keda:"name=buildId,              order=triggerMetadata;resolvedEnv, optional"`
	DeploymentName       string   `keda:"name=deploymentName,       order=triggerMetadata;resolvedEnv, optional"`
	QueueTypes           []string `keda:"name=queueTypes,           order=triggerMetadata, optional"`

	// Auth / TLS
	APIKey        string `keda:"name=apiKey,        order=authParams;resolvedEnv, optional"`
	UnsafeSsl     bool   `keda:"name=unsafeSsl,     order=triggerMetadata, optional"`
	Cert          string `keda:"name=cert,          order=authParams, optional"`
	Key           string `keda:"name=key,           order=authParams, optional"`
	KeyPassword   string `keda:"name=keyPassword,   order=authParams, optional"`
	CA            string `keda:"name=ca,            order=authParams, optional"`
	TLSServerName string `keda:"name=tlsServerName, order=triggerMetadata, optional"`

	triggerIndex int
}

func (a *temporalMetadata) Validate() error {
	if a.TargetQueueSize < 0 {
		return fmt.Errorf("targetQueueSize must be a non-negative number")
	}
	if a.ActivationTargetQueueSize < 0 {
		return fmt.Errorf("activationTargetQueueSize must be a non-negative number")
	}
	if a.MinConnectTimeout < 0 {
		return fmt.Errorf("minConnectTimeout must be a non-negative number")
	}

	if err := a.validateTLS(); err != nil {
		return err
	}

	for _, qt := range a.QueueTypes {
		if _, ok := queueTypeMap[qt]; !ok {
			return fmt.Errorf("unknown queueType %q, must be one of: %s, %s, %s", qt, queueTypeWorkflow, queueTypeActivity, queueTypeNexus)
		}
	}

	if err := a.validateVersioning(); err != nil {
		return err
	}

	return nil
}

func (a *temporalMetadata) validateTLS() error {
	if (a.Cert == "") != (a.Key == "") {
		return fmt.Errorf("both cert and key must be provided when using TLS")
	}
	if a.APIKey != "" && a.Cert != "" {
		return fmt.Errorf("apiKey and cert/key cannot be used together")
	}
	return nil
}

func (a *temporalMetadata) validateVersioning() error {
	switch a.WorkerVersioningType {
	case "", versioningTypeNone:
		if a.BuildID != "" || a.DeploymentName != "" {
			return fmt.Errorf("buildId and deploymentName require a workerVersioningType")
		}
	case versioningTypeBuildID:
		if a.DeploymentName != "" {
			return fmt.Errorf("deploymentName cannot be used with workerVersioningType=%s", versioningTypeBuildID)
		}
	case versioningTypeDeployment:
		if a.BuildID == "" || a.DeploymentName == "" {
			return fmt.Errorf("both buildId and deploymentName are required when workerVersioningType=%s", versioningTypeDeployment)
		}
		if len(a.QueueTypes) > 0 {
			return fmt.Errorf("queueTypes cannot be used with workerVersioningType=%s", versioningTypeDeployment)
		}
	default:
		return fmt.Errorf("unknown workerVersioningType %q, must be one of: %s, %s, %s", a.WorkerVersioningType, versioningTypeNone, versioningTypeBuildID, versioningTypeDeployment)
	}
	return nil
}

func NewTemporalScaler(ctx context.Context, config *scalersconfig.ScalerConfig) (Scaler, error) {
	logger := InitializeLogger(config, "temporal_scaler")

	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get scaler metric type: %w", err)
	}

	meta, err := parseTemporalMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Temporal metadata: %w", err)
	}

	c, err := getTemporalClient(ctx, meta, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Temporal client connection: %w", err)
	}

	switch meta.WorkerVersioningType {
	case versioningTypeDeployment:
		logger.Info("using deployment versioning", "deploymentName", meta.DeploymentName, "buildId", meta.BuildID)
	case versioningTypeBuildID:
		logger.Info("using build-id versioning", "buildId", meta.BuildID)
	default:
		logger.Info("using unversioned mode")
	}

	return &temporalScaler{
		metricType: metricType,
		metadata:   meta,
		client:     c,
		logger:     logger,
	}, nil
}

func parseTemporalMetadata(config *scalersconfig.ScalerConfig) (*temporalMetadata, error) {
	meta := &temporalMetadata{triggerIndex: config.TriggerIndex}
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("failed to parse Temporal metadata: %w", err)
	}
	return meta, nil
}

// Scaler interface methods

func (s *temporalScaler) Close(_ context.Context) error {
	if s.client != nil {
		s.client.Close()
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
	queueSize, err := s.getBacklogCount(ctx)
	if err != nil {
		s.logger.Error(err, "failed to get Temporal queue size")
		return nil, false, fmt.Errorf("failed to get Temporal queue size: %w", err)
	}

	s.logger.V(1).Info("found queue size", "queueSize", queueSize, "namespace", s.metadata.Namespace, "taskQueue", s.metadata.TaskQueue)

	metric := GenerateMetricInMili(metricName, float64(queueSize))
	isActive := queueSize > s.metadata.ActivationTargetQueueSize

	return []external_metrics.ExternalMetricValue{metric}, isActive, nil
}

// Backlog query helpers

func (s *temporalScaler) getBacklogCount(ctx context.Context) (int64, error) {
	switch s.metadata.WorkerVersioningType {
	case versioningTypeDeployment:
		return getDeploymentBacklogCount(ctx, s.client.WorkflowService(), s.metadata.Namespace, s.metadata.DeploymentName, s.metadata.BuildID)
	case versioningTypeBuildID:
		return getBuildIDBacklogCount(ctx, s.client, s.metadata.TaskQueue, s.metadata.QueueTypes, s.metadata.BuildID)
	default:
		return getUnversionedBacklogCount(ctx, s.client, s.metadata.TaskQueue, s.metadata.QueueTypes)
	}
}

func getUnversionedBacklogCount(ctx context.Context, client temporalBacklogClient, taskQueue string, queueTypes []string) (int64, error) {
	resp, err := client.DescribeTaskQueueEnhanced(ctx, sdk.DescribeTaskQueueEnhancedOptions{
		TaskQueue:      taskQueue,
		ReportStats:    true,
		TaskQueueTypes: getQueueTypes(queueTypes),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to describe task queue: %w", err)
	}
	return getCombinedBacklogCount(resp), nil
}

func getBuildIDBacklogCount(ctx context.Context, client temporalBacklogClient, taskQueue string, queueTypes []string, buildID string) (int64, error) {
	selection := &sdk.TaskQueueVersionSelection{}
	if buildID != "" {
		selection.BuildIDs = []string{buildID}
	}

	resp, err := client.DescribeTaskQueueEnhanced(ctx, sdk.DescribeTaskQueueEnhancedOptions{
		TaskQueue:      taskQueue,
		ReportStats:    true,
		Versions:       selection,
		TaskQueueTypes: getQueueTypes(queueTypes),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to describe task queue: %w", err)
	}
	return getCombinedBacklogCount(resp), nil
}

func getDeploymentBacklogCount(ctx context.Context, svc workflowservice.WorkflowServiceClient, namespace, deploymentName, buildID string) (int64, error) {
	resp, err := svc.DescribeWorkerDeploymentVersion(ctx,
		&workflowservice.DescribeWorkerDeploymentVersionRequest{
			Namespace: namespace,
			DeploymentVersion: &deploymentpb.WorkerDeploymentVersion{
				DeploymentName: deploymentName,
				BuildId:        buildID,
			},
			ReportTaskQueueStats: true,
		},
	)
	if err != nil {
		return 0, fmt.Errorf("failed to describe worker deployment version: %w", err)
	}

	var totalBacklog int64
	for _, tq := range resp.GetVersionTaskQueues() {
		if stats := tq.GetStats(); stats != nil {
			totalBacklog += stats.GetApproximateBacklogCount()
		}
	}
	return totalBacklog, nil
}

// Pure utility functions

func getQueueTypes(queueTypes []string) []sdk.TaskQueueType {
	if len(queueTypes) == 0 {
		return temporalDefaultQueueTypes
	}
	result := make([]sdk.TaskQueueType, 0, len(queueTypes))
	for _, t := range queueTypes {
		result = append(result, queueTypeMap[t])
	}
	return result
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

// Client setup

func namespaceInterceptor(namespace string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req any, reply any,
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption,
	) error {
		return invoker(
			metadata.AppendToOutgoingContext(ctx, "temporal-namespace", namespace),
			method, req, reply, cc, opts...,
		)
	}
}

func buildTLSConfig(meta *temporalMetadata) (*tls.Config, error) {
	if meta.Cert != "" && meta.Key != "" {
		tlsConfig, err := kedautil.NewTLSConfigWithPassword(meta.Cert, meta.Key, meta.KeyPassword, meta.CA, meta.UnsafeSsl)
		if err != nil {
			return nil, err
		}
		if meta.TLSServerName != "" {
			tlsConfig.ServerName = meta.TLSServerName
		}
		return tlsConfig, nil
	}

	if meta.APIKey != "" {
		tlsConfig := kedautil.CreateTLSClientConfig(meta.UnsafeSsl)
		if meta.TLSServerName != "" {
			tlsConfig.ServerName = meta.TLSServerName
		}
		return tlsConfig, nil
	}

	return nil, nil
}

func getTemporalClient(ctx context.Context, meta *temporalMetadata, log logr.Logger) (sdk.Client, error) {
	logHandler := logr.ToSlogHandler(log)
	options := sdk.Options{
		HostPort:  meta.Endpoint,
		Namespace: meta.Namespace,
		Logger:    sdklog.NewStructuredLogger(slog.New(logHandler)),
	}

	if meta.APIKey != "" {
		options.Credentials = sdk.NewAPIKeyStaticCredentials(meta.APIKey)
	}

	tlsConfig, err := buildTLSConfig(meta)
	if err != nil {
		return nil, fmt.Errorf("failed to build TLS config: %w", err)
	}

	options.ConnectionOptions = sdk.ConnectionOptions{
		DialOptions: []grpc.DialOption{
			grpc.WithConnectParams(grpc.ConnectParams{
				MinConnectTimeout: time.Duration(meta.MinConnectTimeout) * time.Second,
			}),
			grpc.WithUnaryInterceptor(namespaceInterceptor(meta.Namespace)),
		},
		TLS: tlsConfig,
	}

	return sdk.DialContext(ctx, options)
}
