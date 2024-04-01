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

	"golang.org/x/exp/slices"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var cloudeventsourcelog = logf.Log.WithName("cloudeventsource-validation-webhook")

func (ces *CloudEventSource) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(ces).
		Complete()
}

// +kubebuilder:webhook:path=/validate-eventing-keda-sh-v1alpha1-cloudeventsource,mutating=false,failurePolicy=ignore,sideEffects=None,groups=eventing.keda.sh,resources=cloudeventsources,verbs=create;update,versions=v1alpha1,name=vcloudeventsource.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &CloudEventSource{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (ces *CloudEventSource) ValidateCreate() (admission.Warnings, error) {
	val, _ := json.MarshalIndent(ces, "", "  ")
	cloudeventsourcelog.Info(fmt.Sprintf("validating cloudeventsource creation for %s", string(val)))
	return validateSpec(&ces.Spec)
}

func (ces *CloudEventSource) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	val, _ := json.MarshalIndent(ces, "", "  ")
	cloudeventsourcelog.V(1).Info(fmt.Sprintf("validating cloudeventsource update for %s", string(val)))

	oldCes := old.(*CloudEventSource)
	if isCloudEventSourceRemovingFinalizer(ces.ObjectMeta, oldCes.ObjectMeta, ces.Spec, oldCes.Spec) {
		cloudeventsourcelog.V(1).Info("finalizer removal, skipping validation")
		return nil, nil
	}
	return validateSpec(&ces.Spec)
}

func (ces *CloudEventSource) ValidateDelete() (admission.Warnings, error) {
	return nil, nil
}

func isCloudEventSourceRemovingFinalizer(om metav1.ObjectMeta, oldOm metav1.ObjectMeta, spec CloudEventSourceSpec, oldSpec CloudEventSourceSpec) bool {
	cesSpec, _ := json.MarshalIndent(spec, "", "  ")
	oldCesSpec, _ := json.MarshalIndent(oldSpec, "", "  ")
	cesSpecString := string(cesSpec)
	oldCesSpecString := string(oldCesSpec)

	return len(om.Finalizers) == 0 && len(oldOm.Finalizers) == 1 && cesSpecString == oldCesSpecString
}

func validateSpec(spec *CloudEventSourceSpec) (admission.Warnings, error) {
	if spec.EventSubscription.ExcludedEventTypes != nil {
		for _, excludedEventType := range spec.EventSubscription.ExcludedEventTypes {
			if !slices.Contains(AllEventTypes, excludedEventType) {
				return nil, fmt.Errorf("excludedEventType: %s in cloudeventsource spec is not supported", excludedEventType)
			}
		}
	}

	if spec.EventSubscription.IncludedEventTypes != nil {
		for _, includedEventType := range spec.EventSubscription.IncludedEventTypes {
			if !slices.Contains(AllEventTypes, includedEventType) {
				return nil, fmt.Errorf("includedEventType: %s in cloudeventsource spec is not supported", includedEventType)
			}
		}
	}

	if spec.EventSubscription.ExcludedEventTypes != nil && spec.EventSubscription.IncludedEventTypes != nil {
		for _, excludedEventType := range spec.EventSubscription.ExcludedEventTypes {
			if slices.Contains(spec.EventSubscription.IncludedEventTypes, excludedEventType) {
				return nil, fmt.Errorf("eventType: %s is in both included typs and excluded types, which is not supported", excludedEventType)
			}
		}
	}

	return nil, nil
}
