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
	"strings"
	"time"

	"github.com/go-logr/logr"
	metrics "github.com/rcrowley/go-metrics"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
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
	GetMetricSpecForScaling(ctx context.Context) []v2.MetricSpec

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
	// ScalableObjectName specifies name of the ScaledObject/ScaledJob that owns this scaler
	ScalableObjectName string

	// ScalableObjectNamespace specifies name of the ScaledObject/ScaledJob that owns this scaler
	ScalableObjectNamespace string

	// ScalableObjectType specifies whether this Scaler is owned by ScaledObject or ScaledJob
	ScalableObjectType string

	// The timeout to be used on all HTTP requests from the controller
	GlobalHTTPTimeout time.Duration

	// Name of the trigger
	TriggerName string

	// TriggerMetadata
	TriggerMetadata map[string]string

	// ResolvedEnv
	ResolvedEnv map[string]string

	// AuthParams
	AuthParams map[string]string

	// PodIdentity
	PodIdentity kedav1alpha1.AuthPodIdentity

	// ScalerIndex
	ScalerIndex int

	// MetricType
	MetricType v2.MetricTargetType
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

// RemoveIndexFromMetricName removes the index prefix from the metric name
func RemoveIndexFromMetricName(scalerIndex int, metricName string) (string, error) {
	metricNameSplit := strings.SplitN(metricName, "-", 2)
	if len(metricNameSplit) != 2 {
		return "", fmt.Errorf("metric name without index prefix")
	}

	indexPrefix, metricNameWithoutIndex := metricNameSplit[0], metricNameSplit[1]
	if indexPrefix != fmt.Sprintf("s%d", scalerIndex) {
		return "", fmt.Errorf("metric name contains incorrect index prefix")
	}

	return metricNameWithoutIndex, nil
}

func InitializeLogger(config *ScalerConfig, scalerName string) logr.Logger {
	return logf.Log.WithName(scalerName).WithValues("type", config.ScalableObjectType, "namespace", config.ScalableObjectNamespace, "name", config.ScalableObjectName)
}

// GetMetricTargetType helps getting the metric target type of the scaler
func GetMetricTargetType(config *ScalerConfig) (v2.MetricTargetType, error) {
	switch config.MetricType {
	case v2.UtilizationMetricType:
		return "", fmt.Errorf("'Utilization' metric type is unsupported for external metrics, allowed values are 'Value' or 'AverageValue'")
	case "":
		// Use AverageValue if no metric type was provided
		return v2.AverageValueMetricType, nil
	default:
		return config.MetricType, nil
	}
}

// GetMetricTarget returns a metric target for a valid given metric target type (Value or AverageValue) and value
func GetMetricTarget(metricType v2.MetricTargetType, metricValue int64) v2.MetricTarget {
	target := v2.MetricTarget{
		Type: metricType,
	}

	// Construct the target size as a quantity
	targetQty := resource.NewQuantity(metricValue, resource.DecimalSI)
	if metricType == v2.AverageValueMetricType {
		target.AverageValue = targetQty
	} else {
		target.Value = targetQty
	}

	return target
}

// GetMetricTargetMili returns a metric target for a valid given metric target type (Value or AverageValue) and value in mili scale
func GetMetricTargetMili(metricType v2.MetricTargetType, metricValue float64) v2.MetricTarget {
	target := v2.MetricTarget{
		Type: metricType,
	}

	// Construct the target size as a quantity
	metricValueMili := int64(metricValue * 1000)
	targetQty := resource.NewMilliQuantity(metricValueMili, resource.DecimalSI)
	if metricType == v2.AverageValueMetricType {
		target.AverageValue = targetQty
	} else {
		target.Value = targetQty
	}

	return target
}

// GenerateMetricInMili returns a externalMetricValue with mili as metric scale
func GenerateMetricInMili(metricName string, value float64) external_metrics.ExternalMetricValue {
	valueMili := int64(value * 1000)
	return external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewMilliQuantity(valueMili, resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}
}
