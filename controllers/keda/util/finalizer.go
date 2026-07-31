package util

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	eventingv1alpha1 "github.com/kedacore/keda/v2/apis/eventing/v1alpha1"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/eventemitter"
	"github.com/kedacore/keda/v2/pkg/eventreason"
)

const (
	authenticationFinalizer = "finalizer.keda.sh"
)

// mutateFinalizerWithRetry fetches a fresh copy of obj, applies mutate to its
// finalizer list, and commits the change with a conflict-safe Patch. Unlike a
// plain Update, this retries on resourceVersion conflicts within the same
// call instead of failing the caller's entire reconcile, and syncs the
// caller's obj with the patched state on success.
func mutateFinalizerWithRetry(ctx context.Context, c client.Client, obj client.Object, mutate func(client.Object)) error {
	key := client.ObjectKeyFromObject(obj)
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		fresh := obj.DeepCopyObject().(client.Object)
		if err := c.Get(ctx, key, fresh); err != nil {
			return err
		}

		original := fresh.DeepCopyObject().(client.Object)
		mutate(fresh)

		if reflect.DeepEqual(original.GetFinalizers(), fresh.GetFinalizers()) {
			return nil
		}

		if err := c.Patch(ctx, fresh, client.MergeFrom(original)); err != nil {
			return err
		}

		obj.SetFinalizers(fresh.GetFinalizers())
		obj.SetResourceVersion(fresh.GetResourceVersion())
		return nil
	})
}

// AddFinalizer ensures finalizer is present on obj, retrying on conflict.
func AddFinalizer(ctx context.Context, c client.Client, obj client.Object, finalizer string) error {
	return mutateFinalizerWithRetry(ctx, c, obj, func(o client.Object) {
		if !Contains(o.GetFinalizers(), finalizer) {
			o.SetFinalizers(append(o.GetFinalizers(), finalizer))
		}
	})
}

// RemoveFinalizer ensures finalizer is absent from obj, retrying on conflict.
func RemoveFinalizer(ctx context.Context, c client.Client, obj client.Object, finalizer string) error {
	return mutateFinalizerWithRetry(ctx, c, obj, func(o client.Object) {
		o.SetFinalizers(Remove(o.GetFinalizers(), finalizer))
	})
}

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
		if err := AddFinalizer(ctx, reconciler, authResource, authenticationFinalizer); err != nil {
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
		if err := RemoveFinalizer(ctx, reconciler, authResource, authenticationFinalizer); err != nil {
			logger.Error(err, fmt.Sprintf("Failed to update %s after removing a finalizer", authResourceType), "finalizer", authenticationFinalizer)
			return err
		}

		reconciler.UpdatePromMetricsOnDelete(namespacedName)
	}

	logger.Info(fmt.Sprintf("Successfully finalized %s", authResourceType))
	reconciler.Emit(authResource, namespacedName, corev1.EventTypeNormal, cloudEventType, reason, fmt.Sprintf("%s was deleted", authResourceType))
	return nil
}
