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

package cmd

import (
	"fmt"
	"sync"
	"time"

	"github.com/spf13/pflag"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	openapinamer "k8s.io/apiserver/pkg/endpoints/openapi"
	"k8s.io/apiserver/pkg/features"
	genericapiserver "k8s.io/apiserver/pkg/server"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	openapicommon "k8s.io/kube-openapi/pkg/common"

	"sigs.k8s.io/custom-metrics-apiserver/pkg/apiserver"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/cmd/options"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/dynamicmapper"
	generatedcore "sigs.k8s.io/custom-metrics-apiserver/pkg/generated/openapi/core"
	generatedcustommetrics "sigs.k8s.io/custom-metrics-apiserver/pkg/generated/openapi/custommetrics"
	generatedexternalmetrics "sigs.k8s.io/custom-metrics-apiserver/pkg/generated/openapi/externalmetrics"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"
)

// AdapterBase provides a base set of functionality for any custom metrics adapter.
// Embed it in a struct containing your options, then:
//
// - Use Flags() to add flags, then call Flags().Parse(os.Argv)
// - Use DynamicClient and RESTMapper to fetch handles to common utilities
// - Use WithCustomMetrics(provider) and WithExternalMetrics(provider) to install metrics providers
// - Use Run(stopChannel) to start the server
//
// All methods on this struct are idempotent except for Run -- they'll perform any
// initialization on the first call, then return the existing object on later calls.
// Methods on this struct are not safe to call from multiple goroutines without
// external synchronization.
type AdapterBase struct {
	*options.CustomMetricsAdapterServerOptions

	// Name is the name of the API server.  It defaults to custom-metrics-adapter
	Name string

	// RemoteKubeConfigFile specifies the kubeconfig to use to construct
	// the dynamic client and RESTMapper.  It's set from a flag.
	RemoteKubeConfigFile string
	// DiscoveryInterval specifies the interval at which to recheck discovery
	// information for the discovery RESTMapper.  It's set from a flag.
	DiscoveryInterval time.Duration
	// ClientQPS specifies the maximum QPS for the client-side throttle. It's set from a flag.
	ClientQPS float32
	// ClientBurst specifies the maximum QPS burst for client-side throttle. It's set from a flag.
	ClientBurst int

	// FlagSet is the flagset to add flags to.
	// It defaults to the normal CommandLine flags
	// if not explicitly set.
	FlagSet *pflag.FlagSet

	// OpenAPIConfig
	OpenAPIConfig *openapicommon.Config

	// flagOnce controls initialization of the flags.
	flagOnce sync.Once

	clientConfig    *rest.Config
	discoveryClient discovery.DiscoveryInterface
	restMapper      apimeta.RESTMapper
	dynamicClient   dynamic.Interface
	informers       informers.SharedInformerFactory

	config *apiserver.Config
	server *apiserver.CustomMetricsAdapterServer

	cmProvider provider.CustomMetricsProvider
	emProvider provider.ExternalMetricsProvider
}

// InstallFlags installs the minimum required set of flags into the flagset.
func (b *AdapterBase) InstallFlags() {
	b.initFlagSet()
	b.flagOnce.Do(func() {
		if b.CustomMetricsAdapterServerOptions == nil {
			b.CustomMetricsAdapterServerOptions = options.NewCustomMetricsAdapterServerOptions()
		}

		b.CustomMetricsAdapterServerOptions.AddFlags(b.FlagSet)

		b.FlagSet.StringVar(&b.RemoteKubeConfigFile, "lister-kubeconfig", b.RemoteKubeConfigFile,
			"kubeconfig file pointing at the 'core' kubernetes server with enough rights to list "+
				"any described objects")
		b.FlagSet.DurationVar(&b.DiscoveryInterval, "discovery-interval", b.DiscoveryInterval,
			"Interval at which to refresh API discovery information")
		b.FlagSet.Float32Var(&b.ClientQPS, "client-qps", rest.DefaultQPS, "Maximum QPS for client-side throttle")
		b.FlagSet.IntVar(&b.ClientBurst, "client-burst", rest.DefaultBurst, "Maximum QPS burst for client-side throttle")
	})
}

// initFlagSet populates the flagset to the CommandLine flags if it's not already set.
func (b *AdapterBase) initFlagSet() {
	if b.FlagSet == nil {
		// default to the normal commandline flags
		b.FlagSet = pflag.CommandLine
	}
}

