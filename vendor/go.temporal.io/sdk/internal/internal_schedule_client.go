// The MIT License
//
// Copyright (c) 2022 Temporal Technologies Inc.  All rights reserved.
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
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/pborman/uuid"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	schedulepb "go.temporal.io/api/schedule/v1"
	"go.temporal.io/api/serviceerror"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	workflowpb "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/internal/common"
)

type (

	// ScheduleClient is the client for starting a workflow execution.
	scheduleClient struct {
		workflowClient *WorkflowClient
	}

	// scheduleHandleImpl is the implementation of ScheduleHandle.
	scheduleHandleImpl struct {
		ID     string
		client *WorkflowClient
	}

	// scheduleListIteratorImpl is the implementation of ScheduleListIterator
	scheduleListIteratorImpl struct {
		// nextScheduleIndex - Local cached schedules events and corresponding consuming index
		nextScheduleIndex int

		// err - From getting the latest page of schedules
		err error

		// response - From getting the latest page of schedules
		response *workflowservice.ListSchedulesResponse

		// paginate - Function which use a next token to get next page of schedules events
		paginate func(nexttoken []byte) (*workflowservice.ListSchedulesResponse, error)
	}
)

func (w *workflowClientInterceptor) CreateSchedule(ctx context.Context, in *ScheduleClientCreateInput) (ScheduleHandle, error) {
	// This is always set before interceptor is invoked
	ID := in.Options.ID
	if ID == "" {
		return nil, fmt.Errorf("no schedule ID in options")
	}

	dataConverter := WithContext(ctx, w.client.dataConverter)
	if dataConverter == nil {
		dataConverter = converter.GetDefaultDataConverter()
	}

	if in.Options.Action == nil {
		return nil, fmt.Errorf("no schedule action in options")
	}
	action, err := convertToPBScheduleAction(ctx, w.client, in.Options.Action)
	if err != nil {
		return nil, err
	}

	memo, err := getWorkflowMemo(in.Options.Memo, dataConverter)
	if err != nil {
		return nil, err
	}

	searchAttr, err := serializeSearchAttributes(in.Options.SearchAttributes)
	if err != nil {
		return nil, err
	}

	var triggerImmediately *schedulepb.TriggerImmediatelyRequest
	if in.Options.TriggerImmediately {
		triggerImmediately = &schedulepb.TriggerImmediatelyRequest{
			OverlapPolicy: in.Options.Overlap,
		}
	}

	backfillRequests := convertToPBBackfillList(in.Options.ScheduleBackfill)

	// Only send an initial patch if we need to.
	var initialPatch *schedulepb.SchedulePatch
	if in.Options.TriggerImmediately || len(in.Options.ScheduleBackfill) > 0 {
		initialPatch = &schedulepb.SchedulePatch{
			TriggerImmediately: triggerImmediately,
			BackfillRequest:    backfillRequests,
		}
	}

	catchupWindow := &in.Options.CatchupWindow
	if in.Options.CatchupWindow == 0 {
		// Convert to nil so the server uses the default
		// catchup window,otherwise it will use the minimum (10s).
		catchupWindow = nil
	}

	// run propagators to extract information about tracing and other stuff, store in headers field
	startRequest := &workflowservice.CreateScheduleRequest{
		Namespace:  w.client.namespace,
		ScheduleId: ID,
		RequestId:  uuid.New(),
		Schedule: &schedulepb.Schedule{
			Spec:   convertToPBScheduleSpec(&in.Options.Spec),
			Action: action,
			Policies: &schedulepb.SchedulePolicies{
				OverlapPolicy:  in.Options.Overlap,
				CatchupWindow:  catchupWindow,
				PauseOnFailure: in.Options.PauseOnFailure,
			},
			State: &schedulepb.ScheduleState{
				Notes:            in.Options.Note,
				Paused:           in.Options.Paused,
				LimitedActions:   in.Options.RemainingActions != 0,
				RemainingActions: int64(in.Options.RemainingActions),
			},
		},
		InitialPatch:     initialPatch,
		Identity:         w.client.identity,
		Memo:             memo,
		SearchAttributes: searchAttr,
	}

	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()

	_, err = w.client.workflowService.CreateSchedule(grpcCtx, startRequest)
	if _, ok := err.(*serviceerror.WorkflowExecutionAlreadyStarted); ok {
		return nil, ErrScheduleAlreadyRunning
	}
	if err != nil {
		return nil, err
	}

	return &scheduleHandleImpl{
		ID:     ID,
		client: w.client,
	}, nil
}

