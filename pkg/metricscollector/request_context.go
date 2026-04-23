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

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

// BuildScalerRequestCtx attaches scaler metadata used by HTTP client
// instrumentation to the outbound request context.
func BuildScalerRequestCtx(ctx context.Context, config scalersconfig.ScalerConfig, metricName string) context.Context {
	requestCtx := context.WithValue(ctx, ScalerContextKey, config.TriggerType)
	requestCtx = context.WithValue(requestCtx, TriggerNameContextKey, config.TriggerName)
	requestCtx = context.WithValue(requestCtx, MetricNameContextKey, metricName)
	requestCtx = context.WithValue(requestCtx, NamespaceContextKey, config.ScalableObjectNamespace)
	requestCtx = context.WithValue(requestCtx, ScaledResourceContextKey, config.ScalableObjectName)
	return requestCtx
}
