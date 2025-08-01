package otto

import (
	"errors"
	"fmt"

	"github.com/robertkrimen/otto/file"
)

type exception struct {
	value interface{}
}

func newException(value interface{}) *exception {
	return &exception{
		value: value,
	}
}

func (e *exception) eject() interface{} {
	value := e.value
	e.value = nil // Prevent Go from holding on to the value, whatever it is
	return value
}

type ottoError struct {
	name    string
	message string
	trace   []frame

	offset int
}

func (e ottoError) format() string {
	if len(e.name) == 0 {
		return e.message
	}
	if len(e.message) == 0 {
		return e.name
	}
	return fmt.Sprintf("%s: %s", e.name, e.message)
}

func (e ottoError) formatWithStack() string {
	str := e.format() + "\n"
	for _, frm := range e.trace {
		str += "    at " + frm.location() + "\n"
	}
	return str
}

type frame struct {
	fn         interface{}
	file       *file.File
	nativeFile string
	callee     string
	nativeLine int
	offset     int
	native     bool
}

var nativeFrame = frame{}

type at int

func (fr frame) location() string {
	str := "<unknown>"

	switch {
	case fr.native:
		str = "<native code>"
		if fr.nativeFile != "" && fr.nativeLine != 0 {
			str = fmt.Sprintf("%s:%d", fr.nativeFile, fr.nativeLine)
		}
	case fr.file != nil:
		if p := fr.file.Position(file.Idx(fr.offset)); p != nil {
			path, line, column := p.Filename, p.Line, p.Column

			if path == "" {
				path = "<anonymous>"
			}

			str = fmt.Sprintf("%s:%d:%d", path, line, column)
		}
	}

	if fr.callee != "" {
		str = fmt.Sprintf("%s (%s)", fr.callee, str)
	}

	return str
}

// An Error represents a runtime error, e.g. a TypeError, a ReferenceError, etc.
type Error struct {
	ottoError
}

// Error returns a description of the error
//
//	TypeError: 'def' is not a function
func (e Error) Error() string {
	return e.format()
}

// String returns a description of the error and a trace of where the
// error occurred.
//
//	TypeError: 'def' is not a function
//	    at xyz (<anonymous>:3:9)
//	    at <anonymous>:7:1/
func (e Error) String() string {
	return e.formatWithStack()
}

// GoString returns a description of the error and a trace of where the
// error occurred. Printing with %#v will trigger this behaviour.
func (e Error) GoString() string {
	return e.formatWithStack()
}

func (e ottoError) describe(format string, in ...interface{}) string {
	return fmt.Sprintf(format, in...)
}

func (e ottoError) messageValue() Value {
	if e.message == "" {
		return Value{}
	}
	return stringValue(e.message)
}

func (rt *runtime) typeErrorResult(throw bool) bool {
	if throw {
		panic(rt.panicTypeError())
	}
	return false
}

func newError(rt *runtime, name string, stackFramesToPop int, in ...interface{}) ottoError {
	err := ottoError{
		name:   name,
		offset: -1,
	}
	description := ""
	length := len(in)

	if rt != nil && rt.scope != nil {
		curScope := rt.scope

		for range stackFramesToPop {
			if curScope.outer != nil {
				curScope = curScope.outer
			}
		}

		frm := curScope.frame

		if length > 0 {
			if atv, ok := in[length-1].(at); ok {
				in = in[0 : length-1]
				if curScope != nil {
					frm.offset = int(atv)
				}
				length--
			}
			if length > 0 {
				description, in = in[0].(string), in[1:]
			}
		}

		limit := rt.traceLimit

		err.trace = append(err.trace, frm)
		if curScope != nil {
			for curScope = curScope.outer; curScope != nil; curScope = curScope.outer {
				if limit--; limit == 0 {
					break
				}

				if curScope.frame.offset >= 0 {
					err.trace = append(err.trace, curScope.frame)
				}
			}
		}
	} else if length > 0 {
		description, in = in[0].(string), in[1:]
	}
	err.message = err.describe(description, in...)

	return err
}

func (rt *runtime) panicTypeError(argumentList ...interface{}) *exception {
	return &exception{
		value: newError(rt, "TypeError", 0, argumentList...),
	}
}

func (rt *runtime) panicReferenceError(argumentList ...interface{}) *exception {
	return &exception{
		value: newError(rt, "ReferenceError", 0, argumentList...),
	}
}

func (rt *runtime) panicURIError(argumentList ...interface{}) *exception {
	return &exception{
		value: newError(rt, "URIError", 0, argumentList...),
	}
}

func (rt *runtime) panicSyntaxError(argumentList ...interface{}) *exception {
	return &exception{
		value: newError(rt, "SyntaxError", 0, argumentList...),
	}
}

func (rt *runtime) panicRangeError(argumentList ...interface{}) *exception {
	return &exception{
		value: newError(rt, "RangeError", 0, argumentList...),
	}
}

func catchPanic(function func()) (err error) {
	defer func() {
		if caught := recover(); caught != nil {
			if excep, ok := caught.(*exception); ok {
				caught = excep.eject()
			}
			switch caught := caught.(type) {
			case *Error:
				err = caught
				return
			case ottoError:
				err = &Error{caught}
				return
			case Value:
				if vl := caught.object(); vl != nil {
					if vl, ok := vl.value.(ottoError); ok {
						err = &Error{vl}
						return
					}
				}
				err = errors.New(caught.string())
				return
			}
			panic(caught)
		}
	}()
	function()
	return nil
}
