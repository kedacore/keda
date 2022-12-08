/*
Copyright 2018 The Kubernetes Authors.

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

package apiserver

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	genericapi "k8s.io/apiserver/pkg/endpoints"
	"k8s.io/apiserver/pkg/endpoints/discovery"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/metrics/pkg/apis/external_metrics"

	specificapi "sigs.k8s.io/custom-metrics-apiserver/pkg/apiserver/installer"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"
	metricstorage "sigs.k8s.io/custom-metrics-apiserver/pkg/registry/external_metrics"
)

// InstallExternalMetricsAPI registers the api server in Kube Aggregator
func (s *CustomMetricsAdapterServer) InstallExternalMetricsAPI() error {
	groupInfo := genericapiserver.NewDefaultAPIGroupInfo(external_metrics.GroupName, Scheme, metav1.ParameterCodec, Codecs)

	mainGroupVer := groupInfo.PrioritizedVersions[0]
	preferredVersionForDiscovery := metav1.GroupVersionForDiscovery{
		GroupVersion: mainGroupVer.String(),
		Version:      mainGroupVer.Version,
	}
	groupVersion := metav1.GroupVersionForDiscovery{
		GroupVersion: mainGroupVer.String(),
		Version:      mainGroupVer.Version,
	}
	apiGroup := metav1.APIGroup{
		Name:             mainGroupVer.Group,
		Versions:         []metav1.GroupVersionForDiscovery{groupVersion},
		PreferredVersion: preferredVersionForDiscovery,
	}

	emAPI := s.emAPI(&groupInfo, mainGroupVer)
	if err := emAPI.InstallREST(s.GenericAPIServer.Handler.GoRestfulContainer); err != nil {
		return err
	}

	s.GenericAPIServer.DiscoveryGroupManager.AddGroup(apiGroup)
	s.GenericAPIServer.Handler.GoRestfulContainer.Add(discovery.NewAPIGroupHandler(s.GenericAPIServer.Serializer, apiGroup).WebService())

	return nil
}

func (s *CustomMetricsAdapterServer) emAPI(groupInfo *genericapiserver.APIGroupInfo, groupVersion schema.GroupVersion) *specificapi.MetricsAPIGroupVersion {
	resourceStorage := metricstorage.NewREST(s.externalMetricsProvider)

	return &specificapi.MetricsAPIGroupVersion{
		DynamicStorage: resourceStorage,
		APIGroupVersion: &genericapi.APIGroupVersion{
			Root:             genericapiserver.APIGroupPrefix,
			GroupVersion:     groupVersion,
			MetaGroupVersion: groupInfo.MetaGroupVersion,

			ParameterCodec:  groupInfo.ParameterCodec,
			Serializer:      groupInfo.NegotiatedSerializer,
			Creater:         groupInfo.Scheme,
			Convertor:       groupInfo.Scheme,
			UnsafeConvertor: runtime.UnsafeObjectConvertor(groupInfo.Scheme),
			Typer:           groupInfo.Scheme,
			Namer:           runtime.Namer(meta.NewAccessor()),
		},
		ResourceLister: provider.NewExternalMetricResourceLister(s.externalMetricsProvider),
		Handlers:       &specificapi.EMHandlers{},
	}
}
