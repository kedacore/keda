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
	"fmt"
	"net/url"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	kedaString     = "keda"
	workloadString = "workload"
)

var triggerauthenticationlog = logf.Log.WithName("triggerauthentication-validation-webhook")

func (ta *TriggerAuthentication) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, ta).
		WithValidator(&TriggerAuthenticationCustomValidator{}).
		Complete()
}

func (cta *ClusterTriggerAuthentication) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, cta).
		WithValidator(&ClusterTriggerAuthenticationCustomValidator{}).
		Complete()
}

// +kubebuilder:webhook:path=/validate-keda-sh-v1alpha1-triggerauthentication,mutating=false,failurePolicy=ignore,sideEffects=None,groups=keda.sh,resources=triggerauthentications,verbs=create;update,versions=v1alpha1,name=vstriggerauthentication.kb.io,admissionReviewVersions=v1

// TriggerAuthenticationCustomValidator is a custom validator for TriggerAuthentication objects
type TriggerAuthenticationCustomValidator struct{}

func (tacv TriggerAuthenticationCustomValidator) ValidateCreate(ctx context.Context, ta *TriggerAuthentication) (warnings admission.Warnings, err error) {
	request, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return ta.ValidateCreate(request.DryRun)
}

func (tacv TriggerAuthenticationCustomValidator) ValidateUpdate(ctx context.Context, old, ta *TriggerAuthentication) (warnings admission.Warnings, err error) {
	request, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return ta.ValidateUpdate(old, request.DryRun)
}

func (tacv TriggerAuthenticationCustomValidator) ValidateDelete(ctx context.Context, ta *TriggerAuthentication) (warnings admission.Warnings, err error) {
	request, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return ta.ValidateDelete(request.DryRun)
}

