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
// +kubebuilder:validation:Enum=keda.scaledobject.ready.v1;keda.scaledobject.failed.v1;keda.scaledobject.removed.v1;keda.scaledjob.ready.v1;keda.scaledjob.failed.v1;keda.scaledjob.removed.v1
type CloudEventType string

const (
	// ScaledObjectReadyType is for event when a new ScaledObject is ready
	ScaledObjectReadyType CloudEventType = "keda.scaledobject.ready.v1"

	// ScaledObjectFailedType is for event when creating ScaledObject failed
	ScaledObjectFailedType CloudEventType = "keda.scaledobject.failed.v1"

	// ScaledObjectRemovedType is for event when removed ScaledObject
	ScaledObjectRemovedType CloudEventType = "keda.scaledobject.removed.v1"

	// ScaledJobReadyType is for event when a new ScaledJob is ready
	ScaledJobReadyType CloudEventType = "keda.scaledjob.ready.v1"

	// ScaledJobFailedType is for event when creating ScaledJob failed
	ScaledJobFailedType CloudEventType = "keda.scaledjob.failed.v1"

	// ScaledJobRemovedType is for event when removed ScaledJob
	ScaledJobRemovedType CloudEventType = "keda.scaledjob.removed.v1"
)

var AllEventTypes = []CloudEventType{
	ScaledObjectFailedType, ScaledObjectReadyType, ScaledObjectRemovedType,
	ScaledJobFailedType, ScaledJobReadyType, ScaledJobRemovedType,
}
