// The MIT License
//
// Copyright (c) 2022 Temporal Technologies Inc.  All rights reserved.
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

import failurepb "go.temporal.io/api/failure/v1"

// FailureConverter is used by the sdk to serialize/deserialize errors
// that need to be sent over the wire.
// To use a custom FailureConverter, set FailureConverter in client, through client.Options.
type FailureConverter interface {
	// ErrorToFailure converts a error to a Failure proto message.
	ErrorToFailure(err error) *failurepb.Failure

	// FailureToError converts a Failure proto message to a Go Error.
	FailureToError(failure *failurepb.Failure) error
}

type encodedFailure struct {
	Message    string `json:"message"`
	StackTrace string `json:"stack_trace"`
}

// EncodeCommonFailureAttributes packs failure attributes to a payload so that they flow through a dataconverter.
func EncodeCommonFailureAttributes(dc DataConverter, failure *failurepb.Failure) error {
	var err error

	failure.EncodedAttributes, err = dc.ToPayload(encodedFailure{
		Message:    failure.Message,
		StackTrace: failure.StackTrace,
	})
	if err != nil {
		return err
	}
	failure.Message = "Encoded failure"
	failure.StackTrace = ""

	return nil
}

// DecodeCommonFailureAttributes unpacks failure attributes from a stored payload, if present.
func DecodeCommonFailureAttributes(dc DataConverter, failure *failurepb.Failure) {
	var ea encodedFailure
	if failure.GetEncodedAttributes() != nil && dc.FromPayload(failure.GetEncodedAttributes(), &ea) == nil {
		failure.Message = ea.Message
		failure.StackTrace = ea.StackTrace
	}
}
