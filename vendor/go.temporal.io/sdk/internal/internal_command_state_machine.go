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
	"container/list"
	"fmt"
	"strconv"

	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"
	historypb "go.temporal.io/api/history/v1"

	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/internal/common/util"
)

type (
	commandState int32
	commandType  int32

	commandID struct {
		commandType commandType
		id          string
	}

	commandStateMachine interface {
		getState() commandState
		getID() commandID
		isDone() bool
		getCommand() *commandpb.Command // return nil if there is no command in current state
		cancel()

		handleStartedEvent()
		handleCancelInitiatedEvent()
		handleCanceledEvent()
		handleCancelFailedEvent()
		handleCompletionEvent()
		handleInitiationFailedEvent()
		handleInitiatedEvent()

		handleCommandSent()
		setData(data interface{})
		getData() interface{}
	}

	commandStateMachineBase struct {
		id      commandID
		state   commandState
		history []string
		data    interface{}
		helper  *commandsHelper
	}

	activityCommandStateMachine struct {
		*commandStateMachineBase
		scheduleID int64
		attributes *commandpb.ScheduleActivityTaskCommandAttributes
	}

	cancelActivityStateMachine struct {
		*commandStateMachineBase
		attributes *commandpb.RequestCancelActivityTaskCommandAttributes

		// The commandsHelper.nextCommandEventIDResetCounter when this command
		// incremented commandsHelper.commandsCancelledDuringWFCancellation.
		cancelledOnEventIDResetCounter uint64
	}

	timerCommandStateMachine struct {
		*commandStateMachineBase
		attributes *commandpb.StartTimerCommandAttributes
	}

	cancelTimerCommandStateMachine struct {
		*commandStateMachineBase
		attributes *commandpb.CancelTimerCommandAttributes

		// The commandsHelper.nextCommandEventIDResetCounter when this command
		// incremented commandsHelper.commandsCancelledDuringWFCancellation.
		cancelledOnEventIDResetCounter uint64
	}

	childWorkflowCommandStateMachine struct {
		*commandStateMachineBase
		attributes *commandpb.StartChildWorkflowExecutionCommandAttributes
	}

	naiveCommandStateMachine struct {
		*commandStateMachineBase
		command *commandpb.Command
	}

	// only possible state transition is: CREATED->SENT->INITIATED->COMPLETED
	cancelExternalWorkflowCommandStateMachine struct {
		*naiveCommandStateMachine
	}

	signalExternalWorkflowCommandStateMachine struct {
		*naiveCommandStateMachine
	}

	// only possible state transition is: CREATED->SENT->COMPLETED
	markerCommandStateMachine struct {
		*naiveCommandStateMachine
	}

	// completeOnSendStateMachine is a generic state machine that transition
	// into a comleted state immediately upon a command being sent (i.e. upon
	// handleCommandSent() being called).
	completeOnSendStateMachine struct {
		*naiveCommandStateMachine
	}

	modifyPropertiesCommandStateMachine struct {
		*naiveCommandStateMachine
	}

	versionMarker struct {
		changeID          string
		searchAttrUpdated bool
	}

	commandsHelper struct {
		nextCommandEventID int64
		orderedCommands    *list.List
		commands           map[commandID]*list.Element

		scheduledEventIDToActivityID          map[int64]string
		scheduledEventIDToCancellationID      map[int64]string
		scheduledEventIDToSignalID            map[int64]string
		versionMarkerLookup                   map[int64]versionMarker
		commandsCancelledDuringWFCancellation int64
		workflowExecutionIsCancelling         bool

		// Incremented everytime nextCommandEventID and
		// commandsCancelledDuringWFCancellation is reset (i.e. on new workflow
		// task). Won't ever happen, but technically the way this value is compared
		// is safe for overflow wrap around.
		nextCommandEventIDResetCounter uint64
	}

	// panic when command state machine is in illegal state
	stateMachineIllegalStatePanic struct {
		message string
	}

	// Error returned when a child workflow with the same id already exists and hasn't completed
	// and been removed from internal state.
	childWorkflowExistsWithId struct {
		id string
	}
)

const (
	commandStateCreated                               commandState = 0
	commandStateCommandSent                           commandState = 1
	commandStateCanceledBeforeInitiated               commandState = 2
	commandStateInitiated                             commandState = 3
	commandStateStarted                               commandState = 4
	commandStateCanceledAfterInitiated                commandState = 5
	commandStateCanceledAfterStarted                  commandState = 6
	commandStateCancellationCommandSent               commandState = 7
	commandStateCompletedAfterCancellationCommandSent commandState = 8
	commandStateCompleted                             commandState = 9
	commandStateCanceledBeforeSent                    commandState = 10
	commandStateCancellationCommandAccepted           commandState = 11
)

const (
	commandTypeActivity                  commandType = 0
	commandTypeChildWorkflow             commandType = 1
	commandTypeCancellation              commandType = 2
	commandTypeMarker                    commandType = 3
	commandTypeTimer                     commandType = 4
	commandTypeSignal                    commandType = 5
	commandTypeUpsertSearchAttributes    commandType = 6
	commandTypeCancelTimer               commandType = 7
	commandTypeRequestCancelActivityTask commandType = 8
	commandTypeAcceptWorkflowUpdate      commandType = 9
	commandTypeCompleteWorkflowUpdate    commandType = 10
	commandTypeModifyProperties          commandType = 11
	commandTypeRejectWorkflowUpdate      commandType = 12
	commandTypeProtocolMessage           commandType = 13
)

const (
	eventCancel                                   = "cancel"
	eventCommandSent                              = "handleCommandSent"
	eventInitiated                                = "handleInitiatedEvent"
	eventInitiationFailed                         = "handleInitiationFailedEvent"
	eventStarted                                  = "handleStartedEvent"
	eventCompletion                               = "handleCompletionEvent"
	eventCancelInitiated                          = "handleCancelInitiatedEvent"
	eventCancelFailed                             = "handleCancelFailedEvent"
	eventCanceled                                 = "handleCanceledEvent"
	eventExternalWorkflowExecutionCancelRequested = "handleExternalWorkflowExecutionCancelRequested"
)

const (
	sideEffectMarkerName        = "SideEffect"
	versionMarkerName           = "Version"
	localActivityMarkerName     = "LocalActivity"
	mutableSideEffectMarkerName = "MutableSideEffect"

	sideEffectMarkerIDName            = "side-effect-id"
	sideEffectMarkerDataName          = "data"
	versionMarkerChangeIDName         = "change-id"
	versionMarkerDataName             = "version"
	versionSearchAttributeUpdatedName = "version-search-attribute-updated"
	localActivityMarkerDataName       = "data"
	localActivityResultName           = "result"
	mutableSideEffectCallCounterName  = "mutable-side-effect-call-counter"
)

