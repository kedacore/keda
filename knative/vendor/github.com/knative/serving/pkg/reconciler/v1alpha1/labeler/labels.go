/*
Copyright 2018 The Knative Authors

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

package labeler

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/knative/pkg/logging"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/knative/serving/pkg/apis/serving"
	"github.com/knative/serving/pkg/apis/serving/v1alpha1"
	servingv1alpha1 "github.com/knative/serving/pkg/client/clientset/versioned/typed/serving/v1alpha1"
)

func (c *Reconciler) syncLabels(ctx context.Context, r *v1alpha1.Route) error {
	configs := sets.NewString()
	// Walk the revisions in Route's .status.traffic and build a list
	// of Configurations to label from their OwnerReferences.
	for _, tt := range r.Status.Traffic {
		rev, err := c.revisionLister.Revisions(r.Namespace).Get(tt.RevisionName)
		if err != nil {
			return err
		}
		owner := metav1.GetControllerOf(rev)
		if owner != nil && owner.Kind == "Configuration" {
			configs.Insert(owner.Name)
		}
	}

	if err := c.deleteLabelForOutsideOfGivenConfigurations(ctx, r.Namespace, r.Name, configs); err != nil {
		return err
	}
	return c.setLabelForGivenConfigurations(ctx, r, configs)
}

func (c *Reconciler) setLabelForGivenConfigurations(
	ctx context.Context,
	route *v1alpha1.Route,
	configs sets.String) error {
	logger := logging.FromContext(ctx)

	// The ordered collection of Configurations to which we
	// should patch our Route label.
	configurationOrder := []string{}
	configMap := make(map[string]*v1alpha1.Configuration)

	// Lookup Configurations that are missing our Route label.
	for name := range configs {
		configurationOrder = append(configurationOrder, name)

		config, err := c.configurationLister.Configurations(route.Namespace).Get(name)
		if err != nil {
			return err
		}
		configMap[name] = config
		routeName, ok := config.Labels[serving.RouteLabelKey]
		if !ok {
			continue
		}
		if routeName != route.Name {
			return fmt.Errorf("Configuration %q is already in use by %q, and cannot be used by %q",
				config.Name, routeName, route.Name)
		}
	}
	// Sort the names to give things a deterministic ordering.
	sort.Strings(configurationOrder)

	configClient := c.ServingClientSet.ServingV1alpha1().Configurations(route.Namespace)
	// Set label for newly added configurations as traffic target.
	for _, configName := range configurationOrder {
		config := configMap[configName]
		if config.Labels == nil {
			config.Labels = make(map[string]string)
		} else if _, ok := config.Labels[serving.RouteLabelKey]; ok {
			continue
		}

		if err := setRouteLabelForConfiguration(configClient, config.Name, config.ResourceVersion, &route.Name); err != nil {
			logger.Errorf("Failed to add route label to configuration %q: %s", config.Name, err)
			return err
		}
	}

	return nil
}

func (c *Reconciler) deleteLabelForOutsideOfGivenConfigurations(
	ctx context.Context,
	routeNamespace, routeName string,
	configs sets.String,
) error {

	logger := logging.FromContext(ctx)

	// Get Configurations set as traffic target before this sync.
	selector := labels.SelectorFromSet(labels.Set{serving.RouteLabelKey: routeName})

	oldConfigsList, err := c.configurationLister.Configurations(routeNamespace).List(selector)
	if err != nil {
		return err
	}

	// Delete label for newly removed configurations as traffic target.
	configClient := c.ServingClientSet.ServingV1alpha1().Configurations(routeNamespace)
	for _, config := range oldConfigsList {
		if configs.Has(config.Name) {
			continue
		}

		if err := setRouteLabelForConfiguration(configClient, config.Name, config.ResourceVersion, nil); err != nil {
			logger.Errorf("Failed to remove route label to configuration %q: %s", config.Name, err)
			return err
		}
	}

	return nil
}

func setRouteLabelForConfiguration(
	configClient servingv1alpha1.ConfigurationInterface,
	configName string,
	configVersion string,
	routeName *string, // a nil route name will cause the route label to be deleted
) error {

	mergePatch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"labels": map[string]interface{}{
				serving.RouteLabelKey: routeName,
			},
			"resourceVersion": configVersion,
		},
	}

	patch, err := json.Marshal(mergePatch)
	if err != nil {
		return err
	}

	_, err = configClient.Patch(configName, types.MergePatchType, patch)
	return err
}
