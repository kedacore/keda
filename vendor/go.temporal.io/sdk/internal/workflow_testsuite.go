package internal

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"

	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/internal/common/metrics"
	"go.temporal.io/sdk/log"
)

type (
	// EncodedValues is a type alias used to encapsulate/extract encoded arguments from workflow/activity.
	EncodedValues struct {
		values        *commonpb.Payloads
		dataConverter converter.DataConverter
	}

	// ErrorDetailsValues is a type alias used hold error details objects.
	ErrorDetailsValues []interface{}

	// WorkflowTestSuite is the test suite to run unit tests for workflow/activity.
	//
	// Exposed as: [go.temporal.io/sdk/testsuite.WorkflowTestSuite]
	WorkflowTestSuite struct {
		logger                      log.Logger
		metricsHandler              metrics.Handler
		contextPropagators          []ContextPropagator
		header                      *commonpb.Header
		disableRegistrationAliasing bool
	}

	// TestWorkflowEnvironment is the environment that you use to test workflow
	//
	// Exposed as: [go.temporal.io/sdk/testsuite.TestWorkflowEnvironment]
	TestWorkflowEnvironment struct {
		workflowMock mock.Mock
		activityMock mock.Mock
		nexusMock    mock.Mock
		impl         *testWorkflowEnvironmentImpl
	}

	// TestActivityEnvironment is the environment that you use to test activity
	//
	// Exposed as: [go.temporal.io/sdk/testsuite.TestActivityEnvironment]
	TestActivityEnvironment struct {
		impl *testWorkflowEnvironmentImpl
	}

	// MockCallWrapper is a wrapper to mock.Call. It offers the ability to wait on workflow's clock instead of wall clock.
	//
	// Exposed as: [go.temporal.io/sdk/testsuite.MockCallWrapper]
	MockCallWrapper struct {
		call *mock.Call
		env  *TestWorkflowEnvironment

		runFn        func(args mock.Arguments)
		waitDuration func() time.Duration
	}

	// TestUpdateCallback is a basic implementation of the UpdateCallbacks interface for testing purposes.
	// Tests are welcome to implement their own version of this interface if they need to test more complex
	// update logic. This is a simple implementation to make testing basic Workflow Updates easier.
	//
	// Note: If any of the three fields are omitted, a no-op implementation will be used by default.
	//
	// Exposed as: [go.temporal.io/sdk/testsuite.TestUpdateCallback]
	TestUpdateCallback struct {
		OnAccept   func()
		OnReject   func(error)
		OnComplete func(interface{}, error)
	}
)

func newEncodedValues(values *commonpb.Payloads, dc converter.DataConverter) converter.EncodedValues {
	if dc == nil {
		dc = converter.GetDefaultDataConverter()
	}
	return &EncodedValues{values, dc}
}

// Get extract data from encoded data to desired value type. valuePtr is pointer to the actual value type.
func (b EncodedValues) Get(valuePtr ...interface{}) error {
	if !b.HasValues() {
		return ErrNoData
	}
	return b.dataConverter.FromPayloads(b.values, valuePtr...)
}

// HasValues return whether there are values
func (b EncodedValues) HasValues() bool {
	return b.values != nil
}

// Get extract data from encoded data to desired value type. valuePtr is pointer to the actual value type.
func (b ErrorDetailsValues) Get(valuePtr ...interface{}) error {
	if !b.HasValues() {
		return ErrNoData
	}
	if len(valuePtr) > len(b) {
		return ErrTooManyArg
	}
	for i, item := range valuePtr {
		reflect.ValueOf(item).Elem().Set(reflect.ValueOf(b[i]))
	}
	return nil
}

// HasValues return whether there are values.
func (b ErrorDetailsValues) HasValues() bool {
	return len(b) != 0
}

// NewTestWorkflowEnvironment creates a new instance of TestWorkflowEnvironment. Use the returned TestWorkflowEnvironment
// to run your workflow in the test environment.
func (s *WorkflowTestSuite) NewTestWorkflowEnvironment() *TestWorkflowEnvironment {
	return &TestWorkflowEnvironment{impl: newTestWorkflowEnvironmentImpl(s, nil)}
}

// NewTestActivityEnvironment creates a new instance of TestActivityEnvironment. Use the returned TestActivityEnvironment
// to run your activity in the test environment.
func (s *WorkflowTestSuite) NewTestActivityEnvironment() *TestActivityEnvironment {
	t := &TestActivityEnvironment{impl: newTestWorkflowEnvironmentImpl(s, nil)}
	t.impl.activityEnvOnly = true
	return t
}

// SetLogger sets the logger for this WorkflowTestSuite. If you don't set logger, test suite will create a default logger
// with Debug level logging enabled.
func (s *WorkflowTestSuite) SetLogger(logger log.Logger) {
	s.logger = logger
}

// GetLogger gets the logger for this WorkflowTestSuite.
func (s *WorkflowTestSuite) GetLogger() log.Logger {
	return s.logger
}

// SetMetricsHandler sets the metrics handler for this WorkflowTestSuite. If you don't set handler, test suite will use
// a noop handler.
func (s *WorkflowTestSuite) SetMetricsHandler(metricsHandler metrics.Handler) {
	s.metricsHandler = metricsHandler
}

// SetContextPropagators sets the context propagators for this WorkflowTestSuite. If you don't set context propagators,
// test suite will not use context propagators
func (s *WorkflowTestSuite) SetContextPropagators(ctxProps []ContextPropagator) {
	s.contextPropagators = ctxProps
}

// SetHeader sets the headers for this WorkflowTestSuite. If you don't set header, test suite will not pass headers to
// the workflow
func (s *WorkflowTestSuite) SetHeader(header *commonpb.Header) {
	s.header = header
}

// SetDisableRegistrationAliasing disables registration aliasing the same way it
// is disabled when set for worker.Options.DisableRegistrationAliasing. This
// value should be set to true if it is expected to be set on the worker when
// running (which is strongly recommended for custom-named workflows and
// activities). See the documentation on
// worker.Options.DisableRegistrationAliasing for more details.
//
// This must be set before obtaining new test workflow or activity environments.
func (s *WorkflowTestSuite) SetDisableRegistrationAliasing(disableRegistrationAliasing bool) {
	s.disableRegistrationAliasing = disableRegistrationAliasing
}

// RegisterActivity registers activity implementation with TestWorkflowEnvironment
func (t *TestActivityEnvironment) RegisterActivity(a interface{}) {
	t.impl.RegisterActivity(a)
}

// RegisterActivityWithOptions registers activity implementation with TestWorkflowEnvironment
func (t *TestActivityEnvironment) RegisterActivityWithOptions(a interface{}, options RegisterActivityOptions) {
	t.impl.RegisterActivityWithOptions(a, options)
}

// ExecuteActivity executes an activity. The tested activity will be executed synchronously in the calling goroutinue.
// Caller should use EncodedValue.Get() to extract strong typed result value.
func (t *TestActivityEnvironment) ExecuteActivity(activityFn interface{}, args ...interface{}) (converter.EncodedValue, error) {
	return t.impl.executeActivity(activityFn, args...)
}

// ExecuteLocalActivity executes a local activity. The tested activity will be executed synchronously in the calling goroutinue.
// Caller should use EncodedValue.Get() to extract strong typed result value.
func (t *TestActivityEnvironment) ExecuteLocalActivity(activityFn interface{}, args ...interface{}) (val converter.EncodedValue, err error) {
	return t.impl.executeLocalActivity(activityFn, args...)
}

