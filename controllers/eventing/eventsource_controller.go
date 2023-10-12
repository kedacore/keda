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

// EventSourceReconciler reconciles a EventSource object
type EventSourceReconciler struct {
	client.Client
	EventEmitter eventemitter.EventEmitter

	eventSourceGenerations *sync.Map
}

type eventSourceMetricsData struct {
	namespace string
}

var (
	eventSourcePromMetricsMap  map[string]eventSourceMetricsData
	eventSourcePromMetricsLock *sync.Mutex
)

func init() {
	eventSourcePromMetricsMap = make(map[string]eventSourceMetricsData)
	eventSourcePromMetricsLock = &sync.Mutex{}
}

// +kubebuilder:rbac:groups=eventing.keda.sh,resources=eventsources;eventsources/status,verbs="*"

// Reconcile performs reconciliation on the identified EventSource resource based on the request information passed, returns the result and an error (if any).
func (r *EventSourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)

	// Fetch the EventSource instance
	eventSource := &eventingv1alpha1.EventSource{}
	err := r.Client.Get(ctx, req.NamespacedName, eventSource)
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

	if eventSource.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, r.FinalizeEventSourceResource(ctx, reqLogger, eventSource, req.NamespacedName.String())
	}
	r.updatePromMetrics(eventSource, req.NamespacedName.String())

	// ensure finalizer is set on this CR
	if err := r.EnsureEventSourceResourceFinalizer(ctx, reqLogger, eventSource); err != nil {
		return ctrl.Result{}, err
	}

	eventSourceChanged, err := r.eventSourceGenerationChanged(reqLogger, eventSource)
	if err != nil {
		return ctrl.Result{}, err
	}

	if eventSourceChanged {
		if r.requestEventLoop(ctx, reqLogger, eventSource) != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EventSourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.eventSourceGenerations = &sync.Map{}
	return ctrl.NewControllerManagedBy(mgr).
		For(&eventingv1alpha1.EventSource{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}

// requestEventLoop tries to start EventLoop handler for the respective EventSource
func (r *EventSourceReconciler) requestEventLoop(ctx context.Context, logger logr.Logger, eventSource *eventingv1alpha1.EventSource) error {
	logger.V(1).Info("Notify eventHandler of an update in eventSource")

	key, err := cache.MetaNamespaceKeyFunc(eventSource)
	if err != nil {
		logger.Error(err, "error getting key for eventSource")
		return err
	}

	if err = r.EventEmitter.HandleEventSource(ctx, eventSource); err != nil {
		return err
	}

	// store EventSource's current Generation
	r.eventSourceGenerations.Store(key, eventSource.Generation)

	return nil
}

// stopEventLoop stops EventLoop handler for the respective EventSource
func (r *EventSourceReconciler) stopEventLoop(logger logr.Logger, eventSource *eventingv1alpha1.EventSource) error {
	key, err := cache.MetaNamespaceKeyFunc(eventSource)
	if err != nil {
		logger.Error(err, "error getting key for eventSource")
		return err
	}

	if err := r.EventEmitter.DeleteEventSource(eventSource); err != nil {
		return err
	}
	// delete EventSource's current Generation
	r.eventSourceGenerations.Delete(key)
	return nil
}

// eventSourceGenerationChanged returns true if EventSource's Generation was changed, ie. EventSource.Spec was changed
func (r *EventSourceReconciler) eventSourceGenerationChanged(logger logr.Logger, eventSource *eventingv1alpha1.EventSource) (bool, error) {
	key, err := cache.MetaNamespaceKeyFunc(eventSource)
	if err != nil {
		logger.Error(err, "error getting key for eventSource")
		return true, err
	}

	value, loaded := r.eventSourceGenerations.Load(key)
	if loaded {
		generation := value.(int64)
		if generation == eventSource.Generation {
			return false, nil
		}
	}
	return true, nil
}

func (r *EventSourceReconciler) updatePromMetrics(eventSource *eventingv1alpha1.EventSource, namespacedName string) {
	eventSourcePromMetricsLock.Lock()
	defer eventSourcePromMetricsLock.Unlock()

	if metricsData, ok := eventSourcePromMetricsMap[namespacedName]; ok {
		metricscollector.DecrementCRDTotal(metricscollector.EventSourceResource, metricsData.namespace)
	}

	metricscollector.IncrementCRDTotal(metricscollector.EventSourceResource, eventSource.Namespace)
	eventSourcePromMetricsMap[namespacedName] = eventSourceMetricsData{namespace: eventSource.Namespace}
}

// UpdatePromMetricsOnDelete is idempotent, so it can be called multiple times without side-effects
func (r *EventSourceReconciler) UpdatePromMetricsOnDelete(namespacedName string) {
	eventSourcePromMetricsLock.Lock()
	defer eventSourcePromMetricsLock.Unlock()

	if metricsData, ok := eventSourcePromMetricsMap[namespacedName]; ok {
		metricscollector.DecrementCRDTotal(metricscollector.EventSourceResource, metricsData.namespace)
	}

	delete(eventSourcePromMetricsMap, namespacedName)
}
