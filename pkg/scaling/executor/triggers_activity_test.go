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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"

	"github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/mock/mock_client"
	"github.com/kedacore/keda/v2/pkg/mock/mock_scale"
)

func TestDynamicTriggersActivityUpdates(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := record.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	statusWriter := mock_client.NewMockStatusWriter(ctrl)

	scaleExecutorInterface := NewScaleExecutor(client, mockScaleClient, nil, recorder)
	scaleExecutor := scaleExecutorInterface.(*scaleExecutor)

	// Mock the client's status update
	client.EXPECT().Status().Return(statusWriter).AnyTimes()
	statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

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

	t.Run("Initial state - all triggers inactive", func(t *testing.T) {
		err := scaleExecutor.updateTriggersActivity(
			context.Background(),
			scaleExecutor.logger,
			scaledObject,
			[]string{},
		)
		assert.NoError(t, err)

		// All triggers should exist but be inactive
		assert.Len(t, scaledObject.Status.TriggersActivity, 3)
		assert.False(t, scaledObject.Status.TriggersActivity["trigger-a"].IsActive)
		assert.False(t, scaledObject.Status.TriggersActivity["trigger-b"].IsActive)
		assert.False(t, scaledObject.Status.TriggersActivity["trigger-c"].IsActive)
	})

	t.Run("Activate one trigger", func(t *testing.T) {
		err := scaleExecutor.updateTriggersActivity(
			context.Background(),
			scaleExecutor.logger,
			scaledObject,
			[]string{"trigger-a"},
		)
		assert.NoError(t, err)

		// Only trigger-a should be active
		assert.True(t, scaledObject.Status.TriggersActivity["trigger-a"].IsActive)
		assert.False(t, scaledObject.Status.TriggersActivity["trigger-b"].IsActive)
		assert.False(t, scaledObject.Status.TriggersActivity["trigger-c"].IsActive)
	})

	t.Run("Activate multiple triggers", func(t *testing.T) {
		err := scaleExecutor.updateTriggersActivity(
			context.Background(),
			scaleExecutor.logger,
			scaledObject,
			[]string{"trigger-a", "trigger-c"},
		)
		assert.NoError(t, err)

		// trigger-a and trigger-c should be active
		assert.True(t, scaledObject.Status.TriggersActivity["trigger-a"].IsActive)
		assert.False(t, scaledObject.Status.TriggersActivity["trigger-b"].IsActive)
		assert.True(t, scaledObject.Status.TriggersActivity["trigger-c"].IsActive)
	})

	t.Run("Change active triggers", func(t *testing.T) {
		err := scaleExecutor.updateTriggersActivity(
			context.Background(),
			scaleExecutor.logger,
			scaledObject,
			[]string{"trigger-b"},
		)
		assert.NoError(t, err)

		// Only trigger-b should be active now
		assert.False(t, scaledObject.Status.TriggersActivity["trigger-a"].IsActive)
		assert.True(t, scaledObject.Status.TriggersActivity["trigger-b"].IsActive)
		assert.False(t, scaledObject.Status.TriggersActivity["trigger-c"].IsActive)
	})

	t.Run("Activate all triggers", func(t *testing.T) {
		err := scaleExecutor.updateTriggersActivity(
			context.Background(),
			scaleExecutor.logger,
			scaledObject,
			[]string{"trigger-a", "trigger-b", "trigger-c"},
		)
		assert.NoError(t, err)

		// All triggers should be active
		assert.True(t, scaledObject.Status.TriggersActivity["trigger-a"].IsActive)
		assert.True(t, scaledObject.Status.TriggersActivity["trigger-b"].IsActive)
		assert.True(t, scaledObject.Status.TriggersActivity["trigger-c"].IsActive)
	})

	t.Run("Deactivate all triggers", func(t *testing.T) {
		err := scaleExecutor.updateTriggersActivity(
			context.Background(),
			scaleExecutor.logger,
			scaledObject,
			[]string{},
		)
		assert.NoError(t, err)

		// All triggers should be inactive
		assert.False(t, scaledObject.Status.TriggersActivity["trigger-a"].IsActive)
		assert.False(t, scaledObject.Status.TriggersActivity["trigger-b"].IsActive)
		assert.False(t, scaledObject.Status.TriggersActivity["trigger-c"].IsActive)
	})

	t.Run("Add new trigger to ExternalMetricNames", func(t *testing.T) {
		// Add a new trigger
		scaledObject.Status.ExternalMetricNames = []string{"trigger-a", "trigger-b", "trigger-c", "trigger-d"}

		err := scaleExecutor.updateTriggersActivity(
			context.Background(),
			scaleExecutor.logger,
			scaledObject,
			[]string{"trigger-d"},
		)
		assert.NoError(t, err)

		// New trigger should exist and be active
		assert.Len(t, scaledObject.Status.TriggersActivity, 4)
		assert.Contains(t, scaledObject.Status.TriggersActivity, "trigger-d")
		assert.True(t, scaledObject.Status.TriggersActivity["trigger-d"].IsActive)

		// Other triggers should be inactive
		assert.False(t, scaledObject.Status.TriggersActivity["trigger-a"].IsActive)
		assert.False(t, scaledObject.Status.TriggersActivity["trigger-b"].IsActive)
		assert.False(t, scaledObject.Status.TriggersActivity["trigger-c"].IsActive)
	})

	t.Run("Remove trigger from ExternalMetricNames", func(t *testing.T) {
		// Remove trigger-b
		scaledObject.Status.ExternalMetricNames = []string{"trigger-a", "trigger-c", "trigger-d"}

		err := scaleExecutor.updateTriggersActivity(
			context.Background(),
			scaleExecutor.logger,
			scaledObject,
			[]string{"trigger-a", "trigger-c"},
		)
		assert.NoError(t, err)

		// Removed trigger should no longer exist in activity map
		assert.Len(t, scaledObject.Status.TriggersActivity, 3)
		assert.NotContains(t, scaledObject.Status.TriggersActivity, "trigger-b")

		// Remaining triggers should have correct state
		assert.True(t, scaledObject.Status.TriggersActivity["trigger-a"].IsActive)
		assert.True(t, scaledObject.Status.TriggersActivity["trigger-c"].IsActive)
		assert.False(t, scaledObject.Status.TriggersActivity["trigger-d"].IsActive)
	})

	t.Run("Remove multiple triggers at once", func(t *testing.T) {
		// Remove trigger-c and trigger-d, only keep trigger-a
		scaledObject.Status.ExternalMetricNames = []string{"trigger-a"}

		err := scaleExecutor.updateTriggersActivity(
			context.Background(),
			scaleExecutor.logger,
			scaledObject,
			[]string{"trigger-a"},
		)
		assert.NoError(t, err)

		// Only trigger-a should remain
		assert.Len(t, scaledObject.Status.TriggersActivity, 1)
		assert.Contains(t, scaledObject.Status.TriggersActivity, "trigger-a")
		assert.True(t, scaledObject.Status.TriggersActivity["trigger-a"].IsActive)

		// Removed triggers should not exist
		assert.NotContains(t, scaledObject.Status.TriggersActivity, "trigger-c")
		assert.NotContains(t, scaledObject.Status.TriggersActivity, "trigger-d")
	})

	t.Run("Add and remove triggers simultaneously", func(t *testing.T) {
		// Replace trigger-a with trigger-x and trigger-y
		scaledObject.Status.ExternalMetricNames = []string{"trigger-x", "trigger-y"}

		err := scaleExecutor.updateTriggersActivity(
			context.Background(),
			scaleExecutor.logger,
			scaledObject,
			[]string{"trigger-x"},
		)
		assert.NoError(t, err)

		// New triggers should exist
		assert.Len(t, scaledObject.Status.TriggersActivity, 2)
		assert.Contains(t, scaledObject.Status.TriggersActivity, "trigger-x")
		assert.Contains(t, scaledObject.Status.TriggersActivity, "trigger-y")

		// New active trigger should be active
		assert.True(t, scaledObject.Status.TriggersActivity["trigger-x"].IsActive)
		assert.False(t, scaledObject.Status.TriggersActivity["trigger-y"].IsActive)

		// Old trigger should be removed
		assert.NotContains(t, scaledObject.Status.TriggersActivity, "trigger-a")
	})

	t.Run("No update when nothing changes", func(t *testing.T) {
		// Get current state
		initialActivity := make(map[string]v1alpha1.TriggerActivityStatus)
		for k, v := range scaledObject.Status.TriggersActivity {
			initialActivity[k] = v
		}

		// Call with same state
		err := scaleExecutor.updateTriggersActivity(
			context.Background(),
			scaleExecutor.logger,
			scaledObject,
			[]string{"trigger-x"},
		)
		assert.NoError(t, err)

		// State should be identical
		assert.Equal(t, len(initialActivity), len(scaledObject.Status.TriggersActivity))
		for k, v := range initialActivity {
			assert.Equal(t, v.IsActive, scaledObject.Status.TriggersActivity[k].IsActive)
		}
	})
}