func (d commandState) String() string {
	switch d {
	case commandStateCreated:
		return "Created"
	case commandStateCommandSent:
		return "CommandSent"
	case commandStateCanceledBeforeInitiated:
		return "CanceledBeforeInitiated"
	case commandStateInitiated:
		return "Initiated"
	case commandStateStarted:
		return "Started"
	case commandStateCanceledAfterInitiated:
		return "CanceledAfterInitiated"
	case commandStateCanceledAfterStarted:
		return "CanceledAfterStarted"
	case commandStateCancellationCommandSent:
		return "CancellationCommandSent"
	case commandStateCompletedAfterCancellationCommandSent:
		return "CompletedAfterCancellationCommandSent"
	case commandStateCompleted:
		return "Completed"
	case commandStateCanceledBeforeSent:
		return "CanceledBeforeSent"
	case commandStateCancellationCommandAccepted:
		return "CancellationCommandAccepted"
	default:
		return fmt.Sprintf("Unknown: %d", int32(d))
	}
}

func (d commandType) String() string {
	switch d {
	case commandTypeActivity:
		return "Activity"
	case commandTypeChildWorkflow:
		return "ChildWorkflow"
	case commandTypeCancellation:
		return "Cancellation"
	case commandTypeMarker:
		return "Marker"
	case commandTypeTimer:
		return "Timer"
	case commandTypeSignal:
		return "Signal"
	case commandTypeCancelTimer:
		return "CancelTimer"
	case commandTypeRequestCancelActivityTask:
		return "RequestCancelActivityTask"
	case commandTypeAcceptWorkflowUpdate:
		return "AcceptWorkflowUpdate"
	case commandTypeCompleteWorkflowUpdate:
		return "CompleteWorkflowUpdate"
	case commandTypeRejectWorkflowUpdate:
		return "RejectWorkflowUpdate"
	default:
		return "Unknown"
	}
}

func (d commandID) String() string {
	return fmt.Sprintf("CommandType: %v, ID: %v", d.commandType, d.id)
}

func makeCommandID(commandType commandType, id string) commandID {
	return commandID{commandType: commandType, id: id}
}

func (h *commandsHelper) newCommandStateMachineBase(commandType commandType, id string) *commandStateMachineBase {
	return &commandStateMachineBase{
		id:      makeCommandID(commandType, id),
		state:   commandStateCreated,
		history: []string{commandStateCreated.String()},
		helper:  h,
	}
}

func (h *commandsHelper) newActivityCommandStateMachine(
	scheduleID int64,
	attributes *commandpb.ScheduleActivityTaskCommandAttributes,
) *activityCommandStateMachine {
	base := h.newCommandStateMachineBase(commandTypeActivity, attributes.GetActivityId())
	return &activityCommandStateMachine{
		commandStateMachineBase: base,
		scheduleID:              scheduleID,
		attributes:              attributes,
	}
}

func (h *commandsHelper) newCancelActivityStateMachine(attributes *commandpb.RequestCancelActivityTaskCommandAttributes) *cancelActivityStateMachine {
	base := h.newCommandStateMachineBase(commandTypeRequestCancelActivityTask, strconv.FormatInt(attributes.GetScheduledEventId(), 10))
	return &cancelActivityStateMachine{
		commandStateMachineBase: base,
		attributes:              attributes,
	}
}

func (h *commandsHelper) newTimerCommandStateMachine(attributes *commandpb.StartTimerCommandAttributes) *timerCommandStateMachine {
	base := h.newCommandStateMachineBase(commandTypeTimer, attributes.GetTimerId())
	return &timerCommandStateMachine{
		commandStateMachineBase: base,
		attributes:              attributes,
	}
}

func (h *commandsHelper) newCancelTimerCommandStateMachine(attributes *commandpb.CancelTimerCommandAttributes) *cancelTimerCommandStateMachine {
	base := h.newCommandStateMachineBase(commandTypeCancelTimer, attributes.GetTimerId())
	return &cancelTimerCommandStateMachine{
		commandStateMachineBase: base,
		attributes:              attributes,
	}
}

func (h *commandsHelper) newChildWorkflowCommandStateMachine(attributes *commandpb.StartChildWorkflowExecutionCommandAttributes) *childWorkflowCommandStateMachine {
	base := h.newCommandStateMachineBase(commandTypeChildWorkflow, attributes.GetWorkflowId())
	return &childWorkflowCommandStateMachine{
		commandStateMachineBase: base,
		attributes:              attributes,
	}
}

func (h *commandsHelper) newNaiveCommandStateMachine(commandType commandType, id string, command *commandpb.Command) *naiveCommandStateMachine {
	base := h.newCommandStateMachineBase(commandType, id)
	return &naiveCommandStateMachine{
		commandStateMachineBase: base,
		command:                 command,
	}
}

func (h *commandsHelper) newMarkerCommandStateMachine(id string, attributes *commandpb.RecordMarkerCommandAttributes) *markerCommandStateMachine {
	d := createNewCommand(enumspb.COMMAND_TYPE_RECORD_MARKER)
	d.Attributes = &commandpb.Command_RecordMarkerCommandAttributes{RecordMarkerCommandAttributes: attributes}
	return &markerCommandStateMachine{
		naiveCommandStateMachine: h.newNaiveCommandStateMachine(commandTypeMarker, id, d),
	}
}

func (h *commandsHelper) newCancelExternalWorkflowStateMachine(attributes *commandpb.RequestCancelExternalWorkflowExecutionCommandAttributes, cancellationID string) *cancelExternalWorkflowCommandStateMachine {
	d := createNewCommand(enumspb.COMMAND_TYPE_REQUEST_CANCEL_EXTERNAL_WORKFLOW_EXECUTION)
	d.Attributes = &commandpb.Command_RequestCancelExternalWorkflowExecutionCommandAttributes{RequestCancelExternalWorkflowExecutionCommandAttributes: attributes}
	return &cancelExternalWorkflowCommandStateMachine{
		naiveCommandStateMachine: h.newNaiveCommandStateMachine(commandTypeCancellation, cancellationID, d),
	}
}

func (h *commandsHelper) newSignalExternalWorkflowStateMachine(attributes *commandpb.SignalExternalWorkflowExecutionCommandAttributes, signalID string) *signalExternalWorkflowCommandStateMachine {
	d := createNewCommand(enumspb.COMMAND_TYPE_SIGNAL_EXTERNAL_WORKFLOW_EXECUTION)
	d.Attributes = &commandpb.Command_SignalExternalWorkflowExecutionCommandAttributes{SignalExternalWorkflowExecutionCommandAttributes: attributes}
	return &signalExternalWorkflowCommandStateMachine{
		naiveCommandStateMachine: h.newNaiveCommandStateMachine(commandTypeSignal, signalID, d),
	}
}

func (h *commandsHelper) newUpsertSearchAttributesStateMachine(attributes *commandpb.UpsertWorkflowSearchAttributesCommandAttributes, upsertID string) *completeOnSendStateMachine {
	d := createNewCommand(enumspb.COMMAND_TYPE_UPSERT_WORKFLOW_SEARCH_ATTRIBUTES)
	d.Attributes = &commandpb.Command_UpsertWorkflowSearchAttributesCommandAttributes{UpsertWorkflowSearchAttributesCommandAttributes: attributes}
	return &completeOnSendStateMachine{
		naiveCommandStateMachine: h.newNaiveCommandStateMachine(commandTypeUpsertSearchAttributes, upsertID, d),
	}
}

