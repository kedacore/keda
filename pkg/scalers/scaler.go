package scalers

import (
	"context"
	"time"

	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
)

// Scaler interface
type Scaler interface {

	// The scaler returns the metric values for a metric Name and criteria matching the selector
	GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error)

	// Returns the metrics based on which this scaler determines that the ScaleTarget scales. This is used to construct the HPA spec that is created for
	// this scaled object. The labels used should match the selectors used in GetMetrics
	GetMetricSpecForScaling() []v2beta2.MetricSpec

	IsActive(ctx context.Context) (bool, error)

	// Close any resources that need disposing when scaler is no longer used or destroyed
	Close() error
}

// PushScaler interface
type PushScaler interface {
	Scaler

	// Run is the only writer to the active channel and must close it once done.
	Run(ctx context.Context, active chan<- bool)
}

// ScalerConfig contains config fields common for all scalers
type ScalerConfig struct {
	// Name used for external scalers
	Name string

	// The timeout to be used on all HTTP requests from the controller
	GlobalHTTPTimeout time.Duration

	// Namespace used for external scalers
	Namespace string

	// TriggerMetadata
	TriggerMetadata map[string]string

	// ResolvedEnv
	ResolvedEnv map[string]string

	// AuthParams
	AuthParams map[string]string

	// PodIdentity
	PodIdentity kedav1alpha1.PodIdentityProvider
}
