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
	"time"

	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/internal/common/metrics"
	"go.temporal.io/sdk/log"
)

type nexusTaskPoller struct {
	basePoller
	namespace       string
	taskQueueName   string
	identity        string
	service         workflowservice.WorkflowServiceClient
	taskHandler     *nexusTaskHandler
	logger          log.Logger
	numPollerMetric *numPollerMetric
}

type nexusTask struct {
	task *workflowservice.PollNexusTaskQueueResponse
}

var _ taskPoller = &nexusTaskPoller{}

func newNexusTaskPoller(
	taskHandler *nexusTaskHandler,
	service workflowservice.WorkflowServiceClient,
	params workerExecutionParameters,
) *nexusTaskPoller {
	return &nexusTaskPoller{
		basePoller: basePoller{
			metricsHandler:       params.MetricsHandler,
			stopC:                params.WorkerStopChannel,
			workerBuildID:        params.getBuildID(),
			useBuildIDVersioning: params.UseBuildIDForVersioning,
			deploymentSeriesName: params.DeploymentSeriesName,
			capabilities:         params.capabilities,
		},
		taskHandler:     taskHandler,
		service:         service,
		namespace:       params.Namespace,
		taskQueueName:   params.TaskQueue,
		identity:        params.Identity,
		logger:          params.Logger,
		numPollerMetric: newNumPollerMetric(params.MetricsHandler, metrics.PollerTypeNexusTask),
	}
}

// Poll the nexus task queue and update the num_poller metric
func (ntp *nexusTaskPoller) pollNexusTaskQueue(ctx context.Context, request *workflowservice.PollNexusTaskQueueRequest) (*workflowservice.PollNexusTaskQueueResponse, error) {
	ntp.numPollerMetric.increment()
	defer ntp.numPollerMetric.decrement()

	return ntp.service.PollNexusTaskQueue(ctx, request)
}

func (ntp *nexusTaskPoller) poll(ctx context.Context) (taskForWorker, error) {
	traceLog(func() {
		ntp.logger.Debug("nexusTaskPoller::Poll")
	})
	request := &workflowservice.PollNexusTaskQueueRequest{
		Namespace: ntp.namespace,
		TaskQueue: &taskqueuepb.TaskQueue{Name: ntp.taskQueueName, Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
		Identity:  ntp.identity,
		WorkerVersionCapabilities: &commonpb.WorkerVersionCapabilities{
			BuildId:              ntp.workerBuildID,
			UseVersioning:        ntp.useBuildIDVersioning,
			DeploymentSeriesName: ntp.deploymentSeriesName,
		},
	}

	response, err := ntp.pollNexusTaskQueue(ctx, request)
	if err != nil {
		return nil, err
	}
	if response == nil || len(response.TaskToken) == 0 {
		// No operation info is available on empty poll. Emit using base scope.
		ntp.metricsHandler.Counter(metrics.NexusPollNoTaskCounter).Inc(1)
		return nil, nil
	}

	return &nexusTask{task: response}, nil
}

func (ntp *nexusTaskPoller) Cleanup() error {
	return nil
}

// PollTask polls a new task
func (ntp *nexusTaskPoller) PollTask() (taskForWorker, error) {
	return ntp.doPoll(ntp.poll)
}

// ProcessTask processes a new task
func (ntp *nexusTaskPoller) ProcessTask(task interface{}) error {
	if ntp.stopping() {
		return errStop
	}

	response := task.(*nexusTask).task
	if response.GetRequest() == nil {
		// We didn't get a request, poll must have timed out.
		traceLog(func() {
			ntp.logger.Debug("Empty Nexus poll response")
		})
		return nil
	}

	nctx, handlerErr := ntp.taskHandler.newNexusOperationContext(response)
	if handlerErr != nil {
		// context wasn't propagated to us, use a background context.
		_, err := ntp.taskHandler.client.WorkflowService().RespondNexusTaskFailed(
			context.Background(), ntp.taskHandler.fillInFailure(response.TaskToken, handlerErr))
		return err
	}

	executionStartTime := time.Now()

	// Schedule-to-start (from the time the request hit the frontend).
	scheduleToStartLatency := executionStartTime.Sub(response.GetRequest().GetScheduledTime().AsTime())
	nctx.MetricsHandler.Timer(metrics.NexusTaskScheduleToStartLatency).Record(scheduleToStartLatency)

	// Process the nexus task.
	res, failure, err := ntp.taskHandler.ExecuteContext(nctx, response)

	// Execution latency (in-SDK processing time).
	nctx.MetricsHandler.Timer(metrics.NexusTaskExecutionLatency).Record(time.Since(executionStartTime))

	// Increment failure in all forms of errors:
	// Internal error processing the task.
	// Failure from user handler.
	// Special case for the start response with operation error.
	if err != nil {
		var failureTag string
		if err == errNexusTaskTimeout {
			failureTag = "timeout"
		} else {
			failureTag = "internal_sdk_error"
		}
		nctx.Log.Error("Error processing nexus task", "error", err)
		nctx.MetricsHandler.
			WithTags(metrics.NexusTaskFailureTags(failureTag)).
			Counter(metrics.NexusTaskExecutionFailedCounter).
			Inc(1)
	} else if failure != nil {
		nctx.MetricsHandler.
			WithTags(metrics.NexusTaskFailureTags("handler_error_" + failure.GetError().GetErrorType())).
			Counter(metrics.NexusTaskExecutionFailedCounter).
			Inc(1)
	} else if e := res.Response.GetStartOperation().GetOperationError(); e != nil {
		nctx.MetricsHandler.
			WithTags(metrics.NexusTaskFailureTags("operation_" + e.GetOperationState())).
			Counter(metrics.NexusTaskExecutionFailedCounter).
			Inc(1)
	}

	// Let the poller machinery drop the task, nothing to report back.
	// This is only expected due to context deadline errors.
	if err != nil {
		return err
	}

	if err := ntp.reportCompletion(res, failure); err != nil {
		traceLog(func() {
			ntp.logger.Debug("reportNexusTaskComplete failed", tagError, err)
		})
		return err
	}

	// E2E latency, from frontend until we finished reporting completion.
	nctx.MetricsHandler.
		Timer(metrics.NexusTaskEndToEndLatency).
		Record(time.Since(response.GetRequest().GetScheduledTime().AsTime()))
	return nil
}

func (ntp *nexusTaskPoller) reportCompletion(
	completion *workflowservice.RespondNexusTaskCompletedRequest,
	failure *workflowservice.RespondNexusTaskFailedRequest,
) error {
	ctx := context.Background()
	// No workflow or activity tags to report.
	// Task queue expected to be empty for Respond*Task... requests.
	rpcMetricsHandler := ntp.metricsHandler.WithTags(metrics.RPCTags(metrics.NoneTagValue, metrics.NoneTagValue, metrics.NoneTagValue))
	ctx, cancel := newGRPCContext(ctx, grpcMetricsHandler(rpcMetricsHandler),
		defaultGrpcRetryParameters(ctx))
	defer cancel()

	if failure != nil {
		_, err := ntp.taskHandler.client.WorkflowService().RespondNexusTaskFailed(ctx, failure)
		return err
	}
	_, err := ntp.taskHandler.client.WorkflowService().RespondNexusTaskCompleted(ctx, completion)
	return err
}
