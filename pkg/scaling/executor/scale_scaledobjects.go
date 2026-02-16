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
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/kedacore/keda/v2/pkg/common/action"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/eventreason"
	"github.com/kedacore/keda/v2/pkg/scaling/resolver"
	kedastatus "github.com/kedacore/keda/v2/pkg/status"
)

func (e *scaleExecutor) RequestScale(ctx context.Context, scaledObject *kedav1alpha1.ScaledObject, isActive bool, isError bool, options *ScaleExecutorOptions) {
	logger := e.logger.WithValues("scaledobject.Name", scaledObject.Name,
		"scaledObject.Namespace", scaledObject.Namespace,
		"scaleTarget.Name", scaledObject.Spec.ScaleTargetRef.Name)
	var currentScale *autoscalingv1.Scale
	var currentReplicas int32
	// Get the current replica count
	currentReplicas, err := resolver.GetCurrentReplicas(ctx, e.client, e.scaleClient, scaledObject)
	if err != nil {
		logger.Error(err, "Error getting information on the current Scale")
		return
	}
	// if the ScaledObject's triggers aren't in the error state,
	// but ScaledObject.Status.ReadyCondition is set not set to 'true' -> set it back to 'true'
	readyCondition := scaledObject.Status.Conditions.GetReadyCondition()
	if !isError && !readyCondition.IsTrue() {
		if err := e.setReadyCondition(ctx, logger, scaledObject, metav1.ConditionTrue,
			kedav1alpha1.ScaledObjectConditionReadySuccessReason, kedav1alpha1.ScaledObjectConditionReadySuccessMessage); err != nil {
			logger.Error(err, "error setting ready condition")
		}
	}

	// Check if we are paused, and if we are then update the scale to the desired count.
	pausedCount, err := GetPausedReplicaCount(scaledObject)
	if err != nil {
		if err := e.setReadyCondition(ctx, logger, scaledObject, metav1.ConditionFalse,
			kedav1alpha1.ScaledObjectConditionReadySuccessReason, kedav1alpha1.ScaledObjectConditionReadySuccessMessage); err != nil {
			logger.Error(err, "error setting ready condition")
		}
		logger.Error(err, "error getting the paused replica count on the current ScaledObject.")
		return
	}
	status := scaledObject.Status.DeepCopy()
	if pausedCount != nil {
		// Scale the target to the paused replica count
		if *pausedCount != currentReplicas {
			_, err := e.updateScaleOnScaleTarget(ctx, scaledObject, currentScale, *pausedCount)
			if err != nil {
				logger.Error(err, "error scaling target to paused replicas count", "paused replicas", *pausedCount)
				if err := e.setReadyCondition(ctx, logger, scaledObject, metav1.ConditionUnknown,
					kedav1alpha1.ScaledObjectConditionReadySuccessReason, kedav1alpha1.ScaledObjectConditionReadySuccessMessage); err != nil {
					logger.Error(err, "error setting ready condition")
				}
				return
			}
		}
		if *pausedCount != currentReplicas || status.PausedReplicaCount == nil {
			status.PausedReplicaCount = pausedCount
			err = kedastatus.UpdateScaledObjectStatus(ctx, e.client, logger, scaledObject, status)
			if err != nil {
				logger.Error(err, "error updating status paused replica count")
				return
			}
			logger.Info("Successfully scaled target to paused replicas count", "paused replicas", *pausedCount)
		}
		return
	}

	// if scaledObject.Spec.MinReplicaCount is not set, then set the default value (0)
	minReplicas := int32(0)
	if scaledObject.Spec.MinReplicaCount != nil {
		minReplicas = *scaledObject.Spec.MinReplicaCount
	}

	if isActive || scaledObject.NeedToForceActivation() {
		// A scale target is active if triggers are active or if activation is being forced via annotation
		switch {
		case scaledObject.Spec.IdleReplicaCount != nil && currentReplicas < minReplicas,
			// triggers are active, Idle Replicas mode is enabled
			// AND
			// replica count is less than minimum replica count

			currentReplicas == 0:
			// triggers are active
			// AND
			// replica count is equal to 0

			// Scale the ScaleTarget up
			e.scaleFromZeroOrIdle(ctx, logger, scaledObject, currentScale, options.ActiveTriggers)
		case isError:
			// some triggers are active, but some responded with error

			// Set ScaledObject.Status.ReadyCondition to Unknown
			msg := "Some triggers defined in ScaledObject are not working correctly"
			logger.V(1).Info(msg)
			if !readyCondition.IsUnknown() {
				if err := e.setReadyCondition(ctx, logger, scaledObject, metav1.ConditionUnknown, "PartialTriggerError", msg); err != nil {
					logger.Error(err, "error setting ready condition")
				}
			}
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
			// We need to have this switch case even if just for logging.
			// Otherwise, if we have `minReplicas=zero`, we will fall into the third case expression,
			// which will scale the target to 0. Scaling the target to 0 means the HPA will not scale it to fallback.replicas
			// after fallback.failureThreshold has passed because of what's described here:
			// https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/#implicit-maintenance-mode-deactivation
			logger.V(1).Info("ScaleTarget will fallback to Fallback.Replicas after Fallback.FailureThreshold")
		case isError && scaledObject.Spec.Fallback == nil:
			// there are no active triggers, but a scaler responded with an error
			// AND
			// there is not a fallback replicas count defined

			// Set ScaledObject.Status.ReadyCondition to false
			msg := "Triggers defined in ScaledObject are not working correctly"
			logger.V(1).Info(msg)
			if !readyCondition.IsFalse() {
				if err := e.setReadyCondition(ctx, logger, scaledObject, metav1.ConditionFalse, "TriggerError", msg); err != nil {
					logger.Error(err, "error setting ready condition")
				}
			}
		case scaledObject.Spec.IdleReplicaCount != nil && currentReplicas > *scaledObject.Spec.IdleReplicaCount,
			// there are no active triggers, Idle Replicas mode is enabled
			// AND
			// current replicas count is greater than Idle Replicas count

			currentReplicas > 0 && minReplicas == 0:
			// there are no active triggers, but the ScaleTarget has replicas
			// AND
			// there is no minimum configured or minimum is set to ZERO

			// Try to scale the deployment down, HPA will handle other scale in operations
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

	activeTriggers := []string{}
	if options != nil {
		activeTriggers = options.ActiveTriggers
	}

	condition := scaledObject.Status.Conditions.GetActiveCondition()
	if condition.IsUnknown() || condition.IsTrue() != (isActive || scaledObject.NeedToForceActivation()) {
		switch {
		case isActive:
			// Scaler is active because of metrics
			if err := e.setActiveCondition(ctx, logger, scaledObject, metav1.ConditionTrue, "ScalerActive", "Scaling is performed because triggers are active"); err != nil {
				logger.Error(err, "Error setting active condition when triggers are active")
				return
			}
		case scaledObject.NeedToForceActivation():
			// Annotation is set to force scaler to activate
			if err := e.setActiveCondition(ctx, logger, scaledObject, metav1.ConditionTrue, "ScalerActive", "Scaling is performed because activation is being forced by annotation"); err != nil {
				logger.Error(err, "Error setting active condition when activation is forced")
				return
			}
		default:
			if err := e.setActiveCondition(ctx, logger, scaledObject, metav1.ConditionFalse, "ScalerNotActive", "Scaling is not performed because triggers are not active"); err != nil {
				logger.Error(err, "Error setting active condition when triggers are not active")
				return
			}
		}
	}

	// Update triggers activity if individual trigger states have changed
	if err := e.updateTriggersActivity(ctx, logger, scaledObject, activeTriggers); err != nil {
		logger.Error(err, "Error updating triggers activity")
		return
	}
}

// An object will be scaled down to 0 only if it's passed its cooldown period
// or if LastActiveTime is nil and scale in is not paused
func (e *scaleExecutor) scaleToZeroOrIdle(ctx context.Context, logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject, scale *autoscalingv1.Scale) {
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

	// If the ScaledObject was just created,CreationTimestamp is zero, set the CreationTimestamp to now
	if scaledObject.CreationTimestamp.IsZero() {
		scaledObject.CreationTimestamp = metav1.NewTime(time.Now())
	}

	// LastActiveTime can be nil if the ScaleTarget was scaled outside of KEDA.
	// In this case we will ignore the cooldown period and scale it down
	if (scaledObject.Status.LastActiveTime == nil && scaledObject.CreationTimestamp.Add(initialCooldownPeriod).Before(time.Now())) || (scaledObject.Status.LastActiveTime != nil &&
		scaledObject.Status.LastActiveTime.Add(cooldownPeriod).Before(time.Now())) {
		// or last time a trigger was active was > cooldown period, so scale in.
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

			e.recorder.Eventf(scaledObject, nil, corev1.EventTypeNormal, eventreason.KEDAScaleTargetDeactivated, action.Deactivated,
				"Deactivated %s %s/%s from %d to %d", scaledObject.Status.ScaleTargetKind, scaledObject.Namespace, scaledObject.Spec.ScaleTargetRef.Name, currentReplicas, scaleToReplicas)
			if err := e.setActiveCondition(ctx, logger, scaledObject, metav1.ConditionFalse, "ScalerNotActive", "Scaling is not performed because triggers are not active"); err != nil {
				logger.Error(err, "Error in setting active condition")
				return
			}
		} else {
			e.recorder.Eventf(scaledObject, nil, corev1.EventTypeWarning, eventreason.KEDAScaleTargetDeactivationFailed, action.Failed,
				"Failed to deactivate %s %s/%s from %d to %d", scaledObject.Status.ScaleTargetKind, scaledObject.Namespace, scaledObject.Spec.ScaleTargetRef.Name, currentReplicas, scaleToReplicas)
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

func (e *scaleExecutor) scaleFromZeroOrIdle(ctx context.Context, logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject, scale *autoscalingv1.Scale, activeTriggers []string) {
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

	currentReplicas, err := e.updateScaleOnScaleTarget(ctx, scaledObject, scale, replicas)

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

		e.recorder.Eventf(
			scaledObject,
			nil,
			corev1.EventTypeNormal,
			eventreason.KEDAScaleTargetActivated,
			action.Activated,
			eventMessage,
		)

		// Scale was successful. Update lastScaleTime and lastActiveTime on the scaledObject
		if err := e.updateLastActiveTime(ctx, logger, scaledObject); err != nil {
			logger.Error(err, "Error in Updating lastScaleTime and lastActiveTime on the scaledObject")
			return
		}
	} else {
		e.recorder.Eventf(scaledObject, nil, corev1.EventTypeWarning, eventreason.KEDAScaleTargetActivationFailed, action.Failed, "Failed to scaled %s %s/%s from %d to %d", scaledObject.Status.ScaleTargetKind, scaledObject.Namespace, scaledObject.Spec.ScaleTargetRef.Name, currentReplicas, replicas)
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

	// Update with requested replicas.
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

// GetPausedReplicaCount returns the paused replica count of the ScaledObject.
// If not paused, it returns nil.
func GetPausedReplicaCount(scaledObject *kedav1alpha1.ScaledObject) (*int32, error) {
	if scaledObject.Annotations != nil {
		if val, ok := scaledObject.Annotations[kedav1alpha1.PausedReplicasAnnotation]; ok {
			conv, err := strconv.ParseInt(val, 10, 32)
			if err != nil {
				return nil, err
			}
			count := int32(conv)
			return &count, nil
		}
	}
	return nil, nil
}
