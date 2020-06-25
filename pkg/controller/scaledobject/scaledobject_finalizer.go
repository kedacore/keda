package scaledobject

import (
	"context"

	kedav1alpha1 "github.com/kedacore/keda/pkg/apis/keda/v1alpha1"

	"github.com/go-logr/logr"
)

const (
	scaledObjectFinalizer = "finalizer.keda.sh"
)

// finalizeScaledObject runs finalization logic on ScaledObject if there's finalizer
func (r *ReconcileScaledObject) finalizeScaledObject(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) error {

	if contains(scaledObject.GetFinalizers(), scaledObjectFinalizer) {
		// Run finalization logic for scaledObjectFinalizer. If the
		// finalization logic fails, don't remove the finalizer so
		// that we can retry during the next reconciliation.
		if err := r.stopScaleLoop(logger, scaledObject); err != nil {
			return err
		}

		// Remove scaledObjectFinalizer. Once all finalizers have been
		// removed, the object will be deleted.
		scaledObject.SetFinalizers(remove(scaledObject.GetFinalizers(), scaledObjectFinalizer))
		if err := r.client.Update(context.TODO(), scaledObject); err != nil {
			logger.Error(err, "Failed to update ScaledObject after removing a finalizer", "finalizer", scaledObjectFinalizer)
			return err
		}
	}

	logger.Info("Successfully finalized ScaledObject")
	return nil
}

// ensureFinalizer check there is finalizer present on the ScaledObject, if not it adds one
func (r *ReconcileScaledObject) ensureFinalizer(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) error {

	if !contains(scaledObject.GetFinalizers(), scaledObjectFinalizer) {
		logger.Info("Adding Finalizer for the ScaledObject")
		scaledObject.SetFinalizers(append(scaledObject.GetFinalizers(), scaledObjectFinalizer))

		// Update CR
		err := r.client.Update(context.TODO(), scaledObject)
		if err != nil {
			logger.Error(err, "Failed to update ScaledObject with a finalizer", "finalizer", scaledObjectFinalizer)
			return err
		}
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func remove(list []string, s string) []string {
	for i, v := range list {
		if v == s {
			list = append(list[:i], list[i+1:]...)
		}
	}
	return list
}
