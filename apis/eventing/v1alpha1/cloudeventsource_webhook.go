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
	"context"
	"encoding/json"
	"fmt"
	"slices"

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
		WithValidator(&CloudEventSourceCustomValidator{}).
		For(ces).
		Complete()
}

func (cces *ClusterCloudEventSource) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		WithValidator(&ClusterCloudEventSourceCustomValidator{}).
		For(cces).
		Complete()
}

// +kubebuilder:webhook:path=/validate-eventing-keda-sh-v1alpha1-cloudeventsource,mutating=false,failurePolicy=ignore,sideEffects=None,groups=eventing.keda.sh,resources=cloudeventsources,verbs=create;update,versions=v1alpha1,name=vcloudeventsource.kb.io,admissionReviewVersions=v1

// CloudEventSourceCustomValidator is a custom validator for CloudEventSource objects
type CloudEventSourceCustomValidator struct{}

func (cescv CloudEventSourceCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	request, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil, err
	}
	ces := obj.(*CloudEventSource)
	return ces.ValidateCreate(request.DryRun)
}

func (cescv CloudEventSourceCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (warnings admission.Warnings, err error) {
	request, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil, err
	}
	ces := newObj.(*CloudEventSource)
	old := oldObj.(*CloudEventSource)
	return ces.ValidateUpdate(old, request.DryRun)
}

func (cescv CloudEventSourceCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	request, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil, err
	}
	ces := obj.(*CloudEventSource)
	return ces.ValidateDelete(request.DryRun)
}

var _ webhook.CustomValidator = &CloudEventSourceCustomValidator{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (ces *CloudEventSource) ValidateCreate(_ *bool) (admission.Warnings, error) {
	val, _ := json.MarshalIndent(ces, "", "  ")
	cloudeventsourcelog.Info(fmt.Sprintf("validating cloudeventsource creation for %s", string(val)))
	return validateSpec(&ces.Spec)
}

func (ces *CloudEventSource) ValidateUpdate(old runtime.Object, _ *bool) (admission.Warnings, error) {
	val, _ := json.MarshalIndent(ces, "", "  ")
	cloudeventsourcelog.V(1).Info(fmt.Sprintf("validating cloudeventsource update for %s", string(val)))

	oldCes := old.(*CloudEventSource)
	if isCloudEventSourceRemovingFinalizer(ces.ObjectMeta, oldCes.ObjectMeta, ces.Spec, oldCes.Spec) {
		cloudeventsourcelog.V(1).Info("finalizer removal, skipping validation")
		return nil, nil
	}
	return validateSpec(&ces.Spec)
}

func (ces *CloudEventSource) ValidateDelete(_ *bool) (admission.Warnings, error) {
	return nil, nil
}

// +kubebuilder:webhook:path=/validate-eventing-keda-sh-v1alpha1-clustercloudeventsource,mutating=false,failurePolicy=ignore,sideEffects=None,groups=eventing.keda.sh,resources=clustercloudeventsources,verbs=create;update,versions=v1alpha1,name=vclustercloudeventsource.kb.io,admissionReviewVersions=v1

// ClusterCloudEventSourceCustomValidator is a custom validator for ClusterCloudEventSource objects
type ClusterCloudEventSourceCustomValidator struct{}

func (ccescv ClusterCloudEventSourceCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	request, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil, err
	}
	cces := obj.(*ClusterCloudEventSource)
	return cces.ValidateCreate(request.DryRun)
}

func (ccescv ClusterCloudEventSourceCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (warnings admission.Warnings, err error) {
	request, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil, err
	}
	cces := newObj.(*ClusterCloudEventSource)
	old := oldObj.(*ClusterCloudEventSource)
	return cces.ValidateUpdate(old, request.DryRun)
}

func (ccescv ClusterCloudEventSourceCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	request, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil, err
	}
	cces := obj.(*ClusterCloudEventSource)
	return cces.ValidateDelete(request.DryRun)
}

var _ webhook.CustomValidator = &ClusterCloudEventSourceCustomValidator{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (cces *ClusterCloudEventSource) ValidateCreate(_ *bool) (admission.Warnings, error) {
	val, _ := json.MarshalIndent(cces, "", "  ")
	cloudeventsourcelog.Info(fmt.Sprintf("validating clustercloudeventsource creation for %s", string(val)))
	return validateSpec(&cces.Spec)
}

func (cces *ClusterCloudEventSource) ValidateUpdate(old runtime.Object, _ *bool) (admission.Warnings, error) {
	val, _ := json.MarshalIndent(cces, "", "  ")
	cloudeventsourcelog.V(1).Info(fmt.Sprintf("validating clustercloudeventsource update for %s", string(val)))

	oldCes := old.(*ClusterCloudEventSource)
	if isCloudEventSourceRemovingFinalizer(cces.ObjectMeta, oldCes.ObjectMeta, cces.Spec, oldCes.Spec) {
		cloudeventsourcelog.V(1).Info("finalizer removal, skipping validation")
		return nil, nil
	}
	return validateSpec(&cces.Spec)
}

func (cces *ClusterCloudEventSource) ValidateDelete(_ *bool) (admission.Warnings, error) {
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
	if spec.EventSubscription.ExcludedEventTypes != nil && spec.EventSubscription.IncludedEventTypes != nil {
		return nil, fmt.Errorf("setting included types and excluded types at the same time is not supported")
	}

	if spec.EventSubscription.ExcludedEventTypes != nil {
		for _, excludedEventType := range spec.EventSubscription.ExcludedEventTypes {
			if !slices.Contains(AllEventTypes, excludedEventType) {
				return nil, fmt.Errorf("excludedEventType: %s in cloudeventsource/clustercloudeventsource spec is not supported", excludedEventType)
			}
		}
	}

	if spec.EventSubscription.IncludedEventTypes != nil {
		for _, includedEventType := range spec.EventSubscription.IncludedEventTypes {
			if !slices.Contains(AllEventTypes, includedEventType) {
				return nil, fmt.Errorf("includedEventType: %s in cloudeventsource/clustercloudeventsource spec is not supported", includedEventType)
			}
		}
	}
	return nil, nil
}
