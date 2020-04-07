package util

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	defaultVersion  = "v1"
	defaultGroup    = "apps"
	defaultKind     = "Deployment"
	defaultResource = "deployments"
)

// GroupVersionKindResource provides unified structure for schema.GroupVersionKind and Resource
type GroupVersionKindResource struct {
	Group    string `json:"group"`
	Version  string `json:"version"`
	Kind     string `json:"kind"`
	Resource string `json:"resource"`
}

func (gvkr GroupVersionKindResource) GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{Group: gvkr.Group, Version: gvkr.Version, Kind: gvkr.Kind}
}

func (gvkr GroupVersionKindResource) GroupVersion() schema.GroupVersion {
	return schema.GroupVersion{Group: gvkr.Group, Version: gvkr.Version}
}

func (gvkr GroupVersionKindResource) GroupResource() schema.GroupResource {
	return schema.GroupResource{Group: gvkr.Group, Resource: gvkr.Resource}
}

func (gvkr GroupVersionKindResource) GVKString() string {
	return gvkr.Group + "/" + gvkr.Version + "." + gvkr.Kind
}

// ParseGVKR returns GroupVersionKindResource for specified apiVersion (groupVersion) and Kind
func ParseGVKR(restMapper meta.RESTMapper, apiVersion string, kind string) (GroupVersionKindResource, error) {
	var group, version, resource string

	// if apiVersion is not specified, we suppose the default one should be used
	if apiVersion == "" {
		group = defaultGroup
		version = defaultVersion
	} else {
		groupVersion, err := schema.ParseGroupVersion(apiVersion)
		if err != nil {
			return GroupVersionKindResource{}, err
		}

		group = groupVersion.Group
		version = groupVersion.Version
	}

	// if kind is not specified, we suppose that default one should be used
	if kind == "" {
		kind = defaultKind
	}

	// get resource
	resource, err := getResource(restMapper, group, version, kind)
	if err != nil {
		return GroupVersionKindResource{}, err
	}

	return GroupVersionKindResource{
		Group:    group,
		Version:  version,
		Kind:     kind,
		Resource: resource,
	}, nil
}

func getResource(restMapper meta.RESTMapper, group string, version string, kind string) (string, error) {
	switch kind {
	case defaultKind:
		return defaultResource, nil
	case "StatefulSet":
		return "statefulsets", nil
	default:
		restmapping, err := restMapper.RESTMapping(schema.GroupKind{Group: group, Kind: kind}, version)
		if err == nil {
			return restmapping.Resource.GroupResource().Resource, nil
		} else {
			return "", err
		}
	}
}
