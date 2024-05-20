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
	"fmt"
	"runtime"
	"strconv"
	"time"

	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/kedacore/keda/v2/version"
)

// bestPracticeDeprecatedMsg is a constant string that is used to indicate that a metric is deprecated as
// part of best practice refactoring - https://github.com/kedacore/keda/pull/5174
const bestPracticeDeprecatedMsg = "DEPRECATED - will be removed in 2.16:"

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
	scalerMetricsLatencyDeprecated = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "scaler",
			Name:      "metrics_latency",
			Help:      fmt.Sprintf("%v use 'keda_scaler_metrics_latency_seconds' instead.", bestPracticeDeprecatedMsg),
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
	scalerErrorsDeprecated = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "scaler",
			Name:      "errors",
			Help:      fmt.Sprintf("%v use 'keda_scaler_detail_errors_total' instead.", bestPracticeDeprecatedMsg),
		},
		metricLabels,
	)
	scalerErrorsTotalDeprecated = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "scaler",
			Name:      "errors_total",
			Help:      fmt.Sprintf("%v use use a `sum(keda_scaler_detail_errors_total{scaler!=\"\"})` over all scalers", bestPracticeDeprecatedMsg),
		},
		[]string{},
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
	scaledObjectErrorsDeprecated = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "scaled_object",
			Name:      "errors",
			Help:      fmt.Sprintf("%v use 'keda_scaled_object_errors_total' instead.", bestPracticeDeprecatedMsg),
		},
		[]string{"namespace", "scaledObject"},
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
	scaledJobErrorsDeprecated = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "scaled_job",
			Name:      "errors",
			Help:      fmt.Sprintf("%v use 'keda_scaled_job_errors_total' instead.", bestPracticeDeprecatedMsg),
		},
		[]string{"namespace", "scaledJob"},
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

	triggerTotalsGaugeVecDeprecated = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "trigger",
			Name:      "totals",
			Help:      fmt.Sprintf("%v use 'keda_trigger_registered_total' instead.", bestPracticeDeprecatedMsg),
		},
		[]string{"type"},
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
	crdTotalsGaugeVecDeprecated = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "resource",
			Name:      "totals",
			Help:      fmt.Sprintf("%v use 'keda_resource_registered_total' instead.", bestPracticeDeprecatedMsg),
		},
		[]string{"type", "namespace"},
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
	internalLoopLatencyDeprecated = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: DefaultPromMetricsNamespace,
			Subsystem: "internal_scale_loop",
			Name:      "latency",
			Help:      fmt.Sprintf("%v use 'keda_internal_scale_loop_latency_seconds' instead.", bestPracticeDeprecatedMsg),
		},
		[]string{"namespace", "type", "resource"},
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
	metrics.Registry.MustRegister(triggerRegistered)
	metrics.Registry.MustRegister(crdRegistered)
	metrics.Registry.MustRegister(scaledJobErrorsDeprecated)
	metrics.Registry.MustRegister(scaledJobErrors)

	metrics.Registry.MustRegister(triggerTotalsGaugeVecDeprecated)
	metrics.Registry.MustRegister(crdTotalsGaugeVecDeprecated)
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
func (p *PromMetrics) RecordScalerLatency(namespace string, scaledResource string, scaler string, triggerIndex int, metric string, isScaledObject bool, value time.Duration) {
	scalerMetricsLatency.With(getLabels(namespace, scaledResource, scaler, triggerIndex, metric, isScaledObject)).Set(value.Seconds())
	scalerMetricsLatencyDeprecated.With(getLabels(namespace, scaledResource, scaler, triggerIndex, metric, isScaledObject)).Set(float64(value.Milliseconds()))
}

// RecordScalableObjectLatency create a measurement of the latency executing scalable object loop
func (p *PromMetrics) RecordScalableObjectLatency(namespace string, name string, isScaledObject bool, value time.Duration) {
	internalLoopLatency.WithLabelValues(namespace, getResourceType(isScaledObject), name).Set(value.Seconds())
	internalLoopLatencyDeprecated.WithLabelValues(namespace, getResourceType(isScaledObject), name).Set(float64(value.Milliseconds()))
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
		scalerErrorsDeprecated.With(getLabels(namespace, scaledResource, scaler, triggerIndex, metric, isScaledObject)).Inc()
		p.RecordScaledObjectError(namespace, scaledResource, err)
		scalerErrorsTotalDeprecated.With(prometheus.Labels{}).Inc()
		return
	}
	// initialize metric with 0 if not already set
	_, errscaler := scalerErrors.GetMetricWith(getLabels(namespace, scaledResource, scaler, triggerIndex, metric, isScaledObject))
	if errscaler != nil {
		log.Error(errscaler, "Unable to record metrics: %v")
	}
	_, errscalerdep := scalerErrorsDeprecated.GetMetricWith(getLabels(namespace, scaledResource, scaler, triggerIndex, metric, isScaledObject))
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

// RecordScaledJobError counts the number of errors with the scaled job
func (p *PromMetrics) RecordScaledJobError(namespace string, scaledJob string, err error) {
	labels := prometheus.Labels{"namespace": namespace, "scaledJob": scaledJob}
	if err != nil {
		scaledJobErrorsDeprecated.With(labels).Inc()
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
		triggerRegistered.WithLabelValues(triggerType).Inc()
		triggerTotalsGaugeVecDeprecated.WithLabelValues(triggerType).Inc()
	}
}

func (p *PromMetrics) DecrementTriggerTotal(triggerType string) {
	if triggerType != "" {
		triggerRegistered.WithLabelValues(triggerType).Dec()
		triggerTotalsGaugeVecDeprecated.WithLabelValues(triggerType).Dec()
	}
}

func (p *PromMetrics) IncrementCRDTotal(crdType, namespace string) {
	if namespace == "" {
		namespace = defaultNamespace
	}

	crdRegistered.WithLabelValues(crdType, namespace).Inc()
	crdTotalsGaugeVecDeprecated.WithLabelValues(crdType, namespace).Inc()
}

func (p *PromMetrics) DecrementCRDTotal(crdType, namespace string) {
	if namespace == "" {
		namespace = defaultNamespace
	}

	crdRegistered.WithLabelValues(crdType, namespace).Dec()
	crdTotalsGaugeVecDeprecated.WithLabelValues(crdType, namespace).Dec()
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
