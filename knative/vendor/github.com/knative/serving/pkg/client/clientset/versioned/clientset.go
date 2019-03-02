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
package versioned

import (
	autoscalingv1alpha1 "github.com/knative/serving/pkg/client/clientset/versioned/typed/autoscaling/v1alpha1"
	networkingv1alpha1 "github.com/knative/serving/pkg/client/clientset/versioned/typed/networking/v1alpha1"
	servingv1alpha1 "github.com/knative/serving/pkg/client/clientset/versioned/typed/serving/v1alpha1"
	discovery "k8s.io/client-go/discovery"
	rest "k8s.io/client-go/rest"
	flowcontrol "k8s.io/client-go/util/flowcontrol"
)

type Interface interface {
	Discovery() discovery.DiscoveryInterface
	AutoscalingV1alpha1() autoscalingv1alpha1.AutoscalingV1alpha1Interface
	// Deprecated: please explicitly pick a version if possible.
	Autoscaling() autoscalingv1alpha1.AutoscalingV1alpha1Interface
	NetworkingV1alpha1() networkingv1alpha1.NetworkingV1alpha1Interface
	// Deprecated: please explicitly pick a version if possible.
	Networking() networkingv1alpha1.NetworkingV1alpha1Interface
	ServingV1alpha1() servingv1alpha1.ServingV1alpha1Interface
	// Deprecated: please explicitly pick a version if possible.
	Serving() servingv1alpha1.ServingV1alpha1Interface
}

// Clientset contains the clients for groups. Each group has exactly one
// version included in a Clientset.
type Clientset struct {
	*discovery.DiscoveryClient
	autoscalingV1alpha1 *autoscalingv1alpha1.AutoscalingV1alpha1Client
	networkingV1alpha1  *networkingv1alpha1.NetworkingV1alpha1Client
	servingV1alpha1     *servingv1alpha1.ServingV1alpha1Client
}

// AutoscalingV1alpha1 retrieves the AutoscalingV1alpha1Client
func (c *Clientset) AutoscalingV1alpha1() autoscalingv1alpha1.AutoscalingV1alpha1Interface {
	return c.autoscalingV1alpha1
}

// Deprecated: Autoscaling retrieves the default version of AutoscalingClient.
// Please explicitly pick a version.
func (c *Clientset) Autoscaling() autoscalingv1alpha1.AutoscalingV1alpha1Interface {
	return c.autoscalingV1alpha1
}

// NetworkingV1alpha1 retrieves the NetworkingV1alpha1Client
func (c *Clientset) NetworkingV1alpha1() networkingv1alpha1.NetworkingV1alpha1Interface {
	return c.networkingV1alpha1
}

// Deprecated: Networking retrieves the default version of NetworkingClient.
// Please explicitly pick a version.
func (c *Clientset) Networking() networkingv1alpha1.NetworkingV1alpha1Interface {
	return c.networkingV1alpha1
}

// ServingV1alpha1 retrieves the ServingV1alpha1Client
func (c *Clientset) ServingV1alpha1() servingv1alpha1.ServingV1alpha1Interface {
	return c.servingV1alpha1
}

// Deprecated: Serving retrieves the default version of ServingClient.
// Please explicitly pick a version.
func (c *Clientset) Serving() servingv1alpha1.ServingV1alpha1Interface {
	return c.servingV1alpha1
}

// Discovery retrieves the DiscoveryClient
func (c *Clientset) Discovery() discovery.DiscoveryInterface {
	if c == nil {
		return nil
	}
	return c.DiscoveryClient
}

// NewForConfig creates a new Clientset for the given config.
func NewForConfig(c *rest.Config) (*Clientset, error) {
	configShallowCopy := *c
	if configShallowCopy.RateLimiter == nil && configShallowCopy.QPS > 0 {
		configShallowCopy.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(configShallowCopy.QPS, configShallowCopy.Burst)
	}
	var cs Clientset
	var err error
	cs.autoscalingV1alpha1, err = autoscalingv1alpha1.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	cs.networkingV1alpha1, err = networkingv1alpha1.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	cs.servingV1alpha1, err = servingv1alpha1.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}

	cs.DiscoveryClient, err = discovery.NewDiscoveryClientForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	return &cs, nil
}

// NewForConfigOrDie creates a new Clientset for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *Clientset {
	var cs Clientset
	cs.autoscalingV1alpha1 = autoscalingv1alpha1.NewForConfigOrDie(c)
	cs.networkingV1alpha1 = networkingv1alpha1.NewForConfigOrDie(c)
	cs.servingV1alpha1 = servingv1alpha1.NewForConfigOrDie(c)

	cs.DiscoveryClient = discovery.NewDiscoveryClientForConfigOrDie(c)
	return &cs
}

// New creates a new Clientset for the given RESTClient.
func New(c rest.Interface) *Clientset {
	var cs Clientset
	cs.autoscalingV1alpha1 = autoscalingv1alpha1.New(c)
	cs.networkingV1alpha1 = networkingv1alpha1.New(c)
	cs.servingV1alpha1 = servingv1alpha1.New(c)

	cs.DiscoveryClient = discovery.NewDiscoveryClient(c)
	return &cs
}
