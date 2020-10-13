package controllers

import (
	"context"

	"github.com/go-logr/logr"

	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
	"github.com/kedacore/keda/v2/controllers/util"
)

const (
	scaledJobFinalizer = "finalizer.keda.sh"
)

// finalizeScaledJob runs finalization logic on ScaledJob if there's finalizer
func (r *ScaledJobReconciler) finalizeScaledJob(logger logr.Logger, scaledJob *kedav1alpha1.ScaledJob) error {
	if util.Contains(scaledJob.GetFinalizers(), scaledJobFinalizer) {
		// Run finalization logic for scaledJobFinalizer. If the
		// finalization logic fails, don't remove the finalizer so
		// that we can retry during the next reconciliation.
		if err := r.stopScaleLoop(scaledJob); err != nil {
			return err
		}

		// Remove scaledJobFinalizer. Once all finalizers have been
		// removed, the object will be deleted.
		scaledJob.SetFinalizers(util.Remove(scaledJob.GetFinalizers(), scaledJobFinalizer))
		if err := r.Client.Update(context.TODO(), scaledJob); err != nil {
			logger.Error(err, "Failed to update ScaledJob after removing a finalizer", "finalizer", scaledJobFinalizer)
			return err
		}
	}

	logger.Info("Successfully finalized ScaledJob")
	return nil
}

// ensureFinalizer check there is finalizer present on the ScaledJob, if not it adds one
func (r *ScaledJobReconciler) ensureFinalizer(logger logr.Logger, scaledJob *kedav1alpha1.ScaledJob) error {
	if !util.Contains(scaledJob.GetFinalizers(), scaledJobFinalizer) {
		logger.Info("Adding Finalizer for the ScaledJob")
		scaledJob.SetFinalizers(append(scaledJob.GetFinalizers(), scaledJobFinalizer))

		// Update CR
		err := r.Client.Update(context.TODO(), scaledJob)
		if err != nil {
			logger.Error(err, "Failed to update ScaledJob with a finalizer", "finalizer", scaledJobFinalizer)
			return err
		}
	}
	return nil
}
