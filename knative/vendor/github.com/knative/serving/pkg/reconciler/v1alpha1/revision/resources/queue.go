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

	"github.com/knative/pkg/logging"
	"github.com/knative/pkg/system"
	"github.com/knative/serving/pkg/apis/serving/v1alpha1"
	"github.com/knative/serving/pkg/autoscaler"
	"github.com/knative/serving/pkg/queue"
	"github.com/knative/serving/pkg/reconciler/v1alpha1/revision/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	queueResources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceName("cpu"): queueContainerCPU,
		},
	}
	queuePorts = []corev1.ContainerPort{{
		Name:          v1alpha1.RequestQueuePortName,
		ContainerPort: int32(v1alpha1.RequestQueuePort),
	}, {
		// Provides health checks and lifecycle hooks.
		Name:          v1alpha1.RequestQueueAdminPortName,
		ContainerPort: int32(v1alpha1.RequestQueueAdminPort),
	}, {
		Name:          v1alpha1.RequestQueueMetricsPortName,
		ContainerPort: int32(v1alpha1.RequestQueueMetricsPort),
	}}
	// This handler (1) marks the service as not ready and (2)
	// adds a small delay before the container is killed.
	queueLifecycle = &corev1.Lifecycle{
		PreStop: &corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Port: intstr.FromInt(v1alpha1.RequestQueueAdminPort),
				Path: queue.RequestQueueQuitPath,
			},
		},
	}
	queueReadinessProbe = &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Port: intstr.FromInt(v1alpha1.RequestQueueAdminPort),
				Path: queue.RequestQueueHealthPath,
			},
		},
		// We want to mark the service as not ready as soon as the
		// PreStop handler is called, so we need to check a little
		// bit more often than the default.  It is a small
		// sacrifice for a low rate of 503s.
		PeriodSeconds: 1,
		// We keep the connection open for a while because we're
		// actively probing the user-container on that endpoint and
		// thus don't want to be limited by K8s granularity here.
		TimeoutSeconds: 10,
	}
)

// makeQueueContainer creates the container spec for queue sidecar.
func makeQueueContainer(rev *v1alpha1.Revision, loggingConfig *logging.Config, autoscalerConfig *autoscaler.Config,
	controllerConfig *config.Controller) *corev1.Container {
	configName := ""
	if owner := metav1.GetControllerOf(rev); owner != nil && owner.Kind == "Configuration" {
		configName = owner.Name
	}

	autoscalerAddress := "autoscaler"
	userPort := getUserPort(rev)

	var loggingLevel string
	if ll, ok := loggingConfig.LoggingLevel["queueproxy"]; ok {
		loggingLevel = ll.String()
	}

	return &corev1.Container{
		Name:           QueueContainerName,
		Image:          controllerConfig.QueueSidecarImage,
		Resources:      queueResources,
		Ports:          queuePorts,
		Lifecycle:      queueLifecycle,
		ReadinessProbe: queueReadinessProbe,
		Env: []corev1.EnvVar{{
			Name:  "SERVING_NAMESPACE",
			Value: rev.Namespace,
		}, {
			Name:  "SERVING_CONFIGURATION",
			Value: configName,
		}, {
			Name:  "SERVING_REVISION",
			Value: rev.Name,
		}, {
			Name:  "SERVING_AUTOSCALER",
			Value: autoscalerAddress,
		}, {
			Name:  "SERVING_AUTOSCALER_PORT",
			Value: strconv.Itoa(autoscalerPort),
		}, {
			Name:  "CONTAINER_CONCURRENCY",
			Value: strconv.Itoa(int(rev.Spec.ContainerConcurrency)),
		}, {
			Name:  "REVISION_TIMEOUT_SECONDS",
			Value: strconv.Itoa(int(rev.Spec.TimeoutSeconds)),
		}, {
			Name: "SERVING_POD",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		}, {
			Name:  "SERVING_LOGGING_CONFIG",
			Value: loggingConfig.LoggingConfig,
		}, {
			Name:  "SERVING_LOGGING_LEVEL",
			Value: loggingLevel,
		}, {
			Name:  "USER_PORT",
			Value: strconv.Itoa(int(userPort)),
		}, {
			Name:  system.NamespaceEnvKey,
			Value: system.Namespace(),
		}},
	}
}
