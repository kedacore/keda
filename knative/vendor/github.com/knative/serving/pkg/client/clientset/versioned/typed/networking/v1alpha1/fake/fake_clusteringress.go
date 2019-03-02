/*
Copyright 2018 The Knative Authors

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
package fake

import (
	v1alpha1 "github.com/knative/serving/pkg/apis/networking/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeClusterIngresses implements ClusterIngressInterface
type FakeClusterIngresses struct {
	Fake *FakeNetworkingV1alpha1
}

var clusteringressesResource = schema.GroupVersionResource{Group: "networking.internal.knative.dev", Version: "v1alpha1", Resource: "clusteringresses"}

var clusteringressesKind = schema.GroupVersionKind{Group: "networking.internal.knative.dev", Version: "v1alpha1", Kind: "ClusterIngress"}

// Get takes name of the clusterIngress, and returns the corresponding clusterIngress object, and an error if there is any.
func (c *FakeClusterIngresses) Get(name string, options v1.GetOptions) (result *v1alpha1.ClusterIngress, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(clusteringressesResource, name), &v1alpha1.ClusterIngress{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ClusterIngress), err
}

// List takes label and field selectors, and returns the list of ClusterIngresses that match those selectors.
func (c *FakeClusterIngresses) List(opts v1.ListOptions) (result *v1alpha1.ClusterIngressList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(clusteringressesResource, clusteringressesKind, opts), &v1alpha1.ClusterIngressList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.ClusterIngressList{ListMeta: obj.(*v1alpha1.ClusterIngressList).ListMeta}
	for _, item := range obj.(*v1alpha1.ClusterIngressList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested clusterIngresses.
func (c *FakeClusterIngresses) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(clusteringressesResource, opts))
}

// Create takes the representation of a clusterIngress and creates it.  Returns the server's representation of the clusterIngress, and an error, if there is any.
func (c *FakeClusterIngresses) Create(clusterIngress *v1alpha1.ClusterIngress) (result *v1alpha1.ClusterIngress, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(clusteringressesResource, clusterIngress), &v1alpha1.ClusterIngress{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ClusterIngress), err
}

// Update takes the representation of a clusterIngress and updates it. Returns the server's representation of the clusterIngress, and an error, if there is any.
func (c *FakeClusterIngresses) Update(clusterIngress *v1alpha1.ClusterIngress) (result *v1alpha1.ClusterIngress, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(clusteringressesResource, clusterIngress), &v1alpha1.ClusterIngress{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ClusterIngress), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeClusterIngresses) UpdateStatus(clusterIngress *v1alpha1.ClusterIngress) (*v1alpha1.ClusterIngress, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(clusteringressesResource, "status", clusterIngress), &v1alpha1.ClusterIngress{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ClusterIngress), err
}

// Delete takes name of the clusterIngress and deletes it. Returns an error if one occurs.
func (c *FakeClusterIngresses) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(clusteringressesResource, name), &v1alpha1.ClusterIngress{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeClusterIngresses) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(clusteringressesResource, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.ClusterIngressList{})
	return err
}

// Patch applies the patch and returns the patched clusterIngress.
func (c *FakeClusterIngresses) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.ClusterIngress, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(clusteringressesResource, name, data, subresources...), &v1alpha1.ClusterIngress{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ClusterIngress), err
}
