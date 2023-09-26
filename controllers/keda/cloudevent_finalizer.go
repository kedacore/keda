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
	"fmt"

	"github.com/go-logr/logr"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/controllers/keda/util"
)

const (
	cloudEventsFinalizer = "finalizer.keda.sh"
)

func (r *CloudEventsReconciler) EnsureCloudEventsResourceFinalizer(ctx context.Context, logger logr.Logger, cloudEvent *kedav1alpha1.CloudEvent) error {
	//TODO: Add switch when adding clustercloudevents
	cloudEventsResourceType := "CloudEvents"

	if !util.Contains(cloudEvent.GetFinalizers(), cloudEventsFinalizer) {
		logger.Info(fmt.Sprintf("Adding Finalizer for the %s", cloudEventsResourceType))
		cloudEvent.SetFinalizers(append(cloudEvent.GetFinalizers(), cloudEventsFinalizer))

		// Update CR
		err := r.Update(ctx, cloudEvent)
		if err != nil {
			logger.Error(err, fmt.Sprintf("Failed to update %s with a finalizer", cloudEventsResourceType), "finalizer", cloudEventsFinalizer)
			return err
		}
	}
	return nil
}

func (r *CloudEventsReconciler) FinalizeCloudEventsResource(ctx context.Context, logger logr.Logger, cloudEvent *kedav1alpha1.CloudEvent, namespacedName string) error {
	cloudEventsResourceType := "CloudEvents"

	if util.Contains(cloudEvent.GetFinalizers(), cloudEventsFinalizer) {
		if err := r.stopEventLoop(logger, cloudEvent); err != nil {
			return err
		}
		cloudEvent.SetFinalizers(util.Remove(cloudEvent.GetFinalizers(), cloudEventsFinalizer))
		if err := r.Update(ctx, cloudEvent); err != nil {
			logger.Error(err, fmt.Sprintf("Failed to update %s after removing a finalizer", cloudEventsResourceType), "finalizer", cloudEventsFinalizer)
			return err
		}

		r.UpdatePromMetricsOnDelete(namespacedName)
	}

	logger.Info(fmt.Sprintf("Successfully finalized %s", cloudEventsResourceType))
	return nil
}
