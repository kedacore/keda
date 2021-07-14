package executor

import (
	"context"
	"time"

	"github.com/kedacore/keda/v2/pkg/eventreason"
	corev1 "k8s.io/api/core/v1"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
)

func (e *scaleExecutor) RequestScale(ctx context.Context, scaledObject *kedav1alpha1.ScaledObject, isActive bool, isError bool) {
	logger := e.logger.WithValues("scaledobject.Name", scaledObject.Name,
		"scaledObject.Namespace", scaledObject.Namespace,
		"scaleTarget.Name", scaledObject.Spec.ScaleTargetRef.Name)

	// Get the current replica count. As a special case, Deployments and StatefulSets fetch directly from the object so they can use the informer cache
	// to reduce API calls. Everything else uses the scale subresource.
	var currentScale *autoscalingv1.Scale
	var currentReplicas int32
	targetName := scaledObject.Spec.ScaleTargetRef.Name
	targetGVKR := scaledObject.Status.ScaleTargetGVKR
	switch {
	case targetGVKR.Group == "apps" && targetGVKR.Kind == "Deployment":
		deployment := &appsv1.Deployment{}
		err := e.client.Get(ctx, client.ObjectKey{Name: targetName, Namespace: scaledObject.Namespace}, deployment)
		if err != nil {
			logger.Error(err, "Error getting information on the current Scale (ie. replicas count) on the scaleTarget")
			return
		}
		currentReplicas = *deployment.Spec.Replicas
	case targetGVKR.Group == "apps" && targetGVKR.Kind == "StatefulSet":
		statefulSet := &appsv1.StatefulSet{}
		err := e.client.Get(ctx, client.ObjectKey{Name: targetName, Namespace: scaledObject.Namespace}, statefulSet)
		if err != nil {
			logger.Error(err, "Error getting information on the current Scale (ie. replicas count) on the scaleTarget")
			return
		}
		currentReplicas = *statefulSet.Spec.Replicas
	default:
		var err error
		currentScale, err = e.getScaleTargetScale(ctx, scaledObject)
		if err != nil {
			logger.Error(err, "Error getting information on the current Scale (ie. replicas count) on the scaleTarget")
			return
		}
		currentReplicas = currentScale.Spec.Replicas
	}

	// if scaledObject.Spec.MinReplicaCount is not set, then set the default value (0)
	minReplicas := int32(0)
	if scaledObject.Spec.MinReplicaCount != nil {
		minReplicas = *scaledObject.Spec.MinReplicaCount
	}

	if isActive {
		switch {
		case scaledObject.Spec.IdleReplicaCount != nil && currentReplicas < minReplicas,
			// triggers are active, Idle Replicas mode is enabled
			// AND
			// replica count is less then minimum replica count

			currentReplicas == 0:
			// triggers are active
			// AND
			// replica count is equal to 0

			// Scale the ScaleTarget up
			e.scaleFromZeroOrIdle(ctx, logger, scaledObject, currentScale)
		default:
			// triggers are active, but we didn't need to scale (replica count > 0)

			// update LastActiveTime to now
			err := e.updateLastActiveTime(ctx, logger, scaledObject)
			if err != nil {
				logger.Error(err, "Error updating last active time")
				return
			}
		}
	} else {
		// isActive == false
		switch {
		case isError && scaledObject.Spec.Fallback != nil && scaledObject.Spec.Fallback.Replicas != 0:
			// there are no active triggers, but a scaler responded with an error
			// AND
			// there is a fallback replicas count defined

			// Scale to the fallback replicas count
			e.doFallbackScaling(ctx, scaledObject, currentScale, logger, currentReplicas)
		case scaledObject.Spec.IdleReplicaCount != nil && currentReplicas > *scaledObject.Spec.IdleReplicaCount,
			// there are no active triggers, Idle Replicas mode is enabled
			// AND
			// current replicas count is greater than Idle Replicas count

			currentReplicas > 0 && minReplicas == 0:
			// there are no active triggers, but the ScaleTarget has replicas
			// AND
			// there is no minimum configured or minimum is set to ZERO

			// Try to scale the deployment down, HPA will handle other scale down operations
			e.scaleToZeroOrIdle(ctx, logger, scaledObject, currentScale)
		case currentReplicas < minReplicas && scaledObject.Spec.IdleReplicaCount == nil:
			// there are no active triggers
			// AND
			// ScaleTarget replicas count is less than minimum replica count specified in ScaledObject
			// AND
			// Idle Replicas mode is disabled

			// ScaleTarget replicas count to correct value
			_, err := e.updateScaleOnScaleTarget(ctx, scaledObject, currentScale, *scaledObject.Spec.MinReplicaCount)
			if err == nil {
				logger.Info("Successfully set ScaleTarget replicas count to ScaledObject minReplicaCount",
					"Original Replicas Count", currentReplicas,
					"New Replicas Count", *scaledObject.Spec.MinReplicaCount)
			}
		default:
			// there are no active triggers
			// AND
			// nothing needs to be done (eg. deployment is scaled down)
			logger.V(1).Info("ScaleTarget no change")
		}
	}

	condition := scaledObject.Status.Conditions.GetActiveCondition()
	if condition.IsUnknown() || condition.IsTrue() != isActive {
		if isActive {
			if err := e.setActiveCondition(ctx, logger, scaledObject, metav1.ConditionTrue, "ScalerActive", "Scaling is performed because triggers are active"); err != nil {
				logger.Error(err, "Error setting active condition when triggers are active")
				return
			}
		} else {
			if err := e.setActiveCondition(ctx, logger, scaledObject, metav1.ConditionFalse, "ScalerNotActive", "Scaling is not performed because triggers are not active"); err != nil {
				logger.Error(err, "Error setting active condition when triggers are not active")
				return
			}
		}
	}
}

