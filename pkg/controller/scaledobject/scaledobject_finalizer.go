package scaledobject

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/client-go/tools/cache"

	kedav1alpha1 "github.com/kedacore/keda/pkg/apis/keda/v1alpha1"
)

const (
	scaledObjectFinalizer = "finalizer.keda.k8s.io"
)

// finalizeScaledObject is stopping ScaleLoop for the respective ScaleObject
func (r *ReconcileScaledObject) finalizeScaledObject(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) error {
	key, err := cache.MetaNamespaceKeyFunc(scaledObject)
	if err != nil {
		logger.Error(err, "Error getting key for scaledObject (%s/%s)", scaledObject.GetNamespace(), scaledObject.GetName())
		return err
	}

	result, ok := r.scaleLoopContexts.Load(key)
	if ok {
		cancel, ok := result.(context.CancelFunc)
		if ok {
			cancel()
		}
		r.scaleLoopContexts.Delete(key)
	} else {
		logger.V(1).Info("ScaleObject was not found in controller cache", "key", key)
	}

	logger.Info("Successfully finalized ScaledObject")
	return nil
}

// addFinalizer adds finalizer to the ScaledObject
func (r *ReconcileScaledObject) addFinalizer(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) error {
	logger.Info("Adding Finalizer for the ScaledObject")
	scaledObject.SetFinalizers(append(scaledObject.GetFinalizers(), scaledObjectFinalizer))

	// Update CR
	err := r.client.Update(context.TODO(), scaledObject)
	if err != nil {
		logger.Error(err, "Failed to update ScaledObject with finalizer")
		return err
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func remove(list []string, s string) []string {
	for i, v := range list {
		if v == s {
			list = append(list[:i], list[i+1:]...)
		}
	}
	return list
}
