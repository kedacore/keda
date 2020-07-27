package scaledjob

import (
	"context"

	"github.com/go-logr/logr"
	kedav1alpha1 "github.com/kedacore/keda/pkg/apis/keda/v1alpha1"
)

const (
	scaledJobFinalizer = "finalizer.keda.sh"
)

// finalizescaledJob is stopping ScaleLoop for the respective ScaleJob
func (r *ReconcileScaledJob) finalizeScaledJob(logger logr.Logger, scaledJob *kedav1alpha1.ScaledJob) error {
	// TODO implement finalize logic for ScaledJob
	logger.Info("Successfully finalized ScaledJob")
	return nil
}

// addFinalizer adds finalizer to the scaledJob
func (r *ReconcileScaledJob) addFinalizer(logger logr.Logger, scaledJob *kedav1alpha1.ScaledJob) error {
	logger.Info("Adding Finalizer for the ScaledJob")
	scaledJob.SetFinalizers(append(scaledJob.GetFinalizers(), scaledJobFinalizer))

	// Update CR
	err := r.client.Update(context.TODO(), scaledJob)
	if err != nil {
		logger.Error(err, "Failed to update ScaledJob with finalizer")
		return err
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
