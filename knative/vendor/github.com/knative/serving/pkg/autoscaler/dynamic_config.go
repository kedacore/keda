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
	"sync"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
)

type DynamicConfig struct {
	mutex  sync.RWMutex
	config interface{} // discourage direct access as this is not goroutine safe

	logger *zap.SugaredLogger
}

func NewDynamicConfig(config *Config, logger *zap.SugaredLogger) *DynamicConfig {
	return &DynamicConfig{
		logger: logger,
		config: config.DeepCopy(),
	}
}

func NewDynamicConfigFromMap(rawConfig map[string]string, logger *zap.SugaredLogger) (*DynamicConfig, error) {
	config, err := NewConfigFromMap(rawConfig)
	if err != nil {
		return nil, err
	}
	return NewDynamicConfig(config, logger), nil
}

func (dc *DynamicConfig) Current() *Config {
	dc.mutex.RLock()
	defer dc.mutex.RUnlock()
	return dc.config.(*Config).DeepCopy()
}

func (dc *DynamicConfig) Update(configMap *corev1.ConfigMap) {
	config, err := NewConfigFromConfigMap(configMap)
	if err != nil {
		dc.logger.Errorf("Error updating autoscaler config: %v", err)
		return
	}
	dc.mutex.Lock()
	defer dc.mutex.Unlock()
	dc.config = config
	dc.logger.Infof("Autoscaler configuration updated: %v", configMap)
}
