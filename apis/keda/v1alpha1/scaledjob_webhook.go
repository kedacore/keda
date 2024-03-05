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
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var scaledjoblog = logf.Log.WithName("scaledjob-validation-webhook")

func (s *ScaledJob) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(s).
		Complete()
}

// +kubebuilder:webhook:path=/validate-keda-sh-v1alpha1-scaledjob,mutating=false,failurePolicy=ignore,sideEffects=None,groups=keda.sh,resources=scaledjobs,verbs=create;update,versions=v1alpha1,name=vscaledjob.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &ScaledJob{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (s *ScaledJob) ValidateCreate() (admission.Warnings, error) {
	val, _ := json.MarshalIndent(s, "", "  ")
	scaledjoblog.Info(fmt.Sprintf("validating scaledjob creation for %s", string(val)))
	return nil, verifyTriggers(s, "create", false)
}

func (s *ScaledJob) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	val, _ := json.MarshalIndent(s, "", "  ")
	scaledobjectlog.V(1).Info(fmt.Sprintf("validating scaledjob update for %s", string(val)))

	oldTa := old.(*ScaledJob)
	if isScaledJobRemovingFinalizer(s.ObjectMeta, oldTa.ObjectMeta, s.Spec, oldTa.Spec) {
		scaledjoblog.V(1).Info("finalizer removal, skipping validation")
		return nil, nil
	}
	return nil, verifyTriggers(s, "update", false)
}

func (s *ScaledJob) ValidateDelete() (admission.Warnings, error) {
	return nil, nil
}

func isScaledJobRemovingFinalizer(om metav1.ObjectMeta, oldOm metav1.ObjectMeta, spec ScaledJobSpec, oldSpec ScaledJobSpec) bool {
	taSpec, _ := json.MarshalIndent(spec, "", "  ")
	oldTaSpec, _ := json.MarshalIndent(oldSpec, "", "  ")
	taSpecString := string(taSpec)
	oldTaSpecString := string(oldTaSpec)

	return len(om.Finalizers) == 0 && len(oldOm.Finalizers) == 1 && taSpecString == oldTaSpecString
}
