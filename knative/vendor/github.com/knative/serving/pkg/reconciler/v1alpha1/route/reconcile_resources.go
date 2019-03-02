/*
Copyright 2018 The Knative Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package route

import (
	"context"
	"fmt"
	"reflect"

	"github.com/knative/pkg/apis/duck"
	"github.com/knative/pkg/logging"
	netv1alpha1 "github.com/knative/serving/pkg/apis/networking/v1alpha1"
	"github.com/knative/serving/pkg/apis/serving"
	"github.com/knative/serving/pkg/apis/serving/v1alpha1"
	"github.com/knative/serving/pkg/reconciler/v1alpha1/route/config"
	"github.com/knative/serving/pkg/reconciler/v1alpha1/route/resources"
	resourcenames "github.com/knative/serving/pkg/reconciler/v1alpha1/route/resources/names"
	"github.com/knative/serving/pkg/reconciler/v1alpha1/route/traffic"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
)

func (c *Reconciler) getClusterIngressForRoute(route *v1alpha1.Route) (*netv1alpha1.ClusterIngress, error) {
	// First, look up the fixed name.
	ciName := resourcenames.ClusterIngress(route)
	ci, err := c.clusterIngressLister.Get(ciName)
	if err == nil {
		return ci, nil
	}

	// If that isn't found, then fallback on the legacy selector-based approach.
	selector := routeOwnerLabelSelector(route)
	ingresses, err := c.clusterIngressLister.List(selector)
	if err != nil {
		return nil, err
	}
	if len(ingresses) == 0 {
		return nil, apierrs.NewNotFound(
			v1alpha1.Resource("clusteringress"), resourcenames.ClusterIngress(route))
	}

	if len(ingresses) > 1 {
		// Return error as we expect only one ingress instance for a route.
		return nil, fmt.Errorf("more than one ClusterIngress are found for route %s/%s: %v", route.Namespace, route.Name, ingresses)
	}

	return ingresses[0], nil
}

func routeOwnerLabelSelector(route *v1alpha1.Route) labels.Selector {
	return labels.Set(map[string]string{
		serving.RouteLabelKey:          route.Name,
		serving.RouteNamespaceLabelKey: route.Namespace,
	}).AsSelector()
}

func (c *Reconciler) deleteClusterIngressesForRoute(route *v1alpha1.Route) error {
	selector := routeOwnerLabelSelector(route).String()

	// We always use DeleteCollection because even with a fixed name, we apply the labels.
	return c.ServingClientSet.NetworkingV1alpha1().ClusterIngresses().DeleteCollection(
		nil, metav1.ListOptions{LabelSelector: selector},
	)
}

func (c *Reconciler) reconcileClusterIngress(
	ctx context.Context, r *v1alpha1.Route, desired *netv1alpha1.ClusterIngress) (*netv1alpha1.ClusterIngress, error) {
	logger := logging.FromContext(ctx)
	clusterIngress, err := c.getClusterIngressForRoute(r)
	if apierrs.IsNotFound(err) {
		clusterIngress, err = c.ServingClientSet.NetworkingV1alpha1().ClusterIngresses().Create(desired)
		if err != nil {
			logger.Error("Failed to create ClusterIngress", zap.Error(err))
			c.Recorder.Eventf(r, corev1.EventTypeWarning, "CreationFailed",
				"Failed to create ClusterIngress for route %s/%s: %v", r.Namespace, r.Name, err)
			return nil, err
		}
		c.Recorder.Eventf(r, corev1.EventTypeNormal, "Created",
			"Created ClusterIngress %q", clusterIngress.Name)
		return clusterIngress, nil
	} else if err != nil {
		return nil, err
	} else {
		// TODO(#642): Remove this (needed to avoid continuous updates)
		desired.Spec.DeprecatedGeneration = clusterIngress.Spec.DeprecatedGeneration
		if !equality.Semantic.DeepEqual(clusterIngress.Spec, desired.Spec) {
			// Don't modify the informers copy
			origin := clusterIngress.DeepCopy()
			origin.Spec = desired.Spec

			updated, err := c.ServingClientSet.NetworkingV1alpha1().ClusterIngresses().Update(origin)
			if err != nil {
				logger.Error("Failed to update ClusterIngress", zap.Error(err))
				return nil, err
			}
			return updated, nil
		}
	}

	return clusterIngress, err
}

func (c *Reconciler) reconcilePlaceholderService(ctx context.Context, route *v1alpha1.Route, ingress *netv1alpha1.ClusterIngress) error {
	logger := logging.FromContext(ctx)
	ns := route.Namespace
	name := resourcenames.K8sService(route)

	desiredService, err := resources.MakeK8sService(route, ingress)
	if err != nil {
		// Loadbalancer not ready, no need to create.
		logger.Warnf("Failed to construct placeholder k8s service: %v", err)
		return nil
	}

	service, err := c.serviceLister.Services(ns).Get(name)
	if apierrs.IsNotFound(err) {
		// Doesn't exist, create it.
		service, err = c.KubeClientSet.CoreV1().Services(ns).Create(desiredService)
		if err != nil {
			logger.Error("Failed to create service", zap.Error(err))
			c.Recorder.Eventf(route, corev1.EventTypeWarning, "CreationFailed",
				"Failed to create service %q: %v", name, err)
			return err
		}
		logger.Infof("Created service %s", name)
		c.Recorder.Eventf(route, corev1.EventTypeNormal, "Created", "Created service %q", name)
	} else if err != nil {
		return err
	} else if !metav1.IsControlledBy(service, route) {
		// Surface an error in the route's status, and return an error.
		route.Status.MarkServiceNotOwned(name)
		return fmt.Errorf("Route: %q does not own Service: %q", route.Name, name)
	} else {
		// Make sure that the service has the proper specification.
		if !equality.Semantic.DeepEqual(service.Spec, desiredService.Spec) {
			// Don't modify the informers copy
			existing := service.DeepCopy()
			existing.Spec = desiredService.Spec
			_, err = c.KubeClientSet.CoreV1().Services(ns).Update(existing)
			if err != nil {
				return err
			}
		}
	}

	// TODO(mattmoor): This is where we'd look at the state of the Service and
	// reflect any necessary state into the Route.
	return nil
}

// Update the Status of the route.  Caller is responsible for checking
// for semantic differences before calling.
func (c *Reconciler) updateStatus(desired *v1alpha1.Route) (*v1alpha1.Route, error) {
	route, err := c.routeLister.Routes(desired.Namespace).Get(desired.Name)
	if err != nil {
		return nil, err
	}
	// If there's nothing to update, just return.
	if reflect.DeepEqual(route.Status, desired.Status) {
		return route, nil
	}
	// Don't modify the informers copy
	existing := route.DeepCopy()
	existing.Status = desired.Status
	return c.ServingClientSet.ServingV1alpha1().Routes(desired.Namespace).UpdateStatus(existing)
}

// Update the lastPinned annotation on revisions we target so they don't get GC'd.
func (c *Reconciler) reconcileTargetRevisions(ctx context.Context, t *traffic.Config, route *v1alpha1.Route) error {
	gcConfig := config.FromContext(ctx).GC
	lpDebounce := gcConfig.StaleRevisionLastpinnedDebounce

	eg, _ := errgroup.WithContext(ctx)
	for _, target := range t.Targets {
		for _, rt := range target {
			tt := rt.TrafficTarget
			eg.Go(func() error {
				rev, err := c.revisionLister.Revisions(route.Namespace).Get(tt.RevisionName)
				if apierrs.IsNotFound(err) {
					c.Logger.Infof("Unable to update lastPinned for missing revision %q", tt.RevisionName)
					return nil
				} else if err != nil {
					return err
				}

				newRev := rev.DeepCopy()
				lastPin, err := newRev.GetLastPinned()
				if err != nil {
					// Missing is an expected error case for a not yet pinned revision.
					if err.(v1alpha1.LastPinnedParseError).Type != v1alpha1.AnnotationParseErrorTypeMissing {
						return err
					}
				} else {
					// Enforce a delay before performing an update on lastPinned to avoid excess churn.
					if lastPin.Add(lpDebounce).After(c.clock.Now()) {
						return nil
					}
				}

				if newRev.Annotations == nil {
					newRev.Annotations = make(map[string]string)
				}

				newRev.ObjectMeta.Annotations[serving.RevisionLastPinnedAnnotationKey] = v1alpha1.RevisionLastPinnedString(c.clock.Now())
				patch, err := duck.CreateMergePatch(rev, newRev)
				if err != nil {
					return err
				}

				if _, err := c.ServingClientSet.ServingV1alpha1().Revisions(route.Namespace).Patch(rev.Name, types.MergePatchType, patch); err != nil {
					c.Logger.Errorf("Unable to set revision annotation: %v", err)
					return err
				}
				return nil
			})
		}
	}
	return eg.Wait()
}
