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

package internal

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"

	"go.temporal.io/sdk/converter"
)

/*
If activity fails then *ActivityError is returned to the workflow code. The error has important information about activity
and actual error which caused activity failure. This internal error can be unwrapped using errors.Unwrap() or checked using errors.As().
Below are the possible types of internal error:
1) *ApplicationError: (this should be the most common one)
	*ApplicationError can be returned in two cases:
		- If activity implementation returns *ApplicationError by using NewApplicationError() API.
		  The err would contain a message, details, and NonRetryable flag. Workflow code could check this flag and details to determine
		  what kind of error it was and take actions based on it. The details is encoded payload which workflow code could extract
		  to strong typed variable. Workflow code needs to know what the types of the encoded details are before extracting them.
		- If activity implementation returns errors other than from NewApplicationError() API. In this case GetOriginalType()
		  will return orginal type of an error represented as string. Workflow code could check this type to determine what kind of error it was
		  and take actions based on the type. These errors are retryable by default, unless error type is specified in retry policy.
2) *CanceledError:
	If activity was canceled, internal error will be an instance of *CanceledError. When activity cancels itself by
	returning NewCancelError() it would supply optional details which could be extracted by workflow code.
3) *TimeoutError:
	If activity was timed out (several timeout types), internal error will be an instance of *TimeoutError. The err contains
	details about what type of timeout it was.
4) *PanicError:
	If activity code panic while executing, temporal activity worker will report it as activity failure to temporal server.
	The SDK will present that failure as *PanicError. The err contains a string	representation of the panic message and
	the call stack when panic was happen.

Workflow code could handle errors based on different types of error. Below is sample code of how error handling looks like.

err := workflow.ExecuteActivity(ctx, MyActivity, ...).Get(ctx, nil)
if err != nil {
	var applicationErr *ApplicationError
	if errors.As(err, &applicationError) {
		// handle activity errors (created via NewApplicationError() API)
		if !applicationErr.NonRetryable() {
			// manually retry activity
		}
		var detailMsg string // assuming activity return error by NewApplicationError("message", true, "string details")
		applicationErr.Details(&detailMsg) // extract strong typed details

		// handle activity errors (errors created other than using NewApplicationError() API)
		switch err.Type() {
		case "CustomErrTypeA":
			// handle CustomErrTypeA
		case CustomErrTypeB:
			// handle CustomErrTypeB
		default:
			// newer version of activity could return new errors that workflow was not aware of.
		}
	}

	var canceledErr *CanceledError
	if errors.As(err, &canceledErr) {
		// handle cancellation
	}

	var timeoutErr *TimeoutError
	if errors.As(err, &timeoutErr) {
		// handle timeout, could check timeout type by timeoutErr.TimeoutType()
        switch err.TimeoutType() {
        case commonpb.ScheduleToStart:
                // Handle ScheduleToStart timeout.
        case commonpb.StartToClose:
                // Handle StartToClose timeout.
        case commonpb.Heartbeat:
                // Handle heartbeat timeout.
        default:
        }
	}

	var panicErr *PanicError
	if errors.As(err, &panicErr) {
		// handle panic, message and stack trace are available by panicErr.Error() and panicErr.StackTrace()
	}
}

Errors from child workflow should be handled in a similar way, except that instance of *ChildWorkflowExecutionError is returned to
workflow code. It will contains *ActivityError, which in turn will contains on of the errors above.
When panic happen in workflow implementation code, SDK catches that panic and causing the workflow task timeout.
That workflow task will be retried at a later time (with exponential backoff retry intervals).

Workflow consumers will get an instance of *WorkflowExecutionError. This error will contains one of errors above.
*/

