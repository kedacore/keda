package util

import (
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/equality"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

type PausedReplicasPredicate struct {
	predicate.Funcs
}

func (PausedReplicasPredicate) Update(e event.UpdateEvent) bool {
	return checkAnnotation(e, kedav1alpha1.PausedReplicasAnnotation)
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
	return checkAnnotation(e, kedav1alpha1.PausedAnnotation)
}

type PausedScaleInPredicate struct {
	predicate.Funcs
}

func (PausedScaleInPredicate) Update(e event.UpdateEvent) bool {
	return checkAnnotation(e, kedav1alpha1.PausedScaleInAnnotation)
}

type ForceActivationPredicate struct {
	predicate.Funcs
}

func (ForceActivationPredicate) Update(e event.UpdateEvent) bool {
	return checkAnnotation(e, kedav1alpha1.ForceActivationAnnotation)
}

func checkAnnotation(e event.UpdateEvent, annotation string) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}

	newAnnotations := e.ObjectNew.GetAnnotations()
	oldAnnotations := e.ObjectOld.GetAnnotations()

	newValue := ""
	oldValue := ""

	if newAnnotations != nil {
		newValue = newAnnotations[annotation]
	}

	if oldAnnotations != nil {
		oldValue = oldAnnotations[annotation]
	}

	return newValue != oldValue
}

type HPASpecChangedPredicate struct {
	predicate.Funcs
}

func (HPASpecChangedPredicate) Update(e event.UpdateEvent) bool {
	newObj := e.ObjectNew.(*autoscalingv2.HorizontalPodAutoscaler)
	oldObj := e.ObjectOld.(*autoscalingv2.HorizontalPodAutoscaler)

	return len(newObj.Spec.Metrics) != len(oldObj.Spec.Metrics) || !equality.Semantic.DeepDerivative(newObj.Spec, oldObj.Spec)
}
