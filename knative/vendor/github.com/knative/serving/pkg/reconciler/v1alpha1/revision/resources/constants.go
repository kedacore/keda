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

package resources

import (
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	// UserContainerName is the name of the user-container in the PodSpec
	UserContainerName = "user-container"
	// FluentdContainerName is the name of the fluentd sidecar when enabled
	FluentdContainerName = "fluentd-proxy"
	// EnvoyContainerName is the name of the envoy sidecar when enabled
	EnvoyContainerName = "istio-proxy"
	// QueueContainerName is the name of the queue proxy side car
	QueueContainerName = "queue-proxy"

	sidecarIstioInjectAnnotation = "sidecar.istio.io/inject"
	// TODO(mattmoor): Make this private once we remove revision_test.go
	IstioOutboundIPRangeAnnotation = "traffic.sidecar.istio.io/includeOutboundIPRanges"

	userPortEnvName = "PORT"

	autoscalerPort = 8080

	// ServicePortName is the name of the external port of the service
	ServicePortNameHTTP1 = "http"
	ServicePortNameH2C   = "http2"

	// ServicePort is the external port of the service
	ServicePort = int32(80)
	// MetricsPortName is the name of the external port of the service for metrics
	MetricsPortName = "metrics"
	// MetricsPort is the external port of the service for metrics
	MetricsPort = int32(9090)
	AppLabelKey = "app"
)

var ProgressDeadlineSeconds int32 = 120

// pseudo-constants
var (
	// See https://github.com/knative/serving/pull/1124#issuecomment-397120430
	// for how CPU and memory values were calculated.

	// Each Knative Serving pod gets 500m cpu initially.
	userContainerCPU    = resource.MustParse("400m")
	fluentdContainerCPU = resource.MustParse("25m")
	envoyContainerCPU   = resource.MustParse("50m")
	queueContainerCPU   = resource.MustParse("25m")

	// Limit CPU recommendation to 2000m
	userContainerMaxCPU    = resource.MustParse("1700m")
	fluentdContainerMaxCPU = resource.MustParse("100m")
	envoyContainerMaxCPU   = resource.MustParse("200m")
	queueContainerMaxCPU   = resource.MustParse("200m")

	// Limit memory recommendation to 4G
	userContainerMaxMemory    = resource.MustParse("3700M")
	fluentdContainerMaxMemory = resource.MustParse("100M")
	envoyContainerMaxMemory   = resource.MustParse("100M")
	queueContainerMaxMemory   = resource.MustParse("100M")
)
