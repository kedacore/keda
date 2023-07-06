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
	"strings"
)

// Contains checks if the passed string is present in the given slice of strings.
func Contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// Remove deletes the passed string from the given slice of strings.
func Remove(list []string, s string) []string {
	for i, v := range list {
		if v == s {
			list = append(list[:i], list[i+1:]...)
		}
	}
	return list
}

// AppendIntoString append a new string into a string that has seprator
// For example,
//
//	-- input: `viewer,editor`, `owner`, `,`  output: `viewer,editor,owner`
func AppendIntoString(srcStr string, appendStr string, sep string) string {
	if appendStr == "" {
		return srcStr
	}

	splitStrings := []string{}
	if srcStr != "" {
		splitStrings = strings.Split(srcStr, sep)
	}

	if !Contains(splitStrings, appendStr) {
		splitStrings = append(splitStrings, appendStr)
		srcStr = strings.Join(splitStrings, sep)
	}
	return srcStr
}

// RemoveFromString remove a string from src string that has seprator
// For example,
//
//	-- input: `viewer,editor,owner`, `owner`, `,`  output: `viewer,editor`
//	-- input: `viewer,editor,owner`, `owner`, `:`  output: `viewer,editor,owner`
func RemoveFromString(srcStr string, str string, sep string) string {
	if srcStr == "" {
		return srcStr
	}

	splitStrings := []string{}
	if srcStr != "" {
		splitStrings = strings.Split(srcStr, sep)
	}

	splitStrings = Remove(splitStrings, str)
	srcStr = strings.Join(splitStrings, sep)
	return srcStr
}
