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

	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/authentication"
	"github.com/kedacore/keda/v2/pkg/scalers/gcp"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

const (
	// CloudWorkloadIdentityAudiencesEnvVar configures administrator-approved
	// namespace/serviceAccountName -> cloud federation audience mappings for
	// workload service account pod identity. It is strictly isolated from
	// KEDA_SERVICE_ACCOUNT_TOKEN_AUDIENCES (Vault and boundServiceAccountToken).
	CloudWorkloadIdentityAudiencesEnvVar = "KEDA_CLOUD_WORKLOAD_IDENTITY_AUDIENCES"

	// cloudWorkloadIdentityTokenExpirySeconds requests the Kubernetes TokenRequest
	// minimum (10 minutes) because cloud federation assertions are exchanged
	// immediately and never reused across refreshes.
	cloudWorkloadIdentityTokenExpirySeconds int64 = 600
)

// CloudWorkloadIdentityAudience authorizes cloud federation token minting for an
// exact Kubernetes namespace and service account pair with a specific cloud audience.
type CloudWorkloadIdentityAudience struct {
	Audience           string `json:"audience"`
	Namespace          string `json:"namespace"`
	ServiceAccountName string `json:"serviceAccountName"`
}

// LoadCloudWorkloadIdentityAudiences reads cloud workload identity audience mappings from YAML or JSON.
func (cfg *Config) LoadCloudWorkloadIdentityAudiences(value string) error {
	audiences, err := unmarshalSingleYAMLArray[CloudWorkloadIdentityAudience](value, CloudWorkloadIdentityAudiencesEnvVar)
	if err != nil {
		return err
	}
	cfg.CloudWorkloadIdentityAudiences = audiences
	return cfg.validateCloudWorkloadIdentityPolicy()
}

func (cfg *Config) validateCloudWorkloadIdentityPolicy() error {
	if cfg.IsLegacyServiceAccountTokenMode() {
		if len(cfg.CloudWorkloadIdentityAudiences) != 0 {
			return fmt.Errorf("legacy token mode cannot be combined with %s; use enforce-audience or remove the cloud workload identity audience settings", CloudWorkloadIdentityAudiencesEnvVar)
		}
		return nil
	}
	seen := make(map[types.NamespacedName]bool)
	for _, entry := range cfg.CloudWorkloadIdentityAudiences {
		if !isValidCloudWorkloadIdentityAudience(entry.Audience) {
			return fmt.Errorf("unsupported cloud workload identity audience %q in %s: expected a Google workload identity pool provider or a GKE identitynamespace audience", entry.Audience, CloudWorkloadIdentityAudiencesEnvVar)
		}
		if len(validation.IsDNS1123Label(entry.Namespace)) != 0 || len(validation.IsDNS1123Subdomain(entry.ServiceAccountName)) != 0 {
			return fmt.Errorf("%s entries require valid, exact Kubernetes namespace and service account names; wildcards are not supported", CloudWorkloadIdentityAudiencesEnvVar)
		}
		key := types.NamespacedName{Namespace: entry.Namespace, Name: entry.ServiceAccountName}
		if seen[key] {
			return fmt.Errorf("duplicate cloud workload identity audience mapping for %s", key)
		}
		seen[key] = true
	}
	return nil
}

func isValidCloudWorkloadIdentityAudience(audience string) bool {
	_, _, err := gcp.WorkloadIdentityAudiences(audience)
	return err == nil
}

