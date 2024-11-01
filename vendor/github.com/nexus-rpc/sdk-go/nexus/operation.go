package nexus

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// NoValue is a marker type for an operations that do not accept any input or return a value (nil).
//
//	nexus.NewSyncOperation("my-empty-operation", func(context.Context, nexus.NoValue, options, nexus.StartOperationOptions) (nexus.NoValue, error) {
//		return nil, nil
//	)}
type NoValue *struct{}

// OperationReference provides a typed interface for invoking operations. Every [Operation] is also an
// [OperationReference]. Callers may create references using [NewOperationReference] when the implementation is not
// available.
type OperationReference[I, O any] interface {
	Name() string
	// A type inference helper for implementations of this interface.
	inferType(I, O)
}

type operationReference[I, O any] string

// NewOperationReference creates an [OperationReference] with the provided type parameters and name.
// It provides typed interface for invoking operations when the implementation is not available to the caller.
func NewOperationReference[I, O any](name string) OperationReference[I, O] {
	return operationReference[I, O](name)
}

func (r operationReference[I, O]) Name() string {
	return string(r)
}

func (operationReference[I, O]) inferType(I, O) {} //nolint:unused

// A RegisterableOperation is accepted in [OperationRegistry.Register].
// Embed [UnimplementedOperation] to implement it.
type RegisterableOperation interface {
	// Name of the operation. Used for invocation and registration.
	Name() string
	mustEmbedUnimplementedOperation()
}

// Operation is a handler for a single operation.
//
// Operation implementations must embed the [UnimplementedOperation].
//
// All Operation methods can return a [HandlerError] to fail requests with a custom [HandlerErrorType] and structured [Failure].
// Arbitrary errors from handler methods are turned into [HandlerErrorTypeInternal],their details are logged and hidden
// from the caller.
type Operation[I, O any] interface {
	RegisterableOperation
	OperationReference[I, O]

	// Start handles requests for starting an operation. Return [HandlerStartOperationResultSync] to respond
	// successfully - inline, or [HandlerStartOperationResultAsync] to indicate that an asynchronous operation was
	// started. Return an [UnsuccessfulOperationError] to indicate that an operation completed as failed or
	// canceled.
	Start(context.Context, I, StartOperationOptions) (HandlerStartOperationResult[O], error)
	// GetResult handles requests to get the result of an asynchronous operation. Return non error result to respond
	// successfully - inline, or error with [ErrOperationStillRunning] to indicate that an asynchronous operation is
	// still running. Return an [UnsuccessfulOperationError] to indicate that an operation completed as failed or
	// canceled.
	//
	// When [GetOperationResultOptions.Wait] is greater than zero, this request should be treated as a long poll.
	// Long poll requests have a server side timeout, configurable via [HandlerOptions.GetResultTimeout], and exposed
	// via context deadline. The context deadline is decoupled from the application level Wait duration.
	//
	// It is the implementor's responsiblity to respect the client's wait duration and return in a timely fashion.
	// Consider using a derived context that enforces the wait timeout when implementing this method and return
	// [ErrOperationStillRunning] when that context expires as shown in the [Handler] example.
	GetResult(context.Context, string, GetOperationResultOptions) (O, error)
	// GetInfo handles requests to get information about an asynchronous operation.
	GetInfo(context.Context, string, GetOperationInfoOptions) (*OperationInfo, error)
	// Cancel handles requests to cancel an asynchronous operation.
	// Cancelation in Nexus is:
	//  1. asynchronous - returning from this method only ensures that cancelation is delivered, it may later be
	//  ignored by the underlying operation implemention.
	//  2. idempotent - implementors should ignore duplicate cancelations for the same operation.
	Cancel(context.Context, string, CancelOperationOptions) error
}

type syncOperation[I, O any] struct {
	UnimplementedOperation[I, O]

	Handler func(context.Context, I, StartOperationOptions) (O, error)
	name    string
}

// NewSyncOperation is a helper for creating a synchronous-only [Operation] from a given name and handler function.
func NewSyncOperation[I, O any](name string, handler func(context.Context, I, StartOperationOptions) (O, error)) Operation[I, O] {
	return &syncOperation[I, O]{
		name:    name,
		Handler: handler,
	}
}

// Name implements Operation.
func (h *syncOperation[I, O]) Name() string {
	return h.name
}

