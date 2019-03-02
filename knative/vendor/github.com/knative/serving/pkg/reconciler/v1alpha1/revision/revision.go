/*
Copyright 2018 The Knative Authors.

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

package revision

import (
	"context"
	"net/http"
	"reflect"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	appsv1informers "k8s.io/client-go/informers/apps/v1"
	corev1informers "k8s.io/client-go/informers/core/v1"
	appsv1listers "k8s.io/client-go/listers/apps/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	cachinginformers "github.com/knative/caching/pkg/client/informers/externalversions/caching/v1alpha1"
	cachinglisters "github.com/knative/caching/pkg/client/listers/caching/v1alpha1"
	"github.com/knative/pkg/apis/duck"
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	"github.com/knative/pkg/configmap"
	"github.com/knative/pkg/controller"
	commonlogging "github.com/knative/pkg/logging"
	"github.com/knative/pkg/tracker"
	"github.com/knative/serving/pkg/apis/serving"
	"github.com/knative/serving/pkg/apis/serving/v1alpha1"
	painformers "github.com/knative/serving/pkg/client/informers/externalversions/autoscaling/v1alpha1"
	servinginformers "github.com/knative/serving/pkg/client/informers/externalversions/serving/v1alpha1"
	kpalisters "github.com/knative/serving/pkg/client/listers/autoscaling/v1alpha1"
	listers "github.com/knative/serving/pkg/client/listers/serving/v1alpha1"
	"github.com/knative/serving/pkg/network"
	"github.com/knative/serving/pkg/reconciler"
	"github.com/knative/serving/pkg/reconciler/v1alpha1/revision/config"

	"go.uber.org/zap"
)

const (
	controllerAgentName = "revision-controller"
)

var (
	foregroundDeletion = metav1.DeletePropagationForeground
	fgDeleteOptions    = &metav1.DeleteOptions{
		PropagationPolicy: &foregroundDeletion,
	}
)

type Changed bool

const (
	WasChanged Changed = true
	Unchanged  Changed = false
)

type resolver interface {
	Resolve(string, k8schain.Options, sets.String) (string, error)
}

type configStore interface {
	ToContext(ctx context.Context) context.Context
	WatchConfigs(w configmap.Watcher)
	Load() *config.Config
}

// Reconciler implements controller.Reconciler for Revision resources.
type Reconciler struct {
	*reconciler.Base

	// lister indexes properties about Revision
	revisionLister      listers.RevisionLister
	podAutoscalerLister kpalisters.PodAutoscalerLister
	imageLister         cachinglisters.ImageLister
	deploymentLister    appsv1listers.DeploymentLister
	serviceLister       corev1listers.ServiceLister
	endpointsLister     corev1listers.EndpointsLister
	configMapLister     corev1listers.ConfigMapLister

	buildInformerFactory duck.InformerFactory

	tracker     tracker.Interface
	resolver    resolver
	configStore configStore
}

// Check that our Reconciler implements controller.Reconciler
var _ controller.Reconciler = (*Reconciler)(nil)

// NewController initializes the controller and is called by the generated code
// Registers eventhandlers to enqueue events
// config - client configuration for talking to the apiserver
// si - informer factory shared across all controllers for listening to events and indexing resource properties
// queue - message queue for handling new events.  unique to this controller.
func NewController(
	opt reconciler.Options,
	revisionInformer servinginformers.RevisionInformer,
	podAutoscalerInformer painformers.PodAutoscalerInformer,
	imageInformer cachinginformers.ImageInformer,
	deploymentInformer appsv1informers.DeploymentInformer,
	serviceInformer corev1informers.ServiceInformer,
	endpointsInformer corev1informers.EndpointsInformer,
	configMapInformer corev1informers.ConfigMapInformer,
	buildInformerFactory duck.InformerFactory,
) *controller.Impl {
	transport := http.DefaultTransport
	if rt, err := newResolverTransport(k8sCertPath); err != nil {
		opt.Logger.Errorf("Failed to create resolver transport: %v", err)
	} else {
		transport = rt
	}

	c := &Reconciler{
		Base:                reconciler.NewBase(opt, controllerAgentName),
		revisionLister:      revisionInformer.Lister(),
		podAutoscalerLister: podAutoscalerInformer.Lister(),
		imageLister:         imageInformer.Lister(),
		deploymentLister:    deploymentInformer.Lister(),
		serviceLister:       serviceInformer.Lister(),
		endpointsLister:     endpointsInformer.Lister(),
		configMapLister:     configMapInformer.Lister(),
		resolver: &digestResolver{
			client:    opt.KubeClientSet,
			transport: transport,
		},
	}
	impl := controller.NewImpl(c, c.Logger, "Revisions", reconciler.MustNewStatsReporter("Revisions", c.Logger))

	// Set up an event handler for when the resource types of interest change
	c.Logger.Info("Setting up event handlers")
	revisionInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    impl.Enqueue,
		UpdateFunc: controller.PassNew(impl.Enqueue),
		DeleteFunc: impl.Enqueue,
	})

	endpointsInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    impl.EnqueueLabelOfNamespaceScopedResource("", serving.RevisionLabelKey),
		UpdateFunc: controller.PassNew(impl.EnqueueLabelOfNamespaceScopedResource("", serving.RevisionLabelKey)),
		DeleteFunc: impl.EnqueueLabelOfNamespaceScopedResource("", serving.RevisionLabelKey),
	})

	deploymentInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: controller.Filter(v1alpha1.SchemeGroupVersion.WithKind("Revision")),
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc:    impl.EnqueueControllerOf,
			UpdateFunc: controller.PassNew(impl.EnqueueControllerOf),
			DeleteFunc: impl.EnqueueControllerOf,
		},
	})

	podAutoscalerInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: controller.Filter(v1alpha1.SchemeGroupVersion.WithKind("Revision")),
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc:    impl.EnqueueControllerOf,
			UpdateFunc: controller.PassNew(impl.EnqueueControllerOf),
			DeleteFunc: impl.EnqueueControllerOf,
		},
	})

	c.tracker = tracker.New(impl.EnqueueKey, opt.GetTrackerLease())

	// We don't watch for changes to Image because we don't incorporate any of its
	// properties into our own status and should work completely in the absence of
	// a functioning Image controller.

	configMapInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: controller.Filter(v1alpha1.SchemeGroupVersion.WithKind("Revision")),
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc:    impl.EnqueueControllerOf,
			UpdateFunc: controller.PassNew(impl.EnqueueControllerOf),
			DeleteFunc: impl.EnqueueControllerOf,
		},
	})

	c.buildInformerFactory = newDuckInformerFactory(c.tracker, buildInformerFactory)

	configsToResync := []interface{}{
		&network.Config{},
		&config.Observability{},
		&config.Controller{},
	}

	resync := configmap.TypeFilter(configsToResync...)(func(string, interface{}) {
		// Triggers syncs on all revisions when configuration
		// changes
		impl.GlobalResync(revisionInformer.Informer())
	})

	c.configStore = config.NewStore(c.Logger.Named("config-store"), resync)
	c.configStore.WatchConfigs(opt.ConfigMapWatcher)

	return impl
}

func KResourceTypedInformerFactory(opt reconciler.Options) duck.InformerFactory {
	return &duck.TypedInformerFactory{
		Client:       opt.DynamicClientSet,
		Type:         &duckv1alpha1.KResource{},
		ResyncPeriod: opt.ResyncPeriod,
		StopChannel:  opt.StopChannel,
	}
}

func newDuckInformerFactory(t tracker.Interface, delegate duck.InformerFactory) duck.InformerFactory {
	return &duck.CachedInformerFactory{
		Delegate: &duck.EnqueueInformerFactory{
			Delegate: delegate,
			EventHandler: cache.ResourceEventHandlerFuncs{
				AddFunc:    t.OnChanged,
				UpdateFunc: controller.PassNew(t.OnChanged),
				DeleteFunc: t.OnChanged,
			},
		},
	}
}

// Reconcile compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Revision resource
// with the current status of the resource.
func (c *Reconciler) Reconcile(ctx context.Context, key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		c.Logger.Errorf("invalid resource key: %s", key)
		return nil
	}
	logger := commonlogging.FromContext(ctx)
	logger.Info("Running reconcile Revision")

	ctx = c.configStore.ToContext(ctx)

	// Get the Revision resource with this namespace/name
	original, err := c.revisionLister.Revisions(namespace).Get(name)
	// The resource may no longer exist, in which case we stop processing.
	if apierrs.IsNotFound(err) {
		logger.Errorf("revision %q in work queue no longer exists", key)
		return nil
	} else if err != nil {
		return err
	}

	// Don't modify the informer's copy.
	rev, err := c.migrateConfigurationMetadata(original.DeepCopy())

	if err != nil {
		logger.Warnw("Failed to migrate revision labels", zap.Error(err))
		c.Recorder.Eventf(rev, corev1.EventTypeWarning, "UpdateFailed",
			"Failed to migrate revision %q labels: %v", rev.Name, err)
		return err
	}

	// Reconcile this copy of the revision and then write back any status
	// updates regardless of whether the reconciliation errored out.
	err = c.reconcile(ctx, rev)
	if equality.Semantic.DeepEqual(original.Status, rev.Status) {
		// If we didn't change anything then don't call updateStatus.
		// This is important because the copy we loaded from the informer's
		// cache may be stale and we don't want to overwrite a prior update
		// to status with this stale state.
	} else if _, err := c.updateStatus(rev); err != nil {
		logger.Warn("Failed to update revision status", zap.Error(err))
		c.Recorder.Eventf(rev, corev1.EventTypeWarning, "UpdateFailed",
			"Failed to update status for Revision %q: %v", rev.Name, err)
		return err
	}
	return err
}

func (c *Reconciler) reconcileBuild(ctx context.Context, rev *v1alpha1.Revision) error {
	buildRef := rev.BuildRef()
	if buildRef == nil {
		rev.Status.PropagateBuildStatus(duckv1alpha1.KResourceStatus{
			Conditions: []duckv1alpha1.Condition{{
				Type:   duckv1alpha1.ConditionSucceeded,
				Status: corev1.ConditionTrue,
				Reason: "NoBuild",
			}},
		})
		return nil
	}

	logger := commonlogging.FromContext(ctx)

	if err := c.tracker.Track(*buildRef, rev); err != nil {
		logger.Errorf("Error tracking build '%+v' for Revision %q: %+v", buildRef, rev.Name, err)
		return err
	}

	gvr, _ := meta.UnsafeGuessKindToResource(buildRef.GroupVersionKind())
	_, lister, err := c.buildInformerFactory.Get(gvr)
	if err != nil {
		logger.Errorf("Error getting a lister for a builds resource '%+v': %+v", gvr, err)
		return err
	}

	buildObj, err := lister.ByNamespace(rev.Namespace).Get(buildRef.Name)
	if err != nil {
		logger.Errorf("Error fetching Build %q for Revision %q: %v", buildRef.Name, rev.Name, err)
		return err
	}
	build := buildObj.(*duckv1alpha1.KResource)

	before := rev.Status.GetCondition(v1alpha1.RevisionConditionBuildSucceeded)
	rev.Status.PropagateBuildStatus(build.Status)
	after := rev.Status.GetCondition(v1alpha1.RevisionConditionBuildSucceeded)
	if before.Status != after.Status {
		// Create events when the Build result is in.
		if after.Status == corev1.ConditionTrue {
			c.Recorder.Event(rev, corev1.EventTypeNormal, "BuildSucceeded", after.Message)
		} else if after.Status == corev1.ConditionFalse {
			c.Recorder.Event(rev, corev1.EventTypeWarning, "BuildFailed", after.Message)
		}
	}

	return nil
}

func (c *Reconciler) reconcileDigest(ctx context.Context, rev *v1alpha1.Revision) error {
	// The image digest has already been resolved.
	if rev.Status.ImageDigest != "" {
		return nil
	}

	cfgs := config.FromContext(ctx)
	opt := k8schain.Options{
		Namespace:          rev.Namespace,
		ServiceAccountName: rev.Spec.ServiceAccountName,
		// ImagePullSecrets: Not possible via RevisionSpec, since we
		// don't expose such a field.
	}
	digest, err := c.resolver.Resolve(rev.Spec.Container.Image, opt, cfgs.Controller.RegistriesSkippingTagResolving)
	if err != nil {
		rev.Status.MarkContainerMissing(v1alpha1.RevisionContainerMissingMessage(rev.Spec.Container.Image, err.Error()))
		return err
	}

	rev.Status.ImageDigest = digest

	return nil
}

func (c *Reconciler) reconcile(ctx context.Context, rev *v1alpha1.Revision) error {
	logger := commonlogging.FromContext(ctx)
	if rev.GetDeletionTimestamp() != nil {
		return nil
	}

	// We may be reading a version of the object that was stored at an older version
	// and may not have had all of the assumed defaults specified.  This won't result
	// in this getting written back to the API Server, but lets downstream logic make
	// assumptions about defaulting.
	rev.SetDefaults()

	rev.Status.InitializeConditions()
	c.updateRevisionLoggingURL(ctx, rev)

	readyBeforeReconcile := rev.Status.IsReady()

	if err := c.reconcileBuild(ctx, rev); err != nil {
		return err
	}

	bc := rev.Status.GetCondition(v1alpha1.RevisionConditionBuildSucceeded)
	if bc == nil || bc.Status == corev1.ConditionTrue {
		// There is no build, or the build completed successfully.

		phases := []struct {
			name string
			f    func(context.Context, *v1alpha1.Revision) error
		}{{
			name: "image digest",
			f:    c.reconcileDigest,
		}, {
			name: "user deployment",
			f:    c.reconcileDeployment,
		}, {
			name: "user k8s service",
			f:    c.reconcileService,
		}, {
			// Ensures our namespace has the configuration for the fluentd sidecar.
			name: "fluentd configmap",
			f:    c.reconcileFluentdConfigMap,
		}, {
			name: "KPA",
			f:    c.reconcileKPA,
		}}

		for _, phase := range phases {
			if err := phase.f(ctx, rev); err != nil {
				logger.Errorf("Failed to reconcile %s: %v", phase.name, zap.Error(err))
				return err
			}
		}
	}

	readyAfterReconcile := rev.Status.IsReady()
	if !readyBeforeReconcile && readyAfterReconcile {
		c.Recorder.Eventf(rev, corev1.EventTypeNormal, "RevisionReady",
			"Revision becomes ready upon all resources being ready")
	}

	rev.Status.ObservedGeneration = rev.Generation
	return nil
}

func (c *Reconciler) updateRevisionLoggingURL(
	ctx context.Context,
	rev *v1alpha1.Revision,
) {

	config := config.FromContext(ctx)
	if config.Observability.LoggingURLTemplate == "" {
		return
	}

	uid := string(rev.UID)

	rev.Status.LogURL = strings.Replace(
		config.Observability.LoggingURLTemplate,
		"${REVISION_UID}", uid, -1)
}

func (c *Reconciler) updateStatus(desired *v1alpha1.Revision) (*v1alpha1.Revision, error) {
	rev, err := c.revisionLister.Revisions(desired.Namespace).Get(desired.Name)
	if err != nil {
		return nil, err
	}
	// If there's nothing to update, just return.
	if reflect.DeepEqual(rev.Status, desired.Status) {
		return rev, nil
	}
	// Don't modify the informers copy
	existing := rev.DeepCopy()
	existing.Status = desired.Status
	return c.ServingClientSet.ServingV1alpha1().Revisions(desired.Namespace).UpdateStatus(existing)
}

// TODO(643) Change this logic in 0.5 to only drop the deprecated label
//           Delete this logic in 0.6
func (c *Reconciler) migrateConfigurationMetadata(rev *v1alpha1.Revision) (*v1alpha1.Revision, error) {
	stale := false

	// The /configurationGeneration label key used to be an annotation key
	// This is not the case anymore so if the revision has that annotation
	// we delete it since it used to point to a configuration's spec.generation
	if rev.Annotations[serving.ConfigurationGenerationLabelKey] != "" {
		delete(rev.Annotations, serving.ConfigurationGenerationLabelKey)
		stale = true
	}

	legacyKey := serving.DeprecatedConfigurationMetadataGenerationLabelKey
	targetKey := serving.ConfigurationGenerationLabelKey

	legacyValue, hasLegacy := rev.Labels[legacyKey]
	targetValue, hasTarget := rev.Labels[targetKey]

	// If the two keys are different then set /configurationGeneration
	// to be the value of the label /configurationMetadataGeneration
	if hasLegacy && targetValue != legacyValue {
		stale = true
		rev.Labels[targetKey] = legacyValue
	}

	if hasTarget && !hasLegacy {
		// This occurs if the revision was created with 0.2 and
		// received a /configurationGeneration label but never
		// received a /configurationMetadataGeneration label since
		// it was not the latest created revision
		//
		// We drop this label since it's value was set according
		// to a configuration's spec.generation
		stale = true
		delete(rev.Labels, serving.ConfigurationGenerationLabelKey)
	}

	if !stale {
		return rev, nil
	}

	return c.ServingClientSet.ServingV1alpha1().Revisions(rev.Namespace).Update(rev)
}
