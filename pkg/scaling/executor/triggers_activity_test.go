/*
Copyright 2025 The KEDA Authors

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

package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

func TestGetTriggersActivity(t *testing.T) {
	scaledObject := &v1alpha1.ScaledObject{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-so",
			Namespace: "test",
		},
		Status: v1alpha1.ScaledObjectStatus{
			ScaleTargetKind:     "apps/v1.Deployment",
			ExternalMetricNames: []string{"trigger-a", "trigger-b", "trigger-c"},
		},
	}

	t.Run("All triggers inactive", func(t *testing.T) {
		result := getTriggersActivity(scaledObject, ScaleExecutorOptions{ActiveTriggers: []string{}})
		assert.Len(t, result, 3)
		assert.False(t, result["trigger-a"].IsActive)
		assert.False(t, result["trigger-b"].IsActive)
		assert.False(t, result["trigger-c"].IsActive)
	})

	t.Run("One trigger active", func(t *testing.T) {
		result := getTriggersActivity(scaledObject, ScaleExecutorOptions{ActiveTriggers: []string{"trigger-a"}})
		assert.True(t, result["trigger-a"].IsActive)
		assert.False(t, result["trigger-b"].IsActive)
		assert.False(t, result["trigger-c"].IsActive)
	})

	t.Run("Multiple triggers active", func(t *testing.T) {
		result := getTriggersActivity(scaledObject, ScaleExecutorOptions{ActiveTriggers: []string{"trigger-a", "trigger-c"}})
		assert.True(t, result["trigger-a"].IsActive)
		assert.False(t, result["trigger-b"].IsActive)
		assert.True(t, result["trigger-c"].IsActive)
	})

	t.Run("All triggers active", func(t *testing.T) {
		result := getTriggersActivity(scaledObject, ScaleExecutorOptions{ActiveTriggers: []string{"trigger-a", "trigger-b", "trigger-c"}})
		assert.True(t, result["trigger-a"].IsActive)
		assert.True(t, result["trigger-b"].IsActive)
		assert.True(t, result["trigger-c"].IsActive)
	})

	t.Run("Zero-value options returns all inactive", func(t *testing.T) {
		result := getTriggersActivity(scaledObject, ScaleExecutorOptions{})
		assert.Len(t, result, 3)
		assert.False(t, result["trigger-a"].IsActive)
		assert.False(t, result["trigger-b"].IsActive)
		assert.False(t, result["trigger-c"].IsActive)
	})
}

func TestGetTriggersActivityPushScaler(t *testing.T) {
	scaledObject := &v1alpha1.ScaledObject{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-so",
			Namespace: "test",
		},
		Status: v1alpha1.ScaledObjectStatus{
			ScaleTargetKind:     "apps/v1.Deployment",
			ExternalMetricNames: []string{"trigger-a", "trigger-b", "trigger-c"},
			TriggersActivity: map[string]v1alpha1.TriggerActivityStatus{
				"trigger-a": {IsActive: true},
				"trigger-b": {IsActive: false},
				"trigger-c": {IsActive: true},
			},
		},
	}

	t.Run("Push scaler only updates its own trigger", func(t *testing.T) {
		result := getTriggersActivity(scaledObject, ScaleExecutorOptions{
			ActiveTriggers:      []string{"trigger-b"},
			ForPushScalerMetric: "trigger-b",
		})

		// push scaler only touches its own trigger, preserving existing state for others
		assert.True(t, result["trigger-a"].IsActive) // preserved from existing status
		assert.True(t, result["trigger-b"].IsActive) // updated by push scaler
		assert.True(t, result["trigger-c"].IsActive) // preserved from existing status
	})

	t.Run("Push scaler deactivates its trigger", func(t *testing.T) {
		result := getTriggersActivity(scaledObject, ScaleExecutorOptions{
			ActiveTriggers:      []string{},
			ForPushScalerMetric: "trigger-a",
		})

		assert.False(t, result["trigger-a"].IsActive) // deactivated by push scaler
		assert.False(t, result["trigger-b"].IsActive) // preserved from existing status
		assert.True(t, result["trigger-c"].IsActive)  // preserved from existing status
	})
}

func TestGetTriggersActivityTriggerListChanges(t *testing.T) {
	t.Run("New trigger added", func(t *testing.T) {
		so := &v1alpha1.ScaledObject{
			ObjectMeta: v1.ObjectMeta{Name: "test", Namespace: "test"},
			Status: v1alpha1.ScaledObjectStatus{
				ExternalMetricNames: []string{"trigger-a", "trigger-b", "trigger-new"},
			},
		}
		result := getTriggersActivity(so, ScaleExecutorOptions{ActiveTriggers: []string{"trigger-a", "trigger-new"}})
		assert.Len(t, result, 3)
		assert.True(t, result["trigger-a"].IsActive)
		assert.False(t, result["trigger-b"].IsActive)
		assert.True(t, result["trigger-new"].IsActive)
	})

	t.Run("Trigger removed", func(t *testing.T) {
		so := &v1alpha1.ScaledObject{
			ObjectMeta: v1.ObjectMeta{Name: "test", Namespace: "test"},
			Status: v1alpha1.ScaledObjectStatus{
				// trigger-b was removed from the list
				ExternalMetricNames: []string{"trigger-a", "trigger-c"},
				TriggersActivity: map[string]v1alpha1.TriggerActivityStatus{
					"trigger-a": {IsActive: true},
					"trigger-b": {IsActive: true},
					"trigger-c": {IsActive: false},
				},
			},
		}
		result := getTriggersActivity(so, ScaleExecutorOptions{ActiveTriggers: []string{"trigger-a"}})
		assert.Len(t, result, 2)
		assert.True(t, result["trigger-a"].IsActive)
		assert.False(t, result["trigger-c"].IsActive)
		// trigger-b is no longer in the result since it's not in ExternalMetricNames
		_, exists := result["trigger-b"]
		assert.False(t, exists)
	})

	t.Run("Trigger replaced", func(t *testing.T) {
		so := &v1alpha1.ScaledObject{
			ObjectMeta: v1.ObjectMeta{Name: "test", Namespace: "test"},
			Status: v1alpha1.ScaledObjectStatus{
				// trigger-old was replaced by trigger-new
				ExternalMetricNames: []string{"trigger-a", "trigger-new"},
				TriggersActivity: map[string]v1alpha1.TriggerActivityStatus{
					"trigger-a":   {IsActive: true},
					"trigger-old": {IsActive: true},
				},
			},
		}
		result := getTriggersActivity(so, ScaleExecutorOptions{ActiveTriggers: []string{"trigger-a", "trigger-new"}})
		assert.Len(t, result, 2)
		assert.True(t, result["trigger-a"].IsActive)
		assert.True(t, result["trigger-new"].IsActive)
		_, exists := result["trigger-old"]
		assert.False(t, exists)
	})

	t.Run("Empty metric names returns empty map", func(t *testing.T) {
		so := &v1alpha1.ScaledObject{
			ObjectMeta: v1.ObjectMeta{Name: "test", Namespace: "test"},
			Status:     v1alpha1.ScaledObjectStatus{},
		}
		result := getTriggersActivity(so, ScaleExecutorOptions{ActiveTriggers: []string{}})
		assert.Empty(t, result)
	})
}

func TestGetTriggersActivityScaledJob(t *testing.T) {
	scaledJob := &v1alpha1.ScaledJob{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-sj",
			Namespace: "test",
		},
		Status: v1alpha1.ScaledJobStatus{
			ExternalMetricNames: []string{"trigger-x", "trigger-y"},
		},
	}

	t.Run("ScaledJob triggers activity", func(t *testing.T) {
		result := getTriggersActivity(scaledJob, ScaleExecutorOptions{ActiveTriggers: []string{"trigger-x"}})
		assert.Len(t, result, 2)
		assert.True(t, result["trigger-x"].IsActive)
		assert.False(t, result["trigger-y"].IsActive)
	})
}
