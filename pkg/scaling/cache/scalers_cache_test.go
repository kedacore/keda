/*
Copyright 2025 The KEDA Authors

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

package cache

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/kedacore/keda/v2/pkg/metricscollector"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

func TestBuildScalerRequestCtxWithOtelEnabled(t *testing.T) {
	RegisterTestingT(t)
	metricscollector.ConfigureHTTPClientMetricsInstrumentation(false, true)

	sb := ScalerBuilder{
		ScalerConfig: scalersconfig.ScalerConfig{
			TriggerType:             "prometheus",
			TriggerName:             "my-trigger",
			ScalableObjectNamespace: "my-namespace",
			ScalableObjectName:      "my-scaled-object",
		},
	}

	ctx := metricscollector.BuildScalerRequestCtx(context.Background(), sb.ScalerConfig, "my-metric")

	Expect(ctx.Value(metricscollector.ScalerContextKey)).To(Equal("prometheus"))
	Expect(ctx.Value(metricscollector.TriggerNameContextKey)).To(Equal("my-trigger"))
	Expect(ctx.Value(metricscollector.MetricNameContextKey)).To(Equal("my-metric"))
	Expect(ctx.Value(metricscollector.NamespaceContextKey)).To(Equal("my-namespace"))
	Expect(ctx.Value(metricscollector.ScaledResourceContextKey)).To(Equal("my-scaled-object"))

	labeler, ok := otelhttp.LabelerFromContext(ctx)
	Expect(ok).To(BeTrue())
	Expect(labeler.Get()).To(HaveLen(5))
}

func TestBuildScalerRequestCtxWithOtelDisabled(t *testing.T) {
	RegisterTestingT(t)
	metricscollector.ConfigureHTTPClientMetricsInstrumentation(false, false)

	sb := ScalerBuilder{
		ScalerConfig: scalersconfig.ScalerConfig{
			TriggerType:             "prometheus",
			TriggerName:             "my-trigger",
			ScalableObjectNamespace: "my-namespace",
			ScalableObjectName:      "my-scaled-object",
		},
	}

	ctx := metricscollector.BuildScalerRequestCtx(context.Background(), sb.ScalerConfig, "my-metric")

	Expect(ctx.Value(metricscollector.ScalerContextKey)).To(Equal("prometheus"))
	Expect(ctx.Value(metricscollector.TriggerNameContextKey)).To(Equal("my-trigger"))
	Expect(ctx.Value(metricscollector.MetricNameContextKey)).To(Equal("my-metric"))
	Expect(ctx.Value(metricscollector.NamespaceContextKey)).To(Equal("my-namespace"))
	Expect(ctx.Value(metricscollector.ScaledResourceContextKey)).To(Equal("my-scaled-object"))

	_, ok := otelhttp.LabelerFromContext(ctx)
	Expect(ok).To(BeFalse())
}

func TestEmptyScalersCache(t *testing.T) {
	RegisterTestingT(t)

	cache := &ScalersCache{
		Scalers: make([]ScalerBuilder, 0),
	}
	go func() {
		scalers, configs := cache.GetScalers()
		Expect(scalers).To(BeEmpty())
		Expect(configs).To(BeEmpty())
	}()

	go func() {
		pushScalers := cache.GetPushScalers()
		Expect(pushScalers).To(BeEmpty())
	}()

	go func() {
		metrics := cache.GetMetricSpecForScaling(context.Background())
		Expect(metrics).To(BeEmpty())
	}()

	go func() {
		metrics, err := cache.GetMetricSpecForScalingForScaler(context.Background(), 0)
		Expect(err).To(Not(BeNil()))
		Expect(metrics).To(BeNil())
	}()

	go func() {
		cache.Close(context.Background())
	}()
}
