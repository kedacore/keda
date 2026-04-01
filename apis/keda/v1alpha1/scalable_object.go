/*
Copyright 2025 The KEDA Authors

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:object:generate=false

// ScalableObject is the interface implemented by ScaledObject and ScaledJob,
// providing common status accessors used by the scale handler and executor.
type ScalableObject interface {
	client.Object
	// GetStatusConditions returns a pointer to the status conditions slice,
	// allowing in-place modification.
	GetStatusConditions() *Conditions
	// SetStatusLastActiveTime sets the LastActiveTime in the status.
	SetStatusLastActiveTime(*metav1.Time)
	// SetStatusPausedReplicaCount sets the PausedReplicaCount in the status.
	// For ScaledJob this is a no-op.
	SetStatusPausedReplicaCount(*int32)
	// GetStatusTriggersActivity returns the triggers activity map from status,
	// initializing it if nil.
	GetStatusTriggersActivity() map[string]TriggerActivityStatus
	// SetStatusTriggersActivity sets the triggers activity map in the status.
	SetStatusTriggersActivity(map[string]TriggerActivityStatus)
	// GetStatusExternalMetricNames returns the external metric names from status.
	GetStatusExternalMetricNames() []string
}
