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
	"errors"
	"fmt"
	"reflect"

	commonpb "go.temporal.io/api/common/v1"

	"go.temporal.io/sdk/converter"
)

// encode multiple arguments(arguments to a function).
func encodeArgs(dc converter.DataConverter, args []interface{}) (*commonpb.Payloads, error) {
	return dc.ToPayloads(args...)
}

// decode multiple arguments(arguments to a function).
func decodeArgs(dc converter.DataConverter, fnType reflect.Type, data *commonpb.Payloads) (result []reflect.Value, err error) {
	r, err := decodeArgsToPointerValues(dc, fnType, data)
	if err != nil {
		return
	}
	for i := 0; i < len(r); i++ {
		result = append(result, reflect.ValueOf(r[i]).Elem())
	}
	return
}

func decodeArgsToPointerValues(dc converter.DataConverter, fnType reflect.Type, data *commonpb.Payloads) (result []interface{}, err error) {
argsLoop:
	for i := 0; i < fnType.NumIn(); i++ {
		argT := fnType.In(i)
		if i == 0 && (isActivityContext(argT) || isWorkflowContext(argT)) {
			continue argsLoop
		}
		arg := reflect.New(argT).Interface()
		result = append(result, arg)
	}
	err = dc.FromPayloads(data, result...)
	if err != nil {
		return
	}
	return
}

func decodeArgsToRawValues(dc converter.DataConverter, fnType reflect.Type, data *commonpb.Payloads) ([]interface{}, error) {
	// Build pointers to results
	var pointers []interface{}
	for i := 0; i < fnType.NumIn(); i++ {
		argT := fnType.In(i)
		if i == 0 && (isActivityContext(argT) || isWorkflowContext(argT)) {
			continue
		}
		pointers = append(pointers, reflect.New(argT).Interface())
	}

	// Unmarshal
	if err := dc.FromPayloads(data, pointers...); err != nil {
		return nil, err
	}

	// Convert results back to non-pointer versions
	results := make([]interface{}, len(pointers))
	for i, pointer := range pointers {
		result := reflect.ValueOf(pointer).Elem()
		// Do not set nil pointers
		if result.Kind() != reflect.Ptr || !result.IsNil() {
			results[i] = result.Interface()
		}
	}

	return results, nil
}

// encode single value(like return parameter).
func encodeArg(dc converter.DataConverter, arg interface{}) (*commonpb.Payloads, error) {
	return dc.ToPayloads(arg)
}

// decode single value(like return parameter).
func decodeArg(dc converter.DataConverter, data *commonpb.Payloads, valuePtr interface{}) error {
	return dc.FromPayloads(data, valuePtr)
}

func decodeAndAssignValue(dc converter.DataConverter, from interface{}, toValuePtr interface{}) error {
	if toValuePtr == nil {
		return nil
	}
	if rf := reflect.ValueOf(toValuePtr); rf.Type().Kind() != reflect.Ptr {
		return errors.New("value parameter provided is not a pointer")
	}
	if data, ok := from.(*commonpb.Payloads); ok {
		if err := decodeArg(dc, data, toValuePtr); err != nil {
			return err
		}
	} else if fv := reflect.ValueOf(from); fv.IsValid() {
		fromType := fv.Type()
		toType := reflect.TypeOf(toValuePtr).Elem()
		// If the value set was a pointer and is the same type as the wanted result,
		// instead of panicking because it is not a pointer to a pointer, we will
		// just set the pointer
		if fv.Kind() == reflect.Ptr && fromType.Elem() == toType {
			reflect.ValueOf(toValuePtr).Elem().Set(fv.Elem())
		} else {
			assignable := fromType.AssignableTo(toType)
			if !assignable {
				return fmt.Errorf("%s is not assignable to %s", fromType.Name(), toType.Name())
			}
			reflect.ValueOf(toValuePtr).Elem().Set(fv)
		}
	}
	return nil
}
