/*
Copyright 2026 The KEDA Authors

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

package keda

import (
	"context"
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/util"
)

const (
	triggerAuthSecretRefIndex = ".spec.secretTargetRef.name"
	scalableAuthRefIndex      = ".spec.triggers.authenticationRef"

	authRefTAPrefix  = "ta/"
	authRefCTAPrefix = "cta/"
)

var (
	registerTriggerAuthIndexOnce sync.Once
	triggerAuthIndexErr          error
)

// extractAuthRefKeys returns composite index keys for a set of ScaleTriggers.
// TriggerAuthentication refs produce "ta/<name>", ClusterTriggerAuthentication
// refs produce "cta/<name>".
func extractAuthRefKeys(triggers []kedav1alpha1.ScaleTriggers) []string {
	var keys []string
	for _, trigger := range triggers {
		if trigger.AuthenticationRef == nil {
			continue
		}
		if trigger.AuthenticationRef.Kind == "ClusterTriggerAuthentication" {
			keys = append(keys, authRefCTAPrefix+trigger.AuthenticationRef.Name)
		} else {
			keys = append(keys, authRefTAPrefix+trigger.AuthenticationRef.Name)
		}
	}
	return keys
}

// registerTriggerAuthIndexes creates field indexes so that the
// Secret→ScaledObject/ScaledJob mapping functions can use indexed lookups
// instead of listing every object and scanning in-memory.
//
// Indexes registered:
//   - TriggerAuthentication / ClusterTriggerAuthentication by secret ref name
//   - ScaledObject by trigger authentication ref (composite key)
func registerTriggerAuthIndexes(mgr ctrl.Manager) error {
	registerTriggerAuthIndexOnce.Do(func() {
		ctx := context.Background()

		if err := mgr.GetFieldIndexer().IndexField(ctx,
			&kedav1alpha1.TriggerAuthentication{},
			triggerAuthSecretRefIndex,
			func(obj client.Object) []string {
				ta := obj.(*kedav1alpha1.TriggerAuthentication)
				names := make([]string, 0, len(ta.Spec.SecretTargetRef))
				for _, ref := range ta.Spec.SecretTargetRef {
					names = append(names, ref.Name)
				}
				return names
			}); err != nil {
			triggerAuthIndexErr = fmt.Errorf("failed to register TriggerAuthentication index %q: %w",
				triggerAuthSecretRefIndex, err)
			return
		}

		if err := mgr.GetFieldIndexer().IndexField(ctx,
			&kedav1alpha1.ClusterTriggerAuthentication{},
			triggerAuthSecretRefIndex,
			func(obj client.Object) []string {
				cta := obj.(*kedav1alpha1.ClusterTriggerAuthentication)
				names := make([]string, 0, len(cta.Spec.SecretTargetRef))
				for _, ref := range cta.Spec.SecretTargetRef {
					names = append(names, ref.Name)
				}
				return names
			}); err != nil {
			triggerAuthIndexErr = fmt.Errorf("failed to register ClusterTriggerAuthentication index %q: %w",
				triggerAuthSecretRefIndex, err)
			return
		}

		if err := mgr.GetFieldIndexer().IndexField(ctx,
			&kedav1alpha1.ScaledObject{},
			scalableAuthRefIndex,
			func(obj client.Object) []string {
				return extractAuthRefKeys(obj.(*kedav1alpha1.ScaledObject).Spec.Triggers)
			}); err != nil {
			triggerAuthIndexErr = fmt.Errorf("failed to register ScaledObject index %q: %w",
				scalableAuthRefIndex, err)
			return
		}
	})
	return triggerAuthIndexErr
}

// objectListToRequests converts any ObjectList to reconcile requests by
// extracting Name and Namespace from each item's ObjectMeta.
func objectListToRequests(list client.ObjectList) []reconcile.Request {
	items, err := meta.ExtractList(list)
	if err != nil {
		return nil
	}
	requests := make([]reconcile.Request, 0, len(items))
	for _, item := range items {
		o, ok := item.(client.Object)
		if !ok {
			continue
		}
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      o.GetName(),
				Namespace: o.GetNamespace(),
			},
		})
	}
	return requests
}

// lookupByTriggerAuth returns reconcile requests for scalable objects in the
// given namespace that reference the named TriggerAuthentication via the
// scalableAuthRefIndex.
func lookupByTriggerAuth(ctx context.Context, c client.Reader, list client.ObjectList, taName, namespace string) []reconcile.Request {
	if err := c.List(ctx, list,
		client.InNamespace(namespace),
		client.MatchingFields{scalableAuthRefIndex: authRefTAPrefix + taName}); err != nil {
		log.FromContext(ctx).Error(err, "failed to list scalable objects by TriggerAuthentication",
			"triggerAuthentication", taName, "namespace", namespace)
		return nil
	}
	return objectListToRequests(list)
}

// lookupByClusterTriggerAuth returns reconcile requests for scalable objects
// (across all namespaces) that reference the named ClusterTriggerAuthentication
// via the scalableAuthRefIndex.
func lookupByClusterTriggerAuth(ctx context.Context, c client.Reader, list client.ObjectList, ctaName string) []reconcile.Request {
	if err := c.List(ctx, list,
		client.MatchingFields{scalableAuthRefIndex: authRefCTAPrefix + ctaName}); err != nil {
		log.FromContext(ctx).Error(err, "failed to list scalable objects by ClusterTriggerAuthentication",
			"clusterTriggerAuthentication", ctaName)
		return nil
	}
	return objectListToRequests(list)
}

// mapSecretToScalableObjects finds TriggerAuthentications and
// ClusterTriggerAuthentications that reference the given Secret, then returns
// reconcile requests for all scalable objects that use those auth resources.
// newList must return a fresh, empty ObjectList of the target type
// (e.g. &ScaledObjectList{} or &ScaledJobList{}).
func mapSecretToScalableObjects(ctx context.Context, c client.Reader, newList func() client.ObjectList, secretName, secretNamespace string) []reconcile.Request {
	logger := log.FromContext(ctx)

	kedaNamespace, kedaNsErr := util.GetClusterObjectNamespace()
	if kedaNsErr != nil {
		logger.V(1).Info("Skipping ClusterTriggerAuthentication lookup: cannot determine KEDA namespace",
			"error", kedaNsErr)
	}

	restrictedSecretAccess := util.IsRestrictSecretAccess()
	if restrictedSecretAccess && (kedaNsErr != nil || secretNamespace != kedaNamespace) {
		// With restricted secret access, secrets are only ever resolved from
		// the KEDA namespace, so secrets elsewhere cannot affect any auth.
		return nil
	}

	taListOpts := []client.ListOption{client.MatchingFields{triggerAuthSecretRefIndex: secretName}}
	if !restrictedSecretAccess {
		// Normally a TriggerAuthentication reads secrets from its own
		// namespace only. With restricted secret access every
		// TriggerAuthentication resolves secrets from the KEDA namespace
		// instead, so a KEDA-namespace secret must match TAs cluster-wide.
		taListOpts = append(taListOpts, client.InNamespace(secretNamespace))
	}
	taList := &kedav1alpha1.TriggerAuthenticationList{}
	if err := c.List(ctx, taList, taListOpts...); err != nil {
		logger.Error(err, "failed to list TriggerAuthentications for Secret mapping",
			"secret", secretName, "namespace", secretNamespace)
		return nil
	}

	var ctaNames []string
	if kedaNsErr == nil && secretNamespace == kedaNamespace {
		ctaList := &kedav1alpha1.ClusterTriggerAuthenticationList{}
		if err := c.List(ctx, ctaList,
			client.MatchingFields{triggerAuthSecretRefIndex: secretName}); err == nil {
			for _, cta := range ctaList.Items {
				ctaNames = append(ctaNames, cta.Name)
			}
		} else {
			logger.Error(err, "failed to list ClusterTriggerAuthentications for Secret mapping",
				"secret", secretName, "namespace", secretNamespace)
		}
	}

	if len(taList.Items) == 0 && len(ctaNames) == 0 {
		return nil
	}

	var requests []reconcile.Request
	seen := make(map[types.NamespacedName]bool)

	for _, ta := range taList.Items {
		for _, req := range lookupByTriggerAuth(ctx, c, newList(), ta.Name, ta.Namespace) {
			if !seen[req.NamespacedName] {
				requests = append(requests, req)
				seen[req.NamespacedName] = true
			}
		}
	}

	for _, ctaName := range ctaNames {
		for _, req := range lookupByClusterTriggerAuth(ctx, c, newList(), ctaName) {
			if !seen[req.NamespacedName] {
				requests = append(requests, req)
				seen[req.NamespacedName] = true
			}
		}
	}

	return requests
}
