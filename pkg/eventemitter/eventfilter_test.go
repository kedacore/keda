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

package eventemitter

import (
	"testing"

	eventingv1alpha1 "github.com/kedacore/keda/v2/apis/eventing/v1alpha1"
)

func TestNewEventFilter(t *testing.T) {
	included := []eventingv1alpha1.CloudEventType{eventingv1alpha1.ScaledObjectReadyType}
	excluded := []eventingv1alpha1.CloudEventType{eventingv1alpha1.ScaledObjectFailedType}

	filter := NewEventFilter(included, excluded)

	if filter == nil {
		t.Fatal("expected non-nil EventFilter")
	}
	if len(filter.IncludedEventTypes) != 1 {
		t.Errorf("expected 1 included type, got %d", len(filter.IncludedEventTypes))
	}
	if len(filter.ExcludedEventTypes) != 1 {
		t.Errorf("expected 1 excluded type, got %d", len(filter.ExcludedEventTypes))
	}
}

func TestFilterEvent(t *testing.T) {
	tests := []struct {
		name     string
		included []eventingv1alpha1.CloudEventType
		excluded []eventingv1alpha1.CloudEventType
		event    eventingv1alpha1.CloudEventType
		want     bool
	}{
		{
			name:     "no filters - event passes through",
			included: nil,
			excluded: nil,
			event:    eventingv1alpha1.ScaledObjectReadyType,
			want:     false,
		},
		{
			name:     "included list - matching event passes",
			included: []eventingv1alpha1.CloudEventType{eventingv1alpha1.ScaledObjectReadyType},
			excluded: nil,
			event:    eventingv1alpha1.ScaledObjectReadyType,
			want:     false,
		},
		{
			name:     "included list - non-matching event filtered",
			included: []eventingv1alpha1.CloudEventType{eventingv1alpha1.ScaledObjectReadyType},
			excluded: nil,
			event:    eventingv1alpha1.ScaledObjectFailedType,
			want:     true,
		},
		{
			name:     "excluded list - matching event filtered",
			included: nil,
			excluded: []eventingv1alpha1.CloudEventType{eventingv1alpha1.ScaledObjectFailedType},
			event:    eventingv1alpha1.ScaledObjectFailedType,
			want:     true,
		},
		{
			name:     "excluded list - non-matching event passes",
			included: nil,
			excluded: []eventingv1alpha1.CloudEventType{eventingv1alpha1.ScaledObjectFailedType},
			event:    eventingv1alpha1.ScaledObjectReadyType,
			want:     false,
		},
		{
			name:     "included takes precedence over excluded",
			included: []eventingv1alpha1.CloudEventType{eventingv1alpha1.ScaledObjectReadyType},
			excluded: []eventingv1alpha1.CloudEventType{eventingv1alpha1.ScaledObjectReadyType},
			event:    eventingv1alpha1.ScaledObjectReadyType,
			want:     false,
		},
		{
			name:     "empty included list - event passes through",
			included: []eventingv1alpha1.CloudEventType{},
			excluded: []eventingv1alpha1.CloudEventType{},
			event:    eventingv1alpha1.ScaledObjectReadyType,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewEventFilter(tt.included, tt.excluded)
			got := filter.FilterEvent(tt.event)
			if got != tt.want {
				t.Errorf("FilterEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}
