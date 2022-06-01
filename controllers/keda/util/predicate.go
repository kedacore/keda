package util

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

const PausedReplicasAnnotation = "autoscaling.keda.sh/paused-replicas"

type PausedReplicasPredicate struct {
	predicate.Funcs
}

func (PausedReplicasPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}

	newAnnotations := e.ObjectNew.GetAnnotations()
	oldAnnotations := e.ObjectOld.GetAnnotations()
	if newAnnotations != nil && oldAnnotations != nil {
		if newVal, ok1 := newAnnotations[PausedReplicasAnnotation]; ok1 {
			if oldVal, ok2 := oldAnnotations[PausedReplicasAnnotation]; ok2 {
				return newVal != oldVal
			}
			return true
		}
	}
	return false
}

type ScaleObjectReadyConditionPredicate struct {
	predicate.Funcs
}

func (ScaleObjectReadyConditionPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}

	var newReadyCondition, oldReadyCondition kedav1alpha1.Condition

	oldObj, ok := e.ObjectOld.(*kedav1alpha1.ScaledObject)
	if !ok {
		return false
	}
	oldReadyCondition = oldObj.Status.Conditions.GetReadyCondition()

	newObj, ok := e.ObjectNew.(*kedav1alpha1.ScaledObject)
	if !ok {
		return false
	}
	newReadyCondition = newObj.Status.Conditions.GetReadyCondition()

	// False/Unknown -> True
	if !oldReadyCondition.IsTrue() && newReadyCondition.IsTrue() {
		return true
	}

	return false
}
