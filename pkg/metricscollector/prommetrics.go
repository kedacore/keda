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

package metricscollector

import (
	"runtime"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/kedacore/keda/v2/version"
)

var log = logf.Log.WithName("prometheus_server")

var (
	metricLabels = []string{"namespace", "metric", "scaledObject", "scaler", "scalerIndex"}
	buildInfo    = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: DefaultPromMetricsNamespace,
			Name:      "build_info",
			Help:      "A metric with a constant '1' value labeled by version, git_commit and goversion from which KEDA was built.",
		},
		[]string{"version", "git_commit", "goversion", "goos", "goarch"},
	)
	scalerErrorsTotalDeprecated = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "scaler",
			Name:      "errors_total",
			Help:      "DEPRECATED - use a `sum(scaler_errors_total{scaler!=\"\"})` over all scalers",
		},
		[]string{},
	)
	scalerMetricsValue = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "scaler",
			Name:      "metrics_value",
			Help:      "Current value of the metric obtained from the scaler that the Horizontal Pod Autoscaler (HPA) uses to make scaling decisions.",
		},
		metricLabels,
	)
	scalerMetricsLatencyDeprecated = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "scaler",
			Name:      "metrics_latency",
			Help:      "DEPRECATED - use 'scaler_metrics_latency_seconds' instead.",
		},
		metricLabels,
	)
	scalerMetricsLatency = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "scaler",
			Name:      "metrics_latency_seconds",
			Help:      "Latency observed by a scaler in getting the metric from the source, in seconds.",
		},
		metricLabels,
	)
	scalerActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "scaler",
			Name:      "active",
			Help:      "Indicates whether a scaler is active (1), or not (0).",
		},
		metricLabels,
	)
	scaledObjectPaused = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "scaled_object",
			Name:      "paused",
			Help:      "Indicates whether a ScaledObject is paused (1), or not (0).",
		},
		[]string{"namespace", "scaledObject"},
	)
	scalerErrorsDeprecated = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "scaler",
			Name:      "errors",
			Help:      "DEPRECATED - use 'scaler_errors_total' instead.",
		},
		metricLabels,
	)
	scalerErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "scaler",
			Name:      "errors_total",
			Help:      "Total number of errors observed by a scaler.",
		},
		metricLabels,
	)
	scaledObjectErrorsDeprecated = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "scaled_object",
			Name:      "errors",
			Help:      "DEPRECATED - use 'scaled_object_errors_total' instead.",
		},
		[]string{"namespace", "scaledObject"},
	)
	scaledObjectErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "scaled_object",
			Name:      "errors_total",
			Help:      "Total number of errors observed by a scaled object.",
		},
		[]string{"namespace", "scaledObject"},
	)
	triggerTotalsGaugeVecDeprecated = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "trigger",
			Name:      "totals",
			Help:      "DEPRECATED - use 'trigger_handled_total' instead.",
		},
		[]string{"type"},
	)
	triggerHandled = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "trigger",
			Name:      "handled_total",
			Help:      "Total number of triggers currently handled.",
		},
		[]string{"type"},
	)
	crdTotalsGaugeVecDeprecated = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "resource",
			Name:      "totals",
			Help:      "DEPRECATED - use 'resource_handled_total' instead.",
		},
		[]string{"type", "namespace"},
	)
	crdHandled = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "resource",
			Name:      "handled_total",
			Help:      "Total number of ScaledObjects/ScaledJobs currently handled.",
		},
		[]string{"type", "namespace"},
	)
	internalLoopLatencyDeprecated = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "internal_scale_loop",
			Name:      "latency",
			Help:      "DEPRECATED - use 'internal_scale_loop_latency_seconds' instead.",
		},
		[]string{"namespace", "type", "resource"},
	)
	internalLoopLatency = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "internal_scale_loop",
			Name:      "latency_seconds",
			Help:      "Internal latency of ScaledObject/ScaledJob loop execution in seconds.",
		},
		[]string{"namespace", "type", "resource"},
	)
)

type PromMetrics struct {
}

func NewPromMetrics() *PromMetrics {
	metrics.Registry.MustRegister(scalerErrorsTotalDeprecated)
	metrics.Registry.MustRegister(scalerMetricsValue)
	metrics.Registry.MustRegister(scalerMetricsLatencyDeprecated)
	metrics.Registry.MustRegister(scalerMetricsLatency)
	metrics.Registry.MustRegister(internalLoopLatencyDeprecated)
	metrics.Registry.MustRegister(internalLoopLatency)
	metrics.Registry.MustRegister(scalerActive)
	metrics.Registry.MustRegister(scalerErrorsDeprecated)
	metrics.Registry.MustRegister(scalerErrors)
	metrics.Registry.MustRegister(scaledObjectErrorsDeprecated)
	metrics.Registry.MustRegister(scaledObjectErrors)
	metrics.Registry.MustRegister(scaledObjectPaused)
	metrics.Registry.MustRegister(triggerTotalsGaugeVecDeprecated)
	metrics.Registry.MustRegister(triggerHandled)
	metrics.Registry.MustRegister(crdTotalsGaugeVecDeprecated)
	metrics.Registry.MustRegister(crdHandled)
	metrics.Registry.MustRegister(buildInfo)

	RecordBuildInfo()
	return &PromMetrics{}
}

// RecordBuildInfo publishes information about KEDA version and runtime info through an info metric (gauge).
func RecordBuildInfo() {
	buildInfo.WithLabelValues(version.Version, version.GitCommit, runtime.Version(), runtime.GOOS, runtime.GOARCH).Set(1)
}

