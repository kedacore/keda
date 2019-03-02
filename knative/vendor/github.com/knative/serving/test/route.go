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

// route.go provides methods to perform actions on the route resource.

package test

import (
	"testing"

	"github.com/knative/serving/pkg/apis/serving/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	rtesting "github.com/knative/serving/pkg/reconciler/v1alpha1/testing"
)

// CreateRoute creates a route in the given namespace using the route name in names
func CreateRoute(t *testing.T, clients *Clients, names ResourceNames, fopt ...rtesting.RouteOption) (*v1alpha1.Route, error) {
	route := Route(ServingNamespace, names, fopt...)
	LogResourceObject(t, ResourceObjects{Route: route})
	return clients.ServingClient.Routes.Create(route)
}

// CreateBlueGreenRoute creates a route in the given namespace using the route name in names.
// Traffic is evenly split between the two routes specified by blue and green.
func CreateBlueGreenRoute(t *testing.T, clients *Clients, names, blue, green ResourceNames) error {
	route := BlueGreenRoute(ServingNamespace, names, blue, green)
	LogResourceObject(t, ResourceObjects{Route: route})
	_, err := clients.ServingClient.Routes.Create(route)
	return err
}

// UpdateRoute updates a route in the given namespace using the route name in names
func UpdateBlueGreenRoute(t *testing.T, clients *Clients, names, blue, green ResourceNames) (*v1alpha1.Route, error) {
	route, err := clients.ServingClient.Routes.Get(names.Route, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	newRoute := BlueGreenRoute(ServingNamespace, names, blue, green)
	newRoute.ObjectMeta = route.ObjectMeta
	LogResourceObject(t, ResourceObjects{Route: newRoute})
	patchBytes, err := createPatch(route, newRoute)
	if err != nil {
		return nil, err
	}
	return clients.ServingClient.Routes.Patch(names.Route, types.JSONPatchType, patchBytes, "")
}
