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

package build

import (
	"context"
	"fmt"
	"sync"
	"time"

	v1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	clientset "github.com/knative/build/pkg/client/clientset/versioned"
	buildscheme "github.com/knative/build/pkg/client/clientset/versioned/scheme"
	informers "github.com/knative/build/pkg/client/informers/externalversions/build/v1alpha1"
	listers "github.com/knative/build/pkg/client/listers/build/v1alpha1"
	"github.com/knative/build/pkg/reconciler"
	"github.com/knative/build/pkg/reconciler/build/resources"
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	"github.com/knative/pkg/controller"
	"github.com/knative/pkg/logging"
	"github.com/knative/pkg/logging/logkey"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

const (
	controllerAgentName = "build-controller"
	defaultTimeout      = 10 * time.Minute
)

// Reconciler is the controller.Reconciler implementation for Builds resources
type Reconciler struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// buildclientset is a clientset for our own API group
	buildclientset clientset.Interface
	timeoutHandler *TimeoutSet

	buildsLister                listers.BuildLister
	buildTemplatesLister        listers.BuildTemplateLister
	clusterBuildTemplatesLister listers.ClusterBuildTemplateLister
	podsLister                  corelisters.PodLister

	// Sugared logger is easier to use but is not as performant as the
	// raw logger. In performance critical paths, call logger.Desugar()
	// and use the returned raw logger instead. In addition to the
	// performance benefits, raw logger also preserves type-safety at
	// the expense of slightly greater verbosity.
	Logger *zap.SugaredLogger
}

// Check that we implement the controller.Reconciler interface.
var _ controller.Reconciler = (*Reconciler)(nil)
var statusMap = sync.Map{}

func init() {
	// Add build-controller types to the default Kubernetes Scheme so Events can be
	// logged for build-controller types.
	buildscheme.AddToScheme(scheme.Scheme)
}

// NewController returns a new build template controller
func NewController(
	logger *zap.SugaredLogger,
	kubeclientset kubernetes.Interface,
	podInformer coreinformers.PodInformer,
	buildclientset clientset.Interface,
	buildInformer informers.BuildInformer,
	buildTemplateInformer informers.BuildTemplateInformer,
	clusterBuildTemplateInformer informers.ClusterBuildTemplateInformer,
	timeoutHandler *TimeoutSet,
) *controller.Impl {

	// Enrich the logs with controller name
	logger = logger.Named(controllerAgentName).With(zap.String(logkey.ControllerType, controllerAgentName))

	r := &Reconciler{
		kubeclientset:               kubeclientset,
		buildclientset:              buildclientset,
		buildsLister:                buildInformer.Lister(),
		buildTemplatesLister:        buildTemplateInformer.Lister(),
		clusterBuildTemplatesLister: clusterBuildTemplateInformer.Lister(),
		podsLister:                  podInformer.Lister(),
		Logger:                      logger,
		timeoutHandler:              timeoutHandler,
	}
	impl := controller.NewImpl(r, logger, "Builds",
		reconciler.MustNewStatsReporter("Builds", r.Logger))

	logger.Info("Setting up event handlers")
	// Set up an event handler for when Build resources change
	buildInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    impl.Enqueue,
		UpdateFunc: controller.PassNew(impl.Enqueue),
	})

	// Set up a Pod informer, so that Pod updates trigger Build
	// reconciliations.
	podInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: controller.Filter(v1alpha1.SchemeGroupVersion.WithKind("Build")),
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc:    impl.EnqueueControllerOf,
			UpdateFunc: controller.PassNew(impl.EnqueueControllerOf),
		},
	})

	return impl
}

