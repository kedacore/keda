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

package protocol

import (
	"fmt"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	protocolpb "go.temporal.io/api/protocol/v1"
)

// NameFromMessage extracts the name of the protocol to which the supplied
// message belongs.
func NameFromMessage(msg *protocolpb.Message) (string, error) {
	bodyType, err := types.AnyMessageName(msg.Body)
	if err != nil {
		return "", fmt.Errorf("unrecognized message type: %w", err)
	}
	if lastDot := strings.LastIndex(bodyType, "."); lastDot > -1 {
		bodyType = bodyType[0:lastDot]
	}
	return bodyType, nil
}

// MustMarshalAny serializes a protobuf message into an Any or panics.
func MustMarshalAny(msg proto.Message) *types.Any {
	result, err := types.MarshalAny(msg)
	if err != nil {
		panic(err)
	}
	return result
}
