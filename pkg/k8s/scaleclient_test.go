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

package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/restmapper"
)

// discoveryMapper builds the same kind of RESTMapper controller-runtime hands to the scale client:
// one mapper per group/version behind a PriorityRESTMapper whose priority follows the group order.
func discoveryMapper(groups ...*restmapper.APIGroupResources) meta.RESTMapper {
	return restmapper.NewDiscoveryRESTMapper(groups)
}

func appsGroup() *restmapper.APIGroupResources {
	return &restmapper.APIGroupResources{
		Group: metav1.APIGroup{
			Name:             "apps",
			Versions:         []metav1.GroupVersionForDiscovery{{Version: "v1"}},
			PreferredVersion: metav1.GroupVersionForDiscovery{Version: "v1"},
		},
		VersionedResources: map[string][]metav1.APIResource{
			"v1": {
				{Name: "deployments", SingularName: "deployment", Namespaced: true, Kind: "Deployment"},
				{Name: "statefulsets", SingularName: "statefulset", Namespaced: true, Kind: "StatefulSet"},
			},
		},
	}
}

// kruiseGroup mimics OpenKruise, whose group name starts with "apps" and also serves "statefulsets".
func kruiseGroup() *restmapper.APIGroupResources {
	return &restmapper.APIGroupResources{
		Group: metav1.APIGroup{
			Name:             "apps.kruise.io",
			Versions:         []metav1.GroupVersionForDiscovery{{Version: "v1beta1"}, {Version: "v1alpha1"}},
			PreferredVersion: metav1.GroupVersionForDiscovery{Version: "v1beta1"},
		},
		VersionedResources: map[string][]metav1.APIResource{
			"v1beta1":  {{Name: "statefulsets", SingularName: "statefulset", Namespaced: true, Kind: "StatefulSet"}},
			"v1alpha1": {{Name: "statefulsets", SingularName: "statefulset", Namespaced: true, Kind: "StatefulSet"}, {Name: "clonesets", SingularName: "cloneset", Namespaced: true, Kind: "CloneSet"}},
		},
	}
}

var (
	appsStatefulSets   = schema.GroupVersionResource{Group: "apps", Resource: "statefulsets"}
	appsV1StatefulSets = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}
	kruiseStatefulSets = schema.GroupVersionResource{Group: "apps.kruise.io", Version: "v1beta1", Resource: "statefulsets"}
)

// The scale client asks the mapper for a GroupResource with an empty version. apimachinery falls back to
// group-prefix matching for that shape, and which group wins then depends on the group order of the
// discovery mapper (map iteration order inside controller-runtime's lazy mapper), so the plain mapper
// can return apps.kruise.io for apps/statefulsets. This test pins that behaviour so the wrapper below
// has something real to guard against.
func TestPlainDiscoveryMapperPrefersPrefixGroupDependingOnOrder(t *testing.T) {
	gvr, err := discoveryMapper(kruiseGroup(), appsGroup()).ResourceFor(appsStatefulSets)
	require.NoError(t, err)
	assert.Equal(t, kruiseStatefulSets, gvr, "kruise first: prefix match wins")

	gvr, err = discoveryMapper(appsGroup(), kruiseGroup()).ResourceFor(appsStatefulSets)
	require.NoError(t, err)
	assert.Equal(t, appsV1StatefulSets, gvr, "apps first: exact match wins")
}

// versionsOf mimics discovery for the two test groups.
func versionsOf(group string) ([]string, error) {
	switch group {
	case "apps":
		return []string{"v1"}, nil
	case "apps.kruise.io":
		return []string{"v1beta1", "v1alpha1"}, nil
	}
	return nil, nil
}

func TestExactGroupMapperResourceFor(t *testing.T) {
	for name, groups := range map[string][]*restmapper.APIGroupResources{
		"kruise first": {kruiseGroup(), appsGroup()},
		"apps first":   {appsGroup(), kruiseGroup()},
	} {
		t.Run(name, func(t *testing.T) {
			m := exactGroupMapper{RESTMapper: discoveryMapper(groups...), groupVersions: versionsOf}

			gvr, err := m.ResourceFor(appsStatefulSets)
			require.NoError(t, err)
			assert.Equal(t, appsV1StatefulSets, gvr, "version-less lookup must stay in the requested group")

			gvr, err = m.ResourceFor(schema.GroupVersionResource{Group: "apps.kruise.io", Resource: "statefulsets"})
			require.NoError(t, err)
			assert.Equal(t, kruiseStatefulSets, gvr, "exact group keeps its preferred version")

			gvr, err = m.ResourceFor(schema.GroupVersionResource{Group: "apps.kruise.io", Version: "v1alpha1", Resource: "statefulsets"})
			require.NoError(t, err)
			assert.Equal(t, schema.GroupVersionResource{Group: "apps.kruise.io", Version: "v1alpha1", Resource: "statefulsets"}, gvr, "fully qualified lookups are passed through")
		})
	}
}

// lazyMapper mimics controller-runtime's lazy mapper: it starts with only some groups loaded and loads the
// rest when a lookup fails, and a prefix match is not a failure.
type lazyMapper struct {
	meta.RESTMapper
	full meta.RESTMapper
}

func (l *lazyMapper) ResourceFor(input schema.GroupVersionResource) (schema.GroupVersionResource, error) {
	gvr, err := l.RESTMapper.ResourceFor(input)
	if meta.IsNoMatchError(err) {
		l.RESTMapper = l.full
		return l.full.ResourceFor(input)
	}
	return gvr, err
}

// Only apps.kruise.io is loaded when the scale client asks for apps/statefulsets: the prefix match is the only
// candidate, so the wrapper must make the mapper discover "apps" instead of settling for the prefix match.
func TestExactGroupMapperDiscoversRequestedGroupBeforeFallingBack(t *testing.T) {
	lazy := &lazyMapper{RESTMapper: discoveryMapper(kruiseGroup()), full: discoveryMapper(kruiseGroup(), appsGroup())}
	m := exactGroupMapper{RESTMapper: lazy, groupVersions: versionsOf}

	gvr, err := m.ResourceFor(appsStatefulSets)
	require.NoError(t, err)
	assert.Equal(t, appsV1StatefulSets, gvr)

	gvr, err = m.ResourceFor(appsStatefulSets)
	require.NoError(t, err)
	assert.Equal(t, appsV1StatefulSets, gvr, "second lookup is served from the now loaded group")
}

func TestExactGroupMapperNoMatchWhenGroupHasNoSuchResource(t *testing.T) {
	m := exactGroupMapper{RESTMapper: discoveryMapper(kruiseGroup()), groupVersions: versionsOf}

	// "apps" is not served by this cluster at all: never answer with another group's resource.
	_, err := m.ResourceFor(appsStatefulSets)
	require.Error(t, err)
	assert.True(t, meta.IsNoMatchError(err))

	// Nothing matches anywhere: still NoMatch.
	_, err = m.ResourceFor(schema.GroupVersionResource{Group: "batch", Resource: "jobs"})
	require.Error(t, err)
	assert.True(t, meta.IsNoMatchError(err))
}
