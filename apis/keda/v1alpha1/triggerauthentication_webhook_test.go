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
)

var _ = It("validate triggerauthentication when IdentityTenantID is not nil and not empty", func() {
	namespaceName := "identitytenantidta"
	namespace := createNamespace(namespaceName)
	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	identityID := "12345"
	identityTenantID := "12345"
	spec := createTriggerAuthenticationSpecWithPodIdentity(PodIdentityProviderAzureWorkload, nil, &identityID, &identityTenantID, nil, nil)
	ta := createTriggerAuthentication("identitytenantidta", namespaceName, "TriggerAuthentication", spec)
	Eventually(func() error {
		return k8sClient.Create(context.Background(), ta)
	}).ShouldNot(HaveOccurred())
})

var _ = It("validate triggerauthentication when IdentityTenantID is not nil but empty", func() {
	namespaceName := "emptyidentitytenantidta"
	namespace := createNamespace(namespaceName)
	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	identityID := "12345"
	identityTenantID := ""
	spec := createTriggerAuthenticationSpecWithPodIdentity(PodIdentityProviderAzureWorkload, nil, &identityID, &identityTenantID, nil, nil)
	ta := createTriggerAuthentication("emptyidentitytenantidta", namespaceName, "TriggerAuthentication", spec)
	Eventually(func() error {
		return k8sClient.Create(context.Background(), ta)
	}).Should(HaveOccurred())
})

var _ = It("validate triggerauthentication when IdentityAuthorityHost is not nil and not empty and IdentityTenantID is not nil and not empty", func() {
	namespaceName := "identityauthorityhostta"
	namespace := createNamespace(namespaceName)
	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	identityID := "12345"
	identityTenantID := "12345"
	identityAuthorityHost := "12345"
	spec := createTriggerAuthenticationSpecWithPodIdentity(PodIdentityProviderAzureWorkload, nil, &identityID, &identityTenantID, &identityAuthorityHost, nil)
	ta := createTriggerAuthentication("identityauthorityhostta", namespaceName, "TriggerAuthentication", spec)
	Eventually(func() error {
		return k8sClient.Create(context.Background(), ta)
	}).ShouldNot(HaveOccurred())
})

var _ = It("validate triggerauthentication when IdentityAuthorityHost is not nil and not empty and IdentityTenantID is nil", func() {
	namespaceName := "niltenantidentityauthorityhostta"
	namespace := createNamespace(namespaceName)
	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	identityID := "12345"
	identityAuthorityHost := "12345"
	spec := createTriggerAuthenticationSpecWithPodIdentity(PodIdentityProviderAzureWorkload, nil, &identityID, nil, &identityAuthorityHost, nil)
	ta := createTriggerAuthentication("niltenantidentityauthorityhostta", namespaceName, "TriggerAuthentication", spec)
	Eventually(func() error {
		return k8sClient.Create(context.Background(), ta)
	}).Should(HaveOccurred())
})

var _ = It("validate triggerauthentication when IdentityAuthorityHost is not nil and not empty and IdentityTenantID is not nil but empty", func() {
	namespaceName := "emptytenantidentityauthorityhostta"
	namespace := createNamespace(namespaceName)
	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	identityID := "12345"
	identityTenantID := ""
	identityAuthorityHost := "12345"
	spec := createTriggerAuthenticationSpecWithPodIdentity(PodIdentityProviderAzureWorkload, nil, &identityID, &identityTenantID, &identityAuthorityHost, nil)
	ta := createTriggerAuthentication("emptytenantidentityauthorityhostta", namespaceName, "TriggerAuthentication", spec)
	Eventually(func() error {
		return k8sClient.Create(context.Background(), ta)
	}).Should(HaveOccurred())
})

