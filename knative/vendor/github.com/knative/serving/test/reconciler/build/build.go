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

package build

import (
	"context"
	"fmt"
	"reflect"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"

	"github.com/knative/pkg/controller"
	"github.com/knative/pkg/logging"
	"github.com/knative/pkg/logging/logkey"

	"github.com/knative/serving/pkg/reconciler"
	testing "github.com/knative/serving/test/apis/testing/v1alpha1"
	clientset "github.com/knative/serving/test/client/clientset/versioned"
	buildscheme "github.com/knative/serving/test/client/clientset/versioned/scheme"
	informers "github.com/knative/serving/test/client/informers/externalversions/testing/v1alpha1"
	listers "github.com/knative/serving/test/client/listers/testing/v1alpha1"
)

const controllerAgentName = "testbuild-controller"

// Reconciler is the controller implementation for Build resources
type Reconciler struct {
	buildclientset clientset.Interface
	buildsLister   listers.BuildLister
}

// Check that we implement the controller.Reconciler interface.
var _ controller.Reconciler = (*Reconciler)(nil)

func init() {
	// Add build-controller types to the default Kubernetes Scheme so Events can be
	// logged for build-controller types.
	buildscheme.AddToScheme(scheme.Scheme)
}

// NewController returns a new build controller
func NewController(
	logger *zap.SugaredLogger,
	buildclientset clientset.Interface,
	buildInformer informers.BuildInformer,
) *controller.Impl {

	// Enrich the logs with controller name
	logger = logger.Named(controllerAgentName).
		With(zap.String(logkey.ControllerType, controllerAgentName))

	r := &Reconciler{
		buildclientset: buildclientset,
		buildsLister:   buildInformer.Lister(),
	}
	impl := controller.NewImpl(r, logger, "Builds", reconciler.MustNewStatsReporter("Builds", logger))

	logger.Info("Setting up event handlers")
	// Set up an event handler for when Build resources change
	buildInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    impl.Enqueue,
		UpdateFunc: controller.PassNew(impl.Enqueue),
	})

	return impl
}

// Reconcile implements controller.Reconciler
func (c *Reconciler) Reconcile(ctx context.Context, key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}
	logger := logging.FromContext(ctx)

	// Get the Build resource with this namespace/name
	original, err := c.buildsLister.Builds(namespace).Get(name)
	if errors.IsNotFound(err) {
		// The Build resource may no longer exist, in which case we stop processing.
		logger.Errorf("build %q in work queue no longer exists", key)
		return nil
	} else if err != nil {
		return err
	}
	// Don't modify the informer's copy.
	build := original.DeepCopy()

	// Reconcile this copy of the build and then write back any status
	// updates regardless of whether the reconciliation errored out.
	err = c.reconcile(ctx, build)
	if equality.Semantic.DeepEqual(original.Status, build.Status) {
		// If we didn't change anything then don't call updateStatus.
		// This is important because the copy we loaded from the informer's
		// cache may be stale and we don't want to overwrite a prior update
		// to status with this stale state.
	} else {
		// logger.Infof("Updating Status (-old, +new): %v", cmp.Diff(original, build))
		if _, err := c.updateStatus(build); err != nil {
			logger.Warn("Failed to update build status", zap.Error(err))
			return err
		}
	}
	return err
}

func (c *Reconciler) reconcile(ctx context.Context, build *testing.Build) error {
	// logger := logging.FromContext(ctx)

	build.Status.InitializeConditions()

	if build.Spec.Failure != nil {
		build.Status.MarkFailure(build.Spec.Failure)
	} else {
		build.Status.MarkDone()
	}

	return nil
}

func (c *Reconciler) updateStatus(build *testing.Build) (*testing.Build, error) {
	newBuild, err := c.buildsLister.Builds(build.Namespace).Get(build.Name)
	if err != nil {
		return nil, err
	}
	// Check if there is anything to update.
	if !reflect.DeepEqual(newBuild.Status, build.Status) {
		newBuild.Status = build.Status

		// TODO: for CRD there's no updatestatus, so use normal update
		return c.buildclientset.TestingV1alpha1().Builds(build.Namespace).Update(newBuild)
		//	return prClient.UpdateStatus(newBuild)
	}
	return build, nil
}
