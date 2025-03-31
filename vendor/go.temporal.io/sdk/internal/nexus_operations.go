// The MIT License
//
// Copyright (c) 2024 Temporal Technologies Inc.  All rights reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package internal

import (
	"context"
	"fmt"
	"strconv"

	"github.com/nexus-rpc/sdk-go/nexus"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"
	nexuspb "go.temporal.io/api/nexus/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/workflowservice/v1"

	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/internal/common/metrics"
	"go.temporal.io/sdk/log"
)

// NexusOperationContext is an internal only struct that holds fields used by the temporalnexus functions.
type NexusOperationContext struct {
	Client              Client
	Namespace           string
	TaskQueue           string
	MetricsHandler      metrics.Handler
	Log                 log.Logger
	registry *registry
}

func (nc *NexusOperationContext) ResolveWorkflowName(wf any) (string, error) {
	return getWorkflowFunctionName(nc.registry, wf)
}

type nexusOperationContextKeyType struct{}

// nexusOperationContextKey is a key for associating a [NexusOperationContext] with a [context.Context].
var nexusOperationContextKey = nexusOperationContextKeyType{}

type isWorkflowRunOpContextKeyType struct{}

// IsWorkflowRunOpContextKey is a key to mark that the current context is used within a workflow run operation.
// The fake test env client verifies this key is set on the context to decide whether it should execute a method or
// panic as we don't want to expose a partial client to sync operations.
var IsWorkflowRunOpContextKey = isWorkflowRunOpContextKeyType{}

// NexusOperationContextFromGoContext gets the [NexusOperationContext] associated with the given [context.Context].
func NexusOperationContextFromGoContext(ctx context.Context) (nctx *NexusOperationContext, ok bool) {
	nctx, ok = ctx.Value(nexusOperationContextKey).(*NexusOperationContext)
	return
}

// nexusOperationFailure is a utility in use by the test environment.
func nexusOperationFailure(params executeNexusOperationParams, operationID string, cause *failurepb.Failure) *failurepb.Failure {
	return &failurepb.Failure{
		Message: "nexus operation completed unsuccessfully",
		FailureInfo: &failurepb.Failure_NexusOperationExecutionFailureInfo{
			NexusOperationExecutionFailureInfo: &failurepb.NexusOperationFailureInfo{
				Endpoint:    params.client.Endpoint(),
				Service:     params.client.Service(),
				Operation:   params.operation,
				OperationId: operationID,
			},
		},
		Cause: cause,
	}
}

// unsuccessfulOperationErrorToTemporalFailure is a utility in use by the test environment.
// copied from the server codebase with a slight adaptation: https://github.com/temporalio/temporal/blob/7635cd7dbdc7dd3219f387e8fc66fa117f585ff6/common/nexus/failure.go#L69-L108
func unsuccessfulOperationErrorToTemporalFailure(err *nexuspb.UnsuccessfulOperationError) *failurepb.Failure {
	failure := &failurepb.Failure{
		Message: err.Failure.Message,
	}
	if err.OperationState == string(nexus.OperationStateCanceled) {
		failure.FailureInfo = &failurepb.Failure_CanceledFailureInfo{
			CanceledFailureInfo: &failurepb.CanceledFailureInfo{
				Details: nexusFailureMetadataToPayloads(err.Failure),
			},
		}
	} else {
		failure.FailureInfo = &failurepb.Failure_ApplicationFailureInfo{
			ApplicationFailureInfo: &failurepb.ApplicationFailureInfo{
				// Make up a type here, it's not part of the Nexus Failure spec.
				Type:         "NexusOperationFailure",
				Details:      nexusFailureMetadataToPayloads(err.Failure),
				NonRetryable: true,
			},
		}
	}
	return failure
}

