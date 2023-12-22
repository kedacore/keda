package metricscollector

import (
	"context"
	"fmt"
	"runtime"
	"strconv"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	api "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kedacore/keda/v2/version"
)

var otLog = logf.Log.WithName("otel_collector")

const meterName = "keda-open-telemetry-metrics"
const defaultNamespace = "default"

var (
	meterProvider               *metric.MeterProvider
	meter                       api.Meter
	otScalerErrorsCounter       api.Int64Counter
	otScaledObjectErrorsCounter api.Int64Counter
	otTriggerTotalsCounter      api.Int64UpDownCounter
	otCrdTotalsCounter          api.Int64UpDownCounter

	otelScalerMetricVal         OtelMetricFloat64Val
	otelScalerMetricsLatencyVal OtelMetricFloat64Val
	otelInternalLoopLatencyVal  OtelMetricFloat64Val
	otelBuildInfoVal            OtelMetricInt64Val

	otCloudEventEmittedCounter api.Int64Counter
	otCloudEventQueueStatusVal OtelMetricFloat64Val

	otelScalerActiveVal OtelMetricFloat64Val
)

type OtelMetrics struct {
}

type OtelMetricInt64Val struct {
	val               int64
	measurementOption api.MeasurementOption
}

type OtelMetricFloat64Val struct {
	val               float64
	measurementOption api.MeasurementOption
}

func NewOtelMetrics(options ...metric.Option) *OtelMetrics {
	// create default options with env
	if options == nil {
		exporter, err := otlpmetrichttp.New(context.Background())
		if err != nil {
			fmt.Printf("Error:" + err.Error())
			return nil
		}
		options = []metric.Option{metric.WithReader(metric.NewPeriodicReader(exporter))}
	}

	meterProvider = metric.NewMeterProvider(options...)
	otel.SetMeterProvider(meterProvider)

	meter = meterProvider.Meter(meterName)
	initMeters()

	otel := &OtelMetrics{}
	otel.RecordBuildInfo()
	return otel
}

func initMeters() {
	var err error
	msg := "create opentelemetry counter failed"

	otScalerErrorsCounter, err = meter.Int64Counter("keda.scaler.errors", api.WithDescription("Number of scaler errors"))
	if err != nil {
		otLog.Error(err, msg)
	}

	otScaledObjectErrorsCounter, err = meter.Int64Counter("keda.scaledobject.errors", api.WithDescription("Number of scaled object errors"))
	if err != nil {
		otLog.Error(err, msg)
	}

	otTriggerTotalsCounter, err = meter.Int64UpDownCounter("keda.trigger.totals", api.WithDescription("Total triggers"))
	if err != nil {
		otLog.Error(err, msg)
	}

	otCrdTotalsCounter, err = meter.Int64UpDownCounter("keda.resource.totals", api.WithDescription("Total resources"))
	if err != nil {
		otLog.Error(err, msg)
	}

	_, err = meter.Float64ObservableGauge(
		"keda.scaler.metrics.value",
		api.WithDescription("Metric Value used for HPA"),
		api.WithFloat64Callback(ScalerMetricValueCallback),
	)
	if err != nil {
		otLog.Error(err, msg)
	}

	_, err = meter.Float64ObservableGauge(
		"keda.scaler.metrics.latency",
		api.WithDescription("Scaler Metrics Latency"),
		api.WithFloat64Callback(ScalerMetricsLatencyCallback),
	)
	if err != nil {
		otLog.Error(err, msg)
	}

	_, err = meter.Float64ObservableGauge(
		"keda.internal.scale.loop.latency",
		api.WithDescription("Internal latency of ScaledObject/ScaledJob loop execution"),
		api.WithFloat64Callback(ScalableObjectLatencyCallback),
	)
	if err != nil {
		otLog.Error(err, msg)
	}

	_, err = meter.Float64ObservableGauge(
		"keda.scaler.active",
		api.WithDescription("Activity of a Scaler Metric"),
		api.WithFloat64Callback(ScalerActiveCallback),
	)
	if err != nil {
		otLog.Error(err, msg)
	}

	_, err = meter.Int64ObservableGauge(
		"keda.build.info",
		api.WithDescription("A metric with a constant '1' value labeled by version, git_commit and goversion from which KEDA was built."),
		api.WithInt64Callback(BuildInfoCallback),
	)
	if err != nil {
		otLog.Error(err, msg)
	}

	otCloudEventEmittedCounter, err = meter.Int64Counter("keda.cloudeventsource.events.emitted.count", api.WithDescription("Measured the total number of emitted cloudevents. 'namespace': namespace of CloudEventSource 'cloudeventsource': name of CloudEventSource object. 'eventsink': destination of this emitted event 'state':indicated events emitted successfully or not"))
	if err != nil {
		otLog.Error(err, msg)
	}

	_, err = meter.Float64ObservableGauge(
		"keda.cloudeventsource.events.queued",
		api.WithDescription("Indicates how many events are still queue"),
		api.WithFloat64Callback(CloudeventQueueStatusCallback),
	)
	if err != nil {
		otLog.Error(err, msg)
	}
}

