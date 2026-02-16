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
	"sync"

	"github.com/kedacore/keda/v2/pkg/common/action"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	eventingv1alpha1 "github.com/kedacore/keda/v2/apis/eventing/v1alpha1"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/common/message"
	"github.com/kedacore/keda/v2/pkg/eventemitter"
	"github.com/kedacore/keda/v2/pkg/eventreason"
	"github.com/kedacore/keda/v2/pkg/metricscollector"
	"github.com/kedacore/keda/v2/pkg/util"
)

// TriggerAuthenticationReconciler reconciles a TriggerAuthentication object
type TriggerAuthenticationReconciler struct {
	client.Client
	eventemitter.EventHandler
}

type triggerAuthMetricsData struct {
	namespace string
}

var (
	triggerAuthPromMetricsMap  map[string]triggerAuthMetricsData
	triggerAuthPromMetricsLock *sync.Mutex
)

func init() {
	triggerAuthPromMetricsMap = make(map[string]triggerAuthMetricsData)
	triggerAuthPromMetricsLock = &sync.Mutex{}
}

// +kubebuilder:rbac:groups=keda.sh,resources=triggerauthentications;triggerauthentications/status,verbs=get;list;watch;update;patch

// Reconcile performs reconciliation on the identified TriggerAuthentication resource based on the request information passed, returns the result and an error (if any).
func (r *TriggerAuthenticationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)

	triggerAuthentication := &kedav1alpha1.TriggerAuthentication{}
	err := r.Get(ctx, req.NamespacedName, triggerAuthentication)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		reqLogger.Error(err, "Failed to get TriggerAuthentication")
		return ctrl.Result{}, err
	}

	if triggerAuthentication.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, r.finalizeTriggerAuthentication(ctx, reqLogger, triggerAuthentication, req.String())
	}

	if err := r.ensureFinalizer(ctx, reqLogger, triggerAuthentication); err != nil {
		return ctrl.Result{}, err
	}
	r.updatePromMetrics(triggerAuthentication, req.String())

	if triggerAuthentication.Generation == 1 {
		r.Emit(triggerAuthentication, nil, req.Namespace, corev1.EventTypeNormal, eventingv1alpha1.TriggerAuthenticationCreatedType, eventreason.TriggerAuthenticationAdded, action.Created, message.TriggerAuthenticationCreatedMsg)
	} else {
		msg := fmt.Sprintf(message.TriggerAuthenticationUpdatedMsg, triggerAuthentication.Name)
		r.Emit(triggerAuthentication, nil, req.Namespace, corev1.EventTypeNormal, eventingv1alpha1.TriggerAuthenticationUpdatedType, eventreason.TriggerAuthenticationUpdated, action.Updated, msg)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TriggerAuthenticationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kedav1alpha1.TriggerAuthentication{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		WithEventFilter(util.IgnoreOtherNamespaces()).
		Complete(r)
}

func (r *TriggerAuthenticationReconciler) updatePromMetrics(triggerAuth *kedav1alpha1.TriggerAuthentication, namespacedName string) {
	triggerAuthPromMetricsLock.Lock()
	defer triggerAuthPromMetricsLock.Unlock()

	if metricsData, ok := triggerAuthPromMetricsMap[namespacedName]; ok {
		metricscollector.DecrementCRDTotal(metricscollector.TriggerAuthenticationResource, metricsData.namespace)
	}

	metricscollector.IncrementCRDTotal(metricscollector.TriggerAuthenticationResource, triggerAuth.Namespace)
	triggerAuthPromMetricsMap[namespacedName] = triggerAuthMetricsData{namespace: triggerAuth.Namespace}
}

// UpdatePromMetricsOnDelete is idempotent, so it can be called multiple times without side-effects
func (r *TriggerAuthenticationReconciler) UpdatePromMetricsOnDelete(namespacedName string) {
	triggerAuthPromMetricsLock.Lock()
	defer triggerAuthPromMetricsLock.Unlock()

	if metricsData, ok := triggerAuthPromMetricsMap[namespacedName]; ok {
		metricscollector.DecrementCRDTotal(metricscollector.TriggerAuthenticationResource, metricsData.namespace)
	}

	delete(triggerAuthPromMetricsMap, namespacedName)
}
