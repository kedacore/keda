// Copyright Splunk Inc.
// SPDX-License-Identifier: Apache-2.0

package messages

// The event types used in the control-message messages.  This are not used for
// "event" type messages.
const (
	StreamStartEvent  = "STREAM_START"
	JobStartEvent     = "JOB_START"
	JobProgressEvent  = "JOB_PROGRESS"
	ChannelAbortEvent = "CHANNEL_ABORT"
	EndOfChannelEvent = "END_OF_CHANNEL"
)

type BaseControlMessage struct {
	BaseJSONChannelMessage
	TimestampedMessage
	Event string `json:"event"`
}

type JobStartControlMessage struct {
	BaseControlMessage
	Handle string `json:"handle"`
}

type EndOfChannelControlMessage struct {
	BaseControlMessage
}

type ChannelAbortControlMessage struct {
	BaseControlMessage
}
