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

// CloudEventType contains the list of cloudevent types
// +kubebuilder:validation:Enum=keda.scaledobject.ready.v1;keda.scaledobject.failed.v1;keda.scaledobject.removed.v1;keda.scaledobject.paused.v1;keda.scaledobject.unpaused.v1;keda.scaledjob.ready.v1;keda.scaledjob.failed.v1;keda.scaledjob.removed.v1;keda.scaledjob.paused.v1;keda.scaledjob.unpaused.v1;keda.scaledjob.rolloutcleanup.started.v1;keda.scaledjob.rolloutcleanup.completed.v1;keda.scaledjob.rolloutcleanup.failed.v1;keda.authentication.triggerauthentication.created.v1;keda.authentication.triggerauthentication.updated.v1;keda.authentication.triggerauthentication.removed.v1;keda.authentication.clustertriggerauthentication.created.v1;keda.authentication.clustertriggerauthentication.updated.v1;keda.authentication.clustertriggerauthentication.removed.v1

type CloudEventType string

const (
	// ScaledObjectReadyType is for event when a new ScaledObject is ready
	ScaledObjectReadyType CloudEventType = "keda.scaledobject.ready.v1"

	// ScaledObjectFailedType is for event when creating ScaledObject failed
	ScaledObjectFailedType CloudEventType = "keda.scaledobject.failed.v1"

	// ScaledObjectRemovedType is for event when removed ScaledObject
	ScaledObjectRemovedType CloudEventType = "keda.scaledobject.removed.v1"

	// ScaledObjectPausedType is for event when ScaledObject is paused
	ScaledObjectPausedType CloudEventType = "keda.scaledobject.paused.v1"

	// ScaledObjectUnpausedType is for event when ScaledObject is unpaused
	ScaledObjectUnpausedType CloudEventType = "keda.scaledobject.unpaused.v1"

	// ScaledJobReadyType is for event when a new ScaledJob is ready
	ScaledJobReadyType CloudEventType = "keda.scaledjob.ready.v1"

	// ScaledJobFailedType is for event when creating ScaledJob failed
	ScaledJobFailedType CloudEventType = "keda.scaledjob.failed.v1"

	// ScaledJobRemovedType is for event when removed ScaledJob
	ScaledJobRemovedType CloudEventType = "keda.scaledjob.removed.v1"

	// ScaledJobPausedType is for event when ScaledJob is paused
	ScaledJobPausedType CloudEventType = "keda.scaledjob.paused.v1"

	// ScaledJobUnpausedType is for event when ScaledJob is unpaused
	ScaledJobUnpausedType CloudEventType = "keda.scaledjob.unpaused.v1"

	// ScaledJobRolloutCleanupStartedType is for event when ScaledJob rollout cleanup starts
	ScaledJobRolloutCleanupStartedType CloudEventType = "keda.scaledjob.rolloutcleanup.started.v1"

	// ScaledJobRolloutCleanupCompletedType is for event when ScaledJob rollout cleanup completes
	ScaledJobRolloutCleanupCompletedType CloudEventType = "keda.scaledjob.rolloutcleanup.completed.v1"

	// ScaledJobRolloutCleanupFailedType is for event when ScaledJob rollout cleanup fails
	ScaledJobRolloutCleanupFailedType CloudEventType = "keda.scaledjob.rolloutcleanup.failed.v1"

	// TriggerAuthenticationCreatedType is for event when a new TriggerAuthentication is created
	TriggerAuthenticationCreatedType CloudEventType = "keda.authentication.triggerauthentication.created.v1"

	// TriggerAuthenticationUpdatedType is for event when a TriggerAuthentication is updated
	TriggerAuthenticationUpdatedType CloudEventType = "keda.authentication.triggerauthentication.updated.v1"

	// TriggerAuthenticationRemovedType is for event when a TriggerAuthentication is deleted
	TriggerAuthenticationRemovedType CloudEventType = "keda.authentication.triggerauthentication.removed.v1"

	// ClusterTriggerAuthenticationCreatedType is for event when a new ClusterTriggerAuthentication is created
	ClusterTriggerAuthenticationCreatedType CloudEventType = "keda.authentication.clustertriggerauthentication.created.v1"

	// ClusterTriggerAuthenticationCreatedType is for event when a ClusterTriggerAuthentication is updated
	ClusterTriggerAuthenticationUpdatedType CloudEventType = "keda.authentication.clustertriggerauthentication.updated.v1"

	// ClusterTriggerAuthenticationRemovedType is for event when a ClusterTriggerAuthentication is deleted
	ClusterTriggerAuthenticationRemovedType CloudEventType = "keda.authentication.clustertriggerauthentication.removed.v1"
)

var AllEventTypes = []CloudEventType{
	ScaledObjectFailedType, ScaledObjectReadyType, ScaledObjectRemovedType, ScaledObjectPausedType, ScaledObjectUnpausedType,
	ScaledJobFailedType, ScaledJobReadyType, ScaledJobRemovedType, ScaledJobPausedType, ScaledJobUnpausedType, ScaledJobRolloutCleanupStartedType, ScaledJobRolloutCleanupCompletedType, ScaledJobRolloutCleanupFailedType,
}