type (
	// ApplicationError returned from activity implementations with message and optional details.
	ApplicationError struct {
		temporalError
		msg          string
		errType      string
		nonRetryable bool
		cause        error
		details      converter.EncodedValues
	}

	// TimeoutError returned when activity or child workflow timed out.
	TimeoutError struct {
		temporalError
		msg                  string
		timeoutType          enumspb.TimeoutType
		lastHeartbeatDetails converter.EncodedValues
		cause                error
	}

	// CanceledError returned when operation was canceled.
	CanceledError struct {
		temporalError
		details converter.EncodedValues
	}

	// TerminatedError returned when workflow was terminated.
	TerminatedError struct {
		temporalError
	}

	// PanicError contains information about panicked workflow/activity.
	PanicError struct {
		temporalError
		value      interface{}
		stackTrace string
	}

	// workflowPanicError contains information about panicked workflow.
	// Used to distinguish go panic in the workflow code from a PanicError returned from a workflow function.
	workflowPanicError struct {
		value      interface{}
		stackTrace string
	}

	// ContinueAsNewError contains information about how to continue the workflow as new.
	ContinueAsNewError struct {
		// params *ExecuteWorkflowParams
		WorkflowType        *WorkflowType
		Input               *commonpb.Payloads
		Header              *commonpb.Header
		TaskQueueName       string
		WorkflowRunTimeout  time.Duration
		WorkflowTaskTimeout time.Duration

		// Deprecated: WorkflowExecutionTimeout is deprecated and is never set or
		// used internally.
		WorkflowExecutionTimeout time.Duration

		// VersioningIntent specifies whether the continued workflow should run on a worker with a
		// compatible build ID or not. See VersioningIntent.
		VersioningIntent VersioningIntent
	}

	// UnknownExternalWorkflowExecutionError can be returned when external workflow doesn't exist
	UnknownExternalWorkflowExecutionError struct{}

	// ServerError can be returned from server.
	ServerError struct {
		temporalError
		msg          string
		nonRetryable bool
		cause        error
	}

	// ActivityError is returned from workflow when activity returned an error.
	// Unwrap this error to get actual cause.
	ActivityError struct {
		temporalError
		scheduledEventID int64
		startedEventID   int64
		identity         string
		activityType     *commonpb.ActivityType
		activityID       string
		retryState       enumspb.RetryState
		cause            error
	}

	// ChildWorkflowExecutionError is returned from workflow when child workflow returned an error.
	// Unwrap this error to get actual cause.
	ChildWorkflowExecutionError struct {
		temporalError
		namespace        string
		workflowID       string
		runID            string
		workflowType     string
		initiatedEventID int64
		startedEventID   int64
		retryState       enumspb.RetryState
		cause            error
	}

	// ChildWorkflowExecutionAlreadyStartedError is set as the cause of
	// ChildWorkflowExecutionError when failure is due the child workflow having
	// already started.
	ChildWorkflowExecutionAlreadyStartedError struct{}

	// NamespaceNotFoundError is set as the cause when failure is due namespace not found.
	NamespaceNotFoundError struct{}

	// WorkflowExecutionError is returned from workflow.
	// Unwrap this error to get actual cause.
	WorkflowExecutionError struct {
		workflowID   string
		runID        string
		workflowType string
		cause        error
	}

	// ActivityNotRegisteredError is returned if worker doesn't support activity type.
	ActivityNotRegisteredError struct {
		activityType   string
		supportedTypes []string
	}

	temporalError struct {
		messenger
		originalFailure *failurepb.Failure
	}

	failureHolder interface {
		setFailure(*failurepb.Failure)
		failure() *failurepb.Failure
	}

	messenger interface {
		message() string
	}
)

var (
	// Should be "errorString".
	goErrType = reflect.TypeOf(errors.New("")).Elem().Name()

	// ErrNoData is returned when trying to extract strong typed data while there is no data available.
	ErrNoData = errors.New("no data available")

	// ErrTooManyArg is returned when trying to extract strong typed data with more arguments than available data.
	ErrTooManyArg = errors.New("too many arguments")

	// ErrActivityResultPending is returned from activity's implementation to indicate the activity is not completed when
	// activity method returns. Activity needs to be completed by Client.CompleteActivity() separately. For example, if an
	// activity require human interaction (like approve an expense report), the activity could return activity.ErrResultPending
	// which indicate the activity is not done yet. Then, when the waited human action happened, it needs to trigger something
	// that could report the activity completed event to temporal server via Client.CompleteActivity() API.
	ErrActivityResultPending = errors.New("not error: do not autocomplete, using Client.CompleteActivity() to complete")

	// ErrScheduleAlreadyRunning is returned if there's already a running (not deleted) Schedule with the same ID
	ErrScheduleAlreadyRunning = errors.New("schedule with this ID is already registered")

	// ErrSkipScheduleUpdate is used by a user if they want to skip updating a schedule.
	ErrSkipScheduleUpdate = errors.New("skip schedule update")
)

