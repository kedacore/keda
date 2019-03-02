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

package controller

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/knative/pkg/kmeta"
)

type Callback func(interface{})

func EnsureTypeMeta(f Callback, gvk schema.GroupVersionKind) Callback {
	apiVersion, kind := gvk.ToAPIVersionAndKind()

	return func(untyped interface{}) {
		typed, err := kmeta.DeletionHandlingAccessor(untyped)
		if err != nil {
			// TODO: We should consider logging here.
			return
		}
		// We need to populated TypeMeta, but cannot trample the
		// informer's copy.
		// TODO(mattmoor): Avoid the copy if TypeMeta is set.
		copy := typed.DeepCopyObject()

		accessor, err := meta.TypeAccessor(copy)
		if err != nil {
			return
		}
		accessor.SetAPIVersion(apiVersion)
		accessor.SetKind(kind)

		// Pass in the mutated copy (accessor is not just a type cast)
		f(copy)
	}
}
