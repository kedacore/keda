/*
Copyright 2021 The KEDA Authors

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

package executor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/eventreason"
	"github.com/kedacore/keda/v2/pkg/scaling/resolver"
)

func (e *scaleExecutor) RequestScale(ctx context.Context, scaledObject *kedav1alpha1.ScaledObject, isActive bool, isError bool, options ScaleExecutorOptions) ScaleResult {
	logger := e.logger.WithValues("scaledobject.Name", scaledObject.Name, "scaledObject.Namespace", scaledObject.Namespace, "scaleTarget.Name", scaledObject.Spec.ScaleTargetRef.Name)
	var currentReplicas int32
	result := ScaleResult{Conditions: kedav1alpha1.Conditions{}}
	result.Conditions.SetReadyCondition(metav1.ConditionTrue, kedav1alpha1.ScaledObjectConditionReadySuccessReason, kedav1alpha1.ScaledObjectConditionReadySuccessMessage)
	result.TriggersActivity = getTriggersActivity(scaledObject, options)

	// get the current replica count
	currentReplicas, err := resolver.GetCurrentReplicas(ctx, e.client, e.scaleClient, scaledObject)
	if err != nil {
		logger.Error(err, "Error getting current replicas count for ScaleTarget")
		result.Conditions.SetReadyCondition(metav1.ConditionFalse, "ErrorGettingCurrentReplicas", fmt.Sprintf("Error getting current replicas count for ScaleTarget: %v", err))
		result.Error = fmt.Errorf("error getting current replicas count for ScaleTarget: %w", err)
		return result
	}

	// Return early if paused to skip normal scaling logic
	if e.handlePaused(scaledObject, &result) {
		return result
	}
	// if scaledObject.Spec.MinReplicaCount is not set, then set the default value (0)
	minReplicas := int32(0)
	if scaledObject.Spec.MinReplicaCount != nil {
		minReplicas = *scaledObject.Spec.MinReplicaCount
	}

	if isActive || scaledObject.NeedToForceActivation() {
		if scaledObject.NeedToForceActivation() {
			result.Conditions.SetActiveCondition(metav1.ConditionTrue, "ScalerActive", "Scaling is performed because activation is being forced by annotation")
		} else {
			result.Conditions.SetActiveCondition(metav1.ConditionTrue, "ScalerActive", "Scaling is performed because triggers are active")
		}
		// a scale target is active if triggers are active or if activation is being forced via annotation
		switch {
		case scaledObject.Spec.IdleReplicaCount != nil && currentReplicas < minReplicas, currentReplicas == 0:
			// triggers are active, Idle Replicas mode is enabled AND replica count is less than minimum replica count => scale the ScaleTarget up
			e.scaleFromZeroOrIdle(ctx, logger, scaledObject, options.ActiveTriggers, &result)
		case isError:
			// some triggers are active, but some responded with error
			result.Conditions.SetReadyCondition(metav1.ConditionUnknown, "PartialTriggerError", "Some triggers defined in ScaledObject are not working correctly")
			logger.V(1).Info("Some triggers defined in ScaledObject are not working correctly")
		default:
			// triggers are active, but we didn't need to scale (replica count > 0)
			result.LastActiveTime = &metav1.Time{Time: time.Now()}
		}
	} else {
		// isActive == false
		result.Conditions.SetActiveCondition(metav1.ConditionFalse, "ScalerNotActive", "Scaling is not performed because triggers are not active")
		switch {
		case isError && scaledObject.Spec.Fallback != nil && scaledObject.Spec.Fallback.Replicas != 0:
			// We need to have this switch case even if just for logging.
			// Otherwise, if we have `minReplicas=zero`, we will fall into the third case expression,
			// which will scale the target to 0. Scaling the target to 0 means the HPA will not scale it to fallback.replicas
			// after fallback.failureThreshold has passed because of what's described here:
			// https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/#implicit-maintenance-mode-deactivation
			logger.V(1).Info("ScaleTarget will fallback to Fallback.Replicas after Fallback.FailureThreshold")
		case isError && scaledObject.Spec.Fallback == nil:
			// there are no active triggers, but a scaler responded with an error AND there is not a fallback replicas count defined
			msg := "Triggers defined in ScaledObject are not working correctly"
			result.Conditions.SetReadyCondition(metav1.ConditionFalse, "TriggerError", msg)
			logger.V(1).Info(msg)
		case scaledObject.Spec.IdleReplicaCount != nil && currentReplicas > *scaledObject.Spec.IdleReplicaCount, currentReplicas > 0 && minReplicas == 0:
			// there are no active triggers, Idle Replicas mode is enabled AND current replicas count is greater than Idle Replicas count
			// Try to scale the deployment down, HPA will handle other scale in operations
			e.scaleToZeroOrIdle(ctx, logger, scaledObject, &result)
		case scaledObject.Spec.IdleReplicaCount == nil && currentReplicas < minReplicas:
			// there are no active triggers AND ScaleTarget replicas count is less than minimum replica count specified in ScaledObject AND Idle Replicas mode is disabled
			// ScaleTarget replicas count to correct value
			_, err := e.updateScaleOnScaleTarget(ctx, scaledObject, *scaledObject.Spec.MinReplicaCount)
			if err == nil {
				msg := "Successfully set ScaleTarget replicas count to ScaledObject minReplicaCount"
				logger.Info(msg, "Original Replicas Count", currentReplicas, "New Replicas Count", *scaledObject.Spec.MinReplicaCount)
			}
		default:
			// there are no active triggers AND nothing needs to be done (eg. deployment is scaled down)
			logger.V(1).Info("ScaleTarget no change")
		}
	}
	return result
}

// An object will be scaled down to 0 only if it's passed its cooldown period
// or if LastActiveTime is nil and scale in is not paused
func (e *scaleExecutor) scaleToZeroOrIdle(ctx context.Context, logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject, result *ScaleResult) {
	if scaledObject.NeedToPauseScaleIn() {
		// The Pause Scale Down annotation is set so we should not scale down this target
		logger.Info("Pause Scale Down annotation set on ScaledObject, no scaling down on inactive trigger")
		return
	}

	var initialCooldownPeriod, cooldownPeriod time.Duration

	if scaledObject.Spec.InitialCooldownPeriod != nil {
		initialCooldownPeriod = time.Second * time.Duration(*scaledObject.Spec.InitialCooldownPeriod)
	} else {
		initialCooldownPeriod = time.Second * time.Duration(defaultInitialCooldownPeriod)
	}

	if scaledObject.Spec.CooldownPeriod != nil {
		cooldownPeriod = time.Second * time.Duration(*scaledObject.Spec.CooldownPeriod)
	} else {
		cooldownPeriod = time.Second * time.Duration(defaultCooldownPeriod)
	}

	// LastActiveTime can be nil if the ScaleTarget was scaled outside of KEDA.
	// In this case we will ignore the cooldown period and scale it down
	if (scaledObject.Status.LastActiveTime == nil && scaledObject.CreationTimestamp.Add(initialCooldownPeriod).Before(time.Now())) || (scaledObject.Status.LastActiveTime != nil &&
		scaledObject.Status.LastActiveTime.Add(cooldownPeriod).Before(time.Now())) {
		// or last time a trigger was active was > cooldown period, so scale in.
		idleValue, scaleToReplicas := getIdleOrMinimumReplicaCount(scaledObject)

		currentReplicas, err := e.updateScaleOnScaleTarget(ctx, scaledObject, scaleToReplicas)
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
			result.Conditions.SetActiveCondition(metav1.ConditionFalse, "ScalerNotActive", "Scaling is not performed because triggers are not active")
		} else {
			e.recorder.Eventf(scaledObject, corev1.EventTypeWarning, eventreason.KEDAScaleTargetDeactivationFailed,
				"Failed to deactivate %s %s/%s from %d to %d", scaledObject.Status.ScaleTargetKind, scaledObject.Namespace, scaledObject.Spec.ScaleTargetRef.Name, currentReplicas, scaleToReplicas)
		}
	} else {
		logger.V(1).Info("ScaleTarget cooling down", "LastActiveTime", scaledObject.Status.LastActiveTime, "CoolDownPeriod", cooldownPeriod)
		result.Conditions.SetActiveCondition(metav1.ConditionFalse, "ScalerCooldown", "Scaler cooling down because triggers are not active")
	}
}

func (e *scaleExecutor) scaleFromZeroOrIdle(ctx context.Context, logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject, activeTriggers []string, result *ScaleResult) {
	if scaledObject.NeedToPauseScaleOut() {
		// The Pause Scale Out annotation is set so we should not scale up (out) this target
		logger.Info("Pause Scale Out annotation set on ScaledObject, no scaling out on active trigger")
		return
	}

	var replicas int32
	if scaledObject.Spec.MinReplicaCount != nil && *scaledObject.Spec.MinReplicaCount > 0 {
		replicas = *scaledObject.Spec.MinReplicaCount
	} else {
		replicas = 1
	}

	currentReplicas, err := e.updateScaleOnScaleTarget(ctx, scaledObject, replicas)

	if err == nil {
		logger.Info("Successfully updated ScaleTarget",
			"Original Replicas Count", currentReplicas,
			"New Replicas Count", replicas)

		eventMessage := fmt.Sprintf(
			"Scaled %s %s/%s from %d to %d, triggered by %s",
			scaledObject.Status.ScaleTargetKind,
			scaledObject.Namespace,
			scaledObject.Spec.ScaleTargetRef.Name,
			currentReplicas,
			replicas,
			strings.Join(activeTriggers, ";"),
		)

		if scaledObject.NeedToForceActivation() {
			// If activation is caused by the force annotation, record a different event message
			eventMessage = fmt.Sprintf(
				"Scaled %s %s/%s from %d to %d, caused by forced activation annotation",
				scaledObject.Status.ScaleTargetKind,
				scaledObject.Namespace,
				scaledObject.Spec.ScaleTargetRef.Name,
				currentReplicas,
				replicas,
			)
		}

		e.recorder.Event(
			scaledObject,
			corev1.EventTypeNormal,
			eventreason.KEDAScaleTargetActivated,
			eventMessage,
		)

		// Scale was successful. Record lastActiveTime in the result for the handler to persist.
		result.LastActiveTime = &metav1.Time{Time: time.Now()}
	} else {
		e.recorder.Eventf(scaledObject, corev1.EventTypeWarning, eventreason.KEDAScaleTargetActivationFailed, "Failed to scale %s %s/%s from %d to %d", scaledObject.Status.ScaleTargetKind, scaledObject.Namespace, scaledObject.Spec.ScaleTargetRef.Name, currentReplicas, replicas)
	}
}

func (e *scaleExecutor) getScaleTargetScale(ctx context.Context, scaledObject *kedav1alpha1.ScaledObject) (*autoscalingv1.Scale, error) {
	return e.scaleClient.Scales(scaledObject.Namespace).Get(ctx, scaledObject.Status.ScaleTargetGVKR.GroupResource(), scaledObject.Spec.ScaleTargetRef.Name, metav1.GetOptions{})
}

func (e *scaleExecutor) updateScaleOnScaleTarget(ctx context.Context, scaledObject *kedav1alpha1.ScaledObject, replicas int32) (int32, error) {
	scale, err := e.getScaleTargetScale(ctx, scaledObject)
	if err != nil {
		return -1, err
	}

	// Update with requested replicas.
	currentReplicas := scale.Spec.Replicas
	scale.Spec.Replicas = replicas

	_, err = e.scaleClient.Scales(scaledObject.Namespace).Update(ctx, scaledObject.Status.ScaleTargetGVKR.GroupResource(), scale, metav1.UpdateOptions{})
	return currentReplicas, err
}

// handlePaused skips normal scaling logic while the ScaledObject is paused.
func (e *scaleExecutor) handlePaused(scaledObject *kedav1alpha1.ScaledObject, scaleResult *ScaleResult) bool {
	if scaledObject.NeedToBePausedByAnnotation() {
		scaleResult.Conditions.SetPausedCondition(metav1.ConditionTrue, "ScaledObjectPaused", "ScaledObject is paused")
		return true
	}
	return false
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
