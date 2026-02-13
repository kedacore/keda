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

package status

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	eventingv1alpha1 "github.com/kedacore/keda/v2/apis/eventing/v1alpha1"
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
		case *eventingv1alpha1.CloudEventSource:
			obj.Status.Conditions = *conditions
		case *eventingv1alpha1.ClusterCloudEventSource:
			obj.Status.Conditions = *conditions
		default:
		}
		return nil
	}
	return TransformObject(ctx, client, logger, object, conditions, transform)
}

// UpdateScaledObjectStatus patches the given ScaledObject with the updated status passed to it or returns an error.
func UpdateScaledObjectStatus(ctx context.Context, client runtimeclient.StatusClient, logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject, status *kedav1alpha1.ScaledObjectStatus) error {
	return updateObjectStatus(ctx, client, logger, scaledObject, status)
}

// UpdateScaledJobStatus patches the given ScaledObject with the updated status passed to it or returns an error.
func UpdateScaledJobStatus(ctx context.Context, client runtimeclient.StatusClient, logger logr.Logger, scaledJob *kedav1alpha1.ScaledJob, status *kedav1alpha1.ScaledJobStatus) error {
	return updateObjectStatus(ctx, client, logger, scaledJob, status)
}

// updateObjectStatus patches the given ScaledObject with the updated status passed to it or returns an error.
func updateObjectStatus(ctx context.Context, client runtimeclient.StatusClient, logger logr.Logger, object interface{}, status interface{}) error {
	transform := func(runtimeObj runtimeclient.Object, target interface{}) error {
		switch obj := runtimeObj.(type) {
		case *kedav1alpha1.ScaledObject:
			status, ok := target.(*kedav1alpha1.ScaledObjectStatus)
			if !ok {
				return fmt.Errorf("transform target is not kedav1alpha1.ScaledObjectStatus type %v", target)
			}
			obj.Status = *status
		case *kedav1alpha1.ScaledJob:
			status, ok := target.(*kedav1alpha1.ScaledJobStatus)
			if !ok {
				return fmt.Errorf("transform target is not kedav1alpha1.ScaledJobStatus type %v", target)
			}
			obj.Status = *status
		default:
		}
		return nil
	}
	return TransformObject(ctx, client, logger, object, status, transform)
}

// getTriggerAuth returns TriggerAuthentication/ClusterTriggerAuthentication object and its status from AuthenticationRef or returns an error.
func getTriggerAuth(ctx context.Context, client runtimeclient.Client, triggerAuthRef *kedav1alpha1.AuthenticationRef, namespace string) (runtimeclient.Object, *kedav1alpha1.TriggerAuthenticationStatus, error) {
	if triggerAuthRef == nil {
		return nil, nil, fmt.Errorf("triggerAuthRef is nil")
	}

	switch triggerAuthRef.Kind {
	case "", "TriggerAuthentication":
		triggerAuth := &kedav1alpha1.TriggerAuthentication{}
		err := client.Get(ctx, types.NamespacedName{Name: triggerAuthRef.Name, Namespace: namespace}, triggerAuth)
		if err != nil {
			return nil, nil, err
		}
		return triggerAuth, &triggerAuth.Status, nil
	case "ClusterTriggerAuthentication":
		clusterTriggerAuth := &kedav1alpha1.ClusterTriggerAuthentication{}
		err := client.Get(ctx, types.NamespacedName{Name: triggerAuthRef.Name, Namespace: namespace}, clusterTriggerAuth)
		if err != nil {
			return nil, nil, err
		}
		return clusterTriggerAuth, &clusterTriggerAuth.Status, nil
	default:
		return nil, nil, fmt.Errorf("unknown trigger auth kind %s", triggerAuthRef.Kind)
	}
}

// updateTriggerAuthenticationStatus patches TriggerAuthentication/ClusterTriggerAuthentication from AuthenticationRef with the status that updated by statushanler function or returns an error.
func updateTriggerAuthenticationStatus(ctx context.Context, logger logr.Logger, client runtimeclient.Client, namespace string, triggerAuthRef *kedav1alpha1.AuthenticationRef, statusHandler func(*kedav1alpha1.TriggerAuthenticationStatus) *kedav1alpha1.TriggerAuthenticationStatus) error {
	triggerAuth, triggerAuthStatus, err := getTriggerAuth(ctx, client, triggerAuthRef, namespace)

	if err != nil {
		if k8sErrors.IsNotFound(err) {
			logger.Info("TriggerAuthentication Not Found")
		}
		logger.Error(err, "Failed to get TriggerAuthentication")
		return err
	}

	triggerAuthenticationStatus := statusHandler(triggerAuthStatus.DeepCopy())

	transform := func(runtimeObj runtimeclient.Object, target interface{}) error {
		status, ok := target.(*kedav1alpha1.TriggerAuthenticationStatus)
		if !ok {
			return fmt.Errorf("transform target is not kedav1alpha1.TriggerAuthenticationStatus type %v", target)
		}
		switch obj := runtimeObj.(type) {
		case *kedav1alpha1.TriggerAuthentication:
			obj.Status = *status
		case *kedav1alpha1.ClusterTriggerAuthentication:
			obj.Status = *status
		default:
		}
		return nil
	}

	if err := TransformObject(ctx, client, logger, triggerAuth, triggerAuthenticationStatus, transform); err != nil {
		logger.Error(err, "Failed to update TriggerAuthenticationStatus")
	}

	return err
}

// UpdateTriggerAuthenticationStatusFromTriggers patches triggerAuthenticationStatus From the given Triggers or returns an error.
func UpdateTriggerAuthenticationStatusFromTriggers(ctx context.Context, logger logr.Logger, client runtimeclient.Client, namespace string, scaleTriggers []kedav1alpha1.ScaleTriggers, statusHandler func(*kedav1alpha1.TriggerAuthenticationStatus) *kedav1alpha1.TriggerAuthenticationStatus) (string, error) {
	var errs error
	for _, trigger := range scaleTriggers {
		if trigger.AuthenticationRef == nil {
			continue
		}

		err := updateTriggerAuthenticationStatus(ctx, logger, client, namespace, trigger.AuthenticationRef, statusHandler)
		if err != nil {
			errs = errors.Wrap(errs, err.Error())
		}
	}

	if errs != nil {
		return "Update TriggerAuthentication Status Failed", errs
	}
	return "Update TriggerAuthentication Status Successfully", nil
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
	case *kedav1alpha1.TriggerAuthentication:
		patch = runtimeclient.MergeFrom(obj.DeepCopy())
		if err := transform(obj, target); err != nil {
			logger.Error(err, "failed to patch TriggerAuthentication")
			return err
		}
	case *kedav1alpha1.ClusterTriggerAuthentication:
		patch = runtimeclient.MergeFrom(obj.DeepCopy())
		if err := transform(obj, target); err != nil {
			logger.Error(err, "failed to patch ClusterTriggerAuthentication")
			return err
		}
	case *eventingv1alpha1.CloudEventSource:
		patch = runtimeclient.MergeFrom(obj.DeepCopy())
		if err := transform(obj, target); err != nil {
			logger.Error(err, "failed to patch CloudEventSource")
			return err
		}
	case *eventingv1alpha1.ClusterCloudEventSource:
		patch = runtimeclient.MergeFrom(obj.DeepCopy())
		if err := transform(obj, target); err != nil {
			logger.Error(err, "failed to patch ClusterCloudEventSource")
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