func (h *commandsHelper) newModifyPropertiesStateMachine(
	attributes *commandpb.ModifyWorkflowPropertiesCommandAttributes,
	changeID string,
) *modifyPropertiesCommandStateMachine {
	d := createNewCommand(enumspb.COMMAND_TYPE_MODIFY_WORKFLOW_PROPERTIES)
	d.Attributes = &commandpb.Command_ModifyWorkflowPropertiesCommandAttributes{
		ModifyWorkflowPropertiesCommandAttributes: attributes,
	}
	return &modifyPropertiesCommandStateMachine{
		naiveCommandStateMachine: h.newNaiveCommandStateMachine(
			commandTypeModifyProperties,
			changeID,
			d,
		),
	}
}

func (d *commandStateMachineBase) getState() commandState {
	return d.state
}

func (d *commandStateMachineBase) getID() commandID {
	return d.id
}

func (d *commandStateMachineBase) isDone() bool {
	return d.state == commandStateCompleted || d.state == commandStateCompletedAfterCancellationCommandSent
}

func (d *commandStateMachineBase) setData(data interface{}) {
	d.data = data
}

func (d *commandStateMachineBase) getData() interface{} {
	return d.data
}

func (d *commandStateMachineBase) moveState(newState commandState, event string) {
	d.history = append(d.history, event)
	d.state = newState
	d.history = append(d.history, newState.String())

	if newState == commandStateCompleted {
		if elem, ok := d.helper.commands[d.getID()]; ok {
			d.helper.orderedCommands.Remove(elem)
			delete(d.helper.commands, d.getID())
		}
	}
}

func (d stateMachineIllegalStatePanic) String() string {
	return d.message
}

func panicIllegalState(message string) {
	panic(stateMachineIllegalStatePanic{message: message})
}

func (d *commandStateMachineBase) failStateTransition(event string) {
	// this is when we detect illegal state transition, likely due to ill history sequence or nondeterministic workflow code
	panicIllegalState(fmt.Sprintf("invalid state transition: attempt to %v, %v", event, d))
}

func (d *commandStateMachineBase) handleCommandSent() {
	switch d.state {
	case commandStateCreated:
		d.moveState(commandStateCommandSent, eventCommandSent)
	}
}

func (d *commandStateMachineBase) cancel() {
	switch d.state {
	case commandStateCompleted, commandStateCompletedAfterCancellationCommandSent:
		// No op. This is legit. People could cancel context after timer/activity is done.
	case commandStateCreated:
		d.moveState(commandStateCanceledBeforeSent, eventCancel)
	case commandStateCommandSent:
		d.moveState(commandStateCancellationCommandSent, eventCancel)
	case commandStateInitiated:
		if d.helper.workflowExecutionIsCancelling {
			d.helper.commandsCancelledDuringWFCancellation++
		}
		d.moveState(commandStateCanceledAfterInitiated, eventCancel)
	default:
		d.failStateTransition(eventCancel)
	}
}

func (d *commandStateMachineBase) handleInitiatedEvent() {
	switch d.state {
	case commandStateCommandSent:
		d.moveState(commandStateInitiated, eventInitiated)
	case commandStateCanceledBeforeInitiated, commandStateCanceledBeforeSent, commandStateCancellationCommandSent:
		d.moveState(commandStateCanceledAfterInitiated, eventInitiated)
	default:
		d.failStateTransition(eventInitiated)
	}
}

func (d *commandStateMachineBase) handleInitiationFailedEvent() {
	switch d.state {
	case commandStateInitiated, commandStateCommandSent, commandStateCanceledBeforeInitiated, commandStateCancellationCommandSent:
		d.moveState(commandStateCompleted, eventInitiationFailed)
	default:
		d.failStateTransition(eventInitiationFailed)
	}
}

func (d *commandStateMachineBase) handleStartedEvent() {
	d.history = append(d.history, eventStarted)
}

func (d *commandStateMachineBase) handleCompletionEvent() {
	switch d.state {
	case commandStateCanceledAfterInitiated, commandStateInitiated:
		d.moveState(commandStateCompleted, eventCompletion)
	case commandStateCancellationCommandSent:
		d.moveState(commandStateCompletedAfterCancellationCommandSent, eventCompletion)
	default:
		d.failStateTransition(eventCompletion)
	}
}

func (d *commandStateMachineBase) handleCancelInitiatedEvent() {
	d.history = append(d.history, eventCancelInitiated)
	switch d.state {
	case commandStateCancellationCommandSent, commandStateCanceledAfterInitiated:
	// No state change
	default:
		d.failStateTransition(eventCancelInitiated)
	}
}

func (d *commandStateMachineBase) handleCancelFailedEvent() {
	switch d.state {
	case commandStateCompletedAfterCancellationCommandSent:
		d.moveState(commandStateCompleted, eventCancelFailed)
	default:
		d.failStateTransition(eventCancelFailed)
	}
}

func (d *commandStateMachineBase) handleCanceledEvent() {
	switch d.state {
	case commandStateCancellationCommandSent, commandStateCanceledAfterInitiated, commandStateCanceledAfterStarted, commandStateCancellationCommandAccepted:
		d.moveState(commandStateCompleted, eventCanceled)
	default:
		d.failStateTransition(eventCanceled)
	}
}

func (d *commandStateMachineBase) String() string {
	return fmt.Sprintf("%v, state=%v, isDone()=%v, history=%v",
		d.id, d.state, d.isDone(), d.history)
}

func (d *activityCommandStateMachine) getCommand() *commandpb.Command {
	switch d.state {
	case commandStateCreated, commandStateCanceledBeforeSent:
		command := createNewCommand(enumspb.COMMAND_TYPE_SCHEDULE_ACTIVITY_TASK)
		command.Attributes = &commandpb.Command_ScheduleActivityTaskCommandAttributes{ScheduleActivityTaskCommandAttributes: d.attributes}
		return command
	default:
		return nil
	}
}

func (d *activityCommandStateMachine) handleCommandSent() {
	switch d.state {
	case commandStateCanceledAfterInitiated:
		d.moveState(commandStateCancellationCommandSent, eventCommandSent)
	default:
		d.commandStateMachineBase.handleCommandSent()
	}
}

func (d *activityCommandStateMachine) handleCancelFailedEvent() {
	// Request to cancel activity now results in either activity completion, failed, timedout, or canceled
	// Request to cancel itself can never fail and invalid RequestCancelActivity commands results in the
	// entire command being failed.
	d.failStateTransition(eventCancelFailed)
}

func (d *activityCommandStateMachine) cancel() {
	switch d.state {
	case commandStateCreated, commandStateCommandSent, commandStateInitiated:
		attribs := &commandpb.RequestCancelActivityTaskCommandAttributes{
			ScheduledEventId: d.scheduleID,
		}
		cancelCmd := d.helper.newCancelActivityStateMachine(attribs)
		d.helper.addCommand(cancelCmd)
		// We must mark the event ID reset counter for when we performed this
		// increment so a potential decrement can only decrement if it wasn't
		// reset
		cancelCmd.cancelledOnEventIDResetCounter = d.helper.nextCommandEventIDResetCounter
		// We also mark the schedule command as not eager if we haven't sent it yet.
		// Server behavior differs on eager vs non-eager when scheduling and
		// cancelling during the same task completion. If it has not been sent this
		// means we are cancelling at the same time as scheduling which is not
		// properly supported for eager activities.
		if d.state != commandStateCommandSent {
			d.attributes.RequestEagerExecution = false
		}
	}

	d.commandStateMachineBase.cancel()
}

