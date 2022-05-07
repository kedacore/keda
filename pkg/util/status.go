package util

import (
	"context"
	"fmt"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// SetStatusConditions patches given object with passed list of conditions based on the object's type or returns an error.
func SetStatusConditions(ctx context.Context, client runtimeclient.StatusClient, object interface{}, conditions *kedav1alpha1.Conditions) error {
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
		return err
	}

	return client.Status().Patch(ctx, runtimeObj, patch)
}
