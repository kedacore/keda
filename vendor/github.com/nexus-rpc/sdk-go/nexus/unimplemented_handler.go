package nexus

import (
	"context"
	"reflect"
)

// UnimplementedHandler must be embedded into any [Handler] implementation for future compatibility.
// It implements all methods on the [Handler] interface, returning unimplemented errors if they are not implemented by
// the embedding type.
type UnimplementedHandler struct{}

func (h UnimplementedHandler) mustEmbedUnimplementedHandler() {}

// StartOperation implements the Handler interface.
func (h UnimplementedHandler) StartOperation(ctx context.Context, service, operation string, input *LazyValue, options StartOperationOptions) (HandlerStartOperationResult[any], error) {
	return nil, HandlerErrorf(HandlerErrorTypeNotImplemented, "not implemented")
}

// GetOperationResult implements the Handler interface.
func (h UnimplementedHandler) GetOperationResult(ctx context.Context, service, operation, token string, options GetOperationResultOptions) (any, error) {
	return nil, HandlerErrorf(HandlerErrorTypeNotImplemented, "not implemented")
}

// GetOperationInfo implements the Handler interface.
func (h UnimplementedHandler) GetOperationInfo(ctx context.Context, service, operation, token string, options GetOperationInfoOptions) (*OperationInfo, error) {
	return nil, HandlerErrorf(HandlerErrorTypeNotImplemented, "not implemented")
}

// CancelOperation implements the Handler interface.
func (h UnimplementedHandler) CancelOperation(ctx context.Context, service, operation, token string, options CancelOperationOptions) error {
	return HandlerErrorf(HandlerErrorTypeNotImplemented, "not implemented")
}

// UnimplementedOperation must be embedded into any [Operation] implementation for future compatibility.
// It implements all methods on the [Operation] interface except for `Name`, returning unimplemented errors if they are
// not implemented by the embedding type.
type UnimplementedOperation[I, O any] struct{}

func (*UnimplementedOperation[I, O]) inferType(I, O) {} //nolint:unused

func (*UnimplementedOperation[I, O]) mustEmbedUnimplementedOperation() {}

func (*UnimplementedOperation[I, O]) InputType() reflect.Type {
	var zero [0]I
	return reflect.TypeOf(zero).Elem()
}

func (*UnimplementedOperation[I, O]) OutputType() reflect.Type {
	var zero [0]O
	return reflect.TypeOf(zero).Elem()
}

// Cancel implements Operation.
func (*UnimplementedOperation[I, O]) Cancel(ctx context.Context, token string, options CancelOperationOptions) error {
	return HandlerErrorf(HandlerErrorTypeNotImplemented, "not implemented")
}

// GetInfo implements Operation.
func (*UnimplementedOperation[I, O]) GetInfo(ctx context.Context, token string, options GetOperationInfoOptions) (*OperationInfo, error) {
	return nil, HandlerErrorf(HandlerErrorTypeNotImplemented, "not implemented")
}

// GetResult implements Operation.
func (*UnimplementedOperation[I, O]) GetResult(ctx context.Context, token string, options GetOperationResultOptions) (O, error) {
	var empty O
	return empty, HandlerErrorf(HandlerErrorTypeNotImplemented, "not implemented")
}

// Start implements Operation.
func (h *UnimplementedOperation[I, O]) Start(ctx context.Context, input I, options StartOperationOptions) (HandlerStartOperationResult[O], error) {
	return nil, HandlerErrorf(HandlerErrorTypeNotImplemented, "not implemented")
}
