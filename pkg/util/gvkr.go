package util

import (
	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	defaultVersion  = "v1"
	defaultGroup    = "apps"
	defaultKind     = "Deployment"
	defaultResource = "deployments"
)

// ParseGVKR returns GroupVersionKindResource for specified apiVersion (groupVersion) and Kind
func ParseGVKR(restMapper meta.RESTMapper, apiVersion string, kind string) (kedav1alpha1.GroupVersionKindResource, error) {
	var group, version, resource string

	// if apiVersion is not specified, we suppose the default one should be used
	if apiVersion == "" {
		group = defaultGroup
		version = defaultVersion
	} else {
		groupVersion, err := schema.ParseGroupVersion(apiVersion)
		if err != nil {
			return kedav1alpha1.GroupVersionKindResource{}, err
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
		return kedav1alpha1.GroupVersionKindResource{}, err
	}

	return kedav1alpha1.GroupVersionKindResource{
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
		}

		return "", err
	}
}
