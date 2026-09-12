/*
Copyright 2026 The KEDA Authors

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

// Package pointer provides small generic helpers for pointer-typed fields.
// It is a leaf package with no KEDA-internal dependencies so it can be safely
// imported from any layer (including apis/...).
package pointer

// IsEmpty returns true if the pointer is non-nil but points to the
// zero value of T. It returns false for nil pointers, which are treated as
// "unset" rather than "empty".
//
// This is useful for CRD fields backed by *string/*int style pointers where
// "user omitted the field" (nil) and "user explicitly set the field to its
// zero value" (non-nil zero) must be distinguished.
func IsEmpty[T comparable](p *T) bool {
	if p == nil {
		return false
	}
	var zero T
	return *p == zero
}
