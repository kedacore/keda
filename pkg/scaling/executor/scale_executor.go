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
)

const (
	// Default cooldown period for a ScaleTarget if no cooldownPeriod is defined on the scaledObject
	defaultCooldownPeriod = 5 * 60 // 5 minutes
)

// ScaleExecutor contains methods RequestJobScale and RequestScale
type ScaleExecutor interface {
	RequestJobScale(ctx context.Context, scaledJob *kedav1alpha1.ScaledJob, isActive bool, scaleTo int64, maxScale int64)
	RequestScale(ctx context.Context, scaledObject *kedav1alpha1.ScaledObject, isActive bool, isError bool)
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
	var patch runtimeclient.Patch

	now := metav1.Now()
	runtimeObj := object.(runtimeclient.Object)
	switch obj := runtimeObj.(type) {
	case *kedav1alpha1.ScaledObject:
		patch = runtimeclient.MergeFrom(obj.DeepCopy())
		obj.Status.LastActiveTime = &now
	case *kedav1alpha1.ScaledJob:
		patch = runtimeclient.MergeFrom(obj.DeepCopy())
		obj.Status.LastActiveTime = &now
	default:
		err := fmt.Errorf("unknown scalable object type %v", obj)
		logger.Error(err, "Failed to patch Objects Status")
		return err
	}

	err := e.client.Status().Patch(ctx, runtimeObj, patch)
	if err != nil {
		logger.Error(err, "Failed to patch Objects Status")
	}
	return err
}

func (e *scaleExecutor) setCondition(ctx context.Context, logger logr.Logger, object interface{}, status metav1.ConditionStatus, reason string, message string, setCondition func(kedav1alpha1.Conditions, metav1.ConditionStatus, string, string)) error {
	var patch runtimeclient.Patch

	runtimeObj := object.(runtimeclient.Object)
	switch obj := runtimeObj.(type) {
	case *kedav1alpha1.ScaledObject:
		patch = runtimeclient.MergeFrom(obj.DeepCopy())
		setCondition(obj.Status.Conditions, status, reason, message)
	case *kedav1alpha1.ScaledJob:
		patch = runtimeclient.MergeFrom(obj.DeepCopy())
		setCondition(obj.Status.Conditions, status, reason, message)
	default:
		err := fmt.Errorf("unknown scalable object type %v", obj)
		logger.Error(err, "Failed to patch Objects Status")
		return err
	}

	err := e.client.Status().Patch(ctx, runtimeObj, patch)
	if err != nil {
		logger.Error(err, "Failed to patch Objects Status")
	}
	return err
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
	fallback := func(conditions kedav1alpha1.Conditions, status metav1.ConditionStatus, reason string, message string) {
		conditions.SetFallbackCondition(status, reason, message)
	}
	return e.setCondition(ctx, logger, object, status, reason, message, fallback)
}
