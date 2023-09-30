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
	"fmt"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
)

// ScaleTriggers reference the scaler that will be used
type ScaleTriggers struct {
	Type string `json:"type"`
	// +optional
	Name string `json:"name,omitempty"`

	UseCachedMetrics bool `json:"useCachedMetrics,omitempty"`

	Metadata map[string]string `json:"metadata"`
	// +optional
	AuthenticationRef *AuthenticationRef `json:"authenticationRef,omitempty"`
	// +optional
	MetricType autoscalingv2.MetricTargetType `json:"metricType,omitempty"`
}

// AuthenticationRef points to the TriggerAuthentication or ClusterTriggerAuthentication object that
// is used to authenticate the scaler with the environment
type AuthenticationRef struct {
	Name string `json:"name"`
	// Kind of the resource being referred to. Defaults to TriggerAuthentication.
	// +optional
	Kind string `json:"kind,omitempty"`
}

// ValidateTriggers checks that general trigger metadata are valid, it checks:
// - triggerNames in ScaledObject are unique
// - useCachedMetrics is defined only for a supported triggers
func ValidateTriggers(triggers []ScaleTriggers) error {
	triggersCount := len(triggers)
	if triggers != nil && triggersCount > 0 {
		triggerNames := make(map[string]bool, triggersCount)
		for i := 0; i < triggersCount; i++ {
			trigger := triggers[i]

			if trigger.UseCachedMetrics {
				if trigger.Type == "cpu" || trigger.Type == "memory" || trigger.Type == "cron" {
					return fmt.Errorf("property \"useCachedMetrics\" is not supported for %q scaler", trigger.Type)
				}
			}

			name := trigger.Name
			if name != "" {
				if _, found := triggerNames[name]; found {
					// found duplicate name
					return fmt.Errorf("triggerName %q is defined multiple times in the ScaledObject, but it must be unique", name)
				}
				triggerNames[name] = true
			}
		}
	}

	return nil
}
