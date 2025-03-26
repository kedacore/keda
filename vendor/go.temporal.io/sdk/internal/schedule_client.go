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
	"time"

	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
)

type (
	// ScheduleRange represents a set of integer values, used to match fields of a calendar
	// time in StructuredCalendarSpec. If end < start, then end is interpreted as
	// equal to start. This means you can use a Range with start set to a value, and
	// end and step unset (defaulting to 0) to represent a single value.
	//
	// Exposed as: [go.temporal.io/sdk/client.ScheduleRange]
	ScheduleRange struct {
		// Start of the range (inclusive)
		Start int

		// End of the range (inclusive)
		// Optional: defaulted to Start
		End int

		// Step to be take between each value
		// Optional: defaulted to 1
		Step int
	}

	// ScheduleCalendarSpec is an event specification relative to the calendar, similar to a traditional cron specification.
	// A timestamp matches if at least one range of each field matches the
	// corresponding fields of the timestamp, except for year: if year is missing,
	// that means all years match. For all fields besides year, at least one Range must be present to match anything.
	//
	// Exposed as: [go.temporal.io/sdk/client.ScheduleCalendarSpec]
	ScheduleCalendarSpec struct {
		// Second range to match (0-59).
		//
		// default: matches 0
		Second []ScheduleRange

		// Minute range to match (0-59).
		//
		// default: matches 0
		Minute []ScheduleRange

		// Hour range to match (0-23).
		//
		// default: matches 0
		Hour []ScheduleRange

		// DayOfMonth range to match (1-31)
		//
		// default: matches all days
		DayOfMonth []ScheduleRange

		// Month range to match (1-12)
		//
		// default: matches all months
		Month []ScheduleRange

		// Year range to match.
		//
		// default: empty that matches all years
		Year []ScheduleRange

		// DayOfWeek range to match (0-6; 0 is Sunday)
		//
		// default: matches all days of the week
		DayOfWeek []ScheduleRange

		// Comment - Description of the intention of this schedule.
		Comment string
	}

	// ScheduleBackfill desribes a time periods and policy and takes Actions as if that time passed by right now, all at once.
	//
	// Exposed as: [go.temporal.io/sdk/client.ScheduleBackfill]
	ScheduleBackfill struct {
		// Start - start of the range to evaluate schedule in.
		Start time.Time

		// End - end of the range to evaluate schedule in.
		End time.Time

		// Overlap - Override the Overlap Policy for this request.
		Overlap enumspb.ScheduleOverlapPolicy
	}

	// ScheduleIntervalSpec - matches times that can be expressed as:
	//
	// 	Epoch + (n * every) + offset
	//
	// 	where n is all integers ≥ 0.
	//
	// For example, an `every` of 1 hour with `offset` of zero would match every hour, on the hour. The same `every` but an `offset`
	// of 19 minutes would match every `xx:19:00`. An `every` of 28 days with `offset` zero would match `2022-02-17T00:00:00Z`
	// (among other times). The same `every` with `offset` of 3 days, 5 hours, and 23 minutes would match `2022-02-20T05:23:00Z`
	// instead.
	//
	// Exposed as: [go.temporal.io/sdk/client.ScheduleIntervalSpec]
	ScheduleIntervalSpec struct {
		// Every - describes the period to repeat the interval.
		Every time.Duration

		// Offset - is a fixed offset added to the intervals period.
		// Optional: Defaulted to 0
		Offset time.Duration
	}

	// ScheduleSpec is a complete description of a set of absolute times (possibly infinite) that a action should occur at.
	// The times are the union of Calendars, Intervals, and CronExpressions, minus the Skip times. These times
	// never change, except that the definition of a time zone can change over time (most commonly, when daylight saving
	// time policy changes for an area). To create a totally self-contained ScheduleSpec, use UTC.
	//
	// Exposed as: [go.temporal.io/sdk/client.ScheduleSpec]
	ScheduleSpec struct {
		// Calendars - Calendar-based specifications of times
		Calendars []ScheduleCalendarSpec

		// Intervals - Interval-based specifications of times.
		Intervals []ScheduleIntervalSpec

		// CronExpressions -  CronExpressions-based specifications of times. CronExpressions is provided for easy migration from legacy Cron Workflows. For new
		// use cases, we recommend using ScheduleSpec.Calendars or ScheduleSpec.Intervals for readability and maintainability. Once a schedule is created all
		// expressions in CronExpressions will be translated to ScheduleSpec.Calendars on the server.
		//
		// For example, `0 12 * * MON-WED,FRI` is every M/Tu/W/F at noon, and is equivalent to this ScheduleCalendarSpec:
		//
		// client.ScheduleCalendarSpec{
		// 		Second: []ScheduleRange{{}},
		// 		Minute: []ScheduleRanges{{}},
		// 		Hour: []ScheduleRange{{
		// 			Start: 12,
		// 		}},
		// 		DayOfMonth: []ScheduleRange{
		// 			{
		// 				Start: 1,
		// 				End:   31,
		// 			},
		// 		},
		// 		Month: []ScheduleRange{
		// 			{
		// 				Start: 1,
		// 				End:   12,
		// 			},
		// 		},
		// 		DayOfWeek: []ScheduleRange{
		// 			{
		// 				Start: 1,
		//				End: 3,
		// 			},
		// 			{
		// 				Start: 5,
		// 			},
		// 		},
		// 	}
		//
		//
		// The string can have 5, 6, or 7 fields, separated by spaces, and they are interpreted in the
		// same way as a ScheduleCalendarSpec:
		//	- 5 fields:         Minute, Hour, DayOfMonth, Month, DayOfWeek
		//	- 6 fields:         Minute, Hour, DayOfMonth, Month, DayOfWeek, Year
		//	- 7 fields: Second, Minute, Hour, DayOfMonth, Month, DayOfWeek, Year
		//
		// Notes:
		//	- If Year is not given, it defaults to *.
		//	- If Second is not given, it defaults to 0.
		//	- Shorthands @yearly, @monthly, @weekly, @daily, and @hourly are also
		//		accepted instead of the 5-7 time fields.
		//	- @every <interval>[/<phase>] is accepted and gets compiled into an
		//		IntervalSpec instead. <interval> and <phase> should be a decimal integer
		//		with a unit suffix s, m, h, or d.
		//	- Optionally, the string can be preceded by CRON_TZ=<time zone name> or
		//		TZ=<time zone name>, which will get copied to ScheduleSpec.TimeZoneName. (In which case the ScheduleSpec.TimeZone field should be left empty.)
		//	- Optionally, "#" followed by a comment can appear at the end of the string.
		//	- Note that the special case that some cron implementations have for
		//		treating DayOfMonth and DayOfWeek as "or" instead of "and" when both
		//		are set is not implemented.
		CronExpressions []string

		// Skip - Any matching times will be skipped.
		//
		// All fields of the ScheduleCalendarSpec—including seconds—must match a time for the time to be skipped.
		Skip []ScheduleCalendarSpec

		// StartAt - Any times before `startAt` will be skipped. Together, `startAt` and `endAt` make an inclusive interval.
		// Optional: Defaulted to the beginning of time
		StartAt time.Time

		// EndAt - Any times after `endAt` will be skipped.
		// Optional: Defaulted to the end of time
		EndAt time.Time

		// Jitter - All times will be incremented by a random value from 0 to this amount of jitter, capped
		// by the time until the next schedule.
		// Optional: Defaulted to 0
		Jitter time.Duration

		// TimeZoneName - IANA time zone name, for example `US/Pacific`.
		//
		// The definition will be loaded by Temporal Server from the environment it runs in.
		//
		// Calendar spec matching is based on literal matching of the clock time
		// with no special handling of DST: if you write a calendar spec that fires
		// at 2:30am and specify a time zone that follows DST, that action will not
		// be triggered on the day that has no 2:30am. Similarly, an action that
		// fires at 1:30am will be triggered twice on the day that has two 1:30s.
		//
		// Note: No actions are taken on leap-seconds (e.g. 23:59:60 UTC).
		// Optional: Defaulted to UTC
		TimeZoneName string
	}

	// ScheduleAction represents an action a schedule can take.
	ScheduleAction interface {
		isScheduleAction()
	}

	// ScheduleWorkflowAction implements ScheduleAction to launch a workflow.
	//
	// Exposed as: [go.temporal.io/sdk/client.ScheduleWorkflowAction]
	ScheduleWorkflowAction struct {
		// ID - The business identifier of the workflow execution.
		// The workflow ID of the started workflow may not match this exactly,
		// it may have a timestamp appended for uniqueness.
		// Optional: defaulted to a uuid.
		ID string

		// Workflow - What workflow to run.
		// Workflow can either be the function or workflow type name as a string.
		// On ScheduleHandle.Describe() or ScheduleHandle.Update() it will be the workflow type name.
		Workflow interface{}

		// Args - Arguments to pass to the workflow.
		// On ScheduleHandle.Describe() or ScheduleHandle.Update() Args will be returned as *commonpb.Payload.
		Args []interface{}

		// TaskQueue - The workflow tasks of the workflow are scheduled on the queue with this name.
		// This is also the name of the activity task queue on which activities are scheduled.
		TaskQueue string

		// WorkflowExecutionTimeout - The timeout for duration of workflow execution.
		WorkflowExecutionTimeout time.Duration

		// WorkflowRunTimeout - The timeout for duration of a single workflow run.
		WorkflowRunTimeout time.Duration

		// WorkflowTaskTimeout - The timeout for processing workflow task from the time the worker
		// pulled this task.
		WorkflowTaskTimeout time.Duration

		// RetryPolicy - Retry policy for workflow. If a retry policy is specified, in case of workflow failure
		// server will start new workflow execution if needed based on the retry policy.
		RetryPolicy *RetryPolicy

		// Memo - Optional non-indexed info that will be shown in list workflow.
		// On ScheduleHandle.Describe() or ScheduleHandle.Update() Memo will be returned as *commonpb.Payload.
		Memo map[string]interface{}

		// TypedSearchAttributes - Optional indexed info that can be used in query of List/Scan/Count workflow APIs. The key
		// and value type must be registered on Temporal server side. For supported operations on different server versions
		// see [Visibility].
		//
		// [Visibility]: https://docs.temporal.io/visibility
		TypedSearchAttributes SearchAttributes

		// UntypedSearchAttributes - These are set upon update for older schedules that did not have typed attributes. This
		// should never be used for create.
		//
		// Deprecated - This is only for update of older search attributes. This may be removed in a future version.
		UntypedSearchAttributes map[string]*commonpb.Payload

		// VersioningOverride - Sets the versioning configuration of a specific workflow execution, ignoring current
		// server or worker default policies. This enables running canary tests without affecting existing workflows.
		// To unset the override after the workflow is running, use [Client.UpdateWorkflowExecutionOptions].
		// Optional: defaults to no override.
		//
		// NOTE: Experimental
		VersioningOverride VersioningOverride

		// TODO(cretz): Expose once https://github.com/temporalio/temporal/issues/6412 is fixed
		staticSummary string
		staticDetails string
	}

	// ScheduleOptions configure the parameters for creating a schedule.
	//
	// Exposed as: [go.temporal.io/sdk/client.ScheduleOptions]
	ScheduleOptions struct {
		// ID - The business identifier of the schedule.
		ID string

		// Schedule - Describes when Actions should be taken.
		Spec ScheduleSpec

		// Action - Which Action to take.
		Action ScheduleAction

		// Overlap - Controls what happens when an Action would be started by a Schedule at the same time that an older Action is still
		// running. This can be changed after a Schedule has taken some Actions, and some changes might produce
		// unintuitive results. In general, the later policy overrides the earlier policy.
		//
		// Optional: defaulted to SCHEDULE_OVERLAP_POLICY_SKIP
		Overlap enumspb.ScheduleOverlapPolicy

		// CatchupWindow - The Temporal Server might be down or unavailable at the time when a Schedule should take an Action.
		// When the Server comes back up, CatchupWindow controls which missed Actions should be taken at that point. The default is one
		// minute, which means that the Schedule attempts to take any Actions that wouldn't be more than one minute late. It
		// takes those Actions according to the Overlap. An outage that lasts longer than the Catchup
		// Window could lead to missed Actions.
		// Optional: defaulted to 1 minute
		CatchupWindow time.Duration

		// PauseOnFailure - When an Action times out or reaches the end of its Retry Policy the Schedule will pause.
		//
		// With SCHEDULE_OVERLAP_POLICY_ALLOW_ALL, this pause might not apply to the next Action, because the next Action
		// might have already started previous to the failed one finishing. Pausing applies only to Actions that are scheduled
		// to start after the failed one finishes.
		// Optional: defaulted to false
		PauseOnFailure bool

		// Note - Informative human-readable message with contextual notes, e.g. the reason
		// a Schedule is paused. The system may overwrite this message on certain
		// conditions, e.g. when pause-on-failure happens.
		Note string

		// Paused - Start in paused state.
		// Optional: defaulted to false
		Paused bool

		// RemainingActions - limit the number of Actions to take.
		//
		// This number is decremented after each Action is taken, and Actions are not
		// taken when the number is `0` (unless ScheduleHandle.Trigger is called).
		//
		// Optional: defaulted to zero
		RemainingActions int

		// TriggerImmediately - Trigger one Action immediately on creating the schedule.
		// Optional: defaulted to false
		TriggerImmediately bool

		// ScheduleBackfill - Runs though the specified time periods and takes Actions as if that time passed by right now, all at once. The
		// overlap policy can be overridden for the scope of the ScheduleBackfill.
		ScheduleBackfill []ScheduleBackfill

		// Memo - Optional non-indexed info that will be shown in list schedules.
		Memo map[string]interface{}

		// SearchAttributes - Optional indexed info that can be used in query of List schedules APIs. The key and value type must be registered on Temporal server side.
		// Use GetSearchAttributes API to get valid key and corresponding value type.
		// For supported operations on different server versions see [Visibility].
		//
		// Deprecated: use TypedSearchAttributes instead.
		//
		// [Visibility]: https://docs.temporal.io/visibility
		SearchAttributes map[string]interface{}

		// TypedSearchAttributes - Specifies Search Attributes that will be attached to the Workflow. Search Attributes are
		// additional indexed information attributed to workflow and used for search and visibility. The search attributes
		// can be used in query of List/Scan/Count workflow APIs. The key and its value type must be registered on Temporal
		// server side. For supported operations on different server versions see [Visibility].
		//
		// Optional: default to none.
		//
		// [Visibility]: https://docs.temporal.io/visibility
		TypedSearchAttributes SearchAttributes
	}

	// ScheduleWorkflowExecution contains details on a workflows execution stared by a schedule.
	//
	// Exposed as: [go.temporal.io/sdk/client.ScheduleWorkflowExecution]
	ScheduleWorkflowExecution struct {
		// WorkflowID - The ID of the workflow execution
		WorkflowID string

		// FirstExecutionRunID - The Run Id of the original execution that was started by the Schedule. If the Workflow retried, did
		// Continue-As-New, or was Reset, the following runs would have different Run Ids.
		FirstExecutionRunID string
	}

	// ScheduleInfo describes other information about a schedule.
	//
	// Exposed as: [go.temporal.io/sdk/client.ScheduleInfo]
	ScheduleInfo struct {
		// NumActions - Number of actions taken by this schedule.
		NumActions int

		// NumActionsMissedCatchupWindow - Number of times a scheduled Action was skipped due to missing the catchup window.
		NumActionsMissedCatchupWindow int

		// NumActionsSkippedOverlap - Number of Actions skipped due to overlap.
		NumActionsSkippedOverlap int

		// RunningWorkflows - Currently-running workflows started by this schedule. (There might be
		// more than one if the overlap policy allows overlaps.)
		RunningWorkflows []ScheduleWorkflowExecution

		// RecentActions- Most recent 10 Actions started (including manual triggers).
		//
		// Sorted from older start time to newer.
		RecentActions []ScheduleActionResult

		// NextActionTimes - Next 10 scheduled Action times.
		NextActionTimes []time.Time

		// CreatedAt -  When the schedule was created
		CreatedAt time.Time

		// LastUpdateAt - When a schedule was last updated
		LastUpdateAt time.Time
	}

	// ScheduleDescription describes the current Schedule details from ScheduleHandle.Describe.
	//
	// Exposed as: [go.temporal.io/sdk/client.ScheduleDescription]
	ScheduleDescription struct {
		// Schedule - Describes the modifiable fields of a schedule.
		Schedule Schedule

		// Info - Extra information about the schedule.
		Info ScheduleInfo

		// Memo - Non-indexed user supplied information.
		Memo *commonpb.Memo

		// SearchAttributes - Additional indexed information used for search and visibility. The key and its value type
		// are registered on Temporal server side.
		// For supported operations on different server versions see [Visibility].
		//
		// [Visibility]: https://docs.temporal.io/visibility
		SearchAttributes *commonpb.SearchAttributes

		// TypedSearchAttributes - Additional indexed information used for search and visibility. The key and its value
		// type are registered on Temporal server side.
		// For supported operations on different server versions see [Visibility].
		//
		// [Visibility]: https://docs.temporal.io/visibility
		TypedSearchAttributes SearchAttributes
	}

	// SchedulePolicies describes the current polcies of a schedule.
	//
	// Exposed as: [go.temporal.io/sdk/client.SchedulePolicies]
	SchedulePolicies struct {
		// Overlap - Controls what happens when an Action would be started by a Schedule at the same time that an older Action is still
		// running.
		Overlap enumspb.ScheduleOverlapPolicy

		// CatchupWindow - The Temporal Server might be down or unavailable at the time when a Schedule should take an Action. When the Server
		// comes back up, CatchupWindow controls which missed Actions should be taken at that point.
		CatchupWindow time.Duration

		// PauseOnFailure - When an Action times out or reaches the end of its Retry Policy.
		PauseOnFailure bool
	}

	// ScheduleState describes the current state of a schedule.
	//
	// Exposed as: [go.temporal.io/sdk/client.ScheduleState]
	ScheduleState struct {
		// Note - Informative human-readable message with contextual notes, e.g. the reason
		// a Schedule is paused. The system may overwrite this message on certain
		// conditions, e.g. when pause-on-failure happens.
		Note string

		// Paused - True if the schedule is paused.
		Paused bool

		// LimitedActions - While true RemainingActions will be decremented for each action taken.
		// Skipped actions (due to overlap policy) do not count against remaining actions.
		LimitedActions bool

		// RemainingActions - The Actions remaining in this Schedule. Once this number hits 0, no further Actions are taken.
		// manual actions through backfill or ScheduleHandle.Trigger still run.
		RemainingActions int
	}

	// Schedule describes a created schedule.
	//
	// Exposed as: [go.temporal.io/sdk/client.Schedule]
	Schedule struct {
		// Action - Which Action to take
		Action ScheduleAction

		// Schedule - Describes when Actions should be taken.
		Spec *ScheduleSpec

		// SchedulePolicies - this schedules policies
		Policy *SchedulePolicies

		// State - this schedules state
		State *ScheduleState
	}

	// ScheduleUpdate describes the desired new schedule from ScheduleHandle.Update.
	//
	// Exposed as: [go.temporal.io/sdk/client.ScheduleUpdate]
	ScheduleUpdate struct {
		// Schedule - New schedule to replace the existing schedule with
		Schedule *Schedule

		// TypedSearchAttributes - Optional indexed info that can be used for querying via the List schedules APIs.
		// The key and value type must be registered on Temporal server side.
		//
		// nil: leave any pre-existing assigned search attributes intact
		// empty: remove any and all pre-existing assigned search attributes
		// attributes present: replace any and all pre-existing assigned search attributes with the defined search
		//                     attributes, i.e. upsert
		TypedSearchAttributes *SearchAttributes
	}

	// ScheduleUpdateInput describes the current state of the schedule to be updated.
	//
	// Exposed as: [go.temporal.io/sdk/client.ScheduleUpdateInput]
	ScheduleUpdateInput struct {
		// Description - current description of the schedule
		Description ScheduleDescription
	}

	// ScheduleUpdateOptions configure the parameters for updating a schedule.
	//
	// Exposed as: [go.temporal.io/sdk/client.ScheduleUpdateOptions]
	ScheduleUpdateOptions struct {
		// DoUpdate - Takes a description of the schedule and returns the new desired schedule.
		// If update returns ErrSkipScheduleUpdate response and no update will occur.
		// Any other error will be passed through.
		DoUpdate func(ScheduleUpdateInput) (*ScheduleUpdate, error)
	}

	// ScheduleTriggerOptions configure the parameters for triggering a schedule.
	//
	// Exposed as: [go.temporal.io/sdk/client.ScheduleTriggerOptions]
	ScheduleTriggerOptions struct {
		// Overlap - If specified, policy to override the schedules default overlap policy.
		Overlap enumspb.ScheduleOverlapPolicy
	}

	// SchedulePauseOptions configure the parameters for pausing a schedule.
	//
	// Exposed as: [go.temporal.io/sdk/client.SchedulePauseOptions]
	SchedulePauseOptions struct {
		// Note - Informative human-readable message with contextual notes.
		// Optional: defaulted to 'Paused via Go SDK'
		Note string
	}

	// ScheduleUnpauseOptions configure the parameters for unpausing a schedule.
	//
	// Exposed as: [go.temporal.io/sdk/client.ScheduleUnpauseOptions]
	ScheduleUnpauseOptions struct {
		// Note - Informative human-readable message with contextual notes.
		// Optional: defaulted to 'Unpaused via Go SDK'
		Note string
	}

	// ScheduleBackfillOptions configure the parameters for backfilling a schedule.
	//
	// Exposed as: [go.temporal.io/sdk/client.ScheduleBackfillOptions]
	ScheduleBackfillOptions struct {
		// Backfill  - Time periods to backfill the schedule.
		Backfill []ScheduleBackfill
	}

	// ScheduleHandle represents a created schedule.
	ScheduleHandle interface {
		// GetID returns the schedule ID associated with this handle.
		GetID() string

		// Delete the Schedule
		Delete(ctx context.Context) error

		// Backfill the schedule by going though the specified time periods and taking Actions as if that time passed by right now, all at once.
		Backfill(ctx context.Context, options ScheduleBackfillOptions) error

		// Update the Schedule.
		//
		// NOTE: If two Update calls are made in parallel to the same Schedule there is the potential
		// for a race condition.
		Update(ctx context.Context, options ScheduleUpdateOptions) error

		// Describe fetches the Schedule's description from the Server
		Describe(ctx context.Context) (*ScheduleDescription, error)

		// Trigger an Action to be taken immediately. Will override the schedules default policy
		// with the one specified here. If overlap is SCHEDULE_OVERLAP_POLICY_UNSPECIFIED the schedule
		// policy will be used.
		Trigger(ctx context.Context, options ScheduleTriggerOptions) error

		// Pause the Schedule will also overwrite the Schedules current note with the new note.
		Pause(ctx context.Context, options SchedulePauseOptions) error

		// Unpause the Schedule will also overwrite the Schedules current note with the new note.
		Unpause(ctx context.Context, options ScheduleUnpauseOptions) error
	}

	// ScheduleActionResult describes when a schedule action took place
	//
	// Exposed as: [go.temporal.io/sdk/client.ScheduleActionResult]
	ScheduleActionResult struct {
		// ScheduleTime - Time that the Action was scheduled for, including jitter.
		ScheduleTime time.Time

		// ActualTime - Time that the Action was actually taken.
		ActualTime time.Time

		// StartWorkflowResult - If action was ScheduleWorkflowAction, returns the
		// ID of the workflow.
		StartWorkflowResult *ScheduleWorkflowExecution
	}

	// ScheduleListEntry
	//
	// Exposed as: [go.temporal.io/sdk/client.ScheduleListEntry]
	ScheduleListEntry struct {
		// ID - The business identifier of the schedule.
		ID string

		// Spec - Describes when Actions should be taken.
		Spec *ScheduleSpec

		// Note - Informative human-readable message with contextual notes, e.g. the reason
		// a Schedule is paused. The system may overwrite this message on certain
		// conditions, e.g. when pause-on-failure happens.
		Note string

		// Paused - True if the schedule is paused.
		Paused bool

		// WorkflowType - If the schedule action is a Wokrflow then
		// describes what workflow is run.
		WorkflowType WorkflowType

		// RecentActions- Most recent 5 Actions started (including manual triggers).
		//
		// Sorted from older start time to newer.
		RecentActions []ScheduleActionResult

		// NextActionTimes - Next 5 scheduled Action times.
		NextActionTimes []time.Time

		// Memo - Non-indexed user supplied information.
		Memo *commonpb.Memo

		// SearchAttributes - Indexed info that can be used in query of List schedules APIs. The key and value type must be registered on Temporal server side.
		// Use GetSearchAttributes API to get valid key and corresponding value type.
		// For supported operations on different server versions see [Visibility].
		//
		// [Visibility]: https://docs.temporal.io/visibility
		SearchAttributes *commonpb.SearchAttributes
	}

	// ScheduleListOptions are the parameters for configuring listing schedules
	//
	// Exposed as: [go.temporal.io/sdk/client.ScheduleListOptions]
	ScheduleListOptions struct {
		// PageSize - How many results to fetch from the Server at a time.
		// Optional: defaulted to 1000
		PageSize int

		// Query - Filter results using a SQL-like query.
		// Optional
		Query string
	}

	// ScheduleListIterator represents the interface for
	// schedule iterator
	ScheduleListIterator interface {
		// HasNext return whether this iterator has next value
		HasNext() bool

		// Next returns the next schedule and error
		Next() (*ScheduleListEntry, error)
	}

	// Client for creating Schedules and creating Schedule handles
	ScheduleClient interface {
		// Create a new Schedule.
		Create(ctx context.Context, options ScheduleOptions) (ScheduleHandle, error)

		// List returns an iterator to list all schedules
		//
		// Note: When using advanced visibility List is eventually consistent.
		List(ctx context.Context, options ScheduleListOptions) (ScheduleListIterator, error)

		// GetHandle returns a handle to a Schedule
		//
		// This method does not validate scheduleID. If there is no Schedule with the given scheduleID, handle
		// methods like ScheduleHandle.Describe() will return an error.
		GetHandle(ctx context.Context, scheduleID string) ScheduleHandle
	}
)

func (*ScheduleWorkflowAction) isScheduleAction() {
}