// Start implements Operation.
func (h *syncOperation[I, O]) Start(ctx context.Context, input I, options StartOperationOptions) (HandlerStartOperationResult[O], error) {
	o, err := h.Handler(ctx, input, options)
	if err != nil {
		return nil, err
	}
	return &HandlerStartOperationResultSync[O]{o}, err
}

// A Service is a container for a group of operations.
type Service struct {
	Name string

	operations map[string]RegisterableOperation
}

// NewService constructs a [Service].
func NewService(name string) *Service {
	return &Service{
		Name:       name,
		operations: make(map[string]RegisterableOperation),
	}
}

// Register one or more operations.
// Returns an error if duplicate operations were registered with the same name or when trying to register an operation
// with no name.
//
// Can be called multiple times and is not thread safe.
func (s *Service) Register(operations ...RegisterableOperation) error {
	var dups []string
	for _, op := range operations {
		if op.Name() == "" {
			return fmt.Errorf("tried to register an operation with no name")
		}
		if _, found := s.operations[op.Name()]; found {
			dups = append(dups, op.Name())
		} else {
			s.operations[op.Name()] = op
		}
	}
	if len(dups) > 0 {
		return fmt.Errorf("duplicate operations: %s", strings.Join(dups, ", "))
	}
	return nil
}

// A ServiceRegistry registers services and constructs a [Handler] that dispatches operations requests to those services.
type ServiceRegistry struct {
	services map[string]*Service
}

func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{services: make(map[string]*Service)}
}

// Register one or more service.
// Returns an error if duplicate operations were registered with the same name or when trying to register a service with
// no name.
//
// Can be called multiple times and is not thread safe.
func (r *ServiceRegistry) Register(services ...*Service) error {
	var dups []string
	for _, service := range services {
		if service.Name == "" {
			return fmt.Errorf("tried to register a service with no name")
		}
		if _, found := r.services[service.Name]; found {
			dups = append(dups, service.Name)
		} else {
			r.services[service.Name] = service
		}
	}
	if len(dups) > 0 {
		return fmt.Errorf("duplicate services: %s", strings.Join(dups, ", "))
	}
	return nil
}

// NewHandler creates a [Handler] that dispatches requests to registered operations based on their name.
func (r *ServiceRegistry) NewHandler() (Handler, error) {
	if len(r.services) == 0 {
		return nil, errors.New("must register at least one service")
	}
	for _, service := range r.services {
		if len(service.operations) == 0 {
			return nil, fmt.Errorf("service %q has no operations registered", service.Name)
		}
	}

	return &registryHandler{services: r.services}, nil
}

type registryHandler struct {
	UnimplementedHandler

	services map[string]*Service
}

// CancelOperation implements Handler.
func (r *registryHandler) CancelOperation(ctx context.Context, service, operation string, operationID string, options CancelOperationOptions) error {
	s, ok := r.services[service]
	if !ok {
		return HandlerErrorf(HandlerErrorTypeNotFound, "service %q not found", service)
	}
	h, ok := s.operations[operation]
	if !ok {
		return HandlerErrorf(HandlerErrorTypeNotFound, "operation %q not found", operation)
	}

	// NOTE: We could avoid reflection here if we put the Cancel method on RegisterableOperation but it doesn't seem
	// worth it since we need reflection for the generic methods.
	m, _ := reflect.TypeOf(h).MethodByName("Cancel")
	values := m.Func.Call([]reflect.Value{reflect.ValueOf(h), reflect.ValueOf(ctx), reflect.ValueOf(operationID), reflect.ValueOf(options)})
	if values[0].IsNil() {
		return nil
	}
	return values[0].Interface().(error)
}

// GetOperationInfo implements Handler.
func (r *registryHandler) GetOperationInfo(ctx context.Context, service, operation string, operationID string, options GetOperationInfoOptions) (*OperationInfo, error) {
	s, ok := r.services[service]
	if !ok {
		return nil, HandlerErrorf(HandlerErrorTypeNotFound, "service %q not found", service)
	}
	h, ok := s.operations[operation]
	if !ok {
		return nil, HandlerErrorf(HandlerErrorTypeNotFound, "operation %q not found", operation)
	}

	// NOTE: We could avoid reflection here if we put the Cancel method on RegisterableOperation but it doesn't seem
	// worth it since we need reflection for the generic methods.
	m, _ := reflect.TypeOf(h).MethodByName("GetInfo")
	values := m.Func.Call([]reflect.Value{reflect.ValueOf(h), reflect.ValueOf(ctx), reflect.ValueOf(operationID), reflect.ValueOf(options)})
	if !values[1].IsNil() {
		return nil, values[1].Interface().(error)
	}
	ret := values[0].Interface()
	return ret.(*OperationInfo), nil
}

