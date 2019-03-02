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

package v1alpha1

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/knative/pkg/apis"
	"github.com/knative/pkg/kmp"
	networkingv1alpha1 "github.com/knative/serving/pkg/apis/networking/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
)

var (
	reservedPaths = sets.NewString(
		"/",
		"/dev",
		"/dev/log", // Should be a domain socket
		"/tmp",
		"/var",
		"/var/log",
	)
)

// Validate ensures Revision is properly configured.
func (rt *Revision) Validate() *apis.FieldError {
	return ValidateObjectMetadata(rt.GetObjectMeta()).ViaField("metadata").
		Also(rt.Spec.Validate().ViaField("spec"))
}

// Validate ensures RevisionTemplateSpec is properly configured.
func (rt *RevisionTemplateSpec) Validate() *apis.FieldError {
	return rt.Spec.Validate().ViaField("spec")
}

// Validate ensures RevisionSpec is properly configured.
func (rs *RevisionSpec) Validate() *apis.FieldError {
	if equality.Semantic.DeepEqual(rs, &RevisionSpec{}) {
		return apis.ErrMissingField(apis.CurrentField)
	}

	volumes := sets.NewString()
	var errs *apis.FieldError
	for i, volume := range rs.Volumes {
		if volumes.Has(volume.Name) {
			errs = errs.Also((&apis.FieldError{
				Message: fmt.Sprintf("duplicate volume name %q", volume.Name),
				Paths:   []string{"name"},
			}).ViaFieldIndex("volumes", i))
		}
		errs = errs.Also(validateVolume(volume).ViaFieldIndex("volumes", i))
		volumes.Insert(volume.Name)
	}

	errs = errs.Also(validateContainer(rs.Container, volumes).ViaField("container"))
	errs = errs.Also(validateBuildRef(rs.BuildRef).ViaField("buildRef"))

	if err := rs.DeprecatedConcurrencyModel.Validate().ViaField("concurrencyModel"); err != nil {
		errs = errs.Also(err)
	} else {
		errs = errs.Also(ValidateContainerConcurrency(
			rs.ContainerConcurrency, rs.DeprecatedConcurrencyModel))
	}

	return errs.Also(validateTimeoutSeconds(rs.TimeoutSeconds))
}

func validateTimeoutSeconds(timeoutSeconds int64) *apis.FieldError {
	if timeoutSeconds != 0 {
		if timeoutSeconds > int64(networkingv1alpha1.DefaultTimeout.Seconds()) || timeoutSeconds < 0 {
			return apis.ErrOutOfBoundsValue(fmt.Sprintf("%ds", timeoutSeconds), "0s",
				fmt.Sprintf("%ds", int(networkingv1alpha1.DefaultTimeout.Seconds())),
				"timeoutSeconds")
		}
	}
	return nil
}

// Validate ensures RevisionRequestConcurrencyModelType is properly configured.
func (ss DeprecatedRevisionServingStateType) Validate() *apis.FieldError {
	switch ss {
	case DeprecatedRevisionServingStateType(""),
		DeprecatedRevisionServingStateRetired,
		DeprecatedRevisionServingStateReserve,
		DeprecatedRevisionServingStateActive:
		return nil
	default:
		return apis.ErrInvalidValue(string(ss), apis.CurrentField)
	}
}

// Validate ensures RevisionRequestConcurrencyModelType is properly configured.
func (cm RevisionRequestConcurrencyModelType) Validate() *apis.FieldError {
	switch cm {
	case RevisionRequestConcurrencyModelType(""),
		RevisionRequestConcurrencyModelMulti,
		RevisionRequestConcurrencyModelSingle:
		return nil
	default:
		return apis.ErrInvalidValue(string(cm), apis.CurrentField)
	}
}

// ValidateContainerConcurrency ensures ContainerConcurrency is properly configured.
func ValidateContainerConcurrency(cc RevisionContainerConcurrencyType, cm RevisionRequestConcurrencyModelType) *apis.FieldError {
	// Validate ContainerConcurrency alone
	if cc < 0 || cc > RevisionContainerConcurrencyMax {
		return apis.ErrInvalidValue(strconv.Itoa(int(cc)), "containerConcurrency")
	}

	// Validate combinations of ConcurrencyModel and ContainerConcurrency
	if cc == 0 && cm != RevisionRequestConcurrencyModelMulti && cm != RevisionRequestConcurrencyModelType("") {
		return apis.ErrMultipleOneOf("containerConcurrency", "concurrencyModel")
	}
	if cc == 1 && cm != RevisionRequestConcurrencyModelSingle && cm != RevisionRequestConcurrencyModelType("") {
		return apis.ErrMultipleOneOf("containerConcurrency", "concurrencyModel")
	}
	if cc > 1 && cm != RevisionRequestConcurrencyModelType("") {
		return apis.ErrMultipleOneOf("containerConcurrency", "concurrencyModel")
	}

	return nil
}

