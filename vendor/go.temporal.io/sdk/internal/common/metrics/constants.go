// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
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

package metrics

// Metrics keys
const (
	TemporalMetricsPrefix = "temporal_"

	WorkflowCompletedCounter     = TemporalMetricsPrefix + "workflow_completed"
	WorkflowCanceledCounter      = TemporalMetricsPrefix + "workflow_canceled"
	WorkflowFailedCounter        = TemporalMetricsPrefix + "workflow_failed"
	WorkflowContinueAsNewCounter = TemporalMetricsPrefix + "workflow_continue_as_new"
	WorkflowEndToEndLatency      = TemporalMetricsPrefix + "workflow_endtoend_latency" // measure workflow execution from start to close

	WorkflowTaskReplayLatency           = TemporalMetricsPrefix + "workflow_task_replay_latency"
	WorkflowTaskQueuePollEmptyCounter   = TemporalMetricsPrefix + "workflow_task_queue_poll_empty"
	WorkflowTaskQueuePollSucceedCounter = TemporalMetricsPrefix + "workflow_task_queue_poll_succeed"
	WorkflowTaskScheduleToStartLatency  = TemporalMetricsPrefix + "workflow_task_schedule_to_start_latency"
	WorkflowTaskExecutionLatency        = TemporalMetricsPrefix + "workflow_task_execution_latency"
	WorkflowTaskExecutionFailureCounter = TemporalMetricsPrefix + "workflow_task_execution_failed"
	WorkflowTaskNoCompletionCounter     = TemporalMetricsPrefix + "workflow_task_no_completion"

	ActivityPollNoTaskCounter             = TemporalMetricsPrefix + "activity_poll_no_task"
	ActivityScheduleToStartLatency        = TemporalMetricsPrefix + "activity_schedule_to_start_latency"
	ActivityExecutionFailedCounter        = TemporalMetricsPrefix + "activity_execution_failed"
	UnregisteredActivityInvocationCounter = TemporalMetricsPrefix + "unregistered_activity_invocation"
	ActivityExecutionLatency              = TemporalMetricsPrefix + "activity_execution_latency"
	ActivitySucceedEndToEndLatency        = TemporalMetricsPrefix + "activity_succeed_endtoend_latency"
	ActivityTaskErrorCounter              = TemporalMetricsPrefix + "activity_task_error"

	LocalActivityTotalCounter             = TemporalMetricsPrefix + "local_activity_total"
	LocalActivityCanceledCounter          = TemporalMetricsPrefix + "local_activity_canceled" // Deprecated: Use LocalActivityExecutionCanceledCounter instead.
	LocalActivityExecutionCanceledCounter = TemporalMetricsPrefix + "local_activity_execution_cancelled"
	LocalActivityFailedCounter            = TemporalMetricsPrefix + "local_activity_failed" // Deprecated: Use LocalActivityExecutionFailedCounter instead.
	LocalActivityExecutionFailedCounter   = TemporalMetricsPrefix + "local_activity_execution_failed"
	LocalActivityErrorCounter             = TemporalMetricsPrefix + "local_activity_error"
	LocalActivityExecutionLatency         = TemporalMetricsPrefix + "local_activity_execution_latency"
	LocalActivitySucceedEndToEndLatency   = TemporalMetricsPrefix + "local_activity_succeed_endtoend_latency"

	CorruptedSignalsCounter = TemporalMetricsPrefix + "corrupted_signals"

	WorkerStartCounter       = TemporalMetricsPrefix + "worker_start"
	WorkerTaskSlotsAvailable = TemporalMetricsPrefix + "worker_task_slots_available"
	PollerStartCounter       = TemporalMetricsPrefix + "poller_start"
	NumPoller                = TemporalMetricsPrefix + "num_pollers"

	TemporalRequest                      = TemporalMetricsPrefix + "request"
	TemporalRequestFailure               = TemporalRequest + "_failure"
	TemporalRequestLatency               = TemporalRequest + "_latency"
	TemporalLongRequest                  = TemporalMetricsPrefix + "long_request"
	TemporalLongRequestFailure           = TemporalLongRequest + "_failure"
	TemporalLongRequestLatency           = TemporalLongRequest + "_latency"
	TemporalRequestResourceExhausted     = TemporalRequest + "_resource_exhausted"
	TemporalLongRequestResourceExhausted = TemporalLongRequest + "_resource_exhausted"

	StickyCacheHit                 = TemporalMetricsPrefix + "sticky_cache_hit"
	StickyCacheMiss                = TemporalMetricsPrefix + "sticky_cache_miss"
	StickyCacheTotalForcedEviction = TemporalMetricsPrefix + "sticky_cache_total_forced_eviction"
	StickyCacheSize                = TemporalMetricsPrefix + "sticky_cache_size"

	WorkflowActiveThreadCount = TemporalMetricsPrefix + "workflow_active_thread_count"
)

// Metric tag keys
const (
	NamespaceTagName        = "namespace"
	ClientTagName           = "client_name"
	PollerTypeTagName       = "poller_type"
	WorkerTypeTagName       = "worker_type"
	WorkflowTypeNameTagName = "workflow_type"
	ActivityTypeNameTagName = "activity_type"
	TaskQueueTagName        = "task_queue"
	OperationTagName        = "operation"
	CauseTagName            = "cause"
)

// Metric tag values
const (
	NoneTagValue                 = "none"
	ClientTagValue               = "temporal_go"
	PollerTypeWorkflowTask       = "workflow_task"
	PollerTypeWorkflowStickyTask = "workflow_sticky_task"
	PollerTypeActivityTask       = "activity_task"
)
