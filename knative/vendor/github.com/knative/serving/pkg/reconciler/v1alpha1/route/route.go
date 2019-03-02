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

package route

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	corev1informers "k8s.io/client-go/informers/core/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"

	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	"github.com/knative/pkg/configmap"
	"github.com/knative/pkg/controller"
	"github.com/knative/pkg/logging"
	"github.com/knative/pkg/system"
	"github.com/knative/pkg/tracker"
	"github.com/knative/serving/pkg/apis/networking"
	"github.com/knative/serving/pkg/apis/serving"
	"github.com/knative/serving/pkg/apis/serving/v1alpha1"
	networkinginformers "github.com/knative/serving/pkg/client/informers/externalversions/networking/v1alpha1"
	servinginformers "github.com/knative/serving/pkg/client/informers/externalversions/serving/v1alpha1"
	networkinglisters "github.com/knative/serving/pkg/client/listers/networking/v1alpha1"
	listers "github.com/knative/serving/pkg/client/listers/serving/v1alpha1"
	"github.com/knative/serving/pkg/network"
	"github.com/knative/serving/pkg/reconciler"
	"github.com/knative/serving/pkg/reconciler/v1alpha1/route/config"
	"github.com/knative/serving/pkg/reconciler/v1alpha1/route/resources"
	resourcenames "github.com/knative/serving/pkg/reconciler/v1alpha1/route/resources/names"
	"github.com/knative/serving/pkg/reconciler/v1alpha1/route/traffic"
)

const (
	controllerAgentName = "route-controller"
)

// routeFinalizer is the name that we put into the resource finalizer list, e.g.
//  metadata:
//    finalizers:
//    - routes.serving.knative.dev
var (
	routeResource  = v1alpha1.Resource("routes")
	routeFinalizer = routeResource.String()
)

type configStore interface {
	ToContext(ctx context.Context) context.Context
	WatchConfigs(w configmap.Watcher)
}

// Reconciler implements controller.Reconciler for Route resources.
type Reconciler struct {
	*reconciler.Base

	// Listers index properties about resources
	routeLister          listers.RouteLister
	configurationLister  listers.ConfigurationLister
	revisionLister       listers.RevisionLister
	serviceLister        corev1listers.ServiceLister
	clusterIngressLister networkinglisters.ClusterIngressLister
	configStore          configStore
	tracker              tracker.Interface

	clock system.Clock
}

// Check that our Reconciler implements controller.Reconciler
var _ controller.Reconciler = (*Reconciler)(nil)

// NewController initializes the controller and is called by the generated code
// Registers eventhandlers to enqueue events
// config - client configuration for talking to the apiserver
// si - informer factory shared across all controllers for listening to events and indexing resource properties
// reconcileKey - function for mapping queue keys to resource names
func NewController(
	opt reconciler.Options,
	routeInformer servinginformers.RouteInformer,
	configInformer servinginformers.ConfigurationInformer,
	revisionInformer servinginformers.RevisionInformer,
	serviceInformer corev1informers.ServiceInformer,
	clusterIngressInformer networkinginformers.ClusterIngressInformer,
) *controller.Impl {
	return NewControllerWithClock(opt, routeInformer, configInformer, revisionInformer,
		serviceInformer, clusterIngressInformer, system.RealClock{})
}

