package metrics

import (
	"strconv"

	"google.golang.org/grpc/codes"
)

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

// NexusTags returns a set of tags for Nexus Operations.
func NexusTags(service, operation, taskQueueName string) map[string]string {
	return map[string]string{
		NexusServiceTagName:   service,
		NexusOperationTagName: operation,
		TaskQueueTagName:      taskQueueName,
	}
}

// NexusTaskFailureTags returns a set of tags for Nexus Operation failures.
func NexusTaskFailureTags(reason string) map[string]string {
	return map[string]string{
		FailureReasonTagName: reason,
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

// WorkflowTaskFailedTags returns a set of tags for a workflow task failure.
func WorkflowTaskFailedTags(reason string) map[string]string {
	return map[string]string{
		FailureReasonTagName: reason,
	}
}

// RequestFailureCodeTags returns a set of tags for a request failure.
func RequestFailureCodeTags(statusCode codes.Code) map[string]string {
	asStr := canonicalString(statusCode)
	return map[string]string{
		RequestFailureCode: asStr,
	}
}

// Annoyingly gRPC defines this, but does not expose it publicly.
func canonicalString(c codes.Code) string {
	switch c {
	case codes.OK:
		return "OK"
	case codes.Canceled:
		return "CANCELLED"
	case codes.Unknown:
		return "UNKNOWN"
	case codes.InvalidArgument:
		return "INVALID_ARGUMENT"
	case codes.DeadlineExceeded:
		return "DEADLINE_EXCEEDED"
	case codes.NotFound:
		return "NOT_FOUND"
	case codes.AlreadyExists:
		return "ALREADY_EXISTS"
	case codes.PermissionDenied:
		return "PERMISSION_DENIED"
	case codes.ResourceExhausted:
		return "RESOURCE_EXHAUSTED"
	case codes.FailedPrecondition:
		return "FAILED_PRECONDITION"
	case codes.Aborted:
		return "ABORTED"
	case codes.OutOfRange:
		return "OUT_OF_RANGE"
	case codes.Unimplemented:
		return "UNIMPLEMENTED"
	case codes.Internal:
		return "INTERNAL"
	case codes.Unavailable:
		return "UNAVAILABLE"
	case codes.DataLoss:
		return "DATA_LOSS"
	case codes.Unauthenticated:
		return "UNAUTHENTICATED"
	default:
		return "CODE(" + strconv.FormatInt(int64(c), 10) + ")"
	}
}