func BuildInfoCallback(_ context.Context, obsrv api.Int64Observer) error {
	if otelBuildInfoVal.measurementOption != nil {
		obsrv.Observe(otelBuildInfoVal.val, otelBuildInfoVal.measurementOption)
	}
	otelBuildInfoVal = OtelMetricInt64Val{}
	return nil
}

// RecordBuildInfo publishes information about KEDA version and runtime info through an info metric (gauge).
func (o *OtelMetrics) RecordBuildInfo() {
	opt := api.WithAttributes(
		attribute.Key("version").String(version.Version),
		attribute.Key("git_commit").String(version.GitCommit),
		attribute.Key("goversion").String(runtime.Version()),
		attribute.Key("goos").String(runtime.GOOS),
		attribute.Key("goarch").String(runtime.GOARCH),
	)
	otelBuildInfoVal.val = 1
	otelBuildInfoVal.measurementOption = opt
}

func ScalerMetricValueCallback(_ context.Context, obsrv api.Float64Observer) error {
	if otelScalerMetricVal.measurementOption != nil {
		obsrv.Observe(otelScalerMetricVal.val, otelScalerMetricVal.measurementOption)
	}
	otelScalerMetricVal = OtelMetricFloat64Val{}
	return nil
}

func (o *OtelMetrics) RecordScalerMetric(namespace string, scaledObject string, scaler string, scalerIndex int, metric string, value float64) {
	otelScalerMetricVal.val = value
	otelScalerMetricVal.measurementOption = getScalerMeasurementOption(namespace, scaledObject, scaler, scalerIndex, metric)
}

func ScalerMetricsLatencyCallback(_ context.Context, obsrv api.Float64Observer) error {
	if otelScalerMetricsLatencyVal.measurementOption != nil {
		obsrv.Observe(otelScalerMetricsLatencyVal.val, otelScalerMetricsLatencyVal.measurementOption)
	}
	otelScalerMetricsLatencyVal = OtelMetricFloat64Val{}
	return nil
}

// RecordScalerLatency create a measurement of the latency to external metric
func (o *OtelMetrics) RecordScalerLatency(namespace string, scaledObject string, scaler string, scalerIndex int, metric string, value float64) {
	otelScalerMetricsLatencyVal.val = value
	otelScalerMetricsLatencyVal.measurementOption = getScalerMeasurementOption(namespace, scaledObject, scaler, scalerIndex, metric)
}

func ScalableObjectLatencyCallback(_ context.Context, obsrv api.Float64Observer) error {
	if otelInternalLoopLatencyVal.measurementOption != nil {
		obsrv.Observe(otelInternalLoopLatencyVal.val, otelInternalLoopLatencyVal.measurementOption)
	}
	otelInternalLoopLatencyVal = OtelMetricFloat64Val{}
	return nil
}

// RecordScalableObjectLatency create a measurement of the latency executing scalable object loop
func (o *OtelMetrics) RecordScalableObjectLatency(namespace string, name string, isScaledObject bool, value float64) {
	resourceType := "scaledjob"
	if isScaledObject {
		resourceType = "scaledobject"
	}

	opt := api.WithAttributes(
		attribute.Key("namespace").String(namespace),
		attribute.Key("type").String(resourceType),
		attribute.Key("name").String(name))

	otelInternalLoopLatencyVal.val = value
	otelInternalLoopLatencyVal.measurementOption = opt
}

func ScalerActiveCallback(_ context.Context, obsrv api.Float64Observer) error {
	if otelScalerActiveVal.measurementOption != nil {
		obsrv.Observe(otelScalerActiveVal.val, otelScalerActiveVal.measurementOption)
	}
	otelScalerActiveVal = OtelMetricFloat64Val{}
	return nil
}

// RecordScalerActive create a measurement of the activity of the scaler
func (o *OtelMetrics) RecordScalerActive(namespace string, scaledObject string, scaler string, scalerIndex int, metric string, active bool) {
	activeVal := -1
	if active {
		activeVal = 1
	}

	otelScalerActiveVal.val = float64(activeVal)
	otelScalerActiveVal.measurementOption = getScalerMeasurementOption(namespace, scaledObject, scaler, scalerIndex, metric)
}

// RecordScaledObjectPaused marks whether the current ScaledObject is paused.
func (o *OtelMetrics) RecordScaledObjectPaused(namespace string, scaledObject string, active bool) {
	activeVal := 0
	if active {
		activeVal = 1
	}

	opt := api.WithAttributes(
		attribute.Key("namespace").String(namespace),
		attribute.Key("scaledObject").String(scaledObject),
	)

	cback := func(ctx context.Context, obsrv api.Float64Observer) error {
		obsrv.Observe(float64(activeVal), opt)
		return nil
	}
	_, err := meter.Float64ObservableGauge(
		"keda.scaled.object.paused",
		api.WithDescription("Indicates whether a ScaledObject is paused"),
		api.WithFloat64Callback(cback),
	)
	if err != nil {
		otLog.Error(err, "failed to register scaled object paused metric", "namespace", namespace, "scaledObject", scaledObject)
	}
}

