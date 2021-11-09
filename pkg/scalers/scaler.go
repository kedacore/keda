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

package scalers

import (
	"context"
	"fmt"
	"time"

	"k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	metrics "github.com/rcrowley/go-metrics"
)

func init() {
	// Disable metrics for kafka client (sarama)
	// https://github.com/Shopify/sarama/issues/1321
	metrics.UseNilMetrics = true
}

// Scaler interface
type Scaler interface {

	// The scaler returns the metric values for a metric Name and criteria matching the selector
	GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error)

	// Returns the metrics based on which this scaler determines that the ScaleTarget scales. This is used to construct the HPA spec that is created for
	// this scaled object. The labels used should match the selectors used in GetMetrics
	GetMetricSpecForScaling(ctx context.Context) []v2beta2.MetricSpec

	IsActive(ctx context.Context) (bool, error)

	// Close any resources that need disposing when scaler is no longer used or destroyed
	Close(ctx context.Context) error
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

	// ScalerIndex
	ScalerIndex int
}

// GetFromAuthOrMeta helps getting a field from Auth or Meta sections
func GetFromAuthOrMeta(config *ScalerConfig, field string) (string, error) {
	var result string
	var err error
	if config.AuthParams[field] != "" {
		result = config.AuthParams[field]
	} else if config.TriggerMetadata[field] != "" {
		result = config.TriggerMetadata[field]
	}
	if result == "" {
		err = fmt.Errorf("no %s given", field)
	}
	return result, err
}

// GenerateMetricNameWithIndex helps adding the index prefix to the metric name
func GenerateMetricNameWithIndex(scalerIndex int, metricName string) string {
	return fmt.Sprintf("s%d-%s", scalerIndex, metricName)
}
