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
	assert.Equal(t, latency.Unit, "s")

	data := latency.Data.(metricdata.Gauge[float64]).DataPoints[0]
	assert.Equal(t, data.Value, float64(0.5))
}