func (sc *scheduleClient) Create(ctx context.Context, options ScheduleOptions) (ScheduleHandle, error) {
	if err := sc.workflowClient.ensureInitialized(); err != nil {
		return nil, err
	}

	// Set header before interceptor run
	ctx = contextWithNewHeader(ctx)

	// Run via interceptor
	return sc.workflowClient.interceptor.CreateSchedule(ctx, &ScheduleClientCreateInput{
		Options: &options,
	})
}

func (sc *scheduleClient) GetHandle(ctx context.Context, scheduleID string) ScheduleHandle {
	return &scheduleHandleImpl{
		ID:     scheduleID,
		client: sc.workflowClient,
	}
}

func (sc *scheduleClient) List(ctx context.Context, options ScheduleListOptions) (ScheduleListIterator, error) {
	paginate := func(nextToken []byte) (*workflowservice.ListSchedulesResponse, error) {
		grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
		defer cancel()
		request := &workflowservice.ListSchedulesRequest{
			Namespace:       sc.workflowClient.namespace,
			MaximumPageSize: int32(options.PageSize),
			NextPageToken:   nextToken,
		}

		return sc.workflowClient.workflowService.ListSchedules(grpcCtx, request)
	}

	return &scheduleListIteratorImpl{
		paginate: paginate,
	}, nil
}

func (iter *scheduleListIteratorImpl) HasNext() bool {
	if iter.err == nil {
		if iter.response == nil ||
			(iter.nextScheduleIndex >= len(iter.response.Schedules) && len(iter.response.NextPageToken) > 0) {
			iter.response, iter.err = iter.paginate(iter.response.GetNextPageToken())
			iter.nextScheduleIndex = 0
		}
	}
	return iter.nextScheduleIndex < len(iter.response.GetSchedules()) || iter.err != nil
}

func (iter *scheduleListIteratorImpl) Next() (*ScheduleListEntry, error) {
	if !iter.HasNext() {
		panic("ScheduleListIterator Next() called without checking HasNext()")
	} else if iter.err != nil {
		return nil, iter.err
	}
	schedule := iter.response.Schedules[iter.nextScheduleIndex]
	iter.nextScheduleIndex++
	return convertFromPBScheduleListEntry(schedule), nil
}

func (scheduleHandle *scheduleHandleImpl) GetID() string {
	return scheduleHandle.ID
}

func (scheduleHandle *scheduleHandleImpl) Delete(ctx context.Context) error {
	request := &workflowservice.DeleteScheduleRequest{
		Namespace:  scheduleHandle.client.namespace,
		ScheduleId: scheduleHandle.ID,
		Identity:   scheduleHandle.client.identity,
	}
	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()
	_, err := scheduleHandle.client.workflowService.DeleteSchedule(grpcCtx, request)
	return err
}

func (scheduleHandle *scheduleHandleImpl) Backfill(ctx context.Context, options ScheduleBackfillOptions) error {
	request := &workflowservice.PatchScheduleRequest{
		Namespace:  scheduleHandle.client.namespace,
		ScheduleId: scheduleHandle.ID,
		Patch: &schedulepb.SchedulePatch{
			BackfillRequest: convertToPBBackfillList(options.Backfill),
		},
		Identity:  scheduleHandle.client.identity,
		RequestId: uuid.New(),
	}
	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()
	_, err := scheduleHandle.client.workflowService.PatchSchedule(grpcCtx, request)
	return err
}

