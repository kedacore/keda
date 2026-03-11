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

package message

const (
	ScalerIsBuiltMsg = "Scaler %s is built"

	ScalerStartMsg = "Started scalers watch"

	ScalerReadyMsg = "ScaledObject is ready for scaling"

	ScaleTargetErrMsg = "ScaledObject doesn't have correct scaleTargetRef specification"

	ScaleTargetNotFoundMsg = "Target resource doesn't exist"

	ScaleTargetNoSubresourceMsg = "Target resource doesn't expose /scale subresource"

	ScaledObjectRemoved = "ScaledObject was deleted"

	ScaledObjectFallbackActivatedMsg = "ScaledObject fallback is active"

	ScaledObjectFallbackDeactivatedMsg = "ScaledObject fallback is no longer active"

	ScaledJobReadyMsg = "ScaledJob is ready for scaling"

	ScaledJobRemoved = "ScaledJob was deleted"

	TriggerAuthenticationCreatedMsg = "New TriggerAuthentication configured"

	TriggerAuthenticationUpdatedMsg = "ClusterTriggerAuthentication %s is updated"

	ClusterTriggerAuthenticationCreatedMsg = "New ClusterTriggerAuthentication configured"

	ClusterTriggerAuthenticationUpdatedMsg = "ClusterTriggerAuthentication %s is updated"
)
