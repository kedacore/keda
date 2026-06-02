package internal

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"reflect"
	"time"

	"github.com/google/uuid"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"
	nexuspb "go.temporal.io/api/nexus/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/converter"
	"google.golang.org/protobuf/types/known/durationpb"
)

const pollNexusOperationTimeout = 60 * time.Second

type (
	// ClientStartNexusOperationOptions contains configuration parameters for starting a Nexus operation execution.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.StartNexusOperationOptions]
	ClientStartNexusOperationOptions struct {
		// ID - The business identifier of the operation.
		//
		// Required
		ID string
		// ScheduleToCloseTimeout - The end to end timeout for the Nexus Operation.
		//
		// Optional: defaults to the maximum allowed by the Temporal server.
		ScheduleToCloseTimeout time.Duration
		// ScheduleToStartTimeout - Maximum time to wait for an operation to be started (or completed
		// if synchronous) by the handler.
		//
		// Optional: If not set or zero, no schedule-to-start timeout is enforced.
		ScheduleToStartTimeout time.Duration
		// StartToCloseTimeout - Maximum time to wait for an asynchronous operation to complete after
		// it has been started. Only applies to asynchronous operations. Ignored for synchronous operations.
		//
		// Optional: If not set or zero, no start-to-close timeout is enforced.
		StartToCloseTimeout time.Duration
		// IDConflictPolicy - Defines how to resolve an operation id conflict with a running operation.
		//
		// Optional: Defaults to NEXUS_OPERATION_ID_CONFLICT_POLICY_FAIL.
		IDConflictPolicy enumspb.NexusOperationIdConflictPolicy
		// IDReusePolicy - Defines whether to allow re-using an operation ID from a previously closed operation.
		//
		// Optional: Defaults to NEXUS_OPERATION_ID_REUSE_POLICY_ALLOW_DUPLICATE.
		IDReusePolicy enumspb.NexusOperationIdReusePolicy
		// SearchAttributes - Specifies Search Attributes that will be attached to the operation.
		//
		// Optional: default to none.
		SearchAttributes SearchAttributes
		// Summary is a single-line summary for this operation that will appear in UI/CLI.
		//
		// Optional: defaults to none/empty.
		Summary string
	}

	// ClientNexusClientOptions contains options for creating a NexusClient.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.NexusClientOptions]
	ClientNexusClientOptions struct {
		// Endpoint - The Nexus endpoint name.
		//
		// Required
		Endpoint string
		// Service - The Nexus service name.
		//
		// Required
		Service string
	}

	// ClientGetNexusOperationHandleOptions contains input for GetNexusOperationHandle call.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.GetNexusOperationHandleOptions]
	ClientGetNexusOperationHandleOptions struct {
		// OperationID - The operation ID.
		//
		// Required
		OperationID string
		// RunID - The run ID. Can be empty to target the latest run.
		//
		// Optional: defaults to empty.
		RunID string
	}

	// ClientDescribeNexusOperationOptions contains options for ClientNexusOperationHandle.Describe call.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.DescribeNexusOperationOptions]
	ClientDescribeNexusOperationOptions struct {
	}

	// ClientCancelNexusOperationOptions contains options for ClientNexusOperationHandle.Cancel call.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.CancelNexusOperationOptions]
	ClientCancelNexusOperationOptions struct {
		// Reason is optional description of the reason for cancellation.
		Reason string
	}

	// ClientTerminateNexusOperationOptions contains options for ClientNexusOperationHandle.Terminate call.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.TerminateNexusOperationOptions]
	ClientTerminateNexusOperationOptions struct {
		// Reason is optional description of the reason for termination.
		Reason string
	}

	// ClientListNexusOperationsOptions contains input for ListNexusOperations call.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.ListNexusOperationsOptions]
	ClientListNexusOperationsOptions struct {
		// Query is a visibility query for listing Nexus operations.
		// See https://docs.temporal.io/list-filter for the syntax.
		Query string
	}

	// ClientCountNexusOperationsOptions contains input for CountNexusOperations call.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.CountNexusOperationsOptions]
	ClientCountNexusOperationsOptions struct {
		// Query is a visibility query for counting Nexus operations.
		// See https://docs.temporal.io/list-filter for the syntax.
		Query string
	}

	// ClientNexusOperationMetadata contains information about a Nexus operation execution.
	// This is returned by ListNexusOperations and embedded in ClientNexusOperationExecutionDescription.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.NexusOperationMetadata]
	ClientNexusOperationMetadata struct {
		// RawExecutionListInfo is the raw PB message this struct was built from. This field is nil
		// in the result of ClientNexusOperationHandle.Describe call - use
		// ClientNexusOperationExecutionDescription.RawInfo instead.
		RawExecutionListInfo *nexuspb.NexusOperationExecutionListInfo
		// OperationID is the unique identifier of this operation within its namespace.
		OperationID string
		// OperationRunID is the run ID of the operation.
		OperationRunID string
		// Endpoint is the Nexus endpoint name.
		Endpoint string
		// Service is the Nexus service name.
		Service string
		// Operation is the Nexus operation name.
		Operation string
		// ScheduledTime is the time when the operation was originally scheduled.
		ScheduledTime time.Time
		// CloseTime is the time when the operation transitioned to a terminal state.
		CloseTime time.Time
		// Status is the current execution status of the operation.
		Status enumspb.NexusOperationExecutionStatus
		// SearchAttributes are the search attributes attached to this operation.
		SearchAttributes SearchAttributes
		// StateTransitionCount is incremented each time the operation state is mutated.
		StateTransitionCount int64
		// ExecutionDuration is the difference between close time and scheduled time.
		// Only populated if the operation is closed.
		ExecutionDuration time.Duration
	}

	// ClientNexusOperationExecutionDescription contains detailed information about a Nexus operation execution.
	// This is returned by ClientNexusOperationHandle.Describe.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.NexusOperationExecutionDescription]
	ClientNexusOperationExecutionDescription struct {
		ClientNexusOperationMetadata
		// RawInfo is the raw PB message this struct was built from.
		RawInfo *nexuspb.NexusOperationExecutionInfo
		// State is a more detailed breakdown of the running status.
		State enumspb.PendingNexusOperationState
		// ScheduleToCloseTimeout is the schedule-to-close timeout for this operation.
		ScheduleToCloseTimeout time.Duration
		// ScheduleToStartTimeout is the schedule-to-start timeout for this operation.
		// May not be populated by all server versions.
		ScheduleToStartTimeout time.Duration
		// StartToCloseTimeout is the start-to-close timeout for this operation.
		// May not be populated by all server versions.
		StartToCloseTimeout time.Duration
		// Attempt is the number of attempts made to start/deliver the operation request.
		Attempt int32
		// ExpirationTime is the scheduled time plus schedule-to-close timeout.
		ExpirationTime time.Time
		// LastAttemptCompleteTime is the time when the last attempt completed.
		LastAttemptCompleteTime time.Time
		// NextAttemptScheduleTime is the time when the next attempt is scheduled.
		NextAttemptScheduleTime time.Time
		// LastAttemptFailure is the last attempt's failure, if any.
		LastAttemptFailure *failurepb.Failure
		// BlockedReason provides additional information if the state is BLOCKED.
		BlockedReason string
		// OperationToken is only set for asynchronous operations after a successful StartOperation call.
		OperationToken string
		// Identity is the identity of the client who started this operation.
		Identity string
		// CancellationInfo contains cancellation information if cancellation has been requested.
		CancellationInfo      *ClientNexusOperationCancellationInfo
		dc                    converter.DataConverter
		failureConverter      converter.FailureConverter
		inboundPayloadVisitor PayloadVisitor
	}

	// ClientNexusOperationCancellationInfo contains cancellation information for a Nexus operation.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.NexusOperationCancellationInfo]
	ClientNexusOperationCancellationInfo struct {
		// RawInfo is the raw PB message this struct was built from.
		RawInfo *nexuspb.NexusOperationExecutionCancellationInfo
		// RequestedTime is the time when cancellation was requested.
		RequestedTime time.Time
		// State is the current state of the cancellation request.
		State enumspb.NexusOperationCancellationState
		// Attempt is the number of attempts made to deliver the cancel operation request.
		Attempt int32
		// LastAttemptCompleteTime is the time when the last cancellation attempt completed.
		LastAttemptCompleteTime time.Time
		// NextAttemptScheduleTime is the time when the next cancellation attempt is scheduled.
		NextAttemptScheduleTime time.Time
		// BlockedReason provides additional information if the cancellation state is BLOCKED.
		BlockedReason string
		// Reason is the reason specified in the cancellation request.
		Reason                string
		lastAttemptFailure    *failurepb.Failure
		failureConverter      converter.FailureConverter
		inboundPayloadVisitor PayloadVisitor
	}

	// ClientCountNexusOperationsResult contains the result of the CountNexusOperations call.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.CountNexusOperationsResult]
	ClientCountNexusOperationsResult struct {
		// Count is the approximate number of operations matching the query.
		Count int64
		// Groups contains aggregation groups if the query includes a GROUP BY clause.
		Groups []ClientCountNexusOperationsAggregationGroup
	}

	// ClientCountNexusOperationsAggregationGroup contains groups of Nexus operations if
	// CountNexusOperationExecutions is grouped by a field.
	// The list might not be complete, and the counts of each group is approximate.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.CountNexusOperationsAggregationGroup]
	ClientCountNexusOperationsAggregationGroup struct {
		// GroupValues contains the group-by field values for this group.
		GroupValues []any
		// Count is the approximate number of operations in this group.
		Count int64
	}

	// ClientListNexusOperationsResult contains the result of the ListNexusOperations call.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.ListNexusOperationsResult]
	ClientListNexusOperationsResult struct {
		// Results is an iterator over Nexus operation metadata entries.
		Results iter.Seq2[*ClientNexusOperationMetadata, error]
	}

	// ClientNexusClient is the client for starting Nexus operations bound to a specific endpoint and service.
	// This is for standalone Nexus operations outside of workflow context.
	// For Nexus operations within workflows, use workflow.NexusClient instead.
	//
	// Methods may be added to this interface; implementing it directly is discouraged.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.NexusClient]
	ClientNexusClient interface {
		// ExecuteOperation starts a Nexus operation and returns a handle to it.
		//
		// NOTE: Experimental
		ExecuteOperation(ctx context.Context, operation any, input any, options ClientStartNexusOperationOptions) (ClientNexusOperationHandle, error)
	}

	// ClientNexusOperationHandle represents a running or completed standalone Nexus operation execution.
	// It can be used to get the result, describe, cancel, or terminate the operation.
	//
	// Methods may be added to this interface; implementing it directly is discouraged.
	//
	// NOTE: Experimental
	//
	// Exposed as: [go.temporal.io/sdk/client.NexusOperationHandle]
	ClientNexusOperationHandle interface {
		// GetID returns the ID of the operation this handle points to.
		//
		// NOTE: Experimental
		GetID() string
		// GetRunID returns the run ID that this handle was created with.
		//
		// NOTE: Experimental
		GetRunID() string
		// Get waits until the operation finishes and gets its result. If the operation completes
		// successfully, the result is written to valuePtr and nil is returned. If the operation
		// failed, the failure is returned as an error.
		//
		// NOTE: Experimental
		Get(ctx context.Context, valuePtr any) error
		// Describe returns detailed information about current state of the operation execution.
		//
		// NOTE: Experimental
		Describe(ctx context.Context, options ClientDescribeNexusOperationOptions) (*ClientNexusOperationExecutionDescription, error)
		// Cancel requests cancellation of the operation.
		//
		// NOTE: Experimental
		Cancel(ctx context.Context, options ClientCancelNexusOperationOptions) error
		// Terminate terminates the operation.
		//
		// NOTE: Experimental
		Terminate(ctx context.Context, options ClientTerminateNexusOperationOptions) error
	}

	// nexusClientImpl is the default implementation of ClientNexusClient.
	nexusClientImpl struct {
		client   *WorkflowClient
		endpoint string
		service  string
	}

	// clientNexusOperationHandleImpl is the default implementation of ClientNexusOperationHandle.
	clientNexusOperationHandleImpl struct {
		client *WorkflowClient
		id     string
		runID  string
		result *ClientPollNexusOperationResultOutput
	}
)

