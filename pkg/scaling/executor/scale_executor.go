package executor

import (
	"context"
	"fmt"

	kedav1alpha1 "github.com/kedacore/keda/pkg/apis/keda/v1alpha1"
	"github.com/kedacore/keda/pkg/scalers"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/scale"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// Default cooldown period for a ScaleTarget if no cooldownPeriod is defined on the scaledObject
	defaultCooldownPeriod = 5 * 60 // 5 minutes
)

type ScaleExecutor interface {
	RequestJobScale(ctx context.Context, scalers []scalers.Scaler, scaledObject *kedav1alpha1.ScaledJob)
	RequestScale(ctx context.Context, scalers []scalers.Scaler, scaledObject *kedav1alpha1.ScaledObject)
}

type scaleExecutor struct {
	client           client.Client
	scaleClient      *scale.ScalesGetter
	reconcilerScheme *runtime.Scheme
	logger           logr.Logger
}

func NewScaleExecutor(client client.Client, scaleClient *scale.ScalesGetter, reconcilerScheme *runtime.Scheme) ScaleExecutor {
	return &scaleExecutor{
		client:           client,
		scaleClient:      scaleClient,
		reconcilerScheme: reconcilerScheme,
		logger:           logf.Log.WithName("scaleexecutor"),
	}
}

func (e *scaleExecutor) updateLastActiveTime(ctx context.Context, logger logr.Logger, object interface{}) error {
	var patch client.Patch

	now := metav1.Now()
	runtimeObj := object.(runtime.Object)
	switch obj := runtimeObj.(type) {
	case *kedav1alpha1.ScaledObject:
		patch = client.MergeFrom(obj.DeepCopy())
		obj.Status.LastActiveTime = &now
	case *kedav1alpha1.ScaledJob:
		patch = client.MergeFrom(obj.DeepCopy())
		obj.Status.LastActiveTime = &now
	default:
		err := fmt.Errorf("Unknown scalable object type %v", obj)
		logger.Error(err, "Failed to patch Objects Status")
		return err
	}

	err := e.client.Status().Patch(ctx, runtimeObj, patch)
	if err != nil {
		logger.Error(err, "Failed to patch Objects Status")
	}
	return err
}

func (e *scaleExecutor) setActiveCondition(ctx context.Context, logger logr.Logger, object interface{}, status metav1.ConditionStatus, reason string, mesage string) error {
	var patch client.Patch

	runtimeObj := object.(runtime.Object)
	switch obj := runtimeObj.(type) {
	case *kedav1alpha1.ScaledObject:
		patch = client.MergeFrom(obj.DeepCopy())
		obj.Status.Conditions.SetActiveCondition(status, reason, mesage)
	case *kedav1alpha1.ScaledJob:
		patch = client.MergeFrom(obj.DeepCopy())
		obj.Status.Conditions.SetActiveCondition(status, reason, mesage)
	default:
		err := fmt.Errorf("Unknown scalable object type %v", obj)
		logger.Error(err, "Failed to patch Objects Status")
		return err
	}

	err := e.client.Status().Patch(ctx, runtimeObj, patch)
	if err != nil {
		logger.Error(err, "Failed to patch Objects Status")
	}
	return err
}
