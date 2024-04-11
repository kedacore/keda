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
	scaledObjectPaused = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "scaled_object",
			Name:      "paused",
			Help:      "Indicates whether a ScaledObject is paused",
		},
		[]string{"namespace", "scaledObject"},
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
	metrics.Registry.MustRegister(scalerErrorsTotal)
	metrics.Registry.MustRegister(scalerMetricsValue)
	metrics.Registry.MustRegister(scalerMetricsLatency)
	metrics.Registry.MustRegister(internalLoopLatency)
	metrics.Registry.MustRegister(scalerActive)
	metrics.Registry.MustRegister(scalerErrors)
	metrics.Registry.MustRegister(scaledObjectErrors)
	metrics.Registry.MustRegister(scaledObjectPaused)
	metrics.Registry.MustRegister(scaledJobErrors)

	metrics.Registry.MustRegister(triggerTotalsGaugeVec)
	metrics.Registry.MustRegister(crdTotalsGaugeVec)
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

// RecordScalerLatency create a measurement of the latency to external metric
func (p *PromMetrics) RecordScalerLatency(namespace string, scaledResource string, scaler string, triggerIndex int, metric string, isScaledObject bool, value float64) {
	scalerMetricsLatency.With(getLabels(namespace, scaledResource, scaler, triggerIndex, metric, isScaledObject)).Set(value)
}

// RecordScalableObjectLatency create a measurement of the latency executing scalable object loop
func (p *PromMetrics) RecordScalableObjectLatency(namespace string, name string, isScaledObject bool, value float64) {
	internalLoopLatency.WithLabelValues(namespace, getResourceType(isScaledObject), name).Set(value)
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
		scalerErrorsTotal.With(prometheus.Labels{}).Inc()
		return
	}
	// initialize metric with 0 if not already set
	_, errscaler := scalerErrors.GetMetricWith(getLabels(namespace, scaledResource, scaler, triggerIndex, metric, isScaledObject))
	if errscaler != nil {
		log.Error(errscaler, "Unable to write to metrics to Prometheus Server: %v")
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
		log.Error(errscaledobject, "Unable to write to metrics to Prometheus Server: %v")
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
		log.Error(err, "Unable to write to metrics to Prometheus Server: %v")
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
		triggerTotalsGaugeVec.WithLabelValues(triggerType).Inc()
	}
}

func (p *PromMetrics) DecrementTriggerTotal(triggerType string) {
	if triggerType != "" {
		triggerTotalsGaugeVec.WithLabelValues(triggerType).Dec()
	}
}

func (p *PromMetrics) IncrementCRDTotal(crdType, namespace string) {
	if namespace == "" {
		namespace = defaultNamespace
	}

	crdTotalsGaugeVec.WithLabelValues(crdType, namespace).Inc()
}

func (p *PromMetrics) DecrementCRDTotal(crdType, namespace string) {
	if namespace == "" {
		namespace = defaultNamespace
	}

	crdTotalsGaugeVec.WithLabelValues(crdType, namespace).Dec()
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
