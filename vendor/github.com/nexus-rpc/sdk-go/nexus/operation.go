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
	// InputType the generic input type I for this operation.
	InputType() reflect.Type
	// OutputType the generic out type O for this operation.
	OutputType() reflect.Type
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

func (operationReference[I, O]) InputType() reflect.Type {
	var zero [0]I
	return reflect.TypeOf(zero).Elem()
}

func (operationReference[I, O]) OutputType() reflect.Type {
	var zero [0]O
	return reflect.TypeOf(zero).Elem()
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
// See [OperationHandler] for more information.
type Operation[I, O any] interface {
	RegisterableOperation
	OperationReference[I, O]
	OperationHandler[I, O]
}

// OperationHandler is the interface for the core operation methods. OperationHandler implementations must embed
// [UnimplementedOperation].
//
// All Operation methods can return a [HandlerError] to fail requests with a custom [HandlerErrorType] and structured [Failure].
// Arbitrary errors from handler methods are turned into [HandlerErrorTypeInternal], when using the Nexus SDK's
// HTTP handler, their details are logged and hidden from the caller. Other handler implementations may expose internal
// error information to callers.
type OperationHandler[I, O any] interface {
	// Start handles requests for starting an operation. Return [HandlerStartOperationResultSync] to respond
	// successfully - inline, or [HandlerStartOperationResultAsync] to indicate that an asynchronous operation was
	// started. Return an [OperationError] to indicate that an operation completed as failed or
	// canceled.
	Start(ctx context.Context, input I, options StartOperationOptions) (HandlerStartOperationResult[O], error)
	// GetResult handles requests to get the result of an asynchronous operation. Return non error result to respond
	// successfully - inline, or error with [ErrOperationStillRunning] to indicate that an asynchronous operation is
	// still running. Return an [OperationError] to indicate that an operation completed as failed or
	// canceled.
	//
	// When [GetOperationResultOptions.Wait] is greater than zero, this request should be treated as a long poll.
	// Long poll requests have a server side timeout, configurable via [HandlerOptions.GetResultTimeout], and exposed
	// via context deadline. The context deadline is decoupled from the application level Wait duration.
	//
	// It is the implementor's responsiblity to respect the client's wait duration and return in a timely fashion.
	// Consider using a derived context that enforces the wait timeout when implementing this method and return
	// [ErrOperationStillRunning] when that context expires as shown in the [Handler] example.
	//
	// NOTE: Experimental
	GetResult(ctx context.Context, token string, options GetOperationResultOptions) (O, error)
	// GetInfo handles requests to get information about an asynchronous operation.
	//
	// NOTE: Experimental
	GetInfo(ctx context.Context, token string, options GetOperationInfoOptions) (*OperationInfo, error)
	// Cancel handles requests to cancel an asynchronous operation.
	// Cancelation in Nexus is:
	//  1. asynchronous - returning from this method only ensures that cancelation is delivered, it may later be
	//  ignored by the underlying operation implemention.
	//  2. idempotent - implementors should ignore duplicate cancelations for the same operation.
	Cancel(ctx context.Context, token string, options CancelOperationOptions) error

	mustEmbedUnimplementedOperation()
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
	return &HandlerStartOperationResultSync[O]{Value: o}, err
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

// MustRegister registers one or more operations.
// Panics if duplicate operations were registered with the same name or when trying to register an operation with no
// name.
//
// Can be called multiple times and is not thread safe.
func (s *Service) MustRegister(operations ...RegisterableOperation) {
	if err := s.Register(operations...); err != nil {
		panic(err)
	}
}

// Operation returns an operation by name or nil if not found.
func (s *Service) Operation(name string) RegisterableOperation {
	return s.operations[name]
}

// MiddlewareFunc is a function which receives an OperationHandler and returns another OperationHandler.
// If the middleware wants to stop the chain before any handler is called, it can return an error.
//
// To get [HandlerInfo] for the current handler, call [ExtractHandlerInfo] with the given context.
//
// NOTE: Experimental
type MiddlewareFunc func(ctx context.Context, next OperationHandler[any, any]) (OperationHandler[any, any], error)

// A ServiceRegistry registers services and constructs a [Handler] that dispatches operations requests to those services.
type ServiceRegistry struct {
	services   map[string]*Service
	middleware []MiddlewareFunc
}

// NewServiceRegistry constructs an empty [ServiceRegistry].
func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		services:   make(map[string]*Service),
		middleware: make([]MiddlewareFunc, 0),
	}
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

// Use registers one or more middleware to be applied to all operation method invocations across all registered
// services. Middleware is applied in registration order. If called multiple times, newly registered middleware will be
// applied after any previously registered ones.
//
// NOTE: Experimental
func (s *ServiceRegistry) Use(middleware ...MiddlewareFunc) {
	s.middleware = append(s.middleware, middleware...)
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

	return &registryHandler{services: r.services, middlewares: r.middleware}, nil
}

type registryHandler struct {
	UnimplementedHandler

	services    map[string]*Service
	middlewares []MiddlewareFunc
}

func (r *registryHandler) operationHandler(ctx context.Context) (OperationHandler[any, any], error) {
	options := ExtractHandlerInfo(ctx)
	s, ok := r.services[options.Service]
	if !ok {
		return nil, HandlerErrorf(HandlerErrorTypeNotFound, "service %q not found", options.Service)
	}
	h, ok := s.operations[options.Operation]
	if !ok {
		return nil, HandlerErrorf(HandlerErrorTypeNotFound, "operation %q not found", options.Operation)
	}

	var handler OperationHandler[any, any]
	handler = &rootOperationHandler{h: h}
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		var err error
		handler, err = r.middlewares[i](ctx, handler)
		if err != nil {
			return nil, err
		}
	}
	return handler, nil
}

