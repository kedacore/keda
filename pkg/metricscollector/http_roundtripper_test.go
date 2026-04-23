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
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

type mockRoundTripper struct {
	resp *http.Response
	err  error
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.resp != nil && m.resp.Request == nil {
		m.resp.Request = req
	}

	return m.resp, m.err
}

func fakeResponse(statusCode int) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader("")),
	}
}

func withPromCollector(t *testing.T) {
	t.Helper()

	previousCollectors := collectors
	collectors = []MetricsCollector{&PromMetrics{}}
	t.Cleanup(func() {
		collectors = previousCollectors
	})
}

func TestInstrumentedRoundTripper_RecordsSuccessfulResponses(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
	}{
		{name: "2xx response", statusCode: http.StatusOK},
		{name: "4xx response", statusCode: http.StatusBadRequest},
		{name: "5xx response", statusCode: http.StatusInternalServerError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			RegisterTestingT(t)
			withPromCollector(t)

			scalerName := fmt.Sprintf("prometheus-%d", tc.statusCode)
			triggerName := fmt.Sprintf("trigger-%d", tc.statusCode)
			metricName := fmt.Sprintf("metric-%d", tc.statusCode)
			scaledResource := fmt.Sprintf("so-%d", tc.statusCode)

			rt := NewInstrumentedRoundTripper(&mockRoundTripper{resp: fakeResponse(tc.statusCode)})

			ctx := context.Background()
			ctx = context.WithValue(ctx, ScalerContextKey, scalerName)
			ctx = context.WithValue(ctx, TriggerNameContextKey, triggerName)
			ctx = context.WithValue(ctx, MetricNameContextKey, metricName)
			ctx = context.WithValue(ctx, NamespaceContextKey, "default")
			ctx = context.WithValue(ctx, ScaledResourceContextKey, scaledResource)

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com", nil)
			Expect(err).To(BeNil())

			resp, err := rt.RoundTrip(req)
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())

			counterValue, err := httpClientRequestsTotal.
				GetMetricWithLabelValues("default", scaledResource, scalerName, triggerName, metricName, fmt.Sprintf("%d", tc.statusCode))
			Expect(err).To(BeNil())
			Expect(counterValue).NotTo(BeNil())
			m := &dto.Metric{}
			err = counterValue.Write(m)
			Expect(err).To(BeNil())
			Expect(m.Counter.GetValue()).To(BeNumerically("==", 1))

			durationHistogram, err := httpClientRequestDuration.
				GetMetricWithLabelValues(scalerName, fmt.Sprintf("%d", tc.statusCode))
			Expect(err).To(BeNil())
			Expect(durationHistogram).NotTo(BeNil())
			err = durationHistogram.(prometheus.Metric).Write(m)
			Expect(err).To(BeNil())
			Expect(m.Histogram.GetSampleCount()).To(BeNumerically("==", 1))
		})
	}
}

func TestInstrumentedRoundTripper_RecordsTransportErrors(t *testing.T) {
	RegisterTestingT(t)
	withPromCollector(t)

	scalerName := "prometheus-transport-error"
	triggerName := "trigger-transport-error"
	metricName := "metric-transport-error"
	scaledResource := "so-transport-error"

	transportErr := io.ErrUnexpectedEOF
	rt := NewInstrumentedRoundTripper(&mockRoundTripper{err: transportErr})

	ctx := context.Background()
	ctx = context.WithValue(ctx, ScalerContextKey, scalerName)
	ctx = context.WithValue(ctx, TriggerNameContextKey, triggerName)
	ctx = context.WithValue(ctx, MetricNameContextKey, metricName)
	ctx = context.WithValue(ctx, NamespaceContextKey, "default")
	ctx = context.WithValue(ctx, ScaledResourceContextKey, scaledResource)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com", nil)
	Expect(err).To(BeNil())

	resp, err := rt.RoundTrip(req)
	Expect(err).To(Equal(transportErr))
	Expect(resp).To(BeNil())

	counterValue, err := httpClientRequestsTotal.
		GetMetricWithLabelValues("default", scaledResource, scalerName, triggerName, metricName, "error")
	Expect(err).To(BeNil())
	Expect(counterValue).NotTo(BeNil())
	m := &dto.Metric{}
	err = counterValue.Write(m)
	Expect(err).To(BeNil())
	Expect(m.Counter.GetValue()).To(BeNumerically("==", 1))

	durationHistogram, err := httpClientRequestDuration.
		GetMetricWithLabelValues(scalerName, "error")
	Expect(err).To(BeNil())
	Expect(durationHistogram).NotTo(BeNil())
	err = durationHistogram.(prometheus.Metric).Write(m)
	Expect(err).To(BeNil())
	Expect(m.Histogram.GetSampleCount()).To(BeNumerically("==", 1))
}

func TestInstrumentedRoundTripper_ResponseReturnedUnmodified(t *testing.T) {
	RegisterTestingT(t)

	expected := fakeResponse(http.StatusAccepted)
	rt := NewInstrumentedRoundTripper(&mockRoundTripper{resp: expected})

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com", nil)
	Expect(err).To(BeNil())

	got, err := rt.RoundTrip(req)
	Expect(err).To(BeNil())
	Expect(got).To(Equal(expected))
}

func TestInstrumentedRoundTripper_NilNextUsesDefault(t *testing.T) {
	RegisterTestingT(t)

	rt := NewInstrumentedRoundTripper(nil)
	Expect(fmt.Sprintf("%T", rt)).To(Equal("*metricscollector.InstrumentedRoundTripper"))
}

func TestInstrumentedRoundTripper_ScalerContextKey_Missing(t *testing.T) {
	RegisterTestingT(t)
	rt := NewInstrumentedRoundTripper(&mockRoundTripper{resp: fakeResponse(200)})

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com", nil)
	Expect(err).To(BeNil())

	resp, err := rt.RoundTrip(req)
	Expect(err).To(BeNil())
	Expect(resp).NotTo(BeNil())
}
