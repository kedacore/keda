package nexus

import (
	"context"
)

// UnimplementedHandler must be embedded into any [Handler] implementation for future compatibility.
// It implements all methods on the [Handler] interface, returning unimplemented errors if they are not implemented by
// the embedding type.
type UnimplementedHandler struct{}

func (h UnimplementedHandler) mustEmbedUnimplementedHandler() {}

// StartOperation implements the Handler interface.
func (h UnimplementedHandler) StartOperation(ctx context.Context, service, operation string, input *LazyValue, options StartOperationOptions) (HandlerStartOperationResult[any], error) {
	return nil, &HandlerError{HandlerErrorTypeNotImplemented, &Failure{Message: "not implemented"}}
}

// GetOperationResult implements the Handler interface.
func (h UnimplementedHandler) GetOperationResult(ctx context.Context, service, operation, operationID string, options GetOperationResultOptions) (any, error) {
	return nil, &HandlerError{HandlerErrorTypeNotImplemented, &Failure{Message: "not implemented"}}
}

// GetOperationInfo implements the Handler interface.
func (h UnimplementedHandler) GetOperationInfo(ctx context.Context, service, operation, operationID string, options GetOperationInfoOptions) (*OperationInfo, error) {
	return nil, &HandlerError{HandlerErrorTypeNotImplemented, &Failure{Message: "not implemented"}}
}

// CancelOperation implements the Handler interface.
func (h UnimplementedHandler) CancelOperation(ctx context.Context, service, operation, operationID string, options CancelOperationOptions) error {
	return &HandlerError{HandlerErrorTypeNotImplemented, &Failure{Message: "not implemented"}}
}

// UnimplementedOperation must be embedded into any [Operation] implementation for future compatibility.
// It implements all methods on the [Operation] interface except for `Name`, returning unimplemented errors if they are
// not implemented by the embedding type.
type UnimplementedOperation[I, O any] struct{}

func (*UnimplementedOperation[I, O]) inferType(I, O) {} //nolint:unused

func (*UnimplementedOperation[I, O]) mustEmbedUnimplementedOperation() {}

// Cancel implements Operation.
func (*UnimplementedOperation[I, O]) Cancel(context.Context, string, CancelOperationOptions) error {
	return HandlerErrorf(HandlerErrorTypeNotImplemented, "not implemented")
}

// GetInfo implements Operation.
func (*UnimplementedOperation[I, O]) GetInfo(context.Context, string, GetOperationInfoOptions) (*OperationInfo, error) {
	return nil, HandlerErrorf(HandlerErrorTypeNotImplemented, "not implemented")
}

// GetResult implements Operation.
func (*UnimplementedOperation[I, O]) GetResult(context.Context, string, GetOperationResultOptions) (O, error) {
	var empty O
	return empty, HandlerErrorf(HandlerErrorTypeNotImplemented, "not implemented")
}

// Start implements Operation.
func (h *UnimplementedOperation[I, O]) Start(ctx context.Context, input I, options StartOperationOptions) (HandlerStartOperationResult[O], error) {
	return nil, HandlerErrorf(HandlerErrorTypeNotImplemented, "not implemented")
}