var (
	_ ClientNexusClient          = (*nexusClientImpl)(nil)
	_ ClientNexusOperationHandle = (*clientNexusOperationHandleImpl)(nil)
)

// GetSummary returns summary of the operation. See ClientStartNexusOperationOptions.Summary.
// Returns empty string if there is no summary.
// Uses the data converter of the client used to make the Describe call. Returns error if data conversion fails.
//
// NOTE: Experimental
func (d *ClientNexusOperationExecutionDescription) GetSummary() (string, error) {
	payload := d.RawInfo.GetUserMetadata().GetSummary()
	if payload == nil {
		return "", nil
	}
	var err error
	if payload, err = visitPayload(context.Background(), d.inboundPayloadVisitor, payload); err != nil {
		return "", err
	}
	var summary string
	err = d.dc.FromPayload(payload, &summary)
	if err != nil {
		return "", err
	}
	return summary, nil
}

// GetLastAttemptFailure returns the last attempt failure of the operation, using the failure
// converter of the client used to make the Describe call. Returns nil if there was no failure.
//
// NOTE: Experimental
func (d *ClientNexusOperationExecutionDescription) GetLastAttemptFailure() error {
	failure := d.LastAttemptFailure
	if failure == nil {
		return nil
	}
	if err := visitProtoPayloads(context.Background(), d.inboundPayloadVisitor, failure, 0); err != nil {
		return err
	}
	return d.failureConverter.FailureToError(failure)
}

