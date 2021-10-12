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
	"fmt"
	"time"

	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	kedacontrollerutil "github.com/kedacore/keda/v2/controllers/keda/util"
	"github.com/kedacore/keda/v2/pkg/eventreason"
	"github.com/kedacore/keda/v2/pkg/scaling"
)

// +kubebuilder:rbac:groups=keda.sh,resources=scaledjobs;scaledjobs/finalizers;scaledjobs/status,verbs="*"
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs="*"

// ScaledJobReconciler reconciles a ScaledJob object
type ScaledJobReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	GlobalHTTPTimeout time.Duration
	Recorder          record.EventRecorder
	scaleHandler      scaling.ScaleHandler
}

// SetupWithManager initializes the ScaledJobReconciler instance and starts a new controller managed by the passed Manager instance.
func (r *ScaledJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.scaleHandler = scaling.NewScaleHandler(mgr.GetClient(), nil, mgr.GetScheme(), r.GlobalHTTPTimeout, mgr.GetEventRecorderFor("scale-handler"))

	return ctrl.NewControllerManagedBy(mgr).
		// Ignore updates to ScaledJob Status (in this case metadata.Generation does not change)
		// so reconcile loop is not started on Status updates
		For(&kedav1alpha1.ScaledJob{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}

// Reconcile performs reconciliation on the identified ScaledJob resource based on the request information passed, returns the result and an error (if any).
func (r *ScaledJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)

	// Fetch the ScaledJob instance
	scaledJob := &kedav1alpha1.ScaledJob{}
	err := r.Client.Get(ctx, req.NamespacedName, scaledJob)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Failed to get ScaleJob")
		return ctrl.Result{}, err
	}

	reqLogger.Info("Reconciling ScaledJob")

	// Check if the ScaledJob instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	if scaledJob.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, r.finalizeScaledJob(ctx, reqLogger, scaledJob)
	}

	// ensure finalizer is set on this CR
	if err := r.ensureFinalizer(ctx, reqLogger, scaledJob); err != nil {
		return ctrl.Result{}, err
	}

	// ensure Status Conditions are initialized
	if !scaledJob.Status.Conditions.AreInitialized() {
		conditions := kedav1alpha1.GetInitializedConditions()
		if err := kedacontrollerutil.SetStatusConditions(ctx, r.Client, reqLogger, scaledJob, conditions); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Check jobTargetRef is specified
	if scaledJob.Spec.JobTargetRef == nil {
		errMsg := "scaledJob.spec.jobTargetRef is not set"
		err := fmt.Errorf(errMsg)
		reqLogger.Error(err, "scaledJob.spec.jobTargetRef not found")
		return ctrl.Result{}, err
	}
	msg, err := r.reconcileScaledJob(ctx, reqLogger, scaledJob)
	conditions := scaledJob.Status.Conditions.DeepCopy()
	if err != nil {
		reqLogger.Error(err, msg)
		conditions.SetReadyCondition(metav1.ConditionFalse, "ScaledJobCheckFailed", msg)
		conditions.SetActiveCondition(metav1.ConditionUnknown, "UnknownState", "ScaledJob check failed")
		r.Recorder.Event(scaledJob, corev1.EventTypeWarning, eventreason.ScaledJobCheckFailed, msg)
	} else {
		wasReady := conditions.GetReadyCondition()
		if wasReady.IsFalse() || wasReady.IsUnknown() {
			r.Recorder.Event(scaledJob, corev1.EventTypeNormal, eventreason.ScaledJobReady, "ScaledJob is ready for scaling")
		}
		reqLogger.V(1).Info(msg)
		conditions.SetReadyCondition(metav1.ConditionTrue, "ScaledJobReady", msg)
	}

	if err := kedacontrollerutil.SetStatusConditions(ctx, r.Client, reqLogger, scaledJob, &conditions); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, err
}

// reconcileScaledJob implements reconciler logic for K8s Jobs based ScaledJob
func (r *ScaledJobReconciler) reconcileScaledJob(ctx context.Context, logger logr.Logger, scaledJob *kedav1alpha1.ScaledJob) (string, error) {
	msg, err := r.deletePreviousVersionScaleJobs(ctx, logger, scaledJob)
	if err != nil {
		return msg, err
	}

	// Check ScaledJob is Ready or not
	_, err = r.scaleHandler.GetScalersCache(ctx, scaledJob)
	if err != nil {
		logger.Error(err, "Error getting scalers")
		return "Failed to ensure ScaledJob is correctly created", err
	}

	// scaledJob was created or modified - let's start a new ScaleLoop
	err = r.requestScaleLoop(logger, scaledJob)
	if err != nil {
		return "Failed to start a new scale loop with scaling logic", err
	}
	logger.Info("Initializing Scaling logic according to ScaledJob Specification")
	return "ScaledJob is defined correctly and is ready to scaling", nil
}

// Delete Jobs owned by the previous version of the scaledJob based on the rolloutStrategy given for this scaledJob, if any
func (r *ScaledJobReconciler) deletePreviousVersionScaleJobs(ctx context.Context, logger logr.Logger, scaledJob *kedav1alpha1.ScaledJob) (string, error) {
	switch scaledJob.Spec.RolloutStrategy {
	case "gradual":
		logger.Info("RolloutStrategy: gradual, Not deleting jobs owned by the previous version of the scaleJob")
	default:
		opts := []client.ListOption{
			client.InNamespace(scaledJob.GetNamespace()),
			client.MatchingLabels(map[string]string{"scaledjob.keda.sh/name": scaledJob.GetName()}),
		}
		jobs := &batchv1.JobList{}
		err := r.Client.List(ctx, jobs, opts...)
		if err != nil {
			return "Cannot get list of Jobs owned by this scaledJob", err
		}

		if len(jobs.Items) > 0 {
			logger.Info("RolloutStrategy: immediate, Deleting jobs owned by the previous version of the scaledJob", "numJobsToDelete", len(jobs.Items))
		}
		for _, job := range jobs.Items {
			job := job
			err = r.Client.Delete(ctx, &job, client.PropagationPolicy(metav1.DeletePropagationBackground))
			if err != nil {
				return "Not able to delete job: " + job.Name, err
			}
		}
		return fmt.Sprintf("RolloutStrategy: immediate, deleted jobs owned by the previous version of the scaleJob: %d jobs deleted", len(jobs.Items)), nil
	}
	return fmt.Sprintf("RolloutStrategy: %s", scaledJob.Spec.RolloutStrategy), nil
}

// requestScaleLoop request ScaleLoop handler for the respective ScaledJob
func (r *ScaledJobReconciler) requestScaleLoop(logger logr.Logger, scaledJob *kedav1alpha1.ScaledJob) error {
	logger.V(1).Info("Starting a new ScaleLoop")
	return r.scaleHandler.HandleScalableObject(scaledJob)
}

// stopScaleLoop stops ScaleLoop handler for the respective ScaledJob
func (r *ScaledJobReconciler) stopScaleLoop(logger logr.Logger, scaledJob *kedav1alpha1.ScaledJob) error {
	logger.V(1).Info("Stopping a ScaleLoop")
	return r.scaleHandler.DeleteScalableObject(scaledJob)
}
