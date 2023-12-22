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
	"strconv"
	"sync"
	"time"

	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	kedacontrollerutil "github.com/kedacore/keda/v2/controllers/keda/util"
	"github.com/kedacore/keda/v2/pkg/eventreason"
	"github.com/kedacore/keda/v2/pkg/metricscollector"
	"github.com/kedacore/keda/v2/pkg/scaling"
	kedastatus "github.com/kedacore/keda/v2/pkg/status"
)

// +kubebuilder:rbac:groups=keda.sh,resources=scaledjobs;scaledjobs/finalizers;scaledjobs/status,verbs="*"
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs="*"

// ScaledJobReconciler reconciles a ScaledJob object
type ScaledJobReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	GlobalHTTPTimeout time.Duration
	Recorder          record.EventRecorder

	scaledJobGenerations *sync.Map
	scaleHandler         scaling.ScaleHandler
	SecretsLister        corev1listers.SecretLister
	SecretsSynced        cache.InformerSynced
}

type scaledJobMetricsData struct {
	namespace    string
	triggerTypes []string
}

var (
	scaledJobPromMetricsMap  map[string]scaledJobMetricsData
	scaledJobPromMetricsLock *sync.Mutex
)

func init() {
	scaledJobPromMetricsMap = make(map[string]scaledJobMetricsData)
	scaledJobPromMetricsLock = &sync.Mutex{}
}

// SetupWithManager initializes the ScaledJobReconciler instance and starts a new controller managed by the passed Manager instance.
func (r *ScaledJobReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	r.scaleHandler = scaling.NewScaleHandler(mgr.GetClient(), nil, mgr.GetScheme(), r.GlobalHTTPTimeout, mgr.GetEventRecorderFor("scale-handler"), r.SecretsLister)
	r.scaledJobGenerations = &sync.Map{}
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		// Ignore updates to ScaledJob Status (in this case metadata.Generation does not change)
		// so reconcile loop is not started on Status updates
		For(&kedav1alpha1.ScaledJob{}, builder.WithPredicates(
			predicate.Or(
				kedacontrollerutil.PausedPredicate{},
				predicate.GenerationChangedPredicate{},
			))).
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
		return ctrl.Result{}, r.finalizeScaledJob(ctx, reqLogger, scaledJob, req.NamespacedName.String())
	}
	r.updatePromMetrics(scaledJob, req.NamespacedName.String())

	// ensure finalizer is set on this CR
	if err := r.ensureFinalizer(ctx, reqLogger, scaledJob); err != nil {
		return ctrl.Result{}, err
	}

	// ensure Status Conditions are initialized
	if !scaledJob.Status.Conditions.AreInitialized() {
		conditions := kedav1alpha1.GetInitializedConditions()
		if err := kedastatus.SetStatusConditions(ctx, r.Client, reqLogger, scaledJob, conditions); err != nil {
			r.Recorder.Event(scaledJob, corev1.EventTypeWarning, eventreason.ScaledJobUpdateFailed, err.Error())
			return ctrl.Result{}, err
		}
	}

	// Check jobTargetRef is specified
	if scaledJob.Spec.JobTargetRef == nil {
		errMsg := "ScaledJob.spec.jobTargetRef not found"
		err := fmt.Errorf(errMsg)
		reqLogger.Error(err, errMsg)
		r.Recorder.Event(scaledJob, corev1.EventTypeWarning, eventreason.ScaledJobCheckFailed, errMsg)
		return ctrl.Result{}, err
	}
	conditions := scaledJob.Status.Conditions.DeepCopy()
	msg, err := r.reconcileScaledJob(ctx, reqLogger, scaledJob, &conditions)
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

	if err := kedastatus.SetStatusConditions(ctx, r.Client, reqLogger, scaledJob, &conditions); err != nil {
		r.Recorder.Event(scaledJob, corev1.EventTypeWarning, eventreason.ScaledJobUpdateFailed, err.Error())
		return ctrl.Result{}, err
	}

	if _, err := r.updateTriggerAuthenticationStatus(ctx, reqLogger, scaledJob); err != nil {
		reqLogger.Error(err, "Error updating TriggerAuthentication Status")
	}

	return ctrl.Result{}, err
}

