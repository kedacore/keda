package internal

import (
	"context"
	"errors"
	"sync/atomic"

	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/proxy"
	querypb "go.temporal.io/api/query/v1"
	schedulepb "go.temporal.io/api/schedule/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/log"
	"google.golang.org/protobuf/proto"
)

// PayloadLimitOptions for when payload sizes exceed limits.
//
// NOTE: Experimental
//
// Exposed as: [go.temporal.io/sdk/client.PayloadLimitOptions]
type PayloadLimitOptions struct {
	// The limit (in bytes) at which a payload size warning is logged.
	// If unspecified or zero, defaults to 512 KiB.
	PayloadSizeWarning int
	// The limit (in bytes) at which an aggregate memo size warning is logged.
	// If unspecified or zero, defaults to 512 KiB.
	MemoSizeWarning int
}

type payloadLimitCheckKey struct{}
type memoLimitCheckKey struct{}

func withPayloadLimitChecks(ctx context.Context, checks limitCheck) context.Context {
	return context.WithValue(ctx, payloadLimitCheckKey{}, checks)
}

func withMemoLimitChecks(ctx context.Context, checks limitCheck) context.Context {
	return context.WithValue(ctx, memoLimitCheckKey{}, checks)
}

func getPayloadLimitChecks(ctx context.Context) limitCheck {
	if ctx != nil {
		if v, ok := ctx.Value(payloadLimitCheckKey{}).(limitCheck); ok {
			return v
		}
	}
	return limitCheckAll
}

func getMemoLimitChecks(ctx context.Context) limitCheck {
	if ctx != nil {
		if v, ok := ctx.Value(memoLimitCheckKey{}).(limitCheck); ok {
			return v
		}
	}
	return limitCheckAll
}

type limitCheck uint8

const (
	limitCheckNone    limitCheck = 0
	limitCheckError   limitCheck = 1 << 0
	limitCheckWarning limitCheck = 1 << 1
	limitCheckAll                = limitCheckError | limitCheckWarning
)

type payloadSizeError struct {
	message string
	size    int64
	limit   int64
}

func (e payloadSizeError) Error() string {
	return e.message
}

type payloadLimits struct {
	payloadSize int64
	memoSize    int64
}

func payloadLimitOptionsToLimits(options PayloadLimitOptions) (payloadLimits, error) {
	payloadSizeWarning := int64(options.PayloadSizeWarning)
	if payloadSizeWarning < 0 {
		return payloadLimits{}, errors.New("PayloadSizeWarning must be greater than or equal to zero")
	}
	if payloadSizeWarning == 0 {
		payloadSizeWarning = 512 * 1024
	}
	memoSizeWarning := int64(options.MemoSizeWarning)
	if memoSizeWarning < 0 {
		return payloadLimits{}, errors.New("MemoSizeWarning must be greater than or equal to zero")
	}
	if memoSizeWarning == 0 {
		memoSizeWarning = 2 * 1024
	}
	return payloadLimits{
		payloadSize: payloadSizeWarning,
		memoSize:    memoSizeWarning,
	}, nil
}

type payloadLimitsVisitorImpl struct {
	errorLimits   atomic.Pointer[payloadLimits]
	warningLimits payloadLimits
	logger        log.Logger
}

var _ PayloadVisitor = (*payloadLimitsVisitorImpl)(nil)
var _ PayloadVisitorWithContextHook = (*payloadLimitsVisitorImpl)(nil)

func (v *payloadLimitsVisitorImpl) Visit(ctx *proxy.VisitPayloadsContext, payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	size := int64(0)
	if ctx.SinglePayloadRequired {
		if _, ok := ctx.Parent.(*commandpb.ScheduleNexusOperationCommandAttributes); !ok {
			return payloads, nil
		}
		if len(payloads) > 0 {
			size = int64(payloads[0].Size())
		}
	} else {
		// Rewrap into Payloads to get the measured size that the server would also observe.
		size = int64((&commonpb.Payloads{Payloads: payloads}).Size())
	}

	err := v.checkPayloadSize(size, getPayloadLimitChecks(ctx.Context))
	if err != nil {
		return nil, err
	}
	return payloads, nil
}

