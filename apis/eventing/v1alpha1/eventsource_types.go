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

// EventSource defines how a keda event will be sent to event sink
// +kubebuilder:resource:path=eventsources,scope=Namespaced
// +kubebuilder:subresource:status
type EventSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec EventSourceSpec `json:"spec"`
}

// +kubebuilder:object:root=true

// EventSourceList is a list of EventSource resources
type EventSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []EventSource `json:"items"`
}

// EventSource
type EventSourceSpec struct {
	// +optional
	ClusterName string `json:"clusterName,omitempty"`

	// +optional
	Destination Destination `json:"destination"`
}

type Destination struct {
	HTTP *CloudEventHTTP `json:"http"`
}

type CloudEventHTTP struct {
	URI string `json:"uri"`
}

func init() {
	SchemeBuilder.Register(&EventSource{}, &EventSourceList{})
}

// GenerateIdentifier returns identifier for the object in for "kind.namespace.name"
func (t *EventSource) GenerateIdentifier() string {
	return v1alpha1.GenerateIdentifier("EventSource", t.Namespace, t.Name)
}
