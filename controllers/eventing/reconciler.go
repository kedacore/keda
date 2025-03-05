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
	"sigs.k8s.io/controller-runtime/pkg/client"

	eventingv1alpha1 "github.com/kedacore/keda/v2/apis/eventing/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/eventemitter"
	kedastatus "github.com/kedacore/keda/v2/pkg/status"
)

type cloudEventSourceReconcilerInterface interface {
	GetClient() client.Client
	GetEventEmitter() eventemitter.EventHandler
	GetCloudEventSourceGeneration() *sync.Map
	UpdatePromMetrics(eventSource eventingv1alpha1.CloudEventSourceInterface, namespacedName string)
	UpdatePromMetricsOnDelete(namespacedName string)
}

func Reconcile(ctx context.Context, reqLogger logr.Logger, r cloudEventSourceReconcilerInterface, req ctrl.Request, cloudEventSource eventingv1alpha1.CloudEventSourceInterface) (ctrl.Result, error) {
	err := r.GetClient().Get(ctx, req.NamespacedName, cloudEventSource)
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

	reqLogger.Info("Reconciling CloudEventSource")

	if !cloudEventSource.GetDeletionTimestamp().IsZero() {
		return ctrl.Result{}, FinalizeCloudEventSourceResource(ctx, reqLogger, r, cloudEventSource, req.NamespacedName.String())
	}
	r.UpdatePromMetrics(cloudEventSource, req.NamespacedName.String())

	// ensure finalizer is set on this CR
	if err := EnsureCloudEventSourceResourceFinalizer(ctx, reqLogger, r, cloudEventSource); err != nil {
		return ctrl.Result{}, err
	}

	// ensure Status Conditions are initialized
	if !cloudEventSource.GetStatus().Conditions.AreInitialized() {
		conditions := eventingv1alpha1.GetCloudEventSourceInitializedConditions()
		if err := kedastatus.SetStatusConditions(ctx, r.GetClient(), reqLogger, cloudEventSource, conditions); err != nil {
			return ctrl.Result{}, err
		}
	}

	eventSourceChanged, err := CloudEventSourceGenerationChanged(reqLogger, r, cloudEventSource)
	if err != nil {
		return ctrl.Result{}, err
	}

	if eventSourceChanged {
		if err = RequestEventLoop(ctx, reqLogger, r, cloudEventSource); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// requestEventLoop tries to start EventLoop handler for the respective EventSource
func RequestEventLoop(ctx context.Context, logger logr.Logger, r cloudEventSourceReconcilerInterface, eventSourceI eventingv1alpha1.CloudEventSourceInterface) error {
	logger.V(1).Info("Notify eventHandler of an update in eventSource")

	key, err := cache.MetaNamespaceKeyFunc(eventSourceI)
	if err != nil {
		logger.Error(err, "error getting key for eventSource")
		return err
	}

	if err = r.GetEventEmitter().HandleCloudEventSource(ctx, eventSourceI); err != nil {
		return err
	}

	// store CloudEventSource's current Generation
	r.GetCloudEventSourceGeneration().Store(key, eventSourceI.GetGeneration())
	return nil
}

// stopEventLoop stops EventLoop handler for the respective EventSource
func StopEventLoop(logger logr.Logger, r cloudEventSourceReconcilerInterface, obj client.Object) error {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		logger.Error(err, "error getting key for eventSource")
		return err
	}

	if err := r.GetEventEmitter().DeleteCloudEventSource(obj.(eventingv1alpha1.CloudEventSourceInterface)); err != nil {
		return err
	}
	// delete CloudEventSource's current Generation
	r.GetCloudEventSourceGeneration().Delete(key)
	return nil
}

// eventSourceGenerationChanged returns true if CloudEventSource's Generation was changed, ie. EventSource.Spec was changed
func CloudEventSourceGenerationChanged(logger logr.Logger, r cloudEventSourceReconcilerInterface, eventSourceI eventingv1alpha1.CloudEventSourceInterface) (bool, error) {
	key, err := cache.MetaNamespaceKeyFunc(eventSourceI)
	if err != nil {
		logger.Error(err, "error getting key for eventSource")
		return true, err
	}

	value, loaded := r.GetCloudEventSourceGeneration().Load(key)
	if loaded {
		generation := value.(int64)
		if generation == eventSourceI.GetGeneration() {
			return false, nil
		}
	}
	return true, nil
}