var _ = It("validate triggerauthentication when RoleArn is not empty and IdentityOwner is nil", func() {
	namespaceName := "rolearn"
	namespace := createNamespace(namespaceName)
	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	roleArn := "Hello"
	spec := createTriggerAuthenticationSpecWithPodIdentity(PodIdentityProviderAws, &roleArn, nil, nil, nil, nil)
	ta := createTriggerAuthentication("identityidta", namespaceName, "TriggerAuthentication", spec)
	Eventually(func() error {
		return k8sClient.Create(context.Background(), ta)
	}).ShouldNot(HaveOccurred())
})

var _ = It("validate triggerauthentication when RoleArn is not empty and IdentityOwner is keda", func() {
	namespaceName := "rolearnandkedaowner"
	namespace := createNamespace(namespaceName)
	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	roleArn := "Hello"
	identityOwner := kedaString
	spec := createTriggerAuthenticationSpecWithPodIdentity(PodIdentityProviderAws, &roleArn, nil, nil, nil, &identityOwner)
	ta := createTriggerAuthentication("identityidta", namespaceName, "TriggerAuthentication", spec)
	Eventually(func() error {
		return k8sClient.Create(context.Background(), ta)
	}).ShouldNot(HaveOccurred())
})

var _ = It("validate triggerauthentication when RoleArn is not empty and IdentityOwner is workload", func() {
	namespaceName := "rolearnandworkloadowner"
	namespace := createNamespace(namespaceName)
	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	roleArn := "Hello"
	identityOwner := workloadString
	spec := createTriggerAuthenticationSpecWithPodIdentity(PodIdentityProviderAws, &roleArn, nil, nil, nil, &identityOwner)
	ta := createTriggerAuthentication("identityidta", namespaceName, "TriggerAuthentication", spec)
	Eventually(func() error {
		return k8sClient.Create(context.Background(), ta)
	}).Should(HaveOccurred())
})

var _ = It("validate triggerauthentication when RoleArn is empty and IdentityOwner is keda", func() {
	namespaceName := "kedaowner"
	namespace := createNamespace(namespaceName)
	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	identityOwner := kedaString
	spec := createTriggerAuthenticationSpecWithPodIdentity(PodIdentityProviderAws, nil, nil, nil, nil, &identityOwner)
	ta := createTriggerAuthentication("identityidta", namespaceName, "TriggerAuthentication", spec)
	Eventually(func() error {
		return k8sClient.Create(context.Background(), ta)
	}).ShouldNot(HaveOccurred())
})

var _ = It("validate triggerauthentication when RoleArn is not empty and IdentityOwner is workload", func() {
	namespaceName := "workloadowner"
	namespace := createNamespace(namespaceName)
	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	identityOwner := workloadString
	spec := createTriggerAuthenticationSpecWithPodIdentity(PodIdentityProviderAws, nil, nil, nil, nil, &identityOwner)
	ta := createTriggerAuthentication("identityidta", namespaceName, "TriggerAuthentication", spec)
	Eventually(func() error {
		return k8sClient.Create(context.Background(), ta)
	}).ShouldNot(HaveOccurred())
})

var _ = It("validate clustertriggerauthentication when RoleArn is not empty and IdentityOwner is nil", func() {
	namespaceName := "clusterrolearn"
	namespace := createNamespace(namespaceName)
	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	roleArn := "Hello"
	spec := createTriggerAuthenticationSpecWithPodIdentity(PodIdentityProviderAws, &roleArn, nil, nil, nil, nil)
	ta := createTriggerAuthentication("clusteridentityidta", namespaceName, "ClusterTriggerAuthentication", spec)
	Eventually(func() error {
		return k8sClient.Create(context.Background(), ta)
	}).ShouldNot(HaveOccurred())
})

var _ = It("validate clustertriggerauthentication when RoleArn is not empty and IdentityOwner is keda", func() {
	namespaceName := "clusterrolearnandkedaowner"
	namespace := createNamespace(namespaceName)
	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	roleArn := "Hello"
	identityOwner := kedaString
	spec := createTriggerAuthenticationSpecWithPodIdentity(PodIdentityProviderAws, &roleArn, nil, nil, nil, &identityOwner)
	ta := createTriggerAuthentication("clusteridentityidta", namespaceName, "ClusterTriggerAuthentication", spec)
	Eventually(func() error {
		return k8sClient.Create(context.Background(), ta)
	}).ShouldNot(HaveOccurred())
})

