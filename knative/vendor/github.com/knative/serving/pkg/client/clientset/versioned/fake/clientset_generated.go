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
	clientset "github.com/knative/serving/pkg/client/clientset/versioned"
	autoscalingv1alpha1 "github.com/knative/serving/pkg/client/clientset/versioned/typed/autoscaling/v1alpha1"
	fakeautoscalingv1alpha1 "github.com/knative/serving/pkg/client/clientset/versioned/typed/autoscaling/v1alpha1/fake"
	networkingv1alpha1 "github.com/knative/serving/pkg/client/clientset/versioned/typed/networking/v1alpha1"
	fakenetworkingv1alpha1 "github.com/knative/serving/pkg/client/clientset/versioned/typed/networking/v1alpha1/fake"
	servingv1alpha1 "github.com/knative/serving/pkg/client/clientset/versioned/typed/serving/v1alpha1"
	fakeservingv1alpha1 "github.com/knative/serving/pkg/client/clientset/versioned/typed/serving/v1alpha1/fake"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/testing"
)

// NewSimpleClientset returns a clientset that will respond with the provided objects.
// It's backed by a very simple object tracker that processes creates, updates and deletions as-is,
// without applying any validations and/or defaults. It shouldn't be considered a replacement
// for a real clientset and is mostly useful in simple unit tests.
func NewSimpleClientset(objects ...runtime.Object) *Clientset {
	o := testing.NewObjectTracker(scheme, codecs.UniversalDecoder())
	for _, obj := range objects {
		if err := o.Add(obj); err != nil {
			panic(err)
		}
	}

	cs := &Clientset{}
	cs.discovery = &fakediscovery.FakeDiscovery{Fake: &cs.Fake}
	cs.AddReactor("*", "*", testing.ObjectReaction(o))
	cs.AddWatchReactor("*", func(action testing.Action) (handled bool, ret watch.Interface, err error) {
		gvr := action.GetResource()
		ns := action.GetNamespace()
		watch, err := o.Watch(gvr, ns)
		if err != nil {
			return false, nil, err
		}
		return true, watch, nil
	})

	return cs
}

// Clientset implements clientset.Interface. Meant to be embedded into a
// struct to get a default implementation. This makes faking out just the method
// you want to test easier.
type Clientset struct {
	testing.Fake
	discovery *fakediscovery.FakeDiscovery
}

func (c *Clientset) Discovery() discovery.DiscoveryInterface {
	return c.discovery
}

var _ clientset.Interface = &Clientset{}

// AutoscalingV1alpha1 retrieves the AutoscalingV1alpha1Client
func (c *Clientset) AutoscalingV1alpha1() autoscalingv1alpha1.AutoscalingV1alpha1Interface {
	return &fakeautoscalingv1alpha1.FakeAutoscalingV1alpha1{Fake: &c.Fake}
}

// Autoscaling retrieves the AutoscalingV1alpha1Client
func (c *Clientset) Autoscaling() autoscalingv1alpha1.AutoscalingV1alpha1Interface {
	return &fakeautoscalingv1alpha1.FakeAutoscalingV1alpha1{Fake: &c.Fake}
}

// NetworkingV1alpha1 retrieves the NetworkingV1alpha1Client
func (c *Clientset) NetworkingV1alpha1() networkingv1alpha1.NetworkingV1alpha1Interface {
	return &fakenetworkingv1alpha1.FakeNetworkingV1alpha1{Fake: &c.Fake}
}

// Networking retrieves the NetworkingV1alpha1Client
func (c *Clientset) Networking() networkingv1alpha1.NetworkingV1alpha1Interface {
	return &fakenetworkingv1alpha1.FakeNetworkingV1alpha1{Fake: &c.Fake}
}

// ServingV1alpha1 retrieves the ServingV1alpha1Client
func (c *Clientset) ServingV1alpha1() servingv1alpha1.ServingV1alpha1Interface {
	return &fakeservingv1alpha1.FakeServingV1alpha1{Fake: &c.Fake}
}

// Serving retrieves the ServingV1alpha1Client
func (c *Clientset) Serving() servingv1alpha1.ServingV1alpha1Interface {
	return &fakeservingv1alpha1.FakeServingV1alpha1{Fake: &c.Fake}
}