func (d *timerCommandStateMachine) cancel() {
	switch d.state {
	case commandStateCreated, commandStateCommandSent, commandStateInitiated:
		attribs := &commandpb.CancelTimerCommandAttributes{
			TimerId: d.attributes.TimerId,
		}
		cancelCmd := d.helper.newCancelTimerCommandStateMachine(attribs)
		d.helper.addCommand(cancelCmd)
		// We must mark the event ID reset counter for when we performed this
		// increment so a potential decrement can only decrement if it wasn't
		// reset
		cancelCmd.cancelledOnEventIDResetCounter = d.helper.nextCommandEventIDResetCounter
	}

	d.commandStateMachineBase.cancel()
}

func (d *timerCommandStateMachine) isDone() bool {
	return d.state == commandStateCompleted
}

func (d *timerCommandStateMachine) handleCommandSent() {
	switch d.state {
	case commandStateCanceledAfterInitiated:
		d.moveState(commandStateCancellationCommandSent, eventCommandSent)
	default:
		d.commandStateMachineBase.handleCommandSent()
	}
}

func (d *cancelActivityStateMachine) getCommand() *commandpb.Command {
	switch d.state {
	case commandStateCreated:
		command := createNewCommand(enumspb.COMMAND_TYPE_REQUEST_CANCEL_ACTIVITY_TASK)
		command.Attributes = &commandpb.Command_RequestCancelActivityTaskCommandAttributes{RequestCancelActivityTaskCommandAttributes: d.attributes}
		return command
	default:
		return nil
	}
}

func (d *timerCommandStateMachine) getCommand() *commandpb.Command {
	switch d.state {
	case commandStateCreated, commandStateCanceledBeforeSent:
		command := createNewCommand(enumspb.COMMAND_TYPE_START_TIMER)
		command.Attributes = &commandpb.Command_StartTimerCommandAttributes{StartTimerCommandAttributes: d.attributes}
		return command
	default:
		return nil
	}
}

func (d *cancelTimerCommandStateMachine) getCommand() *commandpb.Command {
	switch d.state {
	case commandStateCreated:
		command := createNewCommand(enumspb.COMMAND_TYPE_CANCEL_TIMER)
		command.Attributes = &commandpb.Command_CancelTimerCommandAttributes{CancelTimerCommandAttributes: d.attributes}
		return command
	default:
		return nil
	}
}

func (d *childWorkflowCommandStateMachine) getCommand() *commandpb.Command {
	switch d.state {
	case commandStateCreated:
		command := createNewCommand(enumspb.COMMAND_TYPE_START_CHILD_WORKFLOW_EXECUTION)
		command.Attributes = &commandpb.Command_StartChildWorkflowExecutionCommandAttributes{StartChildWorkflowExecutionCommandAttributes: d.attributes}
		return command
	case commandStateCanceledAfterStarted:
		command := createNewCommand(enumspb.COMMAND_TYPE_REQUEST_CANCEL_EXTERNAL_WORKFLOW_EXECUTION)
		command.Attributes = &commandpb.Command_RequestCancelExternalWorkflowExecutionCommandAttributes{RequestCancelExternalWorkflowExecutionCommandAttributes: &commandpb.RequestCancelExternalWorkflowExecutionCommandAttributes{
			Namespace:         d.attributes.Namespace,
			WorkflowId:        d.attributes.WorkflowId,
			ChildWorkflowOnly: true,
		}}
		return command
	default:
		return nil
	}
}

func (d *childWorkflowCommandStateMachine) handleCommandSent() {
	switch d.state {
	case commandStateCanceledAfterStarted:
		d.moveState(commandStateCancellationCommandSent, eventCommandSent)
	default:
		d.commandStateMachineBase.handleCommandSent()
	}
}

func (d *childWorkflowCommandStateMachine) handleStartedEvent() {
	switch d.state {
	case commandStateInitiated:
		d.moveState(commandStateStarted, eventStarted)
	case commandStateCanceledAfterInitiated:
		d.moveState(commandStateCanceledAfterStarted, eventStarted)
	default:
		d.commandStateMachineBase.handleStartedEvent()
	}
}

func (d *childWorkflowCommandStateMachine) handleInitiatedEvent() {
	switch d.state {
	case commandStateCancellationCommandSent:
		d.failStateTransition(eventInitiated)
	default:
		d.commandStateMachineBase.handleInitiatedEvent()
	}
}

func (d *childWorkflowCommandStateMachine) handleCancelFailedEvent() {
	switch d.state {
	case commandStateCancellationCommandSent:
		d.moveState(commandStateStarted, eventCancelFailed)
	default:
		d.commandStateMachineBase.handleCancelFailedEvent()
	}
}

func (d *childWorkflowCommandStateMachine) cancel() {
	switch d.state {
	case commandStateStarted:
		if d.helper.workflowExecutionIsCancelling {
			d.helper.commandsCancelledDuringWFCancellation++
		}
		d.moveState(commandStateCanceledAfterStarted, eventCancel)
		// A child workflow may be canceled _after_ something like an activity start
		// happens inside a simulated goroutine. However, since the state of the
		// entire child workflow is recorded based on when it started not when it
		// was canceled, we have to move it to the end once canceled to keep the
		// expected commands in order of when they actually occurred.
		d.helper.moveCommandToBack(d)
		d.helper.incrementNextCommandEventID()
	default:
		d.commandStateMachineBase.cancel()
	}
}

func (d *childWorkflowCommandStateMachine) handleCanceledEvent() {
	switch d.state {
	case commandStateStarted:
		d.moveState(commandStateCompleted, eventCanceled)
	case commandStateCancellationCommandSent:
		// We've sent the command but haven't seen the server accept the cancellation. We must ensure this command hangs
		// around, because it is possible for the child workflow to be canceled before we've seen the event.
		d.moveState(commandStateCompletedAfterCancellationCommandSent, eventCanceled)
	default:
		d.commandStateMachineBase.handleCanceledEvent()
	}
}

func (d *childWorkflowCommandStateMachine) handleCompletionEvent() {
	switch d.state {
	case commandStateStarted, commandStateCanceledAfterStarted,
		commandStateCompletedAfterCancellationCommandSent, commandStateCancellationCommandAccepted:
		d.moveState(commandStateCompleted, eventCompletion)
	default:
		d.commandStateMachineBase.handleCompletionEvent()
	}
}

func (d *childWorkflowCommandStateMachine) handleExternalWorkflowExecutionCancelRequested() {
	if d.getState() == commandStateCompletedAfterCancellationCommandSent {
		// Now we're really done.
		d.handleCompletionEvent()
	} else {
		// We should be in the cancellation command sent stage - new state to indicate we have seen the cancel accepted
		d.moveState(commandStateCancellationCommandAccepted, eventExternalWorkflowExecutionCancelRequested)
	}
}

func (d *naiveCommandStateMachine) getCommand() *commandpb.Command {
	switch d.state {
	case commandStateCreated:
		return d.command
	default:
		return nil
	}
}

