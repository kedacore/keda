/*
Copyright 2022 The KEDA Authors

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

package resolver

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/go-logr/logr"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/keda-scalers/azure"
)

type AzureKeyVaultHandler struct {
	vault          *kedav1alpha1.AzureKeyVault
	keyvaultClient *azsecrets.Client
}

func NewAzureKeyVaultHandler(v *kedav1alpha1.AzureKeyVault) *AzureKeyVaultHandler {
	return &AzureKeyVaultHandler{
		vault: v,
	}
}

func (vh *AzureKeyVaultHandler) Initialize(ctx context.Context, client client.Client, logger logr.Logger, triggerNamespace string, secretsLister corev1listers.SecretLister) error {
	cred, err := vh.getCredentials(ctx, client, logger, triggerNamespace, secretsLister)
	if err != nil {
		return err
	}

	keyvaultClient, err := azsecrets.NewClient(vh.vault.VaultURI, cred, nil)
	if err != nil {
		return err
	}

	vh.keyvaultClient = keyvaultClient
	return nil
}

func (vh *AzureKeyVaultHandler) Read(ctx context.Context, secretName string, version string) (string, error) {
	result, err := vh.keyvaultClient.GetSecret(ctx, secretName, version, nil)
	if err != nil {
		return "", err
	}

	return *result.Value, nil
}

func (vh *AzureKeyVaultHandler) getCredentials(ctx context.Context, client client.Client, logger logr.Logger,
	triggerNamespace string, secretsLister corev1listers.SecretLister) (azcore.TokenCredential, error) {
	podIdentity := vh.vault.PodIdentity
	if podIdentity == nil {
		podIdentity = &kedav1alpha1.AuthPodIdentity{}
	}
	switch podIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		missingErr := fmt.Errorf("clientID, tenantID and clientSecret are expected when not using a pod identity provider")
		if vh.vault.Credentials == nil {
			return nil, missingErr
		}

		clientID := vh.vault.Credentials.ClientID
		tenantID := vh.vault.Credentials.TenantID

		clientSecretName := vh.vault.Credentials.ClientSecret.ValueFrom.SecretKeyRef.Name
		clientSecretKey := vh.vault.Credentials.ClientSecret.ValueFrom.SecretKeyRef.Key
		clientSecret := resolveAuthSecret(ctx, client, logger, clientSecretName, triggerNamespace, clientSecretKey, secretsLister)

		if clientID == "" || tenantID == "" || clientSecret == "" {
			return nil, missingErr
		}
		return azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
	case kedav1alpha1.PodIdentityProviderAzureWorkload:
		return azure.NewChainedCredential(logger, *podIdentity)
	default:
		return nil, fmt.Errorf("key vault does not support pod identity provider - %s", podIdentity.Provider)
	}
}
