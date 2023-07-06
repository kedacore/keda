/*
Copyright 2023 The KEDA Authors

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//+kubebuilder:scaffold:imports
)

var _ = It("validate triggerauthentication when IdentityID is nil", func() {
	namespaceName := "nilidentityid"
	namespace := createNamespace(namespaceName)
	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	spec := createTriggerAuthenticationSpecWithPodIdentity(PodIdentityProviderAzure, nil)
	ta := createTriggerAuthentication("nilidentityidta", namespaceName, "apps/v1", "TriggerAuthentication", spec)
	Eventually(func() error {
		return k8sClient.Create(context.Background(), ta)
	}).ShouldNot(HaveOccurred())
})

var _ = It("validate triggerauthentication when IdentityID is empty", func() {
	namespaceName := "emptyidentityid"
	namespace := createNamespace(namespaceName)
	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	identityId := ""
	spec := createTriggerAuthenticationSpecWithPodIdentity(PodIdentityProviderAzure, &identityId)
	ta := createTriggerAuthentication("emptyidentityidta", namespaceName, "apps/v1", "TriggerAuthentication", spec)
	Eventually(func() error {
		return k8sClient.Create(context.Background(), ta)
	}).Should(HaveOccurred())
})

var _ = It("validate triggerauthentication when IdentityID is not empty", func() {
	namespaceName := "identityid"
	namespace := createNamespace(namespaceName)
	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	identityId := "12345"
	spec := createTriggerAuthenticationSpecWithPodIdentity(PodIdentityProviderAzure, &identityId)
	ta := createTriggerAuthentication("emptyidentityidta", namespaceName, "apps/v1", "TriggerAuthentication", spec)
	Eventually(func() error {
		return k8sClient.Create(context.Background(), ta)
	}).ShouldNot(HaveOccurred())
})

func createTriggerAuthenticationSpecWithPodIdentity(provider PodIdentityProvider, identityId *string) TriggerAuthenticationSpec {
	return TriggerAuthenticationSpec{
		PodIdentity: &AuthPodIdentity{
			Provider:   provider,
			IdentityID: identityId,
		},
	}
}

func createTriggerAuthentication(name, namespace, targetAPI, targetKind string, spec TriggerAuthenticationSpec) *TriggerAuthentication {
	return &TriggerAuthentication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       targetKind,
			APIVersion: "keda.sh",
		},
		Spec: spec,
	}
}
