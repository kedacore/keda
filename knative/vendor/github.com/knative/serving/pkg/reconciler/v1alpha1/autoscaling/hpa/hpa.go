/*
Copyright 2018 The Knative Authors

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

package hpa

import (
	"context"
	"fmt"
	"reflect"

	"github.com/knative/pkg/controller"
	"github.com/knative/pkg/logging"
	"github.com/knative/serving/pkg/apis/autoscaling"
	pav1alpha1 "github.com/knative/serving/pkg/apis/autoscaling/v1alpha1"
	informers "github.com/knative/serving/pkg/client/informers/externalversions/autoscaling/v1alpha1"
	listers "github.com/knative/serving/pkg/client/listers/autoscaling/v1alpha1"
	"github.com/knative/serving/pkg/reconciler"
	"github.com/knative/serving/pkg/reconciler/v1alpha1/autoscaling/hpa/resources"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	autoscalingv1informers "k8s.io/client-go/informers/autoscaling/v1"
	autoscalingv1listers "k8s.io/client-go/listers/autoscaling/v1"
	"k8s.io/client-go/tools/cache"
)

const (
	controllerAgentName = "hpa-class-podautoscaler-controller"
)

type Reconciler struct {
	*reconciler.Base

	paLister  listers.PodAutoscalerLister
	hpaLister autoscalingv1listers.HorizontalPodAutoscalerLister
}

var _ controller.Reconciler = (*Reconciler)(nil)

func NewController(
	opts *reconciler.Options,
	paInformer informers.PodAutoscalerInformer,
	hpaInformer autoscalingv1informers.HorizontalPodAutoscalerInformer,
) *controller.Impl {
	c := &Reconciler{
		Base:      reconciler.NewBase(*opts, controllerAgentName),
		paLister:  paInformer.Lister(),
		hpaLister: hpaInformer.Lister(),
	}
	impl := controller.NewImpl(c, c.Logger, "HPA-Class Autoscaling", reconciler.MustNewStatsReporter("HPA-Class Autoscaling", c.Logger))

	c.Logger.Info("Setting up hpa-class event handlers")
	onlyHpaClass := reconciler.AnnotationFilterFunc(autoscaling.ClassAnnotationKey, autoscaling.HPA, false)
	paInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: onlyHpaClass,
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc:    impl.Enqueue,
			UpdateFunc: controller.PassNew(impl.Enqueue),
			DeleteFunc: impl.Enqueue,
		},
	})

	hpaInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: onlyHpaClass,
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc:    impl.EnqueueControllerOf,
			UpdateFunc: controller.PassNew(impl.EnqueueControllerOf),
			DeleteFunc: impl.EnqueueControllerOf,
		},
	})

	return impl
}

func (c *Reconciler) Reconcile(ctx context.Context, key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key %s: %v", key, err))
		return nil
	}
	logger := logging.FromContext(ctx)
	logger.Debug("Reconcile hpa-class PodAutoscaler")

	original, err := c.paLister.PodAutoscalers(namespace).Get(name)
	if errors.IsNotFound(err) {
		logger.Debug("PA no longer exists")
		return c.deleteHpa(ctx, key)
	} else if err != nil {
		return err
	}

	if original.Class() != autoscaling.HPA {
		logger.Warn("Ignoring non-hpa-class PA")
		return nil
	}

	// Don't modify the informer's copy.
	pa := original.DeepCopy()
	// Reconcile this copy of the pa and then write back any status
	// updates regardless of whether the reconciliation errored out.
	err = c.reconcile(ctx, key, pa)
	if equality.Semantic.DeepEqual(original.Status, pa.Status) {
		// If we didn't change anything then don't call updateStatus.
		// This is important because the copy we loaded from the informer's
		// cache may be stale and we don't want to overwrite a prior update
		// to status with this stale state.
	} else {
		if _, err := c.updateStatus(pa); err != nil {
			logger.Warn("Failed to update pa status", zap.Error(err))
			return err
		}
	}
	return err
}

func (c *Reconciler) reconcile(ctx context.Context, key string, pa *pav1alpha1.PodAutoscaler) error {
	logger := logging.FromContext(ctx)

	if pa.GetDeletionTimestamp() != nil {
		return nil
	}

	// We may be reading a version of the object that was stored at an older version
	// and may not have had all of the assumed defaults specified.  This won't result
	// in this getting written back to the API Server, but lets downstream logic make
	// assumptions about defaulting.
	pa.SetDefaults()

	pa.Status.InitializeConditions()
	logger.Debug("PA exists")

	// HPA-class PAs don't yet support scale-to-zero
	pa.Status.MarkActive()

	// HPA-class PA delegates autoscaling to the Kubernetes Horizontal Pod Autoscaler.
	desiredHpa := resources.MakeHPA(pa)
	hpa, err := c.hpaLister.HorizontalPodAutoscalers(pa.Namespace).Get(desiredHpa.Name)
	if errors.IsNotFound(err) {
		logger.Infof("Creating HPA %q", desiredHpa.Name)
		if _, err := c.KubeClientSet.AutoscalingV1().HorizontalPodAutoscalers(pa.Namespace).Create(desiredHpa); err != nil {
			logger.Errorf("Error creating HPA %q: %v", desiredHpa.Name, err)
			pa.Status.MarkResourceFailedCreation("HorizontalPodAutoscaler", desiredHpa.Name)
			return err
		}
	} else if err != nil {
		logger.Errorf("Error getting existing HPA %q: %v", desiredHpa.Name, err)
		return err
	} else if !metav1.IsControlledBy(hpa, pa) {
		// Surface an error in the PodAutoscaler's status, and return an error.
		pa.Status.MarkResourceNotOwned("HorizontalPodAutoscaler", desiredHpa.Name)
		return fmt.Errorf("PodAutoscaler: %q does not own HPA: %q", pa.Name, desiredHpa.Name)
	} else {
		if !equality.Semantic.DeepEqual(desiredHpa.Spec, hpa.Spec) {
			logger.Infof("Updating HPA %q", desiredHpa.Name)
			if _, err := c.KubeClientSet.AutoscalingV1().HorizontalPodAutoscalers(pa.Namespace).Update(desiredHpa); err != nil {
				logger.Errorf("Error updating HPA %q: %v", desiredHpa.Name, err)
				return err
			}
		}
	}
	pa.Status.ObservedGeneration = pa.Generation
	return nil
}

func (c *Reconciler) deleteHpa(ctx context.Context, key string) error {
	logger := logging.FromContext(ctx)

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	err = c.KubeClientSet.AutoscalingV1().HorizontalPodAutoscalers(namespace).Delete(name, nil)
	if errors.IsNotFound(err) {
		// This is fine.
		return nil
	} else if err != nil {
		logger.Errorf("Error deleting HPA %q: %v", name, err)
		return err
	}
	logger.Infof("Deleted HPA %q", name)
	return nil
}

func (c *Reconciler) updateStatus(desired *pav1alpha1.PodAutoscaler) (*pav1alpha1.PodAutoscaler, error) {
	pa, err := c.paLister.PodAutoscalers(desired.Namespace).Get(desired.Name)
	if err != nil {
		return nil, err
	}
	// Check if there is anything to update.
	if !reflect.DeepEqual(pa.Status, desired.Status) {
		// Don't modify the informers copy
		existing := pa.DeepCopy()
		existing.Status = desired.Status
		return c.ServingClientSet.AutoscalingV1alpha1().PodAutoscalers(pa.Namespace).UpdateStatus(existing)
	}
	return pa, nil
}
