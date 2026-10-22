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
	"errors"
	"io"
	"strconv"
	"strings"

	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/atomic"
	codepb "google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"
)

const (
	grpcStreamMessageSent     = "SENT"
	grpcStreamMessageReceived = "RECEIVED"
)

type scalerRequestLabels struct {
	scaler         string
	triggerName    string
	metricName     string
	namespace      string
	scaledResource string
}

// GRPCClientDialOptions returns the enabled Prometheus and OpenTelemetry client
// instrumentation for external scaler connections.
func GRPCClientDialOptions() []grpc.DialOption {
	var options []grpc.DialOption
	if promClientMetrics != nil {
		labelsOption := grpcprom.WithLabelsFromContext(grpcPromLabelsFromContext)
		options = append(options,
			grpc.WithChainUnaryInterceptor(promClientMetrics.UnaryClientInterceptor(labelsOption)),
			grpc.WithChainStreamInterceptor(promClientMetrics.StreamClientInterceptor(labelsOption)),
		)
	}
	if otelClientHandler != nil {
		options = append(options, grpc.WithStatsHandler(otelClientHandler))
	}
	return options
}

func newOtelGRPCClientHandler(metrics *OtelMetrics) stats.Handler {
	upstream := otelgrpc.NewClientHandler(
		otelgrpc.WithMeterProvider(meterProvider),
		otelgrpc.WithMetricAttributesFn(metrics.grpcClientAttributesFromContext),
	)
	return &otelGRPCClientHandler{
		Handler: upstream,
		metrics: metrics,
	}
}

type otelGRPCClientHandler struct {
	stats.Handler
	metrics *OtelMetrics
}

type grpcClientRPCStateKey struct{}

type grpcClientRPCState struct {
	labels   scalerRequestLabels
	method   string
	isStream atomic.Bool
}

func (h *otelGRPCClientHandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	ctx = h.Handler.TagRPC(ctx, info)
	labels, ok := scalerRequestLabelsFromContext(ctx)
	if !ok {
		return ctx
	}
	return context.WithValue(ctx, grpcClientRPCStateKey{}, &grpcClientRPCState{
		labels: labels,
		method: strings.TrimPrefix(info.FullMethodName, "/"),
	})
}

func (h *otelGRPCClientHandler) HandleRPC(ctx context.Context, rpcStats stats.RPCStats) {
	h.Handler.HandleRPC(ctx, rpcStats)

	state, ok := ctx.Value(grpcClientRPCStateKey{}).(*grpcClientRPCState)
	if !ok {
		return
	}

	switch rpcStats := rpcStats.(type) {
	case *stats.Begin:
		state.isStream.Store(rpcStats.IsClientStream || rpcStats.IsServerStream)
	case *stats.End:
		h.metrics.RecordGRPCClientCall(ctx, grpcStatusCode(rpcStats.Error), state.method, state.labels)
	case *stats.InPayload:
		if state.isStream.Load() {
			h.metrics.RecordGRPCClientStreamMessage(ctx, grpcStreamMessageReceived, state.method, state.labels)
		}
	case *stats.OutPayload:
		if state.isStream.Load() {
			h.metrics.RecordGRPCClientStreamMessage(ctx, grpcStreamMessageSent, state.method, state.labels)
		}
	}
}

func scalerRequestLabelsFromContext(ctx context.Context) (scalerRequestLabels, bool) {
	scaler, scalerOK := ctx.Value(ScalerContextKey).(string)
	triggerName, triggerOK := ctx.Value(TriggerNameContextKey).(string)
	metricName, metricOK := ctx.Value(MetricNameContextKey).(string)
	namespace, namespaceOK := ctx.Value(NamespaceContextKey).(string)
	scaledResource, scaledResourceOK := ctx.Value(ScaledResourceContextKey).(string)
	if !scalerOK || !triggerOK || !metricOK || !namespaceOK || !scaledResourceOK {
		return scalerRequestLabels{}, false
	}

	return scalerRequestLabels{
		scaler:         scaler,
		triggerName:    triggerName,
		metricName:     metricName,
		namespace:      namespace,
		scaledResource: scaledResource,
	}, true
}

func grpcStatusCode(err error) string {
	var code codes.Code
	switch {
	case err == nil, errors.Is(err, io.EOF):
		code = codes.OK
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		code = status.FromContextError(err).Code()
	default:
		code = status.Code(err)
	}

	if canonical, ok := codepb.Code_name[int32(code)]; ok {
		return canonical
	}
	return "CODE(" + strconv.FormatInt(int64(code), 10) + ")"
}