// RecordScalerMetric create a measurement of the external metric used by the HPA
func (p *PromMetrics) RecordScalerMetric(namespace string, scaledObject string, scaler string, scalerIndex int, metric string, value float64) {
	scalerMetricsValue.With(getLabels(namespace, scaledObject, scaler, scalerIndex, metric)).Set(value)
}

// RecordScalerLatency create a measurement of the latency to external metric
func (p *PromMetrics) RecordScalerLatency(namespace string, scaledObject string, scaler string, scalerIndex int, metric string, value time.Duration) {
	scalerMetricsLatency.With(getLabels(namespace, scaledObject, scaler, scalerIndex, metric)).Set(value.Seconds())
	scalerMetricsLatencyDeprecated.With(getLabels(namespace, scaledObject, scaler, scalerIndex, metric)).Set(float64(value.Milliseconds()))
}

// RecordScalableObjectLatency create a measurement of the latency executing scalable object loop
func (p *PromMetrics) RecordScalableObjectLatency(namespace string, name string, isScaledObject bool, value time.Duration) {
	resourceType := "scaledjob"
	if isScaledObject {
		resourceType = "scaledobject"
	}
	internalLoopLatency.WithLabelValues(namespace, resourceType, name).Set(value.Seconds())
	internalLoopLatencyDeprecated.WithLabelValues(namespace, resourceType, name).Set(float64(value.Milliseconds()))
}

// RecordScalerActive create a measurement of the activity of the scaler
func (p *PromMetrics) RecordScalerActive(namespace string, scaledObject string, scaler string, scalerIndex int, metric string, active bool) {
	activeVal := 0
	if active {
		activeVal = 1
	}

	scalerActive.With(getLabels(namespace, scaledObject, scaler, scalerIndex, metric)).Set(float64(activeVal))
}

// RecordScaledObjectPaused marks whether the current ScaledObject is paused.
func (p *PromMetrics) RecordScaledObjectPaused(namespace string, scaledObject string, active bool) {
	labels := prometheus.Labels{"namespace": namespace, "scaledObject": scaledObject}

	activeVal := 0
	if active {
		activeVal = 1
	}

	scaledObjectPaused.With(labels).Set(float64(activeVal))
}

// RecordScalerError counts the number of errors occurred in trying get an external metric used by the HPA
func (p *PromMetrics) RecordScalerError(namespace string, scaledObject string, scaler string, scalerIndex int, metric string, err error) {
	if err != nil {
		scalerErrors.With(getLabels(namespace, scaledObject, scaler, scalerIndex, metric)).Inc()
		scalerErrorsDeprecated.With(getLabels(namespace, scaledObject, scaler, scalerIndex, metric)).Inc()
		p.RecordScaledObjectError(namespace, scaledObject, err)
		scalerErrorsTotalDeprecated.With(prometheus.Labels{}).Inc()
		return
	}
	// initialize metric with 0 if not already set
	_, errscaler := scalerErrors.GetMetricWith(getLabels(namespace, scaledObject, scaler, scalerIndex, metric))
	if errscaler != nil {
		log.Error(errscaler, "Unable to record metrics: %v")
	}
	_, errscalerdep := scalerErrorsDeprecated.GetMetricWith(getLabels(namespace, scaledObject, scaler, scalerIndex, metric))
	if errscalerdep != nil {
		log.Error(errscaler, "Unable to record (deprecated) metrics: %v")
	}
}

// RecordScaledObjectError counts the number of errors with the scaled object
func (p *PromMetrics) RecordScaledObjectError(namespace string, scaledObject string, err error) {
	labels := prometheus.Labels{"namespace": namespace, "scaledObject": scaledObject}
	if err != nil {
		scaledObjectErrors.With(labels).Inc()
		scaledObjectErrorsDeprecated.With(labels).Inc()
		return
	}
	// initialize metric with 0 if not already set
	_, errscaledobject := scaledObjectErrors.GetMetricWith(labels)
	if errscaledobject != nil {
		log.Error(errscaledobject, "Unable to record metrics: %v")
		return
	}
	_, errscaledobjectdep := scaledObjectErrorsDeprecated.GetMetricWith(labels)
	if errscaledobjectdep != nil {
		log.Error(errscaledobject, "Unable to record metrics: %v")
		return
	}
}

func getLabels(namespace string, scaledObject string, scaler string, scalerIndex int, metric string) prometheus.Labels {
	return prometheus.Labels{"namespace": namespace, "scaledObject": scaledObject, "scaler": scaler, "scalerIndex": strconv.Itoa(scalerIndex), "metric": metric}
}

func (p *PromMetrics) IncrementTriggerTotal(triggerType string) {
	if triggerType != "" {
		triggerHandled.WithLabelValues(triggerType).Inc()
		triggerTotalsGaugeVecDeprecated.WithLabelValues(triggerType).Inc()
	}
}

func (p *PromMetrics) DecrementTriggerTotal(triggerType string) {
	if triggerType != "" {
		triggerHandled.WithLabelValues(triggerType).Dec()
		triggerTotalsGaugeVecDeprecated.WithLabelValues(triggerType).Dec()
	}
}

func (p *PromMetrics) IncrementCRDTotal(crdType, namespace string) {
	if namespace == "" {
		namespace = defaultNamespace
	}

	crdHandled.WithLabelValues(crdType, namespace).Inc()
	crdTotalsGaugeVecDeprecated.WithLabelValues(crdType, namespace).Inc()
}

func (p *PromMetrics) DecrementCRDTotal(crdType, namespace string) {
	if namespace == "" {
		namespace = defaultNamespace
	}

	crdHandled.WithLabelValues(crdType, namespace).Dec()
	crdTotalsGaugeVecDeprecated.WithLabelValues(crdType, namespace).Dec()
}
