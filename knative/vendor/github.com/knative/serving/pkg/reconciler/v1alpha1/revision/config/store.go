/*
Copyright 2018 The Knative Authors.

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

package config

import (
	"context"

	"github.com/knative/pkg/configmap"
	pkglogging "github.com/knative/pkg/logging"
	"github.com/knative/serving/pkg/autoscaler"
	"github.com/knative/serving/pkg/logging"
	"github.com/knative/serving/pkg/network"
)

type cfgKey struct{}

// +k8s:deepcopy-gen=false
type Config struct {
	Controller    *Controller
	Network       *network.Config
	Observability *Observability
	Logging       *pkglogging.Config
	Autoscaler    *autoscaler.Config
}

func FromContext(ctx context.Context) *Config {
	return ctx.Value(cfgKey{}).(*Config)
}

func ToContext(ctx context.Context, c *Config) context.Context {
	return context.WithValue(ctx, cfgKey{}, c)
}

// +k8s:deepcopy-gen=false
type Store struct {
	*configmap.UntypedStore
}

// NewStore creates a new store of Configs and optionally calls functions when ConfigMaps are updated for Revisions
func NewStore(logger configmap.Logger, onAfterStore ...func(name string, value interface{})) *Store {
	store := &Store{
		UntypedStore: configmap.NewUntypedStore(
			"revision",
			logger,
			configmap.Constructors{
				ControllerConfigName:    NewControllerConfigFromConfigMap,
				network.ConfigName:      network.NewConfigFromConfigMap,
				ObservabilityConfigName: NewObservabilityFromConfigMap,
				autoscaler.ConfigName:   autoscaler.NewConfigFromConfigMap,
				logging.ConfigName:      logging.NewConfigFromConfigMap,
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
		Controller:    s.UntypedLoad(ControllerConfigName).(*Controller).DeepCopy(),
		Network:       s.UntypedLoad(network.ConfigName).(*network.Config).DeepCopy(),
		Observability: s.UntypedLoad(ObservabilityConfigName).(*Observability).DeepCopy(),
		Logging:       s.UntypedLoad(logging.ConfigName).(*pkglogging.Config).DeepCopy(),
		Autoscaler:    s.UntypedLoad(autoscaler.ConfigName).(*autoscaler.Config).DeepCopy(),
	}
}