// Flags returns the flagset used by this adapter.
// It will initialize the flagset with the minimum required set
// of flags as well.
func (b *AdapterBase) Flags() *pflag.FlagSet {
	b.initFlagSet()
	b.InstallFlags()

	return b.FlagSet
}

// ClientConfig returns the REST client configuration used to construct
// clients for the clients and RESTMapper, and may be used for other
// purposes as well.  If you need to mutate it, be sure to copy it with
// rest.CopyConfig first.
func (b *AdapterBase) ClientConfig() (*rest.Config, error) {
	if b.clientConfig == nil {
		var clientConfig *rest.Config
		var err error
		if len(b.RemoteKubeConfigFile) > 0 {
			loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: b.RemoteKubeConfigFile}
			loader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})

			clientConfig, err = loader.ClientConfig()
		} else {
			clientConfig, err = rest.InClusterConfig()
		}
		if err != nil {
			return nil, fmt.Errorf("unable to construct lister client config to initialize provider: %v", err)
		}
		b.clientConfig = clientConfig
	}

	if b.ClientQPS > 0 {
		b.clientConfig.QPS = b.ClientQPS
	}
	if b.ClientBurst > 0 {
		b.clientConfig.Burst = b.ClientBurst
	}
	return b.clientConfig, nil
}

// DiscoveryClient returns a DiscoveryInterface suitable to for discovering resources
// available on the cluster.
func (b *AdapterBase) DiscoveryClient() (discovery.DiscoveryInterface, error) {
	if b.discoveryClient == nil {
		clientConfig, err := b.ClientConfig()
		if err != nil {
			return nil, err
		}
		discoveryClient, err := discovery.NewDiscoveryClientForConfig(clientConfig)
		if err != nil {
			return nil, fmt.Errorf("unable to construct discovery client for dynamic client: %v", err)
		}
		b.discoveryClient = discoveryClient
	}
	return b.discoveryClient, nil
}

// RESTMapper returns a RESTMapper dynamically populated with discovery information.
// The discovery information will be periodically repopulated according to DiscoveryInterval.
func (b *AdapterBase) RESTMapper() (apimeta.RESTMapper, error) {
	if b.restMapper == nil {
		discoveryClient, err := b.DiscoveryClient()
		if err != nil {
			return nil, err
		}
		// NB: since we never actually look at the contents of
		// the objects we fetch (beyond ObjectMeta), unstructured should be fine
		dynamicMapper, err := dynamicmapper.NewRESTMapper(discoveryClient, b.DiscoveryInterval)
		if err != nil {
			return nil, fmt.Errorf("unable to construct dynamic discovery mapper: %v", err)
		}

		b.restMapper = dynamicMapper
	}
	return b.restMapper, nil
}

// DynamicClient returns a dynamic Kubernetes client capable of listing and fetching
// any resources on the cluster.
func (b *AdapterBase) DynamicClient() (dynamic.Interface, error) {
	if b.dynamicClient == nil {
		clientConfig, err := b.ClientConfig()
		if err != nil {
			return nil, err
		}
		dynClient, err := dynamic.NewForConfig(clientConfig)
		if err != nil {
			return nil, fmt.Errorf("unable to construct lister client to initialize provider: %v", err)
		}
		b.dynamicClient = dynClient
	}
	return b.dynamicClient, nil
}

// WithCustomMetrics populates the custom metrics provider for this adapter.
func (b *AdapterBase) WithCustomMetrics(p provider.CustomMetricsProvider) {
	b.cmProvider = p
}

// WithExternalMetrics populates the external metrics provider for this adapter.
func (b *AdapterBase) WithExternalMetrics(p provider.ExternalMetricsProvider) {
	b.emProvider = p
}

func mergeOpenAPIDefinitions(definitionsGetters []openapicommon.GetOpenAPIDefinitions) openapicommon.GetOpenAPIDefinitions {
	return func(ref openapicommon.ReferenceCallback) map[string]openapicommon.OpenAPIDefinition {
		defsMap := make(map[string]openapicommon.OpenAPIDefinition)
		for _, definitionsGetter := range definitionsGetters {
			definitions := definitionsGetter(ref)
			for k, v := range definitions {
				defsMap[k] = v
			}
		}
		return defsMap
	}
}

