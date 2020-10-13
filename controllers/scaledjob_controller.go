package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scaling"
)

// +kubebuilder:rbac:groups=keda.sh,resources=scaledjobs;scaledjobs/finalizers;scaledjobs/status,verbs="*"
// +kubebuilder:rbac:groups=keda.sh,resources=triggerauthentications;triggerauthentications/status,verbs="*"
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs="*"

// ScaledJobReconciler reconciles a ScaledJob object
type ScaledJobReconciler struct {
	client.Client
	Log               logr.Logger
	Scheme            *runtime.Scheme
	scaleHandler      scaling.ScaleHandler
	globalHTTPTimeout time.Duration
}

// SetupWithManager initializes the ScaledJobReconciler instance and starts a new controller managed by the passed Manager instance.
func (r *ScaledJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.scaleHandler = scaling.NewScaleHandler(mgr.GetClient(), nil, mgr.GetScheme(), r.globalHTTPTimeout)

	return ctrl.NewControllerManagedBy(mgr).
		// Ignore updates to ScaledJob Status (in this case metadata.Generation does not change)
		// so reconcile loop is not started on Status updates
		For(&kedav1alpha1.ScaledJob{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}

// Reconcile performs reconciliation on the identified ScaledJob resource based on the request information passed, returns the result and an error (if any).
func (r *ScaledJobReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("ScaledJob.Namespace", req.Namespace, "ScaledJob.Name", req.Name)

	// Fetch the ScaledJob instance
	scaledJob := &kedav1alpha1.ScaledJob{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, scaledJob)
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
		return ctrl.Result{}, r.finalizeScaledJob(reqLogger, scaledJob)
	}

	// ensure finalizer is set on this CR
	if err := r.ensureFinalizer(reqLogger, scaledJob); err != nil {
		return ctrl.Result{}, err
	}

	var errMsg string
	if scaledJob.Spec.JobTargetRef != nil {
		reqLogger.Info("Detected ScaleType = Job")
		conditions := scaledJob.Status.Conditions.DeepCopy()
		msg, err := r.reconcileScaledJob(reqLogger, scaledJob)
		if err != nil {
			reqLogger.Error(err, msg)
			conditions.SetReadyCondition(metav1.ConditionFalse, "ScaledJobCheckFailed", msg)
			conditions.SetActiveCondition(metav1.ConditionUnknown, "UnknownState", "ScaledJob check failed")
		} else {
			reqLogger.V(1).Info(msg)
			conditions.SetReadyCondition(metav1.ConditionTrue, "ScaledJobReady", msg)
		}

		return ctrl.Result{}, err
	}

	errMsg = "scaledJob.Spec.JobTargetRef is not set"
	err = fmt.Errorf(errMsg)
	reqLogger.Error(err, "scaledJob.Spec.JobTargetRef not found")
	return ctrl.Result{}, err
}

// reconcileJobType implemets reconciler logic for K8s Jobs based ScaleObject
func (r *ScaledJobReconciler) reconcileScaledJob(logger logr.Logger, scaledJob *kedav1alpha1.ScaledJob) (string, error) {
	msg, err := r.deletePreviousVersionScaleJobs(logger, scaledJob)
	if err != nil {
		return msg, err
	}

	// scaledJob was created or modified - let's start a new ScaleLoop
	err = r.requestScaleLoop(logger, scaledJob)
	if err != nil {
		return "Failed to start a new scale loop with scaling logic", err
	}
	logger.Info("Initializing Scaling logic according to ScaledObject Specification")
	return "ScaledJob is defined correctly and is ready to scaling", nil
}

// Delete Jobs owned by the previous version of the scaledJob
func (r *ScaledJobReconciler) deletePreviousVersionScaleJobs(logger logr.Logger, scaledJob *kedav1alpha1.ScaledJob) (string, error) {
	opts := []client.ListOption{
		client.InNamespace(scaledJob.GetNamespace()),
		client.MatchingLabels(map[string]string{"scaledJob": scaledJob.GetName()}),
	}
	jobs := &batchv1.JobList{}
	err := r.Client.List(context.TODO(), jobs, opts...)
	if err != nil {
		return "Cannot get list of Jobs owned by this scaledJob", err
	}

	if jobs.Size() > 0 {
		logger.Info("Deleting jobs owned by the previous version of the scaledJob", "Number of jobs to delete", jobs.Size())
	}
	for _, job := range jobs.Items {
		job := job
		err = r.Client.Delete(context.TODO(), &job, client.PropagationPolicy(metav1.DeletePropagationBackground))
		if err != nil {
			return "Not able to delete job: " + job.Name, err
		}
	}

	return fmt.Sprintf("Deleted jobs owned by the previous version of the scaleJob: %d jobs deleted", jobs.Size()), nil
}

// requestScaleLoop request ScaleLoop handler for the respective ScaledJob
func (r *ScaledJobReconciler) requestScaleLoop(logger logr.Logger, scaledJob *kedav1alpha1.ScaledJob) error {
	logger.V(1).Info("Starting a new ScaleLoop")

	return r.scaleHandler.HandleScalableObject(scaledJob)
}

// stopScaleLoop stops ScaleLoop handler for the respective ScaledJob
func (r *ScaledJobReconciler) stopScaleLoop(scaledJob *kedav1alpha1.ScaledJob) error {
	if err := r.scaleHandler.DeleteScalableObject(scaledJob); err != nil {
		return err
	}

	return nil
}
