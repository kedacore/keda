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

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/scale"
	"k8s.io/client-go/tools/record"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	kedastatus "github.com/kedacore/keda/v2/pkg/status"
)

const (
	// Default (initial) cooldown period for a ScaleTarget if no cooldownPeriod is defined on the scaledObject
	defaultInitialCooldownPeriod = 0      // 0 seconds
	defaultCooldownPeriod        = 5 * 60 // 5 minutes
)

// ScaleExecutor contains methods RequestJobScale and RequestScale
type ScaleExecutor interface {
	RequestJobScale(ctx context.Context, scaledJob *kedav1alpha1.ScaledJob, isActive bool, isError bool, scaleTo int64, maxScale int64, options *ScaleExecutorOptions)
	RequestScale(ctx context.Context, scaledObject *kedav1alpha1.ScaledObject, isActive bool, isError bool, options *ScaleExecutorOptions)
}

// ScaleExecutorOptions contains the optional parameters for the RequestScale method.
type ScaleExecutorOptions struct {
	ActiveTriggers []string
}

type scaleExecutor struct {
	client           runtimeclient.Client
	scaleClient      scale.ScalesGetter
	reconcilerScheme *runtime.Scheme
	logger           logr.Logger
	recorder         record.EventRecorder
}

// NewScaleExecutor creates a ScaleExecutor object
func NewScaleExecutor(client runtimeclient.Client, scaleClient scale.ScalesGetter, reconcilerScheme *runtime.Scheme, recorder record.EventRecorder) ScaleExecutor {
	return &scaleExecutor{
		client:           client,
		scaleClient:      scaleClient,
		reconcilerScheme: reconcilerScheme,
		logger:           logf.Log.WithName("scaleexecutor"),
		recorder:         recorder,
	}
}

func (e *scaleExecutor) updateLastActiveTime(ctx context.Context, logger logr.Logger, object interface{}) error {
	now := metav1.Now()
	transform := func(runtimeObj runtimeclient.Object, target interface{}) error {
		now, ok := target.(metav1.Time)
		if !ok {
			return fmt.Errorf("transform target is not metav1.Time type %v", target)
		}
		switch obj := runtimeObj.(type) {
		case *kedav1alpha1.ScaledObject:
			obj.Status.LastActiveTime = &now
		case *kedav1alpha1.ScaledJob:
			obj.Status.LastActiveTime = &now
		default:
		}
		return nil
	}
	return kedastatus.TransformObject(ctx, e.client, logger, object, now, transform)
}

func (e *scaleExecutor) setCondition(ctx context.Context, logger logr.Logger, object interface{}, status metav1.ConditionStatus, reason string, message string, setCondition func(kedav1alpha1.Conditions, metav1.ConditionStatus, string, string)) error {
	type transformStruct struct {
		status  metav1.ConditionStatus
		reason  string
		message string
	}
	transform := func(runtimeObj runtimeclient.Object, target interface{}) error {
		transformObj := target.(*transformStruct)
		switch obj := runtimeObj.(type) {
		case *kedav1alpha1.ScaledObject:
			setCondition(obj.Status.Conditions, transformObj.status, transformObj.reason, transformObj.message)
		case *kedav1alpha1.ScaledJob:
			setCondition(obj.Status.Conditions, transformObj.status, transformObj.reason, transformObj.message)
		default:
		}
		return nil
	}
	target := transformStruct{
		status:  status,
		reason:  reason,
		message: message,
	}
	return kedastatus.TransformObject(ctx, e.client, logger, object, &target, transform)
}

func (e *scaleExecutor) setReadyCondition(ctx context.Context, logger logr.Logger, object interface{}, status metav1.ConditionStatus, reason string, message string) error {
	active := func(conditions kedav1alpha1.Conditions, status metav1.ConditionStatus, reason string, message string) {
		conditions.SetReadyCondition(status, reason, message)
	}
	return e.setCondition(ctx, logger, object, status, reason, message, active)
}