var _ admission.Validator[*TriggerAuthentication] = &TriggerAuthenticationCustomValidator{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (ta *TriggerAuthentication) ValidateCreate(_ *bool) (admission.Warnings, error) {
	triggerauthenticationlog.Info("validating triggerauthentication creation", "namespace", ta.Namespace, "name", ta.Name, "triggerauthentication", ta)
	return validateSpec(&ta.Spec)
}

func (ta *TriggerAuthentication) ValidateUpdate(old runtime.Object, _ *bool) (admission.Warnings, error) {
	triggerauthenticationlog.V(1).Info("validating triggerauthentication update", "namespace", ta.Namespace, "name", ta.Name, "triggerauthentication", ta)

	oldTa := old.(*TriggerAuthentication)
	if isTriggerAuthenticationRemovingFinalizer(ta.ObjectMeta, oldTa.ObjectMeta, ta.Spec, oldTa.Spec) {
		triggerauthenticationlog.V(1).Info("finalizer removal, skipping validation", "namespace", ta.Namespace, "name", ta.Name)
		return nil, nil
	}
	return validateSpec(&ta.Spec)
}

func (ta *TriggerAuthentication) ValidateDelete(_ *bool) (admission.Warnings, error) {
	return nil, nil
}

// +kubebuilder:webhook:path=/validate-keda-sh-v1alpha1-clustertriggerauthentication,mutating=false,failurePolicy=ignore,sideEffects=None,groups=keda.sh,resources=clustertriggerauthentications,verbs=create;update,versions=v1alpha1,name=vsclustertriggerauthentication.kb.io,admissionReviewVersions=v1

// ClusterTriggerAuthenticationCustomValidator is a custom validator for ClusterTriggerAuthentication objects
type ClusterTriggerAuthenticationCustomValidator struct{}

func (ctacv ClusterTriggerAuthenticationCustomValidator) ValidateCreate(ctx context.Context, cta *ClusterTriggerAuthentication) (warnings admission.Warnings, err error) {
	request, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return cta.ValidateCreate(request.DryRun)
}

func (ctacv ClusterTriggerAuthenticationCustomValidator) ValidateUpdate(ctx context.Context, old, cta *ClusterTriggerAuthentication) (warnings admission.Warnings, err error) {
	request, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return cta.ValidateUpdate(old, request.DryRun)
}

func (ctacv ClusterTriggerAuthenticationCustomValidator) ValidateDelete(ctx context.Context, cta *ClusterTriggerAuthentication) (warnings admission.Warnings, err error) {
	request, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return cta.ValidateDelete(request.DryRun)
}

var _ admission.Validator[*ClusterTriggerAuthentication] = &ClusterTriggerAuthenticationCustomValidator{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (cta *ClusterTriggerAuthentication) ValidateCreate(_ *bool) (admission.Warnings, error) {
	triggerauthenticationlog.Info("validating clustertriggerauthentication creation", "name", cta.Name, "clustertriggerauthentication", cta)
	return validateSpec(&cta.Spec)
}

func (cta *ClusterTriggerAuthentication) ValidateUpdate(old runtime.Object, _ *bool) (admission.Warnings, error) {
	triggerauthenticationlog.V(1).Info("validating clustertriggerauthentication update", "name", cta.Name, "clustertriggerauthentication", cta)

	oldCta := old.(*ClusterTriggerAuthentication)
	if isTriggerAuthenticationRemovingFinalizer(cta.ObjectMeta, oldCta.ObjectMeta, cta.Spec, oldCta.Spec) {
		triggerauthenticationlog.V(1).Info("finalizer removal, skipping validation", "name", cta.Name)
		return nil, nil
	}

	return validateSpec(&cta.Spec)
}

func (cta *ClusterTriggerAuthentication) ValidateDelete(_ *bool) (admission.Warnings, error) {
	return nil, nil
}

func isTriggerAuthenticationRemovingFinalizer(om metav1.ObjectMeta, oldOm metav1.ObjectMeta, spec TriggerAuthenticationSpec, oldSpec TriggerAuthenticationSpec) bool {
	return len(om.Finalizers) == 0 && len(oldOm.Finalizers) == 1 && equality.Semantic.DeepEqual(spec, oldSpec)
}

func validateSpec(spec *TriggerAuthenticationSpec) (admission.Warnings, error) {
	warnings := validateHashiCorpVaultCredential(spec)

	if spec.PodIdentity != nil {
		switch spec.PodIdentity.Provider {
		case PodIdentityProviderAzureWorkload:
			if spec.PodIdentity.IdentityID != nil && *spec.PodIdentity.IdentityID == "" {
				return warnings, fmt.Errorf("identityId of PodIdentity should not be empty. If it's set, identityId has to be different than \"\"")
			}

			if spec.PodIdentity.IdentityAuthorityHost != nil && *spec.PodIdentity.IdentityAuthorityHost != "" {
				if spec.PodIdentity.IdentityTenantID == nil || *spec.PodIdentity.IdentityTenantID == "" {
					return warnings, fmt.Errorf("identityTenantID of PodIdentity should not be nil or empty when identityAuthorityHost of PodIdentity is set")
				}
			} else if spec.PodIdentity.IdentityTenantID != nil && *spec.PodIdentity.IdentityTenantID == "" {
				return warnings, fmt.Errorf("identityTenantId of PodIdentity should not be empty. If it's set, identityTenantId has to be different than \"\"")
			}
		case PodIdentityProviderAws:
			if spec.PodIdentity.RoleArn != nil && *spec.PodIdentity.RoleArn != "" && spec.PodIdentity.IsWorkloadIdentityOwner() {
				return warnings, fmt.Errorf("roleArn of PodIdentity can't be set if KEDA isn't identityOwner")
			}
			if spec.PodIdentity.ExternalID != nil && *spec.PodIdentity.ExternalID != "" {
				if spec.PodIdentity.RoleArn == nil || *spec.PodIdentity.RoleArn == "" {
					return nil, fmt.Errorf("externalID of PodIdentity requires roleArn to be set")
				}
			}
		default:
			return warnings, nil
		}
	}

	if spec.OAuth2 != nil {
		if err := validateOAuth2(spec.OAuth2); err != nil {
			return nil, err
		}
	}

	return warnings, nil
}

func validateHashiCorpVaultCredential(spec *TriggerAuthenticationSpec) admission.Warnings {
	var warnings admission.Warnings

	if spec.HashiCorpVault != nil && spec.HashiCorpVault.Credential != nil {
		if spec.HashiCorpVault.Credential.Token != "" {
			warnings = append(warnings, "spec.hashiCorpVault.credential.token is deprecated; use spec.hashiCorpVault.credential.tokenFrom.secretKeyRef instead")
		}
		if spec.HashiCorpVault.Credential.Token != "" && spec.HashiCorpVault.Credential.TokenFrom != nil {
			warnings = append(warnings, "spec.hashiCorpVault.credential.tokenFrom.secretKeyRef takes precedence over spec.hashiCorpVault.credential.token")
		}
	}

	return warnings
}

func validateOAuth2(oauth2 *OAuth2) error {
	if oauth2.Type != OAuth2GrantTypeClientCredentials {
		return fmt.Errorf("oauth2.type must be 'clientCredentials', got '%s'", oauth2.Type)
	}

	if oauth2.ClientID == "" {
		return fmt.Errorf("oauth2.clientId is required when oauth2 is configured")
	}

	if oauth2.TokenURL == "" {
		return fmt.Errorf("oauth2.tokenUrl is required when oauth2 is configured")
	}

	parsedURL, err := url.Parse(oauth2.TokenURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return fmt.Errorf("oauth2.tokenUrl must be a valid http or https URL")
	}

	return nil
}
