package scalers

import (
	"context"

	v2beta1 "k8s.io/api/autoscaling/v2beta1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

type Scaler interface {
	GetScaleDecision(ctx context.Context) (int32, error)

	// The scaler returns the metric values for a metric Name and criteria matching the selector
	GetMetrics(ctx context.Context, merticName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error)

	//returns the metrics based on which this scaler determines that the deployment scales. This is used to contruct the HPA spec that is created for
	// this scaled object. The labels used should match the selectors used in GetMetrics
	GetMetricSpecForScaling() []v2beta1.MetricSpec
}
