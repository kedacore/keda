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
	"fmt"

	caching "github.com/knative/caching/pkg/apis/caching/v1alpha1"
	"github.com/knative/pkg/kmp"
	"github.com/knative/pkg/logging"
	kpav1alpha1 "github.com/knative/serving/pkg/apis/autoscaling/v1alpha1"
	"github.com/knative/serving/pkg/apis/serving/v1alpha1"
	"github.com/knative/serving/pkg/reconciler/v1alpha1/revision/config"
	"github.com/knative/serving/pkg/reconciler/v1alpha1/revision/resources"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

func (c *Reconciler) createDeployment(ctx context.Context, rev *v1alpha1.Revision) (*appsv1.Deployment, error) {
	cfgs := config.FromContext(ctx)

	deployment := resources.MakeDeployment(
		rev,
		cfgs.Logging,
		cfgs.Network,
		cfgs.Observability,
		cfgs.Autoscaler,
		cfgs.Controller,
	)

	return c.KubeClientSet.AppsV1().Deployments(deployment.Namespace).Create(deployment)
}

func (c *Reconciler) checkAndUpdateDeployment(ctx context.Context, rev *v1alpha1.Revision, have *appsv1.Deployment) (*appsv1.Deployment, Changed, error) {
	logger := logging.FromContext(ctx)
	cfgs := config.FromContext(ctx)

	deployment := resources.MakeDeployment(
		rev,
		cfgs.Logging,
		cfgs.Network,
		cfgs.Observability,
		cfgs.Autoscaler,
		cfgs.Controller,
	)

	// Preserve the current scale of the Deployment.
	deployment.Spec.Replicas = have.Spec.Replicas

	// Preserve the label selector since it's immutable
	// TODO(dprotaso) Determine other immutable properties
	deployment.Spec.Selector = have.Spec.Selector

	// If the spec we want is the spec we have, then we're good.
	if equality.Semantic.DeepEqual(have.Spec, deployment.Spec) {
		return have, Unchanged, nil
	}

	// Otherwise attempt an update (with ONLY the spec changes).
	desiredDeployment := have.DeepCopy()
	desiredDeployment.Spec = deployment.Spec

	// carry over new labels
	for k, v := range deployment.Labels {
		desiredDeployment.Labels[k] = v
	}

	d, err := c.KubeClientSet.AppsV1().Deployments(deployment.Namespace).Update(desiredDeployment)
	if err != nil {
		return nil, Unchanged, err
	}

	// If what comes back from the update (with defaults applied by the API server) is the same
	// as what we have then nothing changed.
	if equality.Semantic.DeepEqual(have.Spec, d.Spec) {
		return d, Unchanged, nil
	}
	diff, err := kmp.SafeDiff(have.Spec, d.Spec)
	if err != nil {
		return nil, Unchanged, err
	}

	// If what comes back has a different spec, then signal the change.
	logger.Infof("Reconciled deployment diff (-desired, +observed): %v", diff)
	return d, WasChanged, nil
}

func (c *Reconciler) createImageCache(ctx context.Context, rev *v1alpha1.Revision, deploy *appsv1.Deployment) (*caching.Image, error) {
	image, err := resources.MakeImageCache(rev, deploy)
	if err != nil {
		return nil, err
	}

	return c.CachingClientSet.CachingV1alpha1().Images(image.Namespace).Create(image)
}

func (c *Reconciler) createKPA(ctx context.Context, rev *v1alpha1.Revision) (*kpav1alpha1.PodAutoscaler, error) {
	kpa := resources.MakeKPA(rev)

	return c.ServingClientSet.AutoscalingV1alpha1().PodAutoscalers(kpa.Namespace).Create(kpa)
}

type serviceFactory func(*v1alpha1.Revision) *corev1.Service

func (c *Reconciler) createService(ctx context.Context, rev *v1alpha1.Revision, sf serviceFactory) (*corev1.Service, error) {
	// Create the service.
	service := sf(rev)

	return c.KubeClientSet.CoreV1().Services(service.Namespace).Create(service)
}

func (c *Reconciler) checkAndUpdateService(ctx context.Context, rev *v1alpha1.Revision, sf serviceFactory, service *corev1.Service) (*corev1.Service, Changed, error) {
	logger := logging.FromContext(ctx)

	// Note: only reconcile the spec we set.
	rawDesiredService := sf(rev)
	desiredService := service.DeepCopy()
	desiredService.Spec.Selector = rawDesiredService.Spec.Selector
	desiredService.Spec.Ports = rawDesiredService.Spec.Ports

	if equality.Semantic.DeepEqual(desiredService.Spec, service.Spec) {
		return service, Unchanged, nil
	}
	diff, err := kmp.SafeDiff(desiredService.Spec, service.Spec)
	if err != nil {
		return nil, Unchanged, fmt.Errorf("failed to diff Service: %v", err)
	}
	logger.Infof("Reconciling service diff (-desired, +observed): %v", diff)

	d, err := c.KubeClientSet.CoreV1().Services(service.Namespace).Update(desiredService)
	return d, WasChanged, err
}
