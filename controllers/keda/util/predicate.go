package util

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
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