func NewControllerWithClock(
	opt reconciler.Options,
	routeInformer servinginformers.RouteInformer,
	configInformer servinginformers.ConfigurationInformer,
	revisionInformer servinginformers.RevisionInformer,
	serviceInformer corev1informers.ServiceInformer,
	clusterIngressInformer networkinginformers.ClusterIngressInformer,
	clock system.Clock,
) *controller.Impl {

	// No need to lock domainConfigMutex yet since the informers that can modify
	// domainConfig haven't started yet.
	c := &Reconciler{
		Base:                 reconciler.NewBase(opt, controllerAgentName),
		routeLister:          routeInformer.Lister(),
		configurationLister:  configInformer.Lister(),
		revisionLister:       revisionInformer.Lister(),
		serviceLister:        serviceInformer.Lister(),
		clusterIngressLister: clusterIngressInformer.Lister(),
		clock:                clock,
	}
	impl := controller.NewImpl(c, c.Logger, "Routes", reconciler.MustNewStatsReporter("Routes", c.Logger))

	c.Logger.Info("Setting up event handlers")
	routeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    impl.Enqueue,
		UpdateFunc: controller.PassNew(impl.Enqueue),
		DeleteFunc: impl.Enqueue,
	})

	serviceInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: controller.Filter(v1alpha1.SchemeGroupVersion.WithKind("Route")),
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc:    impl.EnqueueControllerOf,
			UpdateFunc: controller.PassNew(impl.EnqueueControllerOf),
			DeleteFunc: impl.EnqueueControllerOf,
		},
	})

	clusterIngressInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    impl.EnqueueLabelOfNamespaceScopedResource(serving.RouteNamespaceLabelKey, serving.RouteLabelKey),
		UpdateFunc: controller.PassNew(impl.EnqueueLabelOfNamespaceScopedResource(serving.RouteNamespaceLabelKey, serving.RouteLabelKey)),
		DeleteFunc: impl.EnqueueLabelOfNamespaceScopedResource(serving.RouteNamespaceLabelKey, serving.RouteLabelKey),
	})

	c.tracker = tracker.New(impl.EnqueueKey, opt.GetTrackerLease())
	gvk := v1alpha1.SchemeGroupVersion.WithKind("Configuration")
	configInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.EnsureTypeMeta(c.tracker.OnChanged, gvk),
		UpdateFunc: controller.PassNew(controller.EnsureTypeMeta(c.tracker.OnChanged, gvk)),
		DeleteFunc: controller.EnsureTypeMeta(c.tracker.OnChanged, gvk),
	})
	gvk = v1alpha1.SchemeGroupVersion.WithKind("Revision")
	revisionInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.EnsureTypeMeta(c.tracker.OnChanged, gvk),
		UpdateFunc: controller.PassNew(controller.EnsureTypeMeta(c.tracker.OnChanged, gvk)),
		DeleteFunc: controller.EnsureTypeMeta(c.tracker.OnChanged, gvk),
	})

	c.Logger.Info("Setting up ConfigMap receivers")
	configsToResync := []interface{}{
		&network.Config{},
		&config.Domain{},
	}
	resync := configmap.TypeFilter(configsToResync...)(func(string, interface{}) {
		impl.GlobalResync(routeInformer.Informer())
	})
	c.configStore = config.NewStore(c.Logger.Named("config-store"), resync)
	c.configStore.WatchConfigs(opt.ConfigMapWatcher)
	return impl
}

/////////////////////////////////////////
//  Event handlers
/////////////////////////////////////////

// Reconcile compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Route resource
// with the current status of the resource.
func (c *Reconciler) Reconcile(ctx context.Context, key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		c.Logger.Errorf("invalid resource key: %s", key)
		return nil
	}
	logger := logging.FromContext(ctx)

	ctx = c.configStore.ToContext(ctx)

	// Get the Route resource with this namespace/name.
	original, err := c.routeLister.Routes(namespace).Get(name)
	if apierrs.IsNotFound(err) {
		// The resource may no longer exist, in which case we stop processing.
		logger.Errorf("route %q in work queue no longer exists", key)
		return nil
	} else if err != nil {
		return err
	}
	// Don't modify the informers copy.
	route := original.DeepCopy()

	// Reconcile this copy of the route and then write back any status
	// updates regardless of whether the reconciliation errored out.
	err = c.reconcile(ctx, route)
	if equality.Semantic.DeepEqual(original.Status, route.Status) {
		// If we didn't change anything then don't call updateStatus.
		// This is important because the copy we loaded from the informer's
		// cache may be stale and we don't want to overwrite a prior update
		// to status with this stale state.
	} else if _, err := c.updateStatus(route); err != nil {
		logger.Warn("Failed to update route status", zap.Error(err))
		c.Recorder.Eventf(route, corev1.EventTypeWarning, "UpdateFailed",
			"Failed to update status for Route %q: %v", route.Name, err)
		return err
	}
	return err
}

