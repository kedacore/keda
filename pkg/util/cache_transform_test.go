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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCacheObjectTransform_StripsManagedFields(t *testing.T) {
	obj := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
			ManagedFields: []metav1.ManagedFieldsEntry{
				{Manager: "kubectl", Operation: "Apply"},
			},
		},
	}

	result, err := CacheObjectTransform(obj)
	require.NoError(t, err)
	cm := result.(*corev1.ConfigMap)
	assert.Nil(t, cm.ManagedFields)
	assert.Equal(t, "test", cm.Name)
}

func TestCacheObjectTransform_StripsPodFields(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-pod",
			Namespace: "default",
			Labels: map[string]string{
				"app": "test",
			},
			Annotations: map[string]string{
				"big-annotation": "lots-of-data",
			},
			ManagedFields: []metav1.ManagedFieldsEntry{
				{Manager: "kubelet"},
				{Manager: "kube-scheduler"},
			},
		},
		Spec: corev1.PodSpec{
			NodeName: "node-1",
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "nginx:latest",
					Env: []corev1.EnvVar{
						{Name: "FOO", Value: "bar"},
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{Name: "data", MountPath: "/data"},
					},
				},
				{
					Name:  "sidecar",
					Image: "envoy:latest",
				},
			},
			InitContainers: []corev1.Container{
				{Name: "init", Image: "busybox"},
			},
			Volumes: []corev1.Volume{
				{Name: "data", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
			},
			Tolerations: []corev1.Toleration{
				{Key: "node.kubernetes.io/not-ready"},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: corev1.ConditionTrue},
			},
			PodIP:  "10.0.0.1",
			HostIP: "192.168.1.1",
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "app",
					Ready: true,
					Image: "nginx:latest",
				},
			},
		},
	}

	result, err := CacheObjectTransform(pod)
	require.NoError(t, err)

	p := result.(*corev1.Pod)

	// Metadata: ManagedFields and Annotations stripped, Name/Labels preserved
	assert.Equal(t, "my-pod", p.Name)
	assert.Equal(t, "default", p.Namespace)
	assert.Equal(t, map[string]string{"app": "test"}, p.Labels)
	assert.Nil(t, p.ManagedFields)
	assert.Nil(t, p.Annotations)

	// Spec: only NodeName preserved
	assert.Equal(t, "node-1", p.Spec.NodeName)
	assert.Nil(t, p.Spec.Containers)
	assert.Nil(t, p.Spec.InitContainers)
	assert.Nil(t, p.Spec.Volumes)
	assert.Nil(t, p.Spec.Tolerations)

	// Status: only Phase and Conditions preserved
	assert.Equal(t, corev1.PodRunning, p.Status.Phase)
	assert.Len(t, p.Status.Conditions, 1)
	assert.Equal(t, corev1.PodReady, p.Status.Conditions[0].Type)
	assert.Empty(t, p.Status.PodIP)
	assert.Empty(t, p.Status.HostIP)
	assert.Nil(t, p.Status.ContainerStatuses)
}

func TestCacheObjectTransform_NilManagedFields(t *testing.T) {
	obj := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}

	result, err := CacheObjectTransform(obj)
	require.NoError(t, err)
	cm := result.(*corev1.ConfigMap)
	assert.Nil(t, cm.ManagedFields)
	assert.Equal(t, "test", cm.Name)
}

func TestCacheObjectTransform_NonMetaObject(t *testing.T) {
	plain := "not-a-k8s-object"
	result, err := CacheObjectTransform(plain)
	require.NoError(t, err)
	assert.Equal(t, "not-a-k8s-object", result)
}
