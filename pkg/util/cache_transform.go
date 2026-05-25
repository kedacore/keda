/*
Copyright 2026 The KEDA Authors

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

package util

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CacheObjectTransform is a controller-runtime cache.DefaultTransform that
// strips fields KEDA never reads from cached objects. This significantly
// reduces the memory footprint of the informer cache in large clusters.
//
// For all objects: removes ManagedFields.
// For Pods: preserves ObjectMeta (name, namespace, labels, etc.) but clears
// annotations; replaces Spec with only NodeName; replaces Status with only
// Phase and Conditions. Everything else (containers, volumes, env,
// containerStatuses, etc.) is cleared.
func CacheObjectTransform(obj any) (any, error) {
	if accessor, ok := obj.(metav1.Object); ok {
		if accessor.GetManagedFields() != nil {
			accessor.SetManagedFields(nil)
		}
	}

	if pod, ok := obj.(*corev1.Pod); ok {
		stripPodFields(pod)
	}

	return obj, nil
}

// stripPodFields removes Pod fields that KEDA never reads. KEDA only uses:
//   - status.phase (workload scaler, fallback, scale_jobs)
//   - status.conditions (fallback, scale_jobs)
//   - spec.nodeName (workload scaler with groupByNode)
func stripPodFields(pod *corev1.Pod) {
	pod.Annotations = nil

	nodeName := pod.Spec.NodeName
	pod.Spec = corev1.PodSpec{
		NodeName: nodeName,
	}

	phase := pod.Status.Phase
	conditions := pod.Status.Conditions
	pod.Status = corev1.PodStatus{
		Phase:      phase,
		Conditions: conditions,
	}
}
