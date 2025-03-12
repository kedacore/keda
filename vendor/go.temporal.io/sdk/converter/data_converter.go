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

package converter

import (
	commonpb "go.temporal.io/api/common/v1"
)

type (
	// DataConverter is used by the framework to serialize/deserialize input and output of activity/workflow
	// that need to be sent over the wire.
	// To encode/decode workflow arguments, set DataConverter in client, through client.Options.
	// To override DataConverter for specific activity or child workflow use workflow.WithDataConverter to create new Context,
	// and pass that context to ExecuteActivity/ExecuteChildWorkflow calls.
	// Temporal support using different DataConverters for different activity/childWorkflow in same workflow.
	// For advanced data converters that may exceed the deadlock detection timeout
	// for a workflow, such as ones making remote calls, use
	// workflow.DataConverterWithoutDeadlockDetection.
	DataConverter interface {
		// ToPayload converts single value to payload.
		ToPayload(value interface{}) (*commonpb.Payload, error)
		// FromPayload converts single value from payload.
		//
		// Note, values should not be reused for extraction here because merging on
		// top of existing values may result in unexpected behavior similar to
		// json.Unmarshal.
		FromPayload(payload *commonpb.Payload, valuePtr interface{}) error

		// ToPayloads converts a list of values.
		ToPayloads(value ...interface{}) (*commonpb.Payloads, error)
		// FromPayloads converts to a list of values of different types.
		// Useful for deserializing arguments of function invocations.
		//
		// Note, values should not be reused for extraction here because merging on
		// top of existing values may result in unexpected behavior similar to
		// json.Unmarshal.
		FromPayloads(payloads *commonpb.Payloads, valuePtrs ...interface{}) error

		// ToString converts payload object into human readable string.
		ToString(input *commonpb.Payload) string
		// ToStrings converts payloads object into human readable strings.
		ToStrings(input *commonpb.Payloads) []string
	}
)
