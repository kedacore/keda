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

	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/kedacore/keda/v2/version"
)

var log = logf.Log.WithName("prometheus_server")

var (
	metricLabels = []string{"namespace", "metric", "scaledObject", "scaler", "triggerIndex", "type"}
	buildInfo    = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: DefaultPromMetricsNamespace,
			Name:      "build_info",
			Help:      "Info metric, with static information about KEDA build like: version, git commit and Golang runtime info.",
		},
		[]string{"version", "git_commit", "goversion", "goos", "goarch"},
	)
	scalerMetricsValue = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "scaler",
			Name:      "metrics_value",
			Help:      "The current value for each scaler's metric that would be used by the HPA in computing the target average.",
		},
		metricLabels,
	)
	scalerMetricsLatency = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "scaler",
			Name:      "metrics_latency_seconds",
			Help:      "The latency of retrieving current metric from each scaler, in seconds.",
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
	scalerErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "scaler",
			Name:      "detail_errors_total",
			Help:      "The total number of errors encountered for each scaler.",
		},
		metricLabels,
	)
	scaledObjectErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "scaled_object",
			Name:      "errors_total",
			Help:      "The number of errors that have occurred for each ScaledObject.",
		},
		[]string{"namespace", "scaledObject"},
	)
	scaledJobErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "scaled_job",
			Name:      "errors_total",
			Help:      "Number of scaled job errors",
		},
		[]string{"namespace", "scaledJob"},
	)
	emptyPrometheusMetricError = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "prometheus",
			Name:      "metrics_empty_error_total",
			Help:      "Number of times a prometheus query returns an empty result",
		},
	)
	triggerRegistered = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "trigger",
			Name:      "registered_total",
			Help:      "Total number of triggers per trigger type registered.",
		},
		[]string{"type"},
	)
	crdRegistered = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "resource",
			Name:      "registered_total",
			Help:      "Total number of KEDA custom resources per namespace for each custom resource type (CRD) registered.",
		},
		[]string{"type", "namespace"},
	)
	internalLoopLatency = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "internal_scale_loop",
			Name:      "latency_seconds",
			Help:      "Total deviation (in seconds) between the expected execution time and the actual execution time for the scaling loop.",
		},
		[]string{"namespace", "type", "resource"},
	)

	// Total emitted cloudevents.
	cloudeventEmitted = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "cloudeventsource",
			Name:      "events_emitted_total",
			Help:      "Measured the total number of emitted cloudevents. 'namespace': namespace of CloudEventSource 'cloudeventsource': name of CloudEventSource object. 'eventsink': destination of this emitted event 'state':indicated events emitted successfully or not",
		},
		[]string{"namespace", "cloudeventsource", "eventsink", "state"},
	)

	cloudeventQueueStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "cloudeventsource",
			Name:      "events_queued",
			Help:      "Indicates how many events are still queue",
		},
		[]string{"namespace"},
	)
)

type PromMetrics struct {
}

func NewPromMetrics() *PromMetrics {
	metrics.Registry.MustRegister(scalerMetricsValue)
	metrics.Registry.MustRegister(scalerMetricsLatency)
	metrics.Registry.MustRegister(internalLoopLatency)
	metrics.Registry.MustRegister(scalerActive)
	metrics.Registry.MustRegister(scalerErrors)
	metrics.Registry.MustRegister(scaledObjectErrors)
	metrics.Registry.MustRegister(scaledObjectPaused)
	metrics.Registry.MustRegister(triggerRegistered)
	metrics.Registry.MustRegister(crdRegistered)
	metrics.Registry.MustRegister(scaledJobErrors)
	metrics.Registry.MustRegister(emptyPrometheusMetricError)

	metrics.Registry.MustRegister(buildInfo)

	metrics.Registry.MustRegister(cloudeventEmitted)
	metrics.Registry.MustRegister(cloudeventQueueStatus)

	RecordBuildInfo()
	return &PromMetrics{}
}

// RecordBuildInfo publishes information about KEDA version and runtime info through an info metric (gauge).
func RecordBuildInfo() {
	buildInfo.WithLabelValues(version.Version, version.GitCommit, runtime.Version(), runtime.GOOS, runtime.GOARCH).Set(1)
}

