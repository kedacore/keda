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

package networking

const (
	GroupName = "networking.internal.knative.dev"

	// IngressClassAnnotationKey is the annotation for the
	// explicit class of ClusterIngress that a particular resource has
	// opted into. For example,
	//
	//    networking.knative.dev/ingress.class: some-network-impl
	//
	// This uses a different domain because unlike the resource, it is
	// user-facing.
	//
	// The parent resource may use its own annotations to choose the
	// annotation value for the ClusterIngress it uses.  Based on such
	// value a different reconciliation logic may be used (for examples,
	// Istio-based ClusterIngress will reconcile into a VirtualService).
	IngressClassAnnotationKey = "networking.knative.dev/ingress.class"

	// IngressLabelKey is the label key attached to underlying network programming
	// resources to indicate which ClusterIngress triggered their creation.
	IngressLabelKey = GroupName + "/clusteringress"
)
