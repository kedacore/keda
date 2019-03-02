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
	"errors"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	// ControllerConfigName is the name of config map for the controller.
	ControllerConfigName = "config-controller"

	queueSidecarImageKey           = "queueSidecarImage"
	registriesSkippingTagResolving = "registriesSkippingTagResolving"
)

// NewControllerConfigFromMap creates a Controller from the supplied Map
func NewControllerConfigFromMap(configMap map[string]string) (*Controller, error) {
	nc := &Controller{}

	if qsideCarImage, ok := configMap[queueSidecarImageKey]; !ok {
		return nil, errors.New("Queue sidecar image is missing")
	} else {
		nc.QueueSidecarImage = qsideCarImage
	}

	if registries, ok := configMap[registriesSkippingTagResolving]; !ok {
		// It is ok if registries are missing.
		nc.RegistriesSkippingTagResolving = sets.NewString("ko.local", "dev.local")
	} else {
		nc.RegistriesSkippingTagResolving = sets.NewString(strings.Split(registries, ",")...)
	}
	return nc, nil
}

// NewControllerConfigFromConfigMap creates a Controller from the supplied configMap
func NewControllerConfigFromConfigMap(config *corev1.ConfigMap) (*Controller, error) {
	return NewControllerConfigFromMap(config.Data)
}

// Controller includes the configurations for the controller.
type Controller struct {
	// QueueSidecarImage is the name of the image used for the queue sidecar
	// injected into the revision pod
	QueueSidecarImage string

	// Repositories for which tag to digest resolving should be skipped
	RegistriesSkippingTagResolving sets.String
}
