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
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/go-logr/logr"
	metrics "github.com/rcrowley/go-metrics"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

func init() {
	// Disable metrics for kafka client (sarama)
	// https://github.com/IBM/sarama/issues/1321
	metrics.UseNilMetrics = true
}

// Scaler interface
type Scaler interface {
	// GetMetricsAndActivity returns the metric values and activity for a metric Name
	GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error)

	// GetMetricSpecForScaling returns the metrics based on which this scaler determines that the ScaleTarget scales. This is used to construct the HPA spec that is created for
	// this scaled object. The labels used should match the selectors used in GetMetrics
	GetMetricSpecForScaling(ctx context.Context) []v2.MetricSpec

	// Close any resources that need disposing when scaler is no longer used or destroyed
	Close(ctx context.Context) error
}

// PushScaler interface
type PushScaler interface {
	Scaler

	// Run is the only writer to the active channel and must close it once done.
	Run(ctx context.Context, active chan<- bool)
}

var (
	// ErrScalerUnsupportedUtilizationMetricType is returned when v2.UtilizationMetricType
	// is provided as the metric target type for scaler.
	ErrScalerUnsupportedUtilizationMetricType = errors.New("'Utilization' metric type is unsupported for external metrics, allowed values are 'Value' or 'AverageValue'")
)

// GenerateMetricNameWithIndex helps to add the index prefix to the metric name
func GenerateMetricNameWithIndex(triggerIndex int, metricName string) string {
	return fmt.Sprintf("s%d-%s", triggerIndex, metricName)
}

// RemoveIndexFromMetricName removes the index prefix from the metric name
func RemoveIndexFromMetricName(triggerIndex int, metricName string) (string, error) {
	metricNameSplit := strings.SplitN(metricName, "-", 2)
	if len(metricNameSplit) != 2 {
		return "", fmt.Errorf("metric name without index prefix")
	}

	indexPrefix, metricNameWithoutIndex := metricNameSplit[0], metricNameSplit[1]
	if indexPrefix != fmt.Sprintf("s%d", triggerIndex) {
		return "", fmt.Errorf("metric name contains incorrect index prefix")
	}

	return metricNameWithoutIndex, nil
}

func InitializeLogger(config *scalersconfig.ScalerConfig, scalerName string) logr.Logger {
	return logf.Log.WithName(scalerName).WithValues("type", config.ScalableObjectType, "namespace", config.ScalableObjectNamespace, "name", config.ScalableObjectName)
}

// GetMetricTargetType helps get the metric target type of the scaler
func GetMetricTargetType(config *scalersconfig.ScalerConfig) (v2.MetricTargetType, error) {
	switch config.MetricType {
	case v2.UtilizationMetricType:
		return "", ErrScalerUnsupportedUtilizationMetricType
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

	// Construct the target size as a quantity using string parsing to avoid
	// int64 overflow when metricValue * 1000 exceeds math.MaxInt64 (~9.2e15 threshold).
	// Treat NaN and Inf as zero to avoid a panic in resource.MustParse.
	targetQty := quantityFromFloat64(metricValue)
	if metricType == v2.AverageValueMetricType {
		target.AverageValue = &targetQty
	} else {
		target.Value = &targetQty
	}

	return target
}

// GenerateMetricInMili returns a externalMetricValue with mili as metric scale
func GenerateMetricInMili(metricName string, value float64) external_metrics.ExternalMetricValue {
	// Use string parsing to avoid int64 overflow when value * 1000 exceeds
	// math.MaxInt64 (~9.2e15 threshold).
	// Treat NaN and Inf as zero to avoid a panic in resource.MustParse.
	qty := quantityFromFloat64(value)
	return external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      qty,
		Timestamp:  metav1.Now(),
	}
}

// quantityFromFloat64 converts a float64 to a resource.Quantity with milli-precision.
// NaN and Inf values are treated as zero to prevent panics.
func quantityFromFloat64(value float64) resource.Quantity {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return resource.MustParse("0")
	}
	return resource.MustParse(fmt.Sprintf("%.3f", value))
}