// GetLastAttemptFailure returns the last attempt failure of the cancellation info.
// Returns nil if there was no failure.
//
// NOTE: Experimental
func (c *ClientNexusOperationCancellationInfo) GetLastAttemptFailure() error {
	if c.lastAttemptFailure == nil {
		return nil
	}
	if err := visitProtoPayloads(context.Background(), c.inboundPayloadVisitor, c.lastAttemptFailure, 0); err != nil {
		return err
	}
	return c.failureConverter.FailureToError(c.lastAttemptFailure)
}

func (nc *nexusClientImpl) ExecuteOperation(ctx context.Context, operation any, input any, options ClientStartNexusOperationOptions) (ClientNexusOperationHandle, error) {
	if err := nc.client.ensureInitialized(ctx); err != nil {
		return nil, err
	}

	// Resolve operation name from the operation parameter
	operationName, err := resolveNexusOperationName(operation, input)
	if err != nil {
		return nil, err
	}

	// Set header before interceptor run so interceptors can access it
	ctx = contextWithNewHeader(ctx)

	return nc.client.interceptor.ExecuteNexusOperation(ctx, &ClientExecuteNexusOperationInput{
		Options:       &options,
		Endpoint:      nc.endpoint,
		Service:       nc.service,
		OperationType: operationName,
		Input:         input,
	})
}

