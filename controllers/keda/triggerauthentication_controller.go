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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/eventreason"
)

// TriggerAuthenticationReconciler reconciles a TriggerAuthentication object
type TriggerAuthenticationReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=keda.sh,resources=triggerauthentications;triggerauthentications/status,verbs="*"

// Reconcile performs reconciliation on the identified TriggerAuthentication resource based on the request information passed, returns the result and an error (if any).
func (r *TriggerAuthenticationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)

	triggerAuthentication := &kedav1alpha1.TriggerAuthentication{}
	err := r.Client.Get(ctx, req.NamespacedName, triggerAuthentication)
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

// SetupWithManager sets up the controller with the Manager.
func (r *TriggerAuthenticationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kedav1alpha1.TriggerAuthentication{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}
