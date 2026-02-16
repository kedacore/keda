package util

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/kedacore/keda/v2/pkg/common/action"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	eventingv1alpha1 "github.com/kedacore/keda/v2/apis/eventing/v1alpha1"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/eventemitter"
	"github.com/kedacore/keda/v2/pkg/eventreason"
)

const (
	authenticationFinalizer = "finalizer.keda.sh"
)

type authenticationReconciler interface {
	client.Client
	eventemitter.EventHandler
	UpdatePromMetricsOnDelete(string)
}

func EnsureAuthenticationResourceFinalizer(ctx context.Context, logger logr.Logger, reconciler authenticationReconciler, authResource client.Object) error {
	var authResourceType string
	switch authResource.(type) {
	case *kedav1alpha1.TriggerAuthentication:
		authResourceType = "TriggerAuthentication"
	case *kedav1alpha1.ClusterTriggerAuthentication:
		authResourceType = "ClusterTriggerAuthentication"
	}

	if !Contains(authResource.GetFinalizers(), authenticationFinalizer) {
		logger.Info(fmt.Sprintf("Adding Finalizer for the %s", authResourceType))
		authResource.SetFinalizers(append(authResource.GetFinalizers(), authenticationFinalizer))

		// Update CR
		err := reconciler.Update(ctx, authResource)
		if err != nil {
			logger.Error(err, fmt.Sprintf("Failed to update %s with a finalizer", authResourceType), "finalizer", authenticationFinalizer)
			return err
		}
	}
	return nil
}

func FinalizeAuthenticationResource(ctx context.Context, logger logr.Logger, reconciler authenticationReconciler, authResource client.Object, namespacedName string) error {
	var authResourceType, reason string
	var cloudEventType eventingv1alpha1.CloudEventType
	switch authResource.(type) {
	case *kedav1alpha1.TriggerAuthentication:
		authResourceType = "TriggerAuthentication"
		reason = eventreason.TriggerAuthenticationDeleted
		cloudEventType = eventingv1alpha1.TriggerAuthenticationRemovedType
	case *kedav1alpha1.ClusterTriggerAuthentication:
		authResourceType = "ClusterTriggerAuthentication"
		reason = eventreason.ClusterTriggerAuthenticationDeleted
		cloudEventType = eventingv1alpha1.ClusterTriggerAuthenticationRemovedType
	}

	if Contains(authResource.GetFinalizers(), authenticationFinalizer) {
		authResource.SetFinalizers(Remove(authResource.GetFinalizers(), authenticationFinalizer))
		if err := reconciler.Update(ctx, authResource); err != nil {
			logger.Error(err, fmt.Sprintf("Failed to update %s after removing a finalizer", authResourceType), "finalizer", authenticationFinalizer)
			return err
		}

		reconciler.UpdatePromMetricsOnDelete(namespacedName)
	}

	logger.Info(fmt.Sprintf("Successfully finalized %s", authResourceType))
	reconciler.Emit(authResource, nil, namespacedName, corev1.EventTypeNormal, cloudEventType, reason, action.Unknown, fmt.Sprintf("%s was deleted", authResourceType))
	return nil
}