// nexusFailureMetadataToPayloads is a utility in use by the test environment.
// copied from the server codebase with a slight adaptation: https://github.com/temporalio/temporal/blob/7635cd7dbdc7dd3219f387e8fc66fa117f585ff6/common/nexus/failure.go#L69-L108
func nexusFailureMetadataToPayloads(failure *nexuspb.Failure) *commonpb.Payloads {
	if len(failure.Metadata) == 0 && len(failure.Details) == 0 {
		return nil
	}
	metadata := make(map[string][]byte, len(failure.Metadata))
	for k, v := range failure.Metadata {
		metadata[k] = []byte(v)
	}
	return &commonpb.Payloads{
		Payloads: []*commonpb.Payload{
			{
				Metadata: metadata,
				Data:     failure.Details,
			},
		},
	}
}

// testSuiteClientForNexusOperations is a partial [Client] implementation for the test workflow environment used to
// support running the workflow run operation - and only this operation, all methods will panic when this client is
// passed to sync operations.
type testSuiteClientForNexusOperations struct {
	env *testWorkflowEnvironmentImpl
}

// CancelWorkflow implements Client.
func (t *testSuiteClientForNexusOperations) CancelWorkflow(ctx context.Context, workflowID string, runID string) error {
	if set, ok := ctx.Value(IsWorkflowRunOpContextKey).(bool); !ok || !set {
		panic("not implemented in the test environment")
	}
	doneCh := make(chan error)
	t.env.cancelWorkflowByID(workflowID, runID, func(result *commonpb.Payloads, err error) {
		doneCh <- err
	})
	return <-doneCh
}

// CheckHealth implements Client.
func (t *testSuiteClientForNexusOperations) CheckHealth(ctx context.Context, request *CheckHealthRequest) (*CheckHealthResponse, error) {
	return &CheckHealthResponse{}, nil
}

// Close implements Client.
func (t *testSuiteClientForNexusOperations) Close() {
	// No op.
}

// CompleteActivity implements Client.
func (t *testSuiteClientForNexusOperations) CompleteActivity(ctx context.Context, taskToken []byte, result interface{}, err error) error {
	panic("not implemented in the test environment")
}

// CompleteActivityByID implements Client.
func (t *testSuiteClientForNexusOperations) CompleteActivityByID(ctx context.Context, namespace string, workflowID string, runID string, activityID string, result interface{}, err error) error {
	panic("not implemented in the test environment")
}

// CountWorkflow implements Client.
func (t *testSuiteClientForNexusOperations) CountWorkflow(ctx context.Context, request *workflowservice.CountWorkflowExecutionsRequest) (*workflowservice.CountWorkflowExecutionsResponse, error) {
	panic("not implemented in the test environment")
}

// DescribeTaskQueue implements Client.
func (t *testSuiteClientForNexusOperations) DescribeTaskQueue(ctx context.Context, taskqueue string, taskqueueType enums.TaskQueueType) (*workflowservice.DescribeTaskQueueResponse, error) {
	panic("not implemented in the test environment")
}

// DescribeTaskQueueEnhanced implements Client.
func (t *testSuiteClientForNexusOperations) DescribeTaskQueueEnhanced(ctx context.Context, options DescribeTaskQueueEnhancedOptions) (TaskQueueDescription, error) {
	panic("unimplemented in the test environment")
}

// DescribeWorkflowExecution implements Client.
func (t *testSuiteClientForNexusOperations) DescribeWorkflowExecution(ctx context.Context, workflowID string, runID string) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
	panic("not implemented in the test environment")
}

