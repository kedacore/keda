/*
Copyright 2019 The Knative Authors

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

package ptr

import "time"

// Int32 is a helper for turning integers into pointers for use in
// API types that want *int32.
func Int32(i int32) *int32 {
	return &i
}

// Int64 is a helper for turning integers into pointers for use in
// API types that want *int64.
func Int64(i int64) *int64 {
	return &i
}

// Float32 is a helper for turning floats into pointers for use in
// API types that want *float32.
func Float32(f float32) *float32 {
	return &f
}

// Float64 is a helper for turning floats into pointers for use in
// API types that want *float64.
func Float64(f float64) *float64 {
	return &f
}

// Bool is a helper for turning bools into pointers for use in
// API types that want *bool.
func Bool(b bool) *bool {
	return &b
}

// String is a helper for turning strings into pointers for use in
// API types that want *string.
func String(s string) *string {
	return &s
}

// Duration is a helper for turning time.Duration into pointers for use in
// API types that want *time.Duration.
func Duration(t time.Duration) *time.Duration {
	return &t
}

// Time is a helper for turning a const time.Time into a pointer for use in
// API types that want *time.Duration.
func Time(t time.Time) *time.Time {
	return &t
}