// ResolveServiceAccountTokenProvider resolves an explicitly selected pod identity
// in the scalable object's namespace, including when a CTA is referenced. The
// operator's exact service-account audience mapping and Kubernetes RBAC govern
// delegation; neither the audience nor the namespace comes from the TA/CTA.
func ResolveServiceAccountTokenProvider(ctx context.Context, kubeClient client.Client, triggerAuthRef *kedav1alpha1.AuthenticationRef,
	podIdentity kedav1alpha1.AuthPodIdentity, namespace string, acs *authentication.AuthClientSet,
) (*scalersconfig.ServiceAccountTokenProvider, error) {
	if podIdentity.ServiceAccountName == nil {
		return nil, nil
	}
	name := *podIdentity.ServiceAccountName
	audience, err := workloadServiceAccountAudience(namespace, name)
	if err != nil {
		return nil, err
	}
	tokenAudience, _, err := gcp.WorkloadIdentityAudiences(audience)
	if err != nil {
		return nil, err
	}
	var verify func(context.Context) error
	if triggerAuthRef != nil {
		ref := *triggerAuthRef
		if ref.Kind == "" {
			ref.Kind = "TriggerAuthentication"
		}
		origUID, origGen, _, err := getTriggerAuthPodIdentity(ctx, kubeClient, &ref, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to inspect referenced %s/%s: %w", ref.Kind, ref.Name, err)
		}
		expected := podIdentity.DeepCopy()
		verify = func(ctx context.Context) error {
			currentUID, currentGen, currentPodIdentity, err := getTriggerAuthPodIdentity(ctx, kubeClient, &ref, namespace)
			if err != nil {
				return fmt.Errorf("referenced %s/%s is no longer available: %w", ref.Kind, ref.Name, err)
			}
			if currentUID != origUID || currentGen != origGen || !equality.Semantic.DeepEqual(currentPodIdentity, expected) {
				return fmt.Errorf("referenced %s/%s changed; workload service account credentials must be rebuilt", ref.Kind, ref.Name)
			}
			return nil
		}
	}
	if err := kubeClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &corev1.ServiceAccount{}); err != nil {
		return nil, fmt.Errorf("failed to get pod identity service account %s/%s: %w", namespace, name, err)
	}
	return &scalersconfig.ServiceAccountTokenProvider{
		Audience: audience,
		Verify:   verify,
		GetToken: func(ctx context.Context) (string, error) {
			if verify != nil {
				if err := verify(ctx); err != nil {
					return "", err
				}
			}
			return generateCloudWorkloadIdentityToken(ctx, name, namespace, tokenAudience, acs)
		},
	}, nil
}

func getTriggerAuthPodIdentity(ctx context.Context, kubeClient client.Client, triggerAuthRef *kedav1alpha1.AuthenticationRef, namespace string) (types.UID, int64, *kedav1alpha1.AuthPodIdentity, error) {
	switch triggerAuthRef.Kind {
	case "", "TriggerAuthentication":
		ta := &kedav1alpha1.TriggerAuthentication{}
		if err := kubeClient.Get(ctx, types.NamespacedName{Name: triggerAuthRef.Name, Namespace: namespace}, ta); err != nil {
			return "", 0, nil, err
		}
		return ta.UID, ta.Generation, ta.Spec.PodIdentity, nil
	case "ClusterTriggerAuthentication":
		cta := &kedav1alpha1.ClusterTriggerAuthentication{}
		if err := kubeClient.Get(ctx, types.NamespacedName{Name: triggerAuthRef.Name}, cta); err != nil {
			return "", 0, nil, err
		}
		return cta.UID, cta.Generation, cta.Spec.PodIdentity, nil
	default:
		return "", 0, nil, fmt.Errorf("unknown trigger auth kind %s", triggerAuthRef.Kind)
	}
}

// generateCloudWorkloadIdentityToken mints a Kubernetes assertion with tokenAudience,
// which is derived from the approved cloud federation audience.
func generateCloudWorkloadIdentityToken(ctx context.Context, serviceAccountName, namespace, tokenAudience string, acs *authentication.AuthClientSet) (string, error) {
	audiences := []string{tokenAudience}
	token, err := acs.CoreV1Interface.ServiceAccounts(namespace).CreateToken(ctx, serviceAccountName,
		&authenticationv1.TokenRequest{Spec: authenticationv1.TokenRequestSpec{
			ExpirationSeconds: ptr.To(cloudWorkloadIdentityTokenExpirySeconds),
			Audiences:         audiences,
		}}, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create cloud workload identity token for %s/%s: %w", namespace, serviceAccountName, err)
	}
	if token == nil || token.Status.Token == "" {
		return "", fmt.Errorf("token request for %s/%s returned an empty token", namespace, serviceAccountName)
	}
	if err := validateK8sSATokenAudiences([]byte(token.Status.Token), audiences); err != nil {
		return "", fmt.Errorf("invalid token returned for service account %s/%s: %w", namespace, serviceAccountName, err)
	}
	log.V(1).Info("Cloud workload identity token created successfully", "ServiceAccount.Namespace", namespace, "ServiceAccount.Name", serviceAccountName, "Token.Audiences", audiences)
	return token.Status.Token, nil
}

func workloadServiceAccountAudience(namespace, name string) (string, error) {
	for _, mapping := range globalConfig.CloudWorkloadIdentityAudiences {
		if mapping.Namespace == namespace && mapping.ServiceAccountName == name {
			return mapping.Audience, nil
		}
	}
	return "", fmt.Errorf("no configured cloud workload identity token audience for service account %s/%s in %s", namespace, name, CloudWorkloadIdentityAudiencesEnvVar)
}
