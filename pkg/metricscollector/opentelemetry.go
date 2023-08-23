package metricscollector

import (
	"context"
	"fmt"
	"runtime"
	"strconv"

	// "fmt"

	// "github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kedacore/keda/v2/version"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	// "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	api "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
)

var otLog = logf.Log.WithName("prometheus_server")

const meterName = "keda-open-telemetry-metrics"

var (
	meterProvider                 *metric.MeterProvider
	meter                         api.Meter
	otBuildInfo                   api.Int64Counter
	otScalerMetricsValueCounter   api.Float64UpDownCounter
	otScalerMetricsLatencyCounter api.Float64UpDownCounter
	otInternalLoopLatencyCounter  api.Float64UpDownCounter
	otScalerActiveCounter         api.Int64UpDownCounter
	otErrorsTotalCounter          api.Int64Counter
	otScalerErrorsCounter         api.Int64Counter
	otScaledObjectErrorsCounter   api.Int64Counter
	otTriggerTotalsCounter        api.Int64UpDownCounter
	otCrdTotalsCounter            api.Int64UpDownCounter
)

type OtelMetrics struct {
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
	initCounter()

	otel := &OtelMetrics{}
	otel.RecordBuildInfo()
	return otel
}

func initCounter() {
	var err error
	msg := "create opentelemetry counter failed"
	otBuildInfo, err = meter.Int64Counter("build.info", api.WithDescription("A metric with a constant '1' value labeled by version, git_commit and goversion from which KEDA was built."))
	if err != nil {
		otLog.Error(err, msg)
	}

	otErrorsTotalCounter, err = meter.Int64Counter("scaler.errors.total", api.WithDescription("Total number of errors for all scalers"))
	if err != nil {
		otLog.Error(err, msg)
	}

	otScalerErrorsCounter, err = meter.Int64Counter("scaler.errors", api.WithDescription("Number of scaler errors"))
	if err != nil {
		otLog.Error(err, msg)
	}

	otScaledObjectErrorsCounter, err = meter.Int64Counter("scaledobject.errors", api.WithDescription("Number of scaled object errors"))
	if err != nil {
		otLog.Error(err, msg)
	}

	otTriggerTotalsCounter, err = meter.Int64UpDownCounter("trigger.totals", api.WithDescription("Total triggers"))
	if err != nil {
		otLog.Error(err, msg)
	}

	otCrdTotalsCounter, err = meter.Int64UpDownCounter("resource.totals", api.WithDescription("Total resources"))
	if err != nil {
		otLog.Error(err, msg)
	}
}

func (o *OtelMetrics) RecordScalerMetric(namespace string, scaledObject string, scaler string, scalerIndex int, metric string, value float64) {
	cback := func(ctx context.Context, obsrv api.Float64Observer) error {
		obsrv.Observe(value, getScalerMeasurementOption(namespace, scaledObject, scaler, scalerIndex, metric))
		return nil
	}
	_, err := meter.Float64ObservableGauge(
		"scaler.metrics.value",
		api.WithDescription("Metric Value used for HPA"),
		api.WithFloat64Callback(cback),
	)
	if err != nil {
		fmt.Println("failed to register scaler metrics value")
	}
}

// RecordScalerLatency create a measurement of the latency to external metric
func (o *OtelMetrics) RecordScalerLatency(namespace string, scaledObject string, scaler string, scalerIndex int, metric string, value float64) {
	cback := func(ctx context.Context, obsrv api.Float64Observer) error {
		obsrv.Observe(value, getScalerMeasurementOption(namespace, scaledObject, scaler, scalerIndex, metric))
		return nil
	}
	_, err := meter.Float64ObservableGauge(
		"scaler.metrics.latency",
		api.WithDescription("Scaler Metrics Latency"),
		api.WithFloat64Callback(cback),
	)
	if err != nil {
		fmt.Println("failed to register scaler metrics latency")
	}
}

// RecordScaledObjectLatency create a measurement of the latency executing scalable object loop
func (o *OtelMetrics) RecordScalableObjectLatency(namespace string, name string, isScaledObject bool, value float64) {
	resourceType := "scaledjob"
	if isScaledObject {
		resourceType = "scaledobject"
	}

	opt := api.WithAttributes(
		attribute.Key("namespace").String(namespace),
		attribute.Key("type").String(resourceType),
		attribute.Key("name").String(name))

	cback := func(ctx context.Context, obsrv api.Float64Observer) error {
		obsrv.Observe(value, opt)
		return nil
	}
	_, err := meter.Float64ObservableGauge(
		"internal.scale.loop.latency",
		api.WithDescription("Internal latency of ScaledObject/ScaledJob loop execution"),
		api.WithFloat64Callback(cback),
	)
	if err != nil {
		fmt.Println("failed to register internal scale loop latency")
	}
}

// RecordScalerActive create a measurement of the activity of the scaler
func (o *OtelMetrics) RecordScalerActive(namespace string, scaledObject string, scaler string, scalerIndex int, metric string, active bool) {
	activeVal := -1
	if active {
		activeVal = 1
	}

	cback := func(ctx context.Context, obsrv api.Float64Observer) error {
		obsrv.Observe(float64(activeVal), getScalerMeasurementOption(namespace, scaledObject, scaler, scalerIndex, metric))
		return nil
	}
	_, err := meter.Float64ObservableGauge(
		"scaler.active",
		api.WithDescription("Activity of a Scaler Metric"),
		api.WithFloat64Callback(cback),
	)
	if err != nil {
		fmt.Println("failed to register internal scale loop latency")
	}
}

// RecordScalerError counts the number of errors occurred in trying get an external metric used by the HPA
func (o *OtelMetrics) RecordScalerError(namespace string, scaledObject string, scaler string, scalerIndex int, metric string, err error) {
	if err != nil {
		otScalerErrorsCounter.Add(context.Background(), 1, getScalerMeasurementOption(namespace, scaledObject, scaler, scalerIndex, metric))
		o.RecordScaledObjectError(namespace, scaledObject, err)
		otErrorsTotalCounter.Add(context.Background(), 1)
		return
	}
}

// RecordScaleObjectError counts the number of errors with the scaled object
func (o *OtelMetrics) RecordScaledObjectError(namespace string, scaledObject string, err error) {
	opt := api.WithAttributes(
		attribute.Key("namespace").String(namespace),
		attribute.Key("scaledObject").String(scaledObject))
	if err != nil {
		otScaledObjectErrorsCounter.Add(context.Background(), 1, opt)
		return
	}
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
	cback := func(ctx context.Context, obsrv api.Int64Observer) error {
		obsrv.Observe(1, opt)
		return nil
	}
	_, err := meter.Int64ObservableGauge(
		"build.info",
		api.WithDescription("A metric with a constant '1' value labeled by version, git_commit and goversion from which KEDA was built."),
		api.WithInt64Callback(cback),
	)
	if err != nil {
		fmt.Println("failed to register scaler metrics value")
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
		namespace = "default"
	}
	opt := api.WithAttributes(
		attribute.Key("namespace").String(namespace),
		attribute.Key("type").String(crdType),
	)

	otCrdTotalsCounter.Add(context.Background(), 1, opt)
}

func (o *OtelMetrics) DecrementCRDTotal(crdType, namespace string) {
	if namespace == "" {
		namespace = "default"
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
