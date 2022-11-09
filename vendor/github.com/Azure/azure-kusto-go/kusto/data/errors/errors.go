/*
Package errors provides the error package for Kusto. It wraps all errors for Kusto. No error should be
generated that doesn't come from this package. This borrows heavily fron the Upspin errors paper written
by Rob Pike. See: https://commandcenter.blogspot.com/2017/12/error-handling-in-upspin.html
Key differences are that we support wrapped errors and the 1.13 Unwrap/Is/As additions to the go stdlib
errors package and this is tailored for Kusto and not Upspin.

Usage is simply to pass an Op, a Kind, and either a standard error to be wrapped or string that will become
a string error.  See examples included in the file for more details.
*/
package errors

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
)

// Separator is the string used to separate nested errors. By
// default, to make errors easier on the eye, nested errors are
// indented on a new line. A server may instead choose to keep each
// error on a single line by modifying the separator string, perhaps
// to ":: ".
var Separator = ":\n\t"

// Op field denotes the operation being performed.
type Op uint16

//go:generate stringer -type Op
const (
	OpUnknown      Op = 0 // OpUnknown indicates that the operation that caused the problem is unknown.
	OpQuery        Op = 1 // OpQuery indicates that a Query() call is being made.
	OpMgmt         Op = 2 // OpMgmt indicates that a Mgmt() call is being made.
	OpServConn     Op = 3 // OpServConn indicates that the client is attempting to connect to the service.
	OpIngestStream Op = 4 // OpIngestStream indicates the client is making a streaming ingestion call.
	OpFileIngest   Op = 5 // OpFileIngest indicates the client is making a file ingestion call.
)

// Kind field classifies the error as one of a set of standard conditions.
type Kind uint16

//go:generate stringer -type Kind
const (
	KOther           Kind = 0 // Other indicates the error kind was not defined.
	KIO              Kind = 1 // External I/O error such as network failure.
	KInternal        Kind = 2 // Internal error or inconsistency at the server.
	KDBNotExist      Kind = 3 // Database does not exist.
	KTimeout         Kind = 4 // The request timed out.
	KLimitsExceeded  Kind = 5 // The request was too large.
	KClientArgs      Kind = 6 // The client supplied some type of arg(s) that were invalid.
	KHTTPError       Kind = 7 // The HTTP client gave some type of error. This wraps the http library error types.
	KBlobstore       Kind = 8 // The Blobstore API returned some type of error.
	KLocalFileSystem Kind = 9 // The local fileystem had an error. This could be permission, missing file, etc....
)

// Error is a core error for the Kusto package.
type Error struct {
	// Op is the operations that the client was trying to perform.
	Op Op
	// Kind is the error code we identify the error as.
	Kind Kind
	// Err is the error message. This may be of any error type and may also wrap errors.
	Err error

	// restErrMsg holds the body of an error messsage that was from a REST endpoint.
	restErrMsg []byte
	decoded    map[string]interface{}
	permanent  bool

	inner *Error
}

type KustoError = Error

type HttpError struct {
	KustoError
	StatusCode int
}

// UnmarshalREST will unmarshal an error message from the server if the message is in
// JSON format or will return nil. This only occurs when the error is of Kind KHTTPError
// and the server responded with a JSON error.
func (e *Error) UnmarshalREST() map[string]interface{} {
	if e.decoded != nil {
		return e.decoded
	}

	m := map[string]interface{}{}
	if err := json.Unmarshal(e.restErrMsg, &m); err != nil {
		return nil
	}

	if m != nil {
		if v, ok := m["error"]; ok {
			if errMap, ok := v.(map[string]interface{}); ok {
				if v, ok := errMap["@permanent"]; ok {
					if b, ok := v.(bool); ok {
						e.permanent = b
					}
				}
			}
		}
	}

	e.decoded = m
	return m
}

// SetNoRetry sets this error so that Retry() will always return false.
func (e *Error) SetNoRetry() *Error {
	e.permanent = true
	return e
}

