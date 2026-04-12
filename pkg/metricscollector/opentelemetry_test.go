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

func TestRecordHTTPClientRequest(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		isError        bool
		wantStatusCode string
	}{
		{"200 success", 200, false, "200"},
		{"301 redirect", 301, false, "301"},
		{"404 client error", 404, false, "404"},
		{"503 server error", 503, false, "503"},
		{"transport error", 0, true, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testOtel.RecordHTTPClientRequest(0.1, tt.statusCode, tt.isError, "prometheus", "my-trigger", "my-metric", "default", "my-so")
			got := metricdata.ResourceMetrics{}
			err := testReader.Collect(context.Background(), &got)
			assert.Nil(t, err)

			scopeMetrics := got.ScopeMetrics[0]
			requestCount := retrieveMetric(scopeMetrics.Metrics, "keda.http.client.requests.count")
			assert.NotNil(t, requestCount)

			var found bool
			for _, dp := range requestCount.Data.(metricdata.Sum[int64]).DataPoints {
				code, ok := dp.Attributes.Value("status_code")
				if !ok || code.AsString() != tt.wantStatusCode {
					continue
				}
				scaler, _ := dp.Attributes.Value("scaler")
				assert.Equal(t, "prometheus", scaler.AsString())
				triggerName, _ := dp.Attributes.Value("trigger_name")
				assert.Equal(t, "my-trigger", triggerName.AsString())
				metricName, _ := dp.Attributes.Value("metric_name")
				assert.Equal(t, "my-metric", metricName.AsString())
				ns, _ := dp.Attributes.Value("namespace")
				assert.Equal(t, "default", ns.AsString())
				sr, _ := dp.Attributes.Value("scaled_resource")
				assert.Equal(t, "my-so", sr.AsString())
				found = true
				break
			}
			assert.True(t, found, "expected data point with status_code=%q", tt.wantStatusCode)

			requestDuration := retrieveMetric(scopeMetrics.Metrics, "keda.http.client.request.duration.seconds")
			assert.NotNil(t, requestDuration)
			assert.Equal(t, "s", requestDuration.Unit)
		})
	}
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

	var scaledObjectMetric metricdata.DataPoint[float64]
	for _, v := range dataPoints {
		attribute, _ := v.Attributes.Value("namespace")
		if attribute.AsString() == "testnamespace" {
			scaledObjectMetric = v
		}
	}

	assert.NotEqual(t, scaledObjectMetric, metricdata.DataPoint[float64]{})
	attribute, _ := scaledObjectMetric.Attributes.Value("scaledObject")
	assert.Equal(t, attribute.AsString(), "testresource")
	attribute, _ = scaledObjectMetric.Attributes.Value("scaler")
	assert.Equal(t, attribute.AsString(), "testscaler")
	attribute, _ = scaledObjectMetric.Attributes.Value("metric")
	assert.Equal(t, attribute.AsString(), "testmetric")
	assert.Equal(t, scaledObjectMetric.Value, 1.0)

	var scaledJobMetric metricdata.DataPoint[float64]
	for _, v := range dataPoints {
		attribute, _ := v.Attributes.Value("namespace")
		if attribute.AsString() == "testnamespace2" {
			scaledJobMetric = v
		}
	}

	assert.NotEqual(t, scaledJobMetric, metricdata.DataPoint[float64]{})
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
