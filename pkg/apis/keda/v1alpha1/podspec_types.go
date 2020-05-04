package v1alpha1

import (
	"github.com/kedacore/keda/pkg/apis/duck"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// PodSpecable is implemented by types containing a PodTemplateSpec
// in the manner of ReplicaSet, Deployment, DaemonSet, StatefulSet.
type PodSpecable corev1.PodTemplateSpec

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

// Assert that we implement the interfaces necessary to
// use duck.VerifyType.
var (
	_ duck.Populatable   = (*WithPod)(nil)
	_ duck.Implementable = (*PodSpecable)(nil)
	_ duck.Listable      = (*WithPod)(nil)
)

// GetFullType implements duck.Implementable
func (*PodSpecable) GetFullType() duck.Populatable {
	return &WithPod{}
}

// Populate implements duck.Populatable
func (t *WithPod) Populate() {
	t.Spec.Template = PodSpecable{
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
func (*WithPod) GetListType() runtime.Object {
	return &WithPodList{}
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WithPodList is a list of WithPod resources
type WithPodList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []WithPod `json:"items"`
}