func (d *naiveCommandStateMachine) cancel() {
	panic("unsupported operation")
}

func (d *naiveCommandStateMachine) handleCompletionEvent() {
	panic("unsupported operation")
}

func (d *naiveCommandStateMachine) handleInitiatedEvent() {
	panic("unsupported operation")
}

func (d *naiveCommandStateMachine) handleInitiationFailedEvent() {
	panic("unsupported operation")
}

func (d *naiveCommandStateMachine) handleStartedEvent() {
	panic("unsupported operation")
}

func (d *naiveCommandStateMachine) handleCanceledEvent() {
	panic("unsupported operation")
}

func (d *naiveCommandStateMachine) handleCancelFailedEvent() {
	panic("unsupported operation")
}

func (d *naiveCommandStateMachine) handleCancelInitiatedEvent() {
	panic("unsupported operation")
}

func (d *cancelExternalWorkflowCommandStateMachine) handleInitiatedEvent() {
	switch d.state {
	case commandStateCommandSent:
		d.moveState(commandStateInitiated, eventInitiated)
	default:
		d.failStateTransition(eventInitiated)
	}
}

func (d *cancelExternalWorkflowCommandStateMachine) handleCompletionEvent() {
	switch d.state {
	case commandStateInitiated:
		d.moveState(commandStateCompleted, eventCompletion)
	default:
		d.failStateTransition(eventCompletion)
	}
}

func (d *signalExternalWorkflowCommandStateMachine) handleInitiatedEvent() {
	switch d.state {
	case commandStateCommandSent:
		d.moveState(commandStateInitiated, eventInitiated)
	default:
		d.failStateTransition(eventInitiated)
	}
}

func (d *signalExternalWorkflowCommandStateMachine) handleCompletionEvent() {
	switch d.state {
	case commandStateInitiated:
		d.moveState(commandStateCompleted, eventCompletion)
	default:
		d.failStateTransition(eventCompletion)
	}
}

func (d *markerCommandStateMachine) handleCommandSent() {
	// Marker command state machine is considered as completed once command is sent.
	// For SideEffect/Version markers, when the history event is applied, there is no marker command state machine yet
	// because we preload those marker events.
	// For local activity, when we apply the history event, we use it to create the marker state machine, there is no
	// other event to drive it to completed state.
	switch d.state {
	case commandStateCreated:
		d.moveState(commandStateCompleted, eventCommandSent)
	}
}

func (d *completeOnSendStateMachine) handleCommandSent() {
	// This command is considered as completed once command is sent.
	switch d.state {
	case commandStateCreated:
		d.moveState(commandStateCompleted, eventCommandSent)
	}
}

func (d *modifyPropertiesCommandStateMachine) handleCommandSent() {
	// This command is considered as completed once command is sent.
	switch d.state {
	case commandStateCreated:
		d.moveState(commandStateCompleted, eventCommandSent)
	}
}

func newCommandsHelper() *commandsHelper {
	return &commandsHelper{
		orderedCommands: list.New(),
		commands:        make(map[commandID]*list.Element),

		scheduledEventIDToActivityID:          make(map[int64]string),
		scheduledEventIDToCancellationID:      make(map[int64]string),
		scheduledEventIDToSignalID:            make(map[int64]string),
		versionMarkerLookup:                   make(map[int64]versionMarker),
		commandsCancelledDuringWFCancellation: 0,
	}
}

func (h *commandsHelper) incrementNextCommandEventID() {
	h.nextCommandEventID++
}

func (h *commandsHelper) setCurrentWorkflowTaskStartedEventID(workflowTaskStartedEventID int64) {
	// Server always processes the commands in the same order it is generated by client and each command results in
	// corresponding history event after processing. So we can use workflow task started event id + 2 as the offset as
	// workflow task completed event is always the first event in the workflow task followed by events generated from
	// commands. This allows client sdk to deterministically predict history event ids generated by processing of the
	// command. We must also add the number of cancel commands that were spawned during cancellation of the workflow
	// execution as those canceled command events will show up *after* the workflow task completed event.
	h.nextCommandEventID = workflowTaskStartedEventID + 2 + h.commandsCancelledDuringWFCancellation
	h.commandsCancelledDuringWFCancellation = 0
	// We must change the counter here so that others who mutate
	// commandsCancelledDuringWFCancellation know it has since been reset
	h.nextCommandEventIDResetCounter++
}

func (h *commandsHelper) getNextID() int64 {
	// First check if we have a GetVersion marker in the lookup map
	h.incrementNextCommandEventIDIfVersionMarker()
	if h.nextCommandEventID == 0 {
		panic("Attempt to generate a command before processing WorkflowTaskStarted event")
	}
	return h.nextCommandEventID
}

func (h *commandsHelper) incrementNextCommandEventIDIfVersionMarker() {
	marker, ok := h.versionMarkerLookup[h.nextCommandEventID]
	for ok {
		// Remove the marker from the lookup map and increment nextCommandEventID by 2 because call to GetVersion
		// results in 1 or 2 events in the history.  One is GetVersion marker event for changeID and change version, other
		// is UpsertSearchableAttributes to keep track of executions using particular version of code.
		delete(h.versionMarkerLookup, h.nextCommandEventID)
		h.incrementNextCommandEventID()
		// UpsertSearchableAttributes may not have been written if the search attribute was too large.
		if marker.searchAttrUpdated {
			h.incrementNextCommandEventID()
		}
		marker, ok = h.versionMarkerLookup[h.nextCommandEventID]
	}
}

func (h *commandsHelper) getCommand(id commandID) commandStateMachine {
	command, ok := h.commands[id]
	if !ok {
		panicMsg := fmt.Sprintf("unknown command %v, possible causes are nondeterministic workflow definition code"+
			" or incompatible change in the workflow definition", id)
		panicIllegalState(panicMsg)
	}
	return command.Value.(commandStateMachine)
}

func (h *commandsHelper) addCommand(command commandStateMachine) {
	if _, ok := h.commands[command.getID()]; ok {
		panicMsg := fmt.Sprintf("adding duplicate command %v", command)
		panicIllegalState(panicMsg)
	}
	element := h.orderedCommands.PushBack(command)
	h.commands[command.getID()] = element

	// Every time new command is added increment the counter used for generating ID
	h.incrementNextCommandEventIDIfVersionMarker()
	h.incrementNextCommandEventID()
}

// This really should not exist, but is unavoidable without totally redesigning the Go SDK to avoid
// doing event number counting. EX: Because a workflow execution cancel requested event calls a callback
// on timers that immediately cancels them, we will queue up a cancel timer command even though that timer firing
// might be in the same workflow task. In practice this only seems to happen during unhandled command events.
func (h *commandsHelper) removeCancelOfResolvedCommand(commandID commandID) {
	// Ensure this isn't misused for non-cancel commands
	if commandID.commandType != commandTypeCancelTimer && commandID.commandType != commandTypeRequestCancelActivityTask {
		panic("removeCancelOfResolvedCommand should only be called for cancel timer / activity")
	}
	orderedCmdEl, ok := h.commands[commandID]
	if ok {
		delete(h.commands, commandID)
		command := h.orderedCommands.Remove(orderedCmdEl)
		// Sometimes commandsCancelledDuringWFCancellation was incremented before
		// it was reset and sometimes not. We make sure the workflow execution is
		// actually cancelling since that's the only time we increment the counter
		// in the first place. Also, we use the reset counter to see if we're still
		// on the same iteration where we may have incremented it before.
		if h.workflowExecutionIsCancelling {
			switch command := command.(type) {
			case *cancelActivityStateMachine:
				if command.cancelledOnEventIDResetCounter == h.nextCommandEventIDResetCounter {
					h.commandsCancelledDuringWFCancellation--
				}
			case *cancelTimerCommandStateMachine:
				if command.cancelledOnEventIDResetCounter == h.nextCommandEventIDResetCounter {
					h.commandsCancelledDuringWFCancellation--
				}
			}
		}
	}
}