// NewApplicationError create new instance of *ApplicationError with message, type, and optional details.
func NewApplicationError(msg string, errType string, nonRetryable bool, cause error, details ...interface{}) error {
	applicationErr := &ApplicationError{
		msg:          msg,
		errType:      errType,
		nonRetryable: nonRetryable,
		cause:        cause}

	// When return error to user, use EncodedValues as details and data is ready to be decoded by calling Get
	if len(details) == 1 {
		if d, ok := details[0].(*EncodedValues); ok {
			applicationErr.details = d
			return applicationErr
		}
	}

	// When create error for server, use ErrorDetailsValues as details to hold values and encode later
	applicationErr.details = ErrorDetailsValues(details)
	return applicationErr
}

// NewTimeoutError creates TimeoutError instance.
// Use NewHeartbeatTimeoutError to create heartbeat TimeoutError.
func NewTimeoutError(msg string, timeoutType enumspb.TimeoutType, cause error, lastHeartbeatDetails ...interface{}) error {
	timeoutErr := &TimeoutError{
		msg:         msg,
		timeoutType: timeoutType,
		cause:       cause,
	}

	if len(lastHeartbeatDetails) == 1 {
		if d, ok := lastHeartbeatDetails[0].(*EncodedValues); ok {
			timeoutErr.lastHeartbeatDetails = d
			return timeoutErr
		}
	}
	timeoutErr.lastHeartbeatDetails = ErrorDetailsValues(lastHeartbeatDetails)
	return timeoutErr
}

// NewHeartbeatTimeoutError creates TimeoutError instance.
func NewHeartbeatTimeoutError(details ...interface{}) error {
	return NewTimeoutError("heartbeat timeout", enumspb.TIMEOUT_TYPE_HEARTBEAT, nil, details...)
}

// NewCanceledError creates CanceledError instance.
func NewCanceledError(details ...interface{}) error {
	if len(details) == 1 {
		if d, ok := details[0].(*EncodedValues); ok {
			return &CanceledError{details: d}
		}
	}
	return &CanceledError{details: ErrorDetailsValues(details)}
}

// NewServerError create new instance of *ServerError with message.
func NewServerError(msg string, nonRetryable bool, cause error) error {
	return &ServerError{msg: msg, nonRetryable: nonRetryable, cause: cause}
}

// NewActivityError creates ActivityError instance.
func NewActivityError(
	scheduledEventID int64,
	startedEventID int64,
	identity string,
	activityType *commonpb.ActivityType,
	activityID string,
	retryState enumspb.RetryState,
	cause error,
) *ActivityError {
	return &ActivityError{
		scheduledEventID: scheduledEventID,
		startedEventID:   startedEventID,
		identity:         identity,
		activityType:     activityType,
		activityID:       activityID,
		retryState:       retryState,
		cause:            cause,
	}
}

// NewChildWorkflowExecutionError creates ChildWorkflowExecutionError instance.
func NewChildWorkflowExecutionError(
	namespace string,
	workflowID string,
	runID string,
	workflowType string,
	initiatedEventID int64,
	startedEventID int64,
	retryState enumspb.RetryState,
	cause error,
) *ChildWorkflowExecutionError {
	return &ChildWorkflowExecutionError{
		namespace:        namespace,
		workflowID:       workflowID,
		runID:            runID,
		workflowType:     workflowType,
		initiatedEventID: initiatedEventID,
		startedEventID:   startedEventID,
		retryState:       retryState,
		cause:            cause,
	}
}

// NewWorkflowExecutionError creates WorkflowExecutionError instance.
func NewWorkflowExecutionError(
	workflowID string,
	runID string,
	workflowType string,
	cause error,
) *WorkflowExecutionError {
	return &WorkflowExecutionError{
		workflowID:   workflowID,
		runID:        runID,
		workflowType: workflowType,
		cause:        cause,
	}
}

func (e *temporalError) setFailure(f *failurepb.Failure) {
	e.originalFailure = f
}

func (e *temporalError) failure() *failurepb.Failure {
	return e.originalFailure
}