// GetOperationResult implements Handler.
func (r *registryHandler) GetOperationResult(ctx context.Context, service, operation string, operationID string, options GetOperationResultOptions) (any, error) {
	s, ok := r.services[service]
	if !ok {
		return nil, HandlerErrorf(HandlerErrorTypeNotFound, "service %q not found", service)
	}
	h, ok := s.operations[operation]
	if !ok {
		return nil, HandlerErrorf(HandlerErrorTypeNotFound, "operation %q not found", operation)
	}

	m, _ := reflect.TypeOf(h).MethodByName("GetResult")
	values := m.Func.Call([]reflect.Value{reflect.ValueOf(h), reflect.ValueOf(ctx), reflect.ValueOf(operationID), reflect.ValueOf(options)})
	if !values[1].IsNil() {
		return nil, values[1].Interface().(error)
	}
	ret := values[0].Interface()
	return ret, nil
}

// StartOperation implements Handler.
func (r *registryHandler) StartOperation(ctx context.Context, service, operation string, input *LazyValue, options StartOperationOptions) (HandlerStartOperationResult[any], error) {
	s, ok := r.services[service]
	if !ok {
		return nil, HandlerErrorf(HandlerErrorTypeNotFound, "service %q not found", service)
	}
	h, ok := s.operations[operation]
	if !ok {
		return nil, HandlerErrorf(HandlerErrorTypeNotFound, "operation %q not found", operation)
	}

	m, _ := reflect.TypeOf(h).MethodByName("Start")
	inputType := m.Type.In(2)
	iptr := reflect.New(inputType).Interface()
	if err := input.Consume(iptr); err != nil {
		// TODO: log the error? Do we need to accept a logger for this single line?
		return nil, HandlerErrorf(HandlerErrorTypeBadRequest, "invalid input")
	}
	i := reflect.ValueOf(iptr).Elem()

	values := m.Func.Call([]reflect.Value{reflect.ValueOf(h), reflect.ValueOf(ctx), i, reflect.ValueOf(options)})
	if !values[1].IsNil() {
		return nil, values[1].Interface().(error)
	}
	ret := values[0].Interface()
	return ret.(HandlerStartOperationResult[any]), nil
}

var _ Handler = &registryHandler{}

// ExecuteOperation is the type safe version of [Client.ExecuteOperation].
// It accepts input of type I and returns output of type O, removing the need to consume the [LazyValue] returned by the
// client method.
//
//	ref := NewOperationReference[MyInput, MyOutput]("my-operation")
//	out, err := ExecuteOperation(ctx, client, ref, MyInput{}, options) // returns MyOutput, error
func ExecuteOperation[I, O any](ctx context.Context, client *Client, operation OperationReference[I, O], input I, request ExecuteOperationOptions) (O, error) {
	var o O
	value, err := client.ExecuteOperation(ctx, operation.Name(), input, request)
	if err != nil {
		return o, err
	}
	return o, value.Consume(&o)
}

// StartOperation is the type safe version of [Client.StartOperation].
// It accepts input of type I and returns a [ClientStartOperationResult] of type O, removing the need to consume the
// [LazyValue] returned by the client method.
func StartOperation[I, O any](ctx context.Context, client *Client, operation OperationReference[I, O], input I, request StartOperationOptions) (*ClientStartOperationResult[O], error) {
	result, err := client.StartOperation(ctx, operation.Name(), input, request)
	if err != nil {
		return nil, err
	}
	if result.Successful != nil {
		var o O
		if err := result.Successful.Consume(&o); err != nil {
			return nil, err
		}
		return &ClientStartOperationResult[O]{Successful: o}, nil
	}
	handle := OperationHandle[O]{client: client, Operation: operation.Name(), ID: result.Pending.ID}
	return &ClientStartOperationResult[O]{
		Pending: &handle,
		Links:   result.Links,
	}, nil
}

// NewHandle is the type safe version of [Client.NewHandle].
// The [Handle.GetResult] method will return an output of type O.
func NewHandle[I, O any](client *Client, operation OperationReference[I, O], operationID string) (*OperationHandle[O], error) {
	if operationID == "" {
		return nil, errEmptyOperationID
	}
	return &OperationHandle[O]{client: client, Operation: operation.Name(), ID: operationID}, nil
}
