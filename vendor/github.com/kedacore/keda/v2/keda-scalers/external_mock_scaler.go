package scalers

import (
	"context"
	"errors"
	"sync/atomic"

	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
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

func NewExternalMockScaler(_ *scalersconfig.ScalerConfig) (Scaler, error) {
	return &externalMockScaler{}, nil
}

// Close implements Scaler
func (*externalMockScaler) Close(_ context.Context) error {
	return nil
}

// GetMetricSpecForScaling implements Scaler
func (*externalMockScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	if atomic.LoadInt32(&MockExternalServerStatus) != MockExternalServerStatusOnline {
		return nil
	}

	return getMockMetricsSpecs()
}

// GetMetricsAndActivity implements Scaler
func (*externalMockScaler) GetMetricsAndActivity(_ context.Context, _ string) ([]external_metrics.ExternalMetricValue, bool, error) {
	if atomic.LoadInt32(&MockExternalServerStatus) != MockExternalServerStatusOnline {
		return nil, false, ErrMock
	}

	return getMockExternalMetricsValue(), true, nil
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
