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

package reconciler

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AnnotationFilterFunc creates a FilterFunc only accepting objects with given annotation key and value
func AnnotationFilterFunc(key string, value string, allowUnset bool) func(interface{}) bool {
	return func(obj interface{}) bool {
		if mo, ok := obj.(metav1.Object); ok {
			anno := mo.GetAnnotations()
			annoVal, ok := anno[key]
			if !ok {
				return allowUnset
			}
			return annoVal == value
		}
		return false
	}
}

// LabelExistsFilterFunc creates a FilterFunc only accepting objects which have a given label.
func LabelExistsFilterFunc(label string) func(obj interface{}) bool {
	return func(obj interface{}) bool {
		if mo, ok := obj.(metav1.Object); ok {
			labels := mo.GetLabels()
			_, ok := labels[label]
			return ok
		}
		return false
	}
}