// SetWorkerOptions sets the WorkerOptions that will be use by TestActivityEnvironment. TestActivityEnvironment will
// use options of BackgroundActivityContext, MaxConcurrentSessionExecutionSize, and WorkflowInterceptorChainFactories on the WorkerOptions.
// Other options are ignored.
//
// Note: WorkerOptions is defined in internal package, use public type worker.Options instead.
func (t *TestActivityEnvironment) SetWorkerOptions(options WorkerOptions) *TestActivityEnvironment {
	t.impl.setWorkerOptions(options)
	return t
}

// SetDataConverter sets data converter.
func (t *TestActivityEnvironment) SetDataConverter(dataConverter converter.DataConverter) *TestActivityEnvironment {
	t.impl.setDataConverter(dataConverter)
	return t
}

// SetFailureConverter sets the failure converter.
func (t *TestActivityEnvironment) SetFailureConverter(failureConverter converter.FailureConverter) *TestActivityEnvironment {
	t.impl.setFailureConverter(failureConverter)
	return t
}

// SetIdentity sets identity.
func (t *TestActivityEnvironment) SetIdentity(identity string) *TestActivityEnvironment {
	t.impl.setIdentity(identity)
	return t
}

// SetContextPropagators sets context propagators.
func (t *TestActivityEnvironment) SetContextPropagators(contextPropagators []ContextPropagator) *TestActivityEnvironment {
	t.impl.setContextPropagators(contextPropagators)
	return t
}

// SetHeader sets header.
func (t *TestActivityEnvironment) SetHeader(header *commonpb.Header) {
	t.impl.header = header
}

// SetTestTimeout sets the wall clock timeout for this activity test run. When test timeout happen, it means activity is
// taking too long.
func (t *TestActivityEnvironment) SetTestTimeout(idleTimeout time.Duration) *TestActivityEnvironment {
	t.impl.testTimeout = idleTimeout
	return t
}

// SetHeartbeatDetails sets the heartbeat details to be returned from activity.GetHeartbeatDetails()
func (t *TestActivityEnvironment) SetHeartbeatDetails(details interface{}) {
	t.impl.setHeartbeatDetails(details)
}

// SetWorkerStopChannel sets the worker stop channel to be returned from activity.GetWorkerStopChannel(context)
// To test your activity on worker stop, you can provide a go channel with this function and call ExecuteActivity().
// Then call close(channel) to test the activity worker stop logic.
func (t *TestActivityEnvironment) SetWorkerStopChannel(c chan struct{}) {
	t.impl.setWorkerStopChannel(c)
}

// SetOnActivityHeartbeatListener sets a listener that will be called when
// activity heartbeat is called. ActivityInfo is defined in internal package,
// use public type activity.Info instead.
//
// Note: The provided listener may be called concurrently.
//
// Note: Due to internal caching by the activity system, this may not get called
// for every heartbeat recorded. This is only called when the heartbeat would be
// sent to the server (periodic batch and at the end only on failure).
// Interceptors can be used to intercept/check every heartbeat call.
func (t *TestActivityEnvironment) SetOnActivityHeartbeatListener(
	listener func(activityInfo *ActivityInfo, details converter.EncodedValues)) *TestActivityEnvironment {
	t.impl.onActivityHeartbeatListener = listener
	return t
}

// RegisterWorkflow registers workflow implementation with the TestWorkflowEnvironment
func (e *TestWorkflowEnvironment) RegisterWorkflow(w interface{}) {
	e.impl.RegisterWorkflow(w)
}

// RegisterWorkflowWithOptions registers workflow implementation with the TestWorkflowEnvironment
func (e *TestWorkflowEnvironment) RegisterWorkflowWithOptions(w interface{}, options RegisterWorkflowOptions) {
	if len(e.workflowMock.ExpectedCalls) > 0 {
		panic("RegisterWorkflow calls cannot follow mock related ones like OnWorkflow or similar")
	}
	e.impl.RegisterWorkflowWithOptions(w, options)
}

// RegisterDynamicWorkflow registers a dynamic workflow implementation with the TestWorkflowEnvironment
func (e *TestWorkflowEnvironment) RegisterDynamicWorkflow(w interface{}, options DynamicRegisterWorkflowOptions) {
	if len(e.workflowMock.ExpectedCalls) > 0 {
		panic("RegisterDynamicWorkflow calls cannot follow mock related ones like OnWorkflow or similar")
	}
	e.impl.RegisterDynamicWorkflow(w, options)
}

// RegisterActivity registers activity implementation with TestWorkflowEnvironment
func (e *TestWorkflowEnvironment) RegisterActivity(a interface{}) {
	e.impl.RegisterActivity(a)
}

// RegisterActivityWithOptions registers activity implementation with TestWorkflowEnvironment
func (e *TestWorkflowEnvironment) RegisterActivityWithOptions(a interface{}, options RegisterActivityOptions) {
	if len(e.activityMock.ExpectedCalls) > 0 {
		panic("RegisterActivity calls cannot follow mock related ones like OnActivity or similar")
	}
	e.impl.RegisterActivityWithOptions(a, options)
}

// RegisterDynamicActivity registers the dynamic activity implementation with the TestWorkflowEnvironment
func (e *TestWorkflowEnvironment) RegisterDynamicActivity(a interface{}, options DynamicRegisterActivityOptions) {
	if len(e.workflowMock.ExpectedCalls) > 0 {
		panic("RegisterDynamicActivity calls cannot follow mock related ones like OnWorkflow or similar")
	}
	e.impl.RegisterDynamicActivity(a, options)
}

// RegisterNexusService registers a Nexus Service with the TestWorkflowEnvironment.
func (e *TestWorkflowEnvironment) RegisterNexusService(s *nexus.Service) {
	e.impl.RegisterNexusService(s)
}

// SetStartTime sets the start time of the workflow. This is optional, default start time will be the wall clock time when
// workflow starts. Start time is the workflow.Now(ctx) time at the beginning of the workflow.
func (e *TestWorkflowEnvironment) SetStartTime(startTime time.Time) {
	e.impl.setStartTime(startTime)
}

// SetCurrentHistoryLength sets the value that is returned from
// GetInfo(ctx).GetCurrentHistoryLength().
//
// Note: this value may not be up to date if accessed inside a query.
func (e *TestWorkflowEnvironment) SetCurrentHistoryLength(length int) {
	e.impl.setCurrentHistoryLength(length)
}

// setCurrentHistoryLength sets the value that is returned from
// GetInfo(ctx).GetCurrentHistorySize().
//
// Note: this value may not be up to date if accessed inside a query.
func (e *TestWorkflowEnvironment) SetCurrentHistorySize(length int) {
	e.impl.setCurrentHistorySize(length)
}

// SetContinueAsNewSuggested set sets the value that is returned from
// GetInfo(ctx).GetContinueAsNewSuggested().
//
// Note: this value may not be up to date if accessed inside a query.
func (e *TestWorkflowEnvironment) SetContinueAsNewSuggested(suggest bool) {
	e.impl.setContinueAsNewSuggested(suggest)
}

// SetContinuedExecutionRunID sets the value that is returned from
// GetInfo(ctx).ContinuedExecutionRunID
func (e *TestWorkflowEnvironment) SetContinuedExecutionRunID(rid string) {
	e.impl.setContinuedExecutionRunID(rid)
}

// InOrderMockCalls declares that the given calls should occur in order. Syntax sugar for NotBefore.
func (e *TestWorkflowEnvironment) InOrderMockCalls(calls ...*MockCallWrapper) {
	wrappedCalls := make([]*mock.Call, 0, len(calls))
	for _, call := range calls {
		wrappedCalls = append(wrappedCalls, call.call)
	}

	mock.InOrder(wrappedCalls...)
}