// ExecuteWorkflow implements Client.
func (t *testSuiteClientForNexusOperations) ExecuteWorkflow(ctx context.Context, options StartWorkflowOptions, workflow interface{}, args ...interface{}) (WorkflowRun, error) {
	if set, ok := ctx.Value(IsWorkflowRunOpContextKey).(bool); !ok || !set {
		panic("not implemented in the test environment")
	}
	wfType, input, err := getValidatedWorkflowFunction(workflow, args, t.env.dataConverter, t.env.GetRegistry())
	if err != nil {
		return nil, fmt.Errorf("cannot validate workflow function: %w", err)
	}

	run := &testEnvWorkflowRunForNexusOperations{}
	doneCh := make(chan error)

	var callback *commonpb.Callback

	if len(options.callbacks) > 0 {
		callback = options.callbacks[0]
	}

	t.env.postCallback(func() {
		t.env.executeChildWorkflowWithDelay(options.StartDelay, ExecuteWorkflowParams{
			// Not propagating Header as this client does not support context propagation.
			WorkflowType: wfType,
			Input:        input,
			WorkflowOptions: WorkflowOptions{
				WaitForCancellation:      true,
				Namespace:                t.env.workflowInfo.Namespace,
				TaskQueueName:            t.env.workflowInfo.TaskQueueName,
				WorkflowID:               options.ID,
				WorkflowExecutionTimeout: options.WorkflowExecutionTimeout,
				WorkflowRunTimeout:       options.WorkflowRunTimeout,
				WorkflowTaskTimeout:      options.WorkflowTaskTimeout,
				DataConverter:            t.env.dataConverter,
				WorkflowIDReusePolicy:    options.WorkflowIDReusePolicy,
				ContextPropagators:       t.env.contextPropagators,
				SearchAttributes:         options.SearchAttributes,
				TypedSearchAttributes:    options.TypedSearchAttributes,
				ParentClosePolicy:        enums.PARENT_CLOSE_POLICY_ABANDON,
				Memo:                     options.Memo,
				CronSchedule:             options.CronSchedule,
				RetryPolicy:              convertToPBRetryPolicy(options.RetryPolicy),
			},
		}, func(result *commonpb.Payloads, wfErr error) {
			ncb := callback.GetNexus()
			if ncb == nil {
				return
			}
			seqStr := ncb.GetHeader()["operation-sequence"]
			if seqStr == "" {
				return
			}
			seq, err := strconv.ParseInt(seqStr, 10, 64)
			if err != nil {
				panic(fmt.Errorf("unexpected operation sequence in callback header: %s: %w", seqStr, err))
			}

			if wfErr != nil {
				t.env.resolveNexusOperation(seq, nil, wfErr)
			} else {
				var payload *commonpb.Payload
				if len(result.GetPayloads()) > 0 {
					payload = result.Payloads[0]
				}
				t.env.resolveNexusOperation(seq, payload, nil)
			}
		}, func(r WorkflowExecution, err error) {
			run.WorkflowExecution = r
			doneCh <- err
		})
	}, false)
	err = <-doneCh
	if err != nil {
		return nil, err
	}
	return run, nil
}

func (t *testSuiteClientForNexusOperations) NewWithStartWorkflowOperation(options StartWorkflowOptions, workflow interface{}, args ...interface{}) WithStartWorkflowOperation {
	panic("not implemented in the test environment")
}

// GetSearchAttributes implements Client.
func (t *testSuiteClientForNexusOperations) GetSearchAttributes(ctx context.Context) (*workflowservice.GetSearchAttributesResponse, error) {
	panic("not implemented in the test environment")
}

// GetWorkerBuildIdCompatibility implements Client.
func (t *testSuiteClientForNexusOperations) GetWorkerBuildIdCompatibility(ctx context.Context, options *GetWorkerBuildIdCompatibilityOptions) (*WorkerBuildIDVersionSets, error) {
	panic("not implemented in the test environment")
}

// GetWorkerTaskReachability implements Client.
func (t *testSuiteClientForNexusOperations) GetWorkerTaskReachability(ctx context.Context, options *GetWorkerTaskReachabilityOptions) (*WorkerTaskReachability, error) {
	panic("not implemented in the test environment")
}

// GetWorkerVersioningRules implements Client.
func (t *testSuiteClientForNexusOperations) GetWorkerVersioningRules(ctx context.Context, options GetWorkerVersioningOptions) (*WorkerVersioningRules, error) {
	panic("unimplemented in the test environment")
}

