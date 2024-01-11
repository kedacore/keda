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

const (
	// CloudEventSourceConditionActiveReason defines the active condition reason for CloudEventSource
	CloudEventSourceConditionActiveReason = "CloudEventSourceActive"
	// CloudEventSourceConditionFailedReason defines the failed condition reason for CloudEventSource
	CloudEventSourceConditionFailedReason = "CloudEventSourceFailed"
	// CloudEventSourceConditionActiveMessage defines the active condition message for CloudEventSource
	CloudEventSourceConditionActiveMessage = "Is configured to send events to the configured destination"
	// CloudEventSourceConditionFailedMessage defines the failed condition message for CloudEventSource
	CloudEventSourceConditionFailedMessage = "Failed to send events to the configured destination"
)
