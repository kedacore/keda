//
// DISCLAIMER
//
// Copyright 2017 ArangoDB GmbH, Cologne, Germany
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

package velocypack

import (
	"errors"
	"reflect"
)

// InvalidTypeError is returned when a Slice getter is called on a slice of a different type.
type InvalidTypeError struct {
	Message string
}

// Error implements the error interface for InvalidTypeError.
func (e InvalidTypeError) Error() string {
	return e.Message
}

// IsInvalidType returns true if the given error is an InvalidTypeError.
func IsInvalidType(err error) bool {
	_, ok := Cause(err).(InvalidTypeError)
	return ok
}

var (
	// NumberOutOfRangeError indicates an out of range error.
	NumberOutOfRangeError = errors.New("number out of range")
	// IsNumberOutOfRange returns true if the given error is an NumberOutOfRangeError.
	IsNumberOutOfRange = isCausedByFunc(NumberOutOfRangeError)
	// IndexOutOfBoundsError indicates an index outside of array/object bounds.
	IndexOutOfBoundsError = errors.New("index out of range")
	// IsIndexOutOfBounds returns true if the given error is an IndexOutOfBoundsError.
	IsIndexOutOfBounds = isCausedByFunc(IndexOutOfBoundsError)
	// NeedAttributeTranslatorError indicates a lack of object key translator (smallint|uint -> string).
	NeedAttributeTranslatorError = errors.New("need attribute translator")
	// IsNeedAttributeTranslator returns true if the given error is an NeedAttributeTranslatorError.
	IsNeedAttributeTranslator = isCausedByFunc(NeedAttributeTranslatorError)
	// InternalError indicates an error that the client cannot prevent.
	InternalError = errors.New("internal")
	// IsInternal returns true if the given error is an InternalError.
	IsInternal = isCausedByFunc(InternalError)
	// BuilderNeedOpenArrayError indicates an (invalid) attempt to open an array/object when that is not allowed.
	BuilderNeedOpenArrayError = errors.New("builder need open array")
	// IsBuilderNeedOpenArray returns true if the given error is an BuilderNeedOpenArrayError.
	IsBuilderNeedOpenArray = isCausedByFunc(BuilderNeedOpenArrayError)
	// BuilderNeedOpenObjectError indicates an (invalid) attempt to open an array/object when that is not allowed.
	BuilderNeedOpenObjectError = errors.New("builder need open object")
	// IsBuilderNeedOpenObject returns true if the given error is an BuilderNeedOpenObjectError.
	IsBuilderNeedOpenObject = isCausedByFunc(BuilderNeedOpenObjectError)
	// BuilderNeedOpenCompoundError indicates an (invalid) attempt to close an array/object that is already closed.
	BuilderNeedOpenCompoundError = errors.New("builder need open array or object")
	// IsBuilderNeedOpenCompound returns true if the given error is an BuilderNeedOpenCompoundError.
	IsBuilderNeedOpenCompound   = isCausedByFunc(BuilderNeedOpenCompoundError)
	DuplicateAttributeNameError = errors.New("duplicate key name")
	// IsDuplicateAttributeName returns true if the given error is an DuplicateAttributeNameError.
	IsDuplicateAttributeName = isCausedByFunc(DuplicateAttributeNameError)
	// BuilderNotClosedError is returned when a call is made to Builder.Bytes without being closed.
	BuilderNotClosedError = errors.New("builder not closed")
	// IsBuilderNotClosed returns true if the given error is an BuilderNotClosedError.
	IsBuilderNotClosed = isCausedByFunc(BuilderNotClosedError)
	// BuilderKeyAlreadyWrittenError is returned when a call is made to Builder.Bytes without being closed.
	BuilderKeyAlreadyWrittenError = errors.New("builder key already written")
	// IsBuilderKeyAlreadyWritten returns true if the given error is an BuilderKeyAlreadyWrittenError.
	IsBuilderKeyAlreadyWritten = isCausedByFunc(BuilderKeyAlreadyWrittenError)
	// BuilderKeyMustBeStringError is returned when a key is not of type string.
	BuilderKeyMustBeStringError = errors.New("builder key must be string")
	// IsBuilderKeyMustBeString returns true if the given error is an BuilderKeyMustBeStringError.
	IsBuilderKeyMustBeString = isCausedByFunc(BuilderKeyMustBeStringError)
	// BuilderNeedSubValueError is returned when a RemoveLast is called without any value in an object/array.
	BuilderNeedSubValueError = errors.New("builder need sub value")
	// IsBuilderNeedSubValue returns true if the given error is an BuilderNeedSubValueError.
	IsBuilderNeedSubValue = isCausedByFunc(BuilderNeedSubValueError)
	// InvalidUtf8SequenceError indicates an invalid UTF8 (string) sequence.
	InvalidUtf8SequenceError = errors.New("invalid utf8 sequence")
	// IsInvalidUtf8Sequence returns true if the given error is an InvalidUtf8SequenceError.
	IsInvalidUtf8Sequence = isCausedByFunc(InvalidUtf8SequenceError)
	// NoJSONEquivalentError is returned when a Velocypack type cannot be converted to JSON.
	NoJSONEquivalentError = errors.New("no JSON equivalent")
	// IsNoJSONEquivalent returns true if the given error is an NoJSONEquivalentError.
	IsNoJSONEquivalent = isCausedByFunc(NoJSONEquivalentError)
)

