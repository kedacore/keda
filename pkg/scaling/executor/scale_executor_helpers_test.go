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

package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

func int32Ptr(v int32) *int32 { return &v }

func TestGetIdleOrMinimumReplicaCount(t *testing.T) {
	tests := []struct {
		name             string
		scaledObject     *kedav1alpha1.ScaledObject
		expectedIsIdle   bool
		expectedReplicas int32
	}{
		{
			name: "nil IdleReplicaCount and nil MinReplicaCount returns false and 0",
			scaledObject: &kedav1alpha1.ScaledObject{
				Spec: kedav1alpha1.ScaledObjectSpec{},
			},
			expectedIsIdle:   false,
			expectedReplicas: 0,
		},
		{
			name: "IdleReplicaCount set returns true with its value",
			scaledObject: &kedav1alpha1.ScaledObject{
				Spec: kedav1alpha1.ScaledObjectSpec{
					IdleReplicaCount: int32Ptr(0),
				},
			},
			expectedIsIdle:   true,
			expectedReplicas: 0,
		},
		{
			name: "IdleReplicaCount non-zero returns true with its value",
			scaledObject: &kedav1alpha1.ScaledObject{
				Spec: kedav1alpha1.ScaledObjectSpec{
					IdleReplicaCount: int32Ptr(3),
				},
			},
			expectedIsIdle:   true,
			expectedReplicas: 3,
		},
		{
			name: "MinReplicaCount set with nil IdleReplicaCount returns false with its value",
			scaledObject: &kedav1alpha1.ScaledObject{
				Spec: kedav1alpha1.ScaledObjectSpec{
					MinReplicaCount: int32Ptr(2),
				},
			},
			expectedIsIdle:   false,
			expectedReplicas: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isIdle, replicas := getIdleOrMinimumReplicaCount(tt.scaledObject)
			assert.Equal(t, tt.expectedIsIdle, isIdle)
			assert.Equal(t, tt.expectedReplicas, replicas)
		})
	}
}

func TestIsJobFinished(t *testing.T) {
	e := &scaleExecutor{}

	tests := []struct {
		name     string
		job      batchv1.Job
		expected bool
	}{
		{
			name:     "no conditions returns false",
			job:      batchv1.Job{},
			expected: false,
		},
		{
			name: "JobComplete with ConditionTrue returns true",
			job: batchv1.Job{
				Status: batchv1.JobStatus{
					Conditions: []batchv1.JobCondition{
						{Type: batchv1.JobComplete, Status: corev1.ConditionTrue},
					},
				},
			},
			expected: true,
		},
		{
			name: "JobFailed with ConditionTrue returns true",
			job: batchv1.Job{
				Status: batchv1.JobStatus{
					Conditions: []batchv1.JobCondition{
						{Type: batchv1.JobFailed, Status: corev1.ConditionTrue},
					},
				},
			},
			expected: true,
		},
		{
			name: "JobComplete with ConditionFalse returns false",
			job: batchv1.Job{
				Status: batchv1.JobStatus{
					Conditions: []batchv1.JobCondition{
						{Type: batchv1.JobComplete, Status: corev1.ConditionFalse},
					},
				},
			},
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, e.isJobFinished(&tt.job))
		})
	}
}

func TestGetFinishedJobConditionType(t *testing.T) {
	e := &scaleExecutor{}

	tests := []struct {
		name     string
		job      batchv1.Job
		expected batchv1.JobConditionType
	}{
		{
			name:     "no conditions returns empty string",
			job:      batchv1.Job{},
			expected: "",
		},
		{
			name: "JobComplete with ConditionTrue returns JobComplete",
			job: batchv1.Job{
				Status: batchv1.JobStatus{
					Conditions: []batchv1.JobCondition{
						{Type: batchv1.JobComplete, Status: corev1.ConditionTrue},
					},
				},
			},
			expected: batchv1.JobComplete,
		},
		{
			name: "JobFailed with ConditionTrue returns JobFailed",
			job: batchv1.Job{
				Status: batchv1.JobStatus{
					Conditions: []batchv1.JobCondition{
						{Type: batchv1.JobFailed, Status: corev1.ConditionTrue},
					},
				},
			},
			expected: batchv1.JobFailed,
		},
		{
			name: "condition with ConditionFalse returns empty string",
			job: batchv1.Job{
				Status: batchv1.JobStatus{
					Conditions: []batchv1.JobCondition{
						{Type: batchv1.JobComplete, Status: corev1.ConditionFalse},
					},
				},
			},
			expected: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, e.getFinishedJobConditionType(&tt.job))
		})
	}
}
