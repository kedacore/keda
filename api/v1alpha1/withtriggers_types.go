package v1alpha1

import (
	"knative.dev/pkg/apis"
	"knative.dev/pkg/apis/duck"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// +kubebuilder:object:root=true

// WithTriggers is a specification for a resource with triggers
type WithTriggers struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec WithTriggersSpec `json:"spec"`
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