// resolveNexusOperationName resolves a Nexus operation name from the given value.
// It accepts a string name or a typed operation reference (with Name() and InputType() methods).
// This matches the resolution logic used in workflow context (see prepareNexusOperationParams).
func resolveNexusOperationName(operation any, input any) (string, error) {
	if name, ok := operation.(string); ok {
		if name == "" {
			return "", fmt.Errorf("operation name must not be empty")
		}
		return name, nil
	}
	if regOp, ok := operation.(interface {
		Name() string
		InputType() reflect.Type
	}); ok {
		operationName := regOp.Name()
		inputType := reflect.TypeOf(input)
		if inputType != nil && !inputType.AssignableTo(regOp.InputType()) {
			return "", fmt.Errorf("cannot assign argument of type %q to type %q for operation %q", inputType, regOp.InputType(), operationName)
		}
		return operationName, nil
	}
	return "", fmt.Errorf("invalid 'operation' parameter, must be an OperationReference or a string")
}

func (h *clientNexusOperationHandleImpl) GetID() string {
	return h.id
}

func (h *clientNexusOperationHandleImpl) GetRunID() string {
	return h.runID
}

func (h *clientNexusOperationHandleImpl) Get(ctx context.Context, valuePtr any) error {
	if h.result != nil {
		if h.result.Error != nil {
			return h.result.Error
		}
		if h.result.Result != nil {
			if valuePtr == nil {
				return nil
			}
			return h.result.Result.Get(valuePtr)
		}
	}
	if err := h.client.ensureInitialized(ctx); err != nil {
		return err
	}

	// repeatedly poll, the loop repeats until there's an outcome
	for {
		resp, err := h.client.interceptor.PollNexusOperationResult(ctx, &ClientPollNexusOperationResultInput{
			OperationID: h.id,
			RunID:       h.runID,
		})
		if err != nil {
			return err
		}
		if resp.Error != nil {
			h.result = &ClientPollNexusOperationResultOutput{Error: resp.Error}
			return resp.Error
		}
		if resp.Result != nil {
			h.result = &ClientPollNexusOperationResultOutput{Result: resp.Result}
			if valuePtr == nil {
				return nil
			}
			return resp.Result.Get(valuePtr)
		}
	}
}