// OnActivity setup a mock call for activity. Parameter activity must be activity function (func) or activity name (string).
// You must call Return() with appropriate parameters on the returned *MockCallWrapper instance. The supplied parameters to
// the Return() call should either be a function that has exact same signature as the mocked activity, or it should be
// mock values with the same types as the mocked activity function returns.
// Example: assume the activity you want to mock has function signature as:
//
//	func MyActivity(ctx context.Context, msg string) (string, error)
//
// You can mock it by return a function with exact same signature:
//
//	t.OnActivity(MyActivity, mock.Anything, mock.Anything).Return(func(ctx context.Context, msg string) (string, error) {
//	   // your mock function implementation
//	   return "", nil
//	})
//
// OR return mock values with same types as activity function's return types:
//
//	t.OnActivity(MyActivity, mock.Anything, mock.Anything).Return("mock_result", nil)
//
// Note, when using a method reference with a receiver as an activity, the receiver must be an instance the same as if
// it was being using in RegisterActivity so the parameter types are accurate. In Go, a method reference of
// (*MyStruct).MyFunc makes the first parameter *MyStruct which will not work, whereas a method reference of
// new(MyStruct).MyFunc will.
//
// Mock callbacks here are run on a separate goroutine than the workflow and
// therefore are not concurrency-safe with workflow code.
func (e *TestWorkflowEnvironment) OnActivity(activity interface{}, args ...interface{}) *MockCallWrapper {
	fType := reflect.TypeOf(activity)
	var call *mock.Call
	switch fType.Kind() {
	case reflect.Func:
		fnType := reflect.TypeOf(activity)
		if err := validateFnFormat(fnType, false, false); err != nil {
			panic(err)
		}
		fnName := getActivityFunctionName(e.impl.registry, activity)
		e.impl.registry.RegisterActivityWithOptions(activity, RegisterActivityOptions{DisableAlreadyRegisteredCheck: true})
		call = e.activityMock.On(fnName, args...)

	case reflect.String:
		name := activity.(string)
		_, ok := e.impl.registry.GetActivity(name)
		if !ok {
			registered := strings.Join(e.impl.registry.getRegisteredActivityTypes(), ", ")
			panic(fmt.Sprintf("activity \""+name+"\" is not registered with the TestWorkflowEnvironment, "+
				"registered types are: %v", registered))
		}
		call = e.activityMock.On(name, args...)
	default:
		panic("activity must be function or string")
	}

	return e.wrapActivityCall(call)
}

// ErrMockStartChildWorkflowFailed is special error used to indicate the mocked child workflow should fail to start.
// This error is also exposed as public as testsuite.ErrMockStartChildWorkflowFailed
//
// Exposed as: [go.temporal.io/sdk/testsuite.ErrMockStartChildWorkflowFailed]
var ErrMockStartChildWorkflowFailed = fmt.Errorf("start child workflow failed: %v", enumspb.START_CHILD_WORKFLOW_EXECUTION_FAILED_CAUSE_WORKFLOW_ALREADY_EXISTS)

// OnWorkflow setup a mock call for workflow. Parameter workflow must be workflow function (func) or workflow name (string).
// You must call Return() with appropriate parameters on the returned *MockCallWrapper instance. The supplied parameters to
// the Return() call should either be a function that has exact same signature as the mocked workflow, or it should be
// mock values with the same types as the mocked workflow function returns.
// Example: assume the workflow you want to mock has function signature as:
//
//	func MyChildWorkflow(ctx workflow.Context, msg string) (string, error)
//
// You can mock it by return a function with exact same signature:
//
//	t.OnWorkflow(MyChildWorkflow, mock.Anything, mock.Anything).Return(func(ctx workflow.Context, msg string) (string, error) {
//	   // your mock function implementation
//	   return "", nil
//	})
//
// OR return mock values with same types as workflow function's return types:
//
//	t.OnWorkflow(MyChildWorkflow, mock.Anything, mock.Anything).Return("mock_result", nil)
//
// You could also setup mock to simulate start child workflow failure case by returning ErrMockStartChildWorkflowFailed
// as error.
//
// Mock callbacks here are run on a separate goroutine than the workflow and
// therefore are not concurrency-safe with workflow code.
func (e *TestWorkflowEnvironment) OnWorkflow(workflow interface{}, args ...interface{}) *MockCallWrapper {
	fType := reflect.TypeOf(workflow)
	var call *mock.Call
	switch fType.Kind() {
	case reflect.Func:
		if err := validateFnFormat(fType, true, false); err != nil {
			panic(err)
		}
		fnName, _ := getWorkflowFunctionName(e.impl.registry, workflow)
		if alias, ok := e.impl.registry.getWorkflowAlias(fnName); ok {
			fnName = alias
		}
		e.impl.registry.RegisterWorkflowWithOptions(workflow, RegisterWorkflowOptions{DisableAlreadyRegisteredCheck: true})
		call = e.workflowMock.On(fnName, args...)
	case reflect.String:
		call = e.workflowMock.On(workflow.(string), args...)
	default:
		panic("workflow must be function or string")
	}

	return e.wrapWorkflowCall(call)
}

const mockMethodForSignalExternalWorkflow = "workflow.SignalExternalWorkflow"
const mockMethodForRequestCancelExternalWorkflow = "workflow.RequestCancelExternalWorkflow"
const mockMethodForGetVersion = "workflow.GetVersion"
const mockMethodForUpsertSearchAttributes = "workflow.UpsertSearchAttributes"
const mockMethodForUpsertTypedSearchAttributes = "workflow.UpsertTypedSearchAttributes"
const mockMethodForUpsertMemo = "workflow.UpsertMemo"

// OnSignalExternalWorkflow setup a mock for sending signal to external workflow.
// This TestWorkflowEnvironment handles sending signals between the workflows that are started from the root workflow.
// For example, sending signals between parent and child workflows. Or sending signals between 2 child workflows.
// However, it does not know what to do if your tested workflow code is sending signal to external unknown workflows.
// In that case, you will need to setup mock for those signal calls.
// Some examples of how to setup mock:
//
//   - mock for specific target workflow that matches specific signal name and signal data
//     env.OnSignalExternalWorkflow("test-namespace", "test-workflow-id1", "test-runid1", "test-signal", "test-data").Return(nil).Once()
//   - mock for anything and succeed the send
//     env.OnSignalExternalWorkflow(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
//   - mock for anything and fail the send
//     env.OnSignalExternalWorkflow(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("unknown external workflow")).Once()
//   - mock function for SignalExternalWorkflow
//     env.OnSignalExternalWorkflow(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
//     func(namespace, workflowID, runID, signalName string, arg interface{}) error {
//     // you can do differently based on the parameters
//     return nil
//     })
//
// Mock callbacks here are run on a separate goroutine than the workflow and
// therefore are not concurrency-safe with workflow code.
func (e *TestWorkflowEnvironment) OnSignalExternalWorkflow(namespace, workflowID, runID, signalName, arg interface{}) *MockCallWrapper {
	call := e.workflowMock.On(mockMethodForSignalExternalWorkflow, namespace, workflowID, runID, signalName, arg)
	return e.wrapWorkflowCall(call)
}

