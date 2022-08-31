package scalers

import (
	"context"
	"errors"
	"sync/atomic"

	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

const (
	MockExternalServerStatusOffline int32 = 0
	MockExternalServerStatusOnline  int32 = 1
)

var (
	MockExternalServerStatus       = MockExternalServerStatusOnline
	ErrMock                        = errors.New("mock error")
	MockMetricName                 = "mockMetricName"
	MockMetricTarget         int64 = 50
	MockMetricValue          int64 = 100
)

type externalMockScaler struct{}

func NewExternalMockScaler(config *ScalerConfig) (Scaler, error) {
	return &externalMockScaler{}, nil
}

// IsActive implements Scaler
func (*externalMockScaler) IsActive(ctx context.Context) (bool, error) {
	if atomic.LoadInt32(&MockExternalServerStatus) != MockExternalServerStatusOnline {
		return false, ErrMock
	}

	return true, nil
}

// Close implements Scaler
func (*externalMockScaler) Close(ctx context.Context) error {
	return nil
}

// GetMetricSpecForScaling implements Scaler
func (*externalMockScaler) GetMetricSpecForScaling(ctx context.Context) []v2.MetricSpec {
	if atomic.LoadInt32(&MockExternalServerStatus) != MockExternalServerStatusOnline {
		return nil
	}

	return getMockMetricsSpecs()
}

// GetMetrics implements Scaler
func (*externalMockScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	if atomic.LoadInt32(&MockExternalServerStatus) != MockExternalServerStatusOnline {
		return nil, ErrMock
	}

	return getMockExternalMetricsValue(), nil
}

func getMockMetricsSpecs() []v2.MetricSpec {
	return []v2.MetricSpec{
		{
			Type: v2.ExternalMetricSourceType,
			External: &v2.ExternalMetricSource{
				Metric: v2.MetricIdentifier{
					Name: MockMetricName,
				},
				Target: v2.MetricTarget{
					Type:  v2.ValueMetricType,
					Value: resource.NewQuantity(MockMetricValue, resource.DecimalSI),
				},
			},
		},
	}
}

func getMockExternalMetricsValue() []external_metrics.ExternalMetricValue {
	return []external_metrics.ExternalMetricValue{
		{
			MetricName: MockMetricName,
			Value:      *resource.NewQuantity(MockMetricValue, resource.DecimalSI),
			Timestamp:  metav1.Now(),
		},
	}
}
