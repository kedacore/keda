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

package service

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/knative/pkg/controller"
	"github.com/knative/pkg/kmp"
	"github.com/knative/pkg/logging"
	"github.com/knative/serving/pkg/apis/serving"
	"github.com/knative/serving/pkg/apis/serving/v1alpha1"
	servinginformers "github.com/knative/serving/pkg/client/informers/externalversions/serving/v1alpha1"
	listers "github.com/knative/serving/pkg/client/listers/serving/v1alpha1"
	"github.com/knative/serving/pkg/reconciler"
	"github.com/knative/serving/pkg/reconciler/v1alpha1/service/resources"
	resourcenames "github.com/knative/serving/pkg/reconciler/v1alpha1/service/resources/names"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

const (
	// ReconcilerName is the name of the reconciler
	ReconcilerName      = "Services"
	controllerAgentName = "service-controller"
)

// Reconciler implements controller.Reconciler for Service resources.
type Reconciler struct {
	*reconciler.Base

	// listers index properties about resources
	serviceLister       listers.ServiceLister
	configurationLister listers.ConfigurationLister
	routeLister         listers.RouteLister
}

// Check that our Reconciler implements controller.Reconciler
var _ controller.Reconciler = (*Reconciler)(nil)

// NewController initializes the controller and is called by the generated code
// Registers eventhandlers to enqueue events
func NewController(
	opt reconciler.Options,
	serviceInformer servinginformers.ServiceInformer,
	configurationInformer servinginformers.ConfigurationInformer,
	routeInformer servinginformers.RouteInformer,
) *controller.Impl {

	c := &Reconciler{
		Base:                reconciler.NewBase(opt, controllerAgentName),
		serviceLister:       serviceInformer.Lister(),
		configurationLister: configurationInformer.Lister(),
		routeLister:         routeInformer.Lister(),
	}
	impl := controller.NewImpl(c, c.Logger, ReconcilerName, reconciler.MustNewStatsReporter(ReconcilerName, c.Logger))

	c.Logger.Info("Setting up event handlers")
	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    impl.Enqueue,
		UpdateFunc: controller.PassNew(impl.Enqueue),
		DeleteFunc: impl.Enqueue,
	})

	configurationInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: controller.Filter(v1alpha1.SchemeGroupVersion.WithKind("Service")),
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc:    impl.EnqueueControllerOf,
			UpdateFunc: controller.PassNew(impl.EnqueueControllerOf),
			DeleteFunc: impl.EnqueueControllerOf,
		},
	})

	routeInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: controller.Filter(v1alpha1.SchemeGroupVersion.WithKind("Service")),
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc:    impl.EnqueueControllerOf,
			UpdateFunc: controller.PassNew(impl.EnqueueControllerOf),
			DeleteFunc: impl.EnqueueControllerOf,
		},
	})

	return impl
}

// Reconcile compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Service resource
// with the current status of the resource.
func (c *Reconciler) Reconcile(ctx context.Context, key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		c.Logger.Errorf("invalid resource key: %s", key)
		return nil
	}
	logger := logging.FromContext(ctx)

	// Get the Service resource with this namespace/name
	original, err := c.serviceLister.Services(namespace).Get(name)
	if apierrs.IsNotFound(err) {
		// The resource may no longer exist, in which case we stop processing.
		logger.Errorf("service %q in work queue no longer exists", key)
		return nil
	} else if err != nil {
		return err
	}

	// Don't modify the informers copy
	service := original.DeepCopy()

	if service.Spec.Manual != nil {
		// We do not know the status when in manual mode. The Route can be
		// updated with Configurations not known to the Service which would
		// make attempts to display status potentially incorrect
		service.Status.SetManualStatus()
	} else {
		// Reconcile this copy of the service and then write back any status
		// updates regardless of whether the reconciliation errored out.
		err = c.reconcile(ctx, service)
	}
	if equality.Semantic.DeepEqual(original.Status, service.Status) {
		// If we didn't change anything then don't call updateStatus.
		// This is important because the copy we loaded from the informer's
		// cache may be stale and we don't want to overwrite a prior update
		// to status with this stale state.

	} else if _, uErr := c.updateStatus(service); uErr != nil {
		logger.Warn("Failed to update service status", zap.Error(uErr))
		c.Recorder.Eventf(service, corev1.EventTypeWarning, "UpdateFailed",
			"Failed to update status for Service %q: %v", service.Name, uErr)
		return uErr
	} else if err == nil {
		// If there was a difference and there was no error.
		c.Recorder.Eventf(service, corev1.EventTypeNormal, "Updated", "Updated Service %q", service.GetName())
	}
	return err
}

