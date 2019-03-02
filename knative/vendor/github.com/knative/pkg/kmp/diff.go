/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kmp

import (
	"fmt"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/api/resource"
)

// Commonly used Comparers and other Options go here.
var defaultOpts []cmp.Option

func init() {
	defaultOpts = []cmp.Option{
		cmp.Comparer(func(x, y resource.Quantity) bool {
			return x.Cmp(y) == 0
		}),
	}
}

// SafeDiff wraps cmp.Diff but recovers from panics and uses custom Comparers for:
// * k8s.io/apimachinery/pkg/api/resource.Quantity
func SafeDiff(x, y interface{}, opts ...cmp.Option) (diff string, err error) {
	// cmp.Diff will panic if we miss something; return error instead of crashing.
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered in kmp.SafeDiff: %v", r)
		}
	}()

	opts = append(opts, defaultOpts...)
	diff = cmp.Diff(x, y, opts...)

	return
}

func SafeEqual(x, y interface{}, opts ...cmp.Option) (equal bool, err error) {
	// cmp.Equal will panic if we miss something; return error instead of crashing.
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered in kmp.SafeEqual: %v", r)
		}
	}()

	opts = append(opts, defaultOpts...)
	equal = cmp.Equal(x, y, opts...)

	return
}