// GetWorkflow implements Client.
func (t *testSuiteClientForNexusOperations) GetWorkflow(ctx context.Context, workflowID string, runID string) WorkflowRun {
	panic("not implemented in the test environment")
}

// GetWorkflowHistory implements Client.
func (t *testSuiteClientForNexusOperations) GetWorkflowHistory(ctx context.Context, workflowID string, runID string, isLongPoll bool, filterType enums.HistoryEventFilterType) HistoryEventIterator {
	panic("not implemented in the test environment")
}

// GetWorkflowUpdateHandle implements Client.
func (t *testSuiteClientForNexusOperations) GetWorkflowUpdateHandle(GetWorkflowUpdateHandleOptions) WorkflowUpdateHandle {
	panic("not implemented in the test environment")
}

// ListArchivedWorkflow implements Client.
func (t *testSuiteClientForNexusOperations) ListArchivedWorkflow(ctx context.Context, request *workflowservice.ListArchivedWorkflowExecutionsRequest) (*workflowservice.ListArchivedWorkflowExecutionsResponse, error) {
	panic("not implemented in the test environment")
}

// ListClosedWorkflow implements Client.
func (t *testSuiteClientForNexusOperations) ListClosedWorkflow(ctx context.Context, request *workflowservice.ListClosedWorkflowExecutionsRequest) (*workflowservice.ListClosedWorkflowExecutionsResponse, error) {
	panic("not implemented in the test environment")
}

// ListOpenWorkflow implements Client.
func (t *testSuiteClientForNexusOperations) ListOpenWorkflow(ctx context.Context, request *workflowservice.ListOpenWorkflowExecutionsRequest) (*workflowservice.ListOpenWorkflowExecutionsResponse, error) {
	panic("not implemented in the test environment")
}

// ListWorkflow implements Client.
func (t *testSuiteClientForNexusOperations) ListWorkflow(ctx context.Context, request *workflowservice.ListWorkflowExecutionsRequest) (*workflowservice.ListWorkflowExecutionsResponse, error) {
	panic("not implemented in the test environment")
}

// OperatorService implements Client.
func (t *testSuiteClientForNexusOperations) OperatorService() operatorservice.OperatorServiceClient {
	panic("not implemented in the test environment")
}

// QueryWorkflow implements Client.
func (t *testSuiteClientForNexusOperations) QueryWorkflow(ctx context.Context, workflowID string, runID string, queryType string, args ...interface{}) (converter.EncodedValue, error) {
	panic("not implemented in the test environment")
}

// QueryWorkflowWithOptions implements Client.
func (t *testSuiteClientForNexusOperations) QueryWorkflowWithOptions(ctx context.Context, request *QueryWorkflowWithOptionsRequest) (*QueryWorkflowWithOptionsResponse, error) {
	panic("not implemented in the test environment")
}

// RecordActivityHeartbeat implements Client.
func (t *testSuiteClientForNexusOperations) RecordActivityHeartbeat(ctx context.Context, taskToken []byte, details ...interface{}) error {
	panic("not implemented in the test environment")
}

// RecordActivityHeartbeatByID implements Client.
func (t *testSuiteClientForNexusOperations) RecordActivityHeartbeatByID(ctx context.Context, namespace string, workflowID string, runID string, activityID string, details ...interface{}) error {
	panic("not implemented in the test environment")
}

// ResetWorkflowExecution implements Client.
func (t *testSuiteClientForNexusOperations) ResetWorkflowExecution(ctx context.Context, request *workflowservice.ResetWorkflowExecutionRequest) (*workflowservice.ResetWorkflowExecutionResponse, error) {
	panic("not implemented in the test environment")
}

// ScanWorkflow implements Client.
func (t *testSuiteClientForNexusOperations) ScanWorkflow(ctx context.Context, request *workflowservice.ScanWorkflowExecutionsRequest) (*workflowservice.ScanWorkflowExecutionsResponse, error) {
	panic("not implemented in the test environment")
}

