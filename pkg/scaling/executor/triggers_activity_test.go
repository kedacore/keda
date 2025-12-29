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
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"

	"github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/mock/mock_client"
	"github.com/kedacore/keda/v2/pkg/mock/mock_scale"
)

func TestSetActiveConditionWithTriggers(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := record.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	statusWriter := mock_client.NewMockStatusWriter(ctrl)

	scaleExecutorInterface := NewScaleExecutor(client, mockScaleClient, nil, recorder)
	scaleExecutor := scaleExecutorInterface.(*scaleExecutor)

	tests := []struct {
		name                string
		activeTriggers      []string
		externalMetricNames []string
		expectedIsActive    bool
		expectedActivityMap map[string]v1alpha1.TriggerActivityStatus
		conditionStatus     v1.ConditionStatus
	}{
		{
			name:                "Active triggers should be marked as active",
			activeTriggers:      []string{"trigger1", "trigger2"},
			externalMetricNames: []string{"trigger1", "trigger2", "trigger3"},
			expectedIsActive:    true,
			expectedActivityMap: map[string]v1alpha1.TriggerActivityStatus{
				"trigger1": {IsActive: true},
				"trigger2": {IsActive: true},
				"trigger3": {IsActive: false},
			},
			conditionStatus: v1.ConditionTrue,
		},
		{
			name:                "Inactive triggers should be marked as inactive",
			activeTriggers:      []string{},
			externalMetricNames: []string{"trigger1", "trigger2"},
			expectedIsActive:    false,
			expectedActivityMap: map[string]v1alpha1.TriggerActivityStatus{
				"trigger1": {IsActive: false},
				"trigger2": {IsActive: false},
			},
			conditionStatus: v1.ConditionFalse,
		},
		{
			name:                "Mixed active and inactive triggers",
			activeTriggers:      []string{"trigger2"},
			externalMetricNames: []string{"trigger1", "trigger2", "trigger3"},
			expectedIsActive:    true,
			expectedActivityMap: map[string]v1alpha1.TriggerActivityStatus{
				"trigger1": {IsActive: false},
				"trigger2": {IsActive: true},
				"trigger3": {IsActive: false},
			},
			conditionStatus: v1.ConditionTrue,
		},
		{
			name:                "Empty external metric names",
			activeTriggers:      []string{"trigger1"},
			externalMetricNames: []string{},
			expectedIsActive:    true,
			expectedActivityMap: map[string]v1alpha1.TriggerActivityStatus{},
			conditionStatus:     v1.ConditionTrue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scaledObject := &v1alpha1.ScaledObject{
				ObjectMeta: v1.ObjectMeta{
					Name:      "test-scaledobject",
					Namespace: "test-namespace",
				},
				Status: v1alpha1.ScaledObjectStatus{
					ExternalMetricNames: tt.externalMetricNames,
				},
			}
			scaledObject.Status.Conditions = *v1alpha1.GetInitializedConditions()

			client.EXPECT().Status().Return(statusWriter)
			statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

			// Call the function under test
			err := scaleExecutor.setActiveCondition(
				context.TODO(),
				scaleExecutor.logger,
				scaledObject,
				tt.conditionStatus,
				"TestReason",
				"Test message",
				tt.activeTriggers,
			)

			// Assertions
			assert.NoError(t, err)

			// Check active condition
			activeCondition := scaledObject.Status.Conditions.GetActiveCondition()
			if tt.conditionStatus == v1.ConditionTrue {
				assert.True(t, activeCondition.IsTrue())
			} else {
				assert.True(t, activeCondition.IsFalse())
			}

			// Check triggers activity map
			assert.NotNil(t, scaledObject.Status.TriggersActivity)
			assert.Equal(t, len(tt.expectedActivityMap), len(scaledObject.Status.TriggersActivity))

			for triggerName, expectedStatus := range tt.expectedActivityMap {
				actualStatus, exists := scaledObject.Status.TriggersActivity[triggerName]
				assert.True(t, exists, "Trigger %s should exist in activity map", triggerName)
				assert.Equal(t, expectedStatus.IsActive, actualStatus.IsActive,
					"Trigger %s activity status should match expected", triggerName)

				// If trigger is active, LastActiveTime should be set
				if expectedStatus.IsActive {
					assert.NotNil(t, actualStatus.LastActiveTime,
						"Active trigger %s should have LastActiveTime set", triggerName)
				}
			}
		})
	}
}

