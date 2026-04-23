package metricscollector_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/kedacore/keda/v2/pkg/metricscollector"
)

var (
	testOtel   *metricscollector.OtelMetrics
	testReader metric.Reader
)

func init() {
	testReader = metric.NewManualReader()
	options := metric.WithReader(testReader)
	testOtel = metricscollector.NewOtelMetrics(options)
}

func retrieveMetric(metrics []metricdata.Metrics, metricname string) *metricdata.Metrics {
	for _, m := range metrics {
		if m.Name == metricname {
			return &m
		}
	}
	return nil
}

func retrieveMetricFromScopes(scopeMetrics []metricdata.ScopeMetrics, metricname string) *metricdata.Metrics {
	for _, scopeMetric := range scopeMetrics {
		if metric := retrieveMetric(scopeMetric.Metrics, metricname); metric != nil {
			return metric
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

type mockRoundTripper struct {
	statusCode int
}

func (m mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: m.statusCode,
		Body:       io.NopCloser(strings.NewReader("")),
		Request:    req,
	}, nil
}

func TestHTTPClientDurationMetric(t *testing.T) {
	metricscollector.ConfigureHTTPClientMetricsInstrumentation(false, true)
	reader := metric.NewManualReader()
	metricscollector.NewOtelMetrics(metric.WithReader(reader))
	defer func() {
		testReader = metric.NewManualReader()
		testOtel = metricscollector.NewOtelMetrics(metric.WithReader(testReader))
	}()

	ctx := context.Background()
	ctx = context.WithValue(ctx, metricscollector.ScalerContextKey, "prometheus")
	ctx = context.WithValue(ctx, metricscollector.TriggerNameContextKey, "my-trigger")
	ctx = context.WithValue(ctx, metricscollector.MetricNameContextKey, "my-metric")
	ctx = context.WithValue(ctx, metricscollector.NamespaceContextKey, "default")
	ctx = context.WithValue(ctx, metricscollector.ScaledResourceContextKey, "my-so")

	labeler := &otelhttp.Labeler{}
	labeler.Add(
		attribute.String("scaler", "prometheus"),
		attribute.String("trigger_name", "my-trigger"),
		attribute.String("metric_name", "my-metric"),
		attribute.String("namespace", "default"),
		attribute.String("scaled_resource", "my-so"),
	)
	ctx = otelhttp.ContextWithLabeler(ctx, labeler)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com", nil)
	assert.NoError(t, err)

	rt := metricscollector.NewInstrumentedRoundTripper(mockRoundTripper{statusCode: http.StatusOK})
	resp, err := rt.RoundTrip(req)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())

	got := metricdata.ResourceMetrics{}
	err = reader.Collect(context.Background(), &got)
	assert.Nil(t, err)

	requestDuration := retrieveMetricFromScopes(got.ScopeMetrics, "http.client.request.duration")
	assert.NotNil(t, requestDuration)
	assert.Equal(t, "s", requestDuration.Unit)

	var found bool
	for _, dp := range requestDuration.Data.(metricdata.Histogram[float64]).DataPoints {
		scaler, ok := dp.Attributes.Value("scaler")
		if !ok || scaler.AsString() != "prometheus" {
			continue
		}
		triggerName, _ := dp.Attributes.Value("trigger_name")
		metricName, _ := dp.Attributes.Value("metric_name")
		namespace, _ := dp.Attributes.Value("namespace")
		scaledResource, _ := dp.Attributes.Value("scaled_resource")
		statusCode, ok := dp.Attributes.Value("http.response.status_code")
		assert.True(t, ok)

		assert.Equal(t, "my-trigger", triggerName.AsString())
		assert.Equal(t, "my-metric", metricName.AsString())
		assert.Equal(t, "default", namespace.AsString())
		assert.Equal(t, "my-so", scaledResource.AsString())
		assert.Equal(t, int64(http.StatusOK), statusCode.AsInt64())
		assert.Greater(t, dp.Count, uint64(0))
		found = true
		break
	}

	assert.True(t, found, "expected http.client.request.duration datapoint with scaler labels")
}

func TestHTTPClientDurationMetricDisabled(t *testing.T) {
	metricscollector.ConfigureHTTPClientMetricsInstrumentation(false, false)

	reader := metric.NewManualReader()
	metricscollector.NewOtelMetrics(metric.WithReader(reader))
	defer func() {
		testReader = metric.NewManualReader()
		testOtel = metricscollector.NewOtelMetrics(metric.WithReader(testReader))
	}()

	ctx := context.Background()
	ctx = context.WithValue(ctx, metricscollector.ScalerContextKey, "prometheus")
	ctx = context.WithValue(ctx, metricscollector.TriggerNameContextKey, "my-trigger")
	ctx = context.WithValue(ctx, metricscollector.MetricNameContextKey, "my-metric")
	ctx = context.WithValue(ctx, metricscollector.NamespaceContextKey, "default")
	ctx = context.WithValue(ctx, metricscollector.ScaledResourceContextKey, "my-so")

	labeler := &otelhttp.Labeler{}
	labeler.Add(
		attribute.String("scaler", "prometheus"),
		attribute.String("trigger_name", "my-trigger"),
		attribute.String("metric_name", "my-metric"),
		attribute.String("namespace", "default"),
		attribute.String("scaled_resource", "my-so"),
	)
	ctx = otelhttp.ContextWithLabeler(ctx, labeler)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com", nil)
	assert.NoError(t, err)

	rt := metricscollector.NewInstrumentedRoundTripper(mockRoundTripper{statusCode: http.StatusOK})
	resp, err := rt.RoundTrip(req)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())

	got := metricdata.ResourceMetrics{}
	err = reader.Collect(context.Background(), &got)
	assert.Nil(t, err)

	requestDuration := retrieveMetricFromScopes(got.ScopeMetrics, "http.client.request.duration")
	assert.Nil(t, requestDuration)
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

	assert.NotNil(t, activeMetric)

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