// OnRequestCancelExternalWorkflow setup a mock for cancellation of external workflow.
// This TestWorkflowEnvironment handles cancellation of workflows that are started from the root workflow.
// For example, cancellation sent from parent to child workflows. Or cancellation between 2 child workflows.
// However, it does not know what to do if your tested workflow code is sending cancellation to external unknown workflows.
// In that case, you will need to setup mock for those cancel calls.
// Some examples of how to setup mock:
//
//   - mock for specific target workflow that matches specific workflow ID and run ID
//     env.OnRequestCancelExternalWorkflow("test-namespace", "test-workflow-id1", "test-runid1").Return(nil).Once()
//   - mock for anything and succeed the cancellation
//     env.OnRequestCancelExternalWorkflow(mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
//   - mock for anything and fail the cancellation
//     env.OnRequestCancelExternalWorkflow(mock.Anything, mock.Anything, mock.Anything).Return(errors.New("unknown external workflow")).Once()
//   - mock function for RequestCancelExternalWorkflow
//     env.OnRequestCancelExternalWorkflow(mock.Anything, mock.Anything, mock.Anything).Return(
//     func(namespace, workflowID, runID) error {
//     // you can do differently based on the parameters
//     return nil
//     })
//
// Mock callbacks here are run on a separate goroutine than the workflow and
// therefore are not concurrency-safe with workflow code.
func (e *TestWorkflowEnvironment) OnRequestCancelExternalWorkflow(namespace, workflowID, runID string) *MockCallWrapper {
	call := e.workflowMock.On(mockMethodForRequestCancelExternalWorkflow, namespace, workflowID, runID)
	return e.wrapWorkflowCall(call)
}

// OnGetVersion setup a mock for workflow.GetVersion() call. By default, if mock is not setup, the GetVersion call from
// workflow code will always return the maxSupported version. Make it not possible to test old version branch. With this
// mock support, it is possible to test code branch for different versions.
//
// Note: mock can be setup for a specific changeID. Or if mock.Anything is used as changeID then all calls to GetVersion
// will be mocked. Mock for a specific changeID has higher priority over mock.Anything.
func (e *TestWorkflowEnvironment) OnGetVersion(changeID string, minSupported, maxSupported Version) *MockCallWrapper {
	call := e.workflowMock.On(getMockMethodForGetVersion(changeID), changeID, minSupported, maxSupported)
	return e.wrapWorkflowCall(call)
}

// OnUpsertSearchAttributes setup a mock for workflow.UpsertSearchAttributes call.
// If mock is not setup, the UpsertSearchAttributes call will only validate input attributes.
// If mock is setup, all UpsertSearchAttributes calls in workflow have to be mocked.
//
// Deprecated: use OnUpsertTypedSearchAttributes instead.
func (e *TestWorkflowEnvironment) OnUpsertSearchAttributes(attributes interface{}) *MockCallWrapper {
	call := e.workflowMock.On(mockMethodForUpsertSearchAttributes, attributes)
	return e.wrapWorkflowCall(call)
}

// OnUpsertTypedSearchAttributes setup a mock for workflow.UpsertTypedSearchAttributes call.
// If mock is not setup, the UpsertTypedSearchAttributes call will only validate input attributes.
// If mock is setup, all UpsertTypedSearchAttributes calls in workflow have to be mocked.
//
// Note: The mock is called with a temporal.SearchAttributes constructed from the inputs to workflow.UpsertTypedSearchAttributes.
func (e *TestWorkflowEnvironment) OnUpsertTypedSearchAttributes(attributes interface{}) *MockCallWrapper {
	call := e.workflowMock.On(mockMethodForUpsertTypedSearchAttributes, attributes)
	return e.wrapWorkflowCall(call)
}

// OnUpsertMemo setup a mock for workflow.UpsertMemo call.
// If mock is not setup, the UpsertMemo call will only validate input attributes.
// If mock is setup, all UpsertMemo calls in workflow have to be mocked.
func (e *TestWorkflowEnvironment) OnUpsertMemo(attributes interface{}) *MockCallWrapper {
	call := e.workflowMock.On(mockMethodForUpsertMemo, attributes)
	return e.wrapWorkflowCall(call)
}

// OnNexusOperation setup a mock call for Nexus operation.
// Parameter service must be Nexus service (*nexus.Service) or service name (string).
// Parameter operation must be Nexus operation (nexus.RegisterableOperation), Nexus operation
// reference (nexus.OperationReference), or operation name (string).
// You must call Return() with appropriate parameters on the returned *MockCallWrapper instance.
// The first parameter of Return() is the result of type nexus.HandlerStartOperationResult[T], ie.,
// it must be *nexus.HandlerStartOperationResultSync[T] or *nexus.HandlerStartOperationResultAsync.
// The second parameter of Return() is an error.
// If your mock returns *nexus.HandlerStartOperationResultAsync, then you need to register the
// completion of the async operation by calling RegisterNexusAsyncOperationCompletion.
// Example: assume the Nexus operation input/output types are as follows:
//
//	type (
//		HelloInput struct {
//			Message string
//		}
//		HelloOutput struct {
//			Message string
//		}
//	)
//
// Then, you can mock workflow.NexusClient.ExecuteOperation as follows:
//
//	t.OnNexusOperation(
//		"my-service",
//		nexus.NewOperationReference[HelloInput, HelloOutput]("hello-operation"),
//		HelloInput{Message: "Temporal"},
//		mock.Anything, // NexusOperationOptions
//	).Return(
//		&nexus.HandlerStartOperationResultAsync{
//			OperationToken: "hello-operation-token",
//		},
//		nil,
//	)
//	t.RegisterNexusAsyncOperationCompletion(
//		"service-name",
//		"hello-operation",
//		"hello-operation-token",
//		HelloOutput{Message: "Hello Temporal"},
//		nil,
//		1*time.Second,
//	)
func (e *TestWorkflowEnvironment) OnNexusOperation(
	service any,
	operation any,
	input any,
	options any,
) *MockCallWrapper {
	var s *nexus.Service
	switch stp := service.(type) {
	case *nexus.Service:
		s = stp
		if e.impl.registry.getNexusService(s.Name) == nil {
			e.impl.RegisterNexusService(s)
		}
	case string:
		s = e.impl.registry.getNexusService(stp)
		if s == nil {
			s = nexus.NewService(stp)
			e.impl.RegisterNexusService(s)
		}
	default:
		panic("service must be *nexus.Service or string")
	}

	var opRef testNexusOperationReference
	switch otp := operation.(type) {
	case testNexusOperationReference:
		// This case covers both nexus.RegisterableOperation and nexus.OperationReference.
		// All nexus.RegisterableOperation embeds nexus.UnimplementedOperation which
		// implements nexus.OperationReference.
		opRef = otp
		if s.Operation(opRef.Name()) == nil {
			if err := s.Register(newTestNexusOperation(opRef)); err != nil {
				panic(fmt.Sprintf("cannot register operation %q: %v", opRef.Name(), err.Error()))
			}
		}
	case string:
		if op := s.Operation(otp); op != nil {
			opRef = op.(testNexusOperationReference)
		} else {
			panic(fmt.Sprintf("operation %q not registered in service %q", otp, s.Name))
		}
	default:
		panic("operation must be nexus.RegisterableOperation, nexus.OperationReference, or string")
	}
	e.impl.registerNexusOperationReference(s.Name, opRef)

	if input != mock.Anything {
		if opRef.InputType() != reflect.TypeOf(input) {
			panic(fmt.Sprintf(
				"operation %q expects input type %s, got %T",
				opRef.Name(),
				opRef.InputType(),
				input,
			))
		}
	}

	if options != mock.Anything {
		if _, ok := options.(NexusOperationOptions); !ok {
			panic(fmt.Sprintf(
				"options must be an instance of NexusOperationOptions or mock.Anything, got %T",
				options,
			))
		}
	}

	call := e.nexusMock.On(s.Name, opRef.Name(), input, options)
	return e.wrapNexusOperationCall(call)
}

