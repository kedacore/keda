/*
Copyright 2017 The Kubernetes Authors.

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
	"k8s.io/metrics/pkg/apis/custom_metrics"

	specificapi "sigs.k8s.io/custom-metrics-apiserver/pkg/apiserver/installer"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"
	metricstorage "sigs.k8s.io/custom-metrics-apiserver/pkg/registry/custom_metrics"
)

func (s *CustomMetricsAdapterServer) InstallCustomMetricsAPI() error {
	groupInfo := genericapiserver.NewDefaultAPIGroupInfo(custom_metrics.GroupName, Scheme, runtime.NewParameterCodec(Scheme), Codecs)
	container := s.GenericAPIServer.Handler.GoRestfulContainer

	// Register custom metrics REST handler for all supported API versions.
	for versionIndex, mainGroupVer := range groupInfo.PrioritizedVersions {
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

		cmAPI := s.cmAPI(&groupInfo, mainGroupVer)
		if err := cmAPI.InstallREST(container); err != nil {
			return err
		}

		if versionIndex == 0 {
			s.GenericAPIServer.DiscoveryGroupManager.AddGroup(apiGroup)
			container.Add(discovery.NewAPIGroupHandler(s.GenericAPIServer.Serializer, apiGroup).WebService())
		}
	}
	return nil
}

func (s *CustomMetricsAdapterServer) cmAPI(groupInfo *genericapiserver.APIGroupInfo, groupVersion schema.GroupVersion) *specificapi.MetricsAPIGroupVersion {
	resourceStorage := metricstorage.NewREST(s.customMetricsProvider)

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

		ResourceLister: provider.NewCustomMetricResourceLister(s.customMetricsProvider),
		Handlers:       &specificapi.CMHandlers{},
	}
}