func (h *clientNexusOperationHandleImpl) Describe(ctx context.Context, options ClientDescribeNexusOperationOptions) (*ClientNexusOperationExecutionDescription, error) {
	if err := h.client.ensureInitialized(ctx); err != nil {
		return nil, err
	}
	out, err := h.client.interceptor.DescribeNexusOperation(ctx, &ClientDescribeNexusOperationInput{
		OperationID: h.id,
		RunID:       h.runID,
	})
	if err != nil {
		return nil, err
	}
	return out.Description, nil
}

func (h *clientNexusOperationHandleImpl) Cancel(ctx context.Context, options ClientCancelNexusOperationOptions) error {
	if err := h.client.ensureInitialized(ctx); err != nil {
		return err
	}
	return h.client.interceptor.CancelNexusOperation(ctx, &ClientCancelNexusOperationInput{
		OperationID: h.id,
		RunID:       h.runID,
		Reason:      options.Reason,
	})
}

func (h *clientNexusOperationHandleImpl) Terminate(ctx context.Context, options ClientTerminateNexusOperationOptions) error {
	if err := h.client.ensureInitialized(ctx); err != nil {
		return err
	}
	return h.client.interceptor.TerminateNexusOperation(ctx, &ClientTerminateNexusOperationInput{
		OperationID: h.id,
		RunID:       h.runID,
		Reason:      options.Reason,
	})
}

// WorkflowClient methods for Nexus operations

func (wc *WorkflowClient) NewNexusClient(options ClientNexusClientOptions) (ClientNexusClient, error) {
	if options.Endpoint == "" {
		return nil, errors.New("endpoint is required")
	}
	if options.Service == "" {
		return nil, errors.New("service is required")
	}
	return &nexusClientImpl{client: wc, endpoint: options.Endpoint, service: options.Service}, nil
}

func (wc *WorkflowClient) GetNexusOperationHandle(options ClientGetNexusOperationHandleOptions) ClientNexusOperationHandle {
	return wc.interceptor.GetNexusOperationHandle(&ClientGetNexusOperationHandleInput{
		OperationID: options.OperationID,
		RunID:       options.RunID,
	})
}

