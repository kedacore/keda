/*
Copyright 2021 The KEDA Authors

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

package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

type fakeMetricsServiceClient struct {
	ready         bool
	metrics       *external_metrics.ExternalMetricValueList
	getMetricsErr error
}

func (f *fakeMetricsServiceClient) GetMetrics(_ context.Context, _, _, _ string) (*external_metrics.ExternalMetricValueList, error) {
	return f.metrics, f.getMetricsErr
}

func (f *fakeMetricsServiceClient) WaitForConnectionReady(_ context.Context, _ logr.Logger) bool {
	return f.ready
}

func (f *fakeMetricsServiceClient) GetServerURL() string { return "fake:0" }

func newTestProvider(client metricsServiceClient) *KedaProvider {
	logger = logr.Discard()
	return &KedaProvider{grpcClient: client}
}

// assertAPIStatus fails the test unless err implements apierrors.APIStatus. The custom metrics
// apiserver expects handler errors to be renderable as metav1.Status; a plain error triggers the
// "apiserver received an error that is not an metav1.Status" log spam this change fixes.
func assertAPIStatus(t *testing.T, err error) {
	t.Helper()
	assert.Error(t, err)
	var apiStatus apierrors.APIStatus
	assert.True(t, errors.As(err, &apiStatus),
		"expected error to implement apierrors.APIStatus, got %T: %v", err, err)
}

func TestGetExternalMetricReturnsAPIStatusErrors(t *testing.T) {
	soSelector := labels.SelectorFromSet(labels.Set{kedav1alpha1.ScaledObjectOwnerAnnotation: "my-scaledobject"})
	info := provider.ExternalMetricInfo{Metric: "my-metric"}

	t.Run("gRPC connection not ready is reported as ServiceUnavailable", func(t *testing.T) {
		p := newTestProvider(&fakeMetricsServiceClient{ready: false})
		_, err := p.GetExternalMetric(context.Background(), "default", soSelector, info)
		assertAPIStatus(t, err)
		assert.True(t, apierrors.IsServiceUnavailable(err))
	})

	t.Run("missing scaledObject name is reported as BadRequest", func(t *testing.T) {
		p := newTestProvider(&fakeMetricsServiceClient{ready: true})
		_, err := p.GetExternalMetric(context.Background(), "default", labels.Everything(), info)
		assertAPIStatus(t, err)
		assert.True(t, apierrors.IsBadRequest(err))
	})

	t.Run("metrics service error is reported as InternalError", func(t *testing.T) {
		p := newTestProvider(&fakeMetricsServiceClient{ready: true, getMetricsErr: errors.New("rpc error: code = Unavailable")})
		_, err := p.GetExternalMetric(context.Background(), "default", soSelector, info)
		assertAPIStatus(t, err)
		assert.True(t, apierrors.IsInternalError(err))
	})

	t.Run("metrics are returned without error on success", func(t *testing.T) {
		want := &external_metrics.ExternalMetricValueList{
			Items: []external_metrics.ExternalMetricValue{{MetricName: "my-metric"}},
		}
		p := newTestProvider(&fakeMetricsServiceClient{ready: true, metrics: want})
		got, err := p.GetExternalMetric(context.Background(), "default", soSelector, info)
		assert.NoError(t, err)
		assert.Equal(t, want, got)
	})
}