func (scheduleHandle *scheduleHandleImpl) Update(ctx context.Context, options ScheduleUpdateOptions) error {
	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()
	ctx = contextWithNewHeader(ctx)

	describeRequest := &workflowservice.DescribeScheduleRequest{
		Namespace:  scheduleHandle.client.namespace,
		ScheduleId: scheduleHandle.ID,
	}
	describeResponse, err := scheduleHandle.client.workflowService.DescribeSchedule(grpcCtx, describeRequest)
	if err != nil {
		return err
	}
	scheduleDescription, err := scheduleDescriptionFromPB(describeResponse)
	if err != nil {
		return err
	}
	newSchedule, err := options.DoUpdate(ScheduleUpdateInput{
		Description: *scheduleDescription,
	})
	if err != nil {
		if errors.Is(err, ErrSkipScheduleUpdate) {
			return nil
		}
		return err
	}
	newSchedulePB, err := convertToPBSchedule(ctx, scheduleHandle.client, newSchedule.Schedule)
	if err != nil {
		return err
	}
	_, err = scheduleHandle.client.workflowService.UpdateSchedule(grpcCtx, &workflowservice.UpdateScheduleRequest{
		Namespace:     scheduleHandle.client.namespace,
		ScheduleId:    scheduleHandle.ID,
		Schedule:      newSchedulePB,
		ConflictToken: nil,
		Identity:      scheduleHandle.client.identity,
		RequestId:     uuid.New(),
	})
	return err
}

func (scheduleHandle *scheduleHandleImpl) Describe(ctx context.Context) (*ScheduleDescription, error) {
	request := &workflowservice.DescribeScheduleRequest{
		Namespace:  scheduleHandle.client.namespace,
		ScheduleId: scheduleHandle.ID,
	}
	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()
	describeResponse, err := scheduleHandle.client.workflowService.DescribeSchedule(grpcCtx, request)
	if err != nil {
		return nil, err
	}
	return scheduleDescriptionFromPB(describeResponse)
}

func (scheduleHandle *scheduleHandleImpl) Trigger(ctx context.Context, options ScheduleTriggerOptions) error {
	request := &workflowservice.PatchScheduleRequest{
		Namespace:  scheduleHandle.client.namespace,
		ScheduleId: scheduleHandle.ID,
		Patch: &schedulepb.SchedulePatch{
			TriggerImmediately: &schedulepb.TriggerImmediatelyRequest{
				OverlapPolicy: options.Overlap,
			},
		},
		Identity:  scheduleHandle.client.identity,
		RequestId: uuid.New(),
	}
	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()
	_, err := scheduleHandle.client.workflowService.PatchSchedule(grpcCtx, request)
	return err
}

func (scheduleHandle *scheduleHandleImpl) Pause(ctx context.Context, options SchedulePauseOptions) error {
	pauseNote := "Paused via Go SDK"
	if options.Note != "" {
		pauseNote = options.Note
	}
	request := &workflowservice.PatchScheduleRequest{
		Namespace:  scheduleHandle.client.namespace,
		ScheduleId: scheduleHandle.ID,
		Patch: &schedulepb.SchedulePatch{
			Pause: pauseNote,
		},
		Identity:  scheduleHandle.client.identity,
		RequestId: uuid.New(),
	}
	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()
	_, err := scheduleHandle.client.workflowService.PatchSchedule(grpcCtx, request)
	return err
}

func (scheduleHandle *scheduleHandleImpl) Unpause(ctx context.Context, options ScheduleUnpauseOptions) error {
	unpauseNote := "Unpaused via Go SDK"
	if options.Note != "" {
		unpauseNote = options.Note
	}
	request := &workflowservice.PatchScheduleRequest{
		Namespace:  scheduleHandle.client.namespace,
		ScheduleId: scheduleHandle.ID,
		Patch: &schedulepb.SchedulePatch{
			Unpause: unpauseNote,
		},
		Identity:  scheduleHandle.client.identity,
		RequestId: uuid.New(),
	}
	grpcCtx, cancel := newGRPCContext(ctx, defaultGrpcRetryParameters(ctx))
	defer cancel()
	_, err := scheduleHandle.client.workflowService.PatchSchedule(grpcCtx, request)
	return err
}

