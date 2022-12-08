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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/version"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/client-go/informers"
	cminstall "k8s.io/metrics/pkg/apis/custom_metrics/install"
	eminstall "k8s.io/metrics/pkg/apis/external_metrics/install"

	"sigs.k8s.io/custom-metrics-apiserver/pkg/apiserver/installer"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"
)

var (
	Scheme = runtime.NewScheme()
	Codecs = serializer.NewCodecFactory(Scheme)
)

func init() {
	cminstall.Install(Scheme)
	eminstall.Install(Scheme)

	// we need custom conversion functions to list resources with options
	utilruntime.Must(installer.RegisterConversions(Scheme))

	// we need to add the options to empty v1
	// TODO fix the server code to avoid this
	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})

	// TODO: keep the generic API server from wanting this
	unversioned := schema.GroupVersion{Group: "", Version: "v1"}
	Scheme.AddUnversionedTypes(unversioned,
		&metav1.Status{},
		&metav1.APIVersions{},
		&metav1.APIGroupList{},
		&metav1.APIGroup{},
		&metav1.APIResourceList{},
	)
}

type Config struct {
	GenericConfig *genericapiserver.Config
}

// CustomMetricsAdapterServer contains state for a Kubernetes cluster master/api server.
type CustomMetricsAdapterServer struct {
	GenericAPIServer        *genericapiserver.GenericAPIServer
	customMetricsProvider   provider.CustomMetricsProvider
	externalMetricsProvider provider.ExternalMetricsProvider
}

type CompletedConfig struct {
	genericapiserver.CompletedConfig
}

// Complete fills in any fields not set that are required to have valid data. It's mutating the receiver.
func (c *Config) Complete(informers informers.SharedInformerFactory) CompletedConfig {
	c.GenericConfig.Version = &version.Info{
		Major: "1",
		Minor: "0",
	}
	return CompletedConfig{c.GenericConfig.Complete(informers)}
}

// New returns a new instance of CustomMetricsAdapterServer from the given config.
// name is used to differentiate for logging.
// Each of the arguments: customMetricsProvider, externalMetricsProvider can be set either to
// a provider implementation, or to nil to disable one of the APIs.
func (c CompletedConfig) New(name string, customMetricsProvider provider.CustomMetricsProvider, externalMetricsProvider provider.ExternalMetricsProvider) (*CustomMetricsAdapterServer, error) {
	genericServer, err := c.CompletedConfig.New(name, genericapiserver.NewEmptyDelegate()) // completion is done in Complete, no need for a second time
	if err != nil {
		return nil, err
	}

	s := &CustomMetricsAdapterServer{
		GenericAPIServer:        genericServer,
		customMetricsProvider:   customMetricsProvider,
		externalMetricsProvider: externalMetricsProvider,
	}

	if customMetricsProvider != nil {
		if err := s.InstallCustomMetricsAPI(); err != nil {
			return nil, err
		}
	}
	if externalMetricsProvider != nil {
		if err := s.InstallExternalMetricsAPI(); err != nil {
			return nil, err
		}
	}

	return s, nil
}