func (c *Reconciler) reconcile(ctx context.Context, service *v1alpha1.Service) error {
	logger := logging.FromContext(ctx)
	if service.GetDeletionTimestamp() != nil {
		return nil
	}

	// We may be reading a version of the object that was stored at an older version
	// and may not have had all of the assumed defaults specified.  This won't result
	// in this getting written back to the API Server, but lets downstream logic make
	// assumptions about defaulting.
	service.SetDefaults()

	service.Status.InitializeConditions()

	configName := resourcenames.Configuration(service)
	config, err := c.configurationLister.Configurations(service.Namespace).Get(configName)
	if errors.IsNotFound(err) {
		config, err = c.createConfiguration(service)
		if err != nil {
			logger.Errorf("Failed to create Configuration %q: %v", configName, err)
			c.Recorder.Eventf(service, corev1.EventTypeWarning, "CreationFailed", "Failed to create Configuration %q: %v", configName, err)
			return err
		}
		c.Recorder.Eventf(service, corev1.EventTypeNormal, "Created", "Created Configuration %q", configName)
	} else if err != nil {
		logger.Errorf("Failed to reconcile Service: %q failed to Get Configuration: %q; %v", service.Name, configName, zap.Error(err))
		return err
	} else if !metav1.IsControlledBy(config, service) {
		// Surface an error in the service's status,and return an error.
		service.Status.MarkConfigurationNotOwned(configName)
		return fmt.Errorf("Service: %q does not own Configuration: %q", service.Name, configName)
	} else if config, err = c.reconcileConfiguration(ctx, service, config); err != nil {
		logger.Errorf("Failed to reconcile Service: %q failed to reconcile Configuration: %q; %v", service.Name, configName, zap.Error(err))
		return err
	}

	// Update our Status based on the state of our underlying Configuration.
	service.Status.PropagateConfigurationStatus(&config.Status)

	routeName := resourcenames.Route(service)
	route, err := c.routeLister.Routes(service.Namespace).Get(routeName)
	if errors.IsNotFound(err) {
		route, err = c.createRoute(service)
		if err != nil {
			logger.Errorf("Failed to create Route %q: %v", routeName, err)
			c.Recorder.Eventf(service, corev1.EventTypeWarning, "CreationFailed", "Failed to create Route %q: %v", routeName, err)
			return err
		}
		c.Recorder.Eventf(service, corev1.EventTypeNormal, "Created", "Created Route %q", routeName)
	} else if err != nil {
		logger.Errorf("Failed to reconcile Service: %q failed to Get Route: %q", service.Name, routeName)
		return err
	} else if !metav1.IsControlledBy(route, service) {
		// Surface an error in the service's status, and return an error.
		service.Status.MarkRouteNotOwned(routeName)
		return fmt.Errorf("Service: %q does not own Route: %q", service.Name, routeName)
	} else if route, err = c.reconcileRoute(ctx, service, route); err != nil {
		logger.Errorf("Failed to reconcile Service: %q failed to reconcile Route: %q", service.Name, routeName)
		return err
	}

	// Update our Status based on the state of our underlying Route.
	ss := &service.Status
	ss.PropagateRouteStatus(&route.Status)

	// `manual` is not reconciled.
	if rc := service.Status.GetCondition(v1alpha1.ServiceConditionRoutesReady); rc != nil && rc.Status == corev1.ConditionTrue {
		want, got := route.Spec.DeepCopy().Traffic, route.Status.Traffic
		// Replace `configuration` target with its latest ready revision.
		for idx := range want {
			if want[idx].ConfigurationName == config.Name {
				want[idx].RevisionName = config.Status.LatestReadyRevisionName
				want[idx].ConfigurationName = ""
			}
		}
		if eq, err := kmp.SafeEqual(got, want); !eq || err != nil {
			service.Status.MarkRouteNotYetReady()
		}
	}
	service.Status.ObservedGeneration = service.Generation

	return nil
}

func (c *Reconciler) updateStatus(desired *v1alpha1.Service) (*v1alpha1.Service, error) {
	service, err := c.serviceLister.Services(desired.Namespace).Get(desired.Name)
	if err != nil {
		return nil, err
	}
	// If there's nothing to update, just return.
	if reflect.DeepEqual(service.Status, desired.Status) {
		return service, nil
	}
	becomesReady := desired.Status.IsReady() && !service.Status.IsReady()
	// Don't modify the informers copy.
	existing := service.DeepCopy()
	existing.Status = desired.Status

	svc, err := c.ServingClientSet.ServingV1alpha1().Services(desired.Namespace).UpdateStatus(existing)
	if err == nil && becomesReady {
		duration := time.Now().Sub(svc.ObjectMeta.CreationTimestamp.Time)
		c.Logger.Infof("Service %q became ready after %v", service.Name, duration)
		c.StatsReporter.ReportServiceReady(service.Namespace, service.Name, duration)
	}

	return svc, err
}