func TestRequestScaleWithTriggersActivity(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := record.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)
	statusWriter := mock_client.NewMockStatusWriter(ctrl)

	scaleExecutor := NewScaleExecutor(client, mockScaleClient, nil, recorder)

	activeTriggers := []string{"prometheus", "rabbitmq"}
	externalMetricNames := []string{"prometheus", "rabbitmq", "kafka"}

	scaledObject := v1alpha1.ScaledObject{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-scaledobject",
			Namespace: "test-namespace",
		},
		Spec: v1alpha1.ScaledObjectSpec{
			ScaleTargetRef: &v1alpha1.ScaleTarget{
				Name: "test-deployment",
			},
			MinReplicaCount: func() *int32 { i := int32(1); return &i }(),
		},
		Status: v1alpha1.ScaledObjectStatus{
			ScaleTargetGVKR: &v1alpha1.GroupVersionKindResource{
				Group: "apps",
				Kind:  "Deployment",
			},
			ExternalMetricNames: externalMetricNames,
		},
	}
	scaledObject.Status.Conditions = *v1alpha1.GetInitializedConditions()

	currentReplicas := int32(0)

	client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Replicas: &currentReplicas,
		},
	})

	scale := &autoscalingv1.Scale{
		Spec: autoscalingv1.ScaleSpec{
			Replicas: currentReplicas,
		},
	}

	mockScaleClient.EXPECT().Scales(gomock.Any()).Return(mockScaleInterface).Times(2)
	mockScaleInterface.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(scale, nil)
	mockScaleInterface.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(scale, nil)

	client.EXPECT().Status().Return(statusWriter).AnyTimes()
	statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	// Test active scaling
	scaleExecutor.RequestScale(context.TODO(), &scaledObject, true, false, &ScaleExecutorOptions{
		ActiveTriggers: activeTriggers,
	})

	// Verify active condition is set correctly
	activeCondition := scaledObject.Status.Conditions.GetActiveCondition()
	assert.True(t, activeCondition.IsTrue())

	// Verify triggers activity is properly tracked
	assert.NotNil(t, scaledObject.Status.TriggersActivity)

	// Check specific trigger activities
	prometheusActivity, exists := scaledObject.Status.TriggersActivity["prometheus"]
	assert.True(t, exists)
	assert.True(t, prometheusActivity.IsActive)
	assert.NotNil(t, prometheusActivity.LastActiveTime)

	rabbitmqActivity, exists := scaledObject.Status.TriggersActivity["rabbitmq"]
	assert.True(t, exists)
	assert.True(t, rabbitmqActivity.IsActive)
	assert.NotNil(t, rabbitmqActivity.LastActiveTime)

	kafkaActivity, exists := scaledObject.Status.TriggersActivity["kafka"]
	assert.True(t, exists)
	assert.False(t, kafkaActivity.IsActive)
}

func TestRequestScaleWithInactiveTriggersActivity(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := record.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)
	statusWriter := mock_client.NewMockStatusWriter(ctrl)

	scaleExecutor := NewScaleExecutor(client, mockScaleClient, nil, recorder)

	activeTriggers := []string{} // No active triggers
	externalMetricNames := []string{"prometheus", "rabbitmq"}

	minReplicas := int32(0)
	scaledObject := v1alpha1.ScaledObject{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-scaledobject",
			Namespace: "test-namespace",
		},
		Spec: v1alpha1.ScaledObjectSpec{
			ScaleTargetRef: &v1alpha1.ScaleTarget{
				Name: "test-deployment",
			},
			MinReplicaCount: &minReplicas,
		},
		Status: v1alpha1.ScaledObjectStatus{
			ScaleTargetGVKR: &v1alpha1.GroupVersionKindResource{
				Group: "apps",
				Kind:  "Deployment",
			},
			ExternalMetricNames: externalMetricNames,
		},
	}
	scaledObject.Status.Conditions = *v1alpha1.GetInitializedConditions()

	currentReplicas := int32(1)

	client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Replicas: &currentReplicas,
		},
	})

	scale := &autoscalingv1.Scale{
		Spec: autoscalingv1.ScaleSpec{
			Replicas: currentReplicas,
		},
	}

	mockScaleClient.EXPECT().Scales(gomock.Any()).Return(mockScaleInterface).Times(2)
	mockScaleInterface.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(scale, nil)
	mockScaleInterface.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())

	client.EXPECT().Status().Return(statusWriter).AnyTimes()
	statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	// Test inactive scaling (scale down)
	scaleExecutor.RequestScale(context.TODO(), &scaledObject, false, false, &ScaleExecutorOptions{
		ActiveTriggers: activeTriggers,
	})

	// Verify active condition is set to false
	activeCondition := scaledObject.Status.Conditions.GetActiveCondition()
	assert.True(t, activeCondition.IsFalse())

	// Verify triggers activity is properly tracked (all should be inactive)
	assert.NotNil(t, scaledObject.Status.TriggersActivity)

	// Check that all triggers are marked as inactive
	for _, triggerName := range externalMetricNames {
		triggerActivity, exists := scaledObject.Status.TriggersActivity[triggerName]
		assert.True(t, exists, "Trigger %s should exist in activity map", triggerName)
		assert.False(t, triggerActivity.IsActive, "Trigger %s should be marked as inactive", triggerName)
	}
}

