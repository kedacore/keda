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
	// Ready is for event when ScaledObject or ScaledJob is ready
	Ready = "Ready"

	// CheckFailed is for event when ScaledObject or ScaledJob validation check failed
	CheckFailed = "CheckFailed"

	// Deleted is for event when ScaledObject or ScaledJob is deleted
	Deleted = "Deleted"

	// ScalersStarted is for event when scalers watch started for ScaledObject or ScaledJob
	ScalersStarted = "ScalersStarted"

	// ScalersRestarted is for event when scalers watch was restarted for ScaledObject or ScaledJob
	ScalersRestarted = "ScalersRestarted"

	// ScalersStopped is for event when scalers watch was stopped for ScaledObject or ScaledJob
	ScalersStopped = "ScalersStopped"

	// ScaleTargetActivated is for event when the scale target of ScaledObject was activated
	ScaleTargetActivated = "ScaleTargetActivated"

	// ScaleTargetDeactivated is for event when the scale target for ScaledObject was deactivated
	ScaleTargetDeactivated = "ScaleTargetDeactivated"

	// ScaleTargetActivationFailed is for event when the activation the scale target for ScaledObject fails
	ScaleTargetActivationFailed = "ScaleTargetActivationFailed"

	// ScaleTargetDeactivationFailed is for event when the deactivation of the scale target for ScaledObject fails
	ScaleTargetDeactivationFailed = "ScaleTargetDeactivationFailed"

	// JobsCreated is for event when jobs for ScaledJob are created
	JobsCreated = "JobsCreated"
)
