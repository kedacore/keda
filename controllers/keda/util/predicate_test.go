package util

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func createObjectWithAnnotations(annotations map[string]string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetAnnotations(annotations)
	return obj
}

func TestPausedReplicasPredicate_Update(t *testing.T) {
	predicate := PausedReplicasPredicate{}

	tests := []struct {
		name      string
		oldObject *unstructured.Unstructured
		newObject *unstructured.Unstructured
		expected  bool
	}{
		{
			name:      "Both objects have the same annotation value",
			oldObject: createObjectWithAnnotations(map[string]string{PausedReplicasAnnotation: "8"}),
			newObject: createObjectWithAnnotations(map[string]string{PausedReplicasAnnotation: "8"}),
			expected:  false,
		},
		{
			name:      "Annotation value changed from 8 to 10",
			oldObject: createObjectWithAnnotations(map[string]string{PausedReplicasAnnotation: "8"}),
			newObject: createObjectWithAnnotations(map[string]string{PausedReplicasAnnotation: "10"}),
			expected:  true,
		},
		{
			name:      "Old annotation is nil, new annotation has value",
			oldObject: createObjectWithAnnotations(nil),
			newObject: createObjectWithAnnotations(map[string]string{PausedReplicasAnnotation: "10"}),
			expected:  true,
		},
		{
			name:      "Old annotation has value, new annotation is nil",
			oldObject: createObjectWithAnnotations(map[string]string{PausedReplicasAnnotation: "8"}),
			newObject: createObjectWithAnnotations(nil),
			expected:  true,
		},
		{
			name:      "Both annotations are nil",
			oldObject: createObjectWithAnnotations(nil),
			newObject: createObjectWithAnnotations(nil),
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := event.UpdateEvent{
				ObjectOld: tt.oldObject,
				ObjectNew: tt.newObject,
			}
			result := predicate.Update(event)
			if result != tt.expected {
				t.Errorf("expected %v, but got %v", tt.expected, result)
			}
		})
	}
}
