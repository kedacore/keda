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

package labeler

import (
	"context"

	"github.com/knative/pkg/controller"
	"github.com/knative/pkg/logging"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"

	servinginformers "github.com/knative/serving/pkg/client/informers/externalversions/serving/v1alpha1"
	listers "github.com/knative/serving/pkg/client/listers/serving/v1alpha1"
	"github.com/knative/serving/pkg/reconciler"
)

const (
	controllerAgentName = "labeler-controller"
)

// Reconciler implements controller.Reconciler for Route resources.
type Reconciler struct {
	*reconciler.Base

	// Listers index properties about resources
	routeLister         listers.RouteLister
	configurationLister listers.ConfigurationLister
	revisionLister      listers.RevisionLister
}

// Check that our Reconciler implements controller.Reconciler
var _ controller.Reconciler = (*Reconciler)(nil)

// NewRouteToConfigurationController wraps a new instance of the labeler that labels
// Configurations with Routes in a controller.
func NewRouteToConfigurationController(
	opt reconciler.Options,
	routeInformer servinginformers.RouteInformer,
	configInformer servinginformers.ConfigurationInformer,
	revisionInformer servinginformers.RevisionInformer,
) *controller.Impl {

	c := &Reconciler{
		Base:                reconciler.NewBase(opt, controllerAgentName),
		routeLister:         routeInformer.Lister(),
		configurationLister: configInformer.Lister(),
		revisionLister:      revisionInformer.Lister(),
	}
	impl := controller.NewImpl(c, c.Logger, "Labels", reconciler.MustNewStatsReporter("Labels", c.Logger))

	c.Logger.Info("Setting up event handlers")
	routeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    impl.Enqueue,
		UpdateFunc: controller.PassNew(impl.Enqueue),
		DeleteFunc: impl.Enqueue,
	})

	return impl
}

// Reconcile compares the actual state with the desired, and attempts to
// converge the two. In this case, it attempts to label all Configurations
// with the Routes that direct traffic to their Revisions.
func (c *Reconciler) Reconcile(ctx context.Context, key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		c.Logger.Errorf("invalid resource key: %s", key)
		return nil
	}
	logger := logging.FromContext(ctx)

	// Get the Route resource with this namespace/name
	route, err := c.routeLister.Routes(namespace).Get(name)
	if apierrs.IsNotFound(err) {
		logger.Infof("Clearing labels for deleted Route: %q", key)
		return c.deleteLabelForOutsideOfGivenConfigurations(
			ctx, namespace, name, sets.NewString(),
		)
	} else if err != nil {
		return err
	}

	logger.Infof("Time to sync the labels: %#v", route)
	return c.syncLabels(ctx, route)
}