// ListNexusOperations does not go through the interceptor chain, consistent with ListActivities.
func (wc *WorkflowClient) ListNexusOperations(ctx context.Context, options ClientListNexusOperationsOptions) (ClientListNexusOperationsResult, error) {
	return ClientListNexusOperationsResult{
		Results: func(yield func(*ClientNexusOperationMetadata, error) bool) {
			if err := wc.ensureInitialized(ctx); err != nil {
				yield(nil, err)
				return
			}

			request := &workflowservice.ListNexusOperationExecutionsRequest{
				Namespace: wc.namespace,
				Query:     options.Query,
			}

			for {
				resp, err := wc.getListNexusOperationsPage(ctx, request)
				if err != nil {
					yield(nil, err)
					return
				}

				for _, op := range resp.Operations {
					if !yield(&ClientNexusOperationMetadata{
						RawExecutionListInfo: op,
						OperationID:          op.OperationId,
						OperationRunID:       op.RunId,
						Endpoint:             op.Endpoint,
						Service:              op.Service,
						Operation:            op.Operation,
						ScheduledTime:        op.ScheduleTime.AsTime(),
						CloseTime:            op.CloseTime.AsTime(),
						Status:               op.Status,
						SearchAttributes:     convertToTypedSearchAttributes(wc.logger, op.SearchAttributes.GetIndexedFields()),
						StateTransitionCount: op.StateTransitionCount,
						ExecutionDuration:    op.ExecutionDuration.AsDuration(),
					}, nil) {
						return
					}
				}

				if resp.NextPageToken != nil {
					request.NextPageToken = resp.NextPageToken
				} else {
					return
				}
			}
		},
	}, nil
}

func (wc *WorkflowClient) getListNexusOperationsPage(ctx context.Context, request *workflowservice.ListNexusOperationExecutionsRequest) (*workflowservice.ListNexusOperationExecutionsResponse, error) {
	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()

	return wc.WorkflowService().ListNexusOperationExecutions(grpcCtx, request)
}

// CountNexusOperations does not go through the interceptor chain, consistent with CountActivities.
func (wc *WorkflowClient) CountNexusOperations(ctx context.Context, options ClientCountNexusOperationsOptions) (*ClientCountNexusOperationsResult, error) {
	if err := wc.ensureInitialized(ctx); err != nil {
		return nil, err
	}

	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()

	request := &workflowservice.CountNexusOperationExecutionsRequest{
		Namespace: wc.namespace,
		Query:     options.Query,
	}
	resp, err := wc.WorkflowService().CountNexusOperationExecutions(grpcCtx, request)
	if err != nil {
		return nil, err
	}

	groups := make([]ClientCountNexusOperationsAggregationGroup, len(resp.Groups))
	for i, group := range resp.Groups {
		groupValues := make([]any, len(group.GroupValues))
		for j, groupValue := range group.GroupValues {
			// should never fail, and if it does, leaving nil behind
			_ = converter.GetDefaultDataConverter().FromPayload(groupValue, &groupValues[j])
		}
		groups[i] = ClientCountNexusOperationsAggregationGroup{
			GroupValues: groupValues,
			Count:       group.Count,
		}
	}

	return &ClientCountNexusOperationsResult{
		Count:  resp.Count,
		Groups: groups,
	}, nil
}

// workflowClientInterceptor implementations for Nexus operations

func (w *workflowClientInterceptor) ExecuteNexusOperation(
	ctx context.Context,
	in *ClientExecuteNexusOperationInput,
) (ClientNexusOperationHandle, error) {
	dataConverter := WithContext(ctx, w.client.dataConverter)
	if dataConverter == nil {
		dataConverter = converter.GetDefaultDataConverter()
	}

	if in.Options.ID == "" {
		return nil, errors.New("operation ID is required")
	}
	if in.Options.ScheduleToCloseTimeout < 0 {
		return nil, errors.New("ScheduleToCloseTimeout must not be negative")
	}

	// Encode input as a single Payload (not Payloads)
	inputPayload, err := dataConverter.ToPayload(in.Input)
	if err != nil {
		return nil, err
	}

	searchAttrs, err := serializeTypedSearchAttributes(in.Options.SearchAttributes.GetUntypedValues())
	if err != nil {
		return nil, err
	}

	userMetadata, err := buildUserMetadata(in.Options.Summary, "", dataConverter)
	if err != nil {
		return nil, err
	}

	request := &workflowservice.StartNexusOperationExecutionRequest{
		Namespace:        w.client.namespace,
		Identity:         w.client.identity,
		RequestId:        uuid.NewString(),
		OperationId:      in.Options.ID,
		Endpoint:         in.Endpoint,
		Service:          in.Service,
		Operation:        in.OperationType,
		Input:            inputPayload,
		IdReusePolicy:    in.Options.IDReusePolicy,
		IdConflictPolicy: in.Options.IDConflictPolicy,
		SearchAttributes: searchAttrs,
		UserMetadata:     userMetadata,
	}
	if in.Options.ScheduleToCloseTimeout > 0 {
		request.ScheduleToCloseTimeout = durationpb.New(in.Options.ScheduleToCloseTimeout)
	}
	if in.Options.ScheduleToStartTimeout > 0 {
		request.ScheduleToStartTimeout = durationpb.New(in.Options.ScheduleToStartTimeout)
	}
	if in.Options.StartToCloseTimeout > 0 {
		request.StartToCloseTimeout = durationpb.New(in.Options.StartToCloseTimeout)
	}
	if err := visitProtoPayloads(ctx, w.outboundPayloadVisitor, request, 0); err != nil {
		return nil, err
	}

	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()

	resp, err := w.client.WorkflowService().StartNexusOperationExecution(grpcCtx, request)
	if err != nil {
		return nil, err
	}

	return &clientNexusOperationHandleImpl{
		client: w.client,
		id:     in.Options.ID,
		runID:  resp.RunId,
	}, nil
}