func (b *AdapterBase) openAPIConfig(createConfig func(getDefinitions openapicommon.GetOpenAPIDefinitions, defNamer *openapinamer.DefinitionNamer) *openapicommon.Config) *openapicommon.Config {
	definitionsGetters := []openapicommon.GetOpenAPIDefinitions{generatedcore.GetOpenAPIDefinitions}
	if b.cmProvider != nil {
		definitionsGetters = append(definitionsGetters, generatedcustommetrics.GetOpenAPIDefinitions)
	}
	if b.emProvider != nil {
		definitionsGetters = append(definitionsGetters, generatedexternalmetrics.GetOpenAPIDefinitions)
	}
	getAPIDefinitions := mergeOpenAPIDefinitions(definitionsGetters)
	openAPIConfig := createConfig(getAPIDefinitions, openapinamer.NewDefinitionNamer(apiserver.Scheme))
	openAPIConfig.Info.Title = b.Name
	openAPIConfig.Info.Version = "1.0.0"
	return openAPIConfig
}

func (b *AdapterBase) defaultOpenAPIConfig() *openapicommon.Config {
	return b.openAPIConfig(genericapiserver.DefaultOpenAPIConfig)
}

func (b *AdapterBase) defaultOpenAPIV3Config() *openapicommon.Config {
	return b.openAPIConfig(genericapiserver.DefaultOpenAPIV3Config)
}

// Config fetches the configuration used to ultimately create the custom metrics adapter's
// API server.  While this method is idempotent, it does "cement" values of some of the other
// fields, so make sure to only call it just before `Server` or `Run`.
// Normal users should not need to call this method -- it's for advanced use cases.
func (b *AdapterBase) Config() (*apiserver.Config, error) {
	if b.config == nil {
		b.InstallFlags() // just to be sure

		if b.Name == "" {
			b.Name = "custom-metrics-adapter"
		}

		if b.OpenAPIConfig == nil {
			b.OpenAPIConfig = b.defaultOpenAPIConfig()
		}
		b.CustomMetricsAdapterServerOptions.OpenAPIConfig = b.OpenAPIConfig
		if b.OpenAPIV3Config == nil && utilfeature.DefaultFeatureGate.Enabled(features.OpenAPIV3) {
			b.OpenAPIV3Config = b.defaultOpenAPIV3Config()
		}

		if errList := b.CustomMetricsAdapterServerOptions.Validate(); len(errList) > 0 {
			return nil, utilerrors.NewAggregate(errList)
		}

		serverConfig := genericapiserver.NewConfig(apiserver.Codecs)
		err := b.CustomMetricsAdapterServerOptions.ApplyTo(serverConfig)
		if err != nil {
			return nil, err
		}
		b.config = &apiserver.Config{
			GenericConfig: serverConfig,
		}
	}

	return b.config, nil
}

// Server fetches API server object used to ultimately run the custom metrics adapter.
// While this method is idempotent, it does "cement" values of some of the other
// fields, so make sure to only call it just before `Run`.
// Normal users should not need to call this method -- it's for advanced use cases.
func (b *AdapterBase) Server() (*apiserver.CustomMetricsAdapterServer, error) {
	if b.server == nil {
		config, err := b.Config()
		if err != nil {
			return nil, err
		}

		// we add in the informers if they're not nil, but we don't try and
		// construct them if the user didn't ask for them
		server, err := config.Complete(b.informers).New(b.Name, b.cmProvider, b.emProvider)
		if err != nil {
			return nil, err
		}
		b.server = server
	}

	return b.server, nil
}

// Informers returns a SharedInformerFactory for constructing new informers.
// The informers will be automatically started as part of starting the adapter.
func (b *AdapterBase) Informers() (informers.SharedInformerFactory, error) {
	if b.informers == nil {
		clientConfig, err := b.ClientConfig()
		if err != nil {
			return nil, err
		}
		kubeClient, err := kubernetes.NewForConfig(clientConfig)
		if err != nil {
			return nil, err
		}
		b.informers = informers.NewSharedInformerFactory(kubeClient, 0)
	}

	return b.informers, nil
}

// Run runs this custom metrics adapter until the given stop channel is closed.
func (b *AdapterBase) Run(stopCh <-chan struct{}) error {
	server, err := b.Server()
	if err != nil {
		return err
	}

	return server.GenericAPIServer.PrepareRun().Run(stopCh)
}
