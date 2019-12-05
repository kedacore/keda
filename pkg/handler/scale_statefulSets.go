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

func (h *ScaleHandler) scaleStatefulSet(statefulSet *appsv1.StatefulSet, scaledObject *kedav1alpha1.ScaledObject, isActive bool) {

	if *statefulSet.Spec.Replicas == 0 && isActive {
		// current replica count is 0, but there is an active trigger.
		// scale the deployment up
		h.scaleStatefulSetFromZero(statefulSet, scaledObject)
	} else if !isActive &&
		*statefulSet.Spec.Replicas > 0 &&
		(scaledObject.Spec.MinReplicaCount == nil || *scaledObject.Spec.MinReplicaCount == 0) {
		// there are no active triggers, but the deployment has replicas.
		// AND
		// There is no minimum configured or minimum is set to ZERO. HPA will handles other scale down operations

		// Try to scale it down.
		h.scaleStatefulSetToZero(statefulSet, scaledObject)
	} else if isActive {
		// triggers are active, but we didn't need to scale (replica count > 0)
		// Update LastActiveTime to now.
		now := metav1.Now()
		scaledObject.Status.LastActiveTime = &now
		h.updateScaledObjectStatus(scaledObject)
	} else {
		h.logger.V(1).Info("StatefulSet no change", "StatefulSet.Namespace", statefulSet.GetNamespace(), "StatefulSet.Name", statefulSet.GetName())
	}
}

func (h *ScaleHandler) updateStatefulSet(statefulSet *appsv1.StatefulSet) error {

	err := h.client.Update(context.TODO(), statefulSet)
	if err != nil {
		h.logger.Error(err, "Error updating statefulSet", "StatefulSet.Namespace", statefulSet.GetNamespace(), "StatefulSet.Name", statefulSet.GetName())
		return err
	}
	return nil
}

// A statefulSet will be scaled down to 0 only if it's passed its cooldown period
// or if LastActiveTime is nil
func (h *ScaleHandler) scaleStatefulSetToZero(statefulSet *appsv1.StatefulSet, scaledObject *kedav1alpha1.ScaledObject) {
	var cooldownPeriod time.Duration

	if scaledObject.Spec.CooldownPeriod != nil {
		cooldownPeriod = time.Second * time.Duration(*scaledObject.Spec.CooldownPeriod)
	} else {
		cooldownPeriod = time.Second * time.Duration(defaultCooldownPeriod)
	}

	// LastActiveTime can be nil if the statefulSet was scaled outside of Keda.
	// In this case we will ignore the cooldown period and scale it down
	if scaledObject.Status.LastActiveTime == nil ||
		scaledObject.Status.LastActiveTime.Add(cooldownPeriod).Before(time.Now()) {
		// or last time a trigger was active was > cooldown period, so scale down.
		*statefulSet.Spec.Replicas = 0
		err := h.updateStatefulSet(statefulSet)
		if err == nil {
			h.logger.Info("Successfully scaled statefulSet to 0 replicas", "StatefulSet.Namespace", statefulSet.GetNamespace(), "StatefulSet.Name", statefulSet.GetName())
		}
	} else {
		h.logger.V(1).Info("scaledObject cooling down",
			"LastActiveTime",
			scaledObject.Status.LastActiveTime,
			"CoolDownPeriod",
			cooldownPeriod)
	}
}

func (h *ScaleHandler) scaleStatefulSetFromZero(statefulSet *appsv1.StatefulSet, scaledObject *kedav1alpha1.ScaledObject) {
	currentReplicas := *statefulSet.Spec.Replicas
	if scaledObject.Spec.MinReplicaCount != nil && *scaledObject.Spec.MinReplicaCount > 0 {
		statefulSet.Spec.Replicas = scaledObject.Spec.MinReplicaCount
	} else {
		*statefulSet.Spec.Replicas = 1
	}

	err := h.updateStatefulSet(statefulSet)

	if err == nil {
		h.logger.Info("Successfully updated statefulSet", "StatefulSet.Namespace", statefulSet.GetNamespace(), "StatefulSet.Name", statefulSet.GetName(),
			"Original Replicas Count",
			currentReplicas,
			"New Replicas Count",
			*statefulSet.Spec.Replicas)

		// Scale was successful. Update lastScaleTime and lastActiveTime on the scaledObject
		now := metav1.Now()
		scaledObject.Status.LastActiveTime = &now
		h.updateScaledObjectStatus(scaledObject)
	}
}

func (h *ScaleHandler) resolveStatefulSetEnv(statefulSet *appsv1.StatefulSet, containerName string) (map[string]string, error) {
	statefulSetKey, err := cache.MetaNamespaceKeyFunc(statefulSet)
	if err != nil {
		return nil, err
	}

	if len(statefulSet.Spec.Template.Spec.Containers) < 1 {
		return nil, fmt.Errorf("StatefulSet (%s) doesn't have containers", statefulSetKey)
	}

	var container corev1.Container

	if containerName != "" {
		for _, c := range statefulSet.Spec.Template.Spec.Containers {
			if c.Name == containerName {
				container = c
				break
			}
		}

		if &container == nil {
			return nil, fmt.Errorf("Couldn't find container with name %s on statefulSet %s", containerName, statefulSet.GetName())
		}
	} else {
		container = statefulSet.Spec.Template.Spec.Containers[0]
	}

	return h.resolveEnv(&container, statefulSet.GetNamespace())
}

func (h *ScaleHandler) parseStatefulSetAuthRef(triggerAuthRef *kedav1alpha1.ScaledObjectAuthRef, scaledObject *kedav1alpha1.ScaledObject, statefulSet *appsv1.StatefulSet) (map[string]string, string) {
	return h.parseAuthRef(triggerAuthRef, scaledObject, func(name, containerName string) string {
		env, err := h.resolveStatefulSetEnv(statefulSet, containerName)
		if err != nil {
			return ""
		}
		return env[name]
	})
}
