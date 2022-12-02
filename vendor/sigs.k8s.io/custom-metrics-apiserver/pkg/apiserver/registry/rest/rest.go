/*
Copyright 2017 The Kubernetes Authors.

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

package rest

import (
	"context"

	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	"k8s.io/apimachinery/pkg/runtime"
)

// ListerWithOptions is an object that can retrieve resources that match the provided field
// and label criteria and takes additional options on the list request.
type ListerWithOptions interface {
	// NewList returns an empty object that can be used with the List call.
	// This object must be a pointer type for use with Codec.DecodeInto([]byte, runtime.Object)
	NewList() runtime.Object

	// List selects resources in the storage which match to the selector. 'options' can be nil.
	// The extraOptions object passed to it is of the same type returned by the NewListOptions
	// method.
	List(ctx context.Context, options *metainternalversion.ListOptions, extraOptions runtime.Object) (runtime.Object, error)

	// NewListOptions returns an empty options object that will be used to pass extra options
	// to the List method. It may return a bool and a string, if true, the
	// value of the request path below the list will be included as the named
	// string in the serialization of the runtime object. E.g., returning "path"
	// will convert the trailing request scheme value to "path" in the map[string][]string
	// passed to the converter.
	NewListOptions() (runtime.Object, bool, string)
}