func ingressClassForRoute(ctx context.Context, r *v1alpha1.Route) string {
	if ingressClass, _ := r.Annotations[networking.IngressClassAnnotationKey]; ingressClass != "" {
		return ingressClass
	}
	return config.FromContext(ctx).Network.DefaultClusterIngressClass
}

func (c *Reconciler) reconcile(ctx context.Context, r *v1alpha1.Route) error {
	logger := logging.FromContext(ctx)
	if r.GetDeletionTimestamp() != nil {
		// Check for a DeletionTimestamp.  If present, elide the normal reconcile logic.
		return c.reconcileDeletion(ctx, r)
	}

	// We may be reading a version of the object that was stored at an older version
	// and may not have had all of the assumed defaults specified.  This won't result
	// in this getting written back to the API Server, but lets downstream logic make
	// assumptions about defaulting.
	r.SetDefaults()

	r.Status.InitializeConditions()

	logger.Infof("Reconciling route: %v", r)
	// Configure traffic based on the RouteSpec.
	traffic, err := c.configureTraffic(ctx, r)
	if traffic == nil || err != nil {
		// Traffic targets aren't ready, no need to configure child resources.
		return err
	}

	logger.Info("Updating targeted revisions.")
	// In all cases we will add annotations to the referred targets.  This is so that when they become
	// routable we can know (through a listener) and attempt traffic configuration again.
	if err := c.reconcileTargetRevisions(ctx, traffic, r); err != nil {
		return err
	}

	// Update the information that makes us Addressable.
	r.Status.Domain = routeDomain(ctx, r)
	r.Status.DeprecatedDomainInternal = resourcenames.K8sServiceFullname(r)
	r.Status.Address = &duckv1alpha1.Addressable{
		Hostname: resourcenames.K8sServiceFullname(r),
	}

	// Add the finalizer before creating the ClusterIngress so that we can be sure it gets cleaned up.
	if err := c.ensureFinalizer(r); err != nil {
		return err
	}

	logger.Info("Creating ClusterIngress.")
	desired := resources.MakeClusterIngress(r, traffic, ingressClassForRoute(ctx, r))
	clusterIngress, err := c.reconcileClusterIngress(ctx, r, desired)
	if err != nil {
		return err
	}
	r.Status.PropagateClusterIngressStatus(clusterIngress.Status)

	logger.Info("Creating/Updating placeholder k8s services")
	if err := c.reconcilePlaceholderService(ctx, r, clusterIngress); err != nil {
		return err
	}

	r.Status.ObservedGeneration = r.Generation
	logger.Info("Route successfully synced")
	return nil
}

func (c *Reconciler) reconcileDeletion(ctx context.Context, r *v1alpha1.Route) error {
	logger := logging.FromContext(ctx)

	// If our Finalizer is first, delete the ClusterIngress for this Route
	// and remove the finalizer.
	if len(r.Finalizers) == 0 || r.Finalizers[0] != routeFinalizer {
		return nil
	}

	// Delete the ClusterIngress resources for this Route.
	logger.Info("Cleaning up ClusterIngress")
	if err := c.deleteClusterIngressesForRoute(r); err != nil {
		return err
	}

	// Update the Route to remove the Finalizer.
	logger.Info("Removing Finalizer")
	r.Finalizers = r.Finalizers[1:]
	_, err := c.ServingClientSet.ServingV1alpha1().Routes(r.Namespace).Update(r)
	return err
}

