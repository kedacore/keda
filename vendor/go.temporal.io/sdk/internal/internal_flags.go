// The MIT License
//
// Copyright (c) 2023 Temporal Technologies Inc.  All rights reserved.
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
	SDKFlagUnknown                = math.MaxUint32
)

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
	// Flags that have been recieved from the server
	currentFlags map[sdkFlag]bool
	// Flags that have been set this WFT that have not been sent to the server.
	// Keep track of them sepratly so we know what to send to the server.
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
