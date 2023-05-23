/*
Copyright 2023 The KEDA Authors

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

package v1alpha1

import (
	"testing"
)

func TestScaledJob(t *testing.T) {
	tests := []struct {
		name        string
		expectedMax int64
		expectedMin int64
		maxReplicas *int32
		minReplicas *int32
	}{
		{
			name:        "MaxReplicaCount is set to 10 and MinReplicaCount is set to 0",
			expectedMax: 10,
			expectedMin: 0,
			maxReplicas: int32Ptr(10),
			minReplicas: int32Ptr(0),
		},
		{
			name:        "MaxReplicaCount is set to 10 and MinReplicaCount is nil",
			expectedMax: 10,
			expectedMin: defaultScaledJobMinReplicaCount,
			maxReplicas: int32Ptr(10),
			minReplicas: nil,
		},
		{
			name:        "MaxReplicaCount is set to 10 and MinReplicaCount is set to 1",
			expectedMax: 9,
			expectedMin: 1,
			maxReplicas: int32Ptr(10),
			minReplicas: int32Ptr(1),
		},
		{
			name:        "MaxReplicaCount is nil and MinReplicaCount is set to 1",
			expectedMax: defaultScaledJobMaxReplicaCount,
			expectedMin: 1,
			maxReplicas: nil,
			minReplicas: int32Ptr(1),
		},
		{
			name:        "MaxReplicaCount is nil and MinReplicaCount nil",
			expectedMax: defaultScaledJobMaxReplicaCount,
			expectedMin: defaultScaledJobMinReplicaCount,
			maxReplicas: nil,
			minReplicas: nil,
		},
		{
			name:        "MaxReplicaCount is set to 1 and MinReplicaCount is set to 10",
			expectedMax: 1,
			expectedMin: 1,
			maxReplicas: int32Ptr(1),
			minReplicas: int32Ptr(10),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			scaledJob := &ScaledJob{
				Spec: ScaledJobSpec{
					MaxReplicaCount: test.maxReplicas,
					MinReplicaCount: test.minReplicas,
				},
			}

			if scaledJob.MaxReplicaCount() != test.expectedMax {
				t.Errorf("MaxReplicaCount()=%d, expected %d", scaledJob.MaxReplicaCount(), test.expectedMax)
			}

			if scaledJob.MinReplicaCount() != test.expectedMin {
				t.Errorf("MinReplicaCount()=%d, expected %d", scaledJob.MinReplicaCount(), test.expectedMin)
			}
		})
	}
}

func int32Ptr(i int32) *int32 {
	return &i
}
