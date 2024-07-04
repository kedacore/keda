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
	"fmt"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kedacore/keda/v2/controllers/keda/util"
)

const (
	cloudEventSourceFinalizer = "finalizer.keda.sh"
)

func EnsureCloudEventSourceResourceFinalizer(ctx context.Context, logger logr.Logger, r cloudEventSourceReconcilerInterface, cloudEventSourceResource client.Object) error {
	if !util.Contains(cloudEventSourceResource.GetFinalizers(), cloudEventSourceFinalizer) {
		logger.Info(fmt.Sprintf("Adding Finalizer for the %s", cloudEventSourceResource.GetName()))
		cloudEventSourceResource.SetFinalizers(append(cloudEventSourceResource.GetFinalizers(), cloudEventSourceFinalizer))

		// Update CR
		err := r.GetClient().Update(ctx, cloudEventSourceResource)
		if err != nil {
			logger.Error(err, fmt.Sprintf("Failed to update %s with a finalizer", cloudEventSourceResource.GetName()), "finalizer", cloudEventSourceFinalizer)
			return err
		}
	}
	return nil
}

func FinalizeCloudEventSourceResource(ctx context.Context, logger logr.Logger, r cloudEventSourceReconcilerInterface, cloudEventSourceResource client.Object, namespacedName string) error {
	if util.Contains(cloudEventSourceResource.GetFinalizers(), cloudEventSourceFinalizer) {
		if err := StopEventLoop(logger, r, cloudEventSourceResource); err != nil {
			return err
		}
		cloudEventSourceResource.SetFinalizers(util.Remove(cloudEventSourceResource.GetFinalizers(), cloudEventSourceFinalizer))
		if err := r.GetClient().Update(ctx, cloudEventSourceResource); err != nil {
			logger.Error(err, fmt.Sprintf("Failed to update %s after removing a finalizer", cloudEventSourceResource.GetName()), "finalizer", cloudEventSourceFinalizer)
			return err
		}

		r.UpdatePromMetricsOnDelete(namespacedName)
	}

	logger.Info(fmt.Sprintf("Successfully finalized %s", cloudEventSourceResource.GetName()))
	return nil
}
