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

package eventing

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

	eventingv1alpha1 "github.com/kedacore/keda/v2/apis/eventing/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/eventemitter"
	"github.com/kedacore/keda/v2/pkg/metricscollector"
)

// CloudEventSourceReconciler reconciles a EventSource object
type CloudEventSourceReconciler struct {
	client.Client
	EventEmitter eventemitter.EventEmitter

	cloudEventSourceGenerations *sync.Map
}

type cloudEventSourceMetricsData struct {
	namespace string
}

var (
	eventSourcePromMetricsMap  map[string]cloudEventSourceMetricsData
	eventSourcePromMetricsLock *sync.Mutex
)

func init() {
	eventSourcePromMetricsMap = make(map[string]cloudEventSourceMetricsData)
	eventSourcePromMetricsLock = &sync.Mutex{}
}

// +kubebuilder:rbac:groups=eventing.keda.sh,resources=eventsources;eventsources/status,verbs="*"

// Reconcile performs reconciliation on the identified EventSource resource based on the request information passed, returns the result and an error (if any).
func (r *CloudEventSourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)

	// Fetch the EventSource instance
	cloudEventSource := &eventingv1alpha1.CloudEventSource{}
	err := r.Client.Get(ctx, req.NamespacedName, cloudEventSource)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request eventSource not found, could have been deleted after reconcile request.
			// Owned eventSource are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "failed to get EventSource")
		return ctrl.Result{}, err
	}

	reqLogger.Info("Reconciling EventSource")

	if cloudEventSource.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, r.FinalizeEventSourceResource(ctx, reqLogger, cloudEventSource, req.NamespacedName.String())
	}
	r.updatePromMetrics(cloudEventSource, req.NamespacedName.String())

	// ensure finalizer is set on this CR
	if err := r.EnsureEventSourceResourceFinalizer(ctx, reqLogger, cloudEventSource); err != nil {
		return ctrl.Result{}, err
	}

	eventSourceChanged, err := r.cloudEventSourceGenerationChanged(reqLogger, cloudEventSource)
	if err != nil {
		return ctrl.Result{}, err
	}

	if eventSourceChanged {
		if r.requestEventLoop(ctx, reqLogger, cloudEventSource) != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CloudEventSourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.cloudEventSourceGenerations = &sync.Map{}
	return ctrl.NewControllerManagedBy(mgr).
		For(&eventingv1alpha1.CloudEventSource{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}

// requestEventLoop tries to start EventLoop handler for the respective EventSource
func (r *CloudEventSourceReconciler) requestEventLoop(ctx context.Context, logger logr.Logger, eventSource *eventingv1alpha1.CloudEventSource) error {
	logger.V(1).Info("Notify eventHandler of an update in eventSource")

	key, err := cache.MetaNamespaceKeyFunc(eventSource)
	if err != nil {
		logger.Error(err, "error getting key for eventSource")
		return err
	}

	if err = r.EventEmitter.HandleCloudEventSource(ctx, eventSource); err != nil {
		return err
	}

	// store CloudEventSource's current Generation
	r.cloudEventSourceGenerations.Store(key, eventSource.Generation)

	return nil
}

// stopEventLoop stops EventLoop handler for the respective EventSource
func (r *CloudEventSourceReconciler) stopEventLoop(logger logr.Logger, eventSource *eventingv1alpha1.CloudEventSource) error {
	key, err := cache.MetaNamespaceKeyFunc(eventSource)
	if err != nil {
		logger.Error(err, "error getting key for eventSource")
		return err
	}

	if err := r.EventEmitter.DeleteCloudEventSource(eventSource); err != nil {
		return err
	}
	// delete CloudEventSource's current Generation
	r.cloudEventSourceGenerations.Delete(key)
	return nil
}

// eventSourceGenerationChanged returns true if CloudEventSource's Generation was changed, ie. EventSource.Spec was changed
func (r *CloudEventSourceReconciler) cloudEventSourceGenerationChanged(logger logr.Logger, eventSource *eventingv1alpha1.CloudEventSource) (bool, error) {
	key, err := cache.MetaNamespaceKeyFunc(eventSource)
	if err != nil {
		logger.Error(err, "error getting key for eventSource")
		return true, err
	}

	value, loaded := r.cloudEventSourceGenerations.Load(key)
	if loaded {
		generation := value.(int64)
		if generation == eventSource.Generation {
			return false, nil
		}
	}
	return true, nil
}

func (r *CloudEventSourceReconciler) updatePromMetrics(eventSource *eventingv1alpha1.CloudEventSource, namespacedName string) {
	eventSourcePromMetricsLock.Lock()
	defer eventSourcePromMetricsLock.Unlock()

	if metricsData, ok := eventSourcePromMetricsMap[namespacedName]; ok {
		metricscollector.DecrementCRDTotal(metricscollector.CloudEventSourceResource, metricsData.namespace)
	}

	metricscollector.IncrementCRDTotal(metricscollector.CloudEventSourceResource, eventSource.Namespace)
	eventSourcePromMetricsMap[namespacedName] = cloudEventSourceMetricsData{namespace: eventSource.Namespace}
}

// UpdatePromMetricsOnDelete is idempotent, so it can be called multiple times without side-effects
func (r *CloudEventSourceReconciler) UpdatePromMetricsOnDelete(namespacedName string) {
	eventSourcePromMetricsLock.Lock()
	defer eventSourcePromMetricsLock.Unlock()

	if metricsData, ok := eventSourcePromMetricsMap[namespacedName]; ok {
		metricscollector.DecrementCRDTotal(metricscollector.CloudEventSourceResource, metricsData.namespace)
	}

	delete(eventSourcePromMetricsMap, namespacedName)
}