// reconcileScaledJob implements reconciler logic for K8s Jobs based ScaledJob
func (r *ScaledJobReconciler) reconcileScaledJob(ctx context.Context, logger logr.Logger, scaledJob *kedav1alpha1.ScaledJob, conditions *kedav1alpha1.Conditions) (string, error) {
	isPaused, err := r.checkIfPaused(ctx, logger, scaledJob, conditions)
	if err != nil {
		return "Failed to check if ScaledJob was paused", err
	}
	if isPaused {
		return "ScaledJob is paused, skipping reconcile loop", err
	}

	// nosemgrep: trailofbits.go.invalid-usage-of-modified-variable.invalid-usage-of-modified-variable
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

	for _, trigger := range scaledJob.Spec.Triggers {
		if trigger.UseCachedMetrics {
			logger.Info("Warning: property useCachedMetrics is not supported for ScaledJobs.")
		}
		if trigger.MetricType != "" {
			err := fmt.Errorf("metricType is set in one of the ScaledJob scaler")
			logger.Error(err, "metricType cannot be set in ScaledJob triggers")
			return "Cannot set metricType in ScaledJob triggers", err
		}
	}

	// scaledJob was created or modified - let's start a new ScaleLoop
	err = r.requestScaleLoop(ctx, logger, scaledJob)
	if err != nil {
		return "Failed to start a new scale loop with scaling logic", err
	}
	logger.Info("Initializing Scaling logic according to ScaledJob Specification")
	return "ScaledJob is defined correctly and is ready to scaling", nil
}

// checkIfPaused checks the presence of "autoscaling.keda.sh/paused" annotation on the scaledJob and stop the scale loop.
func (r *ScaledJobReconciler) checkIfPaused(ctx context.Context, logger logr.Logger, scaledJob *kedav1alpha1.ScaledJob, conditions *kedav1alpha1.Conditions) (bool, error) {
	pausedAnnotationValue, pausedAnnotation := scaledJob.GetAnnotations()[kedav1alpha1.PausedAnnotation]
	pausedStatus := conditions.GetPausedCondition().Status == metav1.ConditionTrue
	shouldPause := false
	if pausedAnnotation {
		var err error
		shouldPause, err = strconv.ParseBool(pausedAnnotationValue)
		if err != nil {
			shouldPause = true
		}
	}
	if shouldPause {
		if !pausedStatus {
			logger.Info("ScaledJob is paused, stopping scaling loop.")
			msg := kedav1alpha1.ScaledJobConditionPausedMessage
			if err := r.stopScaleLoop(ctx, logger, scaledJob); err != nil {
				msg = "failed to stop the scale loop for paused ScaledJob"
				conditions.SetPausedCondition(metav1.ConditionFalse, "ScaledJobStopScaleLoopFailed", msg)
				return false, err
			}
			conditions.SetPausedCondition(metav1.ConditionTrue, kedav1alpha1.ScaledJobConditionPausedReason, msg)
		}
		return true, nil
	}
	if pausedStatus {
		logger.Info("Unpausing ScaledJob.")
		msg := kedav1alpha1.ScaledJobConditionUnpausedMessage
		conditions.SetPausedCondition(metav1.ConditionFalse, kedav1alpha1.ScaledJobConditionUnpausedReason, msg)
	}
	return false, nil
}

// Delete Jobs owned by the previous version of the scaledJob based on the rolloutStrategy given for this scaledJob, if any
func (r *ScaledJobReconciler) deletePreviousVersionScaleJobs(ctx context.Context, logger logr.Logger, scaledJob *kedav1alpha1.ScaledJob) (string, error) {
	var rolloutStrategy string
	if len(scaledJob.Spec.RolloutStrategy) > 0 {
		logger.Info("RolloutStrategy is deprecated, please us Rollout.Strategy in order to define the desired strategy for job rollouts")
		rolloutStrategy = scaledJob.Spec.RolloutStrategy
	} else {
		rolloutStrategy = scaledJob.Spec.Rollout.Strategy
	}

	switch rolloutStrategy {
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

			propagationPolicy := metav1.DeletePropagationBackground
			if scaledJob.Spec.Rollout.PropagationPolicy == "foreground" {
				propagationPolicy = metav1.DeletePropagationForeground
			}
			err = r.Client.Delete(ctx, &job, client.PropagationPolicy(propagationPolicy))
			if err != nil {
				return "Not able to delete job: " + job.Name, err
			}
		}
		return fmt.Sprintf("RolloutStrategy: immediate, deleted jobs owned by the previous version of the scaleJob: %d jobs deleted", len(jobs.Items)), nil
	}
	return fmt.Sprintf("RolloutStrategy: %s", scaledJob.Spec.RolloutStrategy), nil
}

