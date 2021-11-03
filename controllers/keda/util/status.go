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

package util

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

// SetStatusConditions patches given object with passed list of conditions based on the object's type or returns an error.
func SetStatusConditions(ctx context.Context, client runtimeclient.StatusClient, logger logr.Logger, object interface{}, conditions *kedav1alpha1.Conditions) error {
	var patch runtimeclient.Patch

	runtimeObj := object.(runtimeclient.Object)
	switch obj := runtimeObj.(type) {
	case *kedav1alpha1.ScaledObject:
		patch = runtimeclient.MergeFrom(obj.DeepCopy())
		obj.Status.Conditions = *conditions
	case *kedav1alpha1.ScaledJob:
		patch = runtimeclient.MergeFrom(obj.DeepCopy())
		obj.Status.Conditions = *conditions
	default:
		err := fmt.Errorf("unknown scalable object type %v", obj)
		logger.Error(err, "Failed to patch Objects Status with Conditions")
		return err
	}

	err := client.Status().Patch(ctx, runtimeObj, patch)
	if err != nil {
		logger.Error(err, "Failed to patch Objects Status with Conditions")
	}
	return err
}

// UpdateScaledObjectStatus patches the given ScaledObject with the updated status passed to it or returns an error.
func UpdateScaledObjectStatus(ctx context.Context, client runtimeclient.StatusClient, logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject, status *kedav1alpha1.ScaledObjectStatus) error {
	patch := runtimeclient.MergeFrom(scaledObject.DeepCopy())
	scaledObject.Status = *status
	err := client.Status().Patch(ctx, scaledObject, patch)
	if err != nil {
		logger.Error(err, "Failed to patch ScaledObjects Status")
	}
	return err
}
