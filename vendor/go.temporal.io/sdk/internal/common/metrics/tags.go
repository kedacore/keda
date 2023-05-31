// The MIT License
//
// Copyright (c) 2021 Temporal Technologies Inc.  All rights reserved.
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

// RootTags returns a set of base tags for all metrics.
func RootTags(namespace string) map[string]string {
	return map[string]string{
		NamespaceTagName:        namespace,
		ClientTagName:           ClientTagValue,
		WorkerTypeTagName:       NoneTagValue,
		WorkflowTypeNameTagName: NoneTagValue,
		ActivityTypeNameTagName: NoneTagValue,
		TaskQueueTagName:        NoneTagValue,
	}
}

// RPCTags returns a set of tags for RPC calls.
func RPCTags(workflowType, activityType, taskQueueName string) map[string]string {
	return map[string]string{
		WorkflowTypeNameTagName: workflowType,
		ActivityTypeNameTagName: activityType,
		TaskQueueTagName:        taskQueueName,
	}
}

// WorkflowTags returns a set of tags for workflows.
func WorkflowTags(workflowType string) map[string]string {
	return map[string]string{
		WorkflowTypeNameTagName: workflowType,
	}
}

// ActivityTags returns a set of tags for activities.
func ActivityTags(workflowType, activityType, taskQueueName string) map[string]string {
	return map[string]string{
		WorkflowTypeNameTagName: workflowType,
		ActivityTypeNameTagName: activityType,
		TaskQueueTagName:        taskQueueName,
	}
}

// LocalActivityTags returns a set of tags for local activities.
func LocalActivityTags(workflowType, activityType string) map[string]string {
	return map[string]string{
		WorkflowTypeNameTagName: workflowType,
		ActivityTypeNameTagName: activityType,
	}
}

// TaskQueueTags returns a set of tags for a task queue.
func TaskQueueTags(taskQueue string) map[string]string {
	return map[string]string{
		TaskQueueTagName: taskQueue,
	}
}

// WorkerTags returns a set of tags for workers.
func WorkerTags(workerType string) map[string]string {
	return map[string]string{
		WorkerTypeTagName: workerType,
	}
}

// PollerTags returns a set of tags for pollers.
func PollerTags(pollerType string) map[string]string {
	return map[string]string{
		PollerTypeTagName: pollerType,
	}
}
