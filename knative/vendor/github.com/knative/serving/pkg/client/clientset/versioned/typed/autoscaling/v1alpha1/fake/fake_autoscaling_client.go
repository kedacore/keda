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
	v1alpha1 "github.com/knative/serving/pkg/client/clientset/versioned/typed/autoscaling/v1alpha1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeAutoscalingV1alpha1 struct {
	*testing.Fake
}

func (c *FakeAutoscalingV1alpha1) PodAutoscalers(namespace string) v1alpha1.PodAutoscalerInterface {
	return &FakePodAutoscalers{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeAutoscalingV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
