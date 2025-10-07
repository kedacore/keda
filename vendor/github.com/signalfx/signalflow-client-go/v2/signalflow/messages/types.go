// Copyright Splunk Inc.
// SPDX-License-Identifier: Apache-2.0

package messages

import (
	"encoding/json"
	"fmt"
	"time"
)

// See https://developers.signalfx.com/signalflow_analytics/rest_api_messages/stream_messages_specification.html
const (
	AuthenticatedType  = "authenticated"
	ControlMessageType = "control-message"
	ErrorType          = "error"
	MetadataType       = "metadata"
	MessageType        = "message"
	DataType           = "data"
	EventType          = "event"
	WebsocketErrorType = "websocket-error"
	ExpiredTSIDType    = "expired-tsid"
)

type BaseMessage struct {
	Typ string `json:"type"`
}

func (bm *BaseMessage) Type() string {
	return bm.Typ
}

func (bm *BaseMessage) String() string {
	return fmt.Sprintf("%s message", bm.Typ)
}

func (bm *BaseMessage) Base() *BaseMessage {
	return bm
}

var _ Message = &BaseMessage{}

type Message interface {
	Type() string
	Base() *BaseMessage
}

type ChannelMessage interface {
	Channel() string
}

type BaseChannelMessage struct {
	Chan string `json:"channel,omitempty"`
}

func (bcm *BaseChannelMessage) Channel() string {
	return bcm.Chan
}

type JSONMessage interface {
	Message
	JSONBase() *BaseJSONMessage
	RawData() map[string]interface{}
}

type BaseJSONMessage struct {
	BaseMessage
	rawMessage []byte
	rawData    map[string]interface{}
}

func (j *BaseJSONMessage) JSONBase() *BaseJSONMessage {
	return j
}

// The raw message deserialized from JSON. Only applicable for JSON
// Useful if the message type doesn't have a concrete struct type implemented
// in this library (e.g. due to an upgrade to the SignalFlow protocol).
func (j *BaseJSONMessage) RawData() map[string]interface{} {
	if j.rawData == nil {
		if err := json.Unmarshal(j.rawMessage, &j.rawData); err != nil {
			// This shouldn't ever error since it wouldn't have been initially
			// deserialized if there were parse errors.  But in case it does
			// just return nil.
			return nil
		}
	}
	return j.rawData
}

func (j *BaseJSONMessage) String() string {
	return j.BaseMessage.String() + string(j.rawMessage)
}

type BaseJSONChannelMessage struct {
	BaseJSONMessage
	BaseChannelMessage
}

func (j *BaseJSONChannelMessage) String() string {
	return string(j.BaseJSONMessage.rawMessage)
}

type TimestampedMessage struct {
	TimestampMillis uint64 `json:"timestampMs"`
}

func (m *TimestampedMessage) Timestamp() time.Time {
	return time.Unix(0, int64(m.TimestampMillis*uint64(time.Millisecond)))
}

type AuthenticatedMessage struct {
	BaseJSONMessage
	OrgID  string `json:"orgId"`
	UserID string `json:"userId"`
}

// The way to distinguish between JSON and binary messages is the websocket
// message type.
func ParseMessage(msg []byte, isText bool) (Message, error) {
	if isText {
		var baseMessage BaseMessage
		if err := json.Unmarshal(msg, &baseMessage); err != nil {
			return nil, fmt.Errorf("couldn't unmarshal JSON websocket message: %w", err)
		}
		return parseJSONMessage(&baseMessage, msg)
	}
	return parseBinaryMessage(msg)
}
