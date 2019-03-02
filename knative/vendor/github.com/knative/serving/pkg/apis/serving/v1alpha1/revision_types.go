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
	"strconv"
	"time"

	"github.com/knative/pkg/apis"
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	"github.com/knative/pkg/kmeta"
	"github.com/knative/serving/pkg/apis/serving"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Revision is an immutable snapshot of code and configuration.  A revision
// references a container image, and optionally a build that is responsible for
// materializing that container image from source. Revisions are created by
// updates to a Configuration.
//
// See also: https://github.com/knative/serving/blob/master/docs/spec/overview.md#revision
type Revision struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec holds the desired state of the Revision (from the client).
	// +optional
	Spec RevisionSpec `json:"spec,omitempty"`

	// Status communicates the observed state of the Revision (from the controller).
	// +optional
	Status RevisionStatus `json:"status,omitempty"`
}

// Check that Revision can be validated, can be defaulted, and has immutable fields.
var _ apis.Validatable = (*Revision)(nil)
var _ apis.Defaultable = (*Revision)(nil)
var _ apis.Immutable = (*Revision)(nil)

// Check that RevisionStatus may have its conditions managed.
var _ duckv1alpha1.ConditionsAccessor = (*RevisionStatus)(nil)

// Check that we can create OwnerReferences to a Revision.
var _ kmeta.OwnerRefable = (*Revision)(nil)

// RevisionTemplateSpec describes the data a revision should have when created from a template.
// Based on: https://github.com/kubernetes/api/blob/e771f807/core/v1/types.go#L3179-L3190
type RevisionTemplateSpec struct {
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +optional
	Spec RevisionSpec `json:"spec,omitempty"`
}

// DeprecatedRevisionServingStateType is an enumeration of the levels of serving readiness of the Revision.
// See also: https://github.com/knative/serving/blob/master/docs/spec/errors.md#error-conditions-and-reporting
type DeprecatedRevisionServingStateType string

const (
	// The revision is ready to serve traffic. It should have Kubernetes
	// resources, and the Istio route should be pointed to the given resources.
	DeprecatedRevisionServingStateActive DeprecatedRevisionServingStateType = "Active"
	// The revision is not currently serving traffic, but could be made to serve
	// traffic quickly. It should have Kubernetes resources, but the Istio route
	// should be pointed to the activator.
	DeprecatedRevisionServingStateReserve DeprecatedRevisionServingStateType = "Reserve"
	// The revision has been decommissioned and is not needed to serve traffic
	// anymore. It should not have any Istio routes or Kubernetes resources.
	// A Revision may be brought out of retirement, but it may take longer than
	// it would from a "Reserve" state.
	// Note: currently not set anywhere. See https://github.com/knative/serving/issues/1203
	DeprecatedRevisionServingStateRetired DeprecatedRevisionServingStateType = "Retired"
)

// RevisionRequestConcurrencyModelType is an enumeration of the
// concurrency models supported by a Revision.
// Deprecated in favor of RevisionContainerConcurrencyType.
type RevisionRequestConcurrencyModelType string

const (
	// RevisionRequestConcurrencyModelSingle guarantees that only one
	// request will be handled at a time (concurrently) per instance
	// of Revision Container.
	RevisionRequestConcurrencyModelSingle RevisionRequestConcurrencyModelType = "Single"
	// RevisionRequestConcurencyModelMulti allows more than one request to
	// be handled at a time (concurrently) per instance of Revision
	// Container.
	RevisionRequestConcurrencyModelMulti RevisionRequestConcurrencyModelType = "Multi"
)

// RevisionContainerConcurrencyType is an integer expressing a number of
// in-flight (concurrent) requests.
type RevisionContainerConcurrencyType int64

const (
	// The maximum configurable container concurrency.
	RevisionContainerConcurrencyMax RevisionContainerConcurrencyType = 1000
)

// RevisionProtocolType is an enumeration of the supported application-layer protocols
// See also: https://github.com/knative/serving/blob/master/docs/runtime-contract.md#protocols-and-ports
type RevisionProtocolType string

const (
	// HTTP/1.1
	RevisionProtocolHTTP1 RevisionProtocolType = "http1"
	// HTTP/2 with Prior Knowledge
	RevisionProtocolH2C RevisionProtocolType = "h2c"
)