func (w *workflowClientInterceptor) GetNexusOperationHandle(
	in *ClientGetNexusOperationHandleInput,
) ClientNexusOperationHandle {
	return &clientNexusOperationHandleImpl{
		client: w.client,
		id:     in.OperationID,
		runID:  in.RunID,
	}
}

func (w *workflowClientInterceptor) PollNexusOperationResult(
	ctx context.Context,
	in *ClientPollNexusOperationResultInput,
) (*ClientPollNexusOperationResultOutput, error) {
	request := &workflowservice.PollNexusOperationExecutionRequest{
		Namespace:   w.client.namespace,
		OperationId: in.OperationID,
		RunId:       in.RunID,
		WaitStage:   enumspb.NEXUS_OPERATION_WAIT_STAGE_CLOSED,
	}

	var resp *workflowservice.PollNexusOperationExecutionResponse
	for resp.GetOutcome() == nil {
		grpcCtx, cancel := newGRPCContext(ctx, grpcLongPoll(true), grpcTimeout(pollNexusOperationTimeout), defaultGrpcRetryParameters(ctx))
		var err error
		resp, err = w.client.WorkflowService().PollNexusOperationExecution(grpcCtx, request)
		cancel()
		if err != nil {
			return nil, err
		}
	}

	if err := visitProtoPayloads(ctx, w.inboundPayloadVisitor, resp, 0); err != nil {
		return nil, err
	}

	switch v := resp.GetOutcome().(type) {
	case *workflowservice.PollNexusOperationExecutionResponse_Result:
		// Wrap single Payload in Payloads for EncodedValue compatibility
		payloads := &commonpb.Payloads{Payloads: []*commonpb.Payload{v.Result}}
		return &ClientPollNexusOperationResultOutput{Result: newEncodedValue(payloads, w.client.dataConverter)}, nil
	case *workflowservice.PollNexusOperationExecutionResponse_Failure:
		return &ClientPollNexusOperationResultOutput{Error: w.client.failureConverter.FailureToError(v.Failure)}, nil
	default:
		return nil, fmt.Errorf("unexpected nexus operation outcome type: %T", v)
	}
}

