// Package errors provides error types for specific error scenarios.
package errors

import (
	"fmt"
	"net/http"
)

// NewNotFound returns a new instance of NotFound with an optional custom error message.
func NewNotFound(err string) *NotFound {
	e := NotFound{
		err: err,
	}

	return &e
}

// NewNotFoundf returns a new instance of NotFound
// with an optional formatted custom error message.
func NewNotFoundf(format string, args ...interface{}) *NotFound {
	return NewNotFound(fmt.Sprintf(format, args...))
}

// NotFound is returned when the target resource cannot be located.
type NotFound struct {
	err string
}

func (e *NotFound) Error() string {
	if e.err == "" {
		return "resource not found"
	}

	return e.err
}

// NewUnexpectedStatusCode returns a new instance of UnexpectedStatusCode
// with an optional custom message.
func NewUnexpectedStatusCode(statusCode int, err string) *UnexpectedStatusCode {
	return &UnexpectedStatusCode{
		err:        err,
		statusCode: statusCode,
	}
}

// NewUnexpectedStatusCodef returns a new instance of UnexpectedStatusCode
// with an optional formatted custom message.
func NewUnexpectedStatusCodef(statusCode int, format string, args ...interface{}) *UnexpectedStatusCode {
	return NewUnexpectedStatusCode(statusCode, fmt.Sprintf(format, args...))
}

// UnexpectedStatusCode is returned when an unexpected status code is returned
// from New Relic's APIs.
type UnexpectedStatusCode struct {
	err        string
	statusCode int
}

func (e *UnexpectedStatusCode) Error() string {
	msg := fmt.Sprintf("%d response returned", e.statusCode)

	if e.err != "" {
		msg += fmt.Sprintf(": %s", e.err)
	}

	return msg
}

// NewUnauthorizedError returns a new instance of UnauthorizedError
// with an optional custom message.
func NewUnauthorizedError() *UnauthorizedError {
	return &UnauthorizedError{
		err:        "Invalid credentials provided. Missing API key or an invalid API key was provided.",
		statusCode: http.StatusUnauthorized,
	}
}

// UnauthorizedError is returned when a 401 HTTP status code is returned
// from New Relic's APIs.
type UnauthorizedError struct {
	err        string
	statusCode int
}

func (e *UnauthorizedError) Error() string {
	msg := fmt.Sprintf("%d response returned", e.statusCode)

	if e.err != "" {
		msg += fmt.Sprintf(": %s", e.err)
	}

	return msg
}

// NewMaxRetriesReached returns a new instance of MaxRetriesReached with an optional custom error message.
func NewMaxRetriesReached(err string) *MaxRetriesReached {
	e := MaxRetriesReached{
		err: err,
	}

	return &e
}

// NewMaxRetriesReachedf returns a new instance of MaxRetriesReached
// with an optional formatted custom error message.
func NewMaxRetriesReachedf(format string, args ...interface{}) *MaxRetriesReached {
	return NewMaxRetriesReached(fmt.Sprintf(format, args...))
}

// MaxRetriesReached is returned when the target resource cannot be located.
type MaxRetriesReached struct {
	err string
}

func (e *MaxRetriesReached) Error() string {
	return fmt.Sprintf("maximum retries reached: %s", e.err)
}

// NewInvalidInput returns a new instance of InvalidInput with an optional custom error message.
func NewInvalidInput(err string) *InvalidInput {
	e := InvalidInput{
		err: err,
	}

	return &e
}

// NewInvalidInputf returns a new instance of InvalidInput
// with an optional formatted custom error message.
func NewInvalidInputf(format string, args ...interface{}) *InvalidInput {
	return NewInvalidInput(fmt.Sprintf(format, args...))
}

// InvalidInput is returned when the user input is invalid.
type InvalidInput struct {
	err string
}

func (e *InvalidInput) Error() string {
	if e.err == "" {
		return "invalid input error"
	}

	return e.err
}

// PaymentRequiredError is returned when a 402 HTTP status code is returned
// from New Relic's APIs.
type PaymentRequiredError struct {
	err        string
	statusCode int
}

func (e *PaymentRequiredError) Error() string {
	return e.err
}

// NewPaymentRequiredError returns a new instance of PaymentRequiredError
// with an optional custom message.
func NewPaymentRequiredError() *PaymentRequiredError {
	return &PaymentRequiredError{
		err:        http.StatusText(http.StatusPaymentRequired),
		statusCode: http.StatusPaymentRequired,
	}
}