// RecordScalerError counts the number of errors occurred in trying get an external metric used by the HPA
func (o *OtelMetrics) RecordScalerError(namespace string, scaledObject string, scaler string, scalerIndex int, metric string, err error) {
	if err != nil {
		otScalerErrorsCounter.Add(context.Background(), 1, getScalerMeasurementOption(namespace, scaledObject, scaler, scalerIndex, metric))
		o.RecordScaledObjectError(namespace, scaledObject, err)
		return
	}
}

// RecordScaledObjectError counts the number of errors with the scaled object
func (o *OtelMetrics) RecordScaledObjectError(namespace string, scaledObject string, err error) {
	opt := api.WithAttributes(
		attribute.Key("namespace").String(namespace),
		attribute.Key("scaledObject").String(scaledObject))
	if err != nil {
		otScaledObjectErrorsCounter.Add(context.Background(), 1, opt)
		return
	}
}

func (o *OtelMetrics) IncrementTriggerTotal(triggerType string) {
	if triggerType != "" {
		otTriggerTotalsCounter.Add(context.Background(), 1, api.WithAttributes(attribute.Key("type").String(triggerType)))
	}
}

func (o *OtelMetrics) DecrementTriggerTotal(triggerType string) {
	if triggerType != "" {
		otTriggerTotalsCounter.Add(context.Background(), -1, api.WithAttributes(attribute.Key("type").String(triggerType)))
	}
}

func (o *OtelMetrics) IncrementCRDTotal(crdType, namespace string) {
	if namespace == "" {
		namespace = defaultNamespace
	}
	opt := api.WithAttributes(
		attribute.Key("namespace").String(namespace),
		attribute.Key("type").String(crdType),
	)

	otCrdTotalsCounter.Add(context.Background(), 1, opt)
}

func (o *OtelMetrics) DecrementCRDTotal(crdType, namespace string) {
	if namespace == "" {
		namespace = defaultNamespace
	}

	opt := api.WithAttributes(
		attribute.Key("namespace").String(namespace),
		attribute.Key("type").String(crdType),
	)
	otCrdTotalsCounter.Add(context.Background(), -1, opt)
}

func getScalerMeasurementOption(namespace string, scaledObject string, scaler string, scalerIndex int, metric string) api.MeasurementOption {
	return api.WithAttributes(
		attribute.Key("namespace").String(namespace),
		attribute.Key("scaledObject").String(scaledObject),
		attribute.Key("scaler").String(scaler),
		attribute.Key("scalerIndex").String(strconv.Itoa(scalerIndex)),
		attribute.Key("metric").String(metric),
	)
}

// RecordCloudEventEmitted counts the number of cloudevent that emitted to user's sink
func (o *OtelMetrics) RecordCloudEventEmitted(namespace string, cloudeventsource string, eventsink string) {
	opt := api.WithAttributes(
		attribute.Key("namespace").String(namespace),
		attribute.Key("cloudEventSource").String(cloudeventsource),
		attribute.Key("eventsink").String(eventsink),
		attribute.Key("state").String("emitted"),
	)
	otCloudEventEmittedCounter.Add(context.Background(), 1, opt)
}

// RecordCloudEventEmitted counts the number of errors occurred in trying emit cloudevent
func (o *OtelMetrics) RecordCloudEventEmittedError(namespace string, cloudeventsource string, eventsink string) {
	opt := api.WithAttributes(
		attribute.Key("namespace").String(namespace),
		attribute.Key("cloudEventSource").String(cloudeventsource),
		attribute.Key("eventsink").String(eventsink),
		attribute.Key("state").String("failed"),
	)
	otCloudEventEmittedCounter.Add(context.Background(), 1, opt)
}

func CloudeventQueueStatusCallback(_ context.Context, obsrv api.Float64Observer) error {
	if otCloudEventQueueStatusVal.measurementOption != nil {
		obsrv.Observe(otCloudEventQueueStatusVal.val, otCloudEventQueueStatusVal.measurementOption)
	}
	otCloudEventQueueStatusVal = OtelMetricFloat64Val{}
	return nil
}

// RecordCloudEventSourceQueueStatus record the number of cloudevents that are waiting for emitting
func (o *OtelMetrics) RecordCloudEventQueueStatus(namespace string, value int) {
	opt := api.WithAttributes(
		attribute.Key("namespace").String(namespace),
	)

	otCloudEventQueueStatusVal.val = float64(value)
	otCloudEventQueueStatusVal.measurementOption = opt
}