// Unwrap implements "interface {Unwrap() error}" as defined internally by the go stdlib errors package.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}

	if e.inner == nil {
		return e.Err
	}

	return e.inner
}

// pad appends str to the buffer if the buffer already has some data.
func pad(b *strings.Builder, str string) {
	if b.Len() == 0 {
		return
	}
	b.WriteString(str)
}

func (e *Error) Error() string {
	b := new(strings.Builder)
	if e.Op != OpUnknown {
		pad(b, ": ")
		b.WriteString(fmt.Sprintf("Op(%s)", e.Op.String()))
	}
	if e.Kind != KOther {
		pad(b, ": ")
		b.WriteString(fmt.Sprintf("Kind(%s)", e.Kind.String()))
	}

	if e.Err != nil {
		pad(b, ": ")
		b.WriteString(e.Err.Error())
	}
	var inner = e.inner
	for {
		if inner == nil {
			break
		}
		pad(b, Separator)
		b.WriteString(inner.Err.Error())
		inner = inner.inner
	}

	if b.Len() == 0 {
		return "no error"
	}
	return b.String()
}

// Retry determines if the error is transient and the action can be retried or not.
// Some errors that can be retried, such as a timeout, may never succeed, so avoid infinite retries.
func Retry(err error) bool {
	var e *Error
	if errors.As(err, &e) {
		// e.permanent can be set multiple ways. If it is true, you can never retry.
		// If it is false, it does not necessarily mean anything, you have to go a little further.
		if e.permanent {
			return false
		}

		switch e.Kind {
		case KOther, KIO, KInternal, KDBNotExist, KLimitsExceeded, KClientArgs, KLocalFileSystem:
			return false
		case KHTTPError:
			m := e.UnmarshalREST()
			if m != nil {
				if e.permanent {
					return false
				}
			}
		}

		if e.inner != nil {
			return Retry(e.inner)
		}
		return true
	}
	return false
}

// E constructs an Error. You may pass in an Op, Kind and error.  This will strip a *errors.Error(the error in this package) if you
// pass one of its Kind and Op and wrap it in here. It will wrap a non-*Error implementation of error.
// If you want to wrap the *Error in an *Error, use W(). If you pass a nil error, it panics.
func E(o Op, k Kind, err error) *Error {
	if err == nil {
		panic("cannot pass a nil error")
	}
	return e(o, k, err)
}

// ES constructs an Error. You may pass in an Op, Kind, string and args to the string (like fmt.Sprintf).
// If the result of strings.TrimSpace(s+args) == "", it panics.
func ES(o Op, k Kind, s string, args ...interface{}) *Error {
	str := fmt.Sprintf(s, args...)
	if strings.TrimSpace(str) == "" {
		panic("errors.ES() cannot have an empty string error")
	}
	return e(k, o, str)
}

// HTTP constructs an *Error from an *http.Response and a prefix to the error message.
func HTTP(o Op, status string, statusCode int, body io.ReadCloser, prefix string) *HttpError {
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		bodyBytes = []byte(fmt.Sprintf("Failed to read body: %v", err))
	}
	e := HttpError{
		KustoError: KustoError{
			Op:         o,
			Kind:       KHTTPError,
			restErrMsg: bodyBytes,
			Err:        fmt.Errorf("%s(%s):\n%s", prefix, status, string(bodyBytes)),
		},
		StatusCode: statusCode,
	}

	e.UnmarshalREST()
	return &e
}

