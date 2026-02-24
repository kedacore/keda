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
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	kedaString     = "keda"
	workloadString = "workload"
)

var triggerauthenticationlog = logf.Log.WithName("triggerauthentication-validation-webhook")

func (ta *TriggerAuthentication) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		WithValidator(&TriggerAuthenticationCustomValidator{}).
		For(ta).
		Complete()
}

func (cta *ClusterTriggerAuthentication) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		WithValidator(&ClusterTriggerAuthenticationCustomValidator{}).
		For(cta).
		Complete()
}

// +kubebuilder:webhook:path=/validate-keda-sh-v1alpha1-triggerauthentication,mutating=false,failurePolicy=ignore,sideEffects=None,groups=keda.sh,resources=triggerauthentications,verbs=create;update,versions=v1alpha1,name=vstriggerauthentication.kb.io,admissionReviewVersions=v1

// TriggerAuthenticationCustomValidator is a custom validator for TriggerAuthentication objects
type TriggerAuthenticationCustomValidator struct{}

func (tacv TriggerAuthenticationCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	request, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil, err
	}
	ta := obj.(*TriggerAuthentication)
	return ta.ValidateCreate(request.DryRun)
}

func (tacv TriggerAuthenticationCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (warnings admission.Warnings, err error) {
	request, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil, err
	}
	ta := newObj.(*TriggerAuthentication)
	old := oldObj.(*TriggerAuthentication)
	return ta.ValidateUpdate(old, request.DryRun)
}

func (tacv TriggerAuthenticationCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	request, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil, err
	}
	ta := obj.(*TriggerAuthentication)
	return ta.ValidateDelete(request.DryRun)
}

var _ webhook.CustomValidator = &TriggerAuthenticationCustomValidator{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (ta *TriggerAuthentication) ValidateCreate(_ *bool) (admission.Warnings, error) {
	val, _ := json.MarshalIndent(ta, "", "  ")
	triggerauthenticationlog.Info(fmt.Sprintf("validating triggerauthentication creation for %s", string(val)))
	return validateSpec(&ta.Spec)
}

func (ta *TriggerAuthentication) ValidateUpdate(old runtime.Object, _ *bool) (admission.Warnings, error) {
	val, _ := json.MarshalIndent(ta, "", "  ")
	scaledobjectlog.V(1).Info(fmt.Sprintf("validating triggerauthentication update for %s", string(val)))

	oldTa := old.(*TriggerAuthentication)
	if isTriggerAuthenticationRemovingFinalizer(ta.ObjectMeta, oldTa.ObjectMeta, ta.Spec, oldTa.Spec) {
		triggerauthenticationlog.V(1).Info("finalizer removal, skipping validation")
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

func (ctacv ClusterTriggerAuthenticationCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	request, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil, err
	}
	cta := obj.(*ClusterTriggerAuthentication)
	return cta.ValidateCreate(request.DryRun)
}

func (ctacv ClusterTriggerAuthenticationCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (warnings admission.Warnings, err error) {
	request, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil, err
	}
	cta := newObj.(*ClusterTriggerAuthentication)
	old := oldObj.(*ClusterTriggerAuthentication)
	return cta.ValidateUpdate(old, request.DryRun)
}

func (ctacv ClusterTriggerAuthenticationCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	request, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil, err
	}
	cta := obj.(*ClusterTriggerAuthentication)
	return cta.ValidateDelete(request.DryRun)
}

var _ webhook.CustomValidator = &ClusterTriggerAuthenticationCustomValidator{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (cta *ClusterTriggerAuthentication) ValidateCreate(_ *bool) (admission.Warnings, error) {
	val, _ := json.MarshalIndent(cta, "", "  ")
	triggerauthenticationlog.Info(fmt.Sprintf("validating clustertriggerauthentication creation for %s", string(val)))
	return validateSpec(&cta.Spec)
}

func (cta *ClusterTriggerAuthentication) ValidateUpdate(old runtime.Object, _ *bool) (admission.Warnings, error) {
	val, _ := json.MarshalIndent(cta, "", "  ")
	scaledobjectlog.V(1).Info(fmt.Sprintf("validating clustertriggerauthentication update for %s", string(val)))

	oldCta := old.(*ClusterTriggerAuthentication)
	if isTriggerAuthenticationRemovingFinalizer(cta.ObjectMeta, oldCta.ObjectMeta, cta.Spec, oldCta.Spec) {
		triggerauthenticationlog.V(1).Info("finalizer removal, skipping validation")
		return nil, nil
	}

	return validateSpec(&cta.Spec)
}

func (cta *ClusterTriggerAuthentication) ValidateDelete(_ *bool) (admission.Warnings, error) {
	return nil, nil
}

func isTriggerAuthenticationRemovingFinalizer(om metav1.ObjectMeta, oldOm metav1.ObjectMeta, spec TriggerAuthenticationSpec, oldSpec TriggerAuthenticationSpec) bool {
	taSpec, _ := json.MarshalIndent(spec, "", "  ")
	oldTaSpec, _ := json.MarshalIndent(oldSpec, "", "  ")
	taSpecString := string(taSpec)
	oldTaSpecString := string(oldTaSpec)

	return len(om.Finalizers) == 0 && len(oldOm.Finalizers) == 1 && taSpecString == oldTaSpecString
}

func validateSpec(spec *TriggerAuthenticationSpec) (admission.Warnings, error) {
	if spec.PodIdentity != nil {
		switch spec.PodIdentity.Provider {
		case PodIdentityProviderAzureWorkload:
			if spec.PodIdentity.IdentityID != nil && *spec.PodIdentity.IdentityID == "" {
				return nil, fmt.Errorf("identityId of PodIdentity should not be empty. If it's set, identityId has to be different than \"\"")
			}

			if spec.PodIdentity.IdentityAuthorityHost != nil && *spec.PodIdentity.IdentityAuthorityHost != "" {
				if spec.PodIdentity.IdentityTenantID == nil || *spec.PodIdentity.IdentityTenantID == "" {
					return nil, fmt.Errorf("identityTenantID of PodIdentity should not be nil or empty when identityAuthorityHost of PodIdentity is set")
				}
			} else if spec.PodIdentity.IdentityTenantID != nil && *spec.PodIdentity.IdentityTenantID == "" {
				return nil, fmt.Errorf("identityTenantId of PodIdentity should not be empty. If it's set, identityTenantId has to be different than \"\"")
			}
		case PodIdentityProviderAws:
			if spec.PodIdentity.RoleArn != nil && *spec.PodIdentity.RoleArn != "" && spec.PodIdentity.IsWorkloadIdentityOwner() {
				return nil, fmt.Errorf("roleArn of PodIdentity can't be set if KEDA isn't identityOwner")
			}
			if spec.PodIdentity.ExternalID != nil && *spec.PodIdentity.ExternalID != "" {
				if spec.PodIdentity.RoleArn == nil || *spec.PodIdentity.RoleArn == "" {
					return nil, fmt.Errorf("externalID of PodIdentity requires roleArn to be set")
				}
			}
		default:
			return nil, nil
		}
	}
	return nil, nil
}
