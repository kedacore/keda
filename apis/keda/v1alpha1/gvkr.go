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
		}

		return "", err
	}
}
