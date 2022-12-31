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

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	prommetrics "github.com/kedacore/keda/v2/pkg/prommetrics/webhook"
)

var scaledobjectlog = logf.Log.WithName("scaledobject-validation-webhook")

var kc client.Client

const (
	defaultAPI  = "apps/v1"
	defaultKind = "Deployment"
)

func (so *ScaledObject) SetupWebhookWithManager(mgr ctrl.Manager) error {
	kc = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(so).
		Complete()
}

// +kubebuilder:webhook:path=/validate-keda-sh-v1alpha1-scaledobject,mutating=false,failurePolicy=ignore,sideEffects=None,groups=keda.sh,resources=scaledobjects,verbs=create;update,versions=v1alpha1,name=vscaledobject.kb.io,admissionReviewVersions=v1
// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=validatingwebhookconfigurations,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups="",namespace=keda,resources=secrets,verbs=get;list;watch;create;update;patch;delete

var _ webhook.Validator = &ScaledObject{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (so *ScaledObject) ValidateCreate() error {
	val, _ := json.MarshalIndent(so, "", "  ")
	scaledobjectlog.V(1).Info(fmt.Sprintf("validating scaledobject creation for %s", string(val)))
	return validateWorkload(so, "create")
}

func (so *ScaledObject) ValidateUpdate(old runtime.Object) error {
	val, _ := json.MarshalIndent(so, "", "  ")
	scaledobjectlog.V(1).Info(fmt.Sprintf("validating scaledobject update for %s", string(val)))
	return validateWorkload(so, "update")
}

func validateWorkload(so *ScaledObject, action string) error {
	prommetrics.RecordScaledObjectValidatingTotal(so.Namespace, action)
	err := verifyScaledObjects(so, action)
	if err != nil {
		return err
	}
	return verifyHpas(so, action)
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

	for _, hpa := range hpaList.Items {
		val, _ := json.MarshalIndent(hpa, "", "  ")
		scaledobjectlog.V(1).Info(fmt.Sprintf("checking hpa %s: %v", hpa.Name, string(val)))
		hpaTarget := hpa.Spec.ScaleTargetRef
		incomingSoTarget := incomingSo.Spec.ScaleTargetRef

		// prepare default values
		hpatargetAPI := defaultAPI
		if hpaTarget.APIVersion != "" {
			hpatargetAPI = hpaTarget.APIVersion
		}
		hpaTargetKind := defaultKind
		if hpaTarget.Kind != "" {
			hpaTargetKind = hpaTarget.Kind
		}
		incomingSotargetAPI := defaultAPI
		if incomingSoTarget.APIVersion != "" {
			incomingSotargetAPI = incomingSoTarget.APIVersion
		}
		incomingSoTargetKind := defaultKind
		if incomingSoTarget.Kind != "" {
			incomingSoTargetKind = incomingSoTarget.Kind
		}

		if hpatargetAPI == incomingSotargetAPI &&
			hpaTargetKind == incomingSoTargetKind &&
			hpaTarget.Name == incomingSoTarget.Name {
			owned := false
			ownerName := ""
			for _, owner := range hpa.OwnerReferences {
				if owner.Kind == incomingSo.Kind {
					ownerName = owner.Name
					if owner.Name == incomingSo.Name {
						owned = true
					}
				}
			}

			if !owned {
				var err error
				if len(hpa.OwnerReferences) == 0 {
					err = fmt.Errorf("the workload '%s' of type '%s/%s' is already managed by the hpa '%s'", incomingSoTarget.Name, incomingSotargetAPI, incomingSoTargetKind, hpa.Name)
				} else {
					err = fmt.Errorf("the workload '%s' of type '%s/%s' is already managed by the ScaledObject '%s'", incomingSoTarget.Name, incomingSotargetAPI, incomingSoTargetKind, ownerName)
				}

				scaledobjectlog.Error(err, "validation error")
				prommetrics.RecordScaledObjectValidatingErrors(incomingSo.Namespace, action, "hpa")
				return err
			}
		}
	}
	scaledobjectlog.V(1).Info(fmt.Sprintf("scaledobject %s is valid", incomingSo.Name))
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

	for _, so := range soList.Items {
		if so.Name == incomingSo.Name {
			continue
		}
		val, _ := json.MarshalIndent(so, "", "  ")
		scaledobjectlog.V(1).Info(fmt.Sprintf("checking scaledobject %s: %v", so.Name, string(val)))
		soTarget := so.Spec.ScaleTargetRef
		incomingTarget := incomingSo.Spec.ScaleTargetRef

		// prepare default values
		sotargetAPI := defaultAPI
		if soTarget.APIVersion != "" {
			sotargetAPI = soTarget.APIVersion
		}
		soTargetKind := defaultKind
		if soTarget.Kind != "" {
			soTargetKind = soTarget.Kind
		}
		incomingSotargetAPI := defaultAPI
		if incomingTarget.APIVersion != "" {
			incomingSotargetAPI = incomingTarget.APIVersion
		}
		incomingSoTargetKind := defaultKind
		if incomingTarget.Kind != "" {
			incomingSoTargetKind = incomingTarget.Kind
		}

		if sotargetAPI == incomingSotargetAPI &&
			soTargetKind == incomingSoTargetKind &&
			soTarget.Name == incomingTarget.Name {
			err = fmt.Errorf("the workload '%s' of type '%s/%s' is already managed by the ScaledObject '%s'", soTarget.Name, sotargetAPI, soTargetKind, so.Name)
			scaledobjectlog.Error(err, "validation error")
			prommetrics.RecordScaledObjectValidatingErrors(incomingSo.Namespace, action, "scaled_object")
			return err
		}
	}

	scaledobjectlog.V(1).Info(fmt.Sprintf("scaledobject %s is valid", incomingSo.Name))
	return nil
}

func (so *ScaledObject) ValidateDelete() error {
	return nil
}
