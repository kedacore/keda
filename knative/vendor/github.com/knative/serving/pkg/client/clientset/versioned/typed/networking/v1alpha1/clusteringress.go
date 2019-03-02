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
package v1alpha1

import (
	v1alpha1 "github.com/knative/serving/pkg/apis/networking/v1alpha1"
	scheme "github.com/knative/serving/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// ClusterIngressesGetter has a method to return a ClusterIngressInterface.
// A group's client should implement this interface.
type ClusterIngressesGetter interface {
	ClusterIngresses() ClusterIngressInterface
}

// ClusterIngressInterface has methods to work with ClusterIngress resources.
type ClusterIngressInterface interface {
	Create(*v1alpha1.ClusterIngress) (*v1alpha1.ClusterIngress, error)
	Update(*v1alpha1.ClusterIngress) (*v1alpha1.ClusterIngress, error)
	UpdateStatus(*v1alpha1.ClusterIngress) (*v1alpha1.ClusterIngress, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.ClusterIngress, error)
	List(opts v1.ListOptions) (*v1alpha1.ClusterIngressList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.ClusterIngress, err error)
	ClusterIngressExpansion
}

// clusterIngresses implements ClusterIngressInterface
type clusterIngresses struct {
	client rest.Interface
}

// newClusterIngresses returns a ClusterIngresses
func newClusterIngresses(c *NetworkingV1alpha1Client) *clusterIngresses {
	return &clusterIngresses{
		client: c.RESTClient(),
	}
}

// Get takes name of the clusterIngress, and returns the corresponding clusterIngress object, and an error if there is any.
func (c *clusterIngresses) Get(name string, options v1.GetOptions) (result *v1alpha1.ClusterIngress, err error) {
	result = &v1alpha1.ClusterIngress{}
	err = c.client.Get().
		Resource("clusteringresses").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ClusterIngresses that match those selectors.
func (c *clusterIngresses) List(opts v1.ListOptions) (result *v1alpha1.ClusterIngressList, err error) {
	result = &v1alpha1.ClusterIngressList{}
	err = c.client.Get().
		Resource("clusteringresses").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested clusterIngresses.
func (c *clusterIngresses) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("clusteringresses").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a clusterIngress and creates it.  Returns the server's representation of the clusterIngress, and an error, if there is any.
func (c *clusterIngresses) Create(clusterIngress *v1alpha1.ClusterIngress) (result *v1alpha1.ClusterIngress, err error) {
	result = &v1alpha1.ClusterIngress{}
	err = c.client.Post().
		Resource("clusteringresses").
		Body(clusterIngress).
		Do().
		Into(result)
	return
}

// Update takes the representation of a clusterIngress and updates it. Returns the server's representation of the clusterIngress, and an error, if there is any.
func (c *clusterIngresses) Update(clusterIngress *v1alpha1.ClusterIngress) (result *v1alpha1.ClusterIngress, err error) {
	result = &v1alpha1.ClusterIngress{}
	err = c.client.Put().
		Resource("clusteringresses").
		Name(clusterIngress.Name).
		Body(clusterIngress).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *clusterIngresses) UpdateStatus(clusterIngress *v1alpha1.ClusterIngress) (result *v1alpha1.ClusterIngress, err error) {
	result = &v1alpha1.ClusterIngress{}
	err = c.client.Put().
		Resource("clusteringresses").
		Name(clusterIngress.Name).
		SubResource("status").
		Body(clusterIngress).
		Do().
		Into(result)
	return
}

// Delete takes name of the clusterIngress and deletes it. Returns an error if one occurs.
func (c *clusterIngresses) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("clusteringresses").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *clusterIngresses) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Resource("clusteringresses").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched clusterIngress.
func (c *clusterIngresses) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.ClusterIngress, err error) {
	result = &v1alpha1.ClusterIngress{}
	err = c.client.Patch(pt).
		Resource("clusteringresses").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
