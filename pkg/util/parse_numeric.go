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
	"fmt"
	"regexp"
	"strconv"
)

var hintedRegexp *regexp.Regexp
var floatRegexp *regexp.Regexp
var intRegexp *regexp.Regexp

func init() {
	hintedRegexp = regexp.MustCompile(`^(?P<Type>[a-z])\((?P<Value>[\-]?(?:(?:0|[1-9]\d*)(?:\.\d*)?|\.\d+))\)$`)
	floatRegexp = regexp.MustCompile(`^-?\d+\.\d+$`)
	intRegexp = regexp.MustCompile(`^-?\d+$`)
}

type typeHintError struct {
	value string
}

func (e typeHintError) Error() string {
	return fmt.Sprintf("ParseNumeric: Type hint \"%s\" does not match any known hints", e.value)
}

type numericParseError struct {
	value string
}

func (e numericParseError) Error() string {
	return fmt.Sprintf("ParseNumeric: Provided string \"%s\" is neither an int nor a float", e.value)
}

// ParseNumeric is a wrapper around ParseInt and ParseFloat
// mostly, but also allows for providing in-string type hinting
// to ensure that numeric values that would be cast to numbers
// by YAML parsers instead forcefully remain as strings until
// they can get parsed at runtime.
//
// For this purpose, we look for a hint which is defined as
// a single letter, followed by a parentheses wrapped numeric
// value.  d or f for decimal/float, i or n for integer.
//
// If we don't get a match, check if s looks like a float and
// parse it as one if so, otherwise parse as an int.
//
// As a safety mechanism, if we want to be certain a float will
// be returned even if the user provides something which appears
// to be an int, the ensureFloat argument can be set to true.
// This will forcefully call ParseFloat against a non-hinted value
// even if it looked like an int.
//
// Examples:
// ParseNumeric("i(50)", 32) returns an int64 containing 50
// ParseNumeric("d(32)", 32) returns a float64 containing 32.0
// ParseNumeric("30.1", 32) returns a float64 containing 30.1
func ParseNumeric(s string, bitSize int, ensureFloat bool) (interface{}, error) {
	ss := []byte(s)
	if r := hintedRegexp.Find(ss); r != nil {
		match := hintedRegexp.FindStringSubmatch(s)
		switch match[1] {
		case "d":
			return strconv.ParseFloat(match[2], bitSize)
		case "f":
			return strconv.ParseFloat(match[2], bitSize)
		case "i":
			return strconv.ParseInt(match[2], 10, bitSize)
		case "n":
			return strconv.ParseInt(match[2], 10, bitSize)
		default:
			return int64(0), typeHintError{value: match[1]}
		}
	} else if r := floatRegexp.Find(ss); r != nil {
		return strconv.ParseFloat(s, bitSize)
	} else if r := intRegexp.Find(ss); r != nil {
		if ensureFloat {
			return strconv.ParseFloat(s, bitSize)
		}
		return strconv.ParseInt(s, 10, bitSize)
	}
	return int64(0), numericParseError{value: s}
}