// CancelOperation implements Handler.
func (r *registryHandler) CancelOperation(ctx context.Context, service, operation, token string, options CancelOperationOptions) error {
	h, err := r.operationHandler(ctx)
	if err != nil {
		return err
	}
	return h.Cancel(ctx, token, options)
}

// operationHandlerInfo implements Handler.
func (r *registryHandler) GetOperationInfo(ctx context.Context, service, operation, token string, options GetOperationInfoOptions) (*OperationInfo, error) {
	h, err := r.operationHandler(ctx)
	if err != nil {
		return nil, err
	}
	return h.GetInfo(ctx, token, options)
}

// operationHandlerResult implements Handler.
func (r *registryHandler) GetOperationResult(ctx context.Context, service, operation, token string, options GetOperationResultOptions) (any, error) {
	h, err := r.operationHandler(ctx)
	if err != nil {
		return nil, err
	}
	return h.GetResult(ctx, token, options)
}

// StartOperation implements Handler.
func (r *registryHandler) StartOperation(ctx context.Context, service, operation string, input *LazyValue, options StartOperationOptions) (HandlerStartOperationResult[any], error) {
	s, ok := r.services[service]
	if !ok {
		return nil, HandlerErrorf(HandlerErrorTypeNotFound, "service %q not found", service)
	}
	ro, ok := s.operations[operation]
	if !ok {
		return nil, HandlerErrorf(HandlerErrorTypeNotFound, "operation %q not found", operation)
	}

	h, err := r.operationHandler(ctx)
	if err != nil {
		return nil, err
	}
	m, _ := reflect.TypeOf(ro).MethodByName("Start")
	inputType := m.Type.In(2)
	iptr := reflect.New(inputType).Interface()
	if err := input.Consume(iptr); err != nil {
		// TODO: log the error? Do we need to accept a logger for this single line?
		return nil, HandlerErrorf(HandlerErrorTypeBadRequest, "invalid input")
	}
	return h.Start(ctx, reflect.ValueOf(iptr).Elem().Interface(), options)
}

type rootOperationHandler struct {
	UnimplementedOperation[any, any]
	h RegisterableOperation
}

func (r *rootOperationHandler) Cancel(ctx context.Context, token string, options CancelOperationOptions) error {
	// NOTE: We could avoid reflection here if we put the Cancel method on RegisterableOperation but it doesn't seem
	// worth it since we need reflection for the generic methods.
	m, _ := reflect.TypeOf(r.h).MethodByName("Cancel")
	values := m.Func.Call([]reflect.Value{reflect.ValueOf(r.h), reflect.ValueOf(ctx), reflect.ValueOf(token), reflect.ValueOf(options)})
	if values[0].IsNil() {
		return nil
	}
	return values[0].Interface().(error)
}

