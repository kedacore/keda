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
	"fmt"
	"sync"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/internal/transform"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	mpb "go.opentelemetry.io/proto/otlp/metrics/v1"
)

// exporter exports metrics data as OTLP.
type exporter struct {
	// Ensure synchronous access to the client across all functionality.
	clientMu sync.Mutex
	client   Client

	shutdownOnce sync.Once
}

// Temporality returns the Temporality to use for an instrument kind.
func (e *exporter) Temporality(k metric.InstrumentKind) metricdata.Temporality {
	e.clientMu.Lock()
	defer e.clientMu.Unlock()
	return e.client.Temporality(k)
}

// Aggregation returns the Aggregation to use for an instrument kind.
func (e *exporter) Aggregation(k metric.InstrumentKind) aggregation.Aggregation {
	e.clientMu.Lock()
	defer e.clientMu.Unlock()
	return e.client.Aggregation(k)
}

// Export transforms and transmits metric data to an OTLP receiver.
func (e *exporter) Export(ctx context.Context, rm *metricdata.ResourceMetrics) error {
	otlpRm, err := transform.ResourceMetrics(rm)
	// Best effort upload of transformable metrics.
	e.clientMu.Lock()
	upErr := e.client.UploadMetrics(ctx, otlpRm)
	e.clientMu.Unlock()
	if upErr != nil {
		if err == nil {
			return fmt.Errorf("failed to upload metrics: %w", upErr)
		}
		// Merge the two errors.
		return fmt.Errorf("failed to upload incomplete metrics (%s): %w", err, upErr)
	}
	return err
}

// ForceFlush flushes any metric data held by an exporter.
func (e *exporter) ForceFlush(ctx context.Context) error {
	// The Exporter does not hold data, forward the command to the client.
	e.clientMu.Lock()
	defer e.clientMu.Unlock()
	return e.client.ForceFlush(ctx)
}

var errShutdown = fmt.Errorf("exporter is shutdown")

// Shutdown flushes all metric data held by an exporter and releases any held
// computational resources.
func (e *exporter) Shutdown(ctx context.Context) error {
	err := errShutdown
	e.shutdownOnce.Do(func() {
		e.clientMu.Lock()
		client := e.client
		e.client = shutdownClient{
			temporalitySelector: client.Temporality,
			aggregationSelector: client.Aggregation,
		}
		e.clientMu.Unlock()
		err = client.Shutdown(ctx)
	})
	return err
}

// New return an Exporter that uses client to transmits the OTLP data it
// produces. The client is assumed to be fully started and able to communicate
// with its OTLP receiving endpoint.
func New(client Client) metric.Exporter {
	return &exporter{client: client}
}

type shutdownClient struct {
	temporalitySelector metric.TemporalitySelector
	aggregationSelector metric.AggregationSelector
}

func (c shutdownClient) err(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return errShutdown
}

func (c shutdownClient) Temporality(k metric.InstrumentKind) metricdata.Temporality {
	return c.temporalitySelector(k)
}

func (c shutdownClient) Aggregation(k metric.InstrumentKind) aggregation.Aggregation {
	return c.aggregationSelector(k)
}

func (c shutdownClient) UploadMetrics(ctx context.Context, _ *mpb.ResourceMetrics) error {
	return c.err(ctx)
}

func (c shutdownClient) ForceFlush(ctx context.Context) error {
	return c.err(ctx)
}

func (c shutdownClient) Shutdown(ctx context.Context) error {
	return c.err(ctx)
}
