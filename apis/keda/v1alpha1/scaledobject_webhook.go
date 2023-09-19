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

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	prommetrics "github.com/kedacore/keda/v2/pkg/prommetrics/webhook"
)

var scaledobjectlog = logf.Log.WithName("scaledobject-validation-webhook")

var kc client.Client
var restMapper meta.RESTMapper

var memoryString = "memory"
var cpuString = "cpu"

func (so *ScaledObject) SetupWebhookWithManager(mgr ctrl.Manager) error {
	kc = mgr.GetClient()
	restMapper = mgr.GetRESTMapper()
	return ctrl.NewWebhookManagedBy(mgr).
		For(so).
		Complete()
}

// +kubebuilder:webhook:path=/validate-keda-sh-v1alpha1-scaledobject,mutating=false,failurePolicy=ignore,sideEffects=None,groups=keda.sh,resources=scaledobjects,verbs=create;update,versions=v1alpha1,name=vscaledobject.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &ScaledObject{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (so *ScaledObject) ValidateCreate() (admission.Warnings, error) {
	val, _ := json.MarshalIndent(so, "", "  ")
	scaledobjectlog.V(1).Info(fmt.Sprintf("validating scaledobject creation for %s", string(val)))
	return validateWorkload(so, "create")
}

func (so *ScaledObject) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	val, _ := json.MarshalIndent(so, "", "  ")
	scaledobjectlog.V(1).Info(fmt.Sprintf("validating scaledobject update for %s", string(val)))

	if isRemovingFinalizer(so, old) {
		scaledobjectlog.V(1).Info("finalizer removal, skipping validation")
		return nil, nil
	}

	return validateWorkload(so, "update")
}

func (so *ScaledObject) ValidateDelete() (admission.Warnings, error) {
	return nil, nil
}

func isRemovingFinalizer(so *ScaledObject, old runtime.Object) bool {
	oldSo := old.(*ScaledObject)

	soSpec, _ := json.MarshalIndent(so.Spec, "", "  ")
	oldSoSpec, _ := json.MarshalIndent(oldSo.Spec, "", "  ")
	soSpecString := string(soSpec)
	oldSoSpecString := string(oldSoSpec)

	return len(so.ObjectMeta.Finalizers) == 0 && len(oldSo.ObjectMeta.Finalizers) == 1 && soSpecString == oldSoSpecString
}

func validateWorkload(so *ScaledObject, action string) (admission.Warnings, error) {
	prommetrics.RecordScaledObjectValidatingTotal(so.Namespace, action)

	verifyFunctions := []func(*ScaledObject, string) error{
		verifyCPUMemoryScalers,
		verifyTriggers,
		verifyScaledObjects,
		verifyHpas,
	}

	for i := range verifyFunctions {
		err := verifyFunctions[i](so, action)
		if err != nil {
			return nil, err
		}
	}

	scaledobjectlog.V(1).Info(fmt.Sprintf("scaledobject %s is valid", so.Name))
	return nil, nil
}

func verifyTriggers(incomingSo *ScaledObject, action string) error {
	err := ValidateTriggers(incomingSo.Spec.Triggers)
	if err != nil {
		scaledobjectlog.WithValues("name", incomingSo.Name).Error(err, "validation error")
		prommetrics.RecordScaledObjectValidatingErrors(incomingSo.Namespace, action, "incorrect-triggers")
	}
	return err
}