func convertToPBScheduleSpec(scheduleSpec *ScheduleSpec) *schedulepb.ScheduleSpec {
	if scheduleSpec == nil {
		return nil
	}

	calendar := convertToPBScheduleCalendarSpecList(scheduleSpec.Calendars)

	intervals := make([]*schedulepb.IntervalSpec, len(scheduleSpec.Intervals))
	for i, interval := range scheduleSpec.Intervals {
		intervalSpec := interval
		intervals[i] = &schedulepb.IntervalSpec{
			Interval: &intervalSpec.Every,
			Phase:    &intervalSpec.Offset,
		}
	}

	skip := convertToPBScheduleCalendarSpecList(scheduleSpec.Skip)

	var startTime *time.Time
	if !scheduleSpec.StartAt.IsZero() {
		startTime = &scheduleSpec.StartAt
	}

	var endTime *time.Time
	if !scheduleSpec.EndAt.IsZero() {
		endTime = &scheduleSpec.EndAt
	}

	return &schedulepb.ScheduleSpec{
		StructuredCalendar:        calendar,
		Interval:                  intervals,
		CronString:                scheduleSpec.CronExpressions,
		ExcludeStructuredCalendar: skip,
		StartTime:                 startTime,
		EndTime:                   endTime,
		Jitter:                    &scheduleSpec.Jitter,
		// TODO support custom time zone data
		TimezoneName: scheduleSpec.TimeZoneName,
	}
}

func convertFromPBScheduleSpec(scheduleSpec *schedulepb.ScheduleSpec) *ScheduleSpec {
	if scheduleSpec == nil {
		return nil
	}

	calendars := convertFromPBScheduleCalendarSpecList(scheduleSpec.GetStructuredCalendar())

	intervals := make([]ScheduleIntervalSpec, len(scheduleSpec.GetInterval()))
	for i, s := range scheduleSpec.GetInterval() {
		intervals[i] = ScheduleIntervalSpec{
			Every:  common.DurationValue(s.Interval),
			Offset: common.DurationValue(s.Phase),
		}
	}

	skip := convertFromPBScheduleCalendarSpecList(scheduleSpec.GetExcludeStructuredCalendar())

	startAt := time.Time{}
	if scheduleSpec.GetStartTime() != nil {
		startAt = *scheduleSpec.GetStartTime()
	}

	endAt := time.Time{}
	if scheduleSpec.GetEndTime() != nil {
		endAt = *scheduleSpec.GetEndTime()
	}

	return &ScheduleSpec{
		Calendars:    calendars,
		Intervals:    intervals,
		Skip:         skip,
		StartAt:      startAt,
		EndAt:        endAt,
		Jitter:       common.DurationValue(scheduleSpec.GetJitter()),
		TimeZoneName: scheduleSpec.GetTimezoneName(),
	}
}

func scheduleDescriptionFromPB(describeResponse *workflowservice.DescribeScheduleResponse) (*ScheduleDescription, error) {
	if describeResponse == nil {
		return nil, nil
	}

	runningWorkflows := make([]ScheduleWorkflowExecution, len(describeResponse.Info.GetRunningWorkflows()))
	for i, s := range describeResponse.Info.GetRunningWorkflows() {
		runningWorkflows[i] = ScheduleWorkflowExecution{
			WorkflowID:          s.GetWorkflowId(),
			FirstExecutionRunID: s.GetRunId(),
		}
	}

	recentActions := convertFromPBScheduleActionResultList(describeResponse.Info.GetRecentActions())

	nextActionTimes := make([]time.Time, len(describeResponse.Info.GetFutureActionTimes()))
	for i, t := range describeResponse.Info.GetFutureActionTimes() {
		nextActionTimes[i] = common.TimeValue(t)
	}

	actionDescription, err := convertFromPBScheduleAction(describeResponse.Schedule.Action)
	if err != nil {
		return nil, err
	}

	return &ScheduleDescription{
		Schedule: Schedule{
			Action: actionDescription,
			Spec:   convertFromPBScheduleSpec(describeResponse.Schedule.Spec),
			Policy: &SchedulePolicies{
				Overlap:        describeResponse.Schedule.Policies.GetOverlapPolicy(),
				CatchupWindow:  common.DurationValue(describeResponse.Schedule.Policies.GetCatchupWindow()),
				PauseOnFailure: describeResponse.Schedule.Policies.GetPauseOnFailure(),
			},
			State: &ScheduleState{
				Note:             describeResponse.Schedule.State.GetNotes(),
				Paused:           describeResponse.Schedule.State.GetPaused(),
				LimitedActions:   describeResponse.Schedule.State.GetLimitedActions(),
				RemainingActions: int(describeResponse.Schedule.State.GetRemainingActions()),
			},
		},
		Info: ScheduleInfo{
			NumActions:                    int(describeResponse.Info.ActionCount),
			NumActionsMissedCatchupWindow: int(describeResponse.Info.MissedCatchupWindow),
			NumActionsSkippedOverlap:      int(describeResponse.Info.OverlapSkipped),
			RunningWorkflows:              runningWorkflows,
			RecentActions:                 recentActions,
			NextActionTimes:               nextActionTimes,
			CreatedAt:                     common.TimeValue(describeResponse.Info.GetCreateTime()),
			LastUpdateAt:                  common.TimeValue(describeResponse.Info.GetUpdateTime()),
		},
		Memo:             describeResponse.Memo,
		SearchAttributes: describeResponse.SearchAttributes,
	}, nil
}

