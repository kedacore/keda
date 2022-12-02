/*
Copyright 2018 The Knative Authors

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

package kmeta

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// OwnerRefable indicates that a particular type has sufficient
// information to produce a metav1.OwnerReference to an object.
type OwnerRefable interface {
	metav1.ObjectMetaAccessor

	// GetGroupVersionKind returns a GroupVersionKind. The name is chosen
	// to avoid collision with TypeMeta's GroupVersionKind() method.
	// See: https://issues.k8s.io/3030
	GetGroupVersionKind() schema.GroupVersionKind
}

// NewControllerRef creates an OwnerReference pointing to the given controller.
func NewControllerRef(obj OwnerRefable) *metav1.OwnerReference {
	return metav1.NewControllerRef(obj.GetObjectMeta(), obj.GetGroupVersionKind())
}
