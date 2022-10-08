package keda

import (
	"context"

	"github.com/go-logr/logr"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/controllers/keda/util"
)

func (r *TriggerAuthenticationReconciler) ensureFinalizer(ctx context.Context, logger logr.Logger, triggerAuth *kedav1alpha1.TriggerAuthentication) error {
	return util.EnsureAuthenticationResourceFinalizer(ctx, logger, r, triggerAuth)
}

func (r *TriggerAuthenticationReconciler) finalizeTriggerAuthentication(ctx context.Context, logger logr.Logger,
	triggerAuth *kedav1alpha1.TriggerAuthentication, namespacedName string) error {
	return util.FinalizeAuthenticationResource(ctx, logger, r, triggerAuth, namespacedName)
}