func convertToPBSchedule(ctx context.Context, client *WorkflowClient, schedule *Schedule) (*schedulepb.Schedule, error) {
	if schedule == nil {
		return nil, nil
	}
	action, err := convertToPBScheduleAction(ctx, client, schedule.Action)
	if err != nil {
		return nil, err
	}
	return &schedulepb.Schedule{
		Spec:   convertToPBScheduleSpec(schedule.Spec),
		Action: action,
		Policies: &schedulepb.SchedulePolicies{
			OverlapPolicy:  schedule.Policy.Overlap,
			CatchupWindow:  &schedule.Policy.CatchupWindow,
			PauseOnFailure: schedule.Policy.PauseOnFailure,
		},
		State: &schedulepb.ScheduleState{
			Notes:            schedule.State.Note,
			Paused:           schedule.State.Paused,
			LimitedActions:   schedule.State.LimitedActions,
			RemainingActions: int64(schedule.State.RemainingActions),
		},
	}, nil
}

func convertFromPBScheduleListEntry(schedule *schedulepb.ScheduleListEntry) *ScheduleListEntry {
	scheduleInfo := schedule.GetInfo()

	recentActions := convertFromPBScheduleActionResultList(scheduleInfo.GetRecentActions())

	nextActionTimes := make([]time.Time, len(schedule.Info.GetFutureActionTimes()))
	for i, t := range schedule.Info.GetFutureActionTimes() {
		nextActionTimes[i] = common.TimeValue(t)
	}

	return &ScheduleListEntry{
		ID:     schedule.ScheduleId,
		Spec:   convertFromPBScheduleSpec(scheduleInfo.GetSpec()),
		Note:   scheduleInfo.GetNotes(),
		Paused: scheduleInfo.GetPaused(),
		WorkflowType: WorkflowType{
			Name: scheduleInfo.GetWorkflowType().GetName(),
		},
		RecentActions:    recentActions,
		NextActionTimes:  nextActionTimes,
		Memo:             schedule.Memo,
		SearchAttributes: schedule.SearchAttributes,
	}
}

