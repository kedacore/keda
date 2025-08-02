package internal

import (
	"math"

	"go.temporal.io/api/workflowservice/v1"
)

// sdkFlag represents a flag used to help version the sdk internally to make breaking changes
// in workflow logic.
type sdkFlag uint32

const (
	SDKFlagUnset sdkFlag = 0
	// LimitChangeVersionSASize will limit the search attribute size of TemporalChangeVersion to 2048 when
	// calling GetVersion. If the limit is exceeded the search attribute is not updated.
	SDKFlagLimitChangeVersionSASize = 1
	// SDKFlagChildWorkflowErrorExecution return errors to child workflow execution future if the child workflow would
	// fail in the synchronous path.
	SDKFlagChildWorkflowErrorExecution = 2
	// SDKFlagProtocolMessageCommand uses ProtocolMessageCommands inserted into
	// a workflow task response's command set to order messages with respect to
	// commands.
	SDKFlagProtocolMessageCommand = 3
	// SDKPriorityUpdateHandling will cause update request to be handled before the main workflow method.
	// It will also cause the SDK to immediately handle updates when a handler is registered.
	SDKPriorityUpdateHandling = 4
	// SDKFlagBlockedSelectorSignalReceive will cause a signal to not be lost
	// when the Default path is blocked.
	SDKFlagBlockedSelectorSignalReceive = 5
	SDKFlagUnknown                      = math.MaxUint32
)

// unblockSelectorSignal exists to allow us to configure the default behavior of
// SDKFlagBlockedSelectorSignalReceive. This is primarily useful with tests.
var unblockSelectorSignal = true

func sdkFlagFromUint(value uint32) sdkFlag {
	switch value {
	case uint32(SDKFlagUnset):
		return SDKFlagUnset
	case uint32(SDKFlagLimitChangeVersionSASize):
		return SDKFlagLimitChangeVersionSASize
	case uint32(SDKFlagChildWorkflowErrorExecution):
		return SDKFlagChildWorkflowErrorExecution
	case uint32(SDKFlagProtocolMessageCommand):
		return SDKFlagProtocolMessageCommand
	case uint32(SDKPriorityUpdateHandling):
		return SDKPriorityUpdateHandling
	case uint32(SDKFlagBlockedSelectorSignalReceive):
		return SDKFlagBlockedSelectorSignalReceive
	default:
		return SDKFlagUnknown
	}
}

func (f sdkFlag) isValid() bool {
	return f != SDKFlagUnset && f != SDKFlagUnknown
}

// sdkFlags represents all the flags that are currently set in a workflow execution.
type sdkFlags struct {
	capabilities *workflowservice.GetSystemInfoResponse_Capabilities
	// Flags that have been received from the server
	currentFlags map[sdkFlag]bool
	// Flags that have been set this WFT that have not been sent to the server.
	// Keep track of them separately so we know what to send to the server.
	newFlags map[sdkFlag]bool
}

func newSDKFlags(capabilities *workflowservice.GetSystemInfoResponse_Capabilities) *sdkFlags {
	return &sdkFlags{
		capabilities: capabilities,
		currentFlags: make(map[sdkFlag]bool),
		newFlags:     make(map[sdkFlag]bool),
	}
}

// tryUse returns true if this flag may currently be used. If record is true, always returns
// true and records the flag as being used.
func (sf *sdkFlags) tryUse(flag sdkFlag, record bool) bool {
	if !sf.capabilities.GetSdkMetadata() {
		return false
	}

	if record && !sf.currentFlags[flag] {
		// Only set new flags
		sf.newFlags[flag] = true
		return true
	} else {
		return sf.currentFlags[flag]
	}
}

// getFlag returns true if the flag is currently set.
func (sf *sdkFlags) getFlag(flag sdkFlag) bool {
	return sf.currentFlags[flag] || sf.newFlags[flag]
}

// set marks a flag as in current use regardless of replay status.
func (sf *sdkFlags) set(flags ...sdkFlag) {
	if !sf.capabilities.GetSdkMetadata() {
		return
	}
	for _, flag := range flags {
		sf.currentFlags[flag] = true
	}
}

// markSDKFlagsSent marks all sdk flags as sent to the server.
func (sf *sdkFlags) markSDKFlagsSent() {
	for flag := range sf.newFlags {
		sf.currentFlags[flag] = true
	}
	sf.newFlags = make(map[sdkFlag]bool)
}

// gatherNewSDKFlags returns all sdkFlags set since the last call to markSDKFlagsSent.
func (sf *sdkFlags) gatherNewSDKFlags() []sdkFlag {
	flags := make([]sdkFlag, 0, len(sf.newFlags))
	for flag := range sf.newFlags {
		flags = append(flags, flag)
	}
	return flags
}

// SetUnblockSelectorSignal toggles the flag to unblock the selector signal.
// For test use only.
func SetUnblockSelectorSignal(unblockSignal bool) {
	unblockSelectorSignal = unblockSignal
}
