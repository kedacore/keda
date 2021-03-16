package controllers

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/eventreason"
)

// +kubebuilder:rbac:groups=keda.sh,resources=clustertriggerauthentications;clustertriggerauthentications/status,verbs="*"

// ClusterTriggerAuthenticationReconciler reconciles a ClusterTriggerAuthentication object
type ClusterTriggerAuthenticationReconciler struct {
	Client   client.Client
	Log      logr.Logger
	Recorder record.EventRecorder
}

// Reconcile performs reconciliation on the identified TriggerAuthentication resource based on the request information passed, returns the result and an error (if any).
func (r *ClusterTriggerAuthenticationReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("ClusterTriggerAuthentication.Namespace", req.Namespace, "ClusterTriggerAuthentication.Name", req.Name)

	clusterTriggerAuthentication := &kedav1alpha1.ClusterTriggerAuthentication{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, clusterTriggerAuthentication)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		reqLogger.Error(err, "Failed ot get ClusterTriggerAuthentication")
		return ctrl.Result{}, err
	}

	if clusterTriggerAuthentication.GetDeletionTimestamp() != nil {
		r.Recorder.Event(clusterTriggerAuthentication, corev1.EventTypeNormal, eventreason.ClusterTriggerAuthenticationDeleted, "ClusterTriggerAuthentication was deleted")
		return ctrl.Result{}, nil
	}

	if clusterTriggerAuthentication.ObjectMeta.Generation == 1 {
		r.Recorder.Event(clusterTriggerAuthentication, corev1.EventTypeNormal, eventreason.ClusterTriggerAuthenticationAdded, "New ClusterTriggerAuthentication configured")
	}

	return ctrl.Result{}, nil
}

// SetupWithManager initializes the ClusterTriggerAuthenticationReconciler instance and starts a new controller managed by the passed Manager instance.
func (r *ClusterTriggerAuthenticationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kedav1alpha1.ClusterTriggerAuthentication{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}