const (
	// UserPortName is the name that will be used for the Port on the
	// Deployment and Pod created by a Revision. This name will be set regardless of if
	// a user specifies a port or the default value is chosen.
	UserPortName = "user-port"

	// DefaultUserPort is the default port value the QueueProxy will
	// use for connecting to the user container.
	DefaultUserPort = 8080

	// RequestQueuePortName specifies the port name to use for http requests
	// in queue-proxy container.
	RequestQueuePortName string = "queue-port"

	// RequestQueuePort specifies the port number to use for http requests
	// in queue-proxy container.
	RequestQueuePort = 8012

	// RequestQueueAdminPortName specifies the port name for
	// health check and lifecyle hooks for queue-proxy.
	RequestQueueAdminPortName string = "queueadm-port"

	// RequestQueueAdminPort specifies the port number for
	// health check and lifecyle hooks for queue-proxy.
	RequestQueueAdminPort = 8022

	// RequestQueueMetricsPort specifies the port number for metrics emitted
	// by queue-proxy.
	RequestQueueMetricsPort = 9090

	// RequestQueueMetricsPortName specifies the port name to use for metrics
	// emitted by queue-proxy.
	RequestQueueMetricsPortName = "queue-metrics"
)

// RevisionSpec holds the desired state of the Revision (from the client).
type RevisionSpec struct {
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

	// DeprecatedServingState holds a value describing the desired state the Kubernetes
	// resources should be in for this Revision.
	// Users must not specify this when creating a revision. These values are no longer
	// updated by the system.
	// +optional
	DeprecatedServingState DeprecatedRevisionServingStateType `json:"servingState,omitempty"`

	// DeprecatedConcurrencyModel specifies the desired concurrency model
	// (Single or Multi) for the
	// Revision. Defaults to Multi.
	// Deprecated in favor of ContainerConcurrency.
	// +optional
	DeprecatedConcurrencyModel RevisionRequestConcurrencyModelType `json:"concurrencyModel,omitempty"`

	// ContainerConcurrency specifies the maximum allowed
	// in-flight (concurrent) requests per container of the Revision.
	// Defaults to `0` which means unlimited concurrency.
	// This field replaces ConcurrencyModel. A value of `1`
	// is equivalent to `Single` and `0` is equivalent to `Multi`.
	// +optional
	ContainerConcurrency RevisionContainerConcurrencyType `json:"containerConcurrency,omitempty"`

	// ServiceAccountName holds the name of the Kubernetes service account
	// as which the underlying K8s resources should be run. If unspecified
	// this will default to the "default" service account for the namespace
	// in which the Revision exists.
	// This may be used to provide access to private container images by
	// following: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#add-imagepullsecrets-to-a-service-account
	// TODO(ZhiminXiang): verify the corresponding service account exists.
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// DeprecatedBuildName optionally holds the name of the Build responsible for
	// producing the container image for its Revision.
	// DEPRECATED: Use BuildRef instead.
	// +optional
	DeprecatedBuildName string `json:"buildName,omitempty"`

	// BuildRef holds the reference to the build (if there is one) responsible
	// for producing the container image for this Revision. Otherwise, nil
	// +optional
	BuildRef *corev1.ObjectReference `json:"buildRef,omitempty"`

	// Container defines the unit of execution for this Revision.
	// In the context of a Revision, we disallow a number of the fields of
	// this Container, including: name and lifecycle.
	// See also the runtime contract for more information about the execution
	// environment:
	// https://github.com/knative/serving/blob/master/docs/runtime-contract.md
	// +optional
	Container corev1.Container `json:"container,omitempty"`

	// Volumes defines a set of Kubernetes volumes to be mounted into the
	// specified Container.  Currently only ConfigMap and Secret volumes are
	// supported.
	Volumes []corev1.Volume `json:"volumes,omitempty"`

	// TimeoutSeconds holds the max duration the instance is allowed for responding to a request.
	// +optional
	TimeoutSeconds int64 `json:"timeoutSeconds,omitempty"`
}

const (
	// RevisionConditionReady is set when the revision is starting to materialize
	// runtime resources, and becomes true when those resources are ready.
	RevisionConditionReady = duckv1alpha1.ConditionReady
	// RevisionConditionBuildSucceeded is set when the revision has an associated build
	// and is marked True if/once the Build has completed successfully.
	RevisionConditionBuildSucceeded duckv1alpha1.ConditionType = "BuildSucceeded"
	// RevisionConditionResourcesAvailable is set when underlying
	// Kubernetes resources have been provisioned.
	RevisionConditionResourcesAvailable duckv1alpha1.ConditionType = "ResourcesAvailable"
	// RevisionConditionContainerHealthy is set when the revision readiness check completes.
	RevisionConditionContainerHealthy duckv1alpha1.ConditionType = "ContainerHealthy"
	// RevisionConditionActive is set when the revision is receiving traffic.
	RevisionConditionActive duckv1alpha1.ConditionType = "Active"
)

