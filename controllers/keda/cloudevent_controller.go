/*
Copyright 2023 The KEDA Authors

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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/eventemitter"
	"github.com/kedacore/keda/v2/pkg/metricscollector"
)

// CloudEventsReconciler reconciles a CloudEvents object
type CloudEventsReconciler struct {
	client.Client
	EventEmitter eventemitter.EventEmitter

	cloudEventsGenerations *sync.Map
}

type cloudEventsMetricsData struct {
	namespace string
}

var (
	cloudEventPromMetricsMap  map[string]cloudEventsMetricsData
	cloudEventPromMetricsLock *sync.Mutex
)

func init() {
	cloudEventPromMetricsMap = make(map[string]cloudEventsMetricsData)
	cloudEventPromMetricsLock = &sync.Mutex{}
}

// +kubebuilder:rbac:groups=eventing.keda.sh,resources=cloudevents;cloudevents/status,verbs="*"

// Reconcile performs reconciliation on the identified CloudEvents resource based on the request information passed, returns the result and an error (if any).
func (r *CloudEventsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)

	// Fetch the CloudEvents instance
	cloudEvent := &kedav1alpha1.CloudEvent{}
	err := r.Client.Get(ctx, req.NamespacedName, cloudEvent)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request cloudevent not found, could have been deleted after reconcile request.
			// Owned cloudevent are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "failed to get CloudEvents")
		return ctrl.Result{}, err
	}

	reqLogger.Info("Reconciling CloudEvents")

	if cloudEvent.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, r.FinalizeCloudEventsResource(ctx, reqLogger, cloudEvent, req.NamespacedName.String())
	}
	r.updatePromMetrics(cloudEvent, req.NamespacedName.String())

	// ensure finalizer is set on this CR
	if err := r.EnsureCloudEventsResourceFinalizer(ctx, reqLogger, cloudEvent); err != nil {
		return ctrl.Result{}, err
	}

	cloudEventsChanged, err := r.cloudEventsGenerationChanged(reqLogger, cloudEvent)
	if err != nil {
		return ctrl.Result{}, err
	}

	if cloudEventsChanged {
		if r.requestEventLoop(ctx, reqLogger, cloudEvent) != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CloudEventsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.cloudEventsGenerations = &sync.Map{}
	return ctrl.NewControllerManagedBy(mgr).
		For(&kedav1alpha1.CloudEvent{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}

// requestEventLoop tries to start EventLoop handler for the respective CloudEvent
func (r *CloudEventsReconciler) requestEventLoop(ctx context.Context, logger logr.Logger, cloudEvent *kedav1alpha1.CloudEvent) error {
	logger.V(1).Info("Notify eventHandler of an update in cloudEvent")

	key, err := cache.MetaNamespaceKeyFunc(cloudEvent)
	if err != nil {
		logger.Error(err, "error getting key for cloudEvent")
		return err
	}

	if err = r.EventEmitter.HandleCloudEvents(ctx, cloudEvent); err != nil {
		return err
	}

	// store CloudEvents's current Generation
	r.cloudEventsGenerations.Store(key, cloudEvent.Generation)

	return nil
}

// stopEventLoop stops EventLoop handler for the respective CloudEvent
func (r *CloudEventsReconciler) stopEventLoop(logger logr.Logger, cloudEvent *kedav1alpha1.CloudEvent) error {
	key, err := cache.MetaNamespaceKeyFunc(cloudEvent)
	if err != nil {
		logger.Error(err, "error getting key for cloudEvent")
		return err
	}

	if err := r.EventEmitter.DeleteCloudEvents(cloudEvent); err != nil {
		return err
	}
	// delete CloudEvent's current Generation
	r.cloudEventsGenerations.Delete(key)
	return nil
}

// cloudEventsGenerationChanged returns true if CloudEvent's Generation was changed, ie. CloudEvent.Spec was changed
func (r *CloudEventsReconciler) cloudEventsGenerationChanged(logger logr.Logger, cloudEvent *kedav1alpha1.CloudEvent) (bool, error) {
	key, err := cache.MetaNamespaceKeyFunc(cloudEvent)
	if err != nil {
		logger.Error(err, "error getting key for cloudEvent")
		return true, err
	}

	value, loaded := r.cloudEventsGenerations.Load(key)
	if loaded {
		generation := value.(int64)
		if generation == cloudEvent.Generation {
			return false, nil
		}
	}
	return true, nil
}

func (r *CloudEventsReconciler) updatePromMetrics(cloudEvent *kedav1alpha1.CloudEvent, namespacedName string) {
	cloudEventPromMetricsLock.Lock()
	defer cloudEventPromMetricsLock.Unlock()

	if metricsData, ok := cloudEventPromMetricsMap[namespacedName]; ok {
		metricscollector.DecrementCRDTotal(metricscollector.CloudEventsResource, metricsData.namespace)
	}

	metricscollector.IncrementCRDTotal(metricscollector.CloudEventsResource, cloudEvent.Namespace)
	cloudEventPromMetricsMap[namespacedName] = cloudEventsMetricsData{namespace: cloudEvent.Namespace}
}

// UpdatePromMetricsOnDelete is idempotent, so it can be called multiple times without side-effects
func (r *CloudEventsReconciler) UpdatePromMetricsOnDelete(namespacedName string) {
	cloudEventPromMetricsLock.Lock()
	defer cloudEventPromMetricsLock.Unlock()

	if metricsData, ok := cloudEventPromMetricsMap[namespacedName]; ok {
		metricscollector.DecrementCRDTotal(metricscollector.CloudEventsResource, metricsData.namespace)
	}

	delete(cloudEventPromMetricsMap, namespacedName)
}