func (w *workflowClientInterceptor) DescribeNexusOperation(
	ctx context.Context,
	in *ClientDescribeNexusOperationInput,
) (*ClientDescribeNexusOperationOutput, error) {
	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()

	request := &workflowservice.DescribeNexusOperationExecutionRequest{
		Namespace:   w.client.namespace,
		OperationId: in.OperationID,
		RunId:       in.RunID,
	}
	resp, err := w.client.WorkflowService().DescribeNexusOperationExecution(grpcCtx, request)
	if err != nil {
		return nil, err
	}
	info := resp.GetInfo()
	if info == nil {
		return nil, errors.New("DescribeNexusOperationExecution response doesn't contain info")
	}

	var cancellationInfo *ClientNexusOperationCancellationInfo
	if info.CancellationInfo != nil {
		cancellationInfo = &ClientNexusOperationCancellationInfo{
			RawInfo:                 info.CancellationInfo,
			RequestedTime:           info.CancellationInfo.RequestedTime.AsTime(),
			State:                   info.CancellationInfo.State,
			Attempt:                 info.CancellationInfo.Attempt,
			LastAttemptCompleteTime: info.CancellationInfo.LastAttemptCompleteTime.AsTime(),
			NextAttemptScheduleTime: info.CancellationInfo.NextAttemptScheduleTime.AsTime(),
			BlockedReason:           info.CancellationInfo.BlockedReason,
			Reason:                  info.CancellationInfo.Reason,
			lastAttemptFailure:      info.CancellationInfo.LastAttemptFailure,
			failureConverter:        w.client.failureConverter,
			inboundPayloadVisitor:   w.inboundPayloadVisitor,
		}
	}

	return &ClientDescribeNexusOperationOutput{
		Description: &ClientNexusOperationExecutionDescription{
			ClientNexusOperationMetadata: ClientNexusOperationMetadata{
				RawExecutionListInfo: nil,
				OperationID:          info.OperationId,
				OperationRunID:       info.RunId,
				Endpoint:             info.Endpoint,
				Service:              info.Service,
				Operation:            info.Operation,
				ScheduledTime:        info.ScheduleTime.AsTime(),
				CloseTime:            info.CloseTime.AsTime(),
				Status:               info.Status,
				SearchAttributes:     convertToTypedSearchAttributes(w.client.logger, info.SearchAttributes.GetIndexedFields()),
				StateTransitionCount: info.StateTransitionCount,
				ExecutionDuration:    info.ExecutionDuration.AsDuration(),
			},
			RawInfo:                 info,
			State:                   info.State,
			ScheduleToCloseTimeout:  info.ScheduleToCloseTimeout.AsDuration(),
			ScheduleToStartTimeout:  info.ScheduleToStartTimeout.AsDuration(),
			StartToCloseTimeout:     info.StartToCloseTimeout.AsDuration(),
			Attempt:                 info.Attempt,
			ExpirationTime:          info.ExpirationTime.AsTime(),
			LastAttemptCompleteTime: info.LastAttemptCompleteTime.AsTime(),
			NextAttemptScheduleTime: info.NextAttemptScheduleTime.AsTime(),
			LastAttemptFailure:      info.LastAttemptFailure,
			BlockedReason:           info.BlockedReason,
			OperationToken:          info.OperationToken,
			Identity:                info.Identity,
			CancellationInfo:        cancellationInfo,
			dc:                      WithContext(ctx, w.client.dataConverter),
			failureConverter:        w.client.failureConverter,
			inboundPayloadVisitor:   w.inboundPayloadVisitor,
		},
	}, nil
}

func (w *workflowClientInterceptor) CancelNexusOperation(
	ctx context.Context,
	in *ClientCancelNexusOperationInput,
) error {
	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()

	request := &workflowservice.RequestCancelNexusOperationExecutionRequest{
		Namespace:   w.client.namespace,
		OperationId: in.OperationID,
		RunId:       in.RunID,
		Identity:    w.client.identity,
		RequestId:   uuid.NewString(),
		Reason:      in.Reason,
	}
	_, err := w.client.WorkflowService().RequestCancelNexusOperationExecution(grpcCtx, request)
	return err
}

func (w *workflowClientInterceptor) TerminateNexusOperation(
	ctx context.Context,
	in *ClientTerminateNexusOperationInput,
) error {
	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()

	request := &workflowservice.TerminateNexusOperationExecutionRequest{
		Namespace:   w.client.namespace,
		OperationId: in.OperationID,
		RunId:       in.RunID,
		Identity:    w.client.identity,
		RequestId:   uuid.NewString(),
		Reason:      in.Reason,
	}
	_, err := w.client.WorkflowService().TerminateNexusOperationExecution(grpcCtx, request)
	return err
}
