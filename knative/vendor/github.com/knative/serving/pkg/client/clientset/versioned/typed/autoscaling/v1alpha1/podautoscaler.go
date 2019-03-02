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
	v1alpha1 "github.com/knative/serving/pkg/apis/autoscaling/v1alpha1"
	scheme "github.com/knative/serving/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// PodAutoscalersGetter has a method to return a PodAutoscalerInterface.
// A group's client should implement this interface.
type PodAutoscalersGetter interface {
	PodAutoscalers(namespace string) PodAutoscalerInterface
}

// PodAutoscalerInterface has methods to work with PodAutoscaler resources.
type PodAutoscalerInterface interface {
	Create(*v1alpha1.PodAutoscaler) (*v1alpha1.PodAutoscaler, error)
	Update(*v1alpha1.PodAutoscaler) (*v1alpha1.PodAutoscaler, error)
	UpdateStatus(*v1alpha1.PodAutoscaler) (*v1alpha1.PodAutoscaler, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.PodAutoscaler, error)
	List(opts v1.ListOptions) (*v1alpha1.PodAutoscalerList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.PodAutoscaler, err error)
	PodAutoscalerExpansion
}

// podAutoscalers implements PodAutoscalerInterface
type podAutoscalers struct {
	client rest.Interface
	ns     string
}

// newPodAutoscalers returns a PodAutoscalers
func newPodAutoscalers(c *AutoscalingV1alpha1Client, namespace string) *podAutoscalers {
	return &podAutoscalers{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the podAutoscaler, and returns the corresponding podAutoscaler object, and an error if there is any.
func (c *podAutoscalers) Get(name string, options v1.GetOptions) (result *v1alpha1.PodAutoscaler, err error) {
	result = &v1alpha1.PodAutoscaler{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("podautoscalers").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of PodAutoscalers that match those selectors.
func (c *podAutoscalers) List(opts v1.ListOptions) (result *v1alpha1.PodAutoscalerList, err error) {
	result = &v1alpha1.PodAutoscalerList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("podautoscalers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested podAutoscalers.
func (c *podAutoscalers) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("podautoscalers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a podAutoscaler and creates it.  Returns the server's representation of the podAutoscaler, and an error, if there is any.
func (c *podAutoscalers) Create(podAutoscaler *v1alpha1.PodAutoscaler) (result *v1alpha1.PodAutoscaler, err error) {
	result = &v1alpha1.PodAutoscaler{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("podautoscalers").
		Body(podAutoscaler).
		Do().
		Into(result)
	return
}

// Update takes the representation of a podAutoscaler and updates it. Returns the server's representation of the podAutoscaler, and an error, if there is any.
func (c *podAutoscalers) Update(podAutoscaler *v1alpha1.PodAutoscaler) (result *v1alpha1.PodAutoscaler, err error) {
	result = &v1alpha1.PodAutoscaler{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("podautoscalers").
		Name(podAutoscaler.Name).
		Body(podAutoscaler).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *podAutoscalers) UpdateStatus(podAutoscaler *v1alpha1.PodAutoscaler) (result *v1alpha1.PodAutoscaler, err error) {
	result = &v1alpha1.PodAutoscaler{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("podautoscalers").
		Name(podAutoscaler.Name).
		SubResource("status").
		Body(podAutoscaler).
		Do().
		Into(result)
	return
}

// Delete takes name of the podAutoscaler and deletes it. Returns an error if one occurs.
func (c *podAutoscalers) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("podautoscalers").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *podAutoscalers) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("podautoscalers").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched podAutoscaler.
func (c *podAutoscalers) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.PodAutoscaler, err error) {
	result = &v1alpha1.PodAutoscaler{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("podautoscalers").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
