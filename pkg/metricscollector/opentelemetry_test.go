package metricscollector

import (
	"context"
	"testing"

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
	buildInfo := retrieveMetric(scopeMetrics.Metrics, "build.info")

	assert.NotNil(t, buildInfo)

	data := buildInfo.Data.(metricdata.Sum[int64]).DataPoints[0]

	assert.True(t, data.Attributes.HasValue("version"))
	assert.True(t, data.Attributes.HasValue("GitCommit"))
	assert.True(t, data.Attributes.HasValue("runtion.version"))
	assert.True(t, data.Attributes.HasValue("runtime.GOOS"))
	assert.True(t, data.Attributes.HasValue("runtime.GOARCH"))

	assert.Equal(t, data.Value, int64(1))
}

func TestIncrementTriggerTotal(t *testing.T) {
	testOtel.IncrementTriggerTotal("testtrigger")
	got := metricdata.ResourceMetrics{}
	err := testReader.Collect(context.Background(), &got)

	assert.Nil(t, err)
	scopeMetrics := got.ScopeMetrics[0]
	assert.NotEqual(t, len(scopeMetrics.Metrics), 0)
	buildInfo := retrieveMetric(scopeMetrics.Metrics, "trigger.totals")

	assert.NotNil(t, buildInfo)

	data := buildInfo.Data.(metricdata.Sum[int64]).DataPoints[0]
	assert.Equal(t, data.Value, int64(1))

	testOtel.DecrementTriggerTotal("testtrigger")
	got = metricdata.ResourceMetrics{}
	err = testReader.Collect(context.Background(), &got)
	assert.Nil(t, err)
	scopeMetrics = got.ScopeMetrics[0]
	assert.NotEqual(t, len(scopeMetrics.Metrics), 0)
	buildInfo = retrieveMetric(scopeMetrics.Metrics, "trigger.totals")

	assert.NotNil(t, buildInfo)

	data = buildInfo.Data.(metricdata.Sum[int64]).DataPoints[0]
	assert.Equal(t, data.Value, int64(0))

}
