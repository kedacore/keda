/*
Copyright 2024 The KEDA Authors

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
	kedastatus "github.com/kedacore/keda/v2/pkg/status"
	"github.com/kedacore/keda/v2/pkg/util"
)

// ClusterCloudEventSourceReconciler reconciles a EventSource object
type ClusterCloudEventSourceReconciler struct {
	client.Client
	eventEmitter eventemitter.EventHandler

	clusterCloudEventSourceGenerations *sync.Map
	eventSourcePromMetricsMap          map[string]string
	eventSourcePromMetricsLock         *sync.Mutex
}

// NewClusterCloudEventSourceReconciler creates a new ClusterCloudEventSourceReconciler
func NewClusterCloudEventSourceReconciler(c client.Client, e eventemitter.EventHandler) *ClusterCloudEventSourceReconciler {
	return &ClusterCloudEventSourceReconciler{
		Client:                             c,
		eventEmitter:                       e,
		clusterCloudEventSourceGenerations: &sync.Map{},
		eventSourcePromMetricsMap:          make(map[string]string),
		eventSourcePromMetricsLock:         &sync.Mutex{},
	}
}

// +kubebuilder:rbac:groups=eventing.keda.sh,resources=clustercloudeventsources;clustercloudeventsources/status,verbs="*"

// Reconcile performs reconciliation on the identified EventSource resource based on the request information passed, returns the result and an error (if any).
//
//nolint:dupl
func (r *ClusterCloudEventSourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)

	// Fetch the EventSource instance
	clustercloudEventSource := &eventingv1alpha1.ClusterCloudEventSource{}
	err := r.Client.Get(ctx, req.NamespacedName, clustercloudEventSource)
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

	reqLogger.Info("Reconciling ClusterCloudEventSource")

	if !clustercloudEventSource.GetDeletionTimestamp().IsZero() {
		return ctrl.Result{}, FinalizeCloudEventSourceResource(ctx, reqLogger, r, clustercloudEventSource, req.NamespacedName.String())
	}
	r.updatePromMetrics(clustercloudEventSource, req.NamespacedName.String())

	// ensure finalizer is set on this CR
	if err := EnsureCloudEventSourceResourceFinalizer(ctx, reqLogger, r, clustercloudEventSource); err != nil {
		return ctrl.Result{}, err
	}

	// ensure Status Conditions are initialized
	if !clustercloudEventSource.Status.Conditions.AreInitialized() {
		conditions := eventingv1alpha1.GetCloudEventSourceInitializedConditions()
		if err := kedastatus.SetStatusConditions(ctx, r.Client, reqLogger, clustercloudEventSource, conditions); err != nil {
			return ctrl.Result{}, err
		}
	}

	eventSourceChanged, err := r.cloudEventSourceGenerationChanged(reqLogger, clustercloudEventSource)
	if err != nil {
		return ctrl.Result{}, err
	}

	if eventSourceChanged {
		if r.requestEventLoop(ctx, reqLogger, clustercloudEventSource) != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterCloudEventSourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&eventingv1alpha1.ClusterCloudEventSource{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		WithEventFilter(util.IgnoreOtherNamespaces()).
		Complete(r)
}

// requestEventLoop tries to start EventLoop handler for the respective EventSource
func (r *ClusterCloudEventSourceReconciler) requestEventLoop(ctx context.Context, logger logr.Logger, eventSource eventingv1alpha1.CloudEventSourceInterface) error {
	logger.V(1).Info("Notify eventHandler of an update in eventSource", "name", eventSource.GetName())

	key, err := cache.MetaNamespaceKeyFunc(eventSource)
	if err != nil {
		logger.Error(err, "error getting key for eventSource")
		return err
	}

	if err = r.eventEmitter.HandleCloudEventSource(ctx, eventSource); err != nil {
		return err
	}

	// store ClusterCloudEventSource's current Generation
	r.clusterCloudEventSourceGenerations.Store(key, eventSource.GetGeneration())

	return nil
}

// stopEventLoop stops EventLoop handler for the respective EventSource
func (r *ClusterCloudEventSourceReconciler) StopEventLoop(logger logr.Logger, obj client.Object) error {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		logger.Error(err, "error getting key for eventSource")
		return err
	}

	if err := r.eventEmitter.DeleteCloudEventSource(obj.(*eventingv1alpha1.ClusterCloudEventSource)); err != nil {
		return err
	}
	// delete CloudEventSource's current Generation
	r.clusterCloudEventSourceGenerations.Delete(key)
	return nil
}

// eventSourceGenerationChanged returns true if ClusterCloudEventSource's Generation was changed, ie. EventSource.Spec was changed
func (r *ClusterCloudEventSourceReconciler) cloudEventSourceGenerationChanged(logger logr.Logger, eventSource *eventingv1alpha1.ClusterCloudEventSource) (bool, error) {
	key, err := cache.MetaNamespaceKeyFunc(eventSource)
	if err != nil {
		logger.Error(err, "error getting key for eventSource")
		return true, err
	}

	value, loaded := r.clusterCloudEventSourceGenerations.Load(key)
	if loaded {
		generation := value.(int64)
		if generation == eventSource.Generation {
			return false, nil
		}
	}
	return true, nil
}

func (r *ClusterCloudEventSourceReconciler) updatePromMetrics(eventSource *eventingv1alpha1.ClusterCloudEventSource, namespacedName string) {
	r.eventSourcePromMetricsLock.Lock()
	defer r.eventSourcePromMetricsLock.Unlock()

	if ns, ok := r.eventSourcePromMetricsMap[namespacedName]; ok {
		metricscollector.DecrementCRDTotal(metricscollector.CloudEventSourceResource, ns)
	}

	metricscollector.IncrementCRDTotal(metricscollector.CloudEventSourceResource, eventSource.Namespace)
	r.eventSourcePromMetricsMap[namespacedName] = eventSource.Namespace
}

// UpdatePromMetricsOnDelete is idempotent, so it can be called multiple times without side-effects
func (r *ClusterCloudEventSourceReconciler) UpdatePromMetricsOnDelete(namespacedName string) {
	r.eventSourcePromMetricsLock.Lock()
	defer r.eventSourcePromMetricsLock.Unlock()

	if ns, ok := r.eventSourcePromMetricsMap[namespacedName]; ok {
		metricscollector.DecrementCRDTotal(metricscollector.CloudEventSourceResource, ns)
	}

	delete(r.eventSourcePromMetricsMap, namespacedName)
}
