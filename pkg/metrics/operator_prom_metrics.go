/*
Copyright 2022 The KEDA Authors

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

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	ClusterTriggerAuthenticationResource = "cluster_trigger_authentication"
	TriggerAuthenticationResource        = "trigger_authentication"
	ScaledObjectResource                 = "scaled_object"
	ScaledJobResource                    = "scaled_job"
)

var (
	triggerTotalsGaugeVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "keda_operator",
			Subsystem: "trigger",
			Name:      "totals",
		},
		[]string{"type"},
	)

	crdTotalsGaugeVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "keda_operator",
			Subsystem: "resource",
			Name:      "totals",
		},
		[]string{"type", "namespace"},
	)
)

func init() {
	metrics.Registry.MustRegister(triggerTotalsGaugeVec)
	metrics.Registry.MustRegister(crdTotalsGaugeVec)
}

func IncrementTriggerTotal(triggerType string) {
	if triggerType != "" {
		triggerTotalsGaugeVec.WithLabelValues(triggerType).Inc()
	}
}

func DecrementTriggerTotal(triggerType string) {
	if triggerType != "" {
		triggerTotalsGaugeVec.WithLabelValues(triggerType).Dec()
	}
}

func IncrementCRDTotal(crdType, namespace string) {
	if namespace == "" {
		namespace = "default"
	}

	crdTotalsGaugeVec.WithLabelValues(crdType, namespace).Inc()
}

func DecrementCRDTotal(crdType, namespace string) {
	if namespace == "" {
		namespace = "default"
	}

	crdTotalsGaugeVec.WithLabelValues(crdType, namespace).Dec()
}
