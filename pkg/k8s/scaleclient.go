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
		clientset.RESTClient(), exactGroupMapper{mgr.GetRESTMapper()},
		dynamic.LegacyAPIPathResolverFunc,
		scale.NewDiscoveryScaleKindResolver(clientset),
	), kubeVersion, nil
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
// KEDA always knows the exact group of the scale target, so for version-less lookups we take the first
// candidate in the requested group (ResourcesFor keeps the preferred version first) and only fall back to
// the mapper's own answer when the group has no such resource at all. Fully qualified lookups are untouched.
type exactGroupMapper struct {
	meta.RESTMapper
}

func (m exactGroupMapper) ResourceFor(input schema.GroupVersionResource) (schema.GroupVersionResource, error) {
	if input.Version != "" || input.Group == "" {
		return m.RESTMapper.ResourceFor(input)
	}
	candidates, err := m.ResourcesFor(input)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}
	for _, gvr := range candidates {
		if gvr.Group == input.Group {
			return gvr, nil
		}
	}
	return m.RESTMapper.ResourceFor(input)
}
