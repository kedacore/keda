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
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

const (
	ObservabilityConfigName = "config-observability"

	defaultLogURLTemplate = "http://localhost:8001/api/v1/namespaces/knative-monitoring/services/kibana-logging/proxy/app/kibana#/discover?_a=(query:(match:(kubernetes.labels.knative-dev%2FrevisionUID:(query:'${REVISION_UID}',type:phrase))))"
)

// Observability contains the configuration defined in the observability ConfigMap.
type Observability struct {
	// EnableVarLogCollection dedicates whether to set up a fluentd sidecar to
	// collect logs under /var/log/.
	EnableVarLogCollection bool

	// TODO(#818): Use the fluentd daemon set to collect /var/log.
	// FluentdSidecarImage is the name of the image used for the fluentd sidecar
	// injected into the revision pod. It is used only when enableVarLogCollection
	// is true.
	FluentdSidecarImage string

	// FluentdSidecarOutputConfig is the config for fluentd sidecar to specify
	// logging output destination.
	FluentdSidecarOutputConfig string

	// LoggingURLTemplate is a string containing the logging url template where
	// the variable REVISION_UID will be replaced with the created revision's UID.
	LoggingURLTemplate string
}

// NewObservabilityFromConfigMap creates a Observability from the supplied ConfigMap
func NewObservabilityFromConfigMap(configMap *corev1.ConfigMap) (*Observability, error) {
	oc := &Observability{}
	if evlc, ok := configMap.Data["logging.enable-var-log-collection"]; ok {
		oc.EnableVarLogCollection = strings.ToLower(evlc) == "true"
	}
	if fsi, ok := configMap.Data["logging.fluentd-sidecar-image"]; ok {
		oc.FluentdSidecarImage = fsi
	} else if oc.EnableVarLogCollection {
		return nil, fmt.Errorf("Received bad Observability ConfigMap, want %q when %q is true",
			"logging.fluentd-sidecar-image", "logging.enable-var-log-collection")
	}

	if fsoc, ok := configMap.Data["logging.fluentd-sidecar-output-config"]; ok {
		oc.FluentdSidecarOutputConfig = fsoc
	}
	if rut, ok := configMap.Data["logging.revision-url-template"]; ok {
		oc.LoggingURLTemplate = rut
	} else {
		oc.LoggingURLTemplate = defaultLogURLTemplate
	}
	return oc, nil
}
