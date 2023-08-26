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
	"runtime"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/kedacore/keda/v2/version"
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
	metricLabels = []string{"namespace", "metric", "scaledObject", "scaler", "scalerIndex", "type"}
	buildInfo    = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: DefaultPromMetricsNamespace,
			Name:      "build_info",
			Help:      "A metric with a constant '1' value labeled by version, git_commit and goversion from which KEDA was built.",
		},
		[]string{"version", "git_commit", "goversion", "goos", "goarch"},
	)
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
	scaledJobErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "scaled_job",
			Name:      "errors",
			Help:      "Number of scaled job errors",
		},
		[]string{"namespace", "scaledJob"},
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

	internalLoopLatency = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "internal_scale_loop",
			Name:      "latency",
			Help:      "Internal latency of ScaledObject/ScaledJob loop execution",
		},
		[]string{"namespace", "type", "resource"},
	)
)

func init() {
	metrics.Registry.MustRegister(scalerErrorsTotal)
	metrics.Registry.MustRegister(scalerMetricsValue)
	metrics.Registry.MustRegister(scalerMetricsLatency)
	metrics.Registry.MustRegister(internalLoopLatency)
	metrics.Registry.MustRegister(scalerActive)
	metrics.Registry.MustRegister(scalerErrors)
	metrics.Registry.MustRegister(scaledObjectErrors)
	metrics.Registry.MustRegister(scaledJobErrors)

	metrics.Registry.MustRegister(triggerTotalsGaugeVec)
	metrics.Registry.MustRegister(crdTotalsGaugeVec)
	metrics.Registry.MustRegister(buildInfo)

	RecordBuildInfo()
}

// RecordScalerMetric create a measurement of the external metric used by the HPA
func RecordScalerMetric(namespace string, scaledResource string, scaler string, scalerIndex int, metric string, isScaledObject bool, value float64) {
	scalerMetricsValue.With(getLabels(namespace, scaledResource, scaler, scalerIndex, metric, getResourceType(isScaledObject))).Set(value)
}

// RecordScalerLatency create a measurement of the latency to external metric
func RecordScalerLatency(namespace string, scaledResource string, scaler string, scalerIndex int, metric string, isScaledObject bool, value float64) {
	scalerMetricsLatency.With(getLabels(namespace, scaledResource, scaler, scalerIndex, metric, getResourceType(isScaledObject))).Set(value)
}

// RecordScaledObjectLatency create a measurement of the latency executing scalable object loop
func RecordScalableObjectLatency(namespace string, name string, isScaledObject bool, value float64) {
	internalLoopLatency.WithLabelValues(namespace, getResourceType(isScaledObject), name).Set(value)
}

// RecordScalerActive create a measurement of the activity of the scaler
func RecordScalerActive(namespace string, scaledResource string, scaler string, scalerIndex int, metric string, isScaledObject bool, active bool) {
	activeVal := 0
	if active {
		activeVal = 1
	}

	scalerActive.With(getLabels(namespace, scaledResource, scaler, scalerIndex, metric, getResourceType(isScaledObject))).Set(float64(activeVal))
}

// RecordScalerError counts the number of errors occurred in trying to get an external metric used by the HPA
func RecordScalerError(namespace string, scaledObject string, scaler string, scalerIndex int, metric string, isScaledObject bool, err error) {
	if err != nil {
		scalerErrors.With(getLabels(namespace, scaledObject, scaler, scalerIndex, metric, getResourceType(isScaledObject))).Inc()
		if isScaledObject {
			RecordScaledObjectError(namespace, scaledObject, err)
		} else {
			RecordScaledJobError(namespace, scaledObject, err)
		}
		scalerErrorsTotal.With(prometheus.Labels{}).Inc()
		return
	}
	// initialize metric with 0 if not already set
	_, errscaler := scalerErrors.GetMetricWith(getLabels(namespace, scaledObject, scaler, scalerIndex, metric, getResourceType(isScaledObject)))
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

// RecordScaleJobError counts the number of errors with the scaled job
func RecordScaledJobError(namespace string, scaledJob string, err error) {
	labels := prometheus.Labels{"namespace": namespace, "scaledJob": scaledJob}
	if err != nil {
		scaledObjectErrors.With(labels).Inc()
		return
	}
	// initialize metric with 0 if not already set
	_, errscaledjob := scaledJobErrors.GetMetricWith(labels)
	if errscaledjob != nil {
		log.Error(err, "Unable to write to metrics to Prometheus Server: %v")
		return
	}
}

// RecordBuildInfo publishes information about KEDA version and runtime info through an info metric (gauge).
func RecordBuildInfo() {
	buildInfo.WithLabelValues(version.Version, version.GitCommit, runtime.Version(), runtime.GOOS, runtime.GOARCH).Set(1)
}

func getLabels(namespace string, scaledObject string, scaler string, scalerIndex int, metric string, resourceType string) prometheus.Labels {
	return prometheus.Labels{"namespace": namespace, "scaledObject": scaledObject, "scaler": scaler, "scalerIndex": strconv.Itoa(scalerIndex), "metric": metric, "type": resourceType}
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

func getResourceType(isScaledObject bool) string {
	if isScaledObject {
		return "scaledobject"
	}
	return "scaledjob"
}