func (r *rootOperationHandler) GetInfo(ctx context.Context, token string, options GetOperationInfoOptions) (*OperationInfo, error) {
	// NOTE: We could avoid reflection here if we put the GetInfo method on RegisterableOperation but it doesn't
	// seem worth it since we need reflection for the generic methods.
	m, _ := reflect.TypeOf(r.h).MethodByName("GetInfo")
	values := m.Func.Call([]reflect.Value{reflect.ValueOf(r.h), reflect.ValueOf(ctx), reflect.ValueOf(token), reflect.ValueOf(options)})
	if !values[1].IsNil() {
		return nil, values[1].Interface().(error)
	}
	ret := values[0].Interface()
	return ret.(*OperationInfo), nil
}

func (r *rootOperationHandler) GetResult(ctx context.Context, token string, options GetOperationResultOptions) (any, error) {
	m, _ := reflect.TypeOf(r.h).MethodByName("GetResult")
	values := m.Func.Call([]reflect.Value{reflect.ValueOf(r.h), reflect.ValueOf(ctx), reflect.ValueOf(token), reflect.ValueOf(options)})
	if !values[1].IsNil() {
		return nil, values[1].Interface().(error)
	}
	ret := values[0].Interface()
	return ret, nil
}

func (r *rootOperationHandler) Start(ctx context.Context, input any, options StartOperationOptions) (HandlerStartOperationResult[any], error) {
	m, _ := reflect.TypeOf(r.h).MethodByName("Start")
	values := m.Func.Call([]reflect.Value{reflect.ValueOf(r.h), reflect.ValueOf(ctx), reflect.ValueOf(input), reflect.ValueOf(options)})
	if !values[1].IsNil() {
		return nil, values[1].Interface().(error)
	}
	ret := values[0].Interface()
	return ret.(HandlerStartOperationResult[any]), nil
}

// ExecuteOperation is the type safe version of [HTTPClient.ExecuteOperation].
// It accepts input of type I and returns output of type O, removing the need to consume the [LazyValue] returned by the
// client method.
//
//	ref := NewOperationReference[MyInput, MyOutput]("my-operation")
//	out, err := ExecuteOperation(ctx, client, ref, MyInput{}, options) // returns MyOutput, error
func ExecuteOperation[I, O any](ctx context.Context, client *HTTPClient, operation OperationReference[I, O], input I, request ExecuteOperationOptions) (O, error) {
	var o O
	value, err := client.ExecuteOperation(ctx, operation.Name(), input, request)
	if err != nil {
		return o, err
	}
	return o, value.Consume(&o)
}

// StartOperation is the type safe version of [HTTPClient.StartOperation].
// It accepts input of type I and returns a [ClientStartOperationResult] of type O, removing the need to consume the
// [LazyValue] returned by the client method.
func StartOperation[I, O any](ctx context.Context, client *HTTPClient, operation OperationReference[I, O], input I, request StartOperationOptions) (*ClientStartOperationResult[O], error) {
	result, err := client.StartOperation(ctx, operation.Name(), input, request)
	if err != nil {
		return nil, err
	}
	if result.Successful != nil {
		var o O
		if err := result.Successful.Consume(&o); err != nil {
			return nil, err
		}
		return &ClientStartOperationResult[O]{
			Successful: o,
			Links:      result.Links,
		}, nil
	}
	handle := OperationHandle[O]{
		client:    client,
		Operation: operation.Name(),
		ID:        result.Pending.ID,
		Token:     result.Pending.Token,
	}
	return &ClientStartOperationResult[O]{
		Pending: &handle,
		Links:   result.Links,
	}, nil
}

// NewHandle is the type safe version of [HTTPClient.NewHandle].
// The [Handle.GetResult] method will return an output of type O.
func NewHandle[I, O any](client *HTTPClient, operation OperationReference[I, O], token string) (*OperationHandle[O], error) {
	if token == "" {
		return nil, errEmptyOperationToken
	}
	return &OperationHandle[O]{
		client:    client,
		Operation: operation.Name(),
		ID:        token, // Duplicate token as ID for the deprecation period.
		Token:     token,
	}, nil
}
