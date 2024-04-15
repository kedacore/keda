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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CloudEventSource defines how a KEDA event will be sent to event sink
// +kubebuilder:resource:path=cloudeventsources,scope=Namespaced
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Active",type="string",JSONPath=".status.conditions[?(@.type==\"Active\")].status"
type CloudEventSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CloudEventSourceSpec   `json:"spec"`
	Status CloudEventSourceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CloudEventSourceList is a list of CloudEventSource resources
type CloudEventSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []CloudEventSource `json:"items"`
}

// CloudEventSourceSpec defines the spec of CloudEventSource
type CloudEventSourceSpec struct {
	// +optional
	ClusterName string `json:"clusterName,omitempty"`

	Destination Destination `json:"destination"`

	// +optional
	AuthenticationRef *v1alpha1.AuthenticationRef `json:"authenticationRef,omitempty"`

	// +optional
	EventSubscription EventSubscription `json:"eventSubscription,omitempty"`
}

// CloudEventSourceStatus defines the observed state of CloudEventSource
// +optional
type CloudEventSourceStatus struct {
	// +optional
	Conditions v1alpha1.Conditions `json:"conditions,omitempty"`
}

// Destination defines the various ways to emit events
type Destination struct {
	// +optional
	HTTP *CloudEventHTTP `json:"http"`

	// +optional
	AzureEventGridTopic *AzureEventGridTopicSpec `json:"azureEventGridTopic"`
}

type CloudEventHTTP struct {
	URI string `json:"uri"`
}

type AzureEventGridTopicSpec struct {
	Endpoint string `json:"endpoint"`
}

// EventSubscription defines filters for events
type EventSubscription struct {
	// +optional
	IncludedEventTypes []CloudEventType `json:"includedEventTypes,omitempty"`

	// +optional
	ExcludedEventTypes []CloudEventType `json:"excludedEventTypes,omitempty"`
}

func init() {
	SchemeBuilder.Register(&CloudEventSource{}, &CloudEventSourceList{})
}

// GenerateIdentifier returns identifier for the object in for "kind.namespace.name"
func (ces *CloudEventSource) GenerateIdentifier() string {
	return v1alpha1.GenerateIdentifier("CloudEventSource", ces.Namespace, ces.Name)
}

// GetCloudEventSourceInitializedConditions returns CloudEventSource Conditions initialized to the default -> Status: Unknown
func GetCloudEventSourceInitializedConditions() *v1alpha1.Conditions {
	return &v1alpha1.Conditions{{Type: v1alpha1.ConditionActive, Status: metav1.ConditionUnknown}}
}
