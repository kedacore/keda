package keda

import (
	"context"

	"github.com/go-logr/logr"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/controllers/keda/util"
)

func (r *ClusterTriggerAuthenticationReconciler) ensureFinalizer(ctx context.Context, logger logr.Logger, clusterTriggerAuth *kedav1alpha1.ClusterTriggerAuthentication) error {
	return util.EnsureAuthenticationResourceFinalizer(ctx, logger, r, clusterTriggerAuth)
}

func (r *ClusterTriggerAuthenticationReconciler) finalizeClusterTriggerAuthentication(ctx context.Context, logger logr.Logger,
	clusterTriggerAuth *kedav1alpha1.ClusterTriggerAuthentication, namespacedName string) error {
	return util.FinalizeAuthenticationResource(ctx, logger, r, clusterTriggerAuth, namespacedName)
}
