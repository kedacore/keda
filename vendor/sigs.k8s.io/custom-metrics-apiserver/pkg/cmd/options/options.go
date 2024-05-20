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

// Package options provides configuration options for the metrics API server.
package options

import (
	"fmt"
	"net"

	"github.com/spf13/pflag"

	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	"k8s.io/client-go/kubernetes"
	openapicommon "k8s.io/kube-openapi/pkg/common"
)

// CustomMetricsAdapterServerOptions contains the of options used to configure
// the metrics API server.
//
// It is based on a subset of [genericoptions.RecommendedOptions].
type CustomMetricsAdapterServerOptions struct {
	SecureServing  *genericoptions.SecureServingOptionsWithLoopback
	Authentication *genericoptions.DelegatingAuthenticationOptions
	Authorization  *genericoptions.DelegatingAuthorizationOptions
	Audit          *genericoptions.AuditOptions
	Features       *genericoptions.FeatureOptions

	OpenAPIConfig   *openapicommon.Config
	OpenAPIV3Config *openapicommon.OpenAPIV3Config
	EnableMetrics   bool
}

// NewCustomMetricsAdapterServerOptions creates a new instance of
// CustomMetricsAdapterServerOptions with its default values.
func NewCustomMetricsAdapterServerOptions() *CustomMetricsAdapterServerOptions {
	o := &CustomMetricsAdapterServerOptions{
		SecureServing:  genericoptions.NewSecureServingOptions().WithLoopback(),
		Authentication: genericoptions.NewDelegatingAuthenticationOptions(),
		Authorization:  genericoptions.NewDelegatingAuthorizationOptions(),
		Audit:          genericoptions.NewAuditOptions(),
		Features:       genericoptions.NewFeatureOptions(),

		EnableMetrics: true,
	}

	// Explicitly disable Priority and Fairness since metric servers are not
	// meant to be queried directly by default.
	o.Features.EnablePriorityAndFairness = false

	return o
}

// Validate validates CustomMetricsAdapterServerOptions
func (o CustomMetricsAdapterServerOptions) Validate() []error {
	errors := []error{}
	errors = append(errors, o.SecureServing.Validate()...)
	errors = append(errors, o.Authentication.Validate()...)
	errors = append(errors, o.Authorization.Validate()...)
	errors = append(errors, o.Audit.Validate()...)
	errors = append(errors, o.Features.Validate()...)
	return errors
}

// AddFlags adds the flags defined for the options, to the given flagset.
func (o *CustomMetricsAdapterServerOptions) AddFlags(fs *pflag.FlagSet) {
	o.SecureServing.AddFlags(fs)
	o.Authentication.AddFlags(fs)
	o.Authorization.AddFlags(fs)
	o.Audit.AddFlags(fs)
	o.Features.AddFlags(fs)
}

// ApplyTo applies CustomMetricsAdapterServerOptions to the server configuration.
func (o *CustomMetricsAdapterServerOptions) ApplyTo(serverConfig *genericapiserver.RecommendedConfig) error {
	// TODO have a "real" external address (have an AdvertiseAddress?)
	if err := o.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", nil, []net.IP{net.ParseIP("127.0.0.1")}); err != nil {
		return fmt.Errorf("error creating self-signed certificates: %v", err)
	}

	if err := o.SecureServing.ApplyTo(&serverConfig.SecureServing, &serverConfig.LoopbackClientConfig); err != nil {
		return err
	}
	if err := o.Authentication.ApplyTo(&serverConfig.Authentication, serverConfig.SecureServing, nil); err != nil {
		return err
	}
	if err := o.Authorization.ApplyTo(&serverConfig.Authorization); err != nil {
		return err
	}
	if err := o.Audit.ApplyTo(&serverConfig.Config); err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(serverConfig.ClientConfig)
	if err != nil {
		return err
	}

	if err := o.Features.ApplyTo(&serverConfig.Config, clientset, serverConfig.SharedInformerFactory); err != nil {
		return err
	}

	// enable OpenAPI schemas
	if o.OpenAPIConfig != nil {
		serverConfig.OpenAPIConfig = o.OpenAPIConfig
	}
	if o.OpenAPIV3Config != nil {
		serverConfig.OpenAPIV3Config = o.OpenAPIV3Config
	}

	serverConfig.EnableMetrics = o.EnableMetrics

	return nil
}
