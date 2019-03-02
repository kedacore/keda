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

package config

import (
	"fmt"
	"sort"
	"strings"

	"github.com/knative/serving/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	// IstioConfigName is the name of the configmap containing all
	// customizations for istio related features.
	IstioConfigName = "config-istio"

	// GatewayKeyPrefix is the prefix of all keys to configure Istio gateways for public ClusterIngresses.
	GatewayKeyPrefix = "gateway."

	// LocalGatewayKeyPrefix is the prefix of all keys to configure Istio gateways for public & private ClusterIngresses.
	LocalGatewayKeyPrefix = "local-gateway."
)

var (
	defaultGateway = Gateway{
		GatewayName: "knative-ingress-gateway",
		ServiceURL:  fmt.Sprintf("istio-ingressgateway.istio-system.svc.%s", utils.GetClusterDomainName()),
	}
)

// Gateway specifies the name of the Gateway and the K8s Service backing it.
type Gateway struct {
	GatewayName string
	ServiceURL  string
}

// Istio contains istio related configuration defined in the
// istio config map.
type Istio struct {
	// IngressGateway specifies the gateway urls for public ClusterIngress.
	IngressGateways []Gateway

	// LocalGateway specifies the gateway urls for public & private ClusterIngress.
	LocalGateways []Gateway
}

func parseGateways(configMap *corev1.ConfigMap, prefix string) ([]Gateway, error) {
	urls := map[string]string{}
	gatewayNames := []string{}
	for k, v := range configMap.Data {
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		gatewayName, serviceURL := k[len(prefix):], v
		if errs := validation.IsDNS1123Subdomain(serviceURL); len(errs) > 0 {
			return nil, fmt.Errorf("invalid gateway format: %v", errs)
		}
		gatewayNames = append(gatewayNames, gatewayName)
		urls[gatewayName] = serviceURL
	}
	sort.Strings(gatewayNames)
	gateways := make([]Gateway, len(gatewayNames))
	for i, gatewayName := range gatewayNames {
		gateways[i] = Gateway{
			GatewayName: gatewayName,
			ServiceURL:  urls[gatewayName],
		}
	}
	return gateways, nil
}

// NewIstioFromConfigMap creates an Istio config from the supplied ConfigMap
func NewIstioFromConfigMap(configMap *corev1.ConfigMap) (*Istio, error) {
	gateways, err := parseGateways(configMap, GatewayKeyPrefix)
	if err != nil {
		return nil, err
	}
	if len(gateways) == 0 {
		gateways = append(gateways, defaultGateway)
	}
	localGateways, err := parseGateways(configMap, LocalGatewayKeyPrefix)
	if err != nil {
		return nil, err
	}
	return &Istio{
		IngressGateways: gateways,
		LocalGateways:   localGateways,
	}, nil
}
