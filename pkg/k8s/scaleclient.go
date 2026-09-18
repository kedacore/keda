/*
Copyright 2022 The KEDA Authors

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

package k8s

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/scale"
	ctrl "sigs.k8s.io/controller-runtime"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

var log = ctrl.Log.WithName("scaleclient")

// InitScaleClient initializes scale client and returns k8s version
func InitScaleClient(mgr ctrl.Manager) (scale.ScalesGetter, kedautil.K8sVersion, error) {
	kubeVersion := kedautil.K8sVersion{}

	// create Discovery clientset
	// TODO If we need to increase the QPS of scaling API calls, copy and tweak this RESTConfig.
	clientset, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		log.Error(err, "not able to create Discovery clientset")
		return nil, kubeVersion, err
	}

	// Find out Kubernetes version
	version, err := clientset.ServerVersion()
	if err == nil {
		kubeVersion = kedautil.NewK8sVersion(version)
	} else {
		log.Error(err, "not able to get Kubernetes version")
		return nil, kubeVersion, err
	}

	return scale.New(
		clientset.RESTClient(), exactGroupMapper{RESTMapper: mgr.GetRESTMapper(), groupVersions: groupVersionsFunc(clientset)},
		dynamic.LegacyAPIPathResolverFunc,
		scale.NewDiscoveryScaleKindResolver(clientset),
	), kubeVersion, nil
}

// groupVersionsFunc returns a lookup of the versions an API group serves, preferred version first.
// It is only consulted when the RESTMapper has no candidate in the requested group (see exactGroupMapper).
func groupVersionsFunc(client discovery.ServerGroupsInterface) func(group string) ([]string, error) {
	return func(group string) ([]string, error) {
		groups, err := client.ServerGroups()
		if err != nil {
			return nil, err
		}
		for _, g := range groups.Groups {
			if g.Name != group {
				continue
			}
			versions := []string{g.PreferredVersion.Version}
			for _, v := range g.Versions {
				if v.Version != g.PreferredVersion.Version {
					versions = append(versions, v.Version)
				}
			}
			return versions, nil
		}
		return nil, nil
	}
}

// exactGroupMapper makes version-less resource lookups stay in the requested API group.
//
// The scale client resolves the GroupResource it is given with RESTMapper.ResourceFor(gr.WithVersion("")).
// For that shape apimachinery's DefaultRESTMapper also accepts group-prefix matches (so that
// "storageclass.storage" can find storage.k8s.io), and the discovery-backed mapper used by controller-runtime
// is one DefaultRESTMapper per group/version behind a PriorityRESTMapper. When another group starts with the
// requested group name and serves a resource of the same name (OpenKruise's apps.kruise.io serves
// "statefulsets"), the prefix match is a candidate too, and which candidate wins follows the mapper's group
// order, which is map-iteration order inside controller-runtime's lazy mapper. The result is that
// apps/statefulsets may resolve to apps.kruise.io/v1beta1 statefulsets and every /scale call for a plain
// StatefulSet fails with "not found" (https://github.com/kedacore/keda/issues/8182).
//
// KEDA always knows the exact group of the scale target, so version-less lookups only accept candidates from
// the requested group (ResourcesFor keeps the preferred version first). controller-runtime's lazy mapper only
// discovers a group when a lookup fails, and a prefix match does not fail, so the requested group may simply
// not be loaded yet: in that case the group's versions are discovered and the mapper is asked with each of
// them, which does make it load the group. A group that has no such resource at all yields NoMatch instead
// of another group's resource. Fully qualified lookups are untouched.
type exactGroupMapper struct {
	meta.RESTMapper
	groupVersions func(group string) ([]string, error)
}

func (m exactGroupMapper) ResourceFor(input schema.GroupVersionResource) (schema.GroupVersionResource, error) {
	if input.Version != "" || input.Group == "" {
		return m.RESTMapper.ResourceFor(input)
	}
	candidates, err := m.ResourcesFor(input)
	if err != nil && !meta.IsNoMatchError(err) {
		return schema.GroupVersionResource{}, err
	}
	for _, gvr := range candidates {
		if gvr.Group == input.Group {
			return gvr, nil
		}
	}
	versions, err := m.groupVersions(input.Group)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}
	for _, version := range versions {
		if gvr, err := m.RESTMapper.ResourceFor(input.GroupResource().WithVersion(version)); err == nil {
			return gvr, nil
		}
	}
	return schema.GroupVersionResource{}, &meta.NoResourceMatchError{PartialResource: input}
}