// isCausedByFunc creates an error test function.
func isCausedByFunc(cause error) func(err error) bool {
	return func(err error) bool {
		return Cause(err) == cause
	}
}

// BuilderUnexpectedTypeError is returned when a Builder function received an invalid type.
type BuilderUnexpectedTypeError struct {
	Message string
}

// Error implements the error interface for BuilderUnexpectedTypeError.
func (e BuilderUnexpectedTypeError) Error() string {
	return e.Message
}

// IsBuilderUnexpectedType returns true if the given error is an BuilderUnexpectedTypeError.
func IsBuilderUnexpectedType(err error) bool {
	_, ok := Cause(err).(BuilderUnexpectedTypeError)
	return ok
}

// MarshalerError is returned when a custom VPack Marshaler returns an error.
type MarshalerError struct {
	Type reflect.Type
	Err  error
}

// Error implements the error interface for MarshalerError.
func (e MarshalerError) Error() string {
	return "error calling MarshalVPack for type " + e.Type.String() + ": " + e.Err.Error()
}

// IsMarshaler returns true if the given error is an MarshalerError.
func IsMarshaler(err error) bool {
	_, ok := Cause(err).(MarshalerError)
	return ok
}

// UnsupportedTypeError is returned when a type is marshaled that cannot be marshaled.
type UnsupportedTypeError struct {
	Type reflect.Type
}

// Error implements the error interface for UnsupportedTypeError.
func (e UnsupportedTypeError) Error() string {
	return "unsupported type " + e.Type.String()
}

// IsUnsupportedType returns true if the given error is an UnsupportedTypeError.
func IsUnsupportedType(err error) bool {
	_, ok := Cause(err).(UnsupportedTypeError)
	return ok
}

// An InvalidUnmarshalError describes an invalid argument passed to Unmarshal.
// (The argument to Unmarshal must be a non-nil pointer.)
type InvalidUnmarshalError struct {
	Type reflect.Type
}

func (e *InvalidUnmarshalError) Error() string {
	if e.Type == nil {
		return "json: Unmarshal(nil)"
	}

	if e.Type.Kind() != reflect.Ptr {
		return "json: Unmarshal(non-pointer " + e.Type.String() + ")"
	}
	return "json: Unmarshal(nil " + e.Type.String() + ")"
}

// IsInvalidUnmarshal returns true if the given error is an InvalidUnmarshalError.
func IsInvalidUnmarshal(err error) bool {
	_, ok := Cause(err).(*InvalidUnmarshalError)
	return ok
}

// An UnmarshalTypeError describes a JSON value that was
// not appropriate for a value of a specific Go type.
type UnmarshalTypeError struct {
	Value  string       // description of JSON value - "bool", "array", "number -5"
	Type   reflect.Type // type of Go value it could not be assigned to
	Struct string       // name of the struct type containing the field
	Field  string       // name of the field holding the Go value
}

func (e *UnmarshalTypeError) Error() string {
	if e.Struct != "" || e.Field != "" {
		return "json: cannot unmarshal " + e.Value + " into Go struct field " + e.Struct + "." + e.Field + " of type " + e.Type.String()
	}
	return "json: cannot unmarshal " + e.Value + " into Go value of type " + e.Type.String()
}

// IsUnmarshalType returns true if the given error is an UnmarshalTypeError.
func IsUnmarshalType(err error) bool {
	_, ok := Cause(err).(*UnmarshalTypeError)
	return ok
}

// An ParseError is returned when JSON cannot be parsed correctly.
type ParseError struct {
	msg    string
	Offset int64
}

func (e *ParseError) Error() string {
	return e.msg
}

// IsParse returns true if the given error is a ParseError.
func IsParse(err error) bool {
	_, ok := Cause(err).(*ParseError)
	return ok
}

var (
	// WithStack is called on every return of an error to add stacktrace information to the error.
	// When setting this function, also set the Cause function.
	// The interface of this function is compatible with functions in github.com/pkg/errors.
	// WithStack(nil) must return nil.
	WithStack = func(err error) error { return err }
	// Cause is used to get the root cause of the given error.
	// The interface of this function is compatible with functions in github.com/pkg/errors.
	// Cause(nil) must return nil.
	Cause = func(err error) error { return err }
)
