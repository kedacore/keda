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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	eventingv1alpha1 "github.com/kedacore/keda/v2/apis/eventing/v1alpha1"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/controllers/keda/util"
	"github.com/kedacore/keda/v2/pkg/common/message"
	"github.com/kedacore/keda/v2/pkg/eventreason"
)

const (
	scaledObjectFinalizer = "finalizer.keda.sh"
)

// finalizeScaledObject runs finalization logic on ScaledObject if there's finalizer
func (r *ScaledObjectReconciler) finalizeScaledObject(ctx context.Context, logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject,
	namespacedName string) error {
	if util.Contains(scaledObject.GetFinalizers(), scaledObjectFinalizer) {
		// Run finalization logic for scaledObjectFinalizer. If the
		// finalization logic fails, don't remove the finalizer so
		// that we can retry during the next reconciliation.
		if err := r.stopScaleLoop(ctx, logger, scaledObject); err != nil {
			return err
		}

		// if enabled, scale scaleTarget back to the original replica count (to the state it was before scaling with KEDA)
		if scaledObject.Spec.Advanced != nil && scaledObject.Spec.Advanced.RestoreToOriginalReplicaCount {
			// If the scaling hasn't been yet initialized (for example due to the missing scaleTarget), we don't have the GVKR information about the scaleTarget.
			// Thus we don't have enough information needed to properly set the number of replicas on the scaleTarget.
			// Let's skip in this case.
			if scaledObject.Status.ScaleTargetGVKR == nil {
				logger.V(1).Info("Failed to restore scaleTarget's replica count back to the original, the scaling haven't been probably initialized yet.")
			} else {
				// We have enough information about the scaleTarget, let's proceed.
				scale, err := r.ScaleClient.Scales(scaledObject.Namespace).Get(ctx, scaledObject.Status.ScaleTargetGVKR.GroupResource(), scaledObject.Spec.ScaleTargetRef.Name, metav1.GetOptions{})
				if err != nil {
					if errors.IsNotFound(err) {
						logger.V(1).Info("Failed to get scaleTarget's scale status, because it was probably deleted", "error", err)
					} else {
						logger.Error(err, "Failed to get scaleTarget's scale status from a finalizer", "finalizer", scaledObjectFinalizer)
					}
				} else {
					scale.Spec.Replicas = *scaledObject.Status.OriginalReplicaCount
					_, err = r.ScaleClient.Scales(scaledObject.Namespace).Update(ctx, scaledObject.Status.ScaleTargetGVKR.GroupResource(), scale, metav1.UpdateOptions{})
					if err != nil {
						logger.Error(err, "Failed to restore scaleTarget's replica count back to the original", "finalizer", scaledObjectFinalizer)
					}
					logger.Info("Successfully restored scaleTarget's replica count back to the original", "replicaCount", scale.Spec.Replicas)
				}
			}
		}

		// Remove scaledObjectFinalizer. Once all finalizers have been
		// removed, the object will be deleted.
		scaledObject.SetFinalizers(util.Remove(scaledObject.GetFinalizers(), scaledObjectFinalizer))
		if err := r.Client.Update(ctx, scaledObject); err != nil {
			logger.Error(err, "Failed to update ScaledObject after removing a finalizer", "finalizer", scaledObjectFinalizer)
			return err
		}

		if _, err := r.updateTriggerAuthenticationStatusOnDelete(ctx, logger, scaledObject); err != nil {
			logger.Error(err, "Failed to update TriggerAuthentication Status after removing a finalizer")
		}
		r.updatePromMetricsOnDelete(namespacedName)
	}

	logger.Info("Successfully finalized ScaledObject")
	r.EventEmitter.Emit(scaledObject, nil, scaledObject.Namespace, corev1.EventTypeWarning, eventingv1alpha1.ScaledObjectRemovedType, eventreason.ScaledObjectDeleted, action.Deleted, message.ScaledObjectRemoved)
	return nil
}

// ensureFinalizer check there is finalizer present on the ScaledObject, if not it adds one
func (r *ScaledObjectReconciler) ensureFinalizer(ctx context.Context, logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) error {
	if !util.Contains(scaledObject.GetFinalizers(), scaledObjectFinalizer) {
		logger.Info("Adding Finalizer for the ScaledObject")
		scaledObject.SetFinalizers(append(scaledObject.GetFinalizers(), scaledObjectFinalizer))

		// Update CR
		err := r.Client.Update(ctx, scaledObject)
		if err != nil {
			logger.Error(err, "Failed to update ScaledObject with a finalizer", "finalizer", scaledObjectFinalizer)
			return err
		}
	}
	return nil
}
