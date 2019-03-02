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
	"context"

	"github.com/knative/pkg/configmap"
	"github.com/knative/serving/pkg/gc"
	"github.com/knative/serving/pkg/network"
)

type cfgKey struct{}

// +k8s:deepcopy-gen=false
type Config struct {
	Domain  *Domain
	GC      *gc.Config
	Network *network.Config
}

func FromContext(ctx context.Context) *Config {
	return ctx.Value(cfgKey{}).(*Config)
}

func ToContext(ctx context.Context, c *Config) context.Context {
	return context.WithValue(ctx, cfgKey{}, c)
}

// Store is based on configmap.UntypedStore and is used to store and watch for
// updates to configuration related to routes (currently only config-domain).
//
// +k8s:deepcopy-gen=false
type Store struct {
	*configmap.UntypedStore
}

// NewStore creates a configmap.UntypedStore based config store.
//
// logger must be non-nil implementation of configmap.Logger (commonly used
// loggers conform)
//
// onAfterStore is a variadic list of callbacks to run
// after the ConfigMap has been processed and stored.
//
// See also: configmap.NewUntypedStore().
func NewStore(logger configmap.Logger, onAfterStore ...func(name string, value interface{})) *Store {
	store := &Store{
		UntypedStore: configmap.NewUntypedStore(
			"route",
			logger,
			configmap.Constructors{
				DomainConfigName:   NewDomainFromConfigMap,
				gc.ConfigName:      gc.NewConfigFromConfigMap,
				network.ConfigName: network.NewConfigFromConfigMap,
			},
			onAfterStore...,
		),
	}

	return store
}

func (s *Store) ToContext(ctx context.Context) context.Context {
	return ToContext(ctx, s.Load())
}

func (s *Store) Load() *Config {
	return &Config{
		Domain:  s.UntypedLoad(DomainConfigName).(*Domain).DeepCopy(),
		GC:      s.UntypedLoad(gc.ConfigName).(*gc.Config).DeepCopy(),
		Network: s.UntypedLoad(network.ConfigName).(*network.Config).DeepCopy(),
	}
}
