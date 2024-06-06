// Copyright Splunk Inc.
// SPDX-License-Identifier: Apache-2.0

package signalflow

import (
	"encoding/json"
	"time"
)

type AuthType string

func (at AuthType) MarshalJSON() ([]byte, error) {
	return []byte(`"authenticate"`), nil
}

type AuthRequest struct {
	// This should not be set manually.
	Type AuthType `json:"type"`
	// The Auth token for the org
	Token     string `json:"token"`
	UserAgent string `json:"userAgent,omitempty"`
}

type ExecuteType string

func (ExecuteType) MarshalJSON() ([]byte, error) {
	return []byte(`"execute"`), nil
}

// See
// https://dev.splunk.com/observability/docs/signalflow/messages/websocket_request_messages#Execute-message-properties
// for details on the fields.
type ExecuteRequest struct {
	// This should not be set manually
	Type         ExecuteType   `json:"type"`
	Program      string        `json:"program"`
	Channel      string        `json:"channel"`
	Start        time.Time     `json:"-"`
	Stop         time.Time     `json:"-"`
	Resolution   time.Duration `json:"-"`
	MaxDelay     time.Duration `json:"-"`
	StartMs      int64         `json:"start"`
	StopMs       int64         `json:"stop"`
	ResolutionMs int64         `json:"resolution"`
	MaxDelayMs   int64         `json:"maxDelay"`
	Immediate    bool          `json:"immediate"`
	Timezone     string        `json:"timezone"`
}

// MarshalJSON does some assignments to allow using more native Go types for
// time/duration.
func (er ExecuteRequest) MarshalJSON() ([]byte, error) {
	if !er.Start.IsZero() {
		er.StartMs = er.Start.UnixNano() / int64(time.Millisecond)
	}
	if !er.Stop.IsZero() {
		er.StopMs = er.Stop.UnixNano() / int64(time.Millisecond)
	}
	if er.Resolution != 0 {
		er.ResolutionMs = er.Resolution.Nanoseconds() / int64(time.Millisecond)
	}
	if er.MaxDelay != 0 {
		er.MaxDelayMs = er.MaxDelay.Nanoseconds() / int64(time.Millisecond)
	}
	type alias ExecuteRequest
	return json.Marshal(alias(er))
}

type DetachType string

func (DetachType) MarshalJSON() ([]byte, error) {
	return []byte(`"detach"`), nil
}

type DetachRequest struct {
	// This should not be set manually
	Type    DetachType `json:"type"`
	Channel string     `json:"channel"`
	Reason  string     `json:"reason"`
}

type StopType string

func (StopType) MarshalJSON() ([]byte, error) {
	return []byte(`"stop"`), nil
}

type StopRequest struct {
	// This should not be set manually
	Type   StopType `json:"type"`
	Handle string   `json:"handle"`
	Reason string   `json:"reason"`
}
