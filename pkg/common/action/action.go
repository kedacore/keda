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

package action

const (
	// Unknown is a placeholder action and should not be used
	Unknown = "Unknown"

	// Created indicates a resource was created
	Created = "Created"

	// Updated indicates a resource was updated
	Updated = "Updated"

	// Deleted indicates a resource was deleted
	Deleted = "Deleted"

	// Paused indicates a resource was paused
	Paused = "Paused"

	// Unpaused indicates a resource was unpaused
	Unpaused = "Unpaused"

	// Ready indicates a resource is ready
	Ready = "Ready"

	// Activated indicates a resource or feature was activated
	Activated = "Activated"

	// Deactivated indicates a resource or feature was deactivated
	Deactivated = "Deactivated"

	// Active indicates a resource is active
	Active = "Active"

	// Inactive indicates a resource is inactive
	Inactive = "Inactive"

	// Started indicates an operation started
	Started = "Started"

	// Stopped indicates an operation stopped
	Stopped = "Stopped"

	// Completed indicates an operation completed
	Completed = "Completed"

	// Failed indicates an operation failed
	Failed = "Failed"

	// Info indicates an informational message
	Info = "Info"
)
