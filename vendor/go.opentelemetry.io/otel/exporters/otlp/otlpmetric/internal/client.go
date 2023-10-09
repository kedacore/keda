// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal // import "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/internal"

import (
	"context"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	mpb "go.opentelemetry.io/proto/otlp/metrics/v1"
)

// Client handles the transmission of OTLP data to an OTLP receiving endpoint.
type Client interface {
	// Temporality returns the Temporality to use for an instrument kind.
	Temporality(metric.InstrumentKind) metricdata.Temporality

	// Aggregation returns the Aggregation to use for an instrument kind.
	Aggregation(metric.InstrumentKind) aggregation.Aggregation

	// UploadMetrics transmits metric data to an OTLP receiver.
	//
	// All retry logic must be handled by UploadMetrics alone, the Exporter
	// does not implement any retry logic. All returned errors are considered
	// unrecoverable.
	UploadMetrics(context.Context, *mpb.ResourceMetrics) error

	// ForceFlush flushes any metric data held by an Client.
	//
	// The deadline or cancellation of the passed context must be honored. An
	// appropriate error should be returned in these situations.
	ForceFlush(context.Context) error

	// Shutdown flushes all metric data held by a Client and closes any
	// connections it holds open.
	//
	// The deadline or cancellation of the passed context must be honored. An
	// appropriate error should be returned in these situations.
	//
	// Shutdown will only be called once by the Exporter. Once a return value
	// is received by the Exporter from Shutdown the Client will not be used
	// anymore. Therefore all computational resources need to be released
	// after this is called so the Client can be garbage collected.
	Shutdown(context.Context) error
}
