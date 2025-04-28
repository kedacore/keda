//
// DISCLAIMER
//
// Copyright 2017-2025 ArangoDB GmbH, Cologne, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright holder is ArangoDB GmbH, Cologne, Germany
//
// Author Ewout Prangsma
//

package driver

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
)

const (
	// general errors
	ErrNotImplemented = 9
	ErrForbidden      = 11
	ErrDisabled       = 36

	// HTTP error status codes
	ErrHttpForbidden = 403
	ErrHttpInternal  = 501

	// Internal ArangoDB storage errors
	ErrArangoReadOnly = 1004

	// External ArangoDB storage errors
	ErrArangoCorruptedDatafile    = 1100
	ErrArangoIllegalParameterFile = 1101
	ErrArangoCorruptedCollection  = 1102
	ErrArangoFileSystemFull       = 1104
	ErrArangoDataDirLocked        = 1107

	// General ArangoDB storage errors
	ErrArangoConflict                 = 1200
	ErrArangoDocumentNotFound         = 1202
	ErrArangoDataSourceNotFound       = 1203
	ErrArangoIllegalName              = 1208
	ErrArangoUniqueConstraintViolated = 1210
	ErrArangoDatabaseNotFound         = 1228
	ErrArangoDatabaseNameInvalid      = 1229

	// ArangoDB cluster errors
	ErrClusterReplicationWriteConcernNotFulfilled = 1429
	ErrClusterLeadershipChallengeOngoing          = 1495
	ErrClusterNotLeader                           = 1496

	// User management errors
	ErrUserDuplicate = 1702
)

// ArangoError is a Go error with arangodb specific error information.
type ArangoError struct {
	HasError     bool   `json:"error"`
	Code         int    `json:"code"`
	ErrorNum     int    `json:"errorNum"`
	ErrorMessage string `json:"errorMessage"`
}

// Error returns the error message of an ArangoError.
func (ae ArangoError) Error() string {
	if ae.ErrorMessage != "" {
		return ae.ErrorMessage
	}
	return fmt.Sprintf("ArangoError: Code %d, ErrorNum %d", ae.Code, ae.ErrorNum)
}

// Timeout returns true when the given error is a timeout error.
func (ae ArangoError) Timeout() bool {
	return ae.HasError && (ae.Code == http.StatusRequestTimeout || ae.Code == http.StatusGatewayTimeout)
}

// Temporary returns true when the given error is a temporary error.
func (ae ArangoError) Temporary() bool {
	return ae.HasError && ae.Code == http.StatusServiceUnavailable
}

// newArangoError creates a new ArangoError with given values.
func newArangoError(code, errorNum int, errorMessage string) error {
	return ArangoError{
		HasError:     true,
		Code:         code,
		ErrorNum:     errorNum,
		ErrorMessage: errorMessage,
	}
}

// IsArangoError returns true when the given error is an ArangoError.
func IsArangoError(err error) bool {
	ae, ok := Cause(err).(ArangoError)
	return ok && ae.HasError
}

// AsArangoError returns true when the given error is an ArangoError together with an object.
func AsArangoError(err error) (ArangoError, bool) {
	ae, ok := Cause(err).(ArangoError)
	if ok {
		return ae, true
	} else {
		return ArangoError{}, false
	}
}

// IsArangoErrorWithCode returns true when the given error is an ArangoError and its Code field is equal to the given code.
func IsArangoErrorWithCode(err error, code int) bool {
	ae, ok := Cause(err).(ArangoError)
	return ok && ae.Code == code
}

// IsArangoErrorWithErrorNum returns true when the given error is an ArangoError and its ErrorNum field is equal to one of the given numbers.
func IsArangoErrorWithErrorNum(err error, errorNum ...int) bool {
	ae, ok := Cause(err).(ArangoError)
	if !ok {
		return false
	}
	for _, x := range errorNum {
		if ae.ErrorNum == x {
			return true
		}
	}
	return false
}

// IsInvalidRequest returns true if the given error is an ArangoError with code 400, indicating an invalid request.
func IsInvalidRequest(err error) bool {
	return IsArangoErrorWithCode(err, http.StatusBadRequest)

}

// IsUnauthorized returns true if the given error is an ArangoError with code 401, indicating an unauthorized request.
func IsUnauthorized(err error) bool {
	return IsArangoErrorWithCode(err, http.StatusUnauthorized)
}

// IsForbidden returns true if the given error is an ArangoError with code 403, indicating a forbidden request.
func IsForbidden(err error) bool {
	return IsArangoErrorWithCode(err, http.StatusForbidden)
}

// Deprecated: Use IsNotFoundGeneral instead.
//
// For ErrArangoDocumentNotFound error there is a chance that we get a different HTTP code if the API requires an existing document as input, which is not found.
//
// IsNotFound returns true if the given error is an ArangoError with code 404, indicating an object not found.
func IsNotFound(err error) bool {
	return IsArangoErrorWithCode(err, http.StatusNotFound) ||
		IsArangoErrorWithErrorNum(err, ErrArangoDocumentNotFound, ErrArangoDataSourceNotFound)
}

// IsNotFoundGeneral returns true if the given error is an ArangoError with code 404, indicating an object is not found.
func IsNotFoundGeneral(err error) bool {
	return IsArangoErrorWithCode(err, http.StatusNotFound)
}