func TestTriggersActivityStateTransitions(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := record.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	statusWriter := mock_client.NewMockStatusWriter(ctrl)

	scaleExecutorInterface := NewScaleExecutor(client, mockScaleClient, nil, recorder)
	scaleExecutor := scaleExecutorInterface.(*scaleExecutor)

	// Mock the client's status update - count how many times we actually call Patch
	client.EXPECT().Status().Return(statusWriter).AnyTimes()

	// Track patch calls
	patchCallCount := 0
	statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, obj interface{}, patch interface{}, opts ...interface{}) error {
			patchCallCount++
			return nil
		},
	).AnyTimes()

	scaledObject := &v1alpha1.ScaledObject{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-so",
			Namespace: "test",
		},
		Status: v1alpha1.ScaledObjectStatus{
			ScaleTargetKind:     "apps/v1.Deployment",
			ExternalMetricNames: []string{"trigger1", "trigger2"},
		},
	}

	t.Run("Status updates only on state changes", func(t *testing.T) {
		patchCallCount = 0

		// First update - should trigger a status update (initial state)
		err := scaleExecutor.updateTriggersActivity(
			context.Background(),
			scaleExecutor.logger,
			scaledObject,
			[]string{"trigger1"},
		)
		assert.NoError(t, err)
		assert.Equal(t, 1, patchCallCount, "Should update on initial state")

		// Second update with same state - should NOT trigger a status update
		err = scaleExecutor.updateTriggersActivity(
			context.Background(),
			scaleExecutor.logger,
			scaledObject,
			[]string{"trigger1"},
		)
		assert.NoError(t, err)
		assert.Equal(t, 1, patchCallCount, "Should NOT update when state unchanged")

		// Third update with different state - should trigger a status update
		err = scaleExecutor.updateTriggersActivity(
			context.Background(),
			scaleExecutor.logger,
			scaledObject,
			[]string{"trigger2"},
		)
		assert.NoError(t, err)
		assert.Equal(t, 2, patchCallCount, "Should update when state changes")

		// Fourth update with same state again - should NOT trigger a status update
		err = scaleExecutor.updateTriggersActivity(
			context.Background(),
			scaleExecutor.logger,
			scaledObject,
			[]string{"trigger2"},
		)
		assert.NoError(t, err)
		assert.Equal(t, 2, patchCallCount, "Should NOT update when state unchanged again")
	})
}
