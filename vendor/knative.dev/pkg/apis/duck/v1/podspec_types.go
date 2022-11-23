/*
Copyright 2018 The Knative Authors

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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"knative.dev/pkg/apis"
	"knative.dev/pkg/apis/duck/ducktypes"
)

// +genduck

// PodSpecable is implemented by types containing a PodTemplateSpec
// in the manner of ReplicaSet, Deployment, DaemonSet, StatefulSet.
type PodSpecable corev1.PodTemplateSpec

// PodSpecable is an Implementable duck type.
var _ ducktypes.Implementable = (*PodSpecable)(nil)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WithPod is the shell that demonstrates how PodSpecable types wrap
// a PodSpec.
type WithPod struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec WithPodSpec `json:"spec,omitempty"`
}

// WithPodSpec is the shell around the PodSpecable within WithPod.
type WithPodSpec struct {
	Template PodSpecable `json:"template,omitempty"`
}

// Verify WithPod resources meet duck contracts.
var (
	_ apis.Validatable      = (*WithPod)(nil)
	_ apis.Defaultable      = (*WithPod)(nil)
	_ apis.Listable         = (*WithPod)(nil)
	_ ducktypes.Populatable = (*WithPod)(nil)
)

// GetFullType implements duck.Implementable
func (wp *PodSpecable) GetFullType() ducktypes.Populatable {
	return &WithPod{}
}

// Populate implements duck.Populatable
func (wp *WithPod) Populate() {
	wp.Spec.Template = PodSpecable{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"foo": "bar",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:  "container-name",
				Image: "container-image:latest",
			}},
		},
	}
}

// GetListType implements apis.Listable
func (wp *WithPod) GetListType() runtime.Object {
	return &WithPodList{}
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WithPodList is a list of WithPod resources
type WithPodList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []WithPod `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Pod is a wrapper around Pod-like resource, which supports our interfaces
// for webhooks
type Pod struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec corev1.PodSpec `json:"spec,omitempty"`
}

// Verify Pod resources meet duck contracts.
var (
	_ apis.Validatable      = (*Pod)(nil)
	_ apis.Defaultable      = (*Pod)(nil)
	_ apis.Listable         = (*Pod)(nil)
	_ ducktypes.Populatable = (*Pod)(nil)
)

// GetFullType implements duck.Implementable
func (p *Pod) GetFullType() ducktypes.Populatable {
	return &Pod{}
}

// Populate implements duck.Populatable
func (p *Pod) Populate() {
	p.Spec = corev1.PodSpec{
		Containers: []corev1.Container{{
			Name:  "container-name",
			Image: "container-image:latest",
		}},
	}
}

// GetListType implements apis.Listable
func (p *Pod) GetListType() runtime.Object {
	return &PodList{}
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PodList is a list of WithPod resources
type PodList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []WithPod `json:"items"`
}