func (e *scaleExecutor) doFallbackScaling(ctx context.Context, scaledObject *kedav1alpha1.ScaledObject, currentScale *autoscalingv1.Scale, logger logr.Logger, currentReplicas int32) {
	_, err := e.updateScaleOnScaleTarget(ctx, scaledObject, currentScale, scaledObject.Spec.Fallback.Replicas)
	if err == nil {
		logger.Info("Successfully set ScaleTarget replicas count to ScaledObject fallback.replicas",
			"Original Replicas Count", currentReplicas,
			"New Replicas Count", scaledObject.Spec.Fallback.Replicas)
	}
	if e := e.setFallbackCondition(ctx, logger, scaledObject, metav1.ConditionTrue, "FallbackExists", "At least one trigger is falling back on this scaled object"); e != nil {
		logger.Error(e, "Error setting fallback condition")
	}
}

// An object will be scaled down to 0 only if it's passed its cooldown period
// or if LastActiveTime is nil
func (e *scaleExecutor) scaleToZeroOrIdle(ctx context.Context, logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject, scale *autoscalingv1.Scale) {
	var cooldownPeriod time.Duration

	if scaledObject.Spec.CooldownPeriod != nil {
		cooldownPeriod = time.Second * time.Duration(*scaledObject.Spec.CooldownPeriod)
	} else {
		cooldownPeriod = time.Second * time.Duration(defaultCooldownPeriod)
	}

	// LastActiveTime can be nil if the ScaleTarget was scaled outside of KEDA.
	// In this case we will ignore the cooldown period and scale it down
	if scaledObject.Status.LastActiveTime == nil ||
		scaledObject.Status.LastActiveTime.Add(cooldownPeriod).Before(time.Now()) {
		// or last time a trigger was active was > cooldown period, so scale down.

		idleValue, scaleToReplicas := getIdleOrMinimumReplicaCount(scaledObject)

		currentReplicas, err := e.updateScaleOnScaleTarget(ctx, scaledObject, scale, scaleToReplicas)
		if err == nil {
			msg := "Successfully set ScaleTarget replicas count to ScaledObject"
			if idleValue {
				msg += " idleReplicaCount"
			} else {
				msg += " minReplicaCount"
			}
			logger.Info(msg, "Original Replicas Count", currentReplicas, "New Replicas Count", scaleToReplicas)

			e.recorder.Eventf(scaledObject, corev1.EventTypeNormal, eventreason.KEDAScaleTargetDeactivated,
				"Deactivated %s %s/%s from %d to %d", scaledObject.Status.ScaleTargetKind, scaledObject.Namespace, scaledObject.Spec.ScaleTargetRef.Name, currentReplicas, scaleToReplicas)
			if err := e.setActiveCondition(ctx, logger, scaledObject, metav1.ConditionFalse, "ScalerNotActive", "Scaling is not performed because triggers are not active"); err != nil {
				logger.Error(err, "Error in setting active condition")
				return
			}
		} else {
			e.recorder.Eventf(scaledObject, corev1.EventTypeWarning, eventreason.KEDAScaleTargetDeactivationFailed,
				"Failed to deactivated %s %s/%s", scaledObject.Status.ScaleTargetKind, scaledObject.Namespace, scaledObject.Spec.ScaleTargetRef.Name, currentReplicas, scaleToReplicas)
		}
	} else {
		logger.V(1).Info("ScaleTarget cooling down",
			"LastActiveTime", scaledObject.Status.LastActiveTime,
			"CoolDownPeriod", cooldownPeriod)

		activeCondition := scaledObject.Status.Conditions.GetActiveCondition()
		if !activeCondition.IsFalse() || activeCondition.Reason != "ScalerCooldown" {
			if err := e.setActiveCondition(ctx, logger, scaledObject, metav1.ConditionFalse, "ScalerCooldown", "Scaler cooling down because triggers are not active"); err != nil {
				logger.Error(err, "Error in setting active condition")
				return
			}
		}
	}
}

