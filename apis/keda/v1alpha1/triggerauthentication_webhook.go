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
		For(ta).
		Complete()
}

func (cta *ClusterTriggerAuthentication) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(cta).
		Complete()
}

// +kubebuilder:webhook:path=/validate-keda-sh-v1alpha1-triggerauthentication,mutating=false,failurePolicy=ignore,sideEffects=None,groups=keda.sh,resources=triggerauthentications,verbs=create;update,versions=v1alpha1,name=vstriggerauthentication.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &TriggerAuthentication{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (ta *TriggerAuthentication) ValidateCreate() (admission.Warnings, error) {
	val, _ := json.MarshalIndent(ta, "", "  ")
	triggerauthenticationlog.Info(fmt.Sprintf("validating triggerauthentication creation for %s", string(val)))
	return validateSpec(&ta.Spec)
}

func (ta *TriggerAuthentication) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	val, _ := json.MarshalIndent(ta, "", "  ")
	scaledobjectlog.V(1).Info(fmt.Sprintf("validating triggerauthentication update for %s", string(val)))

	oldTa := old.(*TriggerAuthentication)
	if isTriggerAuthenticationRemovingFinalizer(ta.ObjectMeta, oldTa.ObjectMeta, ta.Spec, oldTa.Spec) {
		triggerauthenticationlog.V(1).Info("finalizer removal, skipping validation")
		return nil, nil
	}
	return validateSpec(&ta.Spec)
}

func (ta *TriggerAuthentication) ValidateDelete() (admission.Warnings, error) {
	return nil, nil
}

// +kubebuilder:webhook:path=/validate-keda-sh-v1alpha1-clustertriggerauthentication,mutating=false,failurePolicy=ignore,sideEffects=None,groups=keda.sh,resources=clustertriggerauthentications,verbs=create;update,versions=v1alpha1,name=vsclustertriggerauthentication.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &ClusterTriggerAuthentication{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (cta *ClusterTriggerAuthentication) ValidateCreate() (admission.Warnings, error) {
	val, _ := json.MarshalIndent(cta, "", "  ")
	triggerauthenticationlog.Info(fmt.Sprintf("validating clustertriggerauthentication creation for %s", string(val)))
	return validateSpec(&cta.Spec)
}

func (cta *ClusterTriggerAuthentication) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	val, _ := json.MarshalIndent(cta, "", "  ")
	scaledobjectlog.V(1).Info(fmt.Sprintf("validating clustertriggerauthentication update for %s", string(val)))

	oldCta := old.(*ClusterTriggerAuthentication)
	if isTriggerAuthenticationRemovingFinalizer(cta.ObjectMeta, oldCta.ObjectMeta, cta.Spec, oldCta.Spec) {
		triggerauthenticationlog.V(1).Info("finalizer removal, skipping validation")
		return nil, nil
	}

	return validateSpec(&cta.Spec)
}

func (cta *ClusterTriggerAuthentication) ValidateDelete() (admission.Warnings, error) {
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
		case PodIdentityProviderAzure, PodIdentityProviderAzureWorkload:
			if spec.PodIdentity.IdentityID != nil && *spec.PodIdentity.IdentityID == "" {
				return nil, fmt.Errorf("identityid of PodIdentity should not be empty. If it's set, identityId has to be different than \"\"")
			}
		case PodIdentityProviderAws:
			if spec.PodIdentity.RoleArn != "" && spec.PodIdentity.IsWorkloadIdentityOwner() {
				return nil, fmt.Errorf("roleArn of PodIdentity can't be set if KEDA isn't identityOwner")
			}
		default:
			return nil, nil
		}
	}
	return nil, nil
}
