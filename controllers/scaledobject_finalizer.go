package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
	"github.com/kedacore/keda/v2/controllers/util"
)

const (
	scaledObjectFinalizer = "finalizer.keda.sh"
)

// finalizeScaledObject runs finalization logic on ScaledObject if there's finalizer
func (r *ScaledObjectReconciler) finalizeScaledObject(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) error {
	if util.Contains(scaledObject.GetFinalizers(), scaledObjectFinalizer) {
		// Run finalization logic for scaledObjectFinalizer. If the
		// finalization logic fails, don't remove the finalizer so
		// that we can retry during the next reconciliation.
		if err := r.stopScaleLoop(logger, scaledObject); err != nil {
			return err
		}

		// if enabled, scale scaleTarget back to the original replica count (to the state it was before scaling with KEDA)
		if scaledObject.Spec.Advanced != nil && scaledObject.Spec.Advanced.RestoreToOriginalReplicaCount {
			scale, err := (*r.scaleClient).Scales(scaledObject.Namespace).Get(context.TODO(), scaledObject.Status.ScaleTargetGVKR.GroupResource(), scaledObject.Spec.ScaleTargetRef.Name, metav1.GetOptions{})
			if err != nil {
				if errors.IsNotFound(err) {
					logger.V(1).Info("Failed to get scaleTarget's scale status, because it was probably deleted", "error", err)
				} else {
					logger.Error(err, "Failed to get scaleTarget's scale status from a finalizer", "finalizer", scaledObjectFinalizer)
				}
			} else {
				scale.Spec.Replicas = *scaledObject.Status.OriginalReplicaCount
				_, err = (*r.scaleClient).Scales(scaledObject.Namespace).Update(context.TODO(), scaledObject.Status.ScaleTargetGVKR.GroupResource(), scale, metav1.UpdateOptions{})
				if err != nil {
					logger.Error(err, "Failed to restore scaleTarget's replica count back to the original", "finalizer", scaledObjectFinalizer)
				}
				logger.Info("Successfully restored scaleTarget's replica count back to the original", "replicaCount", scale.Spec.Replicas)
			}
		}

		// Remove scaledObjectFinalizer. Once all finalizers have been
		// removed, the object will be deleted.
		scaledObject.SetFinalizers(util.Remove(scaledObject.GetFinalizers(), scaledObjectFinalizer))
		if err := r.Client.Update(context.TODO(), scaledObject); err != nil {
			logger.Error(err, "Failed to update ScaledObject after removing a finalizer", "finalizer", scaledObjectFinalizer)
			return err
		}
	}

	logger.Info("Successfully finalized ScaledObject")
	return nil
}

// ensureFinalizer check there is finalizer present on the ScaledObject, if not it adds one
func (r *ScaledObjectReconciler) ensureFinalizer(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) error {
	if !util.Contains(scaledObject.GetFinalizers(), scaledObjectFinalizer) {
		logger.Info("Adding Finalizer for the ScaledObject")
		scaledObject.SetFinalizers(append(scaledObject.GetFinalizers(), scaledObjectFinalizer))

		// Update CR
		err := r.Client.Update(context.TODO(), scaledObject)
		if err != nil {
			logger.Error(err, "Failed to update ScaledObject with a finalizer", "finalizer", scaledObjectFinalizer)
			return err
		}
	}
	return nil
}
