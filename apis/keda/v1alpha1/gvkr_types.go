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
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GroupVersionKindResource provides unified structure for schema.GroupVersionKind and Resource
type GroupVersionKindResource struct {
	Group    string `json:"group"`
	Version  string `json:"version"`
	Kind     string `json:"kind"`
	Resource string `json:"resource"`
}

// GroupVersionKind returns the group, version and kind of GroupVersionKindResource
func (gvkr GroupVersionKindResource) GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{Group: gvkr.Group, Version: gvkr.Version, Kind: gvkr.Kind}
}

// GroupVersion returns the group and version of GroupVersionKindResource
func (gvkr GroupVersionKindResource) GroupVersion() schema.GroupVersion {
	return schema.GroupVersion{Group: gvkr.Group, Version: gvkr.Version}
}

// GroupResource returns the group and resource of GroupVersionKindResource
func (gvkr GroupVersionKindResource) GroupResource() schema.GroupResource {
	return schema.GroupResource{Group: gvkr.Group, Resource: gvkr.Resource}
}

// GVKString returns the group, version and kind in string format
func (gvkr GroupVersionKindResource) GVKString() string {
	return gvkr.Group + "/" + gvkr.Version + "." + gvkr.Kind
}
