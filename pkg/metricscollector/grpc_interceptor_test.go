/*
Copyright 2026 The KEDA Authors

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
	"context"
	"io"
	"testing"
	"time"

	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

const (
	testGRPCFullMethod = "/externalscaler.ExternalScaler/GetMetrics"
	testGRPCMethod     = "externalscaler.ExternalScaler/GetMetrics"
)

type testGRPCClientStream struct {
	ctx       context.Context
	sendError error
	recvError error
}

func (s *testGRPCClientStream) Header() (metadata.MD, error) { return nil, nil }
func (s *testGRPCClientStream) Trailer() metadata.MD         { return nil }
func (s *testGRPCClientStream) CloseSend() error             { return nil }
func (s *testGRPCClientStream) Context() context.Context     { return s.ctx }
func (s *testGRPCClientStream) SendMsg(any) error            { return s.sendError }
func (s *testGRPCClientStream) RecvMsg(any) error            { return s.recvError }

func testGRPCRequestContext(ctx context.Context, suffix string) context.Context {
	return BuildScalerRequestCtx(ctx, scalersconfig.ScalerConfig{
		TriggerType:             "external-" + suffix,
		TriggerName:             "trigger-" + suffix,
		ScalableObjectNamespace: "namespace-" + suffix,
		ScalableObjectName:      "resource-" + suffix,
	}, "metric-"+suffix)
}

func newTestPromClientMetrics(t *testing.T, enableHighCardinalityLabels bool) (*grpcprom.ClientMetrics, *prometheus.Registry) {
	t.Helper()
	previousRegistry := ctrlmetrics.Registry
	registry := prometheus.NewRegistry()
	ctrlmetrics.Registry = registry
	t.Cleanup(func() {
		ctrlmetrics.Registry = previousRegistry
	})
	return newPromClientMetrics(enableHighCardinalityLabels), registry
}

func TestPromGRPCUnaryClientMetrics(t *testing.T) {
	clientMetrics, registry := newTestPromClientMetrics(t, true)
	ctx := testGRPCRequestContext(t.Context(), "unary")
	wantErr := status.Error(codes.Unavailable, "unavailable")
	err := clientMetrics.UnaryClientInterceptor(grpcprom.WithLabelsFromContext(grpcPromLabelsFromContext))(
		ctx,
		testGRPCFullMethod,
		nil,
		nil,
		nil,
		func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error {
			return wantErr
		},
	)
	assert.ErrorIs(t, err, wantErr)

	families, err := registry.Gather()
	require.NoError(t, err)
	labels := map[string]string{
		"grpc_type":       "unary",
		"grpc_service":    "externalscaler.ExternalScaler",
		"grpc_method":     "GetMetrics",
		"grpc_code":       codes.Unavailable.String(),
		"scaler":          "external-unary",
		"namespace":       "namespace-unary",
		"scaled_resource": "resource-unary",
		"trigger_name":    "trigger-unary",
		"metric_name":     "metric-unary",
	}
	handled := findPromMetric(families, "keda_grpc_client_handled_total", labels)
	require.NotNil(t, handled)
	assert.EqualValues(t, 1, handled.GetCounter().GetValue())

	delete(labels, "grpc_code")
	duration := findPromMetric(families, "keda_grpc_client_handling_seconds", labels)
	require.NotNil(t, duration)
	assert.EqualValues(t, 1, duration.GetHistogram().GetSampleCount())
}

func TestPromGRPCClientMetricsLimitContextLabels(t *testing.T) {
	clientMetrics, registry := newTestPromClientMetrics(t, false)
	ctx := testGRPCRequestContext(t.Context(), "low-cardinality")
	require.NoError(t, clientMetrics.UnaryClientInterceptor(grpcprom.WithLabelsFromContext(grpcPromLabelsFromContext))(
		ctx,
		testGRPCFullMethod,
		nil,
		nil,
		nil,
		func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error {
			return nil
		},
	))

	families, err := registry.Gather()
	require.NoError(t, err)
	metric := findPromMetric(families, "keda_grpc_client_handled_total", map[string]string{
		"grpc_method": "GetMetrics",
		"grpc_code":   codes.OK.String(),
		"scaler":      "external-low-cardinality",
	})
	require.NotNil(t, metric)
	for _, name := range []string{"namespace", "scaled_resource", "trigger_name", "metric_name"} {
		assert.Empty(t, prometheusLabelValue(metric, name), "label %s should be omitted", name)
	}
}

func TestPromGRPCStreamClientMetrics(t *testing.T) {
	clientMetrics, registry := newTestPromClientMetrics(t, true)
	ctx := testGRPCRequestContext(t.Context(), "stream")
	baseStream := &testGRPCClientStream{ctx: ctx}
	stream, err := clientMetrics.StreamClientInterceptor(grpcprom.WithLabelsFromContext(grpcPromLabelsFromContext))(
		ctx,
		&grpc.StreamDesc{ServerStreams: true},
		nil,
		"/externalscaler.ExternalScaler/StreamIsActive",
		func(context.Context, *grpc.StreamDesc, *grpc.ClientConn, string, ...grpc.CallOption) (grpc.ClientStream, error) {
			return baseStream, nil
		},
	)
	require.NoError(t, err)
	require.NoError(t, stream.SendMsg(struct{}{}))
	require.NoError(t, stream.RecvMsg(&struct{}{}))
	baseStream.recvError = io.EOF
	assert.ErrorIs(t, stream.RecvMsg(&struct{}{}), io.EOF)

	families, err := registry.Gather()
	require.NoError(t, err)
	labels := map[string]string{
		"grpc_type":       "server_stream",
		"grpc_service":    "externalscaler.ExternalScaler",
		"grpc_method":     "StreamIsActive",
		"scaler":          "external-stream",
		"namespace":       "namespace-stream",
		"scaled_resource": "resource-stream",
		"trigger_name":    "trigger-stream",
		"metric_name":     "metric-stream",
	}
	sent := findPromMetric(families, "keda_grpc_client_msg_sent_total", labels)
	require.NotNil(t, sent)
	assert.EqualValues(t, 1, sent.GetCounter().GetValue())
	received := findPromMetric(families, "keda_grpc_client_msg_received_total", labels)
	require.NotNil(t, received)
	// The upstream interceptor counts receive attempts, including the final EOF.
	assert.EqualValues(t, 2, received.GetCounter().GetValue())
}

func TestOtelGRPCStreamClientMetrics(t *testing.T) {
	resetTestOtel(true)
	handler := newOtelGRPCClientHandler(testOtel)
	ctx := testGRPCRequestContext(t.Context(), "otel")
	const streamMethod = "externalscaler.ExternalScaler/StreamIsActive"
	ctx = handler.TagRPC(ctx, &stats.RPCTagInfo{FullMethodName: "/" + streamMethod})
	beginTime := time.Now().Add(-200 * time.Millisecond)
	handler.HandleRPC(ctx, &stats.Begin{
		Client:         true,
		BeginTime:      beginTime,
		IsServerStream: true,
	})
	handler.HandleRPC(ctx, &stats.OutPayload{Client: true})
	handler.HandleRPC(ctx, &stats.InPayload{Client: true})
	handler.HandleRPC(ctx, &stats.InPayload{Client: true})
	handler.HandleRPC(ctx, &stats.End{Client: true, BeginTime: beginTime, EndTime: time.Now()})

	got := metricdata.ResourceMetrics{}
	require.NoError(t, testReader.Collect(context.Background(), &got))

	duration := findOtelMetric(got, "keda.rpc.client.call.duration")
	require.NotNil(t, duration)
	durationPoint := duration.Data.(metricdata.Histogram[float64]).DataPoints[0]
	assert.Equal(t, uint64(1), durationPoint.Count)
	assertOtelStringAttribute(t, durationPoint.Attributes, "scaler", "external-otel")
	assertOtelStringAttribute(t, durationPoint.Attributes, "namespace", "namespace-otel")

	callCount := findOtelMetric(got, "keda.rpc.client.call.count")
	require.NotNil(t, callCount)
	callPoint := callCount.Data.(metricdata.Sum[int64]).DataPoints[0]
	assert.Equal(t, int64(1), callPoint.Value)
	assertOtelStringAttribute(t, callPoint.Attributes, "rpc.system.name", "grpc")
	assertOtelStringAttribute(t, callPoint.Attributes, "rpc.method", streamMethod)
	assertOtelStringAttribute(t, callPoint.Attributes, "rpc.response.status_code", codes.OK.String())

	messageCount := findOtelMetric(got, "keda.rpc.client.stream.message.count")
	require.NotNil(t, messageCount)
	points := messageCount.Data.(metricdata.Sum[int64]).DataPoints
	require.Len(t, points, 2)
	counts := make(map[string]int64)
	for _, point := range points {
		assertOtelStringAttribute(t, point.Attributes, "rpc.method", streamMethod)
		assertOtelStringAttribute(t, point.Attributes, "namespace", "namespace-otel")
		direction, ok := point.Attributes.Value(attribute.Key("rpc.message.type"))
		require.True(t, ok)
		counts[direction.AsString()] = point.Value
	}
	assert.Equal(t, map[string]int64{"SENT": 1, "RECEIVED": 2}, counts)
}

func TestOtelGRPCUnaryStatusConsistency(t *testing.T) {
	for _, tc := range []struct {
		name string
		code codes.Code
		want string
	}{
		{name: "success", code: codes.OK, want: "OK"},
		{name: "cancelled", code: codes.Canceled, want: "CANCELLED"},
		{name: "deadline", code: codes.DeadlineExceeded, want: "DEADLINE_EXCEEDED"},
		{name: "unavailable", code: codes.Unavailable, want: "UNAVAILABLE"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetTestOtel(true)
			handler := newOtelGRPCClientHandler(testOtel)
			ctx := handler.TagRPC(testGRPCRequestContext(t.Context(), tc.name), &stats.RPCTagInfo{FullMethodName: testGRPCFullMethod})
			begin := time.Now()
			handler.HandleRPC(ctx, &stats.Begin{Client: true, BeginTime: begin})
			handler.HandleRPC(ctx, &stats.OutPayload{Client: true})
			if tc.code == codes.OK {
				handler.HandleRPC(ctx, &stats.InPayload{Client: true})
			}
			handler.HandleRPC(ctx, &stats.End{
				Client: true, BeginTime: begin, EndTime: begin.Add(200 * time.Millisecond),
				Error: status.Error(tc.code, tc.name),
			})

			var got metricdata.ResourceMetrics
			require.NoError(t, testReader.Collect(t.Context(), &got))
			duration := findOtelMetric(got, "keda.rpc.client.call.duration")
			require.NotNil(t, duration)
			durations := duration.Data.(metricdata.Histogram[float64]).DataPoints
			require.Len(t, durations, 1)
			assert.Equal(t, uint64(1), durations[0].Count)
			assert.InDelta(t, 0.2, durations[0].Sum, 0.000001)
			assertOtelStringAttribute(t, durations[0].Attributes, "rpc.response.status_code", tc.want)
			calls := findOtelMetric(got, "keda.rpc.client.call.count")
			require.NotNil(t, calls)
			counts := calls.Data.(metricdata.Sum[int64]).DataPoints
			require.Len(t, counts, 1)
			assert.Equal(t, int64(1), counts[0].Value)
			assertOtelStringAttribute(t, counts[0].Attributes, "rpc.response.status_code", tc.want)
			assert.Nil(t, findOtelMetric(got, "rpc.client.call.duration"), "the view must rename, not duplicate, the duration metric")
			assert.Nil(t, findOtelMetric(got, "keda.rpc.client.stream.message.count"), "unary payloads must not count as stream messages")
		})
	}
}

func TestOtelGRPCClientMetricsLimitContextAttributes(t *testing.T) {
	resetTestOtel(false)
	handler := newOtelGRPCClientHandler(testOtel)
	ctx := testGRPCRequestContext(t.Context(), "low-cardinality")
	ctx = handler.TagRPC(ctx, &stats.RPCTagInfo{FullMethodName: testGRPCFullMethod})
	now := time.Now()
	handler.HandleRPC(ctx, &stats.Begin{Client: true, BeginTime: now})
	handler.HandleRPC(ctx, &stats.End{Client: true, BeginTime: now, EndTime: now})

	got := metricdata.ResourceMetrics{}
	require.NoError(t, testReader.Collect(context.Background(), &got))
	callCount := findOtelMetric(got, "keda.rpc.client.call.count")
	require.NotNil(t, callCount)
	point := callCount.Data.(metricdata.Sum[int64]).DataPoints[0]
	for _, key := range []string{"namespace", "scaled_resource", "trigger_name", "metric_name"} {
		_, ok := point.Attributes.Value(attribute.Key(key))
		assert.False(t, ok, "attribute %s should be omitted", key)
	}
	assertOtelStringAttribute(t, point.Attributes, "scaler", "external-low-cardinality")
	duration := findOtelMetric(got, "keda.rpc.client.call.duration")
	require.NotNil(t, duration)
	durationPoint := duration.Data.(metricdata.Histogram[float64]).DataPoints[0]
	for _, key := range []string{"namespace", "scaled_resource", "trigger_name", "metric_name"} {
		_, ok := durationPoint.Attributes.Value(attribute.Key(key))
		assert.False(t, ok, "duration attribute %s should be omitted", key)
	}
	assertOtelStringAttribute(t, durationPoint.Attributes, "scaler", "external-low-cardinality")
}

func TestGRPCStatusCode(t *testing.T) {
	assert.Equal(t, "OK", grpcStatusCode(nil))
	assert.Equal(t, "OK", grpcStatusCode(io.EOF))
	assert.Equal(t, "CANCELLED", grpcStatusCode(context.Canceled))
	assert.Equal(t, "DEADLINE_EXCEEDED", grpcStatusCode(context.DeadlineExceeded))
	assert.Equal(t, "INVALID_ARGUMENT", grpcStatusCode(status.Error(codes.InvalidArgument, "invalid argument")))
	assert.Equal(t, "INTERNAL", grpcStatusCode(status.Error(codes.Internal, "internal")))
	assert.Equal(t, "CODE(99)", grpcStatusCode(status.Error(codes.Code(99), "unknown code")))
}

func findPromMetric(families []*dto.MetricFamily, name string, labels map[string]string) *dto.Metric {
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.GetMetric() {
			matches := true
			for label, value := range labels {
				if prometheusLabelValue(metric, label) != value {
					matches = false
					break
				}
			}
			if matches {
				return metric
			}
		}
	}
	return nil
}

func prometheusLabelValue(metric *dto.Metric, name string) string {
	for _, label := range metric.GetLabel() {
		if label.GetName() == name {
			return label.GetValue()
		}
	}
	return ""
}

func findOtelMetric(resourceMetrics metricdata.ResourceMetrics, name string) *metricdata.Metrics {
	for _, scopeMetrics := range resourceMetrics.ScopeMetrics {
		if metric := retrieveMetric(scopeMetrics.Metrics, name); metric != nil {
			return metric
		}
	}
	return nil
}
