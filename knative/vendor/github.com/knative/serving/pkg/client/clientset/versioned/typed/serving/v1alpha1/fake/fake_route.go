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
	v1alpha1 "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeRoutes implements RouteInterface
type FakeRoutes struct {
	Fake *FakeServingV1alpha1
	ns   string
}

var routesResource = schema.GroupVersionResource{Group: "serving.knative.dev", Version: "v1alpha1", Resource: "routes"}

var routesKind = schema.GroupVersionKind{Group: "serving.knative.dev", Version: "v1alpha1", Kind: "Route"}

// Get takes name of the route, and returns the corresponding route object, and an error if there is any.
func (c *FakeRoutes) Get(name string, options v1.GetOptions) (result *v1alpha1.Route, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(routesResource, c.ns, name), &v1alpha1.Route{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Route), err
}

// List takes label and field selectors, and returns the list of Routes that match those selectors.
func (c *FakeRoutes) List(opts v1.ListOptions) (result *v1alpha1.RouteList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(routesResource, routesKind, c.ns, opts), &v1alpha1.RouteList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.RouteList{ListMeta: obj.(*v1alpha1.RouteList).ListMeta}
	for _, item := range obj.(*v1alpha1.RouteList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested routes.
func (c *FakeRoutes) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(routesResource, c.ns, opts))

}

// Create takes the representation of a route and creates it.  Returns the server's representation of the route, and an error, if there is any.
func (c *FakeRoutes) Create(route *v1alpha1.Route) (result *v1alpha1.Route, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(routesResource, c.ns, route), &v1alpha1.Route{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Route), err
}

// Update takes the representation of a route and updates it. Returns the server's representation of the route, and an error, if there is any.
func (c *FakeRoutes) Update(route *v1alpha1.Route) (result *v1alpha1.Route, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(routesResource, c.ns, route), &v1alpha1.Route{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Route), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeRoutes) UpdateStatus(route *v1alpha1.Route) (*v1alpha1.Route, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(routesResource, "status", c.ns, route), &v1alpha1.Route{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Route), err
}

// Delete takes name of the route and deletes it. Returns an error if one occurs.
func (c *FakeRoutes) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(routesResource, c.ns, name), &v1alpha1.Route{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeRoutes) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(routesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.RouteList{})
	return err
}

// Patch applies the patch and returns the patched route.
func (c *FakeRoutes) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Route, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(routesResource, c.ns, name, data, subresources...), &v1alpha1.Route{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Route), err
}
