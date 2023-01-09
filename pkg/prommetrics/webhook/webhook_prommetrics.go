/*
Copyright 2023 The KEDA Authors

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

package webhook

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	DefaultPromMetricsNamespace = "keda"
)

var (
	scaledObjectValidatingTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "webhook",
			Name:      "scaled_object_validation_total",
			Help:      "Total number of scaled object validations",
		},
		[]string{"namespace", "action"},
	)
	scaledObjectValidatingErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "webhook",
			Name:      "scaled_object_validation_errors",
			Help:      "Total number of scaled object validating errors",
		},
		[]string{"namespace", "action", "reason"},
	)
)

func init() {
	metrics.Registry.MustRegister(scaledObjectValidatingTotal)
	metrics.Registry.MustRegister(scaledObjectValidatingErrors)
}

// RecordScaledObjectValidatingTotal counts the number of ScaledObject validations
func RecordScaledObjectValidatingTotal(namespace, action string) {
	labels := prometheus.Labels{"namespace": namespace, "action": action}
	scaledObjectValidatingTotal.With(labels).Inc()
}

// RecordScaledObjectValidatingErrors counts the number of ScaledObject validating errors
func RecordScaledObjectValidatingErrors(namespace, action, reason string) {
	labels := prometheus.Labels{"namespace": namespace, "action": action, "reason": reason}
	scaledObjectValidatingErrors.With(labels).Inc()
}
