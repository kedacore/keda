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

package v1alpha1

import (
	"testing"
)

func TestValidateScalingModifiersTargetActivationTarget(t *testing.T) {
	tests := []struct {
		name             string
		activationTarget string
		wantErr          bool
		noTarget         bool
	}{
		{name: "empty activationTarget is allowed", activationTarget: "", wantErr: false},
		{name: "invalid activationTarget is rejected even without target", activationTarget: "five", wantErr: true, noTarget: true},
		{name: "zero is allowed", activationTarget: "0", wantErr: false},
		{name: "integer is allowed", activationTarget: "5", wantErr: false},
		{name: "decimal is allowed", activationTarget: "1.5", wantErr: false},
		{name: "non-numeric string is rejected", activationTarget: "five", wantErr: true},
		{name: "quantity string is rejected", activationTarget: "1Gi", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := "2"
			if tt.noTarget {
				target = ""
			}
			so := &ScaledObject{
				Spec: ScaledObjectSpec{
					Advanced: &AdvancedConfig{
						ScalingModifiers: ScalingModifiers{
							Formula:          "trig_one",
							Target:           target,
							ActivationTarget: tt.activationTarget,
						},
					},
				},
			}

			err := validateScalingModifiersTarget(so)
			if tt.wantErr && err == nil {
				t.Errorf("expected an error for activationTarget %q, got nil", tt.activationTarget)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error for activationTarget %q, got %v", tt.activationTarget, err)
			}
		})
	}
}
