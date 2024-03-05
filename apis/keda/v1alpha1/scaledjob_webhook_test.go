/*
Copyright 2024 The KEDA Authors

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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = It("should validate empty triggers in ScaledJob", func() {

	namespaceName := "scaledjob-empty-triggers-set"
	namespace := createNamespace(namespaceName)

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	sj := createScaledJob(sjName, namespaceName, []ScaleTriggers{})

	Eventually(func() error {
		return k8sClient.Create(context.Background(), sj)
	}).Should(HaveOccurred())
})

// -------------------------------------------------------------------------- //
// ----------------------------- HELP FUNCTIONS ----------------------------- //
// -------------------------------------------------------------------------- //
func createScaledJob(name string, namespace string, triggers []ScaleTriggers) *ScaledJob {
	return &ScaledJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "ScaledJob",
			APIVersion: "keda.sh",
		},
		Spec: ScaledJobSpec{
			JobTargetRef: &batchv1.JobSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": name,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": name,
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  name,
								Image: name,
							},
						},
					},
				},
			},
			Triggers: triggers,
		},
	}
}
