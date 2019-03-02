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
	v1alpha1 "github.com/knative/serving/pkg/apis/autoscaling/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakePodAutoscalers implements PodAutoscalerInterface
type FakePodAutoscalers struct {
	Fake *FakeAutoscalingV1alpha1
	ns   string
}

var podautoscalersResource = schema.GroupVersionResource{Group: "autoscaling.internal.knative.dev", Version: "v1alpha1", Resource: "podautoscalers"}

var podautoscalersKind = schema.GroupVersionKind{Group: "autoscaling.internal.knative.dev", Version: "v1alpha1", Kind: "PodAutoscaler"}

// Get takes name of the podAutoscaler, and returns the corresponding podAutoscaler object, and an error if there is any.
func (c *FakePodAutoscalers) Get(name string, options v1.GetOptions) (result *v1alpha1.PodAutoscaler, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(podautoscalersResource, c.ns, name), &v1alpha1.PodAutoscaler{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.PodAutoscaler), err
}

// List takes label and field selectors, and returns the list of PodAutoscalers that match those selectors.
func (c *FakePodAutoscalers) List(opts v1.ListOptions) (result *v1alpha1.PodAutoscalerList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(podautoscalersResource, podautoscalersKind, c.ns, opts), &v1alpha1.PodAutoscalerList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.PodAutoscalerList{ListMeta: obj.(*v1alpha1.PodAutoscalerList).ListMeta}
	for _, item := range obj.(*v1alpha1.PodAutoscalerList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested podAutoscalers.
func (c *FakePodAutoscalers) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(podautoscalersResource, c.ns, opts))

}

// Create takes the representation of a podAutoscaler and creates it.  Returns the server's representation of the podAutoscaler, and an error, if there is any.
func (c *FakePodAutoscalers) Create(podAutoscaler *v1alpha1.PodAutoscaler) (result *v1alpha1.PodAutoscaler, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(podautoscalersResource, c.ns, podAutoscaler), &v1alpha1.PodAutoscaler{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.PodAutoscaler), err
}

// Update takes the representation of a podAutoscaler and updates it. Returns the server's representation of the podAutoscaler, and an error, if there is any.
func (c *FakePodAutoscalers) Update(podAutoscaler *v1alpha1.PodAutoscaler) (result *v1alpha1.PodAutoscaler, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(podautoscalersResource, c.ns, podAutoscaler), &v1alpha1.PodAutoscaler{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.PodAutoscaler), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakePodAutoscalers) UpdateStatus(podAutoscaler *v1alpha1.PodAutoscaler) (*v1alpha1.PodAutoscaler, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(podautoscalersResource, "status", c.ns, podAutoscaler), &v1alpha1.PodAutoscaler{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.PodAutoscaler), err
}

// Delete takes name of the podAutoscaler and deletes it. Returns an error if one occurs.
func (c *FakePodAutoscalers) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(podautoscalersResource, c.ns, name), &v1alpha1.PodAutoscaler{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakePodAutoscalers) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(podautoscalersResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.PodAutoscalerList{})
	return err
}

// Patch applies the patch and returns the patched podAutoscaler.
func (c *FakePodAutoscalers) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.PodAutoscaler, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(podautoscalersResource, c.ns, name, data, subresources...), &v1alpha1.PodAutoscaler{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.PodAutoscaler), err
}