func validateVolume(volume corev1.Volume) *apis.FieldError {
	var errs *apis.FieldError
	if volume.Name == "" {
		errs = apis.ErrMissingField("name")
	} else if len(validation.IsDNS1123Label(volume.Name)) != 0 {
		errs = apis.ErrInvalidValue(volume.Name, "name")
	}

	vs := volume.VolumeSource
	switch {
	case vs.Secret != nil, vs.ConfigMap != nil:
		// These are fine.
	default:
		errs = errs.Also(apis.ErrMissingOneOf("secret", "configMap"))
	}
	return errs
}

func validateContainer(container corev1.Container, volumes sets.String) *apis.FieldError {
	if equality.Semantic.DeepEqual(container, corev1.Container{}) {
		return apis.ErrMissingField(apis.CurrentField)
	}

	// Check that volume mounts match names in "volumes", that "volumes" has 100%
	// coverage, and the field restrictions.
	seen := sets.NewString()
	var errs *apis.FieldError
	for i, vm := range container.VolumeMounts {
		// This effectively checks that Name is non-empty because Volume name must be non-empty.
		if !volumes.Has(vm.Name) {
			errs = errs.Also((&apis.FieldError{
				Message: "volumeMount has no matching volume",
				Paths:   []string{"name"},
			}).ViaFieldIndex("volumeMounts", i))
		}
		seen.Insert(vm.Name)

		if vm.MountPath == "" {
			errs = errs.Also(apis.ErrMissingField("mountPath").ViaFieldIndex("volumeMounts", i))
		} else if reservedPaths.Has(filepath.Clean(vm.MountPath)) {
			errs = errs.Also((&apis.FieldError{
				Message: fmt.Sprintf("mountPath %q is a reserved path", filepath.Clean(vm.MountPath)),
				Paths:   []string{"mountPath"},
			}).ViaFieldIndex("volumeMounts", i))
		} else if !filepath.IsAbs(vm.MountPath) {
			errs = errs.Also(apis.ErrInvalidValue(vm.MountPath, "mountPath").ViaFieldIndex("volumeMounts", i))
		}
		if !vm.ReadOnly {
			errs = errs.Also(apis.ErrMissingField("readOnly").ViaFieldIndex("volumeMounts", i))
		}

		if vm.SubPath != "" {
			errs = errs.Also(
				apis.ErrDisallowedFields("subPath").ViaFieldIndex("volumeMounts", i))
		}
		if vm.MountPropagation != nil {
			errs = errs.Also(
				apis.ErrDisallowedFields("mountPropagation").ViaFieldIndex("volumeMounts", i))
		}
	}

	if missing := volumes.Difference(seen); missing.Len() > 0 {
		errs = errs.Also(&apis.FieldError{
			Message: fmt.Sprintf("volumes not mounted: %v", missing.List()),
			Paths:   []string{"volumeMounts"},
		})
	}

	// Some corev1.Container fields are set by Knative Serving controller.  We disallow them
	// here to avoid silently overwriting these fields and causing confusions for
	// the users.  See pkg/controller/revision/resources/deploy.go#makePodSpec.
	var ignoredFields []string
	if container.Name != "" {
		ignoredFields = append(ignoredFields, "name")
	}

	if container.Lifecycle != nil {
		ignoredFields = append(ignoredFields, "lifecycle")
	}
	if len(ignoredFields) > 0 {
		// Complain about all ignored fields so that user can remove them all at once.
		errs = errs.Also(apis.ErrDisallowedFields(ignoredFields...))
	}
	if err := validateContainerPorts(container.Ports); err != nil {
		errs = errs.Also(err.ViaField("ports"))
	}
	// Validate our probes
	if err := validateProbe(container.ReadinessProbe).ViaField("readinessProbe"); err != nil {
		errs = errs.Also(err)
	}
	if err := validateProbe(container.LivenessProbe).ViaField("livenessProbe"); err != nil {
		errs = errs.Also(err)
	}
	if _, err := name.ParseReference(container.Image, name.WeakValidation); err != nil {
		fe := &apis.FieldError{
			Message: "Failed to parse image reference",
			Paths:   []string{"image"},
			Details: fmt.Sprintf("image: %q, error: %v", container.Image, err),
		}
		errs = errs.Also(fe)
	}
	return errs
}

// The port is named "user-port" on the deployment, but a user cannot set an arbitrary name on the port
// in Configuration. The name field is reserved for content-negotiation. Currently 'h2c' and 'http1' are
// allowed.
// https://github.com/knative/serving/blob/master/docs/runtime-contract.md#inbound-network-connectivity
var validPortNames = sets.NewString(
	"h2c",
	"http1",
	"",
)