// RegisterNexusAsyncOperationCompletion registers a delayed completion of an Nexus async operation.
// The delay is counted from the moment the Nexus async operation starts. See the documentation of
// OnNexusOperation for an example.
func (e *TestWorkflowEnvironment) RegisterNexusAsyncOperationCompletion(
	service string,
	operation string,
	token string,
	result any,
	err error,
	delay time.Duration,
) error {
	return e.impl.RegisterNexusAsyncOperationCompletion(
		service,
		operation,
		token,
		result,
		err,
		delay,
	)
}

func (e *TestWorkflowEnvironment) wrapWorkflowCall(call *mock.Call) *MockCallWrapper {
	callWrapper := &MockCallWrapper{call: call, env: e}
	call.Run(e.impl.getWorkflowMockRunFn(callWrapper))
	return callWrapper
}

func (e *TestWorkflowEnvironment) wrapActivityCall(call *mock.Call) *MockCallWrapper {
	callWrapper := &MockCallWrapper{call: call, env: e}
	call.Run(e.impl.getActivityMockRunFn(callWrapper))
	return callWrapper
}

func (e *TestWorkflowEnvironment) wrapNexusOperationCall(call *mock.Call) *MockCallWrapper {
	callWrapper := &MockCallWrapper{call: call, env: e}
	call.Run(e.impl.getNexusOperationMockRunFn(callWrapper))
	return callWrapper
}

// Once indicates that the mock should only return the value once.
func (c *MockCallWrapper) Once() *MockCallWrapper {
	return c.Times(1)
}

// Twice indicates that the mock should only return the value twice.
func (c *MockCallWrapper) Twice() *MockCallWrapper {
	return c.Times(2)
}

// Times indicates that the mock should only return the indicated number of times.
func (c *MockCallWrapper) Times(i int) *MockCallWrapper {
	c.call.Times(i)
	return c
}

// Never indicates that the mock should not be called.
func (c *MockCallWrapper) Never() *MockCallWrapper {
	c.call.Maybe()
	c.call.Panic(fmt.Sprintf("unexpected call: %s(%s)", c.call.Method, c.call.Arguments.String()))
	return c
}

// Maybe indicates that the mock call is optional. Not calling an optional method
// will not cause an error while asserting expectations.
func (c *MockCallWrapper) Maybe() *MockCallWrapper {
	c.call.Maybe()
	return c
}

// Run sets a handler to be called before returning. It can be used when mocking a method such as unmarshalers that
// takes a pointer to a struct and sets properties in such struct.
func (c *MockCallWrapper) Run(fn func(args mock.Arguments)) *MockCallWrapper {
	c.runFn = fn
	return c
}

// After sets how long to wait on workflow's clock before the mock call returns.
func (c *MockCallWrapper) After(d time.Duration) *MockCallWrapper {
	c.waitDuration = func() time.Duration { return d }
	return c
}

// AfterFn sets a function which will tell how long to wait on workflow's clock before the mock call returns.
func (c *MockCallWrapper) AfterFn(fn func() time.Duration) *MockCallWrapper {
	c.waitDuration = fn
	return c
}

// Return specifies the return arguments for the expectation.
func (c *MockCallWrapper) Return(returnArguments ...interface{}) *MockCallWrapper {
	c.call.Return(returnArguments...)
	return c
}

// Panic specifies if the function call should fail and the panic message
func (c *MockCallWrapper) Panic(msg string) *MockCallWrapper {
	c.call.Panic(msg)
	return c
}

// NotBefore indicates that a call to this mock must not happen before the given calls have happened as expected.
// It calls `NotBefore` on the wrapped mock call.
func (c *MockCallWrapper) NotBefore(calls ...*MockCallWrapper) *MockCallWrapper {
	wrappedCalls := make([]*mock.Call, 0, len(calls))
	for _, call := range calls {
		wrappedCalls = append(wrappedCalls, call.call)
	}
	c.call.NotBefore(wrappedCalls...)
	return c
}

func (uc *TestUpdateCallback) Accept() {
	if uc.OnAccept != nil {
		uc.OnAccept()
	}
}

func (uc *TestUpdateCallback) Reject(err error) {
	if uc.OnReject != nil {
		uc.OnReject(err)
	}
}

func (uc *TestUpdateCallback) Complete(success interface{}, err error) {
	if uc.OnComplete != nil {
		uc.OnComplete(success, err)
	}
}

// ExecuteWorkflow executes a workflow, wait until workflow complete. It will fail the test if workflow is blocked and
// cannot complete within TestTimeout (set by SetTestTimeout()).
func (e *TestWorkflowEnvironment) ExecuteWorkflow(workflowFn interface{}, args ...interface{}) {
	e.impl.workflowMock = &e.workflowMock
	e.impl.activityMock = &e.activityMock
	e.impl.nexusMock = &e.nexusMock
	e.impl.executeWorkflow(workflowFn, args...)
}

// Now returns the current workflow time (a.k.a workflow.Now() time) of this TestWorkflowEnvironment.
func (e *TestWorkflowEnvironment) Now() time.Time {
	return e.impl.Now()
}

// SetWorkerOptions sets the WorkerOptions that will be use by TestActivityEnvironment. TestActivityEnvironment will
// use options of BackgroundActivityContext, MaxConcurrentSessionExecutionSize, and WorkflowInterceptorChainFactories on the WorkerOptions.
// Other options are ignored.
//
// Note: WorkerOptions is defined in internal package, use public type worker.Options instead.
func (e *TestWorkflowEnvironment) SetWorkerOptions(options WorkerOptions) *TestWorkflowEnvironment {
	e.impl.setWorkerOptions(options)
	return e
}

// SetStartWorkflowOptions sets StartWorkflowOptions used to specify workflow execution timeout and task queue.
// Note that StartWorkflowOptions is defined in an internal package, use client.StartWorkflowOptions instead.
func (e *TestWorkflowEnvironment) SetStartWorkflowOptions(options StartWorkflowOptions) *TestWorkflowEnvironment {
	e.impl.setStartWorkflowOptions(options)
	return e
}

// SetDataConverter sets data converter.
func (e *TestWorkflowEnvironment) SetDataConverter(dataConverter converter.DataConverter) *TestWorkflowEnvironment {
	e.impl.setDataConverter(dataConverter)
	return e
}

// SetFailureConverter sets the failure converter.
func (t *TestWorkflowEnvironment) SetFailureConverter(failureConverter converter.FailureConverter) *TestWorkflowEnvironment {
	t.impl.setFailureConverter(failureConverter)
	return t
}

// SetContextPropagators sets context propagators.
func (e *TestWorkflowEnvironment) SetContextPropagators(contextPropagators []ContextPropagator) *TestWorkflowEnvironment {
	e.impl.setContextPropagators(contextPropagators)
	return e
}

// SetHeader sets header.
func (e *TestWorkflowEnvironment) SetHeader(header *commonpb.Header) {
	e.impl.header = header
}

// SetIdentity sets identity.
func (e *TestWorkflowEnvironment) SetIdentity(identity string) *TestWorkflowEnvironment {
	e.impl.setIdentity(identity)
	return e
}