// ScheduleClient implements Client.
func (t *testSuiteClientForNexusOperations) ScheduleClient() ScheduleClient {
	panic("not implemented in the test environment")
}

// SignalWithStartWorkflow implements Client.
func (t *testSuiteClientForNexusOperations) SignalWithStartWorkflow(ctx context.Context, workflowID string, signalName string, signalArg interface{}, options StartWorkflowOptions, workflow interface{}, workflowArgs ...interface{}) (WorkflowRun, error) {
	panic("not implemented in the test environment")
}

// SignalWorkflow implements Client.
func (t *testSuiteClientForNexusOperations) SignalWorkflow(ctx context.Context, workflowID string, runID string, signalName string, arg interface{}) error {
	panic("not implemented in the test environment")
}

// TerminateWorkflow implements Client.
func (t *testSuiteClientForNexusOperations) TerminateWorkflow(ctx context.Context, workflowID string, runID string, reason string, details ...interface{}) error {
	panic("not implemented in the test environment")
}

// UpdateWorkflow implements Client.
func (t *testSuiteClientForNexusOperations) UpdateWorkflow(ctx context.Context, options UpdateWorkflowOptions) (WorkflowUpdateHandle, error) {
	panic("unimplemented in the test environment")
}

// UpdateWithStartWorkflow implements Client.
func (t *testSuiteClientForNexusOperations) UpdateWithStartWorkflow(ctx context.Context, options UpdateWithStartWorkflowOptions) (WorkflowUpdateHandle, error) {
	panic("unimplemented in the test environment")
}

// UpdateWorkerBuildIdCompatibility implements Client.
func (t *testSuiteClientForNexusOperations) UpdateWorkerBuildIdCompatibility(ctx context.Context, options *UpdateWorkerBuildIdCompatibilityOptions) error {
	panic("not implemented in the test environment")
}

// UpdateWorkerVersioningRules implements Client.
func (t *testSuiteClientForNexusOperations) UpdateWorkerVersioningRules(ctx context.Context, options UpdateWorkerVersioningRulesOptions) (*WorkerVersioningRules, error) {
	panic("unimplemented in the test environment")
}

// WorkflowService implements Client.
func (t *testSuiteClientForNexusOperations) WorkflowService() workflowservice.WorkflowServiceClient {
	panic("not implemented in the test environment")
}

// DeploymentClient implements Client.
func (t *testSuiteClientForNexusOperations) DeploymentClient() DeploymentClient {
	panic("not implemented in the test environment")
}

// UpdateWorkflowExecutionOptions implements Client.
func (t *testSuiteClientForNexusOperations) UpdateWorkflowExecutionOptions(ctx context.Context, options UpdateWorkflowExecutionOptionsRequest) (WorkflowExecutionOptions, error) {
	panic("not implemented in the test environment")
}

var _ Client = &testSuiteClientForNexusOperations{}

// testEnvWorkflowRunForNexusOperations is a partial [WorkflowRun] implementation for the test workflow environment used
// to support basic Nexus functionality.
type testEnvWorkflowRunForNexusOperations struct {
	WorkflowExecution
}

// Get implements WorkflowRun.
func (t *testEnvWorkflowRunForNexusOperations) Get(ctx context.Context, valuePtr interface{}) error {
	panic("not implemented in the test environment")
}

// GetID implements WorkflowRun.
func (t *testEnvWorkflowRunForNexusOperations) GetID() string {
	return t.ID
}

// GetRunID implements WorkflowRun.
func (t *testEnvWorkflowRunForNexusOperations) GetRunID() string {
	return t.RunID
}

// GetWithOptions implements WorkflowRun.
func (t *testEnvWorkflowRunForNexusOperations) GetWithOptions(ctx context.Context, valuePtr interface{}, options WorkflowRunGetOptions) error {
	panic("not implemented in the test environment")
}

// Exposed as: [go.temporal.io/sdk/client.WorkflowRun]
var _ WorkflowRun = &testEnvWorkflowRunForNexusOperations{}