// IsCanceledError returns whether error in CanceledError.
func IsCanceledError(err error) bool {
	var canceledErr *CanceledError
	return errors.As(err, &canceledErr)
}

// NewContinueAsNewError creates ContinueAsNewError instance
// If the workflow main function returns this error then the current execution is ended and
// the new execution with same workflow ID is started automatically with options
// provided to this function.
//
//	 ctx - use context to override any options for the new workflow like run timeout, task timeout, task queue.
//		  if not mentioned it would use the defaults that the current workflow is using.
//	       ctx := WithWorkflowRunTimeout(ctx, 30 * time.Minute)
//	       ctx := WithWorkflowTaskTimeout(ctx, 5 * time.Second)
//		  ctx := WithWorkflowTaskQueue(ctx, "example-group")
//	 wfn - workflow function. for new execution it can be different from the currently running.
//	 args - arguments for the new workflow.
func NewContinueAsNewError(ctx Context, wfn interface{}, args ...interface{}) error {
	i := getWorkflowOutboundInterceptor(ctx)
	// Put header on context before executing
	ctx = workflowContextWithNewHeader(ctx)
	return i.NewContinueAsNewError(ctx, wfn, args...)
}

func (wc *workflowEnvironmentInterceptor) NewContinueAsNewError(
	ctx Context,
	wfn interface{},
	args ...interface{},
) error {
	// Validate type and its arguments.
	options := getWorkflowEnvOptions(ctx)
	if options == nil {
		panic("context is missing required options for continue as new")
	}
	env := getWorkflowEnvironment(ctx)
	workflowType, input, err := getValidatedWorkflowFunction(wfn, args, options.DataConverter, env.GetRegistry())
	if err != nil {
		panic(err)
	}

	header, err := workflowHeaderPropagated(ctx, options.ContextPropagators)
	if err != nil {
		return err
	}

	return &ContinueAsNewError{
		WorkflowType:             workflowType,
		Input:                    input,
		Header:                   header,
		TaskQueueName:            options.TaskQueueName,
		WorkflowExecutionTimeout: options.WorkflowExecutionTimeout,
		WorkflowRunTimeout:       options.WorkflowRunTimeout,
		WorkflowTaskTimeout:      options.WorkflowTaskTimeout,
		VersioningIntent:         options.VersioningIntent,
	}
}

// NewActivityNotRegisteredError creates a new ActivityNotRegisteredError.
func NewActivityNotRegisteredError(activityType string, supportedTypes []string) error {
	return &ActivityNotRegisteredError{activityType: activityType, supportedTypes: supportedTypes}
}

// Error from error interface.
func (e *ApplicationError) Error() string {
	msg := e.message()
	if e.errType != "" {
		msg = fmt.Sprintf("%s (type: %s, retryable: %v)", msg, e.errType, !e.nonRetryable)
	}
	if e.cause != nil {
		msg = fmt.Sprintf("%s: %v", msg, e.cause)
	}
	return msg
}

func (e *ApplicationError) message() string {
	return e.msg
}

// Message contains just the message string without extras added by Error().
func (e *ApplicationError) Message() string {
	return e.msg
}

// Type returns error type represented as string.
// This type can be passed explicitly to ApplicationError constructor.
// Also any other Go error is converted to ApplicationError and type is set automatically using reflection.
// For example instance of "MyCustomError struct" will be converted to ApplicationError and Type() will return "MyCustomError" string.
func (e *ApplicationError) Type() string {
	return e.errType
}

// HasDetails return if this error has strong typed detail data.
func (e *ApplicationError) HasDetails() bool {
	return e.details != nil && e.details.HasValues()
}

// Details extracts strong typed detail data of this custom error. If there is no details, it will return ErrNoData.
func (e *ApplicationError) Details(d ...interface{}) error {
	if !e.HasDetails() {
		return ErrNoData
	}
	return e.details.Get(d...)
}

// NonRetryable indicated if error is not retryable.
func (e *ApplicationError) NonRetryable() bool {
	return e.nonRetryable
}

func (e *ApplicationError) Unwrap() error {
	return e.cause
}