// configureTraffic attempts to configure traffic based on the RouteSpec.  If there are missing
// targets (e.g. Configurations without a Ready Revision, or Revision that isn't Ready or Inactive),
// no traffic will be configured.
//
// If traffic is configured we update the RouteStatus with AllTrafficAssigned = True.  Otherwise we
// mark AllTrafficAssigned = False, with a message referring to one of the missing target.
func (c *Reconciler) configureTraffic(ctx context.Context, r *v1alpha1.Route) (*traffic.Config, error) {
	logger := logging.FromContext(ctx)
	t, err := traffic.BuildTrafficConfiguration(c.configurationLister, c.revisionLister, r)

	if t != nil {
		// Tell our trackers to reconcile Route whenever the things referred to by our
		// Traffic stanza change.
		gvk := v1alpha1.SchemeGroupVersion.WithKind("Configuration")
		for _, configuration := range t.Configurations {
			if err := c.tracker.Track(objectRef(configuration, gvk), r); err != nil {
				return nil, err
			}
		}
		gvk = v1alpha1.SchemeGroupVersion.WithKind("Revision")
		for _, revision := range t.Revisions {
			if revision.Status.IsActivationRequired() {
				logger.Infof("Revision %s/%s is inactive", revision.Namespace, revision.Name)
			}
			if err := c.tracker.Track(objectRef(revision, gvk), r); err != nil {
				return nil, err
			}
		}
	}

	badTarget, isTargetError := err.(traffic.TargetError)
	if err != nil && !isTargetError {
		// An error that's not due to missing traffic target should
		// make us fail fast.
		r.Status.MarkUnknownTrafficError(err.Error())
		return nil, err
	}
	if badTarget != nil && isTargetError {
		badTarget.MarkBadTrafficTarget(&r.Status)

		// Traffic targets aren't ready, no need to configure Route.
		return nil, nil
	}

	logger.Info("All referred targets are routable, marking AllTrafficAssigned with traffic information.")
	r.Status.Traffic = t.GetRevisionTrafficTargets()
	r.Status.MarkTrafficAssigned()

	return t, nil
}

func (c *Reconciler) ensureFinalizer(route *v1alpha1.Route) error {
	finalizers := sets.NewString(route.Finalizers...)
	if finalizers.Has(routeFinalizer) {
		return nil
	}
	finalizers.Insert(routeFinalizer)

	mergePatch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"finalizers":      finalizers.List(),
			"resourceVersion": route.ResourceVersion,
		},
	}

	patch, err := json.Marshal(mergePatch)
	if err != nil {
		return err
	}

	_, err = c.ServingClientSet.ServingV1alpha1().Routes(route.Namespace).Patch(route.Name, types.MergePatchType, patch)
	return err
}

/////////////////////////////////////////
// Misc helpers.
/////////////////////////////////////////

type accessor interface {
	GroupVersionKind() schema.GroupVersionKind
	GetNamespace() string
	GetName() string
}

func objectRef(a accessor, gvk schema.GroupVersionKind) corev1.ObjectReference {
	// We can't always rely on the TypeMeta being populated.
	// See: https://github.com/knative/serving/issues/2372
	// Also: https://github.com/kubernetes/apiextensions-apiserver/issues/29
	// gvk := a.GroupVersionKind()
	apiVersion, kind := gvk.ToAPIVersionAndKind()
	return corev1.ObjectReference{
		APIVersion: apiVersion,
		Kind:       kind,
		Namespace:  a.GetNamespace(),
		Name:       a.GetName(),
	}
}

func routeDomain(ctx context.Context, route *v1alpha1.Route) string {
	domainConfig := config.FromContext(ctx).Domain
	domain := domainConfig.LookupDomainForLabels(route.ObjectMeta.Labels)
	return fmt.Sprintf("%s.%s.%s", route.Name, route.Namespace, domain)
}
