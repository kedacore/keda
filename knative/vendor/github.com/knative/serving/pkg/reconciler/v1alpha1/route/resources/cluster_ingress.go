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

package resources

import (
	"fmt"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/knative/pkg/system"
	"github.com/knative/serving/pkg/activator"
	"github.com/knative/serving/pkg/apis/networking"
	"github.com/knative/serving/pkg/apis/networking/v1alpha1"
	"github.com/knative/serving/pkg/apis/serving"
	servingv1alpha1 "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	"github.com/knative/serving/pkg/reconciler"
	revisionresources "github.com/knative/serving/pkg/reconciler/v1alpha1/revision/resources"
	"github.com/knative/serving/pkg/reconciler/v1alpha1/route/resources/names"
	"github.com/knative/serving/pkg/reconciler/v1alpha1/route/traffic"
	"github.com/knative/serving/pkg/utils"
)

func isClusterLocal(r *servingv1alpha1.Route) bool {
	return strings.HasSuffix(r.Status.Domain, utils.GetClusterDomainName())
}

// MakeClusterIngress creates ClusterIngress to set up routing rules. Such ClusterIngress specifies
// which Hosts that it applies to, as well as the routing rules.
func MakeClusterIngress(r *servingv1alpha1.Route, tc *traffic.Config, ingressClass string) *v1alpha1.ClusterIngress {
	ci := &v1alpha1.ClusterIngress{
		ObjectMeta: metav1.ObjectMeta{
			// As ClusterIngress resource is cluster-scoped,
			// here we use GenerateName to avoid conflict.
			Name: names.ClusterIngress(r),
			Labels: map[string]string{
				serving.RouteLabelKey:          r.Name,
				serving.RouteNamespaceLabelKey: r.Namespace,
			},
			Annotations: r.ObjectMeta.Annotations,
		},
		Spec: makeClusterIngressSpec(r, tc.Targets),
	}
	// Set the ingress class annotation.
	if ci.ObjectMeta.Annotations == nil {
		ci.ObjectMeta.Annotations = make(map[string]string)
	}
	ci.ObjectMeta.Annotations[networking.IngressClassAnnotationKey] = ingressClass
	return ci
}

func makeClusterIngressSpec(r *servingv1alpha1.Route, targets map[string]traffic.RevisionTargets) v1alpha1.IngressSpec {
	// Domain should have been specified in route status
	// before calling this func.
	names := make([]string, 0, len(targets))
	for name := range targets {
		names = append(names, name)
	}
	// Sort the names to give things a deterministic ordering.
	sort.Strings(names)

	// The routes are matching rule based on domain name to traffic split targets.
	rules := make([]v1alpha1.ClusterIngressRule, 0, len(names))
	for _, name := range names {
		rules = append(rules, *makeClusterIngressRule(
			routeDomains(name, r), r.Namespace, targets[name]))
	}
	spec := v1alpha1.IngressSpec{
		Rules:      rules,
		Visibility: v1alpha1.IngressVisibilityExternalIP,
	}
	if isClusterLocal(r) {
		spec.Visibility = v1alpha1.IngressVisibilityClusterLocal
	}
	return spec
}

func routeDomains(targetName string, r *servingv1alpha1.Route) []string {
	if targetName == traffic.DefaultTarget {
		// Nameless traffic targets correspond to many domains: the
		// Route.Status.Domain.
		domains := []string{
			r.Status.Domain,
			names.K8sServiceFullname(r),
		}
		return dedup(domains)
	}

	return []string{fmt.Sprintf("%s.%s", targetName, r.Status.Domain)}
}
func makeClusterIngressRule(domains []string, ns string, targets traffic.RevisionTargets) *v1alpha1.ClusterIngressRule {
	active, inactive := targets.GroupTargets()
	// Optimistically allocate |active| elements.
	splits := make([]v1alpha1.ClusterIngressBackendSplit, 0, len(active))
	for _, t := range active {
		splits = append(splits, v1alpha1.ClusterIngressBackendSplit{
			ClusterIngressBackend: v1alpha1.ClusterIngressBackend{
				ServiceNamespace: ns,
				ServiceName:      reconciler.GetServingK8SServiceNameForObj(t.TrafficTarget.RevisionName),
				ServicePort:      intstr.FromInt(int(revisionresources.ServicePort)),
			},
			Percent: t.Percent,
		})
	}
	path := &v1alpha1.HTTPClusterIngressPath{
		Splits: splits,
		// TODO(lichuqiang): #2201, plumbing to config timeout and retries.
	}
	path.SetDefaults()
	return &v1alpha1.ClusterIngressRule{
		Hosts: domains,
		HTTP: &v1alpha1.HTTPClusterIngressRuleValue{
			Paths: []v1alpha1.HTTPClusterIngressPath{
				*addInactive(path, ns, inactive),
			},
		},
	}
}

// addInactive constructs Splits for the inactive targets, and add into given IngressPath.
func addInactive(r *v1alpha1.HTTPClusterIngressPath, ns string, inactive traffic.RevisionTargets) *v1alpha1.HTTPClusterIngressPath {
	if len(inactive) == 0 {
		return r
	}
	totalInactivePercent := 0
	maxInactiveTarget := &inactive[0]
	for i, t := range inactive {
		totalInactivePercent += t.Percent
		if t.Percent >= maxInactiveTarget.Percent {
			maxInactiveTarget = &inactive[i]
		}
	}
	r.Splits = append(r.Splits, v1alpha1.ClusterIngressBackendSplit{
		ClusterIngressBackend: v1alpha1.ClusterIngressBackend{
			ServiceNamespace: system.Namespace(),
			ServiceName:      activator.K8sServiceName,
			ServicePort:      intstr.FromInt(int(activator.ServicePort(maxInactiveTarget.Protocol))),
		},
		Percent: totalInactivePercent,
	})
	r.AppendHeaders = map[string]string{
		activator.RevisionHeaderName:      maxInactiveTarget.RevisionName,
		activator.RevisionHeaderNamespace: ns,
	}
	return r
}

func dedup(strs []string) []string {
	existed := sets.NewString()
	unique := []string{}
	for _, s := range strs {
		if !existed.Has(s) {
			existed.Insert(s)
			unique = append(unique, s)
		}
	}
	return unique
}
