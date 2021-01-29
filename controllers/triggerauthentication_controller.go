package controllers

import (
	"context"
	"github.com/go-logr/logr"
	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/eventreason"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// +kubebuilder:rbac:groups=keda.sh,resources=triggerauthentications;triggerauthentications/status,verbs="*"

// TriggerAuthenticationReconciler reconciles a TriggerAuthentication object
type TriggerAuthenticationReconciler struct {
	Client   client.Client
	Log      logr.Logger
	Recorder record.EventRecorder
}

func (r *TriggerAuthenticationReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("TriggerAuthentication.Namespace", req.Namespace, "TriggerAuthentication.Name", req.Name)

	triggerAuthentication := &kedav1alpha1.TriggerAuthentication{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, triggerAuthentication)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		reqLogger.Error(err, "Failed ot get TriggerAuthentication")
		return ctrl.Result{}, err
	}

	if triggerAuthentication.GetDeletionTimestamp() != nil {
		r.Recorder.Event(triggerAuthentication, corev1.EventTypeNormal, eventreason.TriggerAuthenticationDeleted, "TriggerAuthentication was deleted")
		return ctrl.Result{}, nil
	}

	if triggerAuthentication.ObjectMeta.Generation == 1 {
		r.Recorder.Event(triggerAuthentication, corev1.EventTypeNormal, eventreason.TriggerAuthenticationAdded, "New TriggerAuthentication configured")
	}

	return ctrl.Result{}, nil
}

func (r *TriggerAuthenticationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kedav1alpha1.TriggerAuthentication{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}
