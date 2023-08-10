package scalers

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
	tclfilter "go.temporal.io/api/filter/v1"
	workflowservice "go.temporal.io/api/workflowservice/v1"
	sdk "go.temporal.io/sdk/client"
	"google.golang.org/grpc"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

const (
	defaultTargetWorkflowLength           = 5
	defaultActivationTargetWorkflowLength = 0
	temporalClientTimeOut                 = 30
)

type executionInfo struct {
	workflowId string
	runId      string
}

type temporalWorkflowScaler struct {
	metricType v2.MetricTargetType
	metadata   *temporalWorkflowMetadata
	tcl        sdk.Client
	logger     logr.Logger
}

type temporalWorkflowMetadata struct {
	activationTargetWorkflowLength int64
	endpoint                       string
	namespace                      string
	workflowName                   string
	activities                     []string
	scalerIndex                    int
	targetQueueSize                int64
	metricName                     string
}

// NewTemporalWorkflowScaler creates a new instance of temporalWorkflowScaler.
func NewTemporalWorkflowScaler(config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get scaler metric type: %w", err)
	}

	meta, err := parseTemporalMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Temporal metadata: %w", err)
	}

	logger := InitializeLogger(config, "temporal_workflow_scaler")

	c, err := sdk.Dial(sdk.Options{
		HostPort: meta.endpoint,
		ConnectionOptions: sdk.ConnectionOptions{
			DialOptions: []grpc.DialOption{
				grpc.WithTimeout(time.Duration(temporalClientTimeOut) * time.Second),
			},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create Temporal client: %w", err)
	}

	return &temporalWorkflowScaler{
		metricType: metricType,
		metadata:   meta,
		tcl:        c,
		logger:     logger,
	}, nil
}

// Close closes the Temporal client connection.
func (s *temporalWorkflowScaler) Close(context.Context) error {
	if s.tcl != nil {
		s.tcl.Close()
	}
	return nil
}

// GetMetricSpecForScaling returns the metric specification for scaling.
func (s *temporalWorkflowScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: s.metadata.metricName,
		},
		Target: GetMetricTarget(s.metricType, s.metadata.targetQueueSize),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric,
		Type:     externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns metrics and activity for the scaler.
func (s *temporalWorkflowScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	queueSize, err := s.getQueueSize(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get Temporal queue size: %w", err)
	}

	metric := GenerateMetricInMili(metricName, float64(queueSize))

	return []external_metrics.ExternalMetricValue{metric}, queueSize > s.metadata.activationTargetWorkflowLength, nil
}

// getQueueSize returns the queue size of open workflows.
func (s *temporalWorkflowScaler) getQueueSize(ctx context.Context) (int64, error) {

	var executions []executionInfo
	var nextPageToken []byte
	for {
		listOpenWorkflowExecutionsRequest := &workflowservice.ListOpenWorkflowExecutionsRequest{
			Namespace:       s.metadata.namespace,
			MaximumPageSize: 1000,
			NextPageToken:   nextPageToken,
			Filters: &workflowservice.ListOpenWorkflowExecutionsRequest_TypeFilter{
				TypeFilter: &tclfilter.WorkflowTypeFilter{
					Name: s.metadata.workflowName,
				},
			},
		}
		ws, err := s.tcl.ListOpenWorkflow(ctx, listOpenWorkflowExecutionsRequest)
		if err != nil {
			return 0, fmt.Errorf("failed to get workflows: %w", err)
		}

		for _, exec := range ws.GetExecutions() {
			execution := executionInfo{
				workflowId: exec.Execution.GetWorkflowId(),
				runId:      exec.Execution.RunId,
			}
			executions = append(executions, execution)
		}

		if nextPageToken = ws.NextPageToken; len(nextPageToken) == 0 {
			break
		}
	}

	pendingCh := make(chan string, len(executions))
	var wg sync.WaitGroup

	for _, execInfo := range executions {
		wg.Add(1)
		go func(e executionInfo) {
			defer wg.Done()

			workflowId := e.workflowId
			runId := e.runId

			if !s.isActivityRunning(ctx, workflowId, runId) {
				executionId := workflowId + "__" + runId
				pendingCh <- executionId
			}

		}(execInfo)
	}
	wg.Wait()
	close(pendingCh)

	var queueLength int64
	for range pendingCh {
		queueLength++
	}
	return queueLength, nil
}

// isActivityRunning checks if there are running activities associated with a specific workflow execution.
func (s *temporalWorkflowScaler) isActivityRunning(ctx context.Context, workflowId, runId string) bool {
	resp, err := s.tcl.DescribeWorkflowExecution(ctx, workflowId, runId)
	if err != nil {
		s.logger.Error(err, "error describing workflow execution", "workflowId", workflowId, "runId", runId)
		return false
	}

	// If there is no activityName and there are running activities, return true.
	if len(s.metadata.activities) == 0 && len(resp.GetPendingActivities()) > 0 {
		return true
	}

	// Store the IDs of running activities. Make sure no duplicates incase of anything.
	runningActivities := make(map[string]struct{})
	for _, pendingActivity := range resp.GetPendingActivities() {
		activityName := pendingActivity.ActivityType.GetName()
		if s.hasMatchingActivityName(activityName) {
			runningActivities[pendingActivity.ActivityId] = struct{}{}
		}
	}

	// Return true if there are any running activities, otherwise false.
	return len(runningActivities) > 0
}

// hasMatchingActivityName checks if the provided activity name matches any of the defined activity names in the metadata.
func (s *temporalWorkflowScaler) hasMatchingActivityName(activityName string) bool {
	for _, activity := range s.metadata.activities {
		if activityName == activity {
			return true
		}
	}
	return false
}

// parseTemporalMetadata parses the Temporal metadata from the ScalerConfig.
func parseTemporalMetadata(config *ScalerConfig) (*temporalWorkflowMetadata, error) {
	meta := &temporalWorkflowMetadata{}
	meta.activationTargetWorkflowLength = defaultActivationTargetWorkflowLength
	meta.targetQueueSize = defaultTargetWorkflowLength

	if config.TriggerMetadata["endpoint"] == "" {
		return nil, errors.New("no Temporal gRPC endpoint provided")
	}
	meta.endpoint = config.TriggerMetadata["endpoint"]

	if config.TriggerMetadata["namespace"] == "" {
		meta.namespace = "default"
	} else {
		meta.namespace = config.TriggerMetadata["namespace"]
	}

	if config.TriggerMetadata["workflowName"] == "" {
		return nil, errors.New("no workflow name provided")
	}
	meta.workflowName = config.TriggerMetadata["workflowName"]

	if activities := config.TriggerMetadata["activityName"]; activities != "" {
		meta.activities = strings.Split(activities, ",")
	}

	if size, ok := config.TriggerMetadata["targetQueueSize"]; ok {
		queueSize, err := strconv.ParseInt(size, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid targetQueueSize - must be an integer")
		}
		meta.targetQueueSize = queueSize
	}

	if size, ok := config.TriggerMetadata["activationTargetQueueSize"]; ok {
		activationTargetQueueSize, err := strconv.ParseInt(size, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid activationTargetQueueSize - must be an integer")
		}
		meta.activationTargetWorkflowLength = activationTargetQueueSize
	}

	meta.metricName = GenerateMetricNameWithIndex(
		config.ScalerIndex, kedautil.NormalizeString(
			fmt.Sprintf("temporal-%s-%s", meta.namespace, meta.workflowName),
		),
	)
	meta.scalerIndex = config.ScalerIndex

	return meta, nil
}