// ContextHook is used here to specialize the limit check logic based on how server has one-off decisions
// for each proto and field that contains payloads.
func (v *payloadLimitsVisitorImpl) ContextHook(ctx context.Context, msg proto.Message) (context.Context, error) {
	switch msg := msg.(type) {
	// RecordMarkerCommandAttributes.Details is a map[string]Payloads
	// Server has specialized size checking for map[string]Payloads for this field.
	case *commandpb.RecordMarkerCommandAttributes:
		err := v.checkPayloadSize(v.getPayloadsMapSize(msg.Details), limitCheckAll)
		if err != nil {
			return nil, err
		}
		ctx = withPayloadLimitChecks(ctx, limitCheckNone)
	// UpsertWorkflowSearchAttributesCommandAttributes.SearchAttributes is a map[string]Payload
	// Server has specialized size checking for map[string]Payload (that are not Memo fields).
	case *commandpb.UpsertWorkflowSearchAttributesCommandAttributes:
		err := v.checkPayloadSize(v.getPayloadMapSize(msg.GetSearchAttributes().GetIndexedFields()), limitCheckAll)
		if err != nil {
			return nil, err
		}
		ctx = withPayloadLimitChecks(ctx, limitCheckNone)
	// ModifyWorkflowPropertiesCommandAttributes.Properties is a map[string]Payload
	// Server has specialized size checking for map[string]Payload (that are not Memo fields).
	case *commandpb.ModifyWorkflowPropertiesCommandAttributes:
		err := v.checkPayloadSize(v.getPayloadMapSize(msg.GetUpsertedMemo().GetFields()), limitCheckAll)
		if err != nil {
			return nil, err
		}
		ctx = withPayloadLimitChecks(ctx, limitCheckNone)
	case *commonpb.Memo:
		err := v.checkMemoSize(int64(msg.Size()), getMemoLimitChecks(ctx))
		if err != nil {
			return nil, err
		}
		ctx = withPayloadLimitChecks(ctx, limitCheckNone)
	case *querypb.WorkflowQueryResult:
		err := v.checkPayloadSize(int64(msg.GetAnswer().Size()), limitCheckAll)
		// Server translates too large results into failed query results
		if err != nil {
			msg.Answer = nil
			msg.ErrorMessage = err.Error()
			msg.ResultType = enumspb.QUERY_RESULT_TYPE_FAILED
		}
		ctx = withPayloadLimitChecks(ctx, limitCheckNone)
	case *workflowservice.RespondQueryTaskCompletedRequest:
		err := v.checkPayloadSize(int64(msg.GetQueryResult().Size()), limitCheckAll)
		// Server translates too large results into failed query results
		if err != nil {
			msg.ErrorMessage = err.Error()
			msg.QueryResult = nil
			msg.CompletedType = enumspb.QUERY_RESULT_TYPE_FAILED
		}
		ctx = withPayloadLimitChecks(ctx, limitCheckNone)
	// CreateScheduleRequest has a custom size checking algorithm checked against the payload size limit.
	case *workflowservice.CreateScheduleRequest:
		// Server only supports StartWorkflow action; skip if not StartWorkflow and let the server handle it.
		if action, ok := msg.GetSchedule().GetAction().GetAction().(*schedulepb.ScheduleAction_StartWorkflow); ok {
			size := int64(msg.GetMemo().Size()) + int64(action.StartWorkflow.GetInput().Size())
			err := v.checkPayloadSize(size, limitCheckAll)
			if err != nil {
				return nil, err
			}
		}
		ctx = withMemoLimitChecks(ctx, limitCheckNone)
		ctx = withPayloadLimitChecks(ctx, limitCheckNone)
	// UpdateScheduleRequest.Memo is not validated by the server against memo size limits.
	case *workflowservice.UpdateScheduleRequest:
		ctx = withMemoLimitChecks(ctx, limitCheckWarning)
	// Failures are passed through to server which will append failure details instead of failing the workflow.
	// Skip error checking these to allow the server to receive the failures.
	case *workflowservice.RespondActivityTaskFailedRequest:
		ctx = withPayloadLimitChecks(ctx, limitCheckWarning)
		ctx = withMemoLimitChecks(ctx, limitCheckWarning)
	case *workflowservice.RespondActivityTaskFailedByIdRequest:
		ctx = withPayloadLimitChecks(ctx, limitCheckWarning)
		ctx = withMemoLimitChecks(ctx, limitCheckWarning)
	case *workflowservice.RespondWorkflowTaskFailedRequest:
		ctx = withPayloadLimitChecks(ctx, limitCheckWarning)
		ctx = withMemoLimitChecks(ctx, limitCheckWarning)
	}
	// These are the additional protos that server checks that the SDK does not currently check:
	// - UpsertWorkflowSearchAttributesCommandAttributes has another SearchAttributes size check that combines execution info,
	//   which is not available in the SDK. Violating the limit will terminate the workflow.
	// - ModifyWorkflowPropertiesCommandAttributes has another SearchAttributes size check that combines execution info,
	//   which is not available in the SDK. Violating the limit will terminate the workflow.
	// - UpdateScheduleRequest has a complicated payload sizing algorithm that combines multiple fields into another proto,
	//   proto-preferred encodes them, and checks against the payload size limit. Violating the limit returns an error to the client.
	// - PatchScheduleRequest has a Patch field that is proto-preferred encoded and checked against the payload size limit.
	//   Violating the limit returns an error to the client.
	// - ListScheduleMatchingTimesRequest is proto-preferred encoded and checked against the payload size limit.
	//   Violating the limit returns an error to the client.
	return ctx, nil
}