// requestScaleLoop request ScaleLoop handler for the respective ScaledJob
func (r *ScaledJobReconciler) requestScaleLoop(ctx context.Context, logger logr.Logger, scaledJob *kedav1alpha1.ScaledJob) error {
	logger.V(1).Info("Starting a new ScaleLoop")
	key, err := cache.MetaNamespaceKeyFunc(scaledJob)
	if err != nil {
		logger.Error(err, "Error getting key for scaledJob")
		return err
	}

	if err = r.scaleHandler.HandleScalableObject(ctx, scaledJob); err != nil {
		return err
	}

	r.scaledJobGenerations.Store(key, scaledJob.Generation)

	return nil
}

// stopScaleLoop stops ScaleLoop handler for the respective ScaledJob
func (r *ScaledJobReconciler) stopScaleLoop(ctx context.Context, logger logr.Logger, scaledJob *kedav1alpha1.ScaledJob) error {
	logger.V(1).Info("Stopping a ScaleLoop")

	key, err := cache.MetaNamespaceKeyFunc(scaledJob)
	if err != nil {
		logger.Error(err, "Error getting key for scaledJob")
		return err
	}

	if err = r.scaleHandler.DeleteScalableObject(ctx, scaledJob); err != nil {
		return err
	}

	r.scaledJobGenerations.Delete(key)
	return nil
}

func (r *ScaledJobReconciler) updatePromMetrics(scaledJob *kedav1alpha1.ScaledJob, namespacedName string) {
	scaledJobPromMetricsLock.Lock()
	defer scaledJobPromMetricsLock.Unlock()

	metricsData, ok := scaledJobPromMetricsMap[namespacedName]

	if ok {
		metricscollector.DecrementCRDTotal(metricscollector.ScaledJobResource, metricsData.namespace)
		for _, triggerType := range metricsData.triggerTypes {
			metricscollector.DecrementTriggerTotal(triggerType)
		}
	}

	metricscollector.IncrementCRDTotal(metricscollector.ScaledJobResource, scaledJob.Namespace)
	metricsData.namespace = scaledJob.Namespace

	triggerTypes := make([]string, len(scaledJob.Spec.Triggers))
	for _, trigger := range scaledJob.Spec.Triggers {
		metricscollector.IncrementTriggerTotal(trigger.Type)
		triggerTypes = append(triggerTypes, trigger.Type)
	}
	metricsData.triggerTypes = triggerTypes

	scaledJobPromMetricsMap[namespacedName] = metricsData
}

func (r *ScaledJobReconciler) updatePromMetricsOnDelete(namespacedName string) {
	scaledJobPromMetricsLock.Lock()
	defer scaledJobPromMetricsLock.Unlock()

	if metricsData, ok := scaledJobPromMetricsMap[namespacedName]; ok {
		metricscollector.DecrementCRDTotal(metricscollector.ScaledJobResource, metricsData.namespace)
		for _, triggerType := range metricsData.triggerTypes {
			metricscollector.DecrementTriggerTotal(triggerType)
		}
	}

	delete(scaledJobPromMetricsMap, namespacedName)
}

func (r *ScaledJobReconciler) updateTriggerAuthenticationStatus(ctx context.Context, logger logr.Logger, scaledJob *kedav1alpha1.ScaledJob) (string, error) {
	return kedastatus.UpdateTriggerAuthenticationStatusFromTriggers(ctx, logger, r.Client, scaledJob.GetNamespace(), scaledJob.Spec.Triggers, func(triggerAuthenticationStatus *kedav1alpha1.TriggerAuthenticationStatus) *kedav1alpha1.TriggerAuthenticationStatus {
		triggerAuthenticationStatus.ScaledJobNamesStr = kedacontrollerutil.AppendIntoString(triggerAuthenticationStatus.ScaledJobNamesStr, scaledJob.GetName(), ",")
		return triggerAuthenticationStatus
	})
}

func (r *ScaledJobReconciler) updateTriggerAuthenticationStatusOnDelete(ctx context.Context, logger logr.Logger, scaledJob *kedav1alpha1.ScaledJob) (string, error) {
	return kedastatus.UpdateTriggerAuthenticationStatusFromTriggers(ctx, logger, r.Client, scaledJob.GetNamespace(), scaledJob.Spec.Triggers, func(triggerAuthenticationStatus *kedav1alpha1.TriggerAuthenticationStatus) *kedav1alpha1.TriggerAuthenticationStatus {
		triggerAuthenticationStatus.ScaledJobNamesStr = kedacontrollerutil.RemoveFromString(triggerAuthenticationStatus.ScaledJobNamesStr, scaledJob.GetName(), ",")
		return triggerAuthenticationStatus
	})
}
