/*
Copyright 2018 The Knative Authors

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

package autoscaler

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/knative/serving/pkg/apis/serving/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

const (
	// ConfigName is the name of the config map of the autoscaler.
	ConfigName = "config-autoscaler"

	// Default values for several key autoscaler settings.
	DefaultStableWindow           = 60 * time.Second
	DefaultScaleToZeroGracePeriod = 30 * time.Second
)

// Config defines the tunable autoscaler parameters
// +k8s:deepcopy-gen=true
type Config struct {
	// Feature flags.
	EnableScaleToZero bool

	// Target concurrency knobs for different container concurrency configurations.
	ContainerConcurrencyTargetPercentage float64
	ContainerConcurrencyTargetDefault    float64

	// General autoscaler algorithm configuration.
	MaxScaleUpRate float64
	StableWindow   time.Duration
	PanicWindow    time.Duration
	TickInterval   time.Duration

	ScaleToZeroGracePeriod time.Duration
}

// TargetConcurrency calculates the target concurrency for a given container-concurrency
// taking the container-concurrency-target-percentage into account.
func (c *Config) TargetConcurrency(concurrency v1alpha1.RevisionContainerConcurrencyType) float64 {
	if concurrency == 0 {
		return c.ContainerConcurrencyTargetDefault
	}
	return float64(concurrency) * c.ContainerConcurrencyTargetPercentage
}

// NewConfigFromMap creates a Config from the supplied map
func NewConfigFromMap(data map[string]string) (*Config, error) {
	lc := &Config{}

	// Process bool fields
	for _, b := range []struct {
		key          string
		field        *bool
		defaultValue bool
	}{{
		key:          "enable-scale-to-zero",
		field:        &lc.EnableScaleToZero,
		defaultValue: true,
	}} {
		if raw, ok := data[b.key]; !ok {
			*b.field = b.defaultValue
		} else {
			*b.field = strings.ToLower(raw) == "true"
		}
	}

	// Process Float64 fields
	for _, f64 := range []struct {
		key   string
		field *float64
		// specified exactly when optional
		defaultValue float64
	}{{
		key:          "max-scale-up-rate",
		field:        &lc.MaxScaleUpRate,
		defaultValue: 10.0,
	}, {
		key:   "container-concurrency-target-percentage",
		field: &lc.ContainerConcurrencyTargetPercentage,
		// TODO(#1956): tune target usage based on empirical data.
		// TODO(#2016): Revert to 0.7 once incorrect reporting is solved
		defaultValue: 1.0,
	}, {
		key:          "container-concurrency-target-default",
		field:        &lc.ContainerConcurrencyTargetDefault,
		defaultValue: 100.0,
	}} {
		if raw, ok := data[f64.key]; !ok {
			*f64.field = f64.defaultValue
		} else if val, err := strconv.ParseFloat(raw, 64); err != nil {
			return nil, err
		} else {
			*f64.field = val
		}
	}

	// Process Duration fields
	for _, dur := range []struct {
		key          string
		field        *time.Duration
		defaultValue time.Duration
	}{{
		key:          "stable-window",
		field:        &lc.StableWindow,
		defaultValue: DefaultStableWindow,
	}, {
		key:          "panic-window",
		field:        &lc.PanicWindow,
		defaultValue: 6 * time.Second,
	}, {
		key:          "scale-to-zero-grace-period",
		field:        &lc.ScaleToZeroGracePeriod,
		defaultValue: DefaultScaleToZeroGracePeriod,
	}, {
		key:          "tick-interval",
		field:        &lc.TickInterval,
		defaultValue: 2 * time.Second,
	}} {
		if raw, ok := data[dur.key]; !ok {
			*dur.field = dur.defaultValue
		} else if val, err := time.ParseDuration(raw); err != nil {
			return nil, err
		} else {
			*dur.field = val
		}
	}

	if lc.ScaleToZeroGracePeriod < 30*time.Second {
		return nil, fmt.Errorf("scale-to-zero-grace-period must be at least 30s, got %v", lc.ScaleToZeroGracePeriod)
	}

	return lc, nil
}

// NewConfigFromConfigMap creates a Config from the supplied ConfigMap
func NewConfigFromConfigMap(configMap *corev1.ConfigMap) (*Config, error) {
	return NewConfigFromMap(configMap.Data)
}
