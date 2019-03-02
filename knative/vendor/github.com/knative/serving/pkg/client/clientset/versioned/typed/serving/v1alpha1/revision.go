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
	v1alpha1 "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	scheme "github.com/knative/serving/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// RevisionsGetter has a method to return a RevisionInterface.
// A group's client should implement this interface.
type RevisionsGetter interface {
	Revisions(namespace string) RevisionInterface
}

// RevisionInterface has methods to work with Revision resources.
type RevisionInterface interface {
	Create(*v1alpha1.Revision) (*v1alpha1.Revision, error)
	Update(*v1alpha1.Revision) (*v1alpha1.Revision, error)
	UpdateStatus(*v1alpha1.Revision) (*v1alpha1.Revision, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.Revision, error)
	List(opts v1.ListOptions) (*v1alpha1.RevisionList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Revision, err error)
	RevisionExpansion
}

// revisions implements RevisionInterface
type revisions struct {
	client rest.Interface
	ns     string
}

// newRevisions returns a Revisions
func newRevisions(c *ServingV1alpha1Client, namespace string) *revisions {
	return &revisions{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the revision, and returns the corresponding revision object, and an error if there is any.
func (c *revisions) Get(name string, options v1.GetOptions) (result *v1alpha1.Revision, err error) {
	result = &v1alpha1.Revision{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("revisions").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Revisions that match those selectors.
func (c *revisions) List(opts v1.ListOptions) (result *v1alpha1.RevisionList, err error) {
	result = &v1alpha1.RevisionList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("revisions").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested revisions.
func (c *revisions) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("revisions").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a revision and creates it.  Returns the server's representation of the revision, and an error, if there is any.
func (c *revisions) Create(revision *v1alpha1.Revision) (result *v1alpha1.Revision, err error) {
	result = &v1alpha1.Revision{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("revisions").
		Body(revision).
		Do().
		Into(result)
	return
}

// Update takes the representation of a revision and updates it. Returns the server's representation of the revision, and an error, if there is any.
func (c *revisions) Update(revision *v1alpha1.Revision) (result *v1alpha1.Revision, err error) {
	result = &v1alpha1.Revision{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("revisions").
		Name(revision.Name).
		Body(revision).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *revisions) UpdateStatus(revision *v1alpha1.Revision) (result *v1alpha1.Revision, err error) {
	result = &v1alpha1.Revision{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("revisions").
		Name(revision.Name).
		SubResource("status").
		Body(revision).
		Do().
		Into(result)
	return
}

// Delete takes name of the revision and deletes it. Returns an error if one occurs.
func (c *revisions) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("revisions").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *revisions) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("revisions").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched revision.
func (c *revisions) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Revision, err error) {
	result = &v1alpha1.Revision{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("revisions").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