// Reconcile implements controller.Reconciler
func (c *Reconciler) Reconcile(ctx context.Context, key string) error {
	logger := logging.FromContext(ctx)

	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		logger.Errorf("invalid resource key: %s", key)
		return nil
	}

	// Get the Build resource with this namespace/name
	build, err := c.buildsLister.Builds(namespace).Get(name)
	if errors.IsNotFound(err) {
		// The Build resource may no longer exist, in which case we stop processing.
		logger.Errorf("build %q in work queue no longer exists", key)
		return nil
	} else if err != nil {
		return err
	}

	// Don't mutate the informer's copy of our object.
	build = build.DeepCopy()

	// If the build's done, then ignore it.
	if isDone(&build.Status) {
		return nil
	}

	// If the build's status is cancelled, kill resources and update status
	if isCancelled(build.Spec) {
		return c.cancelBuild(build, logger)
	}

	// If the build hasn't started yet, validate it and create a Pod for it
	// and record that pod's name in the build status.
	var p *corev1.Pod
	if build.Status.Cluster == nil || build.Status.Cluster.PodName == "" {
		// Add a unique suffix to avoid confusion when a build
		// is deleted and re-created with the same name.
		// We don't use GenerateName here because k8s fakes don't support it.
		podName, err := resources.GetUniquePodName(build.Name)
		if err != nil {
			return err
		}
		// update with a dummy status first to avoid race condition of another event while the pod is being created
		build.Status = v1alpha1.BuildStatus{
			Builder: v1alpha1.ClusterBuildProvider,
			Cluster: &v1alpha1.ClusterSpec{
				Namespace: build.Namespace,
				PodName:   podName,
			},
			StartTime: &metav1.Time{
				Time: time.Now(),
			},
		}
		if err := c.updateStatus(build); err != nil {
			return err
		}

		if err = c.validateBuild(build); err != nil {
			logger.Errorf("Failed to validate build: %v", err)
			build.Status = v1alpha1.BuildStatus{
				Cluster: &v1alpha1.ClusterSpec{
					PodName: "",
				},
			}
			build.Status.SetCondition(&duckv1alpha1.Condition{
				Type:    v1alpha1.BuildSucceeded,
				Status:  corev1.ConditionFalse,
				Reason:  "BuildValidationFailed",
				Message: err.Error(),
			})
			if err := c.updateStatus(build); err != nil {
				return err
			}
			return err
		}

		p, err = c.startPodForBuild(build)
		if err != nil {
			build.Status.SetCondition(&duckv1alpha1.Condition{
				Type:    v1alpha1.BuildSucceeded,
				Status:  corev1.ConditionFalse,
				Reason:  "BuildExecuteFailed",
				Message: err.Error(),
			})
			if err := c.updateStatus(build); err != nil {
				return err
			}
			return err
		}
		// Start goroutine that waits for either build timeout or build finish
		go c.timeoutHandler.wait(build)
	} else {
		// If the build is ongoing, update its status based on its pod, and
		// check if it's timed out.
		p, err = c.podsLister.Pods(build.Namespace).Get(build.Status.Cluster.PodName)
		if err != nil {
			// TODO: What if the pod is deleted out from under us?
			return err
		}
	}

	// Update the build's status based on the pod's status.
	statusLock(build)
	build.Status = resources.BuildStatusFromPod(p, build.Spec)
	statusUnlock(build)
	if isDone(&build.Status) {
		// release goroutine that waits for build timeout
		c.timeoutHandler.release(build)
		// and remove key from status map
		defer statusMap.Delete(key)
	}

	return c.updateStatus(build)
}

