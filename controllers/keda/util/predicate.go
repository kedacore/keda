package util

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

type PausedReplicasPredicate struct {
	predicate.Funcs
}

func (PausedReplicasPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}

	newAnnotations := e.ObjectNew.GetAnnotations()
	oldAnnotations := e.ObjectOld.GetAnnotations()

	newPausedValue := ""
	oldPausedValue := ""

	if newAnnotations != nil {
		newPausedValue = newAnnotations[kedav1alpha1.PausedReplicasAnnotation]
	}

	if oldAnnotations != nil {
		oldPausedValue = oldAnnotations[kedav1alpha1.PausedReplicasAnnotation]
	}

	return newPausedValue != oldPausedValue
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

type PausedPredicate struct {
	predicate.Funcs
}

func (PausedPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}

	newAnnotations := e.ObjectNew.GetAnnotations()
	oldAnnotations := e.ObjectOld.GetAnnotations()

	newPausedValue := ""
	oldPausedValue := ""

	if newAnnotations != nil {
		newPausedValue = newAnnotations[kedav1alpha1.PausedAnnotation]
	}

	if oldAnnotations != nil {
		oldPausedValue = oldAnnotations[kedav1alpha1.PausedAnnotation]
	}

	return newPausedValue != oldPausedValue
}