// SetDetachedChildWait, if true, will make ExecuteWorkflow wait on all child
// workflows to complete even if their close policy is set to abandon or request
// cancel, meaning they are "detached". If false, ExecuteWorkflow will block
// until only all attached child workflows have completed. This is useful when
// testing endless detached child workflows, as without it ExecuteWorkflow may
// not return while detached children are still running.
//
// Default is true.
func (e *TestWorkflowEnvironment) SetDetachedChildWait(detachedChildWait bool) *TestWorkflowEnvironment {
	e.impl.setDetachedChildWaitDisabled(!detachedChildWait)
	return e
}

// SetWorkerStopChannel sets the activity worker stop channel to be returned from activity.GetWorkerStopChannel(context)
// You can use this function to set the activity worker stop channel and use close(channel) to test your activity execution
// from workflow execution.
func (e *TestWorkflowEnvironment) SetWorkerStopChannel(c chan struct{}) {
	e.impl.setWorkerStopChannel(c)
}

// SetTestTimeout sets the idle timeout based on wall clock for this tested workflow. Idle is when workflow is blocked
// waiting on events (including timer, activity, child workflow, signal etc). If there is no event happening longer than
// this idle timeout, the test framework would stop the workflow and return timeout error.
// This is based on real wall clock time, not the workflow time (a.k.a workflow.Now() time).
func (e *TestWorkflowEnvironment) SetTestTimeout(idleTimeout time.Duration) *TestWorkflowEnvironment {
	e.impl.testTimeout = idleTimeout
	return e
}

// SetWorkflowRunTimeout sets the run timeout for this tested workflow. This test framework uses mock clock internally
// and when workflow is blocked on timer, it will auto forward the mock clock. Use SetWorkflowRunTimeout() to enforce a
// workflow run timeout to return timeout error when the workflow mock clock is moved head of the timeout.
// This is based on the workflow time (a.k.a workflow.Now() time).
func (e *TestWorkflowEnvironment) SetWorkflowRunTimeout(runTimeout time.Duration) *TestWorkflowEnvironment {
	e.impl.runTimeout = runTimeout
	return e
}

// SetOnActivityStartedListener sets a listener that will be called before activity starts execution.
//
// Note: ActivityInfo is defined in internal package, use public type activity.Info instead.
func (e *TestWorkflowEnvironment) SetOnActivityStartedListener(
	listener func(activityInfo *ActivityInfo, ctx context.Context, args converter.EncodedValues)) *TestWorkflowEnvironment {
	e.impl.onActivityStartedListener = listener
	return e
}

// SetOnActivityCompletedListener sets a listener that will be called after an activity is completed.
//
// Note: ActivityInfo is defined in internal package, use public type activity.Info instead.
func (e *TestWorkflowEnvironment) SetOnActivityCompletedListener(
	listener func(activityInfo *ActivityInfo, result converter.EncodedValue, err error)) *TestWorkflowEnvironment {
	e.impl.onActivityCompletedListener = listener
	return e
}

// SetOnActivityCanceledListener sets a listener that will be called after an activity is canceled.
//
// Note: ActivityInfo is defined in internal package, use public type activity.Info instead.
func (e *TestWorkflowEnvironment) SetOnActivityCanceledListener(
	listener func(activityInfo *ActivityInfo)) *TestWorkflowEnvironment {
	e.impl.onActivityCanceledListener = listener
	return e
}

// SetOnActivityHeartbeatListener sets a listener that will be called when activity heartbeat.
//
// Note: ActivityInfo is defined in internal package, use public type activity.Info instead.
//
// Note: The provided listener may be called concurrently.
//
// Note: Due to internal caching by the activity system, this may not get called
// for every heartbeat recorded. This is only called when the heartbeat would be
// sent to the server (periodic batch and at the end only on failure).
// Interceptors can be used to intercept/check every heartbeat call.
func (e *TestWorkflowEnvironment) SetOnActivityHeartbeatListener(
	listener func(activityInfo *ActivityInfo, details converter.EncodedValues)) *TestWorkflowEnvironment {
	e.impl.onActivityHeartbeatListener = listener
	return e
}

// SetOnChildWorkflowStartedListener sets a listener that will be called before a child workflow starts execution.
//
// Note: WorkflowInfo is defined in internal package, use public type workflow.Info instead.
func (e *TestWorkflowEnvironment) SetOnChildWorkflowStartedListener(
	listener func(workflowInfo *WorkflowInfo, ctx Context, args converter.EncodedValues)) *TestWorkflowEnvironment {
	e.impl.onChildWorkflowStartedListener = listener
	return e
}

// SetOnChildWorkflowCompletedListener sets a listener that will be called after a child workflow is completed.
//
// Note: WorkflowInfo is defined in internal package, use public type workflow.Info instead.
func (e *TestWorkflowEnvironment) SetOnChildWorkflowCompletedListener(
	listener func(workflowInfo *WorkflowInfo, result converter.EncodedValue, err error)) *TestWorkflowEnvironment {
	e.impl.onChildWorkflowCompletedListener = listener
	return e
}

// SetOnChildWorkflowCanceledListener sets a listener that will be called when a child workflow is canceled.
//
// Note: WorkflowInfo is defined in internal package, use public type workflow.Info instead.
func (e *TestWorkflowEnvironment) SetOnChildWorkflowCanceledListener(
	listener func(workflowInfo *WorkflowInfo)) *TestWorkflowEnvironment {
	e.impl.onChildWorkflowCanceledListener = listener
	return e
}

// SetOnTimerScheduledListener sets a listener that will be called before a timer is scheduled.
func (e *TestWorkflowEnvironment) SetOnTimerScheduledListener(
	listener func(timerID string, duration time.Duration)) *TestWorkflowEnvironment {
	e.impl.onTimerScheduledListener = listener
	return e
}

// SetOnTimerFiredListener sets a listener that will be called after a timer is fired.
func (e *TestWorkflowEnvironment) SetOnTimerFiredListener(listener func(timerID string)) *TestWorkflowEnvironment {
	e.impl.onTimerFiredListener = listener
	return e
}

// SetOnTimerCanceledListener sets a listener that will be called after a timer is canceled
func (e *TestWorkflowEnvironment) SetOnTimerCanceledListener(listener func(timerID string)) *TestWorkflowEnvironment {
	e.impl.onTimerCanceledListener = listener
	return e
}

// SetOnLocalActivityStartedListener sets a listener that will be called before local activity starts execution.
//
// Note: ActivityInfo is defined in internal package, use public type activity.Info instead.
func (e *TestWorkflowEnvironment) SetOnLocalActivityStartedListener(
	listener func(activityInfo *ActivityInfo, ctx context.Context, args []interface{})) *TestWorkflowEnvironment {
	e.impl.onLocalActivityStartedListener = listener
	return e
}

// SetOnLocalActivityCompletedListener sets a listener that will be called after local activity is completed.
//
// Note: ActivityInfo is defined in internal package, use public type activity.Info instead.
func (e *TestWorkflowEnvironment) SetOnLocalActivityCompletedListener(
	listener func(activityInfo *ActivityInfo, result converter.EncodedValue, err error)) *TestWorkflowEnvironment {
	e.impl.onLocalActivityCompletedListener = listener
	return e
}

// SetOnLocalActivityCanceledListener sets a listener that will be called after local activity is canceled.
//
// Note: ActivityInfo is defined in internal package, use public type activity.Info instead.
func (e *TestWorkflowEnvironment) SetOnLocalActivityCanceledListener(
	listener func(activityInfo *ActivityInfo)) *TestWorkflowEnvironment {
	e.impl.onLocalActivityCanceledListener = listener
	return e
}

func (e *TestWorkflowEnvironment) SetOnNexusOperationStartedListener(
	listener func(service string, operation string, input converter.EncodedValue),
) *TestWorkflowEnvironment {
	e.impl.onNexusOperationStartedListener = listener
	return e
}

