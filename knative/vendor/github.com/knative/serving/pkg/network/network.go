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

package network

import (
	"net"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

const (
	// ProbeHeaderName is the name of a header that can be added to
	// requests to probe the knative networking layer.  Requests
	// with this header will not be passed to the user container or
	// included in request metrics.
	ProbeHeaderName = "k-network-probe"

	// ConfigName is the name of the configmap containing all
	// customizations for networking features.
	ConfigName = "config-network"

	// IstioOutboundIPRangesKey is the name of the configuration entry
	// that specifies Istio outbound ip ranges.
	IstioOutboundIPRangesKey = "istio.sidecar.includeOutboundIPRanges"

	// DefaultClusterIngressClassKey is the name of the configuration entry
	// that specifies the default ClusterIngress.
	DefaultClusterIngressClassKey = "clusteringress.class"

	// IstioIngressClassName value for specifying knative's Istio
	// ClusterIngress reconciler.
	IstioIngressClassName = "istio.ingress.networking.knative.dev"
)

// Config contains the networking configuration defined in the
// network config map.
type Config struct {
	// IstioOutboundIPRange specifies the IP ranges to intercept
	// by Istio sidecar.
	IstioOutboundIPRanges string

	// DefaultClusterIngressClass specifies the default ClusterIngress class.
	DefaultClusterIngressClass string
}

func validateAndNormalizeOutboundIPRanges(s string) (string, error) {
	s = strings.TrimSpace(s)

	// * is a valid value
	if s == "*" {
		return s, nil
	}

	cidrs := strings.Split(s, ",")
	var normalized []string
	for _, cidr := range cidrs {
		cidr = strings.TrimSpace(cidr)
		if len(cidr) == 0 {
			continue
		}
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			return "", err
		}

		normalized = append(normalized, cidr)
	}

	return strings.Join(normalized, ","), nil
}

// NewConfigFromConfigMap creates a Config from the supplied ConfigMap
func NewConfigFromConfigMap(configMap *corev1.ConfigMap) (*Config, error) {
	nc := &Config{}
	if ipr, ok := configMap.Data[IstioOutboundIPRangesKey]; !ok {
		// It is OK for this to be absent, we will elide the annotation.
		nc.IstioOutboundIPRanges = "*"
	} else if normalizedIpr, err := validateAndNormalizeOutboundIPRanges(ipr); err != nil {
		return nil, err
	} else {
		nc.IstioOutboundIPRanges = normalizedIpr
	}
	if ingressClass, ok := configMap.Data[DefaultClusterIngressClassKey]; !ok {
		nc.DefaultClusterIngressClass = IstioIngressClassName
	} else {
		nc.DefaultClusterIngressClass = ingressClass
	}
	return nc, nil
}