// e constructs an Error. You may pass in an Op, Kind, string or error.  This will strip an *Error if you
// pass if of its Kind and Op and put it in here. It will wrap a non-*Error implementation of error.
// If you want to wrap the *Error in an *Error, use W().
func e(args ...interface{}) *Error {
	if len(args) == 0 {
		panic("call to errors.E with no arguments")
	}
	e := &Error{}

	for _, arg := range args {
		switch arg := arg.(type) {
		case Op:
			e.Op = arg
		case string:
			e.Err = errors.New(arg)
		case Kind:
			e.Kind = arg
		case *Error:
			// Make a copy
			argCopy := *arg
			e.Err = argCopy.Err
		case error:
			e.Err = arg
		default:
			if err, ok := arg.(error); ok {
				e.Err = err
			} else {
				_, file, line, _ := runtime.Caller(1)
				e.Kind = KOther
				e.Err = fmt.Errorf("errors.E: bad call from %s:%d: %v, unknown type %T, value %v in error call", file, line, args, arg, arg)
				return e
			}
		}
	}

	return e
}

// W wraps error outer around inner. Both must be of type *Error or this will panic.
func W(inner error, outer error) *Error {
	o, ok := outer.(*Error)
	if !ok {
		panic("W() got an outer error that was not of type *Error")
	}
	i, ok := inner.(*Error)
	if !ok {
		panic("W() got an inner error that was not of type *Error")
	}

	o.inner = i
	return o
}

// OneToErr translates what we think is a Kusto OneApiError into an Error. If we don't recognize it, we return nil.
// This tries to wrap the internal errors, but the errors that are generated are some type of early draft of OneApiError,
// not the current spec. Because the errors we see don't conform to the current OneAPIError spec, had to take guesses on
// what we will receive. The spec says we shouldn't get a list of errors, but we do(we should get an embedded error).
// So I'm taking the guess that these are supposed to be wrapped errors.
func OneToErr(m map[string]interface{}, op Op) *Error {
	if m == nil {
		return nil
	}

	if _, ok := m["OneApiErrors"]; ok {
		var topErr *Error
		if oneErrors, ok := m["OneApiErrors"].([]interface{}); ok {
			var bottomErr *Error
			for _, oneErr := range oneErrors {
				if errMap, ok := oneErr.(map[string]interface{}); ok {
					e := oneToErr(errMap, bottomErr, op)
					if e == nil {
						continue
					}
					if topErr == nil {
						topErr = e
						bottomErr = e
						continue
					}
					bottomErr = e
				}
			}
			return topErr
		}
	}
	return nil
}

func oneToErr(m map[string]interface{}, err *Error, op Op) *Error {
	errJSON, ok := m["error"]
	if !ok {
		return nil
	}
	errMap, ok := errJSON.(map[string]interface{})
	if !ok {
		return nil
	}

	var msg string
	msgInter, ok := errMap["message"]
	if !ok {
		return nil
	}

	if msg, ok = msgInter.(string); !ok {
		return nil
	}

	var code string

	codeInter, ok := errMap["code"]
	if ok {
		codeStr, ok := codeInter.(string)
		if ok {
			code = codeStr
		}
	}

	var kind Kind
	switch code {
	case "LimitsExceeded":
		kind = KLimitsExceeded
		msg = msg + ";See https://docs.microsoft.com/en-us/azure/kusto/concepts/querylimits"
	}

	if err == nil {
		return ES(op, kind, msg)
	}

	err = W(ES(op, kind, msg), err)

	return err
}

func (e *HttpError) IsThrottled() bool {
	return e != nil && (e.StatusCode == http.StatusTooManyRequests)
}

func (e *HttpError) Error() string {
	return e.KustoError.Error()
}

func (e *HttpError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.KustoError.Unwrap()
}

func GetKustoError(err error) (*Error, bool) {
	if err, ok := err.(*Error); ok {
		return err, true
	}
	if err, ok := err.(*HttpError); ok {
		return &err.KustoError, true
	}
	return nil, false
}

type CombinedError struct {
	Errors []error
}

func (c CombinedError) Error() string {
	result := ""
	for _, err := range c.Errors {
		result += fmt.Sprintf("'%s';", err.Error())
	}
	return result
}

func GetCombinedError(errs ...error) *CombinedError {
	return &CombinedError{Errors: errs}
}
