package podautoscaler

import (
	"context"

	korev1alpha1 "github.com/Azure/Kore/pkg/apis/kore/v1alpha1"
	autoscalingv1alpha1 "github.com/knative/serving/pkg/apis/autoscaling/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_podautoscaler")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new PodAutoscaler Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcilePodAutoscaler{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("podautoscaler-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource PodAutoscaler
	err = c.Watch(&source.Kind{Type: &autoscalingv1alpha1.PodAutoscaler{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner PodAutoscaler
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &autoscalingv1alpha1.PodAutoscaler{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcilePodAutoscaler{}

// ReconcilePodAutoscaler reconciles a PodAutoscaler object
type ReconcilePodAutoscaler struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a PodAutoscaler object and makes changes based on the state read
// and what is in the PodAutoscaler.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcilePodAutoscaler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling PodAutoscaler")

	// Fetch the PodAutoscaler instance
	instance := &autoscalingv1alpha1.PodAutoscaler{}
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

	// TODO: We should filter what we watch instead of filtering
	// during reconcile
	if instance.Annotations["autoscaling.knative.dev/class"] != "kore" {
		reqLogger.Info("Ignoring PodAutoscaler", "PodAutoscaler.Name", instance.Name)
		// Not our PodAutoscaler, ignore
		return reconcile.Result{}, nil
	}

	// Define a new ScaledObject
	scaledObject := newScaledObjectForCR(instance)

	// Set PodAutoscaler instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, scaledObject, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this ScaledObject already exists
	found := &korev1alpha1.ScaledObject{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: scaledObject.Name, Namespace: scaledObject.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new ScaledObject", "ScaledObject.Namespace", scaledObject.Namespace, "ScaledObject.Name", scaledObject.Name)
		err = r.client.Create(context.TODO(), scaledObject)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Pod created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Pod already exists - don't requeue
	reqLogger.Info("Skip reconcile: ScaledObject already exists", "ScaledObject.Namespace", found.Namespace, "ScaledObject.Name", found.Name)
	return reconcile.Result{}, nil
}

// newScaledObjectForCR returns a ScaledObject with the same name/namespace as the cr
func newScaledObjectForCR(cr *autoscalingv1alpha1.PodAutoscaler) *korev1alpha1.ScaledObject {
	labels := map[string]string{
		"app": cr.Name,
	}

	return &korev1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: korev1alpha1.ScaledObjectSpec{
			ScaleTargetRef: korev1alpha1.ObjectReference{
				DeploymentName: cr.Spec.ScaleTargetRef.Name,
			},
			Triggers: []korev1alpha1.ScaleTriggers{
				korev1alpha1.ScaleTriggers{
					Type: "azure-queue",
					Name: "azure-queue",
					Metadata: map[string]string{
						"connection": cr.Annotations["kore/connection"],
						"queueName":  cr.Annotations["kore/queueName"],
					},
				},
			},
		},
	}
}
