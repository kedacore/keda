package executor

import (
	"context"
	"time"

	kedav1alpha1 "github.com/kedacore/keda/pkg/apis/keda/v1alpha1"
	"github.com/kedacore/keda/pkg/scalers"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
)

func (e *scaleExecutor) RequestScale(ctx context.Context, scalers []scalers.Scaler, scaledObject *kedav1alpha1.ScaledObject) {
	isActive := false
	for _, scaler := range scalers {
		defer scaler.Close()
		isTriggerActive, err := scaler.IsActive(ctx)

		if err != nil {
			e.logger.V(1).Info("Error getting scale decision", "Error", err)
			continue
		} else if isTriggerActive {
			isActive = true
			e.logger.V(1).Info("Scaler for scaledObject is active", "Scaler", scaler)
		}
	}

	currentScale, err := e.getScaleTargetScale(scaledObject)
	if err != nil {
		e.logger.Error(err, "Error getting Scale")
	}

	if currentScale.Spec.Replicas == 0 && isActive {
		// current replica count is 0, but there is an active trigger.
		// scale the ScaleTarget up
		e.scaleFromZero(ctx, scaledObject, currentScale)
	} else if !isActive &&
		currentScale.Spec.Replicas > 0 &&
		(scaledObject.Spec.MinReplicaCount == nil || *scaledObject.Spec.MinReplicaCount == 0) {
		// there are no active triggers, but the ScaleTarget has replicas.
		// AND
		// There is no minimum configured or minimum is set to ZERO. HPA will handles other scale down operations

		// Try to scale it down.
		e.scaleToZero(scaledObject, currentScale)
	} else if !isActive &&
		scaledObject.Spec.MinReplicaCount != nil &&
		currentScale.Spec.Replicas < *scaledObject.Spec.MinReplicaCount {
		// there are no active triggers
		// AND
		// ScaleTarget replicas count is less than minimum replica count specified in ScaledObject
		// Let's set ScaleTarget replicas count to correct value
		currentScale.Spec.Replicas = *scaledObject.Spec.MinReplicaCount

		err := e.updateScaleOnScaleTarget(scaledObject, currentScale)
		if err == nil {
			e.logger.Info("Successfully set ScaleTarget replicas count to ScaledObject minReplicaCount",
				"ScaleTarget.Name", scaledObject.Spec.ScaleTargetRef.Name,
				"ScaleTarget.Replicas", currentScale.Spec.Replicas)
		}
	} else if isActive {
		// triggers are active, but we didn't need to scale (replica count > 0)
		// Update LastActiveTime to now.
		e.updateLastActiveTime(ctx, scaledObject)
	} else {
		e.logger.V(1).Info("ScaleTarget no change", "ScaleTarget.Name", scaledObject.Spec.ScaleTargetRef.Name)
	}
}

// An object will be scaled down to 0 only if it's passed its cooldown period
// or if LastActiveTime is nil
func (e *scaleExecutor) scaleToZero(scaledObject *kedav1alpha1.ScaledObject, scale *autoscalingv1.Scale) {
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
		err := e.updateScaleOnScaleTarget(scaledObject, scale)
		if err == nil {
			e.logger.Info("Successfully scaled ScaleTarget to 0 replicas", "ScaleTarget.Name", scaledObject.Spec.ScaleTargetRef.Name)
		}
	} else {
		e.logger.V(1).Info("ScaleTarget cooling down",
			"ScaleTarget.Name", scaledObject.Spec.ScaleTargetRef.Name,
			"LastActiveTime", scaledObject.Status.LastActiveTime,
			"CoolDownPeriod", cooldownPeriod)
	}
}

func (e *scaleExecutor) scaleFromZero(ctx context.Context, scaledObject *kedav1alpha1.ScaledObject, scale *autoscalingv1.Scale) {
	currentReplicas := scale.Spec.Replicas
	if scaledObject.Spec.MinReplicaCount != nil && *scaledObject.Spec.MinReplicaCount > 0 {
		scale.Spec.Replicas = *scaledObject.Spec.MinReplicaCount
	} else {
		scale.Spec.Replicas = 1
	}

	err := e.updateScaleOnScaleTarget(scaledObject, scale)

	if err == nil {
		e.logger.Info("Successfully updated ScaleTarget",
			"ScaleTarget.Name", scaledObject.Spec.ScaleTargetRef.Name,
			"Original Replicas Count", currentReplicas,
			"New Replicas Count", scale.Spec.Replicas)

		// Scale was successful. Update lastScaleTime and lastActiveTime on the scaledObject
		e.updateLastActiveTime(ctx, scaledObject)
	}
}

func (e *scaleExecutor) getScaleTargetScale(scaledObject *kedav1alpha1.ScaledObject) (*autoscalingv1.Scale, error) {
	return (*e.scaleClient).Scales(scaledObject.Namespace).Get(scaledObject.Status.ScaleTargetGVKR.GroupResource(), scaledObject.Spec.ScaleTargetRef.Name)
}

func (e *scaleExecutor) updateScaleOnScaleTarget(scaledObject *kedav1alpha1.ScaledObject, scale *autoscalingv1.Scale) error {
	_, err := (*e.scaleClient).Scales(scaledObject.Namespace).Update(scaledObject.Status.ScaleTargetGVKR.GroupResource(), scale)
	return err
}
