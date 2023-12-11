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

package metricscollector

const (
	ClusterTriggerAuthenticationResource = "cluster_trigger_authentication"
	TriggerAuthenticationResource        = "trigger_authentication"
	ScaledObjectResource                 = "scaled_object"
	ScaledJobResource                    = "scaled_job"
	CloudEventSourceResource             = "cloudevent_source"

	DefaultPromMetricsNamespace = "keda"
)

var (
	collectors []MetricsCollector
)

type MetricsCollector interface {
	RecordScalerMetric(namespace string, scaledObject string, scaler string, scalerIndex int, metric string, value float64)

	// RecordScalerLatency create a measurement of the latency to external metric
	RecordScalerLatency(namespace string, scaledObject string, scaler string, scalerIndex int, metric string, value float64)

	// RecordScalableObjectLatency create a measurement of the latency executing scalable object loop
	RecordScalableObjectLatency(namespace string, name string, isScaledObject bool, value float64)

	// RecordScalerActive create a measurement of the activity of the scaler
	RecordScalerActive(namespace string, scaledObject string, scaler string, scalerIndex int, metric string, active bool)

	// RecordScaledObjectPaused marks whether the current ScaledObject is paused.
	RecordScaledObjectPaused(namespace string, scaledObject string, active bool)

	// RecordScalerError counts the number of errors occurred in trying get an external metric used by the HPA
	RecordScalerError(namespace string, scaledObject string, scaler string, scalerIndex int, metric string, err error)

	// RecordScaledObjectError counts the number of errors with the scaled object
	RecordScaledObjectError(namespace string, scaledObject string, err error)

	IncrementTriggerTotal(triggerType string)

	DecrementTriggerTotal(triggerType string)

	IncrementCRDTotal(crdType, namespace string)

	DecrementCRDTotal(crdType, namespace string)

	// RecordCloudEventEmitted counts the number of cloudevent that emitted to user's sink
	RecordCloudEventEmitted(namespace string, cloudeventsource string, eventsink string)

	// RecordCloudEventEmittedError counts the number of errors occurred in trying emit cloudevent
	RecordCloudEventEmittedError(namespace string, cloudeventsource string, eventsink string)

	// RecordCloudEventQueueStatus record the number of cloudevents that are waiting for emitting
	RecordCloudEventQueueStatus(namespace string, value int)
}

func NewMetricsCollectors(enablePrometheusMetrics bool, enableOpenTelemetryMetrics bool) {
	if enablePrometheusMetrics {
		promometrics := NewPromMetrics()
		collectors = append(collectors, promometrics)
	}

	if enableOpenTelemetryMetrics {
		otelmetrics := NewOtelMetrics()
		collectors = append(collectors, otelmetrics)
	}
}

// RecordScalerMetric create a measurement of the external metric used by the HPA
func RecordScalerMetric(namespace string, scaledObject string, scaler string, scalerIndex int, metric string, value float64) {
	for _, element := range collectors {
		element.RecordScalerMetric(namespace, scaledObject, scaler, scalerIndex, metric, value)
	}
}

// RecordScalerLatency create a measurement of the latency to external metric
func RecordScalerLatency(namespace string, scaledObject string, scaler string, scalerIndex int, metric string, value float64) {
	for _, element := range collectors {
		element.RecordScalerLatency(namespace, scaledObject, scaler, scalerIndex, metric, value)
	}
}

// RecordScalableObjectLatency create a measurement of the latency executing scalable object loop
func RecordScalableObjectLatency(namespace string, name string, isScaledObject bool, value float64) {
	for _, element := range collectors {
		element.RecordScalableObjectLatency(namespace, name, isScaledObject, value)
	}
}

// RecordScalerActive create a measurement of the activity of the scaler
func RecordScalerActive(namespace string, scaledObject string, scaler string, scalerIndex int, metric string, active bool) {
	for _, element := range collectors {
		element.RecordScalerActive(namespace, scaledObject, scaler, scalerIndex, metric, active)
	}
}

// RecordScaledObjectPaused marks whether the current ScaledObject is paused.
func RecordScaledObjectPaused(namespace string, scaledObject string, active bool) {
	for _, element := range collectors {
		element.RecordScaledObjectPaused(namespace, scaledObject, active)
	}
}

// RecordScalerError counts the number of errors occurred in trying get an external metric used by the HPA
func RecordScalerError(namespace string, scaledObject string, scaler string, scalerIndex int, metric string, err error) {
	for _, element := range collectors {
		element.RecordScalerError(namespace, scaledObject, scaler, scalerIndex, metric, err)
	}
}

// RecordScaledObjectError counts the number of errors with the scaled object
func RecordScaledObjectError(namespace string, scaledObject string, err error) {
	for _, element := range collectors {
		element.RecordScaledObjectError(namespace, scaledObject, err)
	}
}

func IncrementTriggerTotal(triggerType string) {
	for _, element := range collectors {
		element.IncrementTriggerTotal(triggerType)
	}
}

func DecrementTriggerTotal(triggerType string) {
	for _, element := range collectors {
		element.DecrementTriggerTotal(triggerType)
	}
}

func IncrementCRDTotal(crdType, namespace string) {
	for _, element := range collectors {
		element.IncrementCRDTotal(crdType, namespace)
	}
}

func DecrementCRDTotal(crdType, namespace string) {
	for _, element := range collectors {
		element.DecrementCRDTotal(crdType, namespace)
	}
}

// RecordCloudEventEmitted counts the number of cloudevent that emitted to user's sink
func RecordCloudEventEmitted(namespace string, cloudeventsource string, eventsink string) {
	for _, element := range collectors {
		element.RecordCloudEventEmitted(namespace, cloudeventsource, eventsink)
	}
}

// RecordCloudEventEmittedError counts the number of errors occurred in trying emit cloudevent
func RecordCloudEventEmittedError(namespace string, cloudeventsource string, eventsink string) {
	for _, element := range collectors {
		element.RecordCloudEventEmittedError(namespace, cloudeventsource, eventsink)
	}
}

// RecordCloudEventQueueStatus record the number of cloudevents that are waiting for emitting
func RecordCloudEventQueueStatus(namespace string, value int) {
	for _, element := range collectors {
		element.RecordCloudEventQueueStatus(namespace, value)
	}
}
