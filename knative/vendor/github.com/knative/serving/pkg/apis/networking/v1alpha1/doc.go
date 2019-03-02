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

// +k8s:deepcopy-gen=package
// +groupName=networking.internal.knative.dev
package v1alpha1

// ClusterIngress is heavily based on K8s Ingress
// https://godoc.org/k8s.io/api/extensions/v1beta1#Ingress with some
// highlighted modifications.  See clusteringress_types.go for more
// information about the modifications that we made.
