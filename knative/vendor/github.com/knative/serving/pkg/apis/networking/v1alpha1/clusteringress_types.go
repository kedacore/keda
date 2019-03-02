/*
Copyright 2018 The Knative Authors.

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

package v1alpha1

import (
	"fmt"

	"github.com/knative/pkg/apis"
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced

// ClusterIngress is a collection of rules that allow inbound connections to reach the
// endpoints defined by a backend. An ClusterIngress can be configured to give services
// externally-reachable urls, load balance traffic offer name based virtual hosting etc.
//
// This is heavily based on K8s Ingress https://godoc.org/k8s.io/api/extensions/v1beta1#Ingress
// which some highlighted modifications.
type ClusterIngress struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the desired state of the ClusterIngress.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#spec-and-status
	// +optional
	Spec IngressSpec `json:"spec,omitempty"`

	// Status is the current state of the ClusterIngress.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#spec-and-status
	// +optional
	Status IngressStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterIngressList is a collection of ClusterIngress.
type ClusterIngressList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of ClusterIngress.
	Items []ClusterIngress `json:"items"`
}

// IngressSpec describes the ClusterIngress the user wishes to exist.
//
// In general this follow the same shape as K8s Ingress.  Some notable differences:
// - Backends now can have namespace:
// - Traffic can be split across multiple backends.
// - Timeout & Retry can be configured.
// - Headers can be appended.
type IngressSpec struct {
	// DeprecatedGeneration was used prior in Kubernetes versions <1.11
	// when metadata.generation was not being incremented by the api server
	//
	// This property will be dropped in future Knative releases and should
	// not be used - use metadata.generation
	//
	// Tracking issue: https://github.com/knative/serving/issues/643
	//
	// +optional
	DeprecatedGeneration int64 `json:"generation,omitempty"`

	// TLS configuration. Currently the ClusterIngress only supports a single TLS
	// port, 443. If multiple members of this list specify different hosts, they
	// will be multiplexed on the same port according to the hostname specified
	// through the SNI TLS extension, if the ingress controller fulfilling the
	// ingress supports SNI.
	// +optional
	TLS []ClusterIngressTLS `json:"tls,omitempty"`

	// A list of host rules used to configure the ClusterIngress.
	// +optional
	Rules []ClusterIngressRule `json:"rules,omitempty"`

	// Visibility setting.
	Visibility IngressVisibility `json:"visibility,omitempty"`
}

// IngressVisibility describes whether the Ingress should be exposed to
// public gateways or not.
type IngressVisibility string

const (
	// IngressVisibilityExternalIP is used to denote that the Ingress
	// should be exposed to an external IP, for example a LoadBalancer
	// Service.  This is the default value for IngressVisibility.
	IngressVisibilityExternalIP IngressVisibility = "ExternalIP"
	// IngressVisibilityClusterLocal is used to denote that the Ingress
	// should be only be exposed locally to the cluster.
	IngressVisibilityClusterLocal IngressVisibility = "ClusterLocal"
)

// ClusterIngressTLS describes the transport layer security associated with an ClusterIngress.
type ClusterIngressTLS struct {
	// Hosts are a list of hosts included in the TLS certificate. The values in
	// this list must match the name/s used in the tlsSecret. Defaults to the
	// wildcard host setting for the loadbalancer controller fulfilling this
	// ClusterIngress, if left unspecified.
	// +optional
	Hosts []string `json:"hosts,omitempty"`

	// SecretName is the name of the secret used to terminate SSL traffic.
	SecretName string `json:"secretName,omitempty"`

	// SecretNamespace is the namespace of the secret used to terminate SSL traffic.
	SecretNamespace string `json:"secretNamespace,omitempty"`

	// ServerCertificate identifies the certificate filename in the secret.
	// Defaults to `tls.cert`.
	// +optional
	ServerCertificate string `json:"serverCertificate,omitempty"`

	// PrivateKey identifies the private key filename in the secret.
	// Defaults to `tls.key`.
	// +optional
	PrivateKey string `json:"privateKey,omitempty"`
}

// ClusterIngressRule represents the rules mapping the paths under a specified host to
// the related backend services. Incoming requests are first evaluated for a host
// match, then routed to the backend associated with the matching ClusterIngressRuleValue.
type ClusterIngressRule struct {
	// Host is the fully qualified domain name of a network host, as defined
	// by RFC 3986. Note the following deviations from the "host" part of the
	// URI as defined in the RFC:
	// 1. IPs are not allowed. Currently a rule value can only apply to the
	//	  IP in the Spec of the parent ClusterIngress.
	// 2. The `:` delimiter is not respected because ports are not allowed.
	//	  Currently the port of an ClusterIngress is implicitly :80 for http and
	//	  :443 for https.
	// Both these may change in the future.
	// If the host is unspecified, the ClusterIngress routes all traffic based on the
	// specified ClusterIngressRuleValue.
	// If multiple matching Hosts were provided, the first rule will take precedent.
	// +optional
	Hosts []string `json:"hosts,omitempty"`

	// HTTP represents a rule to apply against incoming requests. If the
	// rule is satisfied, the request is routed to the specified backend.
	HTTP *HTTPClusterIngressRuleValue `json:"http,omitempty"`
}

// HTTPClusterIngressRuleValue is a list of http selectors pointing to backends.
// In the example: http://<host>/<path>?<searchpart> -> backend where
// where parts of the url correspond to RFC 3986, this resource will be used
// to match against everything after the last '/' and before the first '?'
// or '#'.
type HTTPClusterIngressRuleValue struct {
	// A collection of paths that map requests to backends.
	//
	// If they are multiple matching paths, the first match takes precendent.
	Paths []HTTPClusterIngressPath `json:"paths"`

	// TODO: Consider adding fields for ingress-type specific global
	// options usable by a loadbalancer, like http keep-alive.
}

// HTTPClusterIngressPath associates a path regex with a backend. Incoming urls matching
// the path are forwarded to the backend.
type HTTPClusterIngressPath struct {
	// Path is an extended POSIX regex as defined by IEEE Std 1003.1,
	// (i.e this follows the egrep/unix syntax, not the perl syntax)
	// matched against the path of an incoming request. Currently it can
	// contain characters disallowed from the conventional "path"
	// part of a URL as defined by RFC 3986. Paths must begin with
	// a '/'. If unspecified, the path defaults to a catch all sending
	// traffic to the backend.
	// +optional
	Path string `json:"path,omitempty"`

	// Splits defines the referenced service endpoints to which the traffic
	// will be forwarded to.
	Splits []ClusterIngressBackendSplit `json:"splits"`

	// AppendHeaders allow specifying additional HTTP headers to add
	// before forwarding a request to the destination service.
	//
	// NOTE: This differs from K8s Ingress which doesn't allow header appending.
	// +optional
	AppendHeaders map[string]string `json:"appendHeaders,omitempty"`

	// Timeout for HTTP requests.
	//
	// NOTE: This differs from K8s Ingress which doesn't allow setting timeouts.
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// Retry policy for HTTP requests.
	//
	// NOTE: This differs from K8s Ingress which doesn't allow retry settings.
	// +optional
	Retries *HTTPRetry `json:"retries,omitempty"`
}

// ClusterIngressBackend describes all endpoints for a given service and port.
type ClusterIngressBackendSplit struct {
	// Specifies the backend receiving the traffic split.
	ClusterIngressBackend `json:",inline"`

	// Specifies the split percentage, a number between 0 and 100.  If
	// only one split is specified, we default to 100.
	//
	// NOTE: This differs from K8s Ingress to allow percentage split.
	Percent int `json:"percent,omitempty"`
}

// ClusterIngressBackend describes all endpoints for a given service and port.
type ClusterIngressBackend struct {
	// Specifies the namespace of the referenced service.
	//
	// NOTE: This differs from K8s Ingress to allow routing to different namespaces.
	ServiceNamespace string `json:"serviceNamespace"`

	// Specifies the name of the referenced service.
	ServiceName string `json:"serviceName"`

	// Specifies the port of the referenced service.
	ServicePort intstr.IntOrString `json:"servicePort"`
}

// HTTPRetry describes the retry policy to use when a HTTP request fails.
type HTTPRetry struct {
	// Number of retries for a given request.
	Attempts int `json:"attempts"`

	// Timeout per retry attempt for a given request. format: 1h/1m/1s/1ms. MUST BE >=1ms.
	PerTryTimeout *metav1.Duration `json:"perTryTimeout"`
}

// IngressStatus describe the current state of the ClusterIngress.
type IngressStatus struct {
	// +optional
	Conditions duckv1alpha1.Conditions `json:"conditions,omitempty"`
	// LoadBalancer contains the current status of the load-balancer.
	// +optional
	LoadBalancer *LoadBalancerStatus `json:"loadBalancer,omitempty"`

	// ObservedGeneration is the 'Generation' of the ClusterIngress that
	// was last processed by the controller. The observed generation is updated
	// even if the controller failed to process the spec.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// LoadBalancerStatus represents the status of a load-balancer.
type LoadBalancerStatus struct {
	// Ingress is a list containing ingress points for the load-balancer.
	// Traffic intended for the service should be sent to these ingress points.
	// +optional
	Ingress []LoadBalancerIngressStatus `json:"ingress,omitempty"`
}

// LoadBalancerIngress represents the status of a load-balancer ingress point:
// traffic intended for the service should be sent to an ingress point.
type LoadBalancerIngressStatus struct {
	// IP is set for load-balancer ingress points that are IP based
	// (typically GCE or OpenStack load-balancers)
	// +optional
	IP string `json:"ip,omitempty"`

	// Domain is set for load-balancer ingress points that are DNS based
	// (typically AWS load-balancers)
	// +optional
	Domain string `json:"domain,omitempty"`

	// DomainInternal is set if there is a cluster-local DNS name to access the Ingress.
	//
	// NOTE: This differs from K8s Ingress, since we also desire to have a cluster-local
	//       DNS name to allow routing in case of not having a mesh.
	//
	// +optional
	DomainInternal string `json:"domainInternal,omitempty"`

	// MeshOnly is set if the ClusterIngress is only load-balanced through a Service mesh.
	// +optional
	MeshOnly bool `json:"meshOnly,omitempty"`
}

// ConditionType represents a ClusterIngress condition value
const (
	// ClusterIngressConditionReady is set when the clusterIngress networking setting is
	// configured and it has a load balancer address.
	ClusterIngressConditionReady = duckv1alpha1.ConditionReady

	// ClusterIngressConditionNetworkConfigured is set when the ClusterIngress's underlying
	// network programming has been configured.  This doesn't include conditions of the
	// backends, so even if this should remain true when network is configured and backends
	// are not ready.
	ClusterIngressConditionNetworkConfigured duckv1alpha1.ConditionType = "NetworkConfigured"

	// ClusterIngressConditionLoadBalancerReady is set when the ClusterIngress has
	// a ready LoadBalancer.
	ClusterIngressConditionLoadBalancerReady duckv1alpha1.ConditionType = "LoadBalancerReady"
)

var clusterIngressCondSet = duckv1alpha1.NewLivingConditionSet(
	ClusterIngressConditionNetworkConfigured,
	ClusterIngressConditionLoadBalancerReady)

var _ apis.Validatable = (*ClusterIngress)(nil)
var _ apis.Defaultable = (*ClusterIngress)(nil)

func (ci *ClusterIngress) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("ClusterIngress")
}

// IsPublic returns whether the ClusterIngress should be exposed publicly.
func (ci *ClusterIngress) IsPublic() bool {
	return ci.Spec.Visibility == "" || ci.Spec.Visibility == IngressVisibilityExternalIP
}

// GetConditions returns the Conditions array. This enables generic handling of
// conditions by implementing the duckv1alpha1.Conditions interface.
func (cis *IngressStatus) GetConditions() duckv1alpha1.Conditions {
	return cis.Conditions
}

// SetConditions sets the Conditions array. This enables generic handling of
// conditions by implementing the duckv1alpha1.Conditions interface.
func (cis *IngressStatus) SetConditions(conditions duckv1alpha1.Conditions) {
	cis.Conditions = conditions
}

func (cis *IngressStatus) GetCondition(t duckv1alpha1.ConditionType) *duckv1alpha1.Condition {
	return clusterIngressCondSet.Manage(cis).GetCondition(t)
}

func (cis *IngressStatus) InitializeConditions() {
	clusterIngressCondSet.Manage(cis).InitializeConditions()
}

func (cis *IngressStatus) MarkNetworkConfigured() {
	clusterIngressCondSet.Manage(cis).MarkTrue(ClusterIngressConditionNetworkConfigured)
}

// MarkResourceNotOwned changes the "NetworkConfigured" condition to false to reflect that the
// resource of the given kind and name has already been created, and we do not own it.
func (cis *IngressStatus) MarkResourceNotOwned(kind, name string) {
	clusterIngressCondSet.Manage(cis).MarkFalse(ClusterIngressConditionNetworkConfigured, "NotOwned",
		fmt.Sprintf("There is an existing %s %q that we do not own.", kind, name))
}

// MarkLoadBalancerReady marks the Ingress with ClusterIngressConditionLoadBalancerReady,
// and also populate the address of the load balancer.
func (cis *IngressStatus) MarkLoadBalancerReady(lbs []LoadBalancerIngressStatus) {
	cis.LoadBalancer = &LoadBalancerStatus{
		Ingress: []LoadBalancerIngressStatus{},
	}
	for _, lb := range lbs {
		cis.LoadBalancer.Ingress = append(cis.LoadBalancer.Ingress, lb)
	}
	clusterIngressCondSet.Manage(cis).MarkTrue(ClusterIngressConditionLoadBalancerReady)
}

// IsReady looks at the conditions and if the Status has a condition
// ClusterIngressConditionReady returns true if ConditionStatus is True
func (cis *IngressStatus) IsReady() bool {
	return clusterIngressCondSet.Manage(cis).IsHappy()
}
