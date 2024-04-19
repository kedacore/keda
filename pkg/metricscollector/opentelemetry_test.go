package metricscollector

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

var (
	testOtel   *OtelMetrics
	testReader metric.Reader
)

func init() {
	testReader = metric.NewManualReader()
	options := metric.WithReader(testReader)
	testOtel = NewOtelMetrics(options)
}

func retrieveMetric(metrics []metricdata.Metrics, metricname string) *metricdata.Metrics {
	for _, m := range metrics {
		if m.Name == metricname {
			return &m
		}
	}
	return nil
}

func TestBuildInfo(t *testing.T) {
	got := metricdata.ResourceMetrics{}
	err := testReader.Collect(context.Background(), &got)

	assert.Nil(t, err)
	scopeMetrics := got.ScopeMetrics[0]
	assert.NotEqual(t, len(scopeMetrics.Metrics), 0)
	buildInfo := retrieveMetric(scopeMetrics.Metrics, "keda.build.info")

	assert.NotNil(t, buildInfo)

	data := buildInfo.Data.(metricdata.Gauge[int64]).DataPoints[0]

	assert.True(t, data.Attributes.HasValue("version"))
	assert.True(t, data.Attributes.HasValue("git_commit"))
	assert.True(t, data.Attributes.HasValue("version"))
	assert.True(t, data.Attributes.HasValue("goos"))
	assert.True(t, data.Attributes.HasValue("goarch"))

	assert.Equal(t, data.Value, int64(1))
}

func TestIncrementTriggerTotal(t *testing.T) {
	testOtel.IncrementTriggerTotal("testtrigger")
	got := metricdata.ResourceMetrics{}
	err := testReader.Collect(context.Background(), &got)

	assert.Nil(t, err)
	scopeMetrics := got.ScopeMetrics[0]
	assert.NotEqual(t, len(scopeMetrics.Metrics), 0)
	triggercount := retrieveMetric(scopeMetrics.Metrics, "keda.trigger.registered.count")

	assert.NotNil(t, triggercount)

	data := triggercount.Data.(metricdata.Sum[int64]).DataPoints[0]
	assert.Equal(t, data.Value, int64(1))

	testOtel.DecrementTriggerTotal("testtrigger")
	got = metricdata.ResourceMetrics{}
	err = testReader.Collect(context.Background(), &got)
	assert.Nil(t, err)
	scopeMetrics = got.ScopeMetrics[0]
	assert.NotEqual(t, len(scopeMetrics.Metrics), 0)
	triggercount = retrieveMetric(scopeMetrics.Metrics, "keda.trigger.registered.count")

	assert.NotNil(t, triggercount)

	data = triggercount.Data.(metricdata.Sum[int64]).DataPoints[0]
	assert.Equal(t, data.Value, int64(0))
}

func TestLoopLatency(t *testing.T) {
	testOtel.RecordScalableObjectLatency("namespace", "name", true, 500*time.Millisecond)
	got := metricdata.ResourceMetrics{}
	err := testReader.Collect(context.Background(), &got)

	assert.Nil(t, err)
	scopeMetrics := got.ScopeMetrics[0]
	assert.NotEqual(t, len(scopeMetrics.Metrics), 0)

	latency := retrieveMetric(scopeMetrics.Metrics, "keda.internal.scale.loop.latency")
	assert.NotNil(t, latency)
	assert.Equal(t, latency.Unit, "")
	data := latency.Data.(metricdata.Gauge[float64]).DataPoints[0]
	assert.Equal(t, data.Value, float64(500))

	latencySeconds := retrieveMetric(scopeMetrics.Metrics, "keda.internal.scale.loop.latency.seconds")
	assert.NotNil(t, latencySeconds)
	assert.Equal(t, latencySeconds.Unit, "s")
	data = latencySeconds.Data.(metricdata.Gauge[float64]).DataPoints[0]
	assert.Equal(t, data.Value, float64(0.5))
}

func TestContinuousMetrics(t *testing.T) {
	testOtel.RecordScalerActive("testnamespace", "testresource", "testscaler", 0, "testmetric", true, true)
	testOtel.RecordScalerActive("testnamespace2", "testresource2", "testscaler2", 0, "testmetric", false, false)
	got := metricdata.ResourceMetrics{}
	err := testReader.Collect(context.Background(), &got)

	assert.Nil(t, err)
	scopeMetrics := got.ScopeMetrics[0]
	assert.NotEqual(t, len(scopeMetrics.Metrics), 0)
	activeMetric := retrieveMetric(scopeMetrics.Metrics, "keda.scaler.active")

	assert.NotNil(t, buildInfo)

	dataPoints := activeMetric.Data.(metricdata.Gauge[float64]).DataPoints
	assert.Len(t, dataPoints, 2)

	scaledObjectMetric := dataPoints[0]
	attribute, _ := scaledObjectMetric.Attributes.Value("namespace")
	assert.Equal(t, attribute.AsString(), "testnamespace")
	attribute, _ = scaledObjectMetric.Attributes.Value("scaledObject")
	assert.Equal(t, attribute.AsString(), "testresource")
	attribute, _ = scaledObjectMetric.Attributes.Value("scaler")
	assert.Equal(t, attribute.AsString(), "testscaler")
	attribute, _ = scaledObjectMetric.Attributes.Value("metric")
	assert.Equal(t, attribute.AsString(), "testmetric")
	assert.Equal(t, scaledObjectMetric.Value, 1.0)

	scaledJobMetric := dataPoints[1]
	attribute, _ = scaledJobMetric.Attributes.Value("namespace")
	assert.Equal(t, attribute.AsString(), "testnamespace2")
	attribute, _ = scaledJobMetric.Attributes.Value("scaledJob")
	assert.Equal(t, attribute.AsString(), "testresource2")
	attribute, _ = scaledJobMetric.Attributes.Value("scaler")
	assert.Equal(t, attribute.AsString(), "testscaler2")
	attribute, _ = scaledJobMetric.Attributes.Value("metric")
	assert.Equal(t, attribute.AsString(), "testmetric")
	assert.Equal(t, scaledJobMetric.Value, 0.0)
}
