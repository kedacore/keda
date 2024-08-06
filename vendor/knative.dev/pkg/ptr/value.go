/*
Copyright 2021 The Knative Authors

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

// Int32Value is a helper for turning pointers to integers into values for use
// in API types that want int32.
func Int32Value(i *int32) int32 {
	if i == nil {
		return 0
	}
	return *i
}

// Int64Value is a helper for turning pointers to integers into values for use
// in API types that want int64.
func Int64Value(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}

// Float32Value is a helper for turning pointers to floats into values for use
// in API types that want float32.
func Float32Value(f *float32) float32 {
	if f == nil {
		return 0
	}
	return *f
}

// Float64Value is a helper for turning pointers to floats into values for use
// in API types that want float64.
func Float64Value(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

// BoolValue is a helper for turning pointers to bools into values for use in
// API types that want bool.
func BoolValue(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

// StringValue is a helper for turning pointers to strings into values for use
// in API types that want string.
func StringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// DurationValue is a helper for turning *time.Duration into values for use in
// API types that want time.Duration.
func DurationValue(t *time.Duration) time.Duration {
	if t == nil {
		return 0
	}
	return *t
}

// TimeValue is a helper for turning *time.Time into values for use in API
// types that want API types that want time.Time.
func TimeValue(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}
