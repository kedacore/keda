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
	"go.opentelemetry.io/otel/sdk/metric"
)

// ResetAdapterOtelPerformanceMetricsForTest clears adapter OTEL performance metrics state.
func ResetAdapterOtelPerformanceMetricsForTest() {
	adapterOtelMetrics = nil
}

// InitAdapterOtelPerformanceMetricsForTest initializes adapter OTEL performance metrics with a test reader.
func InitAdapterOtelPerformanceMetricsForTest(reader metric.Reader) {
	ResetAdapterOtelPerformanceMetricsForTest()
	initAdapterOtelPerformanceMetricsWithReader(reader)
}
