/*
Copyright 2021 The KEDA Authors

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
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/apis/duck"
)

const (
	// Default polling interval for a ScaledObject triggers if no pollingInterval is defined.
	defaultPollingInterval = 30
)

// +kubebuilder:object:root=true

// WithTriggers is a specification for a resource with triggers
type WithTriggers struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	InternalKind string           `json:"internalKind"`
	Spec         WithTriggersSpec `json:"spec"`
}

// WithTriggersSpec is the spec for a an object with triggers resource
type WithTriggersSpec struct {
	PollingInterval *int32          `json:"pollingInterval,omitempty"`
	Triggers        []ScaleTriggers `json:"triggers"`
}

// Assert that we implement the interfaces necessary to
// use duck.VerifyType.
var (
	_ duck.Populatable   = (*WithTriggers)(nil)
	_ duck.Implementable = (*ScaleTriggers)(nil)
	_ apis.Listable      = (*WithTriggers)(nil)
)

// GetFullType implements duck.Implementable
func (*ScaleTriggers) GetFullType() duck.Populatable {
	return &WithTriggers{}
}

// Populate implements duck.Populatable
func (t *WithTriggers) Populate() {
	t.Spec.Triggers = []ScaleTriggers{{}}
}

// GetListType implements apis.Listable
func (*WithTriggers) GetListType() runtime.Object {
	return &WithTriggersList{}
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WithTriggersList is a list of ScaledObject resources
type WithTriggersList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []WithTriggers `json:"items"`
}

// GetPollingInterval returns defined polling interval, if not set default is being returned
func (t *WithTriggers) GetPollingInterval() time.Duration {
	if t.Spec.PollingInterval != nil {
		return time.Second * time.Duration(*t.Spec.PollingInterval)
	}

	return time.Second * time.Duration(defaultPollingInterval)
}

// GenerateIdentifier returns identifier for the object in for "kind.namespace.name"
func (t *WithTriggers) GenerateIdentifier() string {
	return GenerateIdentifier(t.InternalKind, t.Namespace, t.Name)
}

// AsDuckWithTriggers tries to generate WithTriggers object for input object
// returns error if input object is unknown
func AsDuckWithTriggers(scalableObject interface{}) (*WithTriggers, error) {
	switch obj := scalableObject.(type) {
	case *ScaledObject:
		return &WithTriggers{
			TypeMeta:     obj.TypeMeta,
			ObjectMeta:   obj.ObjectMeta,
			InternalKind: "ScaledObject",
			Spec: WithTriggersSpec{
				PollingInterval: obj.Spec.PollingInterval,
				Triggers:        obj.Spec.Triggers,
			},
		}, nil
	case *ScaledJob:
		return &WithTriggers{
			TypeMeta:     obj.TypeMeta,
			ObjectMeta:   obj.ObjectMeta,
			InternalKind: "ScaledJob",
			Spec: WithTriggersSpec{
				PollingInterval: obj.Spec.PollingInterval,
				Triggers:        obj.Spec.Triggers,
			},
		}, nil
	default:
		// here could be the conversion from unknown Duck type potentially in the future
		return nil, fmt.Errorf("unknown scalable object type %v", scalableObject)
	}
}
