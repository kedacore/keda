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
	"fmt"

	"github.com/go-logr/logr"

	eventingv1alpha1 "github.com/kedacore/keda/v2/apis/eventing/v1alpha1"
	"github.com/kedacore/keda/v2/controllers/keda/util"
)

const (
	eventSourceFinalizer = "finalizer.keda.sh"
)

func (r *EventSourceReconciler) EnsureEventSourceResourceFinalizer(ctx context.Context, logger logr.Logger, eventSource *eventingv1alpha1.EventSource) error {
	//TODO: Add switch when adding clustereventsource
	eventSourceResourceType := "eventSource"

	if !util.Contains(eventSource.GetFinalizers(), eventSourceFinalizer) {
		logger.Info(fmt.Sprintf("Adding Finalizer for the %s", eventSourceResourceType))
		eventSource.SetFinalizers(append(eventSource.GetFinalizers(), eventSourceFinalizer))

		// Update CR
		err := r.Update(ctx, eventSource)
		if err != nil {
			logger.Error(err, fmt.Sprintf("Failed to update %s with a finalizer", eventSourceResourceType), "finalizer", eventSourceFinalizer)
			return err
		}
	}
	return nil
}

func (r *EventSourceReconciler) FinalizeEventSourceResource(ctx context.Context, logger logr.Logger, eventSource *eventingv1alpha1.EventSource, namespacedName string) error {
	eventSourceResourceType := "eventSource"

	if util.Contains(eventSource.GetFinalizers(), eventSourceFinalizer) {
		if err := r.stopEventLoop(logger, eventSource); err != nil {
			return err
		}
		eventSource.SetFinalizers(util.Remove(eventSource.GetFinalizers(), eventSourceFinalizer))
		if err := r.Update(ctx, eventSource); err != nil {
			logger.Error(err, fmt.Sprintf("Failed to update %s after removing a finalizer", eventSourceResourceType), "finalizer", eventSourceFinalizer)
			return err
		}

		r.UpdatePromMetricsOnDelete(namespacedName)
	}

	logger.Info(fmt.Sprintf("Successfully finalized %s", eventSourceResourceType))
	return nil
}