func verifyHpas(incomingSo *ScaledObject, action string) error {
	hpaList := &autoscalingv2.HorizontalPodAutoscalerList{}
	opt := &client.ListOptions{
		Namespace: incomingSo.Namespace,
	}
	err := kc.List(context.Background(), hpaList, opt)
	if err != nil {
		return err
	}

	var incomingSoGckr GroupVersionKindResource
	incomingSoGckr, err = ParseGVKR(restMapper, incomingSo.Spec.ScaleTargetRef.APIVersion, incomingSo.Spec.ScaleTargetRef.Kind)
	if err != nil {
		scaledobjectlog.Error(err, "Failed to parse Group, Version, Kind, Resource from incoming ScaledObject", "apiVersion", incomingSo.Spec.ScaleTargetRef.APIVersion, "kind", incomingSo.Spec.ScaleTargetRef.Kind)
		return err
	}

	for _, hpa := range hpaList.Items {
		val, _ := json.MarshalIndent(hpa, "", "  ")
		scaledobjectlog.V(1).Info(fmt.Sprintf("checking hpa %s: %v", hpa.Name, string(val)))

		hpaGckr, err := ParseGVKR(restMapper, hpa.Spec.ScaleTargetRef.APIVersion, hpa.Spec.ScaleTargetRef.Kind)
		if err != nil {
			scaledobjectlog.Error(err, "Failed to parse Group, Version, Kind, Resource from HPA", "hpaName", hpa.Name, "apiVersion", hpa.Spec.ScaleTargetRef.APIVersion, "kind", hpa.Spec.ScaleTargetRef.Kind)
			return err
		}

		if hpaGckr.GVKString() == incomingSoGckr.GVKString() &&
			hpa.Spec.ScaleTargetRef.Name == incomingSo.Spec.ScaleTargetRef.Name {
			owned := false
			for _, owner := range hpa.OwnerReferences {
				if owner.Kind == incomingSo.Kind {
					if owner.Name == incomingSo.Name {
						owned = true
						break
					}
				}
			}

			if !owned {
				if incomingSo.ObjectMeta.Annotations[ScaledObjectTransferHpaOwnershipAnnotation] == "true" &&
					incomingSo.Spec.Advanced.HorizontalPodAutoscalerConfig.Name == hpa.Name {
					scaledobjectlog.Info(fmt.Sprintf("%s hpa ownership being transferred to %s", hpa.Name, incomingSo.Name))
				} else {
					err = fmt.Errorf("the workload '%s' of type '%s' is already managed by the hpa '%s'", incomingSo.Spec.ScaleTargetRef.Name, incomingSoGckr.GVKString(), hpa.Name)
					scaledobjectlog.Error(err, "validation error")
					prommetrics.RecordScaledObjectValidatingErrors(incomingSo.Namespace, action, "other-hpa")
					return err
				}
			}
		}
	}
	return nil
}

func verifyScaledObjects(incomingSo *ScaledObject, action string) error {
	soList := &ScaledObjectList{}
	opt := &client.ListOptions{
		Namespace: incomingSo.Namespace,
	}
	err := kc.List(context.Background(), soList, opt)
	if err != nil {
		return err
	}

	incomingSoGckr, err := ParseGVKR(restMapper, incomingSo.Spec.ScaleTargetRef.APIVersion, incomingSo.Spec.ScaleTargetRef.Kind)
	if err != nil {
		scaledobjectlog.Error(err, "Failed to parse Group, Version, Kind, Resource from incoming ScaledObject", "apiVersion", incomingSo.Spec.ScaleTargetRef.APIVersion, "kind", incomingSo.Spec.ScaleTargetRef.Kind)
		return err
	}

	for _, so := range soList.Items {
		if so.Name == incomingSo.Name {
			continue
		}
		val, _ := json.MarshalIndent(so, "", "  ")
		scaledobjectlog.V(1).Info(fmt.Sprintf("checking scaledobject %s: %v", so.Name, string(val)))

		soGckr, err := ParseGVKR(restMapper, so.Spec.ScaleTargetRef.APIVersion, so.Spec.ScaleTargetRef.Kind)
		if err != nil {
			scaledobjectlog.Error(err, "Failed to parse Group, Version, Kind, Resource from ScaledObject", "soName", so.Name, "apiVersion", so.Spec.ScaleTargetRef.APIVersion, "kind", so.Spec.ScaleTargetRef.Kind)
			return err
		}

		if soGckr.GVKString() == incomingSoGckr.GVKString() &&
			so.Spec.ScaleTargetRef.Name == incomingSo.Spec.ScaleTargetRef.Name {
			err = fmt.Errorf("the workload '%s' of type '%s' is already managed by the ScaledObject '%s'", so.Spec.ScaleTargetRef.Name, incomingSoGckr.GVKString(), so.Name)
			scaledobjectlog.Error(err, "validation error")
			prommetrics.RecordScaledObjectValidatingErrors(incomingSo.Namespace, action, "other-scaled-object")
			return err
		}
	}

	return nil
}

