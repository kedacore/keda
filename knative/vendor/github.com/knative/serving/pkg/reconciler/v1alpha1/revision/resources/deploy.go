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

package resources

import (
	"strconv"

	"github.com/knative/pkg/kmeta"
	"github.com/knative/pkg/logging"
	"github.com/knative/serving/pkg/apis/serving/v1alpha1"
	"github.com/knative/serving/pkg/autoscaler"
	"github.com/knative/serving/pkg/network"
	"github.com/knative/serving/pkg/queue"
	"github.com/knative/serving/pkg/reconciler/v1alpha1/revision/config"
	"github.com/knative/serving/pkg/reconciler/v1alpha1/revision/resources/names"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const varLogVolumeName = "varlog"

var (
	varLogVolume = corev1.Volume{
		Name: varLogVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}

	varLogVolumeMount = corev1.VolumeMount{
		Name:      varLogVolumeName,
		MountPath: "/var/log",
	}

	userResources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU: userContainerCPU,
		},
	}

	// This PreStop hook is actually calling an endpoint on the queue-proxy
	// because of the way PreStop hooks are called by kubelet. We use this
	// to block the user-container from exiting before the queue-proxy is ready
	// to exit so we can guarantee that there are no more requests in flight.
	userLifecycle = &corev1.Lifecycle{
		PreStop: &corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Port: intstr.FromInt(v1alpha1.RequestQueueAdminPort),
				Path: queue.RequestQueueQuitPath,
			},
		},
	}
)

func rewriteUserProbe(p *corev1.Probe, userPort int) {
	if p == nil {
		return
	}
	switch {
	case p.HTTPGet != nil:
		// For HTTP probes, we route them through the queue container
		// so that we know the queue proxy is ready/live as well.
		p.HTTPGet.Port = intstr.FromInt(v1alpha1.RequestQueuePort)
	case p.TCPSocket != nil:
		p.TCPSocket.Port = intstr.FromInt(userPort)
	}
}

// applyDefaultResource
// Implements a deep merge for ResourceRequirements
// note: DeepCopyInto cannot be used because it replaces limits or requests instead of merging them
func applyDefaultResources(defaults corev1.ResourceRequirements, out *corev1.ResourceRequirements) {
	in := defaults.DeepCopy()
	if in.Limits != nil {
		in, out := &in.Limits, &out.Limits
		for key, val := range *out {
			(*in)[key] = val.DeepCopy()
		}
		(*out) = (*in)
	}
	if in.Requests != nil {
		in, out := &in.Requests, &out.Requests
		for key, val := range *out {
			(*in)[key] = val.DeepCopy()
		}
		(*out) = (*in)
	}
}

func makePodSpec(rev *v1alpha1.Revision, loggingConfig *logging.Config, observabilityConfig *config.Observability, autoscalerConfig *autoscaler.Config, controllerConfig *config.Controller) *corev1.PodSpec {
	userContainer := rev.Spec.Container.DeepCopy()
	// Adding or removing an overwritten corev1.Container field here? Don't forget to
	// update the validations in pkg/webhook.validateContainer.
	userContainer.Name = UserContainerName

	// If client provides for some resources, override default values
	applyDefaultResources(userResources, &userContainer.Resources)

	userContainer.VolumeMounts = append(userContainer.VolumeMounts, varLogVolumeMount)
	userContainer.Lifecycle = userLifecycle
	userPort := getUserPort(rev)
	userPortInt := int(userPort)
	userPortStr := strconv.Itoa(userPortInt)
	// Replacement is safe as only up to a single port is allowed on the Revision
	userContainer.Ports = buildContainerPorts(userPort)
	userContainer.Env = append(userContainer.Env, buildUserPortEnv(userPortStr))
	userContainer.Env = append(userContainer.Env, getKnativeEnvVar(rev)...)

	// Prefer imageDigest from revision if available
	if rev.Status.ImageDigest != "" {
		userContainer.Image = rev.Status.ImageDigest
	}

	if userContainer.TerminationMessagePolicy == "" {
		userContainer.TerminationMessagePolicy = corev1.TerminationMessageFallbackToLogsOnError
	}

	// If the client provides probes, we should fill in the port for them.
	rewriteUserProbe(userContainer.ReadinessProbe, userPortInt)
	rewriteUserProbe(userContainer.LivenessProbe, userPortInt)

	revisionTimeout := rev.Spec.TimeoutSeconds

	podSpec := &corev1.PodSpec{
		Containers: []corev1.Container{
			*userContainer,
			*makeQueueContainer(rev, loggingConfig, autoscalerConfig, controllerConfig),
		},
		Volumes:                       append([]corev1.Volume{varLogVolume}, rev.Spec.Volumes...),
		ServiceAccountName:            rev.Spec.ServiceAccountName,
		TerminationGracePeriodSeconds: &revisionTimeout,
	}

	// Add Fluentd sidecar and its config map volume if var log collection is enabled.
	if observabilityConfig.EnableVarLogCollection {
		podSpec.Containers = append(podSpec.Containers, *makeFluentdContainer(rev, observabilityConfig))
		podSpec.Volumes = append(podSpec.Volumes, *makeFluentdConfigMapVolume(rev))
	}

	return podSpec
}

