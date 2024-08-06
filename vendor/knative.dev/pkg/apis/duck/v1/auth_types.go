/*
Copyright 2022 The Knative Authors

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
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/apis/duck/ducktypes"
	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/ptr"
)

// +genduck

// AuthStatus is meant to provide the generated service account name
// in the resource status.
type AuthStatus struct {
	// ServiceAccountName is the name of the generated service account
	// used for this components OIDC authentication.
	ServiceAccountName *string `json:"serviceAccountName,omitempty"`

	// ServiceAccountNames is the list of names of the generated service accounts
	// used for this components OIDC authentication. This list can have len() > 1,
	// when the component uses multiple identities (e.g. in case of a Parallel).
	ServiceAccountNames []string `json:"serviceAccountNames,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AuthenticatableType is a skeleton type wrapping AuthStatus in the manner we expect
// resource writers defining compatible resources to embed it.  We will
// typically use this type to deserialize AuthenticatableType ObjectReferences and
// access the AuthenticatableType data.  This is not a real resource.
type AuthenticatableType struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status AuthenticatableStatus `json:"status"`
}

type AuthenticatableStatus struct {
	// Auth contains the service account name for the subscription
	// +optional
	Auth *AuthStatus `json:"auth,omitempty"`
}

var (
	// AuthStatus is a Convertible type.
	_ apis.Convertible = (*AuthStatus)(nil)

	// Verify AuthenticatableType resources meet duck contracts.
	_ apis.Listable         = (*AuthenticatableType)(nil)
	_ ducktypes.Populatable = (*AuthenticatableType)(nil)
	_ kmeta.OwnerRefable    = (*AuthenticatableType)(nil)
)

// GetFullType implements duck.Implementable
func (*AuthStatus) GetFullType() ducktypes.Populatable {
	return &AuthenticatableType{}
}

// ConvertTo implements apis.Convertible
func (a *AuthStatus) ConvertTo(_ context.Context, to apis.Convertible) error {
	return fmt.Errorf("v1 is the highest known version, got: %T", to)
}

// ConvertFrom implements apis.Convertible
func (a *AuthStatus) ConvertFrom(_ context.Context, from apis.Convertible) error {
	return fmt.Errorf("v1 is the highest known version, got: %T", from)
}

// Populate implements duck.Populatable
func (t *AuthenticatableType) Populate() {
	t.Status = AuthenticatableStatus{
		Auth: &AuthStatus{
			// Populate ALL fields
			ServiceAccountName: ptr.String("foo"),
			ServiceAccountNames: []string{
				"bar",
				"baz",
			},
		},
	}
}

// GetGroupVersionKind implements kmeta.OwnerRefable
func (t *AuthenticatableType) GetGroupVersionKind() schema.GroupVersionKind {
	return t.GroupVersionKind()
}

// GetListType implements apis.Listable
func (*AuthenticatableType) GetListType() runtime.Object {
	return &AuthenticatableTypeList{}
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AuthenticatableTypeList is a list of AuthenticatableType resources
type AuthenticatableTypeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AuthenticatableType `json:"items"`
}