var revCondSet = duckv1alpha1.NewLivingConditionSet(
	RevisionConditionResourcesAvailable,
	RevisionConditionContainerHealthy,
	RevisionConditionBuildSucceeded,
)

var buildCondSet = duckv1alpha1.NewBatchConditionSet()

// RevisionStatus communicates the observed state of the Revision (from the controller).
type RevisionStatus struct {
	// ServiceName holds the name of a core Kubernetes Service resource that
	// load balances over the pods backing this Revision. When the Revision
	// is Active, this service would be an appropriate ingress target for
	// targeting the revision.
	// +optional
	ServiceName string `json:"serviceName,omitempty"`

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

	// LogURL specifies the generated logging url for this particular revision
	// based on the revision url template specified in the controller's config.
	// +optional
	LogURL string `json:"logUrl,omitempty"`

	// ImageDigest holds the resolved digest for the image specified
	// within .Spec.Container.Image. The digest is resolved during the creation
	// of Revision. This field holds the digest value regardless of whether
	// a tag or digest was originally specified in the Container object. It
	// may be empty if the image comes from a registry listed to skip resolution.
	// +optional
	ImageDigest string `json:"imageDigest,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RevisionList is a list of Revision resources
type RevisionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Revision `json:"items"`
}

func (r *Revision) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("Revision")
}

func (r *Revision) BuildRef() *corev1.ObjectReference {
	if r.Spec.BuildRef != nil {
		buildRef := r.Spec.BuildRef.DeepCopy()
		if buildRef.Namespace == "" {
			buildRef.Namespace = r.Namespace
		}
		return buildRef
	}

	if r.Spec.DeprecatedBuildName != "" {
		return &corev1.ObjectReference{
			APIVersion: "build.knative.dev/v1alpha1",
			Kind:       "Build",
			Namespace:  r.Namespace,
			Name:       r.Spec.DeprecatedBuildName,
		}
	}

	return nil
}

func (r *Revision) GetProtocol() RevisionProtocolType {
	ports := r.Spec.Container.Ports
	if len(ports) > 0 && ports[0].Name == "h2c" {
		return RevisionProtocolH2C
	}

	return RevisionProtocolHTTP1
}

// IsReady looks at the conditions and if the Status has a condition
// RevisionConditionReady returns true if ConditionStatus is True
func (rs *RevisionStatus) IsReady() bool {
	return revCondSet.Manage(rs).IsHappy()
}

func (rs *RevisionStatus) IsActivationRequired() bool {
	if c := revCondSet.Manage(rs).GetCondition(RevisionConditionActive); c != nil {
		return c.Status != corev1.ConditionTrue
	}
	return false
}

func (rs *RevisionStatus) GetCondition(t duckv1alpha1.ConditionType) *duckv1alpha1.Condition {
	return revCondSet.Manage(rs).GetCondition(t)
}

func (rs *RevisionStatus) InitializeConditions() {
	revCondSet.Manage(rs).InitializeConditions()
}

func (rs *RevisionStatus) PropagateBuildStatus(bs duckv1alpha1.KResourceStatus) {
	bc := buildCondSet.Manage(&bs).GetCondition(duckv1alpha1.ConditionSucceeded)
	if bc == nil {
		return
	}
	switch {
	case bc.Status == corev1.ConditionUnknown:
		revCondSet.Manage(rs).MarkUnknown(RevisionConditionBuildSucceeded, "Building", bc.Message)
	case bc.Status == corev1.ConditionTrue:
		revCondSet.Manage(rs).MarkTrue(RevisionConditionBuildSucceeded)
	case bc.Status == corev1.ConditionFalse:
		revCondSet.Manage(rs).MarkFalse(RevisionConditionBuildSucceeded, bc.Reason, bc.Message)
	}
}

// MarkResourceNotOwned changes the "ResourcesAvailable" condition to false to reflect that the
// resource of the given kind and name has already been created, and we do not own it.
func (rs *RevisionStatus) MarkResourceNotOwned(kind, name string) {
	revCondSet.Manage(rs).MarkFalse(RevisionConditionResourcesAvailable, "NotOwned",
		fmt.Sprintf("There is an existing %s %q that we do not own.", kind, name))
}

func (rs *RevisionStatus) MarkDeploying(reason string) {
	revCondSet.Manage(rs).MarkUnknown(RevisionConditionResourcesAvailable, reason, "")
	revCondSet.Manage(rs).MarkUnknown(RevisionConditionContainerHealthy, reason, "")
}

func (rs *RevisionStatus) MarkServiceTimeout() {
	revCondSet.Manage(rs).MarkFalse(RevisionConditionResourcesAvailable, "ServiceTimeout",
		"Timed out waiting for a service endpoint to become ready")
}

func (rs *RevisionStatus) MarkProgressDeadlineExceeded(message string) {
	revCondSet.Manage(rs).MarkFalse(RevisionConditionResourcesAvailable, "ProgressDeadlineExceeded", message)
}

func (rs *RevisionStatus) MarkContainerHealthy() {
	revCondSet.Manage(rs).MarkTrue(RevisionConditionContainerHealthy)
}

func (rs *RevisionStatus) MarkContainerExiting(exitCode int32, message string) {
	exitCodeString := fmt.Sprintf("ExitCode%d", exitCode)
	revCondSet.Manage(rs).MarkFalse(RevisionConditionContainerHealthy, exitCodeString, RevisionContainerExitingMessage(message))
}

func (rs *RevisionStatus) MarkResourcesAvailable() {
	revCondSet.Manage(rs).MarkTrue(RevisionConditionResourcesAvailable)
}

func (rs *RevisionStatus) MarkActive() {
	revCondSet.Manage(rs).MarkTrue(RevisionConditionActive)
}

func (rs *RevisionStatus) MarkActivating(reason, message string) {
	revCondSet.Manage(rs).MarkUnknown(RevisionConditionActive, reason, message)
}

func (rs *RevisionStatus) MarkInactive(reason, message string) {
	revCondSet.Manage(rs).MarkFalse(RevisionConditionActive, reason, message)
}

func (rs *RevisionStatus) MarkContainerMissing(message string) {
	revCondSet.Manage(rs).MarkFalse(RevisionConditionContainerHealthy, "ContainerMissing", message)
}

// GetConditions returns the Conditions array. This enables generic handling of
// conditions by implementing the duckv1alpha1.Conditions interface.
func (rs *RevisionStatus) GetConditions() duckv1alpha1.Conditions {
	return rs.Conditions
}

// SetConditions sets the Conditions array. This enables generic handling of
// conditions by implementing the duckv1alpha1.Conditions interface.
func (rs *RevisionStatus) SetConditions(conditions duckv1alpha1.Conditions) {
	rs.Conditions = conditions
}

// RevisionContainerMissingMessage constructs the status message if a given image
// cannot be pulled correctly.
func RevisionContainerMissingMessage(image string, message string) string {
	return fmt.Sprintf("Unable to fetch image %q: %s", image, message)
}

// RevisionContainerExitingMessage constructs the status message if a container
// fails to come up.
func RevisionContainerExitingMessage(message string) string {
	return fmt.Sprintf("Container failed with: %s", message)
}

const (
	AnnotationParseErrorTypeMissing = "Missing"
	AnnotationParseErrorTypeInvalid = "Invalid"
	LabelParserErrorTypeMissing     = "Missing"
	LabelParserErrorTypeInvalid     = "Invalid"
)

// +k8s:deepcopy-gen=false
type AnnotationParseError struct {
	Type  string
	Value string
	Err   error
}

// +k8s:deepcopy-gen=false
type LastPinnedParseError AnnotationParseError

func (e LastPinnedParseError) Error() string {
	return fmt.Sprintf("%v lastPinned value: %q", e.Type, e.Value)
}

// +k8s:deepcopy-gen=false
type configurationGenerationParseError AnnotationParseError

func (e configurationGenerationParseError) Error() string {
	return fmt.Sprintf("%v configurationGeneration value: %q", e.Type, e.Value)
}

func RevisionLastPinnedString(t time.Time) string {
	return fmt.Sprintf("%d", t.Unix())
}

func (r *Revision) SetLastPinned(t time.Time) {
	if r.ObjectMeta.Annotations == nil {
		r.ObjectMeta.Annotations = make(map[string]string)
	}

	r.ObjectMeta.Annotations[serving.RevisionLastPinnedAnnotationKey] = RevisionLastPinnedString(t)
}

func (r *Revision) GetLastPinned() (time.Time, error) {
	if r.Annotations == nil {
		return time.Time{}, LastPinnedParseError{
			Type: AnnotationParseErrorTypeMissing,
		}
	}

	str, ok := r.ObjectMeta.Annotations[serving.RevisionLastPinnedAnnotationKey]
	if !ok {
		// If a revision is past the create delay without an annotation it is stale
		return time.Time{}, LastPinnedParseError{
			Type: AnnotationParseErrorTypeMissing,
		}
	}

	secs, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return time.Time{}, LastPinnedParseError{
			Type:  AnnotationParseErrorTypeInvalid,
			Value: str,
			Err:   err,
		}
	}

	return time.Unix(secs, 0), nil
}
