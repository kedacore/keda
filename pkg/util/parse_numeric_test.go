/*
Copyright 2021 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"testing"
)

const defaultFloat float64 = 0
const defaultInt int64 = 0

type testNumericMetadata struct {
	comment       string
	input         string
	expectedInt   int64
	expectedFloat float64
	expectedErr   error
	expectedType  reflect.Type
	ensureFloat   bool
}

var testNumericMetadatas = []testNumericMetadata{
	{
		comment:       "Testing 1.0",
		input:         "1.0",
		expectedInt:   0,
		expectedFloat: 1.0,
		expectedErr:   nil,
		expectedType:  reflect.TypeOf(defaultFloat),
		ensureFloat:   false,
	},
	{
		comment:       "Testing -1.0",
		input:         "-1.0",
		expectedInt:   0,
		expectedFloat: -1.0,
		expectedErr:   nil,
		expectedType:  reflect.TypeOf(defaultFloat),
		ensureFloat:   false,
	},
	{
		comment:       "Testing d(1.0)",
		input:         "d(1.0)",
		expectedInt:   0,
		expectedFloat: 1.0,
		expectedErr:   nil,
		expectedType:  reflect.TypeOf(defaultFloat),
		ensureFloat:   false,
	},
	{
		comment:       "Testing d(-1.0)",
		input:         "d(-1.0)",
		expectedInt:   0,
		expectedFloat: -1.0,
		expectedErr:   nil,
		expectedType:  reflect.TypeOf(defaultFloat),
		ensureFloat:   false,
	},
	{
		comment:       "Testing foo",
		input:         "foo",
		expectedInt:   0,
		expectedFloat: 0,
		expectedErr:   numericParseError{value: "foo"},
		expectedType:  reflect.TypeOf(defaultInt),
		ensureFloat:   false,
	},
	{
		comment:       "Testing x(1)",
		input:         "x(1)",
		expectedInt:   0,
		expectedFloat: 0,
		expectedErr:   typeHintError{value: "x"},
		expectedType:  reflect.TypeOf(defaultInt),
		ensureFloat:   false,
	},
	{
		comment:       "Testing i(1.1)",
		input:         "i(1.1)",
		expectedInt:   0,
		expectedFloat: 0,
		expectedErr:   &strconv.NumError{Func: "ParseInt", Num: "1.1", Err: strconv.ErrSyntax},
		expectedType:  reflect.TypeOf(defaultInt),
		ensureFloat:   false,
	},
	{
		comment:       "Testing d(1)",
		input:         "d(1)",
		expectedInt:   0,
		expectedFloat: 1,
		expectedErr:   nil,
		expectedType:  reflect.TypeOf(defaultFloat),
		ensureFloat:   false,
	},
	{
		comment:       "Testing i(1)",
		input:         "i(1)",
		expectedInt:   1,
		expectedFloat: 0,
		expectedErr:   nil,
		expectedType:  reflect.TypeOf(defaultInt),
		ensureFloat:   false,
	},
	{
		comment:       "Testing i(-1)",
		input:         "i(-1)",
		expectedInt:   -1,
		expectedFloat: 0,
		expectedErr:   nil,
		expectedType:  reflect.TypeOf(defaultInt),
		ensureFloat:   false,
	},
	{
		comment:       "Testing 1",
		input:         "1",
		expectedInt:   1,
		expectedFloat: 0,
		expectedErr:   nil,
		expectedType:  reflect.TypeOf(defaultInt),
		ensureFloat:   false,
	},
	{
		comment:       "Testing -1",
		input:         "-1",
		expectedInt:   -1,
		expectedFloat: 0,
		expectedErr:   nil,
		expectedType:  reflect.TypeOf(defaultInt),
		ensureFloat:   false,
	},
	{
		comment:       "Testing 1 (ensureFloat)",
		input:         "1",
		expectedInt:   0,
		expectedFloat: 1,
		expectedErr:   nil,
		expectedType:  reflect.TypeOf(defaultFloat),
		ensureFloat:   true,
	},
}

func TestParseNumeric(t *testing.T) {
	for _, testData := range testNumericMetadatas {
		t.Log(testData.comment)

		r, err := ParseNumeric(testData.input, 64, testData.ensureFloat)

		if reflect.ValueOf(testData.expectedErr).Kind() == reflect.Ptr {
			if reflect.TypeOf(err) != reflect.TypeOf(testData.expectedErr) {
				t.Error("Unexpected error returned", "wants", testData.expectedErr, "got", err)
			}
			if fmt.Sprintf("%s", err) != fmt.Sprintf("%s", testData.expectedErr) {
				t.Error("Unexpected error returned", "wants", testData.expectedErr, "got", err)
			}
		} else if !errors.Is(err, testData.expectedErr) {
			t.Error("Unexpected error returned", "wants", testData.expectedErr, "got", err)
		}

		rType := reflect.TypeOf(r)

		if rType != testData.expectedType {
			t.Error("Type does not match expectation", "wants", testData.expectedType, "got", rType)
		}

		switch rType {
		case reflect.TypeOf(defaultInt):
			if r != testData.expectedInt {
				t.Error("Value returned does not match expectation", "wants", testData.expectedInt, "got", r)
			}
		case reflect.TypeOf(defaultFloat):
			if r != testData.expectedFloat {
				t.Error("Value returned does not match expectation", "wants", testData.expectedFloat, "got", r)
			}
		}
	}
}
