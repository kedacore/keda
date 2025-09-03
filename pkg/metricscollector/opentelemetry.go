package metricscollector

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
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
	meterProvider                    *metric.MeterProvider
	meter                            api.Meter
	otScalerErrorsCounter            api.Int64Counter
	otScaledObjectErrorsCounter      api.Int64Counter
	otScaledJobErrorsCounter         api.Int64Counter
	otTriggerTotalsCounterDeprecated api.Int64UpDownCounter
	otCrdTotalsCounterDeprecated     api.Int64UpDownCounter
	otTriggerRegisteredTotalsCounter api.Int64UpDownCounter
	otCrdRegisteredTotalsCounter     api.Int64UpDownCounter
	otEmptyPrometheusMetricError     api.Int64Counter

	otelScalerMetricVals                  []OtelMetricFloat64Val
	otelScalerMetricsLatencyVals          []OtelMetricFloat64Val
	otelScalerMetricsLatencyValDeprecated []OtelMetricFloat64Val
	otelInternalLoopLatencyVals           []OtelMetricFloat64Val
	otelInternalLoopLatencyValDeprecated  []OtelMetricFloat64Val
	otelBuildInfoVal                      OtelMetricInt64Val

	otCloudEventEmittedCounter  api.Int64Counter
	otCloudEventQueueStatusVals []OtelMetricFloat64Val

	otelScalerActiveVals []OtelMetricFloat64Val
	otelScalerPauseVals  []OtelMetricFloat64Val
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
		protocol := os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL")

		var exporter metric.Exporter
		var err error
		switch protocol {
		case "grpc":
			otLog.V(1).Info("start OTEL grpc client")
			exporter, err = otlpmetricgrpc.New(context.Background())
		default:
			otLog.V(1).Info("start OTEL http client")
			exporter, err = otlpmetrichttp.New(context.Background())
		}

		if err != nil {
			fmt.Printf("Error: %s", err.Error())
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

	otScaledJobErrorsCounter, err = meter.Int64Counter("keda.scaledjob.errors", api.WithDescription("Number of scaled job errors"))
	if err != nil {
		otLog.Error(err, msg)
	}

	otTriggerTotalsCounterDeprecated, err = meter.Int64UpDownCounter("keda.trigger.totals", api.WithDescription("DEPRECATED - will be removed in 2.16 - use 'keda.trigger.registered.count' instead"))
	if err != nil {
		otLog.Error(err, msg)
	}

	otTriggerRegisteredTotalsCounter, err = meter.Int64UpDownCounter("keda.trigger.registered.count", api.WithDescription("Total number of triggers per trigger type registered"))
	if err != nil {
		otLog.Error(err, msg)
	}

	otCrdTotalsCounterDeprecated, err = meter.Int64UpDownCounter("keda.resource.totals", api.WithDescription("DEPRECATED - will be removed in 2.16 - use 'keda.resource.registered.count' instead"))
	if err != nil {
		otLog.Error(err, msg)
	}

	otCrdRegisteredTotalsCounter, err = meter.Int64UpDownCounter("keda.resource.registered.count", api.WithDescription("Total number of KEDA custom resources per namespace for each custom resource type (CRD) registered"))
	if err != nil {
		otLog.Error(err, msg)
	}

	otEmptyPrometheusMetricError, err = meter.Int64Counter("keda.prometheus.metrics.empty.error", api.WithDescription("Number of times a prometheus query returns an empty result"))
	if err != nil {
		otLog.Error(err, msg)
	}

	_, err = meter.Float64ObservableGauge(
		"keda.scaler.metrics.value",
		api.WithDescription("The current value for each scaler's metric that would be used by the HPA in computing the target average"),
		api.WithFloat64Callback(ScalerMetricValueCallback),
	)
	if err != nil {
		otLog.Error(err, msg)
	}

	_, err = meter.Float64ObservableGauge(
		"keda.scaler.metrics.latency",
		api.WithDescription("DEPRECATED - use `keda.scaler.metrics.latency.seconds` instead"),
		api.WithFloat64Callback(ScalerMetricsLatencyCallbackDeprecated),
	)
	if err != nil {
		otLog.Error(err, msg)
	}
	_, err = meter.Float64ObservableGauge(
		"keda.scaler.metrics.latency.seconds",
		api.WithDescription("The latency of retrieving current metric from each scaler"),
		api.WithUnit("s"),
		api.WithFloat64Callback(ScalerMetricsLatencyCallback),
	)
	if err != nil {
		otLog.Error(err, msg)
	}

	_, err = meter.Float64ObservableGauge(
		"keda.internal.scale.loop.latency",
		api.WithDescription("DEPRECATED - use `keda.internal.scale.loop.latency.seconds` instead"),
		api.WithFloat64Callback(ScalableObjectLatencyCallbackDeprecated),
	)
	if err != nil {
		otLog.Error(err, msg)
	}
	_, err = meter.Float64ObservableGauge(
		"keda.internal.scale.loop.latency.seconds",
		api.WithDescription("Internal latency of ScaledObject/ScaledJob loop execution"),
		api.WithUnit("s"),
		api.WithFloat64Callback(ScalableObjectLatencyCallback),
	)
	if err != nil {
		otLog.Error(err, msg)
	}

	_, err = meter.Float64ObservableGauge(
		"keda.scaler.active",
		api.WithDescription("Indicates whether a scaler is active (1), or not (0)"),
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

	_, err = meter.Float64ObservableGauge(
		"keda.scaled.object.paused",
		api.WithDescription("Indicates whether a ScaledObject is paused"),
		api.WithFloat64Callback(PausedStatusCallback),
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
	for _, v := range otelScalerMetricVals {
		obsrv.Observe(v.val, v.measurementOption)
	}
	otelScalerMetricVals = []OtelMetricFloat64Val{}
	return nil
}

func (o *OtelMetrics) RecordScalerMetric(namespace string, scaledResource string, scaler string, triggerIndex int, metric string, isScaledObject bool, value float64) {
	otelScalerMetric := OtelMetricFloat64Val{}
	otelScalerMetric.val = value
	otelScalerMetric.measurementOption = getScalerMeasurementOption(namespace, scaledResource, scaler, triggerIndex, metric, isScaledObject)
	otelScalerMetricVals = append(otelScalerMetricVals, otelScalerMetric)
}

func (o *OtelMetrics) DeleteScalerMetrics(string, string, bool) {
	// noop for OTel
}

func ScalerMetricsLatencyCallback(_ context.Context, obsrv api.Float64Observer) error {
	for _, v := range otelScalerMetricsLatencyVals {
		obsrv.Observe(v.val, v.measurementOption)
	}
	otelScalerMetricsLatencyVals = []OtelMetricFloat64Val{}
	return nil
}

func ScalerMetricsLatencyCallbackDeprecated(_ context.Context, obsrv api.Float64Observer) error {
	for _, v := range otelScalerMetricsLatencyValDeprecated {
		obsrv.Observe(v.val, v.measurementOption)
	}
	otelScalerMetricsLatencyValDeprecated = []OtelMetricFloat64Val{}
	return nil
}

// RecordScalerLatency create a measurement of the latency to external metric
func (o *OtelMetrics) RecordScalerLatency(namespace string, scaledResource string, scaler string, triggerIndex int, metric string, isScaledObject bool, value time.Duration) {
	otelScalerMetricsLatency := OtelMetricFloat64Val{}
	otelScalerMetricsLatency.val = value.Seconds()
	otelScalerMetricsLatency.measurementOption = getScalerMeasurementOption(namespace, scaledResource, scaler, triggerIndex, metric, isScaledObject)
	otelScalerMetricsLatencyVals = append(otelScalerMetricsLatencyVals, otelScalerMetricsLatency)

	otelScalerMetricsLatencyValD := OtelMetricFloat64Val{}
	otelScalerMetricsLatencyValD.val = float64(value.Milliseconds())
	otelScalerMetricsLatencyValD.measurementOption = getScalerMeasurementOption(namespace, scaledResource, scaler, triggerIndex, metric, isScaledObject)
	otelScalerMetricsLatencyValDeprecated = append(otelScalerMetricsLatencyValDeprecated, otelScalerMetricsLatencyValD)
}

func ScalableObjectLatencyCallback(_ context.Context, obsrv api.Float64Observer) error {
	for _, v := range otelInternalLoopLatencyVals {
		obsrv.Observe(v.val, v.measurementOption)
	}
	otelInternalLoopLatencyVals = []OtelMetricFloat64Val{}
	return nil
}

func ScalableObjectLatencyCallbackDeprecated(_ context.Context, obsrv api.Float64Observer) error {
	for _, v := range otelInternalLoopLatencyValDeprecated {
		obsrv.Observe(v.val, v.measurementOption)
	}
	otelInternalLoopLatencyValDeprecated = []OtelMetricFloat64Val{}
	return nil
}

// RecordScalableObjectLatency create a measurement of the latency executing scalable object loop
func (o *OtelMetrics) RecordScalableObjectLatency(namespace string, name string, isScaledObject bool, value time.Duration) {
	resourceType := "scaledjob"
	if isScaledObject {
		resourceType = "scaledobject"
	}

	opt := api.WithAttributes(
		attribute.Key("namespace").String(namespace),
		attribute.Key("type").String(resourceType),
		attribute.Key("name").String(name))

	otelInternalLoopLatency := OtelMetricFloat64Val{}
	otelInternalLoopLatency.val = value.Seconds()
	otelInternalLoopLatency.measurementOption = opt
	otelInternalLoopLatencyVals = append(otelInternalLoopLatencyVals, otelInternalLoopLatency)

	otelInternalLoopLatencyD := OtelMetricFloat64Val{}
	otelInternalLoopLatencyD.val = float64(value.Milliseconds())
	otelInternalLoopLatencyD.measurementOption = opt
	otelInternalLoopLatencyValDeprecated = append(otelInternalLoopLatencyValDeprecated, otelInternalLoopLatencyD)
}

func ScalerActiveCallback(_ context.Context, obsrv api.Float64Observer) error {
	for _, v := range otelScalerActiveVals {
		obsrv.Observe(v.val, v.measurementOption)
	}
	otelScalerActiveVals = []OtelMetricFloat64Val{}
	return nil
}

// RecordScalerActive create a measurement of the activity of the scaler
func (o *OtelMetrics) RecordScalerActive(namespace string, scaledResource string, scaler string, triggerIndex int, metric string, isScaledObject bool, active bool) {
	activeVal := 0
	if active {
		activeVal = 1
	}
	otelScalerActive := OtelMetricFloat64Val{}
	otelScalerActive.val = float64(activeVal)
	otelScalerActive.measurementOption = getScalerMeasurementOption(namespace, scaledResource, scaler, triggerIndex, metric, isScaledObject)
	otelScalerActiveVals = append(otelScalerActiveVals, otelScalerActive)
}

func PausedStatusCallback(_ context.Context, obsrv api.Float64Observer) error {
	for _, v := range otelScalerPauseVals {
		obsrv.Observe(v.val, v.measurementOption)
	}
	otelScalerPauseVals = []OtelMetricFloat64Val{}
	return nil
}

// RecordScaledObjectPaused marks whether the current ScaledObject is paused.
func (o *OtelMetrics) RecordScaledObjectPaused(namespace string, scaledObject string, active bool) {
	activeVal := 0
	if active {
		activeVal = 1
	}

	opt := api.WithAttributes(
		attribute.Key("namespace").String(namespace),
		attribute.Key("scaledObject").String(scaledObject))

	otelScalerPause := OtelMetricFloat64Val{}
	otelScalerPause.val = float64(activeVal)
	otelScalerPause.measurementOption = opt
	otelScalerPauseVals = append(otelScalerPauseVals, otelScalerPause)
}

// RecordScalerError counts the number of errors occurred in trying to get an external metric used by the HPA
func (o *OtelMetrics) RecordScalerError(namespace string, scaledResource string, scaler string, triggerIndex int, metric string, isScaledObject bool, err error) {
	if err != nil {
		otScalerErrorsCounter.Add(context.Background(), 1, getScalerMeasurementOption(namespace, scaledResource, scaler, triggerIndex, metric, isScaledObject))
		o.RecordScaledObjectError(namespace, scaledResource, err)
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

// RecordScaledJobError counts the number of errors with the scaled job
func (o *OtelMetrics) RecordScaledJobError(namespace string, scaledJob string, err error) {
	opt := api.WithAttributes(
		attribute.Key("namespace").String(namespace),
		attribute.Key("scaledJob").String(scaledJob))
	if err != nil {
		otScaledJobErrorsCounter.Add(context.Background(), 1, opt)
		return
	}
}

func (o *OtelMetrics) IncrementTriggerTotal(triggerType string) {
	if triggerType != "" {
		otTriggerTotalsCounterDeprecated.Add(context.Background(), 1, api.WithAttributes(attribute.Key("type").String(triggerType)))
		otTriggerRegisteredTotalsCounter.Add(context.Background(), 1, api.WithAttributes(attribute.Key("type").String(triggerType)))
	}
}

func (o *OtelMetrics) DecrementTriggerTotal(triggerType string) {
	if triggerType != "" {
		otTriggerTotalsCounterDeprecated.Add(context.Background(), -1, api.WithAttributes(attribute.Key("type").String(triggerType)))
		otTriggerRegisteredTotalsCounter.Add(context.Background(), -1, api.WithAttributes(attribute.Key("type").String(triggerType)))
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

	otCrdTotalsCounterDeprecated.Add(context.Background(), 1, opt)
	otCrdRegisteredTotalsCounter.Add(context.Background(), 1, opt)
}

func (o *OtelMetrics) DecrementCRDTotal(crdType, namespace string) {
	if namespace == "" {
		namespace = defaultNamespace
	}

	opt := api.WithAttributes(
		attribute.Key("namespace").String(namespace),
		attribute.Key("type").String(crdType),
	)
	otCrdTotalsCounterDeprecated.Add(context.Background(), -1, opt)
	otCrdRegisteredTotalsCounter.Add(context.Background(), -1, opt)
}

func getScalerMeasurementOption(namespace string, scaledResource string, scaler string, triggerIndex int, metric string, isScaledObject bool) api.MeasurementOption {
	if isScaledObject {
		return api.WithAttributes(
			attribute.Key("namespace").String(namespace),
			attribute.Key("scaledObject").String(scaledResource),
			attribute.Key("scaler").String(scaler),
			attribute.Key("scalerIndex").String(strconv.Itoa(triggerIndex)),
			attribute.Key("metric").String(metric),
		)
	}
	return api.WithAttributes(
		attribute.Key("namespace").String(namespace),
		attribute.Key("scaledJob").String(scaledResource),
		attribute.Key("scaler").String(scaler),
		attribute.Key("triggerIndex").String(strconv.Itoa(triggerIndex)),
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

// RecordCloudEventEmittedError counts the number of errors occurred in trying emit cloudevent
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
	for _, v := range otCloudEventQueueStatusVals {
		obsrv.Observe(v.val, v.measurementOption)
	}
	otCloudEventQueueStatusVals = []OtelMetricFloat64Val{}
	return nil
}

// RecordCloudEventQueueStatus record the number of cloudevents that are waiting for emitting
func (o *OtelMetrics) RecordCloudEventQueueStatus(namespace string, value int) {
	opt := api.WithAttributes(
		attribute.Key("namespace").String(namespace),
	)

	otCloudEventQueueStatus := OtelMetricFloat64Val{}
	otCloudEventQueueStatus.val = float64(value)
	otCloudEventQueueStatus.measurementOption = opt
	otCloudEventQueueStatusVals = append(otCloudEventQueueStatusVals, otCloudEventQueueStatus)
}

// RecordEmptyPrometheusMetricError counts the number of times a prometheus query returns an empty result
func (o *OtelMetrics) RecordEmptyPrometheusMetricError() {
	otEmptyPrometheusMetricError.Add(context.Background(), 1, nil)
}
