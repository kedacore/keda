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

package provider

import (
	"fmt"
	"net/http"

	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NewMetricNotFoundError returns a StatusError indicating the given metric could not be found.
// It is similar to NewNotFound, but more specialized
func NewMetricNotFoundError(resource schema.GroupResource, metricName string) *apierr.StatusError {
	return &apierr.StatusError{ErrStatus: metav1.Status{
		Status:  metav1.StatusFailure,
		Code:    int32(http.StatusNotFound),
		Reason:  metav1.StatusReasonNotFound,
		Message: fmt.Sprintf("the server could not find the metric %s for %s", metricName, resource.String()),
	}}
}

// NewMetricNotFoundForError returns a StatusError indicating the given metric could not be found for
// the given named object. It is similar to NewNotFound, but more specialized
func NewMetricNotFoundForError(resource schema.GroupResource, metricName string, resourceName string) *apierr.StatusError {
	return &apierr.StatusError{ErrStatus: metav1.Status{
		Status:  metav1.StatusFailure,
		Code:    int32(http.StatusNotFound),
		Reason:  metav1.StatusReasonNotFound,
		Message: fmt.Sprintf("the server could not find the metric %s for %s %s", metricName, resource.String(), resourceName),
	}}
}

// NewMetricNotFoundForError returns a StatusError indicating the given metric could not be found for
// the given named object. It is similar to NewNotFound, but more specialized
func NewMetricNotFoundForSelectorError(resource schema.GroupResource, metricName string, resourceName string, selector labels.Selector) *apierr.StatusError {
	return &apierr.StatusError{ErrStatus: metav1.Status{
		Status:  metav1.StatusFailure,
		Code:    int32(http.StatusNotFound),
		Reason:  metav1.StatusReasonNotFound,
		Message: fmt.Sprintf("the server could not find the metric %s for %s %s with selector %s", metricName, resource.String(), resourceName, selector.String()),
	}}
}
