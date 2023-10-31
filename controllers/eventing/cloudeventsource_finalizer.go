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
	cloudEventSourceFinalizer    = "finalizer.keda.sh"
	cloudEventSourceResourceType = "cloudEventSource"
)

func (r *CloudEventSourceReconciler) EnsureEventSourceResourceFinalizer(ctx context.Context, logger logr.Logger, cloudEventSource *eventingv1alpha1.CloudEventSource) error {
	if !util.Contains(cloudEventSource.GetFinalizers(), cloudEventSourceFinalizer) {
		logger.Info(fmt.Sprintf("Adding Finalizer to %s %s/%s", cloudEventSourceResourceType, cloudEventSource.Namespace, cloudEventSource.Name))
		cloudEventSource.SetFinalizers(append(cloudEventSource.GetFinalizers(), cloudEventSourceFinalizer))

		// Update CR
		err := r.Update(ctx, cloudEventSource)
		if err != nil {
			logger.Error(err, fmt.Sprintf("Failed to update %s with a finalizer", cloudEventSourceResourceType), "finalizer", cloudEventSourceFinalizer)
			return err
		}
	}
	return nil
}

func (r *CloudEventSourceReconciler) FinalizeEventSourceResource(ctx context.Context, logger logr.Logger, cloudEventSource *eventingv1alpha1.CloudEventSource, namespacedName string) error {
	if util.Contains(cloudEventSource.GetFinalizers(), cloudEventSourceFinalizer) {
		if err := r.stopEventLoop(logger, cloudEventSource); err != nil {
			return err
		}
		cloudEventSource.SetFinalizers(util.Remove(cloudEventSource.GetFinalizers(), cloudEventSourceFinalizer))
		if err := r.Update(ctx, cloudEventSource); err != nil {
			logger.Error(err, fmt.Sprintf("Failed to update %s after removing a finalizer", cloudEventSourceResourceType), "finalizer", cloudEventSourceFinalizer)
			return err
		}

		r.UpdatePromMetricsOnDelete(namespacedName)
	}

	logger.Info(fmt.Sprintf("Successfully finalized %s", cloudEventSourceResourceType))
	return nil
}
