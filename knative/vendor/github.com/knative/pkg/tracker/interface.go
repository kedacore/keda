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

package tracker

import (
	corev1 "k8s.io/api/core/v1"
)

// Interface defines the interface through which an object can register
// that it is tracking another object by reference.
type Interface interface {
	// Track tells us that "obj" is tracking changes to the
	// referenced object.
	Track(ref corev1.ObjectReference, obj interface{}) error

	// OnChanged is a callback to register with the InformerFactory
	// so that we are notified for appropriate object changes.
	OnChanged(obj interface{})
}
