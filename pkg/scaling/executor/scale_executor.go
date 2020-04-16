package executor

import (
	"context"

	"github.com/go-logr/logr"
	kedav1alpha1 "github.com/kedacore/keda/pkg/apis/keda/v1alpha1"
	"github.com/kedacore/keda/pkg/scalers"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/scale"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// Default cooldown period for a ScaleTarget if no cooldownPeriod is defined on the scaledObject
	defaultCooldownPeriod = 5 * 60 // 5 minutes
)

type ScaleExecutor interface {
	RequestJobScale(ctx context.Context, scalers []scalers.Scaler, scaledObject *kedav1alpha1.ScaledJob)
	RequestScale(ctx context.Context, scalers []scalers.Scaler, scaledObject *kedav1alpha1.ScaledObject)
}

type scaleExecutor struct {
	client           client.Client
	scaleClient      *scale.ScalesGetter
	reconcilerScheme *runtime.Scheme
	logger           logr.Logger
}

func NewScaleExecutor(client client.Client, scaleClient *scale.ScalesGetter, reconcilerScheme *runtime.Scheme) ScaleExecutor {
	return &scaleExecutor{
		client:           client,
		scaleClient:      scaleClient,
		reconcilerScheme: reconcilerScheme,
		logger:           logf.Log.WithName("scaleexecutor"),
	}
}

func (e *scaleExecutor) updateLastActiveTime(ctx context.Context, object interface{}) error {
	key, err := cache.MetaNamespaceKeyFunc(object)
	if err != nil {
		return err
	}

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	for i := 0; i < 10; i++ {
		// ScaledObject's metadata that are not necessary to restart the ScaleLoop were updated (eg. labels)
		// we should try to fetch the scaledObject again and process the update once again
		runtimeObj := object.(runtime.Object)
		logger := e.logger.WithValues("object", runtimeObj)
		logger.V(1).Info("Trying to fetch updated version of object to properly update it's Status")

		if err := e.client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, runtimeObj); err != nil {
			logger.Error(err, "Error getting updated version of object before updating it's Status")
			return err
		}

		now := metav1.Now()
		switch obj := object.(type) {
		case *kedav1alpha1.ScaledObject:
			obj.Status.LastActiveTime = &now
		case *kedav1alpha1.ScaledJob:
			obj.Status.LastActiveTime = &now
		}

		if err := e.client.Status().Update(ctx, runtimeObj); err != nil {
			if errors.IsConflict(err) {
				logger.Error(err, "conflict updating object", "iteration", i)
				continue
			}
			logger.Error(err, "Error updating scaledObject status")
			return err
		}
		logger.V(1).Info("Object's Status was properly updated on re-fetched object")
		break
	}
	return nil
}
