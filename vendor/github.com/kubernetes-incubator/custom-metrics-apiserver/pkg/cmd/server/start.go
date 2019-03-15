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

package server

import (
	"fmt"
	"net"

	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/apiserver"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
)

type CustomMetricsAdapterServerOptions struct {
	// genericoptions.ReccomendedOptions - EtcdOptions
	SecureServing  *genericoptions.SecureServingOptionsWithLoopback
	Authentication *genericoptions.DelegatingAuthenticationOptions
	Authorization  *genericoptions.DelegatingAuthorizationOptions
	Features       *genericoptions.FeatureOptions
}

func NewCustomMetricsAdapterServerOptions() *CustomMetricsAdapterServerOptions {
	o := &CustomMetricsAdapterServerOptions{
		SecureServing:  genericoptions.NewSecureServingOptions().WithLoopback(),
		Authentication: genericoptions.NewDelegatingAuthenticationOptions(),
		Authorization:  genericoptions.NewDelegatingAuthorizationOptions(),
		Features:       genericoptions.NewFeatureOptions(),
	}

	return o
}

func (o CustomMetricsAdapterServerOptions) Validate(args []string) error {
	return nil
}

func (o *CustomMetricsAdapterServerOptions) Complete() error {
	return nil
}

func (o CustomMetricsAdapterServerOptions) Config() (*apiserver.Config, error) {
	// TODO have a "real" external address (have an AdvertiseAddress?)
	if err := o.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", nil, []net.IP{net.ParseIP("127.0.0.1")}); err != nil {
		return nil, fmt.Errorf("error creating self-signed certificates: %v", err)
	}

	serverConfig := genericapiserver.NewConfig(apiserver.Codecs)
	if err := o.SecureServing.ApplyTo(&serverConfig.SecureServing, &serverConfig.LoopbackClientConfig); err != nil {
		return nil, err
	}

	if err := o.Authentication.ApplyTo(&serverConfig.Authentication, serverConfig.SecureServing, serverConfig.OpenAPIConfig); err != nil {
		return nil, err
	}
	if err := o.Authorization.ApplyTo(&serverConfig.Authorization); err != nil {
		return nil, err
	}

	// TODO: we can't currently serve swagger because we don't have a good way to dynamically update it
	// serverConfig.SwaggerConfig = genericapiserver.DefaultSwaggerConfig()

	config := &apiserver.Config{
		GenericConfig: serverConfig,
	}
	return config, nil
}