// RecordScalerMetric create a measurement of the external metric used by the HPA
func (p *PromMetrics) RecordScalerMetric(namespace string, scaledResource string, scaler string, triggerIndex int, metric string, isScaledObject bool, value float64) {
	scalerMetricsValue.With(getLabels(namespace, scaledResource, scaler, triggerIndex, metric, isScaledObject)).Set(value)
}

// DeleteScalerMetrics deletes the scaler-related metrics so that we don't report stale values when trigger is gone
func (p *PromMetrics) DeleteScalerMetrics(namespace string, scaledResource string, isScaledObject bool) {
	scalerMetricsValue.DeletePartialMatch(prometheus.Labels{"namespace": namespace, "scaledObject": scaledResource, "type": getResourceType(isScaledObject)})
	scalerActive.DeletePartialMatch(prometheus.Labels{"namespace": namespace, "scaledObject": scaledResource, "type": getResourceType(isScaledObject)})
	scalerErrors.DeletePartialMatch(prometheus.Labels{"namespace": namespace, "scaledObject": scaledResource, "type": getResourceType(isScaledObject)})
	scalerMetricsLatency.DeletePartialMatch(prometheus.Labels{"namespace": namespace, "scaledObject": scaledResource, "type": getResourceType(isScaledObject)})
}

// RecordScalerLatency create a measurement of the latency to external metric
func (p *PromMetrics) RecordScalerLatency(namespace string, scaledResource string, scaler string, triggerIndex int, metric string, isScaledObject bool, value time.Duration) {
	scalerMetricsLatency.With(getLabels(namespace, scaledResource, scaler, triggerIndex, metric, isScaledObject)).Set(value.Seconds())
}

// RecordScalableObjectLatency create a measurement of the latency executing scalable object loop
func (p *PromMetrics) RecordScalableObjectLatency(namespace string, name string, isScaledObject bool, value time.Duration) {
	internalLoopLatency.WithLabelValues(namespace, getResourceType(isScaledObject), name).Set(value.Seconds())
}

