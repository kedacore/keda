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
	v1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeBuildTemplates implements BuildTemplateInterface
type FakeBuildTemplates struct {
	Fake *FakeBuildV1alpha1
	ns   string
}

var buildtemplatesResource = schema.GroupVersionResource{Group: "build.knative.dev", Version: "v1alpha1", Resource: "buildtemplates"}

var buildtemplatesKind = schema.GroupVersionKind{Group: "build.knative.dev", Version: "v1alpha1", Kind: "BuildTemplate"}

// Get takes name of the buildTemplate, and returns the corresponding buildTemplate object, and an error if there is any.
func (c *FakeBuildTemplates) Get(name string, options v1.GetOptions) (result *v1alpha1.BuildTemplate, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(buildtemplatesResource, c.ns, name), &v1alpha1.BuildTemplate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.BuildTemplate), err
}

// List takes label and field selectors, and returns the list of BuildTemplates that match those selectors.
func (c *FakeBuildTemplates) List(opts v1.ListOptions) (result *v1alpha1.BuildTemplateList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(buildtemplatesResource, buildtemplatesKind, c.ns, opts), &v1alpha1.BuildTemplateList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.BuildTemplateList{ListMeta: obj.(*v1alpha1.BuildTemplateList).ListMeta}
	for _, item := range obj.(*v1alpha1.BuildTemplateList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested buildTemplates.
func (c *FakeBuildTemplates) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(buildtemplatesResource, c.ns, opts))

}

// Create takes the representation of a buildTemplate and creates it.  Returns the server's representation of the buildTemplate, and an error, if there is any.
func (c *FakeBuildTemplates) Create(buildTemplate *v1alpha1.BuildTemplate) (result *v1alpha1.BuildTemplate, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(buildtemplatesResource, c.ns, buildTemplate), &v1alpha1.BuildTemplate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.BuildTemplate), err
}

// Update takes the representation of a buildTemplate and updates it. Returns the server's representation of the buildTemplate, and an error, if there is any.
func (c *FakeBuildTemplates) Update(buildTemplate *v1alpha1.BuildTemplate) (result *v1alpha1.BuildTemplate, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(buildtemplatesResource, c.ns, buildTemplate), &v1alpha1.BuildTemplate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.BuildTemplate), err
}

// Delete takes name of the buildTemplate and deletes it. Returns an error if one occurs.
func (c *FakeBuildTemplates) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(buildtemplatesResource, c.ns, name), &v1alpha1.BuildTemplate{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeBuildTemplates) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(buildtemplatesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.BuildTemplateList{})
	return err
}

// Patch applies the patch and returns the patched buildTemplate.
func (c *FakeBuildTemplates) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.BuildTemplate, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(buildtemplatesResource, c.ns, name, data, subresources...), &v1alpha1.BuildTemplate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.BuildTemplate), err
}
