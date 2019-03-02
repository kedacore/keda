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
	v1alpha1 "github.com/knative/serving/test/apis/testing/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeBuilds implements BuildInterface
type FakeBuilds struct {
	Fake *FakeTestingV1alpha1
	ns   string
}

var buildsResource = schema.GroupVersionResource{Group: "testing.internal.knative.dev", Version: "v1alpha1", Resource: "builds"}

var buildsKind = schema.GroupVersionKind{Group: "testing.internal.knative.dev", Version: "v1alpha1", Kind: "Build"}

// Get takes name of the build, and returns the corresponding build object, and an error if there is any.
func (c *FakeBuilds) Get(name string, options v1.GetOptions) (result *v1alpha1.Build, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(buildsResource, c.ns, name), &v1alpha1.Build{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Build), err
}

// List takes label and field selectors, and returns the list of Builds that match those selectors.
func (c *FakeBuilds) List(opts v1.ListOptions) (result *v1alpha1.BuildList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(buildsResource, buildsKind, c.ns, opts), &v1alpha1.BuildList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.BuildList{ListMeta: obj.(*v1alpha1.BuildList).ListMeta}
	for _, item := range obj.(*v1alpha1.BuildList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested builds.
func (c *FakeBuilds) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(buildsResource, c.ns, opts))

}

// Create takes the representation of a build and creates it.  Returns the server's representation of the build, and an error, if there is any.
func (c *FakeBuilds) Create(build *v1alpha1.Build) (result *v1alpha1.Build, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(buildsResource, c.ns, build), &v1alpha1.Build{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Build), err
}

// Update takes the representation of a build and updates it. Returns the server's representation of the build, and an error, if there is any.
func (c *FakeBuilds) Update(build *v1alpha1.Build) (result *v1alpha1.Build, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(buildsResource, c.ns, build), &v1alpha1.Build{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Build), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeBuilds) UpdateStatus(build *v1alpha1.Build) (*v1alpha1.Build, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(buildsResource, "status", c.ns, build), &v1alpha1.Build{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Build), err
}

// Delete takes name of the build and deletes it. Returns an error if one occurs.
func (c *FakeBuilds) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(buildsResource, c.ns, name), &v1alpha1.Build{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeBuilds) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(buildsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.BuildList{})
	return err
}

// Patch applies the patch and returns the patched build.
func (c *FakeBuilds) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Build, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(buildsResource, c.ns, name, data, subresources...), &v1alpha1.Build{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Build), err
}