func (e *scaleExecutor) setActiveCondition(ctx context.Context, logger logr.Logger, object interface{}, status metav1.ConditionStatus, reason string, message string) error {
	active := func(conditions kedav1alpha1.Conditions, status metav1.ConditionStatus, reason string, message string) {
		conditions.SetActiveCondition(status, reason, message)
	}
	return e.setCondition(ctx, logger, object, status, reason, message, active)
}

func (e *scaleExecutor) setFallbackCondition(ctx context.Context, logger logr.Logger, object interface{}, status metav1.ConditionStatus, reason string, message string) error {
	fb := func(conditions kedav1alpha1.Conditions, status metav1.ConditionStatus, reason string, message string) {
		conditions.SetFallbackCondition(status, reason, message)
	}
	return e.setCondition(ctx, logger, object, status, reason, message, fb)
}

func (e *scaleExecutor) updateTriggersActivity(ctx context.Context, logger logr.Logger, object interface{}, activeTriggers []string) error {
	// Get the current status to check if update is needed
	var triggersActivity map[string]kedav1alpha1.TriggerActivityStatus
	var allTriggerNames []string

	switch obj := object.(type) {
	case *kedav1alpha1.ScaledObject:
		triggersActivity = obj.Status.TriggersActivity
		allTriggerNames = obj.Status.ExternalMetricNames
	case *kedav1alpha1.ScaledJob:
		triggersActivity = obj.Status.TriggersActivity
		allTriggerNames = obj.Status.ExternalMetricNames
	default:
		return fmt.Errorf("unsupported object type %T for updateTriggersActivity", object)
	}

	// Create set of active triggers for quick lookup
	activeSet := make(map[string]bool, len(activeTriggers))
	for _, trigger := range activeTriggers {
		activeSet[trigger] = true
	}

	// Create set of current triggers for efficient lookup
	currentTriggerSet := make(map[string]bool, len(allTriggerNames))
	for _, trigger := range allTriggerNames {
		currentTriggerSet[trigger] = true
	}

	// Check if any trigger's state has changed
	needsUpdate := false

	// Check if the map needs initialization
	if triggersActivity == nil {
		needsUpdate = len(allTriggerNames) > 0
	} else {
		// Check for state changes in existing triggers
		for _, trigger := range allTriggerNames {
			currentStatus, exists := triggersActivity[trigger]
			if !exists || currentStatus.IsActive != activeSet[trigger] {
				needsUpdate = true
				break
			}
		}

		// Check if there are stale triggers to remove
		if !needsUpdate {
			for triggerName := range triggersActivity {
				if !currentTriggerSet[triggerName] {
					needsUpdate = true
					break
				}
			}
		}
	}

	// If no update needed, return early without calling TransformObject
	if !needsUpdate {
		return nil
	}

	// Update is needed, perform the transformation
	transform := func(runtimeObj runtimeclient.Object, _ interface{}) error {
		var triggersActivity *map[string]kedav1alpha1.TriggerActivityStatus
		var allTriggerNames []string

		switch obj := runtimeObj.(type) {
		case *kedav1alpha1.ScaledObject:
			triggersActivity = &obj.Status.TriggersActivity
			allTriggerNames = obj.Status.ExternalMetricNames
		case *kedav1alpha1.ScaledJob:
			triggersActivity = &obj.Status.TriggersActivity
			allTriggerNames = obj.Status.ExternalMetricNames
		default:
			return fmt.Errorf("unsupported object type %T for updateTriggersActivity", runtimeObj)
		}

		// Initialize if needed
		if *triggersActivity == nil {
			*triggersActivity = make(map[string]kedav1alpha1.TriggerActivityStatus, len(allTriggerNames))
		}

		// Update IsActive status for all triggers
		for _, trigger := range allTriggerNames {
			(*triggersActivity)[trigger] = kedav1alpha1.TriggerActivityStatus{
				IsActive: activeSet[trigger],
			}
		}

		// Remove stale triggers
		for triggerName := range *triggersActivity {
			if !currentTriggerSet[triggerName] {
				delete(*triggersActivity, triggerName)
			}
		}

		return nil
	}

	return kedastatus.TransformObject(ctx, e.client, logger, object, nil, transform)
}
