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

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/scale"
	"k8s.io/client-go/tools/record"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

const (
	// Default (initial) cooldown period for a ScaleTarget if no cooldownPeriod is defined on the scaledObject
	defaultInitialCooldownPeriod = 0      // 0 seconds
	defaultCooldownPeriod        = 5 * 60 // 5 minutes
)

// ScaleResult contains the result of ScaleExecutor actions
type ScaleResult struct {
	Conditions       kedav1alpha1.Conditions
	PauseReplicas    *int32
	Error            error
	LastActiveTime   *metav1.Time
	TriggersActivity map[string]kedav1alpha1.TriggerActivityStatus
}

// ScaleExecutor contains methods RequestJobScale and RequestScale
type ScaleExecutor interface {
	RequestJobScale(ctx context.Context, scaledJob *kedav1alpha1.ScaledJob, isActive bool, isError bool, scaleTo int64, maxScale int64, options ScaleExecutorOptions) ScaleResult
	RequestScale(ctx context.Context, scaledObject *kedav1alpha1.ScaledObject, isActive bool, isError bool, options ScaleExecutorOptions) ScaleResult
}

// ScaleExecutorOptions contains the optional parameters for the RequestScale and RequestJobScale method.
type ScaleExecutorOptions struct {
	ActiveTriggers      []string
	ForPushScalerMetric string
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

// getTriggersActivity returns a map of trigger names to their activity status based on the provided active triggers and the triggers defined in the scaled object.
func getTriggersActivity(object kedav1alpha1.ScalableObject, options ScaleExecutorOptions) map[string]kedav1alpha1.TriggerActivityStatus {
	activeTriggers := options.ActiveTriggers
	pushScalerMetric := options.ForPushScalerMetric
	isPushScaler := pushScalerMetric != ""

	var allTriggerNames []string
	triggersActivity := make(map[string]kedav1alpha1.TriggerActivityStatus)

	if isPushScaler {
		// for push scaler, copy existing activity and update the specific metric
		for k, v := range object.GetStatusTriggersActivity() {
			triggersActivity[k] = v
		}
		allTriggerNames = []string{pushScalerMetric}
	} else {
		allTriggerNames = object.GetStatusExternalMetricNames()
	}

	activeSet := make(map[string]bool, len(activeTriggers))
	for _, trigger := range activeTriggers {
		activeSet[trigger] = true
	}
	for _, triggerName := range allTriggerNames {
		triggersActivity[triggerName] = kedav1alpha1.TriggerActivityStatus{IsActive: activeSet[triggerName]}
	}
	return triggersActivity
}
