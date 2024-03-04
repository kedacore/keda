/*
Copyright 2024 The KEDA Authors

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

package eventemitter

import (
	eventingv1alpha1 "github.com/kedacore/keda/v2/apis/eventing/v1alpha1"
)

// EventFilter defines the behavior for different event handlers
type EventFilter struct {
	IncludedEventTypes []eventingv1alpha1.CloudEventType

	ExcludedEventTypes []eventingv1alpha1.CloudEventType
}

// NewEventFilter creates a new EventFilter
func NewEventFilter(includedEventTypes []eventingv1alpha1.CloudEventType, excludedEventTypes []eventingv1alpha1.CloudEventType) *EventFilter {
	return &EventFilter{
		IncludedEventTypes: includedEventTypes,
		ExcludedEventTypes: excludedEventTypes,
	}
}

// FilterEvent returns true if the event should be handled
func (e *EventFilter) FilterEvent(eventType eventingv1alpha1.CloudEventType) bool {
	if len(e.IncludedEventTypes) > 0 {
		return e.filterIncludedEventTypes(eventType)
	}

	if len(e.ExcludedEventTypes) > 0 {
		return e.filterExcludedEventTypes(eventType)
	}

	return true
}

// FilterIncludedEventTypes returns true if the event included in the includedEventTypes
func (e *EventFilter) filterIncludedEventTypes(eventType eventingv1alpha1.CloudEventType) bool {
	for _, includedEventType := range e.IncludedEventTypes {
		if includedEventType == eventType {
			return true
		}
	}

	return false
}

// FilterExcludedEventTypes returns true if the event not included in the excludedEventTypes
func (e *EventFilter) filterExcludedEventTypes(eventType eventingv1alpha1.CloudEventType) bool {
	for _, excludedEventType := range e.ExcludedEventTypes {
		if excludedEventType == eventType {
			return false
		}
	}

	return true
}