func TestLastActiveTimePreservation(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := record.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	statusWriter := mock_client.NewMockStatusWriter(ctrl)

	scaleExecutorInterface := NewScaleExecutor(client, mockScaleClient, nil, recorder)
	scaleExecutor := scaleExecutorInterface.(*scaleExecutor)

	// Prepare ScaledObject
	scaledObject := &v1alpha1.ScaledObject{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-scaled-object",
			Namespace: "test",
		},
		Status: v1alpha1.ScaledObjectStatus{
			ScaleTargetKind:     "apps/v1.Deployment",
			ExternalMetricNames: []string{"trigger1", "trigger2"},
		},
	}

	// Mock the client's status update
	client.EXPECT().Status().Return(statusWriter).AnyTimes()
	statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	// Step 1: Set triggers as active
	activeTriggers := []string{"trigger1", "trigger2"}
	err := scaleExecutor.setActiveCondition(
		context.Background(),
		scaleExecutor.logger,
		scaledObject,
		v1.ConditionTrue,
		"ScalerActive",
		"Test message",
		activeTriggers,
	)
	assert.NoError(t, err)

	// Verify both triggers are active and have LastActiveTime set
	assert.NotNil(t, scaledObject.Status.TriggersActivity)
	trigger1Activity := scaledObject.Status.TriggersActivity["trigger1"]
	trigger2Activity := scaledObject.Status.TriggersActivity["trigger2"]

	assert.True(t, trigger1Activity.IsActive)
	assert.True(t, trigger2Activity.IsActive)
	assert.NotNil(t, trigger1Activity.LastActiveTime)
	assert.NotNil(t, trigger2Activity.LastActiveTime)

	// Store the LastActiveTime for comparison
	savedTrigger1Time := trigger1Activity.LastActiveTime

	// Step 2: Set only trigger1 as inactive (trigger2 remains active)
	activeTriggers = []string{"trigger2"}
	err = scaleExecutor.setActiveCondition(
		context.Background(),
		scaleExecutor.logger,
		scaledObject,
		v1.ConditionTrue,
		"ScalerActive",
		"Test message",
		activeTriggers,
	)
	assert.NoError(t, err)

	// Step 3: Verify LastActiveTime preservation
	trigger1Activity = scaledObject.Status.TriggersActivity["trigger1"]
	trigger2Activity = scaledObject.Status.TriggersActivity["trigger2"]

	// trigger1 should be inactive but LastActiveTime should be preserved
	assert.False(t, trigger1Activity.IsActive, "trigger1 should be inactive")
	assert.Equal(t, savedTrigger1Time, trigger1Activity.LastActiveTime, "trigger1 LastActiveTime should be preserved")

	// trigger2 should still be active with updated LastActiveTime
	assert.True(t, trigger2Activity.IsActive, "trigger2 should be active")
	assert.NotNil(t, trigger2Activity.LastActiveTime, "trigger2 should have LastActiveTime")

	// Step 4: Set all triggers as inactive
	activeTriggers = []string{}
	err = scaleExecutor.setActiveCondition(
		context.Background(),
		scaleExecutor.logger,
		scaledObject,
		v1.ConditionFalse,
		"ScalerNotActive",
		"Test message",
		activeTriggers,
	)
	assert.NoError(t, err)

	// Step 5: Verify LastActiveTime is still preserved for both triggers
	trigger1Activity = scaledObject.Status.TriggersActivity["trigger1"]
	trigger2Activity = scaledObject.Status.TriggersActivity["trigger2"]

	// Both triggers should be inactive but LastActiveTime should be preserved
	assert.False(t, trigger1Activity.IsActive, "trigger1 should be inactive")
	assert.False(t, trigger2Activity.IsActive, "trigger2 should be inactive")
	assert.Equal(t, savedTrigger1Time, trigger1Activity.LastActiveTime, "trigger1 LastActiveTime should still be preserved")
	assert.NotNil(t, trigger2Activity.LastActiveTime, "trigger2 LastActiveTime should still exist")
}