func (c *Reconciler) updateStatus(u *v1alpha1.Build) error {
	statusLock(u)
	defer statusUnlock(u)
	newb, err := c.buildclientset.BuildV1alpha1().Builds(u.Namespace).Get(u.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	cond := newb.Status.GetCondition(v1alpha1.BuildSucceeded)
	if cond != nil && cond.Status == corev1.ConditionFalse {
		return fmt.Errorf("can't update status of failed build %q", newb.Name)
	}

	newb.Status = u.Status

	_, err = c.buildclientset.BuildV1alpha1().Builds(u.Namespace).UpdateStatus(newb)
	return err
}

// startPodForBuild starts a new Pod to execute the build.
//
// This applies any build template that's specified, and creates the pod.
func (c *Reconciler) startPodForBuild(build *v1alpha1.Build) (*corev1.Pod, error) {
	namespace := build.Namespace
	var tmpl v1alpha1.BuildTemplateInterface
	var err error
	if build.Spec.Template != nil {
		if build.Spec.Template.Kind == v1alpha1.ClusterBuildTemplateKind {
			tmpl, err = c.clusterBuildTemplatesLister.Get(build.Spec.Template.Name)
			if err != nil {
				// The ClusterBuildTemplate resource may not exist.
				if errors.IsNotFound(err) {
					runtime.HandleError(fmt.Errorf("cluster build template %q does not exist", build.Spec.Template.Name))
				}
				return nil, err
			}
		} else {
			tmpl, err = c.buildTemplatesLister.BuildTemplates(namespace).Get(build.Spec.Template.Name)
			if err != nil {
				// The BuildTemplate resource may not exist.
				if errors.IsNotFound(err) {
					runtime.HandleError(fmt.Errorf("build template %q in namespace %q does not exist", build.Spec.Template.Name, namespace))
				}
				return nil, err
			}
		}
	}
	build, err = ApplyTemplate(build, tmpl)
	if err != nil {
		return nil, err
	}

	p, err := resources.MakePod(build, c.kubeclientset)
	if err != nil {
		return nil, err
	}
	c.Logger.Infof("Creating pod %q in namespace %q for build %q", p.Name, p.Namespace, build.Name)
	return c.kubeclientset.CoreV1().Pods(p.Namespace).Create(p)
}

// isCancelled returns true if the build's spec indicates the build is cancelled.
func isCancelled(buildSpec v1alpha1.BuildSpec) bool {
	return buildSpec.Status == v1alpha1.BuildSpecStatusCancelled
}

func (c *Reconciler) cancelBuild(build *v1alpha1.Build, logger *zap.SugaredLogger) error {
	logger.Warnf("Build has been cancelled: %v", build.Name)
	build.Status.SetCondition(&duckv1alpha1.Condition{
		Type:    v1alpha1.BuildSucceeded,
		Status:  corev1.ConditionFalse,
		Reason:  "BuildCancelled",
		Message: fmt.Sprintf("Build %q was cancelled", build.Name),
	})
	if err := c.updateStatus(build); err != nil {
		return err
	}
	if build.Status.Cluster == nil {
		logger.Warnf("build %q has no pod running yet", build.Name)
		return nil
	}
	if err := c.kubeclientset.CoreV1().Pods(build.Namespace).Delete(build.Status.Cluster.PodName, &metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

// isDone returns true if the build's status indicates the build is done.
func isDone(status *v1alpha1.BuildStatus) bool {
	cond := status.GetCondition(v1alpha1.BuildSucceeded)
	return cond != nil && cond.Status != corev1.ConditionUnknown
}

func (c *Reconciler) checkTimeout(build *v1alpha1.Build) error {
	// If build has not started timeout, startTime should be zero.
	if build.Status.StartTime.IsZero() {
		return nil
	}

	// Use default timeout to 10 minute if build timeout is not set.
	timeout := defaultTimeout
	if build.Spec.Timeout != nil {
		timeout = build.Spec.Timeout.Duration
	}
	runtime := time.Since(build.Status.StartTime.Time)
	if runtime > timeout {
		c.Logger.Infof("Build %q is timeout (runtime %s over %s), deleting pod", build.Name, runtime, timeout)
		if err := c.kubeclientset.CoreV1().Pods(build.Namespace).Delete(build.Status.Cluster.PodName, &metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
			c.Logger.Errorf("Failed to terminate pod: %v", err)
			return err
		}

		timeoutMsg := fmt.Sprintf("Build %q failed to finish within %q", build.Name, timeout.String())
		build.Status.SetCondition(&duckv1alpha1.Condition{
			Type:    v1alpha1.BuildSucceeded,
			Status:  corev1.ConditionFalse,
			Reason:  "BuildTimeout",
			Message: timeoutMsg,
		})
		// update build completed time
		build.Status.CompletionTime = &metav1.Time{time.Now()}
	}
	return nil
}

func statusLock(build *v1alpha1.Build) {
	key := fmt.Sprintf("%s/%s", build.Namespace, build.Name)
	m, _ := statusMap.LoadOrStore(key, &sync.Mutex{})
	mut := m.(*sync.Mutex)
	mut.Lock()
}

func statusUnlock(build *v1alpha1.Build) {
	key := fmt.Sprintf("%s/%s", build.Namespace, build.Name)
	m, ok := statusMap.Load(key)
	if !ok {
		return
	}
	mut := m.(*sync.Mutex)
	mut.Unlock()
}