func verifyCPUMemoryScalers(incomingSo *ScaledObject, action string) error {
	var podSpec *corev1.PodSpec
	for _, trigger := range incomingSo.Spec.Triggers {
		if trigger.Type == cpuString || trigger.Type == memoryString {
			if podSpec == nil {
				key := types.NamespacedName{
					Namespace: incomingSo.Namespace,
					Name:      incomingSo.Spec.ScaleTargetRef.Name,
				}
				incomingSoGckr, err := ParseGVKR(restMapper, incomingSo.Spec.ScaleTargetRef.APIVersion, incomingSo.Spec.ScaleTargetRef.Kind)
				if err != nil {
					scaledobjectlog.Error(err, "Failed to parse Group, Version, Kind, Resource from incoming ScaledObject", "apiVersion", incomingSo.Spec.ScaleTargetRef.APIVersion, "kind", incomingSo.Spec.ScaleTargetRef.Kind)
					return err
				}

				switch incomingSoGckr.GVKString() {
				case "apps/v1.Deployment":
					deployment := &appsv1.Deployment{}
					err := kc.Get(context.Background(), key, deployment, &client.GetOptions{})
					if err != nil {
						return err
					}
					podSpec = &deployment.Spec.Template.Spec
				case "apps/v1.StatefulSet":
					statefulset := &appsv1.StatefulSet{}
					err := kc.Get(context.Background(), key, statefulset, &client.GetOptions{})
					if err != nil {
						return err
					}
					podSpec = &statefulset.Spec.Template.Spec
				default:
					return nil
				}
			}
			conainerName := trigger.Metadata["containerName"]
			for _, container := range podSpec.Containers {
				if conainerName != "" && container.Name != conainerName {
					continue
				}
				if trigger.Type == cpuString {
					if container.Resources.Requests == nil ||
						container.Resources.Requests.Cpu() == nil ||
						container.Resources.Requests.Cpu().AsApproximateFloat64() == 0 {
						err := fmt.Errorf("the scaledobject has a cpu trigger but the container %s doesn't have the cpu request defined", container.Name)
						scaledobjectlog.Error(err, "validation error")
						prommetrics.RecordScaledObjectValidatingErrors(incomingSo.Namespace, action, "missing-requests")
						return err
					}
				} else if trigger.Type == memoryString {
					if container.Resources.Requests == nil ||
						container.Resources.Requests.Memory() == nil ||
						container.Resources.Requests.Memory().AsApproximateFloat64() == 0 {
						err := fmt.Errorf("the scaledobject has a memory trigger but the container %s doesn't have the memory request defined", container.Name)
						scaledobjectlog.Error(err, "validation error")
						prommetrics.RecordScaledObjectValidatingErrors(incomingSo.Namespace, action, "missing-requests")
						return err
					}
				}
			}

			// validate scaledObject with cpu/mem triggers:
			// If scaled object has only cpu/mem triggers AND has minReplicaCount 0
			// return an error because it will never scale to zero
			scaleToZeroErr := true
			for _, trig := range incomingSo.Spec.Triggers {
				if trig.Type != cpuString && trig.Type != memoryString {
					scaleToZeroErr = false
					break
				}
			}

			if (scaleToZeroErr && incomingSo.Spec.MinReplicaCount == nil) || (scaleToZeroErr && *incomingSo.Spec.MinReplicaCount == 0) {
				err := fmt.Errorf("scaledobject has only cpu/memory triggers AND minReplica is 0 (scale to zero doesn't work in this case)")
				scaledobjectlog.Error(err, "validation error")
				prommetrics.RecordScaledObjectValidatingErrors(incomingSo.Namespace, action, "scale-to-zero-requirements-not-met")
				return err
			}
		}
	}
	return nil
}
