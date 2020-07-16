package scaledjob

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	kedav1alpha1 "github.com/kedacore/keda/pkg/apis/keda/v1alpha1"
	"github.com/kedacore/keda/pkg/scaling"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	//"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	//"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_scaledjob")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new ScaledJob Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileScaledJob{
		client:       mgr.GetClient(),
		scheme:       mgr.GetScheme(),
		scaleHandler: scaling.NewScaleHandler(mgr.GetClient(), nil, mgr.GetScheme())}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("scaledjob-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ScaledJob
	err = c.Watch(&source.Kind{Type: &kedav1alpha1.ScaledJob{}},
		&handler.EnqueueRequestForObject{},
		predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				// Ignore updates to ScaledObject Status (in this case metadata.Generation does not change)
				// so reconcile loop is not started on Status updates
				return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
			},
		})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileScaledJob implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileScaledJob{}

// ReconcileScaledJob reconciles a ScaledJob object
type ReconcileScaledJob struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client       client.Client
	scheme       *runtime.Scheme
	scaleHandler scaling.ScaleHandler
}

// Reconcile reads that state of the cluster for a ScaledJob object and makes changes based on the state read
// and what is in the ScaledJob.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileScaledJob) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)

	// Fetch the ScaledJob instance
	scaledJob := &kedav1alpha1.ScaledJob{}
	err := r.client.Get(context.TODO(), request.NamespacedName, scaledJob)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Failed to get ScaleJob")
		return reconcile.Result{}, err
	}

	reqLogger.Info("Reconciling ScaledJob")

	isScaledJobMarkedToBeDeleted := scaledJob.GetDeletionTimestamp() != nil
	if isScaledJobMarkedToBeDeleted {
		if contains(scaledJob.GetFinalizers(), scaledJobFinalizer) {
			// Run finalization logic for scaledJobFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizeScaledJob(reqLogger, scaledJob); err != nil {
				return reconcile.Result{}, err
			}

			// Remove scaledJobFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			scaledJob.SetFinalizers(remove(scaledJob.GetFinalizers(), scaledJobFinalizer))
			err := r.client.Update(context.TODO(), scaledJob)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	if !contains(scaledJob.GetFinalizers(), scaledJobFinalizer) {
		if err := r.addFinalizer(reqLogger, scaledJob); err != nil {
			return reconcile.Result{}, err
		}
	}

	reqLogger.V(1).Info("Detecting ScaleType from scaledJob")

	var errMsg string
	if scaledJob.Spec.JobTargetRef != nil {
		reqLogger.Info("Detected ScaleType = Job")
		conditions := scaledJob.Status.Conditions.DeepCopy()
		msg, err := r.reconcileScaledJob(reqLogger, scaledJob)
		if err != nil {
			reqLogger.Error(err, msg)
			conditions.SetReadyCondition(metav1.ConditionFalse, "ScaledObjectCheckFailed", msg)
			conditions.SetActiveCondition(metav1.ConditionUnknown, "UnknownState", "ScaledJob check failed")
		} else {
			reqLogger.V(1).Info(msg)
			conditions.SetReadyCondition(metav1.ConditionTrue, "ScaledJobReady", msg)
		}

		return reconcile.Result{}, err
	} else {
		errMsg = "scaledJob.Spec.JobTargetRef is not set"
		err = fmt.Errorf(errMsg)
		reqLogger.Error(err, "Failed to detect ScaleType")
		return reconcile.Result{}, err
	}
}

// reconcileJobType implemets reconciler logic for K8s Jobs based ScaleObject
func (r *ReconcileScaledJob) reconcileScaledJob(logger logr.Logger, scaledJob *kedav1alpha1.ScaledJob) (string, error) {

	msg, err := r.deletePreviousVersionScaleJobs(logger, scaledJob)
	if err != nil {
		return msg, err
	}

	// scaledJob was created or modified - let's start a new ScaleLoop
	err = r.requestScaleLoop(logger, scaledJob)
	if err != nil {
		return "Failed to start a new scale loop with scaling logic", err
	} else {
		logger.Info("Initializing Scaling logic according to ScaledObject Specification")
	}

	return "ScaledJob is defined correctly and is ready to scaling", nil
}

// Delete Jobs owned by the previous version of the scaledJob
func (r *ReconcileScaledJob) deletePreviousVersionScaleJobs(logger logr.Logger, scaledJob *kedav1alpha1.ScaledJob) (string, error) {
	opts := []client.ListOption{
		client.InNamespace(scaledJob.GetNamespace()),
		client.MatchingLabels(map[string]string{"scaledJob": scaledJob.GetName()}),
	}
	jobs := &batchv1.JobList{}
	err := r.client.List(context.TODO(), jobs, opts...)
	if err != nil {
		return "Cannot get list of Jobs owned by this scaledJob", err
	}

	if jobs.Size() > 0 {
		logger.Info("Deleting jobs owned by the previous version of the scaledJob", "Number of jobs to delete", jobs.Size())
	}
	for _, job := range jobs.Items {
		err = r.client.Delete(context.TODO(), &job, client.PropagationPolicy(metav1.DeletePropagationBackground))
		if err != nil {
			return "Not able to delete job: " + job.Name, err
		}
	}

	return fmt.Sprintf("Deleted jobs owned by the previous version of the scaleJob: %d jobs deleted", jobs.Size()), nil
}

// requestScaleLoop request ScaleLoop handler for the respective ScaledJob
func (r *ReconcileScaledJob) requestScaleLoop(logger logr.Logger, scaledJob *kedav1alpha1.ScaledJob) error {

	logger.V(1).Info("Starting a new ScaleLoop")

	if err := r.scaleHandler.HandleScalableObject(scaledJob); err != nil {
		return err
	}

	return nil
}
