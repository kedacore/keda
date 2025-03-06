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
	protocolpb "go.temporal.io/api/protocol/v1"
)

type eventMsgIndex []*protocolpb.Message

// indexMessagesByEventID creates an index over a set of input messages that
// allows for access to messages with an event ID less than or equal to a
// specific upper bound. The order of messages with the same event ID will be
// preserved.
func indexMessagesByEventID(msgs []*protocolpb.Message) *eventMsgIndex {
	emi := eventMsgIndex(msgs)
	return &emi
}

// takeLTE removes and returns the messages in this index that have an event ID
// less than or equal to the input argument.
func (emi *eventMsgIndex) takeLTE(eventID int64) []*protocolpb.Message {
	n := 0
	var out []*protocolpb.Message
	for _, msg := range *emi {
		if msg.GetEventId() > eventID {
			(*emi)[n] = msg
			n++
		} else {
			out = append(out, msg)
		}
	}
	*emi = (*emi)[:n]
	return out
}
