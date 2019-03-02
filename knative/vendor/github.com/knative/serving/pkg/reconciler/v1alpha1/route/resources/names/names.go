/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package names

import (
	"fmt"

	"github.com/knative/serving/pkg/apis/serving/v1alpha1"
	"github.com/knative/serving/pkg/reconciler"
)

func K8sService(route *v1alpha1.Route) string {
	return route.Name
}

func K8sServiceFullname(route *v1alpha1.Route) string {
	return reconciler.GetK8sServiceFullname(K8sService(route), route.Namespace)
}

// ClusterIngress returns the name for the ClusterIngress
// child resource for the given Route.
func ClusterIngress(route *v1alpha1.Route) string {
	return fmt.Sprintf("route-%s", route.UID)
}
