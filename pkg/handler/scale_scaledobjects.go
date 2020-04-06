package handler

import (
	"context"
	"fmt"
	"time"

	kedav1alpha1 "github.com/kedacore/keda/pkg/apis/keda/v1alpha1"
	"github.com/kedacore/keda/pkg/scalers"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/apis/duck"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TODO majority of this function could be reused for Jobs, needs refactoring
// GetScaledObjectScalers returns list of Scalers for the specified ScaledObject
func (h *ScaleHandler) GetScaledObjectScalers(scaledObject *kedav1alpha1.ScaledObject) ([]scalers.Scaler, error) {
	scalersRes := []scalers.Scaler{}

	//// TODO move into a separate function ?
	unstruct := &unstructured.Unstructured{}
	unstruct.SetGroupVersionKind(scaledObject.Status.ScaleTargetGVKR.GroupVersionKind())
	if err := h.client.Get(context.TODO(), client.ObjectKey{Namespace: scaledObject.Namespace, Name: scaledObject.Spec.ScaleTargetRef.Name}, unstruct); err != nil {
		// resource doesn't exist
		h.logger.Error(err, "Target resource doesn't exist", "resource", scaledObject.Status.ScaleTargetGVKR.GVKString(), "name", scaledObject.Spec.ScaleTargetRef.Name)
		return scalersRes, err
	}

	obj := &duckv1.WithPod{}
	if err := duck.FromUnstructured(unstruct, obj); err != nil {
		h.logger.Error(err, "Cannot convert unstructured into PodSpecable Duck-type", "object", unstruct)
	}

	if obj.Spec.Template.Spec.Containers == nil {
		h.logger.Info("There aren't any containers in the ScaleTarget", "resource", scaledObject.Status.ScaleTargetGVKR.GVKString(), "name", scaledObject.Spec.ScaleTargetRef.Name)
		return scalersRes, nil
	}
	/////

	resolvedEnv, err := h.resolveContainerEnv(&obj.Spec.Template.Spec, scaledObject.Spec.ScaleTargetRef.ContainerName, scaledObject.Namespace)
	if err != nil {
		return scalersRes, fmt.Errorf("error resolving secrets for ScaleTarget: %s", err)
	}

	for i, trigger := range scaledObject.Spec.Triggers {
		authParams, podIdentity := h.parseScaledObjectAuthRef(trigger.AuthenticationRef, scaledObject, &obj.Spec.Template.Spec)

		if podIdentity == kedav1alpha1.PodIdentityProviderAwsEKS {
			serviceAccountName := obj.Spec.Template.Spec.ServiceAccountName
			serviceAccount := &corev1.ServiceAccount{}
			err = h.client.Get(context.TODO(), types.NamespacedName{Name: serviceAccountName, Namespace: scaledObject.GetNamespace()}, serviceAccount)
			if err != nil {
				closeScalers(scalersRes)
				return []scalers.Scaler{}, fmt.Errorf("error getting service account: %s", err)
			}
			authParams["awsRoleArn"] = serviceAccount.Annotations[kedav1alpha1.PodIdentityAnnotationEKS]
		} else if podIdentity == kedav1alpha1.PodIdentityProviderAwsKiam {
			authParams["awsRoleArn"] = obj.Spec.Template.ObjectMeta.Annotations[kedav1alpha1.PodIdentityAnnotationKiam]
		}

		scaler, err := h.getScaler(scaledObject.Name, scaledObject.Namespace, trigger.Type, resolvedEnv, trigger.Metadata, authParams, podIdentity)
		if err != nil {
			closeScalers(scalersRes)
			return []scalers.Scaler{}, fmt.Errorf("error getting scaler for trigger #%d: %s", i, err)
		}

		scalersRes = append(scalersRes, scaler)
	}

	return scalersRes, nil
}

func (h *ScaleHandler) scaleScaledObject(scaledObject *kedav1alpha1.ScaledObject, isActive bool) {

	currentScale, err := h.getScaleTargetScale(scaledObject)
	if err != nil {
		h.logger.Error(err, "Error getting Scale")
	}

	if currentScale.Spec.Replicas == 0 && isActive {
		// current replica count is 0, but there is an active trigger.
		// scale the ScaleTarget up
		h.scaleFromZero(scaledObject, currentScale)
	} else if !isActive &&
		currentScale.Spec.Replicas > 0 &&
		(scaledObject.Spec.MinReplicaCount == nil || *scaledObject.Spec.MinReplicaCount == 0) {
		// there are no active triggers, but the ScaleTarget has replicas.
		// AND
		// There is no minimum configured or minimum is set to ZERO. HPA will handles other scale down operations

		// Try to scale it down.
		h.scaleToZero(scaledObject, currentScale)
	} else if !isActive &&
		scaledObject.Spec.MinReplicaCount != nil &&
		currentScale.Spec.Replicas < *scaledObject.Spec.MinReplicaCount {
		// there are no active triggers
		// AND
		// ScaleTarget replicas count is less than minimum replica count specified in ScaledObject
		// Let's set ScaleTarget replicas count to correct value
		currentScale.Spec.Replicas = *scaledObject.Spec.MinReplicaCount

		err := h.updateScaleOnScaleTarget(scaledObject, currentScale)
		if err == nil {
			h.logger.Info("Successfully set ScaleTarget replicas count to ScaledObject minReplicaCount",
				"ScaleTarget.Name", scaledObject.Spec.ScaleTargetRef.Name,
				"ScaleTarget.Replicas", currentScale.Spec.Replicas)
		}
	} else if isActive {
		// triggers are active, but we didn't need to scale (replica count > 0)
		// Update LastActiveTime to now.
		now := metav1.Now()
		scaledObject.Status.LastActiveTime = &now
		h.updateScaledObjectStatus(scaledObject)
	} else {
		h.logger.V(1).Info("ScaleTarget no change", "ScaleTarget.Name", scaledObject.Spec.ScaleTargetRef.Name)
	}
}

// An object will be scaled down to 0 only if it's passed its cooldown period
// or if LastActiveTime is nil
func (h *ScaleHandler) scaleToZero(scaledObject *kedav1alpha1.ScaledObject, scale *autoscalingv1.Scale) {
	var cooldownPeriod time.Duration

	if scaledObject.Spec.CooldownPeriod != nil {
		cooldownPeriod = time.Second * time.Duration(*scaledObject.Spec.CooldownPeriod)
	} else {
		cooldownPeriod = time.Second * time.Duration(defaultCooldownPeriod)
	}

	// LastActiveTime can be nil if the ScaleTarget was scaled outside of Keda.
	// In this case we will ignore the cooldown period and scale it down
	if scaledObject.Status.LastActiveTime == nil ||
		scaledObject.Status.LastActiveTime.Add(cooldownPeriod).Before(time.Now()) {
		// or last time a trigger was active was > cooldown period, so scale down.
		scale.Spec.Replicas = 0
		err := h.updateScaleOnScaleTarget(scaledObject, scale)
		if err == nil {
			h.logger.Info("Successfully scaled ScaleTarget to 0 replicas", "ScaleTarget.Name", scaledObject.Spec.ScaleTargetRef.Name)
		}
	} else {
		h.logger.V(1).Info("ScaleTarget cooling down",
			"ScaleTarget.Name", scaledObject.Spec.ScaleTargetRef.Name,
			"LastActiveTime", scaledObject.Status.LastActiveTime,
			"CoolDownPeriod", cooldownPeriod)
	}
}

func (h *ScaleHandler) scaleFromZero(scaledObject *kedav1alpha1.ScaledObject, scale *autoscalingv1.Scale) {
	currentReplicas := scale.Spec.Replicas
	if scaledObject.Spec.MinReplicaCount != nil && *scaledObject.Spec.MinReplicaCount > 0 {
		scale.Spec.Replicas = *scaledObject.Spec.MinReplicaCount
	} else {
		scale.Spec.Replicas = 1
	}

	err := h.updateScaleOnScaleTarget(scaledObject, scale)

	if err == nil {
		h.logger.Info("Successfully updated ScaleTarget",
			"ScaleTarget.Name", scaledObject.Spec.ScaleTargetRef.Name,
			"Original Replicas Count", currentReplicas,
			"New Replicas Count", scale.Spec.Replicas)

		// Scale was successful. Update lastScaleTime and lastActiveTime on the scaledObject
		now := metav1.Now()
		scaledObject.Status.LastActiveTime = &now
		h.updateScaledObjectStatus(scaledObject)
	}
}

func (h *ScaleHandler) getScaleTargetScale(scaledObject *kedav1alpha1.ScaledObject) (*autoscalingv1.Scale, error) {
	return h.scaleClient.Scales(scaledObject.Namespace).Get(scaledObject.Status.ScaleTargetGVKR.GroupResource(), scaledObject.Spec.ScaleTargetRef.Name)
}

func (h *ScaleHandler) updateScaleOnScaleTarget(scaledObject *kedav1alpha1.ScaledObject, scale *autoscalingv1.Scale) error {
	_, err := h.scaleClient.Scales(scaledObject.Namespace).Update(scaledObject.Status.ScaleTargetGVKR.GroupResource(), scale)
	return err
}

func (h *ScaleHandler) parseScaledObjectAuthRef(triggerAuthRef *kedav1alpha1.ScaledObjectAuthRef, scaledObject *kedav1alpha1.ScaledObject, podSpec *corev1.PodSpec) (map[string]string, string) {
	return h.parseAuthRef(triggerAuthRef, scaledObject, func(name, containerName string) string {
		env, err := h.resolveContainerEnv(podSpec, containerName, scaledObject.Namespace)
		if err != nil {
			return ""
		}
		return env[name]
	})
}