func (e *TestWorkflowEnvironment) SetOnNexusOperationCompletedListener(
	listener func(service string, operation string, result converter.EncodedValue, err error),
) *TestWorkflowEnvironment {
	e.impl.onNexusOperationCompletedListener = listener
	return e
}

func (e *TestWorkflowEnvironment) SetOnNexusOperationCanceledListener(
	listener func(service string, operation string),
) *TestWorkflowEnvironment {
	e.impl.onNexusOperationCanceledListener = listener
	return e
}

// IsWorkflowCompleted check if test is completed or not
func (e *TestWorkflowEnvironment) IsWorkflowCompleted() bool {
	return e.impl.isWorkflowCompleted
}

// GetWorkflowResult extracts the encoded result from test workflow, it returns error if the extraction failed.
func (e *TestWorkflowEnvironment) GetWorkflowResult(valuePtr interface{}) error {
	if !e.impl.isWorkflowCompleted {
		panic("workflow is not completed")
	}
	if e.impl.testError != nil || e.impl.testResult == nil || valuePtr == nil {
		return e.impl.testError
	}
	return e.impl.testResult.Get(valuePtr)
}

// GetWorkflowResultByID extracts the encoded result from workflow by ID, it returns error if the extraction failed.
func (e *TestWorkflowEnvironment) GetWorkflowResultByID(workflowID string, valuePtr interface{}) error {
	if workflowHandle, ok := e.impl.runningWorkflows[workflowID]; ok {
		if !workflowHandle.env.isWorkflowCompleted {
			panic("workflow is not completed")
		}
		if workflowHandle.env.testError != nil || workflowHandle.env.testResult == nil || valuePtr == nil {
			return e.impl.testError
		}
		return e.impl.testResult.Get(valuePtr)
	}
	return serviceerror.NewNotFound(fmt.Sprintf("Workflow %v not exists", workflowID))
}

// GetWorkflowError return the error from test workflow
func (e *TestWorkflowEnvironment) GetWorkflowError() error {
	return e.impl.testError
}

// GetWorkflowErrorByID return the error from test workflow
func (e *TestWorkflowEnvironment) GetWorkflowErrorByID(workflowID string) error {
	if workflowHandle, ok := e.impl.runningWorkflows[workflowID]; ok {
		return workflowHandle.env.testError
	}
	return serviceerror.NewNotFound(fmt.Sprintf("Workflow %v not exists", workflowID))
}

// CompleteActivity complete an activity that had returned activity.ErrResultPending error
func (e *TestWorkflowEnvironment) CompleteActivity(taskToken []byte, result interface{}, err error) error {
	return e.impl.CompleteActivity(taskToken, result, err)
}

// CancelWorkflow requests cancellation (through workflow Context) to the currently running test workflow.
func (e *TestWorkflowEnvironment) CancelWorkflow() {
	e.impl.cancelWorkflow(func(result *commonpb.Payloads, err error) {})
}

// CancelWorkflowByID requests cancellation (through workflow Context) to the specified workflow.
func (e *TestWorkflowEnvironment) CancelWorkflowByID(workflowID string, runID string) {
	e.impl.cancelWorkflowByID(workflowID, runID, func(result *commonpb.Payloads, err error) {})
}

// SignalWorkflow sends signal to the currently running test workflow.
func (e *TestWorkflowEnvironment) SignalWorkflow(name string, input interface{}) {
	e.impl.signalWorkflow(name, input, true)
}

// SignalWorkflowSkippingWorkflowTask sends signal to the currently running test workflow without invoking workflow code.
// Used to test processing of multiple buffered signals before completing workflow.
// It must be followed by SignalWorkflow, CancelWorkflow or CompleteActivity to force a workflow task.
func (e *TestWorkflowEnvironment) SignalWorkflowSkippingWorkflowTask(name string, input interface{}) {
	e.impl.signalWorkflow(name, input, false)
}

// SignalWorkflowByID signals a workflow by its ID.
func (e *TestWorkflowEnvironment) SignalWorkflowByID(workflowID, signalName string, input interface{}) error {
	return e.impl.signalWorkflowByID(workflowID, signalName, input)
}

// QueryWorkflow queries to the currently running test workflow and returns result synchronously.
func (e *TestWorkflowEnvironment) QueryWorkflow(queryType string, args ...interface{}) (converter.EncodedValue, error) {
	return e.impl.queryWorkflow(queryType, args...)
}

// UpdateWorkflow sends an update to the currently running workflow. The updateName is the name of the update handler
// to be invoked. The updateID is a unique identifier for the update. If updateID is an empty string a UUID will be generated.
// The update callbacks are used to handle the update. The args are the arguments to be passed to the update handler.
func (e *TestWorkflowEnvironment) UpdateWorkflow(updateName, updateID string, uc UpdateCallbacks, args ...interface{}) {
	e.impl.updateWorkflow(updateName, updateID, uc, args...)
}

// UpdateWorkflowByID sends an update to a running workflow by its ID.
func (e *TestWorkflowEnvironment) UpdateWorkflowByID(workflowID, updateName, updateID string, uc UpdateCallbacks, args ...interface{}) error {
	return e.impl.updateWorkflowByID(workflowID, updateName, updateID, uc, args...)
}

// UpdateWorkflowNoRejection is a convenience function that handles a common test scenario of only validating
// that an update isn't rejected.
func (e *TestWorkflowEnvironment) UpdateWorkflowNoRejection(updateName string, updateID string, t mock.TestingT, args ...interface{}) {
	uc := &TestUpdateCallback{
		OnReject: func(err error) {
			require.Fail(t, "update should not be rejected")
		},
		OnAccept:   func() {},
		OnComplete: func(interface{}, error) {},
	}

	e.UpdateWorkflow(updateName, updateID, uc, args...)
}

// QueryWorkflowByID queries a child workflow by its ID and returns the result synchronously
func (e *TestWorkflowEnvironment) QueryWorkflowByID(workflowID, queryType string, args ...interface{}) (converter.EncodedValue, error) {
	return e.impl.queryWorkflowByID(workflowID, queryType, args...)
}

// RegisterDelayedCallback creates a new timer with specified delayDuration using workflow clock (not wall clock). When
// the timer fires, the callback will be called. By default, this test suite uses mock clock which automatically move
// forward to fire next timer when workflow is blocked. Use this API to make some event (like activity completion,
// signal or workflow cancellation) at desired time.
//
// Use 0 delayDuration to send a signal to simulate SignalWithStart. Note that a 0 duration delay will *not* work with
// Queries, as the workflow will not have had a chance to register any query handlers.
func (e *TestWorkflowEnvironment) RegisterDelayedCallback(callback func(), delayDuration time.Duration) {
	e.impl.registerDelayedCallback(callback, delayDuration)
}

// SetActivityTaskQueue set the affinity between activity and taskqueue. By default, activity can be invoked by any taskqueue
// in this test environment. Use this SetActivityTaskQueue() to set affinity between activity and a taskqueue. Once
// activity is set to a particular taskqueue, that activity will only be available to that taskqueue.
func (e *TestWorkflowEnvironment) SetActivityTaskQueue(taskqueue string, activityFn ...interface{}) {
	e.impl.setActivityTaskQueue(taskqueue, activityFn...)
}

// SetLastCompletionResult sets the result to be returned from workflow.GetLastCompletionResult().
func (e *TestWorkflowEnvironment) SetLastCompletionResult(result interface{}) {
	e.impl.setLastCompletionResult(result)
}

