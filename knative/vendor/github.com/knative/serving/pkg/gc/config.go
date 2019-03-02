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

package gc

import (
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
)

const (
	ConfigName = "config-gc"
)

type Config struct {
	// Delay duration after a revision create before considering it for GC
	StaleRevisionCreateDelay time.Duration
	// Timeout since a revision lastPinned before it should be GC'd
	// This must be longer than the controller resync period
	StaleRevisionTimeout time.Duration
	// Minimum number of generations of revisions to keep before considering for GC
	StaleRevisionMinimumGenerations int64
	// Minimum staleness duration before updating lastPinned
	StaleRevisionLastpinnedDebounce time.Duration
}

func NewConfigFromConfigMap(configMap *corev1.ConfigMap) (*Config, error) {
	c := Config{}

	for _, dur := range []struct {
		key          string
		field        *time.Duration
		defaultValue time.Duration
	}{{
		key:          "stale-revision-create-delay",
		field:        &c.StaleRevisionCreateDelay,
		defaultValue: 24 * time.Hour,
	}, {
		key:          "stale-revision-timeout",
		field:        &c.StaleRevisionTimeout,
		defaultValue: 15 * time.Hour,
	}, {
		key:          "stale-revision-lastpinned-debounce",
		field:        &c.StaleRevisionLastpinnedDebounce,
		defaultValue: 5 * time.Hour,
	}} {
		if raw, ok := configMap.Data[dur.key]; !ok {
			*dur.field = dur.defaultValue
		} else if val, err := time.ParseDuration(raw); err != nil {
			return nil, err
		} else {
			*dur.field = val
		}
	}

	if raw, ok := configMap.Data["stale-revision-minimum-generations"]; !ok {
		c.StaleRevisionMinimumGenerations = 1
	} else if val, err := strconv.ParseInt(raw, 10, 64); err != nil {
		return nil, err
	} else {
		c.StaleRevisionMinimumGenerations = val
	}

	return &c, nil
}
