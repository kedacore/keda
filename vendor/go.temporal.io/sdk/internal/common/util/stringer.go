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

package util

import (
	"bytes"
	"fmt"
	"reflect"

	commandpb "go.temporal.io/api/command/v1"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
)

func anyToString(d interface{}) string {
	v := reflect.ValueOf(d)
	switch v.Kind() {
	case reflect.Ptr:
		return anyToString(v.Elem().Interface())
	case reflect.Struct:
		var buf bytes.Buffer
		t := reflect.TypeOf(d)
		buf.WriteString("(")
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			if f.Kind() == reflect.Invalid {
				continue
			}
			fieldValue := valueToString(f)
			if len(fieldValue) == 0 {
				continue
			}
			if buf.Len() > 1 {
				buf.WriteString(", ")
			}
			buf.WriteString(fmt.Sprintf("%s:%s", t.Field(i).Name, fieldValue))
		}
		buf.WriteString(")")
		return buf.String()
	default:
		return fmt.Sprint(d)
	}
}

func valueToString(v reflect.Value) string {
	switch v.Kind() {
	case reflect.Ptr:
		return valueToString(v.Elem())
	case reflect.Struct:
		if v.CanInterface() {
			return anyToString(v.Interface())
		}
	case reflect.Invalid:
		return ""
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			return fmt.Sprintf("[%v]", string(v.Bytes()))
		}
		return fmt.Sprintf("[len=%d]", v.Len())
	default:
		if v.CanInterface() {
			return fmt.Sprint(v.Interface())
		}
	}

	return ""
}

// HistoryEventToString convert HistoryEvent to string
func HistoryEventToString(e *historypb.HistoryEvent) string {
	var data interface{}
	switch e.GetEventType() {
	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED:
		data = e.GetWorkflowExecutionStartedEventAttributes()

	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED:
		data = e.GetWorkflowExecutionCompletedEventAttributes()

	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_FAILED:
		data = e.GetWorkflowExecutionFailedEventAttributes()

	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_TIMED_OUT:
		data = e.GetWorkflowExecutionTimedOutEventAttributes()

	case enumspb.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED:
		data = e.GetWorkflowTaskScheduledEventAttributes()

	case enumspb.EVENT_TYPE_WORKFLOW_TASK_STARTED:
		data = e.GetWorkflowTaskStartedEventAttributes()

	case enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED:
		data = e.GetWorkflowTaskCompletedEventAttributes()

	case enumspb.EVENT_TYPE_WORKFLOW_TASK_TIMED_OUT:
		data = e.GetWorkflowTaskTimedOutEventAttributes()

	case enumspb.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED:
		data = e.GetActivityTaskScheduledEventAttributes()

	case enumspb.EVENT_TYPE_ACTIVITY_TASK_STARTED:
		data = e.GetActivityTaskStartedEventAttributes()

	case enumspb.EVENT_TYPE_ACTIVITY_TASK_COMPLETED:
		data = e.GetActivityTaskCompletedEventAttributes()

	case enumspb.EVENT_TYPE_ACTIVITY_TASK_FAILED:
		data = e.GetActivityTaskFailedEventAttributes()

	case enumspb.EVENT_TYPE_ACTIVITY_TASK_TIMED_OUT:
		data = e.GetActivityTaskTimedOutEventAttributes()

	case enumspb.EVENT_TYPE_ACTIVITY_TASK_CANCEL_REQUESTED:
		data = e.GetActivityTaskCancelRequestedEventAttributes()

	case enumspb.EVENT_TYPE_ACTIVITY_TASK_CANCELED:
		data = e.GetActivityTaskCanceledEventAttributes()

	case enumspb.EVENT_TYPE_TIMER_STARTED:
		data = e.GetTimerStartedEventAttributes()

	case enumspb.EVENT_TYPE_TIMER_FIRED:
		data = e.GetTimerFiredEventAttributes()

	case enumspb.EVENT_TYPE_TIMER_CANCELED:
		data = e.GetTimerCanceledEventAttributes()

	case enumspb.EVENT_TYPE_MARKER_RECORDED:
		data = e.GetMarkerRecordedEventAttributes()

	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_TERMINATED:
		data = e.GetWorkflowExecutionTerminatedEventAttributes()

	default:
		data = e
	}

	return e.GetEventType().String() + ": " + anyToString(data)
}

// CommandToString convert Command to string
func CommandToString(d *commandpb.Command) string {
	var data interface{}
	switch d.GetCommandType() {
	case enumspb.COMMAND_TYPE_SCHEDULE_ACTIVITY_TASK:
		data = d.GetScheduleActivityTaskCommandAttributes()

	case enumspb.COMMAND_TYPE_REQUEST_CANCEL_ACTIVITY_TASK:
		data = d.GetRequestCancelActivityTaskCommandAttributes()

	case enumspb.COMMAND_TYPE_START_TIMER:
		data = d.GetStartTimerCommandAttributes()

	case enumspb.COMMAND_TYPE_CANCEL_TIMER:
		data = d.GetCancelTimerCommandAttributes()

	case enumspb.COMMAND_TYPE_COMPLETE_WORKFLOW_EXECUTION:
		data = d.GetCompleteWorkflowExecutionCommandAttributes()

	case enumspb.COMMAND_TYPE_FAIL_WORKFLOW_EXECUTION:
		data = d.GetFailWorkflowExecutionCommandAttributes()

	case enumspb.COMMAND_TYPE_RECORD_MARKER:
		data = d.GetRecordMarkerCommandAttributes()

	default:
		data = d
	}

	return d.GetCommandType().String() + ": " + anyToString(data)
}
