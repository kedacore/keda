package scaledjob

import (
	"context"

	kedav1alpha1 "github.com/kedacore/keda/pkg/apis/keda/v1alpha1"

	"k8s.io/apimachinery/pkg/api/errors"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	//"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	//"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
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
	return &ReconcileScaledJob{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("scaledjob-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ScaledJob
	err = c.Watch(&source.Kind{Type: &kedav1alpha1.ScaledJob{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner ScaledJob
	// err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
	// 	IsController: true,
	// 	OwnerType:    &kedav1alpha1.ScaledJob{},
	// })
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
	client client.Client
	scheme *runtime.Scheme
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
	reqLogger.Info("Reconciling ScaledJob")

	// Fetch the ScaledJob instance
	instance := &kedav1alpha1.ScaledJob{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	reqLogger.Info("Reconciling ScaledJob is NOT IMPLEMENTED yet")

	return reconcile.Result{}, nil
}

// FIXME use ScaledJob
// reconcileJobType implemets reconciler logic for K8s Jobs based ScaleObject
// func (r *ReconcileScaledObject) reconcileJobType(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) ( error) {
// 	//	scaledObject.Spec.ScaleType = kedav1alpha1.ScaleTypeJob

// 	// Delete Jobs owned by the previous version of the ScaledObject
// 	opts := []client.ListOption{
// 		client.InNamespace(scaledObject.GetNamespace()),
// 		client.MatchingLabels(map[string]string{"scaledobject": scaledObject.GetName()}),
// 	}
// 	jobs := &batchv1.JobList{}
// 	err := r.client.List(context.TODO(), jobs, opts...)
// 	if err != nil {
// 		logger.Error(err, "Cannot get list of Jobs owned by this ScaledObject")
// 		return err
// 	}

// 	if jobs.Size() > 0 {
// 		logger.Info("Deleting jobs owned by the previous version of the ScaledObject", "Number of jobs to delete", jobs.Size())
// 	}
// 	for _, job := range jobs.Items {
// 		err = r.client.Delete(context.TODO(), &job, client.PropagationPolicy(metav1.DeletePropagationBackground))
// 		if err != nil {
// 			logger.Error(err, "Not able to delete job", "Job", job.Name)
// 			return err
// 		}
// 	}

// 	// ScaledObject was created or modified - let's start a new ScaleLoop
// 	err = r.startScaleLoop(logger, scaledObject)
// 	if err != nil {
// 		logger.Error(err, "Failed to start a new ScaleLoop")
// 		return err
// 	}

// 	return nil
// }