var _ = It("validate clustertriggerauthentication when RoleArn is not empty and IdentityOwner is workload", func() {
	namespaceName := "clusterrolearnandworkloadowner"
	namespace := createNamespace(namespaceName)
	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	roleArn := "Hello"
	identityOwner := workloadString
	spec := createTriggerAuthenticationSpecWithPodIdentity(PodIdentityProviderAws, &roleArn, nil, nil, nil, &identityOwner)
	ta := createTriggerAuthentication("clusteridentityidta", namespaceName, "ClusterTriggerAuthentication", spec)
	Eventually(func() error {
		return k8sClient.Create(context.Background(), ta)
	}).Should(HaveOccurred())
})

var _ = It("validate clustertriggerauthentication when RoleArn is empty and IdentityOwner is keda", func() {
	namespaceName := "clusterandkedaowner"
	namespace := createNamespace(namespaceName)
	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	identityOwner := kedaString
	spec := createTriggerAuthenticationSpecWithPodIdentity(PodIdentityProviderAws, nil, nil, nil, nil, &identityOwner)
	ta := createTriggerAuthentication("clusteridentityidta", namespaceName, "ClusterTriggerAuthentication", spec)
	Eventually(func() error {
		return k8sClient.Create(context.Background(), ta)
	}).ShouldNot(HaveOccurred())
})

var _ = It("validate clustertriggerauthentication when RoleArn is not empty and IdentityOwner is workload", func() {
	namespaceName := "clusterandworkloadowner"
	namespace := createNamespace(namespaceName)
	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	identityOwner := workloadString
	spec := createTriggerAuthenticationSpecWithPodIdentity(PodIdentityProviderAws, nil, nil, nil, nil, &identityOwner)
	ta := createTriggerAuthentication("clusteridentityidta", namespaceName, "TriggerAuthentication", spec)
	Eventually(func() error {
		return k8sClient.Create(context.Background(), ta)
	}).ShouldNot(HaveOccurred())
})

var _ = It("validate triggerauthentication when FilePath is set", func() {
	namespaceName := "filepathvalidation"
	namespace := createNamespace(namespaceName)
	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	spec := TriggerAuthenticationSpec{
		FilePath: "/mnt/auth/creds.json",
	}
	ta := createTriggerAuthentication("filepathvalidation", namespaceName, "TriggerAuthentication", spec)
	Eventually(func() error {
		return k8sClient.Create(context.Background(), ta)
	}).Should(HaveOccurred())
})

var _ = It("validate clustertriggerauthentication when FilePath is set", func() {
	namespaceName := "clusterfilepathvalidation"
	namespace := createNamespace(namespaceName)
	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).ToNot(HaveOccurred())

	spec := TriggerAuthenticationSpec{
		FilePath: "/mnt/auth/creds.json",
	}
	ta := createTriggerAuthentication("clusterfilepathvalidation", namespaceName, "ClusterTriggerAuthentication", spec)
	Eventually(func() error {
		return k8sClient.Create(context.Background(), ta)
	}).ShouldNot(HaveOccurred())
})

func createTriggerAuthenticationSpecWithPodIdentity(provider PodIdentityProvider, roleArn, identityID, identityTenantID, identityAuthorityHost, identityOwner *string) TriggerAuthenticationSpec {
	return TriggerAuthenticationSpec{
		PodIdentity: &AuthPodIdentity{
			Provider:              provider,
			IdentityID:            identityID,
			IdentityTenantID:      identityTenantID,
			IdentityAuthorityHost: identityAuthorityHost,
			RoleArn:               roleArn,
			IdentityOwner:         identityOwner,
		},
	}
}

func createTriggerAuthentication(name, namespace, targetKind string, spec TriggerAuthenticationSpec) *TriggerAuthentication {
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
