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

package resolver

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/authentication"
	"github.com/kedacore/keda/v2/pkg/scalers/gcp"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

// ResolveServiceAccountTokenProvider resolves an explicitly selected pod identity
// in the scalable object's namespace, including when a CTA is referenced. The
// operator's exact service-account audience mapping and Kubernetes RBAC govern
// delegation; neither the audience nor the namespace comes from the TA/CTA.
func ResolveServiceAccountTokenProvider(ctx context.Context, kubeClient client.Client, podIdentity kedav1alpha1.AuthPodIdentity,
	namespace string, acs *authentication.AuthClientSet,
) (*scalersconfig.ServiceAccountTokenProvider, error) {
	if err := podIdentity.ValidateServiceAccountName(); err != nil {
		return nil, err
	}
	if podIdentity.ServiceAccountName == nil {
		return nil, nil
	}
	name := *podIdentity.ServiceAccountName
	audience, err := workloadServiceAccountAudience(namespace, name)
	if err != nil {
		return nil, err
	}
	if err := validatePodIdentityAudience(podIdentity.Provider, audience); err != nil {
		return nil, err
	}
	if acs == nil || acs.CoreV1Interface == nil {
		return nil, fmt.Errorf("service account token client is required for pod identity serviceAccountName")
	}
	if err := kubeClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &corev1.ServiceAccount{}); err != nil {
		return nil, fmt.Errorf("failed to get pod identity service account %s/%s: %w", namespace, name, err)
	}
	return &scalersconfig.ServiceAccountTokenProvider{
		Namespace:          namespace,
		ServiceAccountName: name,
		Audience:           audience,
		GetToken: func(ctx context.Context) (string, error) {
			// Refresh under the same approval used to construct the cloud credential.
			// Legacy compatibility must never relax this new auth path.
			currentAudience, err := workloadServiceAccountAudience(namespace, name)
			if err != nil {
				return "", err
			}
			if currentAudience != audience {
				return "", fmt.Errorf("configured token audience changed for service account %s/%s", namespace, name)
			}
			ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			return GenerateBoundServiceAccountToken(ctx, name, namespace, acs)
		},
	}, nil
}

func validatePodIdentityAudience(provider kedav1alpha1.PodIdentityProvider, audience string) error {
	switch provider {
	case kedav1alpha1.PodIdentityProviderGCP:
		_, err := gcp.WorkloadIdentityProviderAudience(audience)
		return err
	case kedav1alpha1.PodIdentityProviderAzureWorkload:
		if audience != "api://AzureADTokenExchange" {
			return fmt.Errorf("azure-workload service account token audience must be api://AzureADTokenExchange")
		}
	case kedav1alpha1.PodIdentityProviderAws:
		if audience != "sts.amazonaws.com" {
			return fmt.Errorf("aws service account token audience must be sts.amazonaws.com")
		}
	default:
		return fmt.Errorf("service account selection is not supported for provider %s", provider)
	}
	return nil
}

func workloadServiceAccountAudience(namespace, name string) (string, error) {
	if globalConfig.IsLegacyServiceAccountTokenMode() {
		return "", fmt.Errorf("pod identity serviceAccountName requires enforce-audience service account token mode")
	}
	audiences, err := globalConfig.serviceAccountTokenAudiences(namespace, name)
	if err != nil {
		return "", err
	}
	return audiences[0], nil
}