func getUserPort(rev *v1alpha1.Revision) int32 {
	if len(rev.Spec.Container.Ports) == 1 {
		return rev.Spec.Container.Ports[0].ContainerPort
	}

	//TODO(#2258): Use container EXPOSE metadata from image before falling back to default value

	return v1alpha1.DefaultUserPort
}

func buildContainerPorts(userPort int32) []corev1.ContainerPort {
	return []corev1.ContainerPort{{
		Name:          v1alpha1.UserPortName,
		ContainerPort: userPort,
	}}
}

func buildUserPortEnv(userPort string) corev1.EnvVar {
	return corev1.EnvVar{
		Name:  userPortEnvName,
		Value: userPort,
	}
}

func MakeDeployment(rev *v1alpha1.Revision,
	loggingConfig *logging.Config, networkConfig *network.Config, observabilityConfig *config.Observability,
	autoscalerConfig *autoscaler.Config, controllerConfig *config.Controller) *appsv1.Deployment {

	podTemplateAnnotations := makeAnnotations(rev)
	// TODO(nghia): Remove the need for this
	podTemplateAnnotations[sidecarIstioInjectAnnotation] = "true"
	// TODO(mattmoor): Once we have a mechanism for decorating arbitrary deployments (and opting
	// out via annotation) we should explicitly disable that here to avoid redundant Image
	// resources.

	// Inject the IP ranges for istio sidecar configuration.
	// We will inject this value only if all of the following are true:
	// - the config map contains a non-empty value
	// - the user doesn't specify this annotation in configuration's pod template
	// - configured values are valid CIDR notation IP addresses
	// If these conditions are not met, this value will be left untouched.
	// * is a special value that is accepted as a valid.
	// * intercepts calls to all IPs: in cluster as well as outside the cluster.
	if _, ok := podTemplateAnnotations[IstioOutboundIPRangeAnnotation]; !ok {
		if len(networkConfig.IstioOutboundIPRanges) > 0 {
			podTemplateAnnotations[IstioOutboundIPRangeAnnotation] = networkConfig.IstioOutboundIPRanges
		}
	}

	one := int32(1)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:            names.Deployment(rev),
			Namespace:       rev.Namespace,
			Labels:          makeLabels(rev),
			Annotations:     makeAnnotations(rev),
			OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(rev)},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas:                &one,
			Selector:                makeSelector(rev),
			ProgressDeadlineSeconds: &ProgressDeadlineSeconds,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      makeLabels(rev),
					Annotations: podTemplateAnnotations,
				},
				Spec: *makePodSpec(rev, loggingConfig, observabilityConfig, autoscalerConfig, controllerConfig),
			},
		},
	}
}