// SetLastError sets the result to be returned from workflow.GetLastError().
func (e *TestWorkflowEnvironment) SetLastError(err error) {
	e.impl.setLastError(err)
}

// SetMemoOnStart sets the memo when start workflow.
func (e *TestWorkflowEnvironment) SetMemoOnStart(memo map[string]interface{}) error {
	memoStruct, err := getWorkflowMemo(memo, e.impl.GetDataConverter())
	if err != nil {
		return err
	}
	e.impl.workflowInfo.Memo = memoStruct
	return nil
}

// SetSearchAttributesOnStart sets the search attributes when start workflow.
//
// Deprecated: Use SetTypedSearchAttributes instead.
func (e *TestWorkflowEnvironment) SetSearchAttributesOnStart(searchAttributes map[string]interface{}) error {
	attr, err := serializeUntypedSearchAttributes(searchAttributes)
	if err != nil {
		return err
	}
	e.impl.workflowInfo.SearchAttributes = attr
	return nil
}

// SetTypedSearchAttributesOnStart sets the search attributes when start workflow.
func (e *TestWorkflowEnvironment) SetTypedSearchAttributesOnStart(searchAttributes SearchAttributes) error {
	attr, err := serializeSearchAttributes(nil, searchAttributes)
	if err != nil {
		return err
	}
	e.impl.workflowInfo.SearchAttributes = attr
	return nil
}

// AssertExpectations asserts that everything specified with OnWorkflow, OnActivity, OnNexusOperation
// was in fact called as expected. Calls may have occurred in any order.
func (e *TestWorkflowEnvironment) AssertExpectations(t mock.TestingT) bool {
	return e.workflowMock.AssertExpectations(t) &&
		e.activityMock.AssertExpectations(t) &&
		e.nexusMock.AssertExpectations(t)
}

// AssertCalled asserts that the method (workflow or activity) was called with the supplied arguments.
// Useful to assert that an Activity was called from within a workflow with the expected arguments.
// Since the first argument is a context, consider using mock.Anything for that argument.
//
//	env.OnActivity(namedActivity, mock.Anything, mock.Anything).Return("mock_result", nil)
//	env.ExecuteWorkflow(workflowThatCallsActivityWithItsArgument, "Hello")
//	env.AssertCalled(t, "namedActivity", mock.Anything, "Hello")
//
// It can produce a false result when an argument is a pointer type and the underlying value changed after calling the mocked method.
func (e *TestWorkflowEnvironment) AssertCalled(t mock.TestingT, methodName string, arguments ...interface{}) bool {
	dummyT := &testing.T{}
	return e.AssertWorkflowCalled(dummyT, methodName, arguments...) ||
		e.AssertActivityCalled(dummyT, methodName, arguments...) ||
		e.AssertWorkflowCalled(t, methodName, arguments...) ||
		e.AssertActivityCalled(t, methodName, arguments...)
}

// AssertWorkflowCalled asserts that the workflow method was called with the supplied arguments.
// Special method for workflows, doesn't assert activity calls.
func (e *TestWorkflowEnvironment) AssertWorkflowCalled(t mock.TestingT, methodName string, arguments ...interface{}) bool {
	return e.workflowMock.AssertCalled(t, methodName, arguments...)
}

// AssertActivityCalled asserts that the activity method was called with the supplied arguments.
// Special method for activities, doesn't assert workflow calls.
func (e *TestWorkflowEnvironment) AssertActivityCalled(t mock.TestingT, methodName string, arguments ...interface{}) bool {
	return e.activityMock.AssertCalled(t, methodName, arguments...)
}

// AssertNotCalled asserts that the method (workflow or activity) was not called with the given arguments.
// See AssertCalled for more info.
func (e *TestWorkflowEnvironment) AssertNotCalled(t mock.TestingT, methodName string, arguments ...interface{}) bool {
	dummyT := &testing.T{}
	// Calling the individual functions instead of negating AssertCalled so the error message is more clear.
	return e.AssertWorkflowNotCalled(dummyT, methodName, arguments...) &&
		e.AssertActivityNotCalled(dummyT, methodName, arguments...) &&
		e.AssertWorkflowNotCalled(t, methodName, arguments...) &&
		e.AssertActivityNotCalled(t, methodName, arguments...)
}

// AssertWorkflowNotCalled asserts that the workflow method was not called with the given arguments.
// Special method for workflows, doesn't assert activity calls.
// See AssertCalled for more info.
func (e *TestWorkflowEnvironment) AssertWorkflowNotCalled(t mock.TestingT, methodName string, arguments ...interface{}) bool {
	return e.workflowMock.AssertNotCalled(t, methodName, arguments...)
}

// AssertActivityNotCalled asserts that the activity method was not called with the given arguments.
// Special method for activities, doesn't assert workflow calls.
// See AssertCalled for more info.
func (e *TestWorkflowEnvironment) AssertActivityNotCalled(t mock.TestingT, methodName string, arguments ...interface{}) bool {
	return e.activityMock.AssertNotCalled(t, methodName, arguments...)
}

// AssertNumberOfCalls asserts that a method (workflow or activity) was called expectedCalls times.
func (e *TestWorkflowEnvironment) AssertNumberOfCalls(t mock.TestingT, methodName string, expectedCalls int) bool {
	dummyT := &testing.T{}
	return e.workflowMock.AssertNumberOfCalls(dummyT, methodName, expectedCalls) ||
		e.activityMock.AssertNumberOfCalls(dummyT, methodName, expectedCalls) ||
		e.workflowMock.AssertNumberOfCalls(t, methodName, expectedCalls) ||
		e.activityMock.AssertNumberOfCalls(t, methodName, expectedCalls)
}

// AssertWorkflowNumberOfCalls asserts that a workflow method was called expectedCalls times.
// Special method for workflows, doesn't assert activity calls.
func (e *TestWorkflowEnvironment) AssertWorkflowNumberOfCalls(t mock.TestingT, methodName string, expectedCalls int) bool {
	return e.workflowMock.AssertNumberOfCalls(t, methodName, expectedCalls)
}

// AssertActivityNumberOfCalls asserts that a activity method was called expectedCalls times.
// Special method for activities, doesn't assert workflow calls.
func (e *TestWorkflowEnvironment) AssertActivityNumberOfCalls(t mock.TestingT, methodName string, expectedCalls int) bool {
	return e.activityMock.AssertNumberOfCalls(t, methodName, expectedCalls)
}

// AssertNexusOperationCalled asserts that the Nexus operation was called with the supplied arguments.
// Special method for Nexus operations only.
func (e *TestWorkflowEnvironment) AssertNexusOperationCalled(t mock.TestingT, service string, operation string, input any, options any) bool {
	return e.nexusMock.AssertCalled(t, service, operation, input, options)
}

// AssertNexusOperationNotCalled asserts that the Nexus operation was called with the supplied arguments.
// Special method for Nexus operations only.
// See AssertNexusOperationCalled for more info.
func (e *TestWorkflowEnvironment) AssertNexusOperationNotCalled(t mock.TestingT, service string, operation string, input any, options any) bool {
	return e.nexusMock.AssertNotCalled(t, service, operation, input, options)
}

// AssertNexusOperationNumberOfCalls asserts that a Nexus operation was called expectedCalls times.
// Special method for Nexus operation only.
func (e *TestWorkflowEnvironment) AssertNexusOperationNumberOfCalls(t mock.TestingT, service string, expectedCalls int) bool {
	return e.nexusMock.AssertNumberOfCalls(t, service, expectedCalls)
}