func (c *Reconciler) createConfiguration(service *v1alpha1.Service) (*v1alpha1.Configuration, error) {
	cfg, err := resources.MakeConfiguration(service)
	if err != nil {
		return nil, err
	}
	return c.ServingClientSet.ServingV1alpha1().Configurations(service.Namespace).Create(cfg)
}

func configSemanticEquals(desiredConfig, config *v1alpha1.Configuration) bool {
	return equality.Semantic.DeepEqual(desiredConfig.Spec, config.Spec) &&
		equality.Semantic.DeepEqual(desiredConfig.ObjectMeta.Labels, config.ObjectMeta.Labels)
}

// ignoreRouteLabelChange sets desiredConfig[serving.RouteLabelKey] to
// same as config[serving.RouteLabelKey], so that we do nothing about
// the configuration label serving.RouteLabelKey in our
// reconciliation.
func ignoreRouteLabelChange(desiredConfig, config *v1alpha1.Configuration) {
	routeLabel, existed := config.ObjectMeta.Labels[serving.RouteLabelKey]
	if !existed {
		delete(desiredConfig.ObjectMeta.Labels, serving.RouteLabelKey)
	} else {
		desiredConfig.ObjectMeta.Labels[serving.RouteLabelKey] = routeLabel
	}
}

func (c *Reconciler) reconcileConfiguration(ctx context.Context, service *v1alpha1.Service, config *v1alpha1.Configuration) (*v1alpha1.Configuration, error) {
	logger := logging.FromContext(ctx)
	desiredConfig, err := resources.MakeConfiguration(service)
	if err != nil {
		return nil, err
	}
	// Route label is automatically set by another reconciler.  We
	// want to ignore that label in our reconciliation here by setting
	// desiredConfig[serving.RouteLabelKey] to the same as
	// config[erving.RouteLabelKey].
	ignoreRouteLabelChange(desiredConfig, config)

	// TODO(#642): Remove this (needed to avoid continuous updates)
	desiredConfig.Spec.DeprecatedGeneration = config.Spec.DeprecatedGeneration

	if configSemanticEquals(desiredConfig, config) {
		// No differences to reconcile.
		return config, nil
	}
	diff, err := kmp.SafeDiff(desiredConfig.Spec, config.Spec)
	if err != nil {
		return nil, fmt.Errorf("failed to diff Configuration: %v", err)
	}
	logger.Infof("Reconciling configuration diff (-desired, +observed): %s", diff)

	// Don't modify the informers copy.
	existing := config.DeepCopy()
	// Preserve the rest of the object (e.g. ObjectMeta except for labels).
	existing.Spec = desiredConfig.Spec
	existing.ObjectMeta.Labels = desiredConfig.ObjectMeta.Labels
	return c.ServingClientSet.ServingV1alpha1().Configurations(service.Namespace).Update(existing)
}

func (c *Reconciler) createRoute(service *v1alpha1.Service) (*v1alpha1.Route, error) {
	route, err := resources.MakeRoute(service)
	if err != nil {
		// This should be unreachable as configuration creation
		// happens first in `reconcile()` and it verifies the edge cases
		// that would make `MakeRoute` fail as well.
		return nil, err
	}
	return c.ServingClientSet.ServingV1alpha1().Routes(service.Namespace).Create(route)
}

func routeSemanticEquals(desiredRoute, route *v1alpha1.Route) bool {
	return equality.Semantic.DeepEqual(desiredRoute.Spec, route.Spec) &&
		equality.Semantic.DeepEqual(desiredRoute.ObjectMeta.Labels, route.ObjectMeta.Labels)
}

func (c *Reconciler) reconcileRoute(ctx context.Context, service *v1alpha1.Service, route *v1alpha1.Route) (*v1alpha1.Route, error) {
	logger := logging.FromContext(ctx)
	desiredRoute, err := resources.MakeRoute(service)
	if err != nil {
		// This should be unreachable as configuration creation
		// happens first in `reconcile()` and it verifies the edge cases
		// that would make `MakeRoute` fail as well.
		return nil, err
	}

	// TODO(#642): Remove this (needed to avoid continuous updates).
	desiredRoute.Spec.DeprecatedGeneration = route.Spec.DeprecatedGeneration

	if routeSemanticEquals(desiredRoute, route) {
		// No differences to reconcile.
		return route, nil
	}
	diff, err := kmp.SafeDiff(desiredRoute.Spec, route.Spec)
	if err != nil {
		return nil, fmt.Errorf("failed to diff Route: %v", err)
	}
	logger.Infof("Reconciling route diff (-desired, +observed): %s", diff)

	// Don't modify the informers copy.
	existing := route.DeepCopy()
	// Preserve the rest of the object (e.g. ObjectMeta except for labels).
	existing.Spec = desiredRoute.Spec
	existing.ObjectMeta.Labels = desiredRoute.ObjectMeta.Labels
	return c.ServingClientSet.ServingV1alpha1().Routes(service.Namespace).Update(existing)
}