func (h *commandsHelper) moveCommandToBack(command commandStateMachine) {
	elem := h.commands[command.getID()]
	if elem == nil {
		panicIllegalState(fmt.Sprintf("moving command not present %v", command))
	}
	h.orderedCommands.Remove(elem)
	h.commands[command.getID()] = h.orderedCommands.PushBack(command)
}

func (h *commandsHelper) scheduleActivityTask(
	scheduleID int64,
	attributes *commandpb.ScheduleActivityTaskCommandAttributes,
) commandStateMachine {
	h.scheduledEventIDToActivityID[scheduleID] = attributes.GetActivityId()
	command := h.newActivityCommandStateMachine(scheduleID, attributes)
	h.addCommand(command)
	return command
}

func (h *commandsHelper) requestCancelActivityTask(activityID string) commandStateMachine {
	id := makeCommandID(commandTypeActivity, activityID)
	command := h.getCommand(id)
	command.cancel()
	return command
}

func (h *commandsHelper) handleActivityTaskClosed(activityID string, scheduledEventID int64) commandStateMachine {
	command := h.getCommand(makeCommandID(commandTypeActivity, activityID))
	// If, for whatever reason, we were going to send an activity cancel request, don't do that anymore
	// since we already know the activity is resolved.
	possibleCancelID := makeCommandID(commandTypeRequestCancelActivityTask, activityID)
	h.removeCancelOfResolvedCommand(possibleCancelID)
	command.handleCompletionEvent()
	delete(h.scheduledEventIDToActivityID, scheduledEventID)
	return command
}

func (h *commandsHelper) handleActivityTaskScheduled(activityID string, scheduledEventID int64) {
	if _, ok := h.scheduledEventIDToActivityID[scheduledEventID]; !ok {
		panicMsg := fmt.Sprintf("lookup failed for scheduledEventID to activityID: scheduleEventID: %v, activityID: %v",
			scheduledEventID, activityID)
		panicIllegalState(panicMsg)
	}

	command := h.getCommand(makeCommandID(commandTypeActivity, activityID))
	command.handleInitiatedEvent()
}

func (h *commandsHelper) handleActivityTaskCancelRequested(scheduledEventID int64) {
	activityID, ok := h.scheduledEventIDToActivityID[scheduledEventID]
	if !ok {
		panicIllegalState(fmt.Sprintf("unable to find activityID for the scheduledEventID: %v", scheduledEventID))
	}
	command := h.getCommand(makeCommandID(commandTypeActivity, activityID))
	command.handleCancelInitiatedEvent()
}

func (h *commandsHelper) handleActivityTaskCanceled(activityID string, scheduledEventID int64) commandStateMachine {
	command := h.getCommand(makeCommandID(commandTypeActivity, activityID))
	command.handleCanceledEvent()
	delete(h.scheduledEventIDToActivityID, scheduledEventID)
	return command
}

func (h *commandsHelper) getActivityAndScheduledEventIDs(event *historypb.HistoryEvent) (string, int64) {
	var scheduledEventID int64 = -1
	switch event.GetEventType() {
	case enumspb.EVENT_TYPE_ACTIVITY_TASK_CANCELED:
		scheduledEventID = event.GetActivityTaskCanceledEventAttributes().GetScheduledEventId()
	case enumspb.EVENT_TYPE_ACTIVITY_TASK_COMPLETED:
		scheduledEventID = event.GetActivityTaskCompletedEventAttributes().GetScheduledEventId()
	case enumspb.EVENT_TYPE_ACTIVITY_TASK_FAILED:
		scheduledEventID = event.GetActivityTaskFailedEventAttributes().GetScheduledEventId()
	case enumspb.EVENT_TYPE_ACTIVITY_TASK_TIMED_OUT:
		scheduledEventID = event.GetActivityTaskTimedOutEventAttributes().GetScheduledEventId()
	default:
		panicIllegalState(fmt.Sprintf("unexpected event type: %v", event.GetEventType()))
	}

	activityID, ok := h.scheduledEventIDToActivityID[scheduledEventID]
	if !ok {
		panicIllegalState(fmt.Sprintf("unable to find activityID for the event: %v", util.HistoryEventToString(event)))
	}
	return activityID, scheduledEventID
}

func (h *commandsHelper) recordVersionMarker(changeID string, version Version, dc converter.DataConverter, searchAttributeWasUpdated bool) commandStateMachine {
	markerID := fmt.Sprintf("%v_%v", versionMarkerName, changeID)

	changeIDPayload, err := dc.ToPayloads(changeID)
	if err != nil {
		panic(err)
	}

	versionPayload, err := dc.ToPayloads(version)
	if err != nil {
		panic(err)
	}

	recordMarker := &commandpb.RecordMarkerCommandAttributes{
		MarkerName: versionMarkerName,
		Details: map[string]*commonpb.Payloads{
			versionMarkerChangeIDName: changeIDPayload,
			versionMarkerDataName:     versionPayload,
		},
	}

	if !searchAttributeWasUpdated {
		searchAttributeWasUpdatedPayload, err := dc.ToPayloads(searchAttributeWasUpdated)
		if err != nil {
			panic(err)
		}
		recordMarker.Details[versionSearchAttributeUpdatedName] = searchAttributeWasUpdatedPayload
	}

	command := h.newMarkerCommandStateMachine(markerID, recordMarker)
	h.addCommand(command)
	return command
}

func (h *commandsHelper) handleVersionMarker(eventID int64, changeID string, searchAttrUpdated bool) {
	if _, ok := h.versionMarkerLookup[eventID]; ok {
		panicMsg := fmt.Sprintf("marker event already exists for eventID in lookup: eventID: %v, changeID: %v",
			eventID, changeID)
		panicIllegalState(panicMsg)
	}

	// During processing of a workflow task we reorder all GetVersion markers and process them first.
	// Keep track of all GetVersion marker events during the processing of workflow task so we can
	// generate correct eventIDs for other events during replay.
	h.versionMarkerLookup[eventID] = versionMarker{
		changeID:          changeID,
		searchAttrUpdated: searchAttrUpdated,
	}
}

