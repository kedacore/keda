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
	transform := func(runtimeObj runtimeclient.Object, target interface{}) error {
		conditions, ok := target.(*kedav1alpha1.Conditions)
		if !ok {
			return fmt.Errorf("transform target is not kedav1alpha1.Conditions type %v", target)
		}
		switch obj := runtimeObj.(type) {
		case *kedav1alpha1.ScaledObject:
			obj.Status.Conditions = *conditions
		case *kedav1alpha1.ScaledJob:
			obj.Status.Conditions = *conditions
		default:
		}
		return nil
	}
	return TransformObject(ctx, client, logger, object, conditions, transform)
}

// UpdateScaledObjectStatus patches the given ScaledObject with the updated status passed to it or returns an error.
func UpdateScaledObjectStatus(ctx context.Context, client runtimeclient.StatusClient, logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject, status *kedav1alpha1.ScaledObjectStatus) error {
	transform := func(runtimeObj runtimeclient.Object, target interface{}) error {
		status, ok := target.(*kedav1alpha1.ScaledObjectStatus)
		if !ok {
			return fmt.Errorf("transform target is not kedav1alpha1.ScaledObjectStatus type %v", target)
		}
		switch obj := runtimeObj.(type) {
		case *kedav1alpha1.ScaledObject:
			obj.Status = *status
		default:
		}
		return nil
	}
	return TransformObject(ctx, client, logger, scaledObject, status, transform)
}

// TransformObject patches the given object with the targeted passed to it through a transformer function or returns an error.
func TransformObject(ctx context.Context, client runtimeclient.StatusClient, logger logr.Logger, object interface{}, target interface{}, transform func(runtimeclient.Object, interface{}) error) error {
	var patch runtimeclient.Patch

	runtimeObj := object.(runtimeclient.Object)
	switch obj := runtimeObj.(type) {
	case *kedav1alpha1.ScaledObject:
		patch = runtimeclient.MergeFrom(obj.DeepCopy())
		if err := transform(obj, target); err != nil {
			logger.Error(err, "failed to patch ScaledObject")
			return err
		}
	case *kedav1alpha1.ScaledJob:
		patch = runtimeclient.MergeFrom(obj.DeepCopy())
		if err := transform(obj, target); err != nil {
			logger.Error(err, "failed to patch ScaledJob")
			return err
		}
	default:
		err := fmt.Errorf("unknown scalable object type %v", obj)
		logger.Error(err, "failed to patch Objects")
		return err
	}

	err := client.Status().Patch(ctx, runtimeObj, patch)
	if err != nil {
		logger.Error(err, "failed to patch Objects")
	}
	return err
}
