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
	v1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	scheme "github.com/knative/build/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// ClusterBuildTemplatesGetter has a method to return a ClusterBuildTemplateInterface.
// A group's client should implement this interface.
type ClusterBuildTemplatesGetter interface {
	ClusterBuildTemplates() ClusterBuildTemplateInterface
}

// ClusterBuildTemplateInterface has methods to work with ClusterBuildTemplate resources.
type ClusterBuildTemplateInterface interface {
	Create(*v1alpha1.ClusterBuildTemplate) (*v1alpha1.ClusterBuildTemplate, error)
	Update(*v1alpha1.ClusterBuildTemplate) (*v1alpha1.ClusterBuildTemplate, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.ClusterBuildTemplate, error)
	List(opts v1.ListOptions) (*v1alpha1.ClusterBuildTemplateList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.ClusterBuildTemplate, err error)
	ClusterBuildTemplateExpansion
}

// clusterBuildTemplates implements ClusterBuildTemplateInterface
type clusterBuildTemplates struct {
	client rest.Interface
}

// newClusterBuildTemplates returns a ClusterBuildTemplates
func newClusterBuildTemplates(c *BuildV1alpha1Client) *clusterBuildTemplates {
	return &clusterBuildTemplates{
		client: c.RESTClient(),
	}
}

// Get takes name of the clusterBuildTemplate, and returns the corresponding clusterBuildTemplate object, and an error if there is any.
func (c *clusterBuildTemplates) Get(name string, options v1.GetOptions) (result *v1alpha1.ClusterBuildTemplate, err error) {
	result = &v1alpha1.ClusterBuildTemplate{}
	err = c.client.Get().
		Resource("clusterbuildtemplates").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ClusterBuildTemplates that match those selectors.
func (c *clusterBuildTemplates) List(opts v1.ListOptions) (result *v1alpha1.ClusterBuildTemplateList, err error) {
	result = &v1alpha1.ClusterBuildTemplateList{}
	err = c.client.Get().
		Resource("clusterbuildtemplates").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested clusterBuildTemplates.
func (c *clusterBuildTemplates) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("clusterbuildtemplates").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a clusterBuildTemplate and creates it.  Returns the server's representation of the clusterBuildTemplate, and an error, if there is any.
func (c *clusterBuildTemplates) Create(clusterBuildTemplate *v1alpha1.ClusterBuildTemplate) (result *v1alpha1.ClusterBuildTemplate, err error) {
	result = &v1alpha1.ClusterBuildTemplate{}
	err = c.client.Post().
		Resource("clusterbuildtemplates").
		Body(clusterBuildTemplate).
		Do().
		Into(result)
	return
}

// Update takes the representation of a clusterBuildTemplate and updates it. Returns the server's representation of the clusterBuildTemplate, and an error, if there is any.
func (c *clusterBuildTemplates) Update(clusterBuildTemplate *v1alpha1.ClusterBuildTemplate) (result *v1alpha1.ClusterBuildTemplate, err error) {
	result = &v1alpha1.ClusterBuildTemplate{}
	err = c.client.Put().
		Resource("clusterbuildtemplates").
		Name(clusterBuildTemplate.Name).
		Body(clusterBuildTemplate).
		Do().
		Into(result)
	return
}

// Delete takes name of the clusterBuildTemplate and deletes it. Returns an error if one occurs.
func (c *clusterBuildTemplates) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("clusterbuildtemplates").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *clusterBuildTemplates) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Resource("clusterbuildtemplates").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched clusterBuildTemplate.
func (c *clusterBuildTemplates) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.ClusterBuildTemplate, err error) {
	result = &v1alpha1.ClusterBuildTemplate{}
	err = c.client.Patch(pt).
		Resource("clusterbuildtemplates").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
