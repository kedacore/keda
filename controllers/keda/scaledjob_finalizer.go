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

package keda

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/kedacore/keda/v2/pkg/common/action"
	corev1 "k8s.io/api/core/v1"

	eventingv1alpha1 "github.com/kedacore/keda/v2/apis/eventing/v1alpha1"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/controllers/keda/util"
	"github.com/kedacore/keda/v2/pkg/common/message"
	"github.com/kedacore/keda/v2/pkg/eventreason"
)

const (
	scaledJobFinalizer = "finalizer.keda.sh"
)

// finalizeScaledJob runs finalization logic on ScaledJob if there's finalizer
func (r *ScaledJobReconciler) finalizeScaledJob(ctx context.Context, logger logr.Logger, scaledJob *kedav1alpha1.ScaledJob,
	namespacedName string) error {
	if util.Contains(scaledJob.GetFinalizers(), scaledJobFinalizer) {
		// Run finalization logic for scaledJobFinalizer. If the
		// finalization logic fails, don't remove the finalizer so
		// that we can retry during the next reconciliation.
		if err := r.stopScaleLoop(ctx, logger, scaledJob); err != nil {
			return err
		}

		// Remove scaledJobFinalizer. Once all finalizers have been
		// removed, the object will be deleted.
		scaledJob.SetFinalizers(util.Remove(scaledJob.GetFinalizers(), scaledJobFinalizer))
		if err := r.Update(ctx, scaledJob); err != nil {
			logger.Error(err, "Failed to update ScaledJob after removing a finalizer", "finalizer", scaledJobFinalizer)
			return err
		}

		if _, err := r.updateTriggerAuthenticationStatusOnDelete(ctx, logger, scaledJob); err != nil {
			logger.Error(err, "Failed to update TriggerAuthentication Status after removing a finalizer")
		}
		r.updatePromMetricsOnDelete(namespacedName)
	}

	logger.Info("Successfully finalized ScaledJob")
	r.EventEmitter.Emit(scaledJob, nil, namespacedName, corev1.EventTypeWarning, eventingv1alpha1.ScaledJobRemovedType, eventreason.ScaledJobDeleted, action.Deleted, message.ScaledJobRemoved)
	return nil
}

// ensureFinalizer check there is finalizer present on the ScaledJob, if not it adds one
func (r *ScaledJobReconciler) ensureFinalizer(ctx context.Context, logger logr.Logger, scaledJob *kedav1alpha1.ScaledJob) error {
	if !util.Contains(scaledJob.GetFinalizers(), scaledJobFinalizer) {
		logger.Info("Adding Finalizer for the ScaledJob")
		scaledJob.SetFinalizers(append(scaledJob.GetFinalizers(), scaledJobFinalizer))

		// Update CR
		err := r.Update(ctx, scaledJob)
		if err != nil {
			logger.Error(err, "Failed to update ScaledJob with a finalizer", "finalizer", scaledJobFinalizer)
			return err
		}
	}
	return nil
}