func convertToPBScheduleAction(ctx context.Context, client *WorkflowClient, scheduleAction ScheduleAction) (*schedulepb.ScheduleAction, error) {
	switch action := scheduleAction.(type) {
	case *ScheduleWorkflowAction:
		// Set header before interceptor run
		dataConverter := WithContext(ctx, client.dataConverter)

		// Default workflow ID
		if action.ID == "" {
			action.ID = uuid.New()
		}

		// Validate function and get name
		if err := validateFunctionArgs(action.Workflow, action.Args, true); err != nil {
			return nil, err
		}
		workflowType, err := getWorkflowFunctionName(client.registry, action.Workflow)
		if err != nil {
			return nil, err
		}
		// Encode workflow inputs that may already be encoded
		input, err := encodeScheduleWorklowArgs(dataConverter, action.Args)
		if err != nil {
			return nil, err
		}
		// Encode workflow memos that may already be encoded
		memo, err := encodeScheduleWorkflowMemo(dataConverter, action.Memo)
		if err != nil {
			return nil, err
		}

		searchAttr, err := serializeSearchAttributes(action.SearchAttributes)
		if err != nil {
			return nil, err
		}

		// get workflow headers from the context
		header, err := headerPropagated(ctx, client.contextPropagators)
		if err != nil {
			return nil, err
		}

		return &schedulepb.ScheduleAction{
			Action: &schedulepb.ScheduleAction_StartWorkflow{
				StartWorkflow: &workflowpb.NewWorkflowExecutionInfo{
					WorkflowId:               action.ID,
					WorkflowType:             &commonpb.WorkflowType{Name: workflowType},
					TaskQueue:                &taskqueuepb.TaskQueue{Name: action.TaskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
					Input:                    input,
					WorkflowExecutionTimeout: &action.WorkflowExecutionTimeout,
					WorkflowRunTimeout:       &action.WorkflowRunTimeout,
					WorkflowTaskTimeout:      &action.WorkflowTaskTimeout,
					RetryPolicy:              convertToPBRetryPolicy(action.RetryPolicy),
					Memo:                     memo,
					SearchAttributes:         searchAttr,
					Header:                   header,
				},
			},
		}, nil
	default:
		// TODO maybe just panic instead?
		return nil, fmt.Errorf("could not parse ScheduleAction")
	}
}

func convertFromPBScheduleAction(action *schedulepb.ScheduleAction) (ScheduleAction, error) {
	switch action := action.Action.(type) {
	case *schedulepb.ScheduleAction_StartWorkflow:
		workflow := action.StartWorkflow

		args := make([]interface{}, len(workflow.GetInput().GetPayloads()))
		for i, p := range workflow.GetInput().GetPayloads() {
			args[i] = p
		}

		memos := make(map[string]interface{})
		for key, element := range workflow.GetMemo().GetFields() {
			memos[key] = element
		}

		searchAttributes := make(map[string]interface{})
		for key, element := range workflow.GetSearchAttributes().GetIndexedFields() {
			searchAttributes[key] = element
		}

		return &ScheduleWorkflowAction{
			ID:                       workflow.GetWorkflowId(),
			Workflow:                 workflow.WorkflowType.GetName(),
			Args:                     args,
			TaskQueue:                workflow.TaskQueue.GetName(),
			WorkflowExecutionTimeout: common.DurationValue(workflow.GetWorkflowExecutionTimeout()),
			WorkflowRunTimeout:       common.DurationValue(workflow.GetWorkflowRunTimeout()),
			WorkflowTaskTimeout:      common.DurationValue(workflow.GetWorkflowTaskTimeout()),
			RetryPolicy:              convertFromPBRetryPolicy(workflow.RetryPolicy),
			Memo:                     memos,
			SearchAttributes:         searchAttributes,
		}, nil
	default:
		// TODO maybe just panic instead?
		return nil, fmt.Errorf("could not parse ScheduleAction")
	}
}

func convertToPBBackfillList(backfillRequests []ScheduleBackfill) []*schedulepb.BackfillRequest {
	backfillRequestsPB := make([]*schedulepb.BackfillRequest, len(backfillRequests))
	for i, b := range backfillRequests {
		backfill := b
		backfillRequestsPB[i] = &schedulepb.BackfillRequest{
			StartTime:     &backfill.Start,
			EndTime:       &backfill.End,
			OverlapPolicy: backfill.Overlap,
		}
	}
	return backfillRequestsPB
}

func convertToPBRangeList(scheduleRange []ScheduleRange) []*schedulepb.Range {
	rangesPB := make([]*schedulepb.Range, len(scheduleRange))
	for i, r := range scheduleRange {
		rangesPB[i] = &schedulepb.Range{
			Start: int32(r.Start),
			End:   int32(r.End),
			Step:  int32(r.Step),
		}
	}
	return rangesPB
}

func convertFromPBRangeList(scheduleRangePB []*schedulepb.Range) []ScheduleRange {
	scheduleRange := make([]ScheduleRange, len(scheduleRangePB))
	for i, r := range scheduleRangePB {
		if r == nil {
			continue
		}
		scheduleRange[i] = ScheduleRange{
			Start: int(r.Start),
			End:   int(r.End),
			Step:  int(r.Step),
		}
	}
	return scheduleRange
}

func convertFromPBScheduleCalendarSpecList(calendarSpecPB []*schedulepb.StructuredCalendarSpec) []ScheduleCalendarSpec {
	calendarSpec := make([]ScheduleCalendarSpec, len(calendarSpecPB))
	for i, e := range calendarSpecPB {
		calendarSpec[i] = ScheduleCalendarSpec{
			Second:     convertFromPBRangeList(e.Second),
			Minute:     convertFromPBRangeList(e.Minute),
			Hour:       convertFromPBRangeList(e.Hour),
			DayOfMonth: convertFromPBRangeList(e.DayOfMonth),
			Month:      convertFromPBRangeList(e.Month),
			Year:       convertFromPBRangeList(e.Year),
			DayOfWeek:  convertFromPBRangeList(e.DayOfWeek),
			Comment:    e.Comment,
		}
	}
	return calendarSpec
}

func applyScheduleCalendarSpecDefault(calendarSpec *ScheduleCalendarSpec) {
	if calendarSpec.Second == nil {
		calendarSpec.Second = []ScheduleRange{{Start: 0}}
	}

	if calendarSpec.Minute == nil {
		calendarSpec.Minute = []ScheduleRange{{Start: 0}}
	}

	if calendarSpec.Hour == nil {
		calendarSpec.Hour = []ScheduleRange{{Start: 0}}
	}

	if calendarSpec.DayOfMonth == nil {
		calendarSpec.DayOfMonth = []ScheduleRange{{Start: 1, End: 31}}
	}

	if calendarSpec.Month == nil {
		calendarSpec.Month = []ScheduleRange{{Start: 1, End: 12}}
	}

	if calendarSpec.DayOfWeek == nil {
		calendarSpec.DayOfWeek = []ScheduleRange{{Start: 0, End: 6}}
	}
}

func convertToPBScheduleCalendarSpecList(calendarSpec []ScheduleCalendarSpec) []*schedulepb.StructuredCalendarSpec {
	calendarSpecPB := make([]*schedulepb.StructuredCalendarSpec, len(calendarSpec))
	for i, e := range calendarSpec {
		applyScheduleCalendarSpecDefault(&e)

		calendarSpecPB[i] = &schedulepb.StructuredCalendarSpec{
			Second:     convertToPBRangeList(e.Second),
			Minute:     convertToPBRangeList(e.Minute),
			Hour:       convertToPBRangeList(e.Hour),
			DayOfMonth: convertToPBRangeList(e.DayOfMonth),
			Month:      convertToPBRangeList(e.Month),
			Year:       convertToPBRangeList(e.Year),
			DayOfWeek:  convertToPBRangeList(e.DayOfWeek),
			Comment:    e.Comment,
		}
	}
	return calendarSpecPB
}

func convertFromPBScheduleActionResultList(aa []*schedulepb.ScheduleActionResult) []ScheduleActionResult {
	recentActions := make([]ScheduleActionResult, len(aa))
	for i, a := range aa {
		var workflowExecution *ScheduleWorkflowExecution
		if a.GetStartWorkflowResult() != nil {
			workflowExecution = &ScheduleWorkflowExecution{
				WorkflowID:          a.GetStartWorkflowResult().GetWorkflowId(),
				FirstExecutionRunID: a.GetStartWorkflowResult().GetRunId(),
			}
		}
		recentActions[i] = ScheduleActionResult{
			ScheduleTime:        common.TimeValue(a.GetScheduleTime()),
			ActualTime:          common.TimeValue(a.GetActualTime()),
			StartWorkflowResult: workflowExecution,
		}
	}
	return recentActions
}

func encodeScheduleWorklowArgs(dc converter.DataConverter, args []interface{}) (*commonpb.Payloads, error) {
	payloads := make([]*commonpb.Payload, len(args))
	for i, arg := range args {
		// arg is already encoded
		if enc, ok := arg.(*commonpb.Payload); ok {
			payloads[i] = enc
		} else {
			payload, err := dc.ToPayload(arg)
			if err != nil {
				return nil, err
			}
			payloads[i] = payload
		}
	}
	return &commonpb.Payloads{
		Payloads: payloads,
	}, nil
}

func encodeScheduleWorkflowMemo(dc converter.DataConverter, input map[string]interface{}) (*commonpb.Memo, error) {
	if input == nil {
		return nil, nil
	}

	memo := make(map[string]*commonpb.Payload)
	for k, v := range input {
		if enc, ok := v.(*commonpb.Payload); ok {
			memo[k] = enc
		} else {
			memoBytes, err := converter.GetDefaultDataConverter().ToPayload(v)
			if err != nil {
				return nil, fmt.Errorf("encode workflow memo error: %v", err.Error())
			}
			memo[k] = memoBytes
		}
	}
	return &commonpb.Memo{Fields: memo}, nil
}