// IsDataSourceOrDocumentNotFound returns true if the given error is an Arango storage error, indicating an object is not found.
func IsDataSourceOrDocumentNotFound(err error) bool {
	return IsArangoErrorWithCode(err, http.StatusNotFound) &&
		IsArangoErrorWithErrorNum(err, ErrArangoDocumentNotFound, ErrArangoDataSourceNotFound)
}

// IsExternalStorageError returns true if ArangoDB is having an error with accessing or writing to storage.
func IsExternalStorageError(err error) bool {
	return IsArangoErrorWithErrorNum(
		err,
		ErrArangoCorruptedDatafile,
		ErrArangoIllegalParameterFile,
		ErrArangoCorruptedCollection,
		ErrArangoFileSystemFull,
		ErrArangoDataDirLocked,
	)
}

// IsConflict returns true if the given error is an ArangoError with code 409, indicating a conflict.
func IsConflict(err error) bool {
	return IsArangoErrorWithCode(err, http.StatusConflict) || IsArangoErrorWithErrorNum(err, ErrUserDuplicate)
}

// IsPreconditionFailed returns true if the given error is an ArangoError with code 412, indicating a failed precondition.
func IsPreconditionFailed(err error) bool {
	return IsArangoErrorWithCode(err, http.StatusPreconditionFailed) ||
		IsArangoErrorWithErrorNum(err, ErrArangoConflict, ErrArangoUniqueConstraintViolated)
}

// IsNoLeader returns true if the given error is an ArangoError with code 503 error number 1496.
func IsNoLeader(err error) bool {
	return IsArangoErrorWithCode(err, http.StatusServiceUnavailable) && IsArangoErrorWithErrorNum(err, ErrClusterNotLeader)
}

// IsNoLeaderOrOngoing return true if the given error is an ArangoError with code 503 and error number 1496 or 1495
func IsNoLeaderOrOngoing(err error) bool {
	return IsArangoErrorWithCode(err, http.StatusServiceUnavailable) &&
		IsArangoErrorWithErrorNum(err, ErrClusterLeadershipChallengeOngoing, ErrClusterNotLeader)
}

// InvalidArgumentError is returned when a go function argument is invalid.
type InvalidArgumentError struct {
	Message string
}

// Error implements the error interface for InvalidArgumentError.
func (e InvalidArgumentError) Error() string {
	return e.Message
}

// IsInvalidArgument returns true if the given error is an InvalidArgumentError.
func IsInvalidArgument(err error) bool {
	_, ok := Cause(err).(InvalidArgumentError)
	return ok
}

// NoMoreDocumentsError is returned by Cursor's, when an attempt is made to read documents when there are no more.
type NoMoreDocumentsError struct{}

// Error implements the error interface for NoMoreDocumentsError.
func (e NoMoreDocumentsError) Error() string {
	return "no more documents"
}

// IsNoMoreDocuments returns true if the given error is an NoMoreDocumentsError.
func IsNoMoreDocuments(err error) bool {
	_, ok := Cause(err).(NoMoreDocumentsError)
	return ok
}

// A ResponseError is returned when a request was completely written to a server, but
// the server did not respond, or some kind of network error occurred during the response.
type ResponseError struct {
	Err error
}

// Error returns the Error() result of the underlying error.
func (e *ResponseError) Error() string {
	return e.Err.Error()
}

// IsResponse returns true if the given error is (or is caused by) a ResponseError.
func IsResponse(err error) bool {
	return isCausedBy(err, func(e error) bool { _, ok := e.(*ResponseError); return ok })
}

// IsCanceled returns true if the given error is the result on a cancelled context.
func IsCanceled(err error) bool {
	return isCausedBy(err, func(e error) bool { return e == context.Canceled })
}

// IsTimeout returns true if the given error is the result on a deadline that has been exceeded.
func IsTimeout(err error) bool {
	return isCausedBy(err, func(e error) bool { return e == context.DeadlineExceeded })
}

// isCausedBy returns true if the given error returns true on the given predicate,
// unwrapping various standard library error wrappers.
func isCausedBy(err error, p func(error) bool) bool {
	if p(err) {
		return true
	}
	err = Cause(err)
	for {
		if p(err) {
			return true
		} else if err == nil {
			return false
		}
		if xerr, ok := err.(*ResponseError); ok {
			err = xerr.Err
		} else if xerr, ok := err.(*url.Error); ok {
			err = xerr.Err
		} else if xerr, ok := err.(*net.OpError); ok {
			err = xerr.Err
		} else if xerr, ok := err.(*os.SyscallError); ok {
			err = xerr.Err
		} else {
			return false
		}
	}
}

var (
	// WithStack is called on every return of an error to add stacktrace information to the error.
	// When setting this function, also set the Cause function.
	// The interface of this function is compatible with functions in github.com/pkg/errors.
	WithStack = func(err error) error { return err }
	// Cause is used to get the root cause of the given error.
	// The interface of this function is compatible with functions in github.com/pkg/errors.
	Cause = func(err error) error { return err }
)

// ErrorSlice is a slice of errors
type ErrorSlice []error

// FirstNonNil returns the first error in the slice that is not nil.
// If all errors in the slice are nil, nil is returned.
func (l ErrorSlice) FirstNonNil() error {
	for _, e := range l {
		if e != nil {
			return e
		}
	}
	return nil
}
