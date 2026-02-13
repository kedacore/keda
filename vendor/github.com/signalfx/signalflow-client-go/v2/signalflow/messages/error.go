// Copyright Splunk Inc.
// SPDX-License-Identifier: Apache-2.0

package messages

type ErrorContext struct {
	BindingName string      `json:"bindingName"`
	Column      int         `json:"column"`
	Line        int         `json:"line"`
	ProgramText string      `json:"programText"`
	Reference   string      `json:"reference"`
	Traceback   interface{} `json:"traceback"`
}

type ErrorMessage struct {
	BaseJSONChannelMessage

	Context   ErrorContext `json:"context"`
	Error     int          `json:"error"`
	ErrorType string       `json:"errorType"`
	Message   string       `json:"message"`
}