func (e *scaleExecutor) scaleFromZeroOrIdle(ctx context.Context, logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject, scale *autoscalingv1.Scale) {
	var replicas int32
	if scaledObject.Spec.MinReplicaCount != nil && *scaledObject.Spec.MinReplicaCount > 0 {
		replicas = *scaledObject.Spec.MinReplicaCount
	} else {
		replicas = 1
	}

	currentReplicas, err := e.updateScaleOnScaleTarget(ctx, scaledObject, scale, replicas)

	if err == nil {
		logger.Info("Successfully updated ScaleTarget",
			"Original Replicas Count", currentReplicas,
			"New Replicas Count", replicas)
		e.recorder.Eventf(scaledObject, corev1.EventTypeNormal, eventreason.KEDAScaleTargetActivated, "Scaled %s %s/%s from %d to %d", scaledObject.Status.ScaleTargetKind, scaledObject.Namespace, scaledObject.Spec.ScaleTargetRef.Name, currentReplicas, replicas)

		// Scale was successful. Update lastScaleTime and lastActiveTime on the scaledObject
		if err := e.updateLastActiveTime(ctx, logger, scaledObject); err != nil {
			logger.Error(err, "Error in Updating lastScaleTime and lastActiveTime on the scaledObject")
			return
		}
	} else {
		e.recorder.Eventf(scaledObject, corev1.EventTypeWarning, eventreason.KEDAScaleTargetActivationFailed, "Failed to scaled %s %s/%s from %d to %d", scaledObject.Status.ScaleTargetKind, scaledObject.Namespace, scaledObject.Spec.ScaleTargetRef.Name, currentReplicas, replicas)
	}
}

func (e *scaleExecutor) getScaleTargetScale(ctx context.Context, scaledObject *kedav1alpha1.ScaledObject) (*autoscalingv1.Scale, error) {
	return e.scaleClient.Scales(scaledObject.Namespace).Get(ctx, scaledObject.Status.ScaleTargetGVKR.GroupResource(), scaledObject.Spec.ScaleTargetRef.Name, metav1.GetOptions{})
}

func (e *scaleExecutor) updateScaleOnScaleTarget(ctx context.Context, scaledObject *kedav1alpha1.ScaledObject, scale *autoscalingv1.Scale, replicas int32) (int32, error) {
	if scale == nil {
		// Wasn't retrieved earlier, grab it now.
		var err error
		scale, err = e.getScaleTargetScale(ctx, scaledObject)
		if err != nil {
			return -1, err
		}
	}

	// Update with requested repliacs.
	currentReplicas := scale.Spec.Replicas
	scale.Spec.Replicas = replicas

	_, err := e.scaleClient.Scales(scaledObject.Namespace).Update(ctx, scaledObject.Status.ScaleTargetGVKR.GroupResource(), scale, metav1.UpdateOptions{})
	return currentReplicas, err
}

// getIdleOrMinimumReplicaCount returns true if the second value returned is from IdleReplicaCount
// it returns false if it is from MinReplicaCount followed by the actual value
func getIdleOrMinimumReplicaCount(scaledObject *kedav1alpha1.ScaledObject) (bool, int32) {
	if scaledObject.Spec.IdleReplicaCount != nil {
		return true, *scaledObject.Spec.IdleReplicaCount
	}

	if scaledObject.Spec.MinReplicaCount == nil {
		return false, 0
	}

	return false, *scaledObject.Spec.MinReplicaCount
}
