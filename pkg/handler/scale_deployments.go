package handler

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	keda_v1alpha1 "github.com/kedacore/keda/pkg/apis/keda/v1alpha1"
	apps_v1 "k8s.io/api/apps/v1"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

func (h *ScaleHandler) scaleDeployment(deployment *apps_v1.Deployment, scaledObject *keda_v1alpha1.ScaledObject, isActive bool) {

	if *deployment.Spec.Replicas == 0 && isActive {
		// current replica count is 0, but there is an active trigger.
		// scale the deployment up
		h.scaleDeploymentFromZero(deployment, scaledObject)
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
		now := meta_v1.Now()
		scaledObject.Status.LastActiveTime = &now
		h.updateScaledObject(scaledObject)
	} else {
		log.Debugf("deployment (%s/%s) no change", deployment.GetNamespace(), deployment.GetName())
	}
}

func (h *ScaleHandler) updateDeployment(deployment *apps_v1.Deployment) error {
	_, err := h.kubeClient.AppsV1().Deployments(deployment.GetNamespace()).Update(deployment)
	if err != nil {
		log.Errorf("Error updating deployment (%s/%s)  Error: %s", deployment.GetNamespace(), deployment.GetName(), err)
	}
	return err
}

// A deployment will be scaled down to 0 only if it's passed its cooldown period
// or if LastActiveTime is nil
func (h *ScaleHandler) scaleDeploymentToZero(deployment *apps_v1.Deployment, scaledObject *keda_v1alpha1.ScaledObject) {
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
			log.Infof("Successfully scaled deployment (%s/%s) to 0 replicas", deployment.GetNamespace(), deployment.GetName())
		}
	} else {
		log.Debugf("scaledObject (%s/%s) cooling down. Last active time %v, cooldownPeriod %d",
			scaledObject.GetNamespace(),
			scaledObject.GetName(),
			scaledObject.Status.LastActiveTime,
			cooldownPeriod)
	}
}
func (h *ScaleHandler) scaleDeploymentFromZero(deployment *apps_v1.Deployment, scaledObject *keda_v1alpha1.ScaledObject) {
	currentReplicas := *deployment.Spec.Replicas
	if scaledObject.Spec.MinReplicaCount != nil && *scaledObject.Spec.MinReplicaCount > 0 {
		deployment.Spec.Replicas = scaledObject.Spec.MinReplicaCount
	} else {
		*deployment.Spec.Replicas = 1
	}

	err := h.updateDeployment(deployment)

	if err == nil {
		log.Infof("Successfully updated deployment (%s/%s) from %d to %d replicas",
			deployment.GetNamespace(),
			deployment.GetName(),
			currentReplicas,
			*deployment.Spec.Replicas)

		// Scale was successful. Update lastScaleTime and lastActiveTime on the scaledObject
		now := meta_v1.Now()
		scaledObject.Status.LastActiveTime = &now
		h.updateScaledObject(scaledObject)
	}
}

func (h *ScaleHandler) resolveDeploymentEnv(deployment *apps_v1.Deployment, containerName string) (map[string]string, error) {
	deploymentKey, err := cache.MetaNamespaceKeyFunc(deployment)
	if err != nil {
		return nil, err
	}

	if len(deployment.Spec.Template.Spec.Containers) < 1 {
		return nil, fmt.Errorf("Deployment (%s) doesn't have containers", deploymentKey)
	}

	var container core_v1.Container

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