func validateContainerPorts(ports []corev1.ContainerPort) *apis.FieldError {
	if len(ports) == 0 {
		return nil
	}

	var errs *apis.FieldError

	// user can set container port which names "user-port" to define application's port.
	// Queue-proxy will use it to send requests to application
	// if user didn't set any port, it will set default port user-port=8080.
	if len(ports) > 1 {
		errs = errs.Also(&apis.FieldError{
			Message: "More than one container port is set",
			Paths:   []string{apis.CurrentField},
			Details: "Only a single port is allowed",
		})
	}

	userPort := ports[0]
	// Only allow empty (defaulting to "TCP") or explicit TCP for protocol
	if userPort.Protocol != "" && userPort.Protocol != corev1.ProtocolTCP {
		errs = errs.Also(apis.ErrInvalidValue(string(userPort.Protocol), "Protocol"))
	}

	// Don't allow HostIP or HostPort to be set
	var disallowedFields []string
	if userPort.HostIP != "" {
		disallowedFields = append(disallowedFields, "HostIP")

	}
	if userPort.HostPort != 0 {
		disallowedFields = append(disallowedFields, "HostPort")
	}
	if len(disallowedFields) != 0 {
		errs = errs.Also(apis.ErrDisallowedFields(disallowedFields...))
	}

	// Don't allow userPort to conflict with QueueProxy sidecar
	if userPort.ContainerPort == RequestQueuePort ||
		userPort.ContainerPort == RequestQueueAdminPort ||
		userPort.ContainerPort == RequestQueueMetricsPort {
		errs = errs.Also(apis.ErrInvalidValue(strconv.Itoa(int(userPort.ContainerPort)), "ContainerPort"))
	}

	if userPort.ContainerPort < 1 || userPort.ContainerPort > 65535 {
		errs = errs.Also(apis.ErrOutOfBoundsValue(strconv.Itoa(int(userPort.ContainerPort)), "1", "65535", "ContainerPort"))
	}

	if !validPortNames.Has(userPort.Name) {
		errs = errs.Also(&apis.FieldError{
			Message: fmt.Sprintf("Port name %v is not allowed", ports[0].Name),
			Paths:   []string{apis.CurrentField},
			Details: "Name must be empty, or one of: 'h2c', 'http1'",
		})
	}

	return errs
}

func validateBuildRef(buildRef *corev1.ObjectReference) *apis.FieldError {
	if buildRef == nil {
		return nil
	}
	if len(validation.IsQualifiedName(buildRef.APIVersion)) != 0 {
		return apis.ErrInvalidValue(buildRef.APIVersion, "apiVersion")
	}
	if len(validation.IsCIdentifier(buildRef.Kind)) != 0 {
		return apis.ErrInvalidValue(buildRef.Kind, "kind")
	}
	if len(validation.IsDNS1123Label(buildRef.Name)) != 0 {
		return apis.ErrInvalidValue(buildRef.Name, "name")
	}
	var disallowedFields []string
	if buildRef.Namespace != "" {
		disallowedFields = append(disallowedFields, "namespace")
	}
	if buildRef.FieldPath != "" {
		disallowedFields = append(disallowedFields, "fieldPath")
	}
	if buildRef.ResourceVersion != "" {
		disallowedFields = append(disallowedFields, "resourceVersion")
	}
	if buildRef.UID != "" {
		disallowedFields = append(disallowedFields, "uid")
	}
	if len(disallowedFields) != 0 {
		return apis.ErrDisallowedFields(disallowedFields...)
	}
	return nil
}

func validateProbe(p *corev1.Probe) *apis.FieldError {
	if p == nil {
		return nil
	}
	emptyPort := intstr.IntOrString{}
	switch {
	case p.Handler.HTTPGet != nil:
		if p.Handler.HTTPGet.Port != emptyPort {
			return apis.ErrDisallowedFields("httpGet.port")
		}
	case p.Handler.TCPSocket != nil:
		if p.Handler.TCPSocket.Port != emptyPort {
			return apis.ErrDisallowedFields("tcpSocket.port")
		}
	}
	return nil
}

// CheckImmutableFields checks the immutable fields are not modified.
func (current *Revision) CheckImmutableFields(og apis.Immutable) *apis.FieldError {
	original, ok := og.(*Revision)
	if !ok {
		return &apis.FieldError{Message: "The provided original was not a Revision"}
	}

	if diff, err := kmp.SafeDiff(original.Spec, current.Spec); err != nil {
		return &apis.FieldError{
			Message: "Failed to diff Revision",
			Paths:   []string{"spec"},
			Details: err.Error(),
		}
	} else if diff != "" {
		return &apis.FieldError{
			Message: "Immutable fields changed (-old +new)",
			Paths:   []string{"spec"},
			Details: diff,
		}
	}

	return nil
}