// RecordScalerActive create a measurement of the activity of the scaler
func (p *PromMetrics) RecordScalerActive(namespace string, scaledResource string, scaler string, triggerIndex int, metric string, isScaledObject bool, active bool) {
	activeVal := 0
	if active {
		activeVal = 1
	}

	scalerActive.With(getLabels(namespace, scaledResource, scaler, triggerIndex, metric, isScaledObject)).Set(float64(activeVal))
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

// RecordScalerError counts the number of errors occurred in trying to get an external metric used by the HPA
func (p *PromMetrics) RecordScalerError(namespace string, scaledResource string, scaler string, triggerIndex int, metric string, isScaledObject bool, err error) {
	if err != nil {
		scalerErrors.With(getLabels(namespace, scaledResource, scaler, triggerIndex, metric, isScaledObject)).Inc()
		p.RecordScaledObjectError(namespace, scaledResource, err)
		return
	}
	// initialize metric with 0 if not already set
	_, errscaler := scalerErrors.GetMetricWith(getLabels(namespace, scaledResource, scaler, triggerIndex, metric, isScaledObject))
	if errscaler != nil {
		log.Error(errscaler, "Unable to record metrics: %v")
	}
}

// RecordScaledObjectError counts the number of errors with the scaled object
func (p *PromMetrics) RecordScaledObjectError(namespace string, scaledObject string, err error) {
	labels := prometheus.Labels{"namespace": namespace, "scaledObject": scaledObject}
	if err != nil {
		scaledObjectErrors.With(labels).Inc()
		return
	}
	// initialize metric with 0 if not already set
	_, errscaledobject := scaledObjectErrors.GetMetricWith(labels)
	if errscaledobject != nil {
		log.Error(errscaledobject, "Unable to record metrics: %v")
		return
	}
}

// RecordScaledJobError counts the number of errors with the scaled job
func (p *PromMetrics) RecordScaledJobError(namespace string, scaledJob string, err error) {
	labels := prometheus.Labels{"namespace": namespace, "scaledJob": scaledJob}
	if err != nil {
		scaledJobErrors.With(labels).Inc()
		return
	}
	// initialize metric with 0 if not already set
	_, errscaledjob := scaledJobErrors.GetMetricWith(labels)
	if errscaledjob != nil {
		log.Error(errscaledjob, "Unable to write to metrics to Prometheus Server: %v")
		return
	}
}

func getLabels(namespace string, scaledObject string, scaler string, triggerIndex int, metric string, isScaledObject bool) prometheus.Labels {
	return prometheus.Labels{"namespace": namespace, "scaledObject": scaledObject, "scaler": scaler, "triggerIndex": strconv.Itoa(triggerIndex), "metric": metric, "type": getResourceType(isScaledObject)}
}

func getResourceType(isScaledObject bool) string {
	if isScaledObject {
		return "scaledobject"
	}
	return "scaledjob"
}

func (p *PromMetrics) IncrementTriggerTotal(triggerType string) {
	if triggerType != "" {
		triggerRegistered.WithLabelValues(triggerType).Inc()
	}
}

func (p *PromMetrics) DecrementTriggerTotal(triggerType string) {
	if triggerType != "" {
		triggerRegistered.WithLabelValues(triggerType).Dec()
	}
}

func (p *PromMetrics) IncrementCRDTotal(crdType, namespace string) {
	if namespace == "" {
		namespace = defaultNamespace
	}

	crdRegistered.WithLabelValues(crdType, namespace).Inc()
}

func (p *PromMetrics) DecrementCRDTotal(crdType, namespace string) {
	if namespace == "" {
		namespace = defaultNamespace
	}

	crdRegistered.WithLabelValues(crdType, namespace).Dec()
}

// RecordCloudEventEmitted counts the number of cloudevent that emitted to user's sink
func (p *PromMetrics) RecordCloudEventEmitted(namespace string, cloudeventsource string, eventsink string) {
	labels := prometheus.Labels{"namespace": namespace, "cloudeventsource": cloudeventsource, "eventsink": eventsink, "state": "emitted"}
	cloudeventEmitted.With(labels).Inc()
}

// RecordCloudEventEmittedError counts the number of errors occurred in trying emit cloudevent
func (p *PromMetrics) RecordCloudEventEmittedError(namespace string, cloudeventsource string, eventsink string) {
	labels := prometheus.Labels{"namespace": namespace, "cloudeventsource": cloudeventsource, "eventsink": eventsink, "state": "failed"}
	cloudeventEmitted.With(labels).Inc()
}

// RecordCloudEventQueueStatus record the number of cloudevents that are waiting for emitting
func (p *PromMetrics) RecordCloudEventQueueStatus(namespace string, value int) {
	cloudeventQueueStatus.With(prometheus.Labels{"namespace": namespace}).Set(float64(value))
}

// RecordEmptyPrometheusMetricError counts the number of times a prometheus query returns an empty result
func (p *PromMetrics) RecordEmptyPrometheusMetricError() {
	emptyPrometheusMetricError.Inc()
}

// Returns a grpcprom server Metrics object and registers the metrics. The object contains
// interceptors to chain to the server so that all requests served are observed. Intended to be called
// as part of initialization of metricscollector, hence why this function is not exported
func newPromServerMetrics() *grpcprom.ServerMetrics {
	metricsNamespace := "keda_internal_metricsservice"

	counterNamespace := func(o *prometheus.CounterOpts) {
		o.Namespace = metricsNamespace
	}

	histogramNamespace := func(o *prometheus.HistogramOpts) {
		o.Namespace = metricsNamespace
	}

	serverMetrics := grpcprom.NewServerMetrics(
		grpcprom.WithServerHandlingTimeHistogram(
			grpcprom.WithHistogramBuckets([]float64{0.001, 0.01, 0.1, 0.3, 0.6, 1, 3, 6, 9, 20, 30, 60, 90, 120}),
			histogramNamespace,
		),
		grpcprom.WithServerCounterOptions(counterNamespace),
	)
	metrics.Registry.MustRegister(serverMetrics)

	return serverMetrics
}