func (h *commandsHelper) recordSideEffectMarker(sideEffectID int64, data *commonpb.Payloads, dc converter.DataConverter) commandStateMachine {
	markerID := fmt.Sprintf("%v_%v", sideEffectMarkerName, sideEffectID)
	sideEffectIDPayload, err := dc.ToPayloads(sideEffectID)
	if err != nil {
		panic(err)
	}

	attributes := &commandpb.RecordMarkerCommandAttributes{
		MarkerName: sideEffectMarkerName,
		Details: map[string]*commonpb.Payloads{
			sideEffectMarkerIDName:   sideEffectIDPayload,
			sideEffectMarkerDataName: data,
		},
	}
	command := h.newMarkerCommandStateMachine(markerID, attributes)
	h.addCommand(command)
	return command
}

func (h *commandsHelper) recordLocalActivityMarker(activityID string, details map[string]*commonpb.Payloads, failure *failurepb.Failure) commandStateMachine {
	markerID := fmt.Sprintf("%v_%v", localActivityMarkerName, activityID)
	attributes := &commandpb.RecordMarkerCommandAttributes{
		MarkerName: localActivityMarkerName,
		Failure:    failure,
		Details:    details,
	}
	command := h.newMarkerCommandStateMachine(markerID, attributes)
	// LocalActivity marker is added only when it completes and schedule logic never relies on GenerateSequence to
	// create a unique activity id like in the case of ExecuteActivity.  This causes the problem as we only perform
	// the check to increment counter to account for GetVersion special handling as part of it.  This will result
	// in wrong IDs to be generated if there is GetVersion call before local activities.  Explicitly calling getNextID
	// to correctly incrementing counter before adding the command.
	h.getNextID()
	h.addCommand(command)
	return command
}

func (h *commandsHelper) recordMutableSideEffectMarker(mutableSideEffectID string, callCountHint int, data *commonpb.Payloads, dc converter.DataConverter) commandStateMachine {
	// In order to avoid duplicate marker IDs, we must append the counter to the
	// user-provided ID
	mutableSideEffectID = fmt.Sprintf("%v_%v", mutableSideEffectID, h.nextCommandEventID)
	markerID := fmt.Sprintf("%v_%v", mutableSideEffectMarkerName, mutableSideEffectID)

	mutableSideEffectIDPayload, err := dc.ToPayloads(mutableSideEffectID)
	if err != nil {
		panic(err)
	}

	mutableSideEffectCounterPayload, err := dc.ToPayloads(callCountHint)
	if err != nil {
		panic(err)
	}

	attributes := &commandpb.RecordMarkerCommandAttributes{
		MarkerName: mutableSideEffectMarkerName,
		Details: map[string]*commonpb.Payloads{
			sideEffectMarkerIDName:           mutableSideEffectIDPayload,
			sideEffectMarkerDataName:         data,
			mutableSideEffectCallCounterName: mutableSideEffectCounterPayload,
		},
	}
	command := h.newMarkerCommandStateMachine(markerID, attributes)
	h.addCommand(command)
	return command
}

// startChildWorkflowExecution can return an error in the event that there is already a child wf
// with the same ID which exists as a command in memory. Other SDKs actually will send this command
// to server, and have it reject it - but here the command ID is exactly equal to the child's wf ID,
// and changing that without potentially blowing up backwards compatability is difficult. So we
// return the error eagerly locally, which is at least an improvement on panicking.
func (h *commandsHelper) startChildWorkflowExecution(attributes *commandpb.StartChildWorkflowExecutionCommandAttributes) (commandStateMachine, error) {
	command := h.newChildWorkflowCommandStateMachine(attributes)
	if h.commands[command.getID()] != nil {
		return nil, &childWorkflowExistsWithId{id: attributes.WorkflowId}
	}
	h.addCommand(command)
	return command, nil
}

func (h *commandsHelper) handleStartChildWorkflowExecutionInitiated(workflowID string) {
	command := h.getCommand(makeCommandID(commandTypeChildWorkflow, workflowID))
	command.handleInitiatedEvent()
}

func (h *commandsHelper) handleStartChildWorkflowExecutionFailed(workflowID string) commandStateMachine {
	command := h.getCommand(makeCommandID(commandTypeChildWorkflow, workflowID))
	command.handleInitiationFailedEvent()
	return command
}

func (h *commandsHelper) requestCancelExternalWorkflowExecution(namespace, workflowID, runID string, cancellationID string, childWorkflowOnly bool) commandStateMachine {
	if childWorkflowOnly {
		// For cancellation of child workflow only, we do not use cancellation ID
		// since the child workflow cancellation go through the existing child workflow
		// state machine, and we use workflow ID as identifier
		// we also do not use run ID, since child workflow can do continue-as-new
		// which will have different run ID
		// there will be server side validation that target workflow is child workflow

		// sanity check that cancellation ID is not set
		if len(cancellationID) != 0 {
			panic("cancellation on child workflow should not use cancellation ID")
		}
		// sanity check that run ID is not set
		if len(runID) != 0 {
			panic("cancellation on child workflow should not use run ID")
		}
		// targeting child workflow
		command := h.getCommand(makeCommandID(commandTypeChildWorkflow, workflowID))
		command.cancel()
		return command
	}

	// For cancellation of external workflow, we have to use cancellation ID
	// to identify different cancellation request (command) / response (history event)
	// client can also use this code path to cancel its own child workflow, however, there will
	// be no server side validation that target workflow is the child

	// sanity check that cancellation ID is set
	if len(cancellationID) == 0 {
		panic("cancellation on external workflow should use cancellation ID")
	}
	attributes := &commandpb.RequestCancelExternalWorkflowExecutionCommandAttributes{
		Namespace:         namespace,
		WorkflowId:        workflowID,
		RunId:             runID,
		Control:           cancellationID,
		ChildWorkflowOnly: false,
	}
	command := h.newCancelExternalWorkflowStateMachine(attributes, cancellationID)
	h.addCommand(command)

	return command
}

func (h *commandsHelper) handleRequestCancelExternalWorkflowExecutionInitiated(initiatedeventID int64, workflowID, cancellationID string) {
	if h.isCancelExternalWorkflowEventForChildWorkflow(cancellationID) {
		// this is cancellation for child workflow only
		command := h.getCommand(makeCommandID(commandTypeChildWorkflow, workflowID))
		command.handleCancelInitiatedEvent()
	} else {
		// this is cancellation for external workflow
		h.scheduledEventIDToCancellationID[initiatedeventID] = cancellationID
		command := h.getCommand(makeCommandID(commandTypeCancellation, cancellationID))
		command.handleInitiatedEvent()
	}
}

func (h *commandsHelper) handleExternalWorkflowExecutionCancelRequested(initiatedeventID int64, workflowID string) (bool, commandStateMachine) {
	var command commandStateMachine
	cancellationID, isExternal := h.scheduledEventIDToCancellationID[initiatedeventID]
	if !isExternal {
		command = h.getCommand(makeCommandID(commandTypeChildWorkflow, workflowID))
		asChildWfCmd := command.(*childWorkflowCommandStateMachine)
		asChildWfCmd.handleExternalWorkflowExecutionCancelRequested()
	} else {
		// this is cancellation for external workflow
		command = h.getCommand(makeCommandID(commandTypeCancellation, cancellationID))
		command.handleCompletionEvent()
	}
	return isExternal, command
}

