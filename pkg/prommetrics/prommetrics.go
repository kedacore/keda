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

package prommetrics

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var log = logf.Log.WithName("prometheus_server")

const (
	ClusterTriggerAuthenticationResource = "cluster_trigger_authentication"
	TriggerAuthenticationResource        = "trigger_authentication"
	ScaledObjectResource                 = "scaled_object"
	ScaledJobResource                    = "scaled_job"

	DefaultPromMetricsNamespace = "keda"
)

var (
	metricLabels      = []string{"namespace", "metric", "scaledObject", "scaler", "scalerIndex"}
	scalerErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "scaler",
			Name:      "errors_total",
			Help:      "Total number of errors for all scalers",
		},
		[]string{},
	)
	scalerMetricsValue = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "scaler",
			Name:      "metrics_value",
			Help:      "Metric Value used for HPA",
		},
		metricLabels,
	)
	scalerMetricsLatency = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "scaler",
			Name:      "metrics_latency",
			Help:      "Scaler Metrics Latency",
		},
		metricLabels,
	)
	scalerActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "scaler",
			Name:      "active",
			Help:      "Activity of a Scaler Metric",
		},
		metricLabels,
	)
	scalerErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "scaler",
			Name:      "errors",
			Help:      "Number of scaler errors",
		},
		metricLabels,
	)
	scaledObjectErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "scaled_object",
			Name:      "errors",
			Help:      "Number of scaled object errors",
		},
		[]string{"namespace", "scaledObject"},
	)

	triggerTotalsGaugeVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "trigger",
			Name:      "totals",
		},
		[]string{"type"},
	)

	crdTotalsGaugeVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "resource",
			Name:      "totals",
		},
		[]string{"type", "namespace"},
	)
)

func init() {
	metrics.Registry.MustRegister(scalerErrorsTotal)
	metrics.Registry.MustRegister(scalerMetricsValue)
	metrics.Registry.MustRegister(scalerMetricsLatency)
	metrics.Registry.MustRegister(scalerActive)
	metrics.Registry.MustRegister(scalerErrors)
	metrics.Registry.MustRegister(scaledObjectErrors)

	metrics.Registry.MustRegister(triggerTotalsGaugeVec)
	metrics.Registry.MustRegister(crdTotalsGaugeVec)
}

// RecordScalerMetric create a measurement of the external metric used by the HPA
func RecordScalerMetric(namespace string, scaledObject string, scaler string, scalerIndex int, metric string, value float64) {
	scalerMetricsValue.With(getLabels(namespace, scaledObject, scaler, scalerIndex, metric)).Set(value)
}

// RecordScalerLatency create a measurement of the latency to external metric
func RecordScalerLatency(namespace string, scaledObject string, scaler string, scalerIndex int, metric string, value float64) {
	scalerMetricsLatency.With(getLabels(namespace, scaledObject, scaler, scalerIndex, metric)).Set(value)
}

// RecordScalerActive create a measurement of the activity of the scaler
func RecordScalerActive(namespace string, scaledObject string, scaler string, scalerIndex int, metric string, active bool) {
	activeVal := 0
	if active {
		activeVal = 1
	}

	scalerActive.With(getLabels(namespace, scaledObject, scaler, scalerIndex, metric)).Set(float64(activeVal))
}

// RecordScalerError counts the number of errors occurred in trying get an external metric used by the HPA
func RecordScalerError(namespace string, scaledObject string, scaler string, scalerIndex int, metric string, err error) {
	if err != nil {
		scalerErrors.With(getLabels(namespace, scaledObject, scaler, scalerIndex, metric)).Inc()
		RecordScaledObjectError(namespace, scaledObject, err)
		scalerErrorsTotal.With(prometheus.Labels{}).Inc()
		return
	}
	// initialize metric with 0 if not already set
	_, errscaler := scalerErrors.GetMetricWith(getLabels(namespace, scaledObject, scaler, scalerIndex, metric))
	if errscaler != nil {
		log.Error(errscaler, "Unable to write to metrics to Prometheus Server: %v")
	}
}

// RecordScaleObjectError counts the number of errors with the scaled object
func RecordScaledObjectError(namespace string, scaledObject string, err error) {
	labels := prometheus.Labels{"namespace": namespace, "scaledObject": scaledObject}
	if err != nil {
		scaledObjectErrors.With(labels).Inc()
		return
	}
	// initialize metric with 0 if not already set
	_, errscaledobject := scaledObjectErrors.GetMetricWith(labels)
	if errscaledobject != nil {
		log.Error(errscaledobject, "Unable to write to metrics to Prometheus Server: %v")
		return
	}
}

func getLabels(namespace string, scaledObject string, scaler string, scalerIndex int, metric string) prometheus.Labels {
	return prometheus.Labels{"namespace": namespace, "scaledObject": scaledObject, "scaler": scaler, "scalerIndex": strconv.Itoa(scalerIndex), "metric": metric}
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
