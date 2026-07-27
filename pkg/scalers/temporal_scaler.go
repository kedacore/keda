package scalers

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"regexp"
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
	"google.golang.org/grpc/status"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

var (
	temporalDefaultQueueTypes = []sdk.TaskQueueType{
		sdk.TaskQueueTypeActivity,
		sdk.TaskQueueTypeWorkflow,
		sdk.TaskQueueTypeNexus,
	}

	// validVisibilityQueryLiteral matches values that are safe to interpolate into a Temporal
	// visibility query. Temporal task queue and deployment identifiers permit alphanumerics,
	// hyphens, underscores, dots, forward slashes, and colons. Rejecting anything outside this
	// set prevents query injection since the SDK offers no parameterized visibility queries.
	validVisibilityQueryLiteral = regexp.MustCompile(`^[a-zA-Z0-9\-_./:]+$`)
)

// workerDeploymentVersionSeparator joins deployment name and build id to form the
// TemporalWorkerDeploymentVersion search attribute value (e.g. "my-deploy:v1").
// This matches the modern (v1.32+) Temporal server format written by
// ExternalWorkerDeploymentVersionToString in temporal/common/worker_versioning.
// The legacy v0.31 delimiter ('.') is not used here — visibility queries against
// modern servers will not match values written with the old delimiter.
const workerDeploymentVersionSeparator = ":"

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
	BuildID                   string   `keda:"name=buildId,                   order=triggerMetadata;resolvedEnv, optional, deprecatedAnnounce=The 'buildId' setting is DEPRECATED and will be removed in v2.21 because Temporal Server is dropping support for the Rules-Based Versioning APIs - Use 'workerDeploymentName' and 'workerDeploymentBuildId' instead"`
	WorkerDeploymentName      string   `keda:"name=workerDeploymentName,      order=triggerMetadata;resolvedEnv, optional"`
	WorkerDeploymentBuildID   string   `keda:"name=workerDeploymentBuildId,   order=triggerMetadata;resolvedEnv, optional"`
	AllActive                 bool     `keda:"name=selectAllActive,           order=triggerMetadata, default=false, deprecatedAnnounce=The 'selectAllActive' setting is DEPRECATED and will be removed in v2.21 because Temporal Server is dropping support for the Rules-Based Versioning APIs - Use 'workerDeploymentName' and 'workerDeploymentBuildId' instead"`
	Unversioned               bool     `keda:"name=selectUnversioned,         order=triggerMetadata, default=false, deprecatedAnnounce=The 'selectUnversioned' setting is DEPRECATED and will be removed in v2.21 because Temporal Server is dropping support for the Rules-Based Versioning APIs - Remove it if your workers are unversioned. Or use 'workerDeploymentName' and 'workerDeploymentBuildId' if you're migrating to Worker Deployment Versioning"`
	// IncludeRunningWorkflowCount, when true, keeps the scaler active if there are running
	// workflows on the task queue even after the backlog drains to zero. It only affects the
	// activity decision, never the reported metric value. Defaults to false to preserve prior
	// behavior. Activity-worker scalers can still use this by pointing WorkflowTaskQueueForCount
	// at the workflow task queue whose in-flight workflows should keep the activity workers alive.
	IncludeRunningWorkflowCount bool   `keda:"name=includeRunningWorkflowCount, order=triggerMetadata, default=false"`
	WorkflowTaskQueueForCount   string `keda:"name=workflowTaskQueueForCount,   order=triggerMetadata;resolvedEnv, optional"`
	APIKey                      string `keda:"name=apiKey,                    order=authParams;resolvedEnv, optional"`
	MinConnectTimeout           int    `keda:"name=minConnectTimeout,         order=triggerMetadata, default=5"`

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

	if (a.WorkerDeploymentName == "") != (a.WorkerDeploymentBuildID == "") {
		return fmt.Errorf("workerDeploymentName and workerDeploymentBuildId must both be set")
	}
	if a.WorkerDeploymentName != "" && (a.BuildID != "" || a.AllActive || a.Unversioned) {
		return fmt.Errorf("workerDeploymentName/workerDeploymentBuildId cannot be combined with buildId, selectAllActive, or selectUnversioned")
	}

	if a.WorkflowTaskQueueForCount != "" && !a.IncludeRunningWorkflowCount {
		return fmt.Errorf("workflowTaskQueueForCount has no effect unless includeRunningWorkflowCount is true")
	}
	if a.IncludeRunningWorkflowCount {
		for _, name := range []string{a.TaskQueue, a.WorkflowTaskQueueForCount, a.WorkerDeploymentName, a.WorkerDeploymentBuildID} {
			if name != "" && !validVisibilityQueryLiteral.MatchString(name) {
				return fmt.Errorf("value %q contains characters not allowed in Temporal visibility queries when includeRunningWorkflowCount is enabled", name)
			}
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

	c, err := getTemporalClient(ctx, meta, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Temporal client connection: %w", err)
	}

	kv := []any{
		"mode", scalerMode(meta),
		"endpoint", meta.Endpoint,
		"namespace", meta.Namespace,
		"taskQueue", meta.TaskQueue,
		"targetQueueSize", meta.TargetQueueSize,
		"activationTargetQueueSize", meta.ActivationTargetQueueSize,
		"authType", authType(meta),
		"unsafeSsl", meta.UnsafeSsl,
	}
	if meta.TLSServerName != "" {
		kv = append(kv, "tlsServerName", meta.TLSServerName)
	}
	if meta.BuildID != "" {
		kv = append(kv, "buildId", meta.BuildID)
	}
	if meta.WorkerDeploymentName != "" {
		kv = append(kv, "workerDeploymentName", meta.WorkerDeploymentName, "workerDeploymentBuildId", meta.WorkerDeploymentBuildID)
	}
	if len(meta.QueueTypes) > 0 {
		kv = append(kv, "queueTypes", meta.QueueTypes)
	}
	logger.Info("Temporal scaler initialized", kv...)

	return &temporalScaler{
		metricType: metricType,
		metadata:   meta,
		tcl:        c,
		logger:     logger,
	}, nil
}

func (s *temporalScaler) Close(_ context.Context) error {
	s.logger.V(1).Info("closing Temporal scaler")
	if s.tcl != nil {
		s.tcl.Close()
	}
	return nil
}

func (s *temporalScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	metricName := fmt.Sprintf("temporal-%s-%s", s.metadata.Namespace, s.metadata.TaskQueue)
	if s.metadata.WorkerDeploymentName != "" {
		metricName = fmt.Sprintf("%s-%s-%s", metricName, s.metadata.WorkerDeploymentName, s.metadata.WorkerDeploymentBuildID)
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
		backlogCount  int64
		versionStatus enumspb.WorkerDeploymentVersionStatus
		hasStatus     bool
		err           error
	)
	switch {
	case s.metadata.WorkerDeploymentName != "":
		backlogCount, versionStatus, err = s.getDeploymentBacklogCountAndStatus(ctx)
		hasStatus = true
	case s.metadata.BuildID != "" || s.metadata.AllActive || s.metadata.Unversioned:
		backlogCount, err = s.getBuildIDBacklogCount(ctx)
	default:
		backlogCount, err = s.getUnversionedBacklogCount(ctx)
	}
	if err != nil {
		s.logger.Error(err, "failed to get Temporal backlog count",
			"mode", scalerMode(s.metadata),
			"namespace", s.metadata.Namespace,
			"taskQueue", s.metadata.TaskQueue,
			"grpcCode", status.Code(err).String())
		return nil, false, fmt.Errorf("failed to get Temporal backlog count: %w", err)
	}

	metric := GenerateMetricInMili(metricName, float64(backlogCount))
	isActive := backlogCount > s.metadata.ActivationTargetQueueSize
	if !isActive && s.metadata.IncludeRunningWorkflowCount {
		isActive = s.isActiveWithoutBacklog(ctx, hasStatus, versionStatus)
	}

	s.logger.V(1).Info("polled Temporal backlog",
		"mode", scalerMode(s.metadata),
		"namespace", s.metadata.Namespace,
		"taskQueue", s.metadata.TaskQueue,
		"buildId", s.metadata.BuildID,
		"workerDeploymentName", s.metadata.WorkerDeploymentName,
		"workerDeploymentBuildId", s.metadata.WorkerDeploymentBuildID,
		"workerDeploymentVersionStatus", versionStatus.String(),
		"backlogCount", backlogCount,
		"activationThreshold", s.metadata.ActivationTargetQueueSize,
		"isActive", isActive)

	return []external_metrics.ExternalMetricValue{metric}, isActive, nil
}

// isActiveWithoutBacklog decides whether the scaler should remain active when the
// backlog is at or below the activation threshold. It is only invoked when the user
// has opted in via includeRunningWorkflowCount.
//
// For deployment-version scalers the Worker Deployment Version status short-circuits:
//   - DRAINING versions are kept active because Temporal keeps replaying workflows
//     pinned to that version until they complete, and the server checks completion
//     periodically (roughly every 3 minutes); polling visibility here would just
//     duplicate that signal at the risk of a false negative during eventual consistency.
//   - CURRENT / RAMPING versions fall through to a visibility count; new work can
//     land on these versions, so the running-count check is the only signal available.
//   - DRAINED / INACTIVE / UNSPECIFIED versions are safe to scale down.
//
// For every other mode (build-id, unversioned) we always fall through to the
// visibility count.
//
// Callers should be aware that this signal is approximate: it does not cover
// activity-only workers, visibility indexing has no SLA (eventual consistency is
// typically a few seconds but has no upper bound), CountWorkflow itself is
// documented as approximate, and the Temporal frontend rate-limits all visibility
// calls to roughly 10 RPS per namespace per instance.
func (s *temporalScaler) isActiveWithoutBacklog(ctx context.Context, hasStatus bool, versionStatus enumspb.WorkerDeploymentVersionStatus) bool {
	if hasStatus {
		switch versionStatus {
		case enumspb.WORKER_DEPLOYMENT_VERSION_STATUS_DRAINING:
			return true
		case enumspb.WORKER_DEPLOYMENT_VERSION_STATUS_CURRENT,
			enumspb.WORKER_DEPLOYMENT_VERSION_STATUS_RAMPING:
			// fall through to the visibility count
		default:
			return false
		}
	}

	runningCount, err := s.getRunningWorkflowCount(ctx)
	if err != nil {
		s.logger.V(1).Info("failed to get running workflow count, treating as no-signal",
			"mode", scalerMode(s.metadata),
			"error", err)
		return false
	}
	return runningCount > 0
}

func (s *temporalScaler) getBuildIDBacklogCount(ctx context.Context) (int64, error) {
	selection := &sdk.TaskQueueVersionSelection{
		AllActive:   s.metadata.AllActive,
		Unversioned: s.metadata.Unversioned,
		BuildIDs:    []string{s.metadata.BuildID},
	}

	resp, err := s.tcl.DescribeTaskQueueEnhanced(ctx, sdk.DescribeTaskQueueEnhancedOptions{
		TaskQueue:      s.metadata.TaskQueue,
		ReportStats:    true,
		Versions:       selection,
		TaskQueueTypes: getQueueTypes(s.metadata.QueueTypes),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to describe task queue enhanced (taskQueue=%q, buildId=%q): %w", s.metadata.TaskQueue, s.metadata.BuildID, err)
	}

	return getCombinedBacklogCount(resp), nil
}

// getUnversionedBacklogCount queries DescribeTaskQueue for each configured queue
// type and returns the summed backlog count. This replaces DescribeTaskQueueEnhanced
// for the unversioned path, which is deprecated and returns the default Build ID's
// backlog when no Versions selector is provided (not the unversioned queue).
func (s *temporalScaler) getUnversionedBacklogCount(ctx context.Context) (int64, error) {
	var total int64
	for _, qt := range getQueueTypes(s.metadata.QueueTypes) {
		tqt := enumspb.TaskQueueType(qt)
		resp, err := s.tcl.WorkflowService().DescribeTaskQueue(ctx, &workflowservice.DescribeTaskQueueRequest{
			Namespace:     s.metadata.Namespace,
			TaskQueue:     &taskqueuepb.TaskQueue{Name: s.metadata.TaskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
			TaskQueueType: tqt,
			ReportStats:   true,
		})
		if err != nil {
			return 0, fmt.Errorf("failed to describe task queue %q (type %s): %w", s.metadata.TaskQueue, tqt, err)
		}
		if stats := resp.GetStats(); stats != nil {
			total += stats.GetApproximateBacklogCount()
		}
	}
	return total, nil
}

func (s *temporalScaler) getDeploymentBacklogCountAndStatus(ctx context.Context) (int64, enumspb.WorkerDeploymentVersionStatus, error) {
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
		return 0, enumspb.WORKER_DEPLOYMENT_VERSION_STATUS_UNSPECIFIED,
			fmt.Errorf("failed to describe worker deployment version (deploymentName=%q, buildId=%q): %w",
				s.metadata.WorkerDeploymentName, s.metadata.WorkerDeploymentBuildID, err)
	}

	backlog := sumDeploymentBacklog(resp.GetVersionTaskQueues(), s.metadata.TaskQueue, s.metadata.QueueTypes)
	return backlog, resp.GetWorkerDeploymentVersionInfo().GetStatus(), nil
}

// getRunningWorkflowCount queries the Temporal Visibility Count API for running workflow
// executions owned by this scaler. It is only called when includeRunningWorkflowCount is
// enabled and the backlog signal alone says "scale to zero" — its role is to block premature
// scale-down when the queue is momentarily empty but workers are still busy.
//
// Scoping rules:
//   - deployment-version mode restricts by TemporalWorkerDeploymentVersion so that a
//     draining version's workflows do not keep a newer version alive, and vice versa.
//   - the default (unversioned) mode restricts by "TemporalWorkerDeploymentVersion is null"
//     so it never picks up workflows owned by versioned workers on the same task queue.
//   - the deprecated build-id selectors (buildId / selectAllActive / selectUnversioned)
//     do not map cleanly to Worker Deployment Version search attributes, so the query
//     falls back to task-queue scoping only. This is intentional: the parameter is meant
//     for users on the modern Worker Deployment Versioning path.
func (s *temporalScaler) getRunningWorkflowCount(ctx context.Context) (int64, error) {
	query, err := s.runningWorkflowCountQuery()
	if err != nil {
		return 0, err
	}
	resp, err := s.tcl.CountWorkflow(ctx, &workflowservice.CountWorkflowExecutionsRequest{
		Namespace: s.metadata.Namespace,
		Query:     query,
	})
	if err != nil {
		return 0, fmt.Errorf("count workflow: %w", err)
	}
	return resp.GetCount(), nil
}

// runningWorkflowCountQuery builds the visibility query used by getRunningWorkflowCount.
// Values that reach this method must already have passed the Validate() literal check,
// but we re-validate defensively in case a caller constructs a scaler outside the normal
// factory path.
func (s *temporalScaler) runningWorkflowCountQuery() (string, error) {
	taskQueue := s.metadata.WorkflowTaskQueueForCount
	if taskQueue == "" {
		taskQueue = s.metadata.TaskQueue
	}
	if !validVisibilityQueryLiteral.MatchString(taskQueue) {
		return "", fmt.Errorf("task queue name %q contains characters not allowed in visibility queries", taskQueue)
	}

	query := fmt.Sprintf("ExecutionStatus = 'Running' AND TaskQueue = '%s'", taskQueue)
	switch {
	case s.metadata.WorkerDeploymentName != "":
		if !validVisibilityQueryLiteral.MatchString(s.metadata.WorkerDeploymentName) ||
			!validVisibilityQueryLiteral.MatchString(s.metadata.WorkerDeploymentBuildID) {
			return "", fmt.Errorf("workerDeploymentName/workerDeploymentBuildId contain characters not allowed in visibility queries")
		}
		version := s.metadata.WorkerDeploymentName + workerDeploymentVersionSeparator + s.metadata.WorkerDeploymentBuildID
		query = fmt.Sprintf("%s AND TemporalWorkerDeploymentVersion = '%s'", query, version)
	case s.metadata.BuildID != "" || s.metadata.AllActive || s.metadata.Unversioned:
		// Deprecated Build ID selectors do not map to Worker Deployment Version search
		// attributes; leave the query scoped by task queue only.
	default:
		query += " AND TemporalWorkerDeploymentVersion is null"
	}
	return query, nil
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

func scalerMode(m *temporalMetadata) string {
	switch {
	case m.WorkerDeploymentName != "":
		return "deployment-version"
	case m.BuildID != "" || m.AllActive || m.Unversioned:
		return "build-id"
	default:
		return "unversioned"
	}
}

func authType(m *temporalMetadata) string {
	switch {
	case m.APIKey != "":
		return "apiKey"
	case m.Cert != "":
		return "mtls"
	default:
		return "none"
	}
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
		return temporalDefaultQueueTypes
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