// Error from error interface
func (e *TimeoutError) Error() string {
	msg := fmt.Sprintf("%s (type: %v)", e.message(), e.timeoutType)
	if e.cause != nil {
		msg = fmt.Sprintf("%s: %v", msg, e.cause)
	}
	return msg
}

func (e *TimeoutError) message() string {
	return e.msg
}

// Message contains just the message string without extras added by Error().
func (e *TimeoutError) Message() string {
	return e.msg
}

func (e *TimeoutError) Unwrap() error {
	return e.cause
}

// TimeoutType return timeout type of this error
func (e *TimeoutError) TimeoutType() enumspb.TimeoutType {
	return e.timeoutType
}

// HasLastHeartbeatDetails return if this error has strong typed detail data.
func (e *TimeoutError) HasLastHeartbeatDetails() bool {
	return e.lastHeartbeatDetails != nil && e.lastHeartbeatDetails.HasValues()
}

// LastHeartbeatDetails extracts strong typed detail data of this error. If there is no details, it will return ErrNoData.
func (e *TimeoutError) LastHeartbeatDetails(d ...interface{}) error {
	if !e.HasLastHeartbeatDetails() {
		return ErrNoData
	}
	return e.lastHeartbeatDetails.Get(d...)
}

// Error from error interface
func (e *CanceledError) Error() string {
	return e.message()
}

func (e *CanceledError) message() string {
	return "canceled"
}

// HasDetails return if this error has strong typed detail data.
func (e *CanceledError) HasDetails() bool {
	return e.details != nil && e.details.HasValues()
}

// Details extracts strong typed detail data of this error.
func (e *CanceledError) Details(d ...interface{}) error {
	if !e.HasDetails() {
		return ErrNoData
	}
	return e.details.Get(d...)
}

func newPanicError(value interface{}, stackTrace string) error {
	return &PanicError{value: value, stackTrace: stackTrace}
}

func newWorkflowPanicError(value interface{}, stackTrace string) error {
	return &workflowPanicError{value: value, stackTrace: stackTrace}
}

// Error from error interface
func (e *PanicError) Error() string {
	return e.message()
}

func (e *PanicError) message() string {
	return fmt.Sprintf("%v", e.value)
}

// StackTrace return stack trace of the panic
func (e *PanicError) StackTrace() string {
	return e.stackTrace
}

// Error from error interface
func (e *workflowPanicError) Error() string {
	return fmt.Sprintf("%v", e.value)
}

// StackTrace return stack trace of the panic
func (e *workflowPanicError) StackTrace() string {
	return e.stackTrace
}

// Error from error interface
func (e *ContinueAsNewError) Error() string {
	return e.message()
}

func (e *ContinueAsNewError) message() string {
	return "continue as new"
}

// newTerminatedError creates NewTerminatedError instance
func newTerminatedError() *TerminatedError {
	return &TerminatedError{}
}

// Error from error interface
func (e *TerminatedError) Error() string {
	return e.message()
}

func (e *TerminatedError) message() string {
	return "terminated"
}

// newUnknownExternalWorkflowExecutionError creates UnknownExternalWorkflowExecutionError instance
func newUnknownExternalWorkflowExecutionError() *UnknownExternalWorkflowExecutionError {
	return &UnknownExternalWorkflowExecutionError{}
}

// Error from error interface
func (e *UnknownExternalWorkflowExecutionError) Error() string {
	return "unknown external workflow execution"
}

// Error from error interface
func (e *ServerError) Error() string {
	msg := e.message()
	if e.cause != nil {
		msg = fmt.Sprintf("%s: %v", msg, e.cause)
	}
	return msg
}

func (e *ServerError) message() string {
	return e.msg
}

// Message contains just the message string without extras added by Error().
func (e *ServerError) Message() string {
	return e.msg
}

func (e *ServerError) Unwrap() error {
	return e.cause
}

func (e *ActivityError) Error() string {
	msg := fmt.Sprintf("%s (type: %s, scheduledEventID: %d, startedEventID: %d, identity: %s)", e.message(), e.activityType.GetName(), e.scheduledEventID, e.startedEventID, e.identity)
	if e.cause != nil {
		msg = fmt.Sprintf("%s: %v", msg, e.cause)
	}
	return msg
}

func (e *ActivityError) message() string {
	return "activity error"
}

func (e *ActivityError) Unwrap() error {
	return e.cause
}