func (h *commandsHelper) handleRequestCancelExternalWorkflowExecutionFailed(initiatedeventID int64, workflowID string) (bool, commandStateMachine) {
	var command commandStateMachine
	cancellationID, isExternal := h.scheduledEventIDToCancellationID[initiatedeventID]
	if !isExternal {
		// this is cancellation for child workflow only
		command = h.getCommand(makeCommandID(commandTypeChildWorkflow, workflowID))
		command.handleCancelFailedEvent()
	} else {
		// this is cancellation for external workflow
		command = h.getCommand(makeCommandID(commandTypeCancellation, cancellationID))
		command.handleCompletionEvent()
	}
	return isExternal, command
}

func (h *commandsHelper) signalExternalWorkflowExecution(
	namespace string,
	workflowID string,
	runID string,
	signalName string,
	input *commonpb.Payloads,
	header *commonpb.Header,
	signalID string,
	childWorkflowOnly bool,
) commandStateMachine {
	attributes := &commandpb.SignalExternalWorkflowExecutionCommandAttributes{
		Namespace: namespace,
		Execution: &commonpb.WorkflowExecution{
			WorkflowId: workflowID,
			RunId:      runID,
		},
		SignalName:        signalName,
		Input:             input,
		Control:           signalID,
		ChildWorkflowOnly: childWorkflowOnly,
		Header:            header,
	}
	command := h.newSignalExternalWorkflowStateMachine(attributes, signalID)
	h.addCommand(command)
	return command
}

func (h *commandsHelper) addProtocolMessage(msgID string) commandStateMachine {
	cmd := createNewCommand(enumspb.COMMAND_TYPE_PROTOCOL_MESSAGE)
	cmd.Attributes = &commandpb.Command_ProtocolMessageCommandAttributes{
		ProtocolMessageCommandAttributes: &commandpb.ProtocolMessageCommandAttributes{MessageId: msgID},
	}
	sm := &completeOnSendStateMachine{
		naiveCommandStateMachine: h.newNaiveCommandStateMachine(commandTypeProtocolMessage, msgID, cmd),
	}
	h.addCommand(sm)
	return sm
}

func (h *commandsHelper) upsertSearchAttributes(upsertID string, searchAttr *commonpb.SearchAttributes) commandStateMachine {
	attributes := &commandpb.UpsertWorkflowSearchAttributesCommandAttributes{
		SearchAttributes: searchAttr,
	}
	command := h.newUpsertSearchAttributesStateMachine(attributes, upsertID)
	h.addCommand(command)
	return command
}

func (h *commandsHelper) modifyProperties(changeID string, memo *commonpb.Memo) commandStateMachine {
	attributes := &commandpb.ModifyWorkflowPropertiesCommandAttributes{
		UpsertedMemo: memo,
	}
	command := h.newModifyPropertiesStateMachine(attributes, changeID)
	h.addCommand(command)
	return command
}

func (h *commandsHelper) handleSignalExternalWorkflowExecutionInitiated(initiatedEventID int64, signalID string) {
	h.scheduledEventIDToSignalID[initiatedEventID] = signalID
	command := h.getCommand(makeCommandID(commandTypeSignal, signalID))
	command.handleInitiatedEvent()
}

func (h *commandsHelper) handleSignalExternalWorkflowExecutionCompleted(initiatedEventID int64) commandStateMachine {
	command := h.getCommand(makeCommandID(commandTypeSignal, h.getSignalID(initiatedEventID)))
	command.handleCompletionEvent()
	return command
}

func (h *commandsHelper) handleSignalExternalWorkflowExecutionFailed(initiatedEventID int64) commandStateMachine {
	command := h.getCommand(makeCommandID(commandTypeSignal, h.getSignalID(initiatedEventID)))
	command.handleCompletionEvent()
	return command
}

func (h *commandsHelper) getSignalID(initiatedEventID int64) string {
	signalID, ok := h.scheduledEventIDToSignalID[initiatedEventID]
	if !ok {
		panic(fmt.Sprintf("unable to find signalID for initiatedEventID: %v", initiatedEventID))
	}
	return signalID
}

func (h *commandsHelper) startTimer(attributes *commandpb.StartTimerCommandAttributes) commandStateMachine {
	command := h.newTimerCommandStateMachine(attributes)
	h.addCommand(command)
	return command
}

func (h *commandsHelper) cancelTimer(timerID TimerID) commandStateMachine {
	command := h.getCommand(makeCommandID(commandTypeTimer, timerID.id))
	command.cancel()

	return command
}

func (h *commandsHelper) handleTimerClosed(timerID string) commandStateMachine {
	command := h.getCommand(makeCommandID(commandTypeTimer, timerID))
	// If, for whatever reason, we were going to send a timer cancel command, don't do that anymore
	// since we already know the timer is resolved.
	possibleCancelID := makeCommandID(commandTypeCancelTimer, timerID)
	h.removeCancelOfResolvedCommand(possibleCancelID)
	command.handleCompletionEvent()
	return command
}

func (h *commandsHelper) handleTimerStarted(timerID string) {
	command := h.getCommand(makeCommandID(commandTypeTimer, timerID))
	command.handleInitiatedEvent()
}

func (h *commandsHelper) handleTimerCanceled(timerID string) {
	command := h.getCommand(makeCommandID(commandTypeTimer, timerID))
	command.handleCanceledEvent()
}

func (h *commandsHelper) handleChildWorkflowExecutionStarted(workflowID string) commandStateMachine {
	command := h.getCommand(makeCommandID(commandTypeChildWorkflow, workflowID))
	command.handleStartedEvent()
	return command
}

func (h *commandsHelper) handleChildWorkflowExecutionClosed(workflowID string) commandStateMachine {
	command := h.getCommand(makeCommandID(commandTypeChildWorkflow, workflowID))
	command.handleCompletionEvent()
	return command
}

func (h *commandsHelper) handleChildWorkflowExecutionCanceled(workflowID string) commandStateMachine {
	command := h.getCommand(makeCommandID(commandTypeChildWorkflow, workflowID))
	command.handleCanceledEvent()
	return command
}

func (h *commandsHelper) getCommands(markAsSent bool) []*commandpb.Command {
	var result []*commandpb.Command
	for curr := h.orderedCommands.Front(); curr != nil; {
		next := curr.Next() // get next item here as we might need to remove curr in the loop
		d := curr.Value.(commandStateMachine)
		command := d.getCommand()
		if command != nil {
			result = append(result, command)
		}

		if markAsSent {
			d.handleCommandSent()
		}

		// remove completed command state machines
		if d.getState() == commandStateCompleted {
			h.orderedCommands.Remove(curr)
			delete(h.commands, d.getID())
		}

		curr = next
	}

	return result
}

func (h *commandsHelper) isCancelExternalWorkflowEventForChildWorkflow(cancellationID string) bool {
	// the cancellationID, i.e. Control in RequestCancelExternalWorkflowExecutionInitiatedEventAttributes
	// will be empty if the event is for child workflow.
	// for cancellation external workflow, Control in RequestCancelExternalWorkflowExecutionInitiatedEventAttributes
	// will have a client generated sequence ID
	return len(cancellationID) == 0
}

func (e *childWorkflowExistsWithId) Error() string {
	return fmt.Sprintf("child workflow already exists with id: %v", e.id)
}
