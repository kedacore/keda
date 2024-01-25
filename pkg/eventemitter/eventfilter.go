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

// EventFilter defines the behavior for different event handlers
type EventFilter struct {
	IncludedEventTypes []string

	ExcludedEventTypes []string
}

// NewEventFilter creates a new EventFilter
func NewEventFilter(includedEventTypes []string, excludedEventTypes []string) *EventFilter {
	return &EventFilter{
		IncludedEventTypes: includedEventTypes,
		ExcludedEventTypes: excludedEventTypes,
	}
}

// FilterEvent returns true if the event should be handled
func (e *EventFilter) FilterEvent(eventType string) bool {
	if len(e.IncludedEventTypes) > 0 {
		return e.filterIncludedEventTypes(eventType)
	}

	if len(e.ExcludedEventTypes) > 0 {
		return e.filterExcludedEventTypes(eventType)
	}

	return true
}

// FilterIncludedEventTypes returns true if the event included in the includedEventTypes
func (e *EventFilter) filterIncludedEventTypes(eventType string) bool {
	for _, includedEventType := range e.IncludedEventTypes {
		if includedEventType == eventType {
			return true
		}
	}

	return false
}

// FilterExcludedEventTypes returns true if the event not included in the excludedEventTypes
func (e *EventFilter) filterExcludedEventTypes(eventType string) bool {
	for _, excludedEventType := range e.ExcludedEventTypes {
		if excludedEventType == eventType {
			return false
		}
	}

	return true
}