// ScheduledEventID returns event id of the scheduled workflow task corresponding to the activity.
func (e *ActivityError) ScheduledEventID() int64 {
	return e.scheduledEventID
}

// StartedEventID returns event id of the started workflow task corresponding to the activity.
func (e *ActivityError) StartedEventID() int64 {
	return e.startedEventID
}

// Identity returns identity of the worker that attempted activity execution.
func (e *ActivityError) Identity() string {
	return e.identity
}

// ActivityType returns declared type of the activity.
func (e *ActivityError) ActivityType() *commonpb.ActivityType {
	return e.activityType
}

// ActivityID return assigned identifier for the activity.
func (e *ActivityError) ActivityID() string {
	return e.activityID
}

// RetryState returns details on why activity failed.
func (e *ActivityError) RetryState() enumspb.RetryState {
	return e.retryState
}

// Error from error interface
func (e *ChildWorkflowExecutionError) Error() string {
	msg := fmt.Sprintf("%s (type: %s, workflowID: %s, runID: %s, initiatedEventID: %d, startedEventID: %d)",
		e.message(), e.workflowType, e.workflowID, e.runID, e.initiatedEventID, e.startedEventID)
	if e.cause != nil {
		msg = fmt.Sprintf("%s: %v", msg, e.cause)
	}
	return msg
}

func (e *ChildWorkflowExecutionError) message() string {
	return "child workflow execution error"
}

func (e *ChildWorkflowExecutionError) Unwrap() error {
	return e.cause
}

// Error from error interface
func (*NamespaceNotFoundError) Error() string {
	return "namespace not found"
}

// Error from error interface
func (*ChildWorkflowExecutionAlreadyStartedError) Error() string {
	return "child workflow execution already started"
}

// Error from error interface
func (e *WorkflowExecutionError) Error() string {
	msg := fmt.Sprintf("workflow execution error (type: %s, workflowID: %s, runID: %s)",
		e.workflowType, e.workflowID, e.runID)
	if e.cause != nil {
		msg = fmt.Sprintf("%s: %v", msg, e.cause)
	}
	return msg
}

func (e *WorkflowExecutionError) Unwrap() error {
	return e.cause
}

func (e *ActivityNotRegisteredError) Error() string {
	supported := strings.Join(e.supportedTypes, ", ")
	return fmt.Sprintf("unable to find activityType=%v. Supported types: [%v]", e.activityType, supported)
}

func convertErrDetailsToPayloads(details converter.EncodedValues, dc converter.DataConverter) *commonpb.Payloads {
	switch d := details.(type) {
	case ErrorDetailsValues:
		data, err := encodeArgs(dc, d)
		if err != nil {
			panic(err)
		}
		return data
	case *EncodedValues:
		return d.values
	default:
		panic(fmt.Sprintf("unknown error details type %T", details))
	}
}

// IsRetryable returns if error retryable or not.
func IsRetryable(err error, nonRetryableTypes []string) bool {
	if err == nil {
		return false
	}

	var terminatedErr *TerminatedError
	var canceledErr *CanceledError
	var workflowPanicErr *workflowPanicError
	if errors.As(err, &terminatedErr) || errors.As(err, &canceledErr) || errors.As(err, &workflowPanicErr) {
		return false
	}

	var timeoutErr *TimeoutError
	if errors.As(err, &timeoutErr) {
		return timeoutErr.timeoutType == enumspb.TIMEOUT_TYPE_START_TO_CLOSE || timeoutErr.timeoutType == enumspb.TIMEOUT_TYPE_HEARTBEAT
	}

	var applicationErr *ApplicationError
	var errType string
	if errors.As(err, &applicationErr) {
		if applicationErr.nonRetryable {
			return false
		}
		errType = applicationErr.errType
	} else {
		// If it is generic Go error.
		errType = getErrType(err)
	}

	for _, nonRetryableType := range nonRetryableTypes {
		if nonRetryableType == errType {
			return false
		}
	}

	return true
}

func getErrType(err error) string {
	var t reflect.Type
	for t = reflect.TypeOf(err); t.Kind() == reflect.Ptr; t = t.Elem() {
	}

	if t.Name() == goErrType {
		return ""
	}

	return t.Name()
}
