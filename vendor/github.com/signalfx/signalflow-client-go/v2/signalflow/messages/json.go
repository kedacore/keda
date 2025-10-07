// Copyright Splunk Inc.
// SPDX-License-Identifier: Apache-2.0

package messages

import (
	"encoding/json"
)

func parseJSONMessage(baseMessage Message, msg []byte) (JSONMessage, error) {
	var out JSONMessage
	switch baseMessage.Type() {
	case AuthenticatedType:
		out = &AuthenticatedMessage{}
	case ControlMessageType:
		var base BaseControlMessage
		if err := json.Unmarshal(msg, &base); err != nil {
			return nil, err
		}

		switch base.Event {
		case JobStartEvent:
			out = &JobStartControlMessage{}
		case EndOfChannelEvent:
			out = &EndOfChannelControlMessage{}
		case ChannelAbortEvent:
			out = &ChannelAbortControlMessage{}
		default:
			return &base, nil
		}
	case ErrorType:
		out = &ErrorMessage{}
	case MetadataType:
		out = &MetadataMessage{}
	case ExpiredTSIDType:
		out = &ExpiredTSIDMessage{}
	case MessageType:
		out = &InfoMessage{}
	case EventType:
		out = &EventMessage{}
	default:
		out = &BaseJSONMessage{}
	}
	err := json.Unmarshal(msg, out)
	out.JSONBase().rawMessage = msg
	return out, err
}
