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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/knative/pkg/apis"
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	"github.com/knative/pkg/kmeta"
	"github.com/knative/serving/pkg/apis/networking/v1alpha1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Route is responsible for configuring ingress over a collection of Revisions.
// Some of the Revisions a Route distributes traffic over may be specified by
// referencing the Configuration responsible for creating them; in these cases
// the Route is additionally responsible for monitoring the Configuration for
// "latest ready" revision changes, and smoothly rolling out latest revisions.
// See also: https://github.com/knative/serving/blob/master/docs/spec/overview.md#route
type Route struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec holds the desired state of the Route (from the client).
	// +optional
	Spec RouteSpec `json:"spec,omitempty"`

	// Status communicates the observed state of the Route (from the controller).
	// +optional
	Status RouteStatus `json:"status,omitempty"`
}

// Check that Route may be validated and defaulted.
var _ apis.Validatable = (*Route)(nil)
var _ apis.Defaultable = (*Route)(nil)

// Check that we can create OwnerReferences to a Route.
var _ kmeta.OwnerRefable = (*Route)(nil)

// Check that RouteStatus may have its conditions managed.
var _ duckv1alpha1.ConditionsAccessor = (*RouteStatus)(nil)

// TrafficTarget holds a single entry of the routing table for a Route.
type TrafficTarget struct {
	// Name is optionally used to expose a dedicated hostname for referencing this
	// target exclusively. It has the form: {name}.${route.status.domain}
	// +optional
	Name string `json:"name,omitempty"`

	// RevisionName of a specific revision to which to send this portion of traffic.
	// This is mutually exclusive with ConfigurationName.
	// +optional
	RevisionName string `json:"revisionName,omitempty"`

	// ConfigurationName of a configuration to whose latest revision we will send
	// this portion of traffic. When the "status.latestReadyRevisionName" of the
	// referenced configuration changes, we will automatically migrate traffic
	// from the prior "latest ready" revision to the new one.
	// This field is never set in Route's status, only its spec.
	// This is mutually exclusive with RevisionName.
	// +optional
	ConfigurationName string `json:"configurationName,omitempty"`

	// Percent specifies percent of the traffic to this Revision or Configuration.
	// This defaults to zero if unspecified.
	Percent int `json:"percent"`
}

// RouteSpec holds the desired state of the Route (from the client).
type RouteSpec struct {
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

	// Traffic specifies how to distribute traffic over a collection of Knative Serving Revisions and Configurations.
	// +optional
	Traffic []TrafficTarget `json:"traffic,omitempty"`
}

const (
	// RouteConditionReady is set when the service is configured
	// and has available backends ready to receive traffic.
	RouteConditionReady = duckv1alpha1.ConditionReady

	// RouteConditionAllTrafficAssigned is set to False when the
	// service is not configured properly or has no available
	// backends ready to receive traffic.
	RouteConditionAllTrafficAssigned duckv1alpha1.ConditionType = "AllTrafficAssigned"

	// RouteConditionIngressReady is set to False when the
	// ClusterIngress fails to become Ready.
	RouteConditionIngressReady duckv1alpha1.ConditionType = "IngressReady"
)

var routeCondSet = duckv1alpha1.NewLivingConditionSet(RouteConditionAllTrafficAssigned, RouteConditionIngressReady)

// RouteStatus communicates the observed state of the Route (from the controller).
type RouteStatus struct {
	// Domain holds the top-level domain that will distribute traffic over the provided targets.
	// It generally has the form {route-name}.{route-namespace}.{cluster-level-suffix}
	// +optional
	Domain string `json:"domain,omitempty"`

	// DeprecatedDomainInternal holds the top-level domain that will distribute traffic over the provided
	// targets from inside the cluster. It generally has the form
	// {route-name}.{route-namespace}.svc.{cluster-domain-name}
	// DEPRECATED: Use Address instead.
	// +optional
	DeprecatedDomainInternal string `json:"domainInternal,omitempty"`

	// Address holds the information needed for a Route to be the target of an event.
	// +optional
	Address *duckv1alpha1.Addressable `json:"address,omitempty"`

	// Traffic holds the configured traffic distribution.
	// These entries will always contain RevisionName references.
	// When ConfigurationName appears in the spec, this will hold the
	// LatestReadyRevisionName that we last observed.
	// +optional
	Traffic []TrafficTarget `json:"traffic,omitempty"`

	// Conditions communicates information about ongoing/complete
	// reconciliation processes that bring the "spec" inline with the observed
	// state of the world.
	// +optional
	Conditions duckv1alpha1.Conditions `json:"conditions,omitempty"`

	// ObservedGeneration is the 'Generation' of the Configuration that
	// was last processed by the controller. The observed generation is updated
	// even if the controller failed to process the spec and create the Revision.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RouteList is a list of Route resources
type RouteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Route `json:"items"`
}

