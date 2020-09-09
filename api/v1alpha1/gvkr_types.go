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
