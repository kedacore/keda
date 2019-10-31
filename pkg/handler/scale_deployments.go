package handler

import (
	"context"
	"fmt"
	"time"

	kedav1alpha1 "github.com/kedacore/keda/pkg/apis/keda/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

func (h *ScaleHandler) scaleDeployment(deployment *appsv1.Deployment, scaledObject *kedav1alpha1.ScaledObject, isActive bool) {

	if *deployment.Spec.Replicas == 0 && isActive {
		// current replica count is 0, but there is an active trigger.
		// scale the deployment up
		h.scaleFromZero(deployment, scaledObject)
	} else if !isActive &&
		*deployment.Spec.Replicas > 0 &&
		(scaledObject.Spec.MinReplicaCount == nil || *scaledObject.Spec.MinReplicaCount == 0) {
		// there are no active triggers, but the deployment has replicas.
		// AND
		// There is no minimum configured or minimum is set to ZERO. HPA will handles other scale down operations

		// Try to scale it down.
		h.scaleDeploymentToZero(deployment, scaledObject)
	} else if isActive {
		// triggers are active, but we didn't need to scale (replica count > 0)
		// Update LastActiveTime to now.
		now := metav1.Now()
		scaledObject.Status.LastActiveTime = &now
		h.updateScaledObjectStatus(scaledObject)
	} else {
		h.logger.V(1).Info("Deployment no change", "Deployment.Namespace", deployment.GetNamespace(), "Deployment.Name", deployment.GetName())
	}
}

func (h *ScaleHandler) updateDeployment(deployment *appsv1.Deployment) error {

	err := h.client.Update(context.TODO(), deployment)
	if err != nil {
		h.logger.Error(err, "Error updating deployment", "Deployment.Namespace", deployment.GetNamespace(), "Deployment.Name", deployment.GetName())
		return err
	}
	return nil
}

// A deployment will be scaled down to 0 only if it's passed its cooldown period
// or if LastActiveTime is nil
func (h *ScaleHandler) scaleDeploymentToZero(deployment *appsv1.Deployment, scaledObject *kedav1alpha1.ScaledObject) {
	var cooldownPeriod time.Duration

	if scaledObject.Spec.CooldownPeriod != nil {
		cooldownPeriod = time.Second * time.Duration(*scaledObject.Spec.CooldownPeriod)
	} else {
		cooldownPeriod = time.Second * time.Duration(defaultCooldownPeriod)
	}

	// LastActiveTime can be nil if the deployment was scaled outside of Keda.
	// In this case we will ignore the cooldown period and scale it down
	if scaledObject.Status.LastActiveTime == nil ||
		scaledObject.Status.LastActiveTime.Add(cooldownPeriod).Before(time.Now()) {
		// or last time a trigger was active was > cooldown period, so scale down.
		*deployment.Spec.Replicas = 0
		err := h.updateDeployment(deployment)
		if err == nil {
			h.logger.Info("Successfully scaled deployment to 0 replicas", "Deployment.Namespace", deployment.GetNamespace(), "Deployment.Name", deployment.GetName())
		}
	} else {
		h.logger.V(1).Info("scaledObject cooling down",
			"LastActiveTime",
			scaledObject.Status.LastActiveTime,
			"CoolDownPeriod",
			cooldownPeriod)
	}
}

func (h *ScaleHandler) scaleFromZero(deployment *appsv1.Deployment, scaledObject *kedav1alpha1.ScaledObject) {
	currentReplicas := *deployment.Spec.Replicas
	if scaledObject.Spec.MinReplicaCount != nil && *scaledObject.Spec.MinReplicaCount > 0 {
		deployment.Spec.Replicas = scaledObject.Spec.MinReplicaCount
	} else {
		*deployment.Spec.Replicas = 1
	}

	err := h.updateDeployment(deployment)

	if err == nil {
		h.logger.Info("Successfully updated deployment", "Deployment.Namespace", deployment.GetNamespace(), "Deployment.Name", deployment.GetName(),
			"Original Replicas Count",
			currentReplicas,
			"New Replicas Count",
			*deployment.Spec.Replicas)

		// Scale was successful. Update lastScaleTime and lastActiveTime on the scaledObject
		now := metav1.Now()
		scaledObject.Status.LastActiveTime = &now
		h.updateScaledObjectStatus(scaledObject)
	}
}

func (h *ScaleHandler) resolveDeploymentEnv(deployment *appsv1.Deployment, containerName string) (map[string]string, error) {
	deploymentKey, err := cache.MetaNamespaceKeyFunc(deployment)
	if err != nil {
		return nil, err
	}

	if len(deployment.Spec.Template.Spec.Containers) < 1 {
		return nil, fmt.Errorf("Deployment (%s) doesn't have containers", deploymentKey)
	}

	var container corev1.Container

	if containerName != "" {
		for _, c := range deployment.Spec.Template.Spec.Containers {
			if c.Name == containerName {
				container = c
				break
			}
		}

		if &container == nil {
			return nil, fmt.Errorf("Couldn't find container with name %s on deployment %s", containerName, deployment.GetName())
		}
	} else {
		container = deployment.Spec.Template.Spec.Containers[0]
	}

	return h.resolveEnv(&container, deployment.GetNamespace())
}

func (h *ScaleHandler) parseDeploymentAuthRef(triggerAuthRef kedav1alpha1.ScaledObjectAuthRef, scaledObject *kedav1alpha1.ScaledObject, deployment *appsv1.Deployment) (map[string]string, string) {
	return h.parseAuthRef(triggerAuthRef, scaledObject, func(name, containerName string) string {
		env, err := h.resolveDeploymentEnv(deployment, containerName)
		if err != nil {
			return ""
		}
		return env[name]
	})
}
