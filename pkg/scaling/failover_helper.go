/*
Copyright 2021 The KEDA Authors

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

package scaling

import (
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

// isFailoverEnabled checks if failover is properly configured for a ScaledObject.
// Returns true if all failover requirements are met:
// - Primary trigger (index 0) has fallback configuration
// - Behavior is set to "failover"
// - FailoverThresholds are configured
// - At least 2 triggers are defined (primary and secondary)
func isFailoverEnabled(scaledObject *kedav1alpha1.ScaledObject) bool {
	if len(scaledObject.Spec.Triggers) < 2 {
		return false
	}

	// Check primary trigger's fallback configuration
	primaryTrigger := scaledObject.Spec.Triggers[0]
	if primaryTrigger.Fallback == nil {
		return false
	}
	if primaryTrigger.Fallback.Behavior != "failover" {
		return false
	}
	if primaryTrigger.Fallback.FailoverThresholds == nil {
		return false
	}
	return true
}

// getActiveTriggerIndex implements failover trigger selection with debouncing.
//
// Failover Logic (Primary → Secondary):
// - Switches to secondary (index 1) when primary failures >= FailAfter threshold
// - Prevents premature failover on transient errors
//
// Recovery Logic (Secondary → Primary):
// - Returns to primary (index 0) when failures < RecoverAfter threshold
// - Uses optimistic approach: attempts primary, re-fails if still unhealthy
// - Avoids need for separate success counter tracking
func getActiveTriggerIndex(scaledObject *kedav1alpha1.ScaledObject, currentTriggerIndex int) int {
	if !isFailoverEnabled(scaledObject) {
		return 0 // Always use primary when failover disabled
	}

	failAfter := scaledObject.Spec.Triggers[0].Fallback.FailoverThresholds.FailAfter
	recoverAfter := scaledObject.Spec.Triggers[0].Fallback.FailoverThresholds.RecoverAfter

	// Find primary trigger's health by looking for metrics starting with "s0-"
	// (trigger index 0) in the health map
	var primaryHealth *kedav1alpha1.HealthStatus
	for metricName, health := range scaledObject.Status.Health {
		if len(metricName) >= 3 && metricName[:3] == "s0-" {
			healthCopy := health
			primaryHealth = &healthCopy
			break
		}
	}

	if primaryHealth == nil || primaryHealth.NumberOfFailures == nil {
		// No health data yet for primary, default to primary
		return 0
	}

	// Failover to secondary if primary failures exceed threshold
	if *primaryHealth.NumberOfFailures >= failAfter {
		return 1 // Use secondary
	}

	// Optimistic recovery: If failures dropped below RecoverAfter threshold, return to primary
	// This implements a simple recovery heuristic without additional state tracking
	// If primary is still unhealthy, failures will increment again and trigger failover
	if currentTriggerIndex == 1 && *primaryHealth.NumberOfFailures < recoverAfter {
		return 0 // Recover to primary
	}

	return 0 // Default to primary
}

// shouldQueryTrigger determines if a specific trigger should be queried for metrics.
// When failover is enabled, only the active trigger (determined by getActiveTriggerIndex)
// should be queried. When failover is disabled, all triggers should be queried.
//
// Parameters:
// - scaledObject: The ScaledObject being evaluated
// - triggerIndex: The index of the trigger being considered for querying
// - activeTriggerIndex: The currently active trigger index (from getActiveTriggerIndex)
//
// Returns true if the trigger should be queried, false otherwise.
func shouldQueryTrigger(scaledObject *kedav1alpha1.ScaledObject, triggerIndex int, activeTriggerIndex int) bool {
	// When failover is disabled, query all triggers (backward compatible behavior)
	if !isFailoverEnabled(scaledObject) {
		return true
	}

	// Only query the active trigger when failover is enabled
	return triggerIndex == activeTriggerIndex
}
