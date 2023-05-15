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

	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scaling"
)

type MetricsScaledObjectReconciler struct {
	Client                  client.Client
	ScaleHandler            scaling.ScaleHandler
	MaxConcurrentReconciles int
}

var (
	scaledObjectsMetrics     = map[string][]string{}
	scaledObjectsMetricsLock = &sync.Mutex{}
)

func (r *MetricsScaledObjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)

	// Fetch the ScaledObject instance
	scaledObject := &kedav1alpha1.ScaledObject{}
	err := r.Client.Get(ctx, req.NamespacedName, scaledObject)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			err := r.ScaleHandler.ClearScalersCache(ctx, scaledObject)
			if err != nil {
				reqLogger.Error(err, "error clearing scalers cache")
			}
			r.removeFromMetricsCache(req.NamespacedName.String())
			return ctrl.Result{}, err
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Failed to get ScaledObject")
		return ctrl.Result{}, err
	}

	// Check if the ScaledObject instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	// This depends on the preexisting finalizer setup in ScaledObjectController.
	if scaledObject.GetDeletionTimestamp() != nil {
		err := r.ScaleHandler.ClearScalersCache(ctx, scaledObject)
		if err != nil {
			reqLogger.Error(err, "error clearing scalers cache")
		}
		r.removeFromMetricsCache(req.NamespacedName.String())
		return ctrl.Result{}, err
	}

	reqLogger.V(1).Info("Reconciling ScaledObject", "externalMetricNames", scaledObject.Status.ExternalMetricNames)

	// The ScaledObject hasn't yet been properly initialized and ExternalMetricsNames list popoluted => requeue
	if scaledObject.Status.ExternalMetricNames == nil || len(scaledObject.Status.ExternalMetricNames) < 1 {
		return ctrl.Result{Requeue: true}, nil
	}

	r.addToMetricsCache(req.NamespacedName.String(), scaledObject.Status.ExternalMetricNames)
	err = r.ScaleHandler.ClearScalersCache(ctx, scaledObject)
	if err != nil {
		reqLogger.Error(err, "error clearing scalers cache")
	}
	return ctrl.Result{}, err
}

func (r *MetricsScaledObjectReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kedav1alpha1.ScaledObject{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&kedav1alpha1.ScaledObject{}).
		WithOptions(options).
		Complete(r)
}

func (r *MetricsScaledObjectReconciler) addToMetricsCache(namespacedName string, metrics []string) {
	scaledObjectsMetricsLock.Lock()
	defer scaledObjectsMetricsLock.Unlock()
	scaledObjectsMetrics[namespacedName] = metrics
}

func (r *MetricsScaledObjectReconciler) removeFromMetricsCache(namespacedName string) {
	scaledObjectsMetricsLock.Lock()
	defer scaledObjectsMetricsLock.Unlock()
	delete(scaledObjectsMetrics, namespacedName)
}
