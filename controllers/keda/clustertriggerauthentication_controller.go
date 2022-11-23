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
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/eventreason"
	"github.com/kedacore/keda/v2/pkg/prommetrics"
)

// ClusterTriggerAuthenticationReconciler reconciles a ClusterTriggerAuthentication object
type ClusterTriggerAuthenticationReconciler struct {
	client.Client
	record.EventRecorder
}

type clusterTriggerAuthMetricsData struct {
	namespace string
}

var (
	clusterTriggerAuthPromMetricsMap  map[string]clusterTriggerAuthMetricsData
	clusterTriggerAuthPromMetricsLock *sync.Mutex
)

func init() {
	clusterTriggerAuthPromMetricsMap = make(map[string]clusterTriggerAuthMetricsData)
	clusterTriggerAuthPromMetricsLock = &sync.Mutex{}
}

// +kubebuilder:rbac:groups=keda.sh,resources=clustertriggerauthentications;clustertriggerauthentications/status,verbs="*"

// Reconcile performs reconciliation on the identified TriggerAuthentication resource based on the request information passed, returns the result and an error (if any).
func (r *ClusterTriggerAuthenticationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)

	clusterTriggerAuthentication := &kedav1alpha1.ClusterTriggerAuthentication{}
	err := r.Client.Get(ctx, req.NamespacedName, clusterTriggerAuthentication)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		reqLogger.Error(err, "Failed to get ClusterTriggerAuthentication")
		return ctrl.Result{}, err
	}

	if clusterTriggerAuthentication.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, r.finalizeClusterTriggerAuthentication(ctx, reqLogger, clusterTriggerAuthentication, req.NamespacedName.String())
	}

	if err := r.ensureFinalizer(ctx, reqLogger, clusterTriggerAuthentication); err != nil {
		return ctrl.Result{}, err
	}
	r.updatePromMetrics(clusterTriggerAuthentication, req.NamespacedName.String())

	if clusterTriggerAuthentication.ObjectMeta.Generation == 1 {
		r.EventRecorder.Event(clusterTriggerAuthentication, corev1.EventTypeNormal, eventreason.ClusterTriggerAuthenticationAdded, "New ClusterTriggerAuthentication configured")
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterTriggerAuthenticationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kedav1alpha1.ClusterTriggerAuthentication{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}

func (r *ClusterTriggerAuthenticationReconciler) updatePromMetrics(clusterTriggerAuth *kedav1alpha1.ClusterTriggerAuthentication, namespacedName string) {
	clusterTriggerAuthPromMetricsLock.Lock()
	defer clusterTriggerAuthPromMetricsLock.Unlock()

	if metricsData, ok := clusterTriggerAuthPromMetricsMap[namespacedName]; ok {
		prommetrics.DecrementCRDTotal(prommetrics.ClusterTriggerAuthenticationResource, metricsData.namespace)
	}

	prommetrics.IncrementCRDTotal(prommetrics.ClusterTriggerAuthenticationResource, clusterTriggerAuth.Namespace)
	clusterTriggerAuthPromMetricsMap[namespacedName] = clusterTriggerAuthMetricsData{namespace: clusterTriggerAuth.Namespace}
}

// UpdatePromMetricsOnDelete is idempotent, so it can be called multiple times without side-effects
func (r *ClusterTriggerAuthenticationReconciler) UpdatePromMetricsOnDelete(namespacedName string) {
	clusterTriggerAuthPromMetricsLock.Lock()
	defer clusterTriggerAuthPromMetricsLock.Unlock()

	if metricsData, ok := clusterTriggerAuthPromMetricsMap[namespacedName]; ok {
		prommetrics.DecrementCRDTotal(prommetrics.ClusterTriggerAuthenticationResource, metricsData.namespace)
	}

	delete(clusterTriggerAuthPromMetricsMap, namespacedName)
}