func (v *payloadLimitsVisitorImpl) checkPayloadSize(size int64, checks limitCheck) error {
	if checks&limitCheckError != 0 {
		errorLimits := v.errorLimits.Load()
		if errorLimits != nil && errorLimits.payloadSize > 0 && size > errorLimits.payloadSize {
			return payloadSizeError{
				message: "[TMPRL1103] Attempted to upload payloads with size that exceeded the error limit.",
				size:    size,
				limit:   errorLimits.payloadSize,
			}
		}
	}
	if checks&limitCheckWarning != 0 && v.warningLimits.payloadSize > 0 && size > v.warningLimits.payloadSize && v.logger != nil {
		v.logger.Warn(
			"[TMPRL1103] Attempted to upload payloads with size that exceeded the warning limit.",
			tagPayloadSize, size,
			tagPayloadSizeLimit, v.warningLimits.payloadSize,
		)
	}
	return nil
}

func (v *payloadLimitsVisitorImpl) checkMemoSize(size int64, checks limitCheck) error {
	if checks&limitCheckError != 0 {
		errorLimits := v.errorLimits.Load()
		if errorLimits != nil && errorLimits.memoSize > 0 && size > errorLimits.memoSize {
			return payloadSizeError{
				message: "[TMPRL1103] Attempted to upload memo with size that exceeded the error limit.",
				size:    size,
				limit:   errorLimits.memoSize,
			}
		}
	}
	if checks&limitCheckWarning != 0 && v.warningLimits.memoSize > 0 && size > v.warningLimits.memoSize && v.logger != nil {
		v.logger.Warn(
			"[TMPRL1103] Attempted to upload memo with size that exceeded the warning limit.",
			tagMemoSize, size,
			tagMemoSizeLimit, v.warningLimits.memoSize,
		)
	}
	return nil
}

func (v *payloadLimitsVisitorImpl) getPayloadMapSize(fields map[string]*commonpb.Payload) int64 {
	result := int64(0)

	for k, v := range fields {
		result += int64(len(k))
		// Intentionally measure data size, not payload size, to match server behavior.
		result += int64(len(v.GetData()))
	}
	return result
}

func (v *payloadLimitsVisitorImpl) getPayloadsMapSize(data map[string]*commonpb.Payloads) int64 {
	size := int64(0)
	for key, payloads := range data {
		size += int64(len(key))
		size += int64(payloads.Size())
	}
	return size
}

func (v *payloadLimitsVisitorImpl) setErrorLimits(errorLimits *payloadLimits) {
	v.errorLimits.Store(errorLimits)
}

func newPayloadLimitsVisitor(warningLimits payloadLimits, logger log.Logger) (PayloadVisitor, func(*payloadLimits)) {
	visitor := &payloadLimitsVisitorImpl{
		warningLimits: warningLimits,
		logger:        logger,
	}
	return visitor, visitor.setErrorLimits
}
