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

package v1alpha1

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestGroupVersionKindResource_GroupVersionKind(t *testing.T) {
	gvkr := GroupVersionKindResource{
		Group:    "apps",
		Version:  "v1",
		Kind:     "Deployment",
		Resource: "deployments",
	}

	got := gvkr.GroupVersionKind()
	want := schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}

	if got != want {
		t.Errorf("GroupVersionKind() = %v, want %v", got, want)
	}
}

func TestGroupVersionKindResource_GroupVersion(t *testing.T) {
	gvkr := GroupVersionKindResource{
		Group:    "apps",
		Version:  "v1",
		Kind:     "Deployment",
		Resource: "deployments",
	}

	got := gvkr.GroupVersion()
	want := schema.GroupVersion{Group: "apps", Version: "v1"}

	if got != want {
		t.Errorf("GroupVersion() = %v, want %v", got, want)
	}
}

func TestGroupVersionKindResource_GroupResource(t *testing.T) {
	gvkr := GroupVersionKindResource{
		Group:    "apps",
		Version:  "v1",
		Kind:     "Deployment",
		Resource: "deployments",
	}

	got := gvkr.GroupResource()
	want := schema.GroupResource{Group: "apps", Resource: "deployments"}

	if got != want {
		t.Errorf("GroupResource() = %v, want %v", got, want)
	}
}

func TestGroupVersionKindResource_GVKString(t *testing.T) {
	tests := []struct {
		name string
		gvkr GroupVersionKindResource
		want string
	}{
		{
			name: "standard resource",
			gvkr: GroupVersionKindResource{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			},
			want: "apps/v1.Deployment",
		},
		{
			name: "core group resource",
			gvkr: GroupVersionKindResource{
				Group:   "",
				Version: "v1",
				Kind:    "Pod",
			},
			want: "/v1.Pod",
		},
		{
			name: "custom resource",
			gvkr: GroupVersionKindResource{
				Group:   "keda.sh",
				Version: "v1alpha1",
				Kind:    "ScaledObject",
			},
			want: "keda.sh/v1alpha1.ScaledObject",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.gvkr.GVKString()
			if got != tt.want {
				t.Errorf("GVKString() = %q, want %q", got, tt.want)
			}
		})
	}
}