func (r *Route) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("Route")
}

func (rs *RouteStatus) IsReady() bool {
	return routeCondSet.Manage(rs).IsHappy()
}

func (rs *RouteStatus) GetCondition(t duckv1alpha1.ConditionType) *duckv1alpha1.Condition {
	return routeCondSet.Manage(rs).GetCondition(t)
}

func (rs *RouteStatus) InitializeConditions() {
	routeCondSet.Manage(rs).InitializeConditions()
}

// MarkServiceNotOwned changes the IngressReady status to be false with the reason being that
// there is a pre-existing placeholder service with the name we wanted to use.
func (rs *RouteStatus) MarkServiceNotOwned(name string) {
	routeCondSet.Manage(rs).MarkFalse(RouteConditionIngressReady, "NotOwned",
		fmt.Sprintf("There is an existing placeholder Service %q that we do not own.", name))
}

func (rs *RouteStatus) MarkTrafficAssigned() {
	routeCondSet.Manage(rs).MarkTrue(RouteConditionAllTrafficAssigned)
}

func (rs *RouteStatus) MarkUnknownTrafficError(msg string) {
	routeCondSet.Manage(rs).MarkUnknown(RouteConditionAllTrafficAssigned, "Unknown", msg)
}

func (rs *RouteStatus) MarkConfigurationNotReady(name string) {
	routeCondSet.Manage(rs).MarkUnknown(RouteConditionAllTrafficAssigned,
		"RevisionMissing",
		"Configuration %q is waiting for a Revision to become ready.", name)
}

func (rs *RouteStatus) MarkConfigurationFailed(name string) {
	routeCondSet.Manage(rs).MarkFalse(RouteConditionAllTrafficAssigned,
		"RevisionMissing",
		"Configuration %q does not have any ready Revision.", name)
}

func (rs *RouteStatus) MarkRevisionNotReady(name string) {
	routeCondSet.Manage(rs).MarkUnknown(RouteConditionAllTrafficAssigned,
		"RevisionMissing",
		"Revision %q is not yet ready.", name)
}

func (rs *RouteStatus) MarkRevisionFailed(name string) {
	routeCondSet.Manage(rs).MarkFalse(RouteConditionAllTrafficAssigned,
		"RevisionMissing",
		"Revision %q failed to become ready.", name)
}

func (rs *RouteStatus) MarkMissingTrafficTarget(kind, name string) {
	routeCondSet.Manage(rs).MarkFalse(RouteConditionAllTrafficAssigned,
		kind+"Missing",
		"%s %q referenced in traffic not found.", kind, name)
}

// PropagateClusterIngressStatus update RouteConditionIngressReady condition
// in RouteStatus according to IngressStatus.
func (rs *RouteStatus) PropagateClusterIngressStatus(cs v1alpha1.IngressStatus) {
	cc := cs.GetCondition(v1alpha1.ClusterIngressConditionReady)
	if cc == nil {
		return
	}
	switch {
	case cc.Status == corev1.ConditionUnknown:
		routeCondSet.Manage(rs).MarkUnknown(RouteConditionIngressReady, cc.Reason, cc.Message)
	case cc.Status == corev1.ConditionTrue:
		routeCondSet.Manage(rs).MarkTrue(RouteConditionIngressReady)
	case cc.Status == corev1.ConditionFalse:
		routeCondSet.Manage(rs).MarkFalse(RouteConditionIngressReady, cc.Reason, cc.Message)
	}
}

// GetConditions returns the Conditions array. This enables generic handling of
// conditions by implementing the duckv1alpha1.Conditions interface.
func (rs *RouteStatus) GetConditions() duckv1alpha1.Conditions {
	return rs.Conditions
}

// SetConditions sets the Conditions array. This enables generic handling of
// conditions by implementing the duckv1alpha1.Conditions interface.
func (rs *RouteStatus) SetConditions(conditions duckv1alpha1.Conditions) {
	rs.Conditions = conditions
}
