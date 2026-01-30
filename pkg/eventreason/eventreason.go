/*
Copyright 2020 The KEDA Authors

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

package eventreason

const (
	// ScaledObjectReady is for event when a new ScaledObject is ready
	ScaledObjectReady = "ScaledObjectReady"

	// ScaledJobReady is for event when a new ScaledJob is ready
	ScaledJobReady = "ScaledJobReady"

	// ScaledObjectCheckFailed is for event when ScaledObject validation check fails
	ScaledObjectCheckFailed = "ScaledObjectCheckFailed"

	// ScaledJobCheckFailed is for event when ScaledJob validation check fails
	ScaledJobCheckFailed = "ScaledJobCheckFailed"

	// ScaledObjectUpdateFailed is for event when ScaledObject update status fails
	ScaledObjectUpdateFailed = "ScaledObjectUpdateFailed"

	// ScaledJobUpdateFailed is for event when ScaledJob update status fails
	ScaledJobUpdateFailed = "ScaledJobUpdateFailed"

	// ScaledObjectDeleted is for event when ScaledObject is deleted
	ScaledObjectDeleted = "ScaledObjectDeleted"

	// ScaledObjectPaused is for event when ScaledObject is paused
	ScaledObjectPaused = "ScaledObjectPaused"

	// ScaledObjectUnpaused is for event when ScaledObject is unpaused
	ScaledObjectUnpaused = "ScaledObjectUnpaused"

	// ScaledObjectFallbackActivated is for event when ScaledObject fallback is activated
	ScaledObjectFallbackActivated = "ScaledObjectFallbackActivated"

	// ScaledObjectFallbackDeactivated is for event when ScaledObject fallback is deactivated
	ScaledObjectFallbackDeactivated = "ScaledObjectFallbackDeactivated"

	// ScaledObjectTriggerExcluded is for event when a trigger is excluded from formula evaluation due to failure threshold
	ScaledObjectTriggerExcluded = "ScaledObjectTriggerExcluded"

	// ScaledObjectTriggerRestored is for event when a previously excluded trigger is restored in formula evaluation
	ScaledObjectTriggerRestored = "ScaledObjectTriggerRestored"

	// ScaledJobDeleted is for event when ScaledJob is deleted
	ScaledJobDeleted = "ScaledJobDeleted"

	// ScaledJobPaused is for event when ScaledJob is paused
	ScaledJobPaused = "ScaledJobPaused"

	// ScaledJobUnpaused is for event when ScaledJob is unpaused
	ScaledJobUnpaused = "ScaledJobUnpaused"

	// ScaledJobPauseFailed is for event when pausing ScaledJob fails
	ScaledJobPauseFailed = "ScaledJobPauseFailed"

	// ScaledJobRolloutCleanupStarted is for event when ScaledJob rollout cleanup starts
	ScaledJobRolloutCleanupStarted = "ScaledJobRolloutCleanupStarted"

	// ScaledJobRolloutCleanupCompleted is for event when ScaledJob rollout cleanup completes
	ScaledJobRolloutCleanupCompleted = "ScaledJobRolloutCleanupCompleted"

	// ScaledJobRolloutCleanupFailed is for event when ScaledJob rollout cleanup fails
	ScaledJobRolloutCleanupFailed = "ScaledJobRolloutCleanupFailed"

	// ScaledJobActive is for event when ScaledJob becomes active
	ScaledJobActive = "ScaledJobActive"

	// ScaledJobInactive is for event when ScaledJob becomes inactive
	ScaledJobInactive = "ScaledJobInactive"

	// KEDAScalersInfo is for event when Scaler has additional info
	KEDAScalersInfo = "KEDAScalerInfo"

	// KEDAScalersStarted is for event when scalers watch started for ScaledObject or ScaledJob
	KEDAScalersStarted = "KEDAScalersStarted"

	// KEDAScalersStopped is for event when scalers watch was stopped for ScaledObject or ScaledJob
	KEDAScalersStopped = "KEDAScalersStopped"

	// KEDAScalerFailed is for event when a scaler fails for a ScaledJob or a ScaledObject
	KEDAScalerFailed = "KEDAScalerFailed"

	// KEDAMetricSourceFailed is for event when a scaler fails as metric source for custom formula
	KEDAMetricSourceFailed = "KEDAMetricSourceFailed"

	// KEDAScaleTargetActivated is for event when the scale target of ScaledObject was activated
	KEDAScaleTargetActivated = "KEDAScaleTargetActivated"

	// KEDAScaleTargetDeactivated is for event when the scale target for ScaledObject was deactivated
	KEDAScaleTargetDeactivated = "KEDAScaleTargetDeactivated"

	// KEDAScaleTargetActivationFailed is for event when the activation the scale target for ScaledObject fails
	KEDAScaleTargetActivationFailed = "KEDAScaleTargetActivationFailed"

	// KEDAScaleTargetDeactivationFailed is for event when the deactivation of the scale target for ScaledObject fails
	KEDAScaleTargetDeactivationFailed = "KEDAScaleTargetDeactivationFailed"

	// KEDAJobCreateFailed is for event when a Job creation fails for ScaledJob
	KEDAJobCreateFailed = "KEDAJobCreateFailed"

	// KEDAJobsCreated is for event when jobs for ScaledJob are created
	KEDAJobsCreated = "KEDAJobsCreated"

	// TriggerAuthenticationDeleted is for event when a TriggerAuthentication is deleted
	TriggerAuthenticationDeleted = "TriggerAuthenticationDeleted"

	// TriggerAuthenticationAdded is for event when a TriggerAuthentication is added
	TriggerAuthenticationAdded = "TriggerAuthenticationAdded"

	// TriggerAuthenticationFailed is for event when a TriggerAuthentication occurs error
	TriggerAuthenticationFailed = "TriggerAuthenticationFailed"

	// TriggerAuthenticationUpdated is for event when a TriggerAuthentication is updated
	TriggerAuthenticationUpdated = "ClusterTriggerAuthenticationUpdated"

	// ClusterTriggerAuthenticationDeleted is for event when a ClusterTriggerAuthentication is deleted
	ClusterTriggerAuthenticationDeleted = "ClusterTriggerAuthenticationDeleted"

	// ClusterTriggerAuthenticationAdded is for event when a ClusterTriggerAuthentication is added
	ClusterTriggerAuthenticationAdded = "ClusterTriggerAuthenticationAdded"

	// ClusterTriggerAuthenticationFailed is for event when a ClusterTriggerAuthentication occurs error
	ClusterTriggerAuthenticationFailed = "ClusterTriggerAuthenticationFailed"

	// ClusterTriggerAuthenticationUpdated is for event when a ClusterTriggerAuthentication is updated
	ClusterTriggerAuthenticationUpdated = "ClusterTriggerAuthenticationUpdated"
)
